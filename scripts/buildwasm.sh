#!/bin/sh

set -e

GOARCH=wasm GOOS=js go build -o lib.wasm ./cmd/wasm/main.go

