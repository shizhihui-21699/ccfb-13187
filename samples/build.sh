#!/bin/bash
# Build stripped Go samples across architectures for testing gores.
set -euo pipefail
cd "$(dirname "$0")/hello"
go mod tidy

OUT=../bin
mkdir -p "$OUT"
FLAGS="-s -w"   # strip symbol table & DWARF

build() {
  local goos=$1 goarch=$2 name=$3
  echo "==> $name ($goos/$goarch)"
  GOOS=$goos GOARCH=$goarch go build -ldflags="$FLAGS" -o "$OUT/$name" .
}

build linux   amd64 sample-linux-amd64
build linux   arm64 sample-linux-arm64
build linux   386   sample-linux-386
build windows amd64 sample-windows-amd64.exe
build darwin  arm64 sample-darwin-arm64
build darwin  amd64 sample-darwin-amd64

echo
ls -lh "$OUT"
