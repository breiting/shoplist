# ShopList

A tiny, self-hosted shopping list for families.

- Works on iPhone (Safari) and GrapheneOS (any Chromium-based browser)
- Web-based (no native apps), installable as a PWA (Add to Home Screen)
- Minimal backend: Go + SQLite (single DB file)
- Designed for simple homelab deployments (OPNsense/HAProxy, Proxmox, Debian, LXC)

## Motivation

Most shopping list apps are either:

- cloud-first (accounts, sync services, subscriptions),
- heavy (framework stacks, multiple services),
- or not ideal for privacy-focused setups.

**shoplist** is the opposite:

- Self-hosted
- Single binary
- Single SQLite database file
- No user management, just a shared household password
- Intentionally small and focused

It follows a Unix philosophy: do one thing well.

## Features

- Shared household login via password session
- Multiple shops (e.g. Spar, Billa, Bauernladen)
- No duplicates per shop
- Check/uncheck items
- Clear completed items
- “Last used” history per shop
- Optional quantity per item (free text: `2`, `10 dag`, `250 g`, …)
- PWA installable (iOS + GrapheneOS)

## Tech Stack

- Backend: Go (net/http)
- Database: SQLite (single file, WAL enabled)
- Frontend: HTML + minimal JS + CSS
- No frameworks

# Quick Start

## Requirements

- Go 1.25+

## Run

```bash
export SHOPLIST_PASSWORD='your-long-household-passphrase'
export SHOPLIST_DATA_DIR='./data'
export SHOPLIST_ADDR=':8080'
export SHOPLIST_SHOPS='Spar,Billa,Bauernladen'
export SHOPLIST_DEFAULT_SHOP='Spar'
export SHOPLIST_COOKIE_SECURE='0'

go run ./cmd/shoplist
```

Open `http://localhost:8080`

# Build

```bash
go build -o shoplist ./cmd/shoplist
```

## Cross compile (Linux amd64)

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o shoplist ./cmd/shoplist
```

# Configuration

| Variable                  | Description                |
| ------------------------- | -------------------------- |
| SHOPLIST_ADDR             | Listen address             |
| SHOPLIST_PASSWORD         | Shared password (required) |
| SHOPLIST_DATA_DIR         | Directory for SQLite DB    |
| SHOPLIST_SESSION_TTL_DAYS | Session duration           |
| SHOPLIST_COOKIE_SECURE    | Set 1 when behind HTTPS    |
| SHOPLIST_SHOPS            | Comma-separated shop list  |
| SHOPLIST_DEFAULT_SHOP     | Default shop               |

# Deployment

## Debian LXC (Proxmox recommended)

1.  Create Debian LXC (1 vCPU, 512MB RAM)
2.  Copy binary to /usr/local/bin/shoplist
3.  Create data dir: /var/lib/shoplist
4.  Create systemd service
5.  Run behind HTTPS reverse proxy

Logs:

```bash
journalctl -u shoplist -f
```

## Docker

```bash
docker compose up -d --build
```

# Backups

SQLite database file is located at:

`SHOPLIST_DATA_DIR/shoplist.db`

Back up the entire directory.

# Security

- Use a long password (20+ chars)
- Run behind HTTPS
- Enable SHOPLIST_COOKIE_SECURE=1

# License

MIT License
