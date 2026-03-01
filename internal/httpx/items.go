package httpx

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Item struct {
	ID        int64  `json:"id"`
	Shop      string `json:"shop"`
	Text      string `json:"text"`
	Done      bool   `json:"done"`
	UpdatedAt int64  `json:"updatedAt"`
}

type Template struct {
	Shop       string `json:"shop"`
	Text       string `json:"text"`
	LastUsedAt int64  `json:"lastUsedAt"`
	UseCount   int64  `json:"useCount"`
}

func (a *App) registerItemRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/items", a.requireAuth(http.HandlerFunc(a.handleListItems)))
	mux.Handle("POST /api/items", a.requireAuth(http.HandlerFunc(a.handleAddItem)))
	mux.Handle("POST /api/items/{id}/toggle", a.requireAuth(http.HandlerFunc(a.handleToggleItem)))
	mux.Handle("DELETE /api/items/{id}", a.requireAuth(http.HandlerFunc(a.handleDeleteItem)))
	mux.Handle("POST /api/items/clear-done", a.requireAuth(http.HandlerFunc(a.handleClearDone)))

	mux.Handle("GET /api/history", a.requireAuth(http.HandlerFunc(a.handleHistory)))
}

func (a *App) shopFromQuery(r *http.Request) (string, bool) {
	shop := strings.TrimSpace(r.URL.Query().Get("shop"))
	if shop == "" {
		shop = a.DefaultShop
	}
	if !contains(a.Shops, shop) {
		return "", false
	}
	return shop, true
}

func (a *App) handleListItems(w http.ResponseWriter, r *http.Request) {
	shop, ok := a.shopFromQuery(r)
	if !ok {
		http.Error(w, "invalid shop", http.StatusBadRequest)
		return
	}

	rows, err := a.DB.Query(
		`SELECT id, shop, text, done, updated_at
		 FROM items
		 WHERE shop = ?
		 ORDER BY done ASC, updated_at DESC, id DESC`,
		shop,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var out []Item
	for rows.Next() {
		var it Item
		var doneInt int
		if err := rows.Scan(&it.ID, &it.Shop, &it.Text, &doneInt, &it.UpdatedAt); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if it.Shop == "" {
			it.Shop = a.DefaultShop
		}
		it.Done = doneInt != 0
		out = append(out, it)
	}

	writeJSON(w, out)
}

func (a *App) handleAddItem(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Shop string `json:"shop"`
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	shop := strings.TrimSpace(req.Shop)
	if shop == "" {
		shop = a.DefaultShop
	}
	if !contains(a.Shops, shop) {
		http.Error(w, "invalid shop", http.StatusBadRequest)
		return
	}

	txt := normalizeText(req.Text)
	if txt == "" {
		http.Error(w, "text required", http.StatusBadRequest)
		return
	}

	now := time.Now().Unix()

	// Insert item
	res, err := a.DB.Exec(
		`INSERT INTO items(shop, text, done, created_at, updated_at) VALUES(?, ?, 0, ?, ?)`,
		shop, txt, now, now,
	)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	// Upsert template history per shop
	_, _ = a.DB.Exec(`
INSERT INTO templates(shop, text, last_used_at, use_count)
VALUES(?, ?, ?, 1)
ON CONFLICT(shop, text) DO UPDATE SET
	last_used_at=excluded.last_used_at,
	use_count=use_count+1
`, shop, txt, now)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, Item{ID: id, Shop: shop, Text: txt, Done: false, UpdatedAt: now})
}

func (a *App) handleToggleItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	now := time.Now().Unix()

	res, err := a.DB.Exec(`
UPDATE items
SET done = CASE done WHEN 0 THEN 1 ELSE 0 END,
    updated_at = ?
WHERE id = ?
`, now, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	var it Item
	var doneInt int
	err = a.DB.QueryRow(`SELECT id, shop, text, done, updated_at FROM items WHERE id = ?`, id).
		Scan(&it.ID, &it.Shop, &it.Text, &doneInt, &it.UpdatedAt)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if it.Shop == "" {
		it.Shop = a.DefaultShop
	}
	it.Done = doneInt != 0

	writeJSON(w, it)
}

func (a *App) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r.PathValue("id"))
	if !ok {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	res, err := a.DB.Exec(`DELETE FROM items WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleClearDone(w http.ResponseWriter, r *http.Request) {
	shop, ok := a.shopFromQuery(r)
	if !ok {
		http.Error(w, "invalid shop", http.StatusBadRequest)
		return
	}

	_, err := a.DB.Exec(`DELETE FROM items WHERE shop = ? AND done = 1`, shop)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleHistory(w http.ResponseWriter, r *http.Request) {
	shop, ok := a.shopFromQuery(r)
	if !ok {
		http.Error(w, "invalid shop", http.StatusBadRequest)
		return
	}

	limit := int64(20)
	if s := r.URL.Query().Get("limit"); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil && v > 0 && v <= 200 {
			limit = v
		}
	}

	rows, err := a.DB.Query(`
SELECT shop, text, last_used_at, use_count
FROM templates
WHERE shop = ?
ORDER BY last_used_at DESC
LIMIT ?
`, shop, limit)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var out []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.Shop, &t.Text, &t.LastUsedAt, &t.UseCount); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if t.Shop == "" {
			t.Shop = a.DefaultShop
		}
		out = append(out, t)
	}

	writeJSON(w, out)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func parseID(s string) (int64, bool) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func normalizeText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}
