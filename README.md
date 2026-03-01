# shoplist

A minimal family shopping list web app.

Goals:

- Works on iPhone (Safari) and GrapheneOS (Chromium) as a PWA.
- Single small backend (Go) + SQLite (next steps).
- Simple, Unix-like deployment (Docker Compose).

## Development

Run locally:

```bash
go test ./...
SHOPLIST_ADDR=:8080 go run ./cmd/shoplist
```
