#!/usr/bin/env bash
# Build kbsink.wasm and copy wasm_exec.js next to it (repo root by default).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export CGO_ENABLED=0
export GOOS=js
export GOARCH=wasm

echo "building kbsink.wasm ..."
go build -trimpath -ldflags="-s -w" -o kbsink.wasm ./cmd/kbsink-wasm

GOROOT="$(go env GOROOT)"
for cand in "$GOROOT/lib/wasm/wasm_exec.js" "$GOROOT/misc/wasm/wasm_exec.js"; do
  if [[ -f "$cand" ]]; then
    cp "$cand" wasm_exec.js
    echo "copied wasm_exec.js from $cand"
    exit 0
  fi
done

echo "wasm_exec.js not found under GOROOT=$GOROOT" >&2
exit 1
