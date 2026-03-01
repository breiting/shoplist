package httpx

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web/*
var webFS embed.FS

type Config struct {
	Addr string
}

func NewServer(cfg Config) *http.Server {
	app, err := NewApp()
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok\n"))
	})

	app.registerAuthRoutes(mux)

	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		panic(err)
	}

	mux.Handle("/", cacheControl(http.FileServer(http.FS(staticFS))))

	h := securityHeaders(mux)

	return &http.Server{
		Addr:    cfg.Addr,
		Handler: h,
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func cacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
