package httpx

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type App struct {
	DB           *sql.DB
	Password     string
	SessionTTL   time.Duration
	CookieSecure bool
	CookieName   string

	Shops       []string
	DefaultShop string
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

	shops := parseCSV(getenv("SHOPLIST_SHOPS", "Default"))
	if len(shops) == 0 {
		shops = []string{"Default"}
	}
	defaultShop := getenv("SHOPLIST_DEFAULT_SHOP", shops[0])
	if !contains(shops, defaultShop) {
		defaultShop = shops[0]
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "shoplist.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Pragmas
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

		Shops:       shops,
		DefaultShop: defaultShop,
	}, nil
}

// migrate is idempotent and upgrades older DBs in place.
func migrate(db *sql.DB) error {
	// Base schema (v1-ish)
	base := `
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
	if _, err := db.Exec(base); err != nil {
		return err
	}

	// --- Upgrade: add shop to items if missing
	hasShop, err := tableHasColumn(db, "items", "shop")
	if err != nil {
		return err
	}
	if !hasShop {
		if _, err := db.Exec(`ALTER TABLE items ADD COLUMN shop TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
		_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_items_shop_done_updated ON items(shop, done, updated_at)`)
	}

	// --- v1.1: add qty to items if missing (optional text)
	hasQty, err := tableHasColumn(db, "items", "qty")
	if err != nil {
		return err
	}
	if !hasQty {
		if _, err := db.Exec(`ALTER TABLE items ADD COLUMN qty TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}

	// --- Upgrade: templates must become (shop,text) unique to preserve history per shop
	templatesHasShop, err := tableHasColumn(db, "templates", "shop")
	if err != nil {
		return err
	}
	if !templatesHasShop {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		_, err = tx.Exec(`
CREATE TABLE IF NOT EXISTS templates_new (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	shop TEXT NOT NULL,
	text TEXT NOT NULL,
	last_used_at INTEGER NOT NULL,
	use_count INTEGER NOT NULL DEFAULT 1,
	UNIQUE(shop, text)
);
CREATE INDEX IF NOT EXISTS idx_templates_new_last_used ON templates_new(last_used_at);
`)
		if err != nil {
			return err
		}

		// migrate old templates into default shop "" bucket
		_, err = tx.Exec(`
INSERT INTO templates_new(shop, text, last_used_at, use_count)
SELECT '', text, last_used_at, use_count FROM templates
`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`DROP TABLE templates`)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`ALTER TABLE templates_new RENAME TO templates`)
		if err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	}

	// --- Upgrade: enforce uniqueness per (shop,text) for items
	// Deduplicate existing rows first (keep most recently updated row).
	_, _ = db.Exec(`
DELETE FROM items
WHERE id NOT IN (
  SELECT id FROM (
    SELECT id
    FROM items i1
    WHERE i1.id = (
      SELECT i2.id
      FROM items i2
      WHERE i2.shop = i1.shop AND i2.text = i1.text
      ORDER BY i2.updated_at DESC, i2.id DESC
      LIMIT 1
    )
  )
)
`)
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_items_shop_text ON items(shop, text)`); err != nil {
		return err
	}

	return nil
}

func tableHasColumn(db *sql.DB, table, col string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, col) {
			return true, nil
		}
	}
	return false, nil
}

func parseCSV(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func contains(xs []string, v string) bool {
	return slices.Contains(xs, v)
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
