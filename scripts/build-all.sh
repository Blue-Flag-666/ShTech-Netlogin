#!/usr/bin/env sh
set -eu

version="${1:-dev}"
project="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
mkdir -p "$project/dist"
cd "$project"

build() {
  os="$1" arch="$2" arm="${3:-}" suffix="${4:-}"
  label="$os-$arch"
  [ -z "$arm" ] || label="${label}v${arm}"
  echo "Building $label"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" GOARM="$arm" \
    go build -trimpath -ldflags "-s -w -X main.version=$version" \
    -o "dist/shtu-net-login-${label}${suffix}" ./cmd/shtu-net-login
}

build windows amd64 "" .exe
build windows arm64 "" .exe
build linux amd64
build linux arm64
build linux arm 7
build darwin amd64
build darwin arm64
