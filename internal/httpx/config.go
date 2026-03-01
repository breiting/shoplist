package httpx

import (
	"net/http"
)

func (a *App) registerConfigRoute(mux *http.ServeMux) {
	mux.Handle("GET /api/config", a.requireAuth(http.HandlerFunc(a.handleConfig)))
}

func (a *App) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"shops":       a.Shops,
		"defaultShop": a.DefaultShop,
	})
}
