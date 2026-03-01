package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	srv := NewServer(Config{Addr: ":0"})
	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	b, _ := io.ReadAll(resp.Body)
	if string(b) != "ok\n" {
		t.Fatalf("unexpected body: %q", string(b))
	}
}
