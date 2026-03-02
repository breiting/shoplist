# syntax=docker/dockerfile:1

FROM golang:1.25-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME=unknown

COPY . .

ARG RUN_TESTS=1
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN if [ "$RUN_TESTS" = "1" ]; then CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go test ./...; fi
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath \
	-ldflags="-s -w \
    -X 'main.Version=${VERSION}' \
    -X 'main.Commit=${COMMIT}' \
    -X 'main.BuildTime=${BUILD_TIME}'" \
	-o /out/shoplist ./cmd/shoplist

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/shoplist /shoplist

ENV SHOPLIST_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/shoplist"]
