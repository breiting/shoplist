# Build stage
FROM golang:1.22-bookworm AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/shoplist ./cmd/shoplist

# Runtime stage (distroless)
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/shoplist /shoplist

ENV SHOPLIST_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/shoplist"]
