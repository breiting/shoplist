package httpx

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"
)

type contextKey string

const ctxAuth contextKey = "auth"

func (a *App) registerAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /login", a.handleLogin)
	mux.HandleFunc("POST /logout", a.handleLogout)
	mux.Handle("GET /api/me", a.requireAuth(http.HandlerFunc(a.handleMe)))
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.Password != a.Password {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := randomToken(32)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	exp := time.Now().Add(a.SessionTTL).Unix()

	_, err = a.DB.Exec(`INSERT INTO sessions(token, expires_at) VALUES(?, ?)`, token, exp)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     a.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(exp, 0),
	})

	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(a.CookieName)
	if err == nil {
		a.DB.Exec(`DELETE FROM sessions WHERE token = ?`, c.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     a.CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(a.CookieName)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var expires int64
		err = a.DB.QueryRow(`SELECT expires_at FROM sessions WHERE token = ?`, c.Value).Scan(&expires)
		if err != nil || time.Now().Unix() > expires {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ctxAuth, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
