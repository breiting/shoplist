package httpx

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestItemsFlow(t *testing.T) {
	td := t.TempDir()
	os.Setenv("SHOPLIST_PASSWORD", "testpw")
	os.Setenv("SHOPLIST_DATA_DIR", filepath.Join(td, "data"))
	os.Setenv("SHOPLIST_SESSION_TTL_DAYS", "10")
	os.Setenv("SHOPLIST_COOKIE_SECURE", "0")
	t.Cleanup(func() {
		os.Unsetenv("SHOPLIST_PASSWORD")
		os.Unsetenv("SHOPLIST_DATA_DIR")
		os.Unsetenv("SHOPLIST_SESSION_TTL_DAYS")
		os.Unsetenv("SHOPLIST_COOKIE_SECURE")
	})

	srv := NewServer(Config{Addr: ":0"})
	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(ts.Close)

	// login
	loginBody := []byte(`{"password":"testpw"}`)
	res, err := http.Post(ts.URL+"/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 204 {
		t.Fatalf("login status: %d", res.StatusCode)
	}
	var cookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == "shoplist_session" {
			cookie = c
			break
		}
	}
	if cookie == nil {
		t.Fatal("missing session cookie")
	}

	client := &http.Client{}

	// add item
	req, _ := http.NewRequest("POST", ts.URL+"/api/items", bytes.NewReader([]byte(`{"text":"Milk"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	res, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 201 {
		t.Fatalf("add status: %d", res.StatusCode)
	}

	// list items
	req, _ = http.NewRequest("GET", ts.URL+"/api/items", nil)
	req.AddCookie(cookie)
	res, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("list status: %d", res.StatusCode)
	}

	// history should contain Milk
	req, _ = http.NewRequest("GET", ts.URL+"/api/history?limit=20", nil)
	req.AddCookie(cookie)
	res, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("history status: %d", res.StatusCode)
	}
}
