#!/bin/sh
set -e

GOOS=js GOARCH=wasm go build -o web/public/main.wasm ./cmd/wasm

cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/public/

echo "web/public/main.wasm + web/public/wasm_exec.js"
