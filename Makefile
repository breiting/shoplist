VERSION ?= dev
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -trimpath -ldflags="-s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.Commit=$(GIT_COMMIT)' \
	-X 'main.BuildTime=$(BUILD_TIME)'" \
	-o shoplist ./cmd/shoplist
