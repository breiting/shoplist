package httpx

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
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

	ttlDays := getenv("SHOPLIST_SESSION_TTL_DAYS", "180")
	ttl, _ := time.ParseDuration(ttlDays + "24h")
	if ttl == 0 {
		ttl = 180 * 24 * time.Hour
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "shoplist.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		return nil, err
	}

	return &App{
		DB:           db,
		Password:     password,
		SessionTTL:   ttl,
		CookieSecure: false, // hinter TLS später true setzen
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
