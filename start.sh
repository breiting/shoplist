#!/bin/sh

export SHOPLIST_ADDR=:8080
export SHOPLIST_PASSWORD=changeme
export SHOPLIST_DATA_DIR=./data
export SHOPLIST_SESSION_TTL_DAYS=180
export SHOPLIST_COOKIE_SECURE=0
export SHOPLIST_SHOPS=Spar,Billa,Bauernladen
export SHOPLIST_DEFAULT_SHOP=Spar

go run ./cmd/shoplist
