package httpx

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "modernc.org/sqlite"
)

type App struct {
	DB           *sql.DB
	Password     string
	SessionTTL   time.Duration
	CookieSecure bool
	CookieName   string
}

func NewApp() (*App, error) {
	dataDir := getenv("SHOPLIST_DATA_DIR", "/data")
	password := os.Getenv("SHOPLIST_PASSWORD")
	if password == "" {
		return nil, errors.New("SHOPLIST_PASSWORD must be set")
	}

	ttlDays, err := strconv.Atoi(getenv("SHOPLIST_SESSION_TTL_DAYS", "180"))
	if err != nil || ttlDays <= 0 {
		ttlDays = 180
	}
	ttl := time.Duration(ttlDays) * 24 * time.Hour

	cookieSecure := getenv("SHOPLIST_COOKIE_SECURE", "0") == "1"

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "shoplist.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Pragmas: decent defaults for sqlite in a small homelab app.
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		return nil, fmt.Errorf("pragma wal: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout=3000;`); err != nil {
		return nil, fmt.Errorf("pragma busy_timeout: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON;`); err != nil {
		return nil, fmt.Errorf("pragma foreign_keys: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, err
	}

	return &App{
		DB:           db,
		Password:     password,
		SessionTTL:   ttl,
		CookieSecure: cookieSecure,
		CookieName:   "shoplist_session",
	}, nil
}

func migrate(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS sessions (
	token TEXT PRIMARY KEY,
	expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	text TEXT NOT NULL,
	done INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_items_done_updated ON items(done, updated_at);

CREATE TABLE IF NOT EXISTS templates (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	text TEXT NOT NULL UNIQUE,
	last_used_at INTEGER NOT NULL,
	use_count INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_templates_last_used ON templates(last_used_at);
`
	_, err := db.Exec(schema)
	return err
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
