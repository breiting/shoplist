package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/breiting/shoplist/internal/httpx"
)

func main() {
	addr := getenv("SHOPLIST_ADDR", ":8080")

	srv := httpx.NewServer(httpx.Config{
		Addr: addr,
	})

	// Start server
	errCh := make(chan error, 1)
	go func() {
		log.Printf("shoplist listening on %s", addr)
		errCh <- srv.ListenAndServe()
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("signal received: %s", sig.String())
	case err := <-errCh:
		// http.Server returns ErrServerClosed on normal shutdown
		log.Printf("server stopped: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Printf("bye")
}

func getenv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
