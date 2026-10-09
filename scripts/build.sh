#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOAMD64=v1 go build \
  -mod=readonly -trimpath -ldflags "-s -w -X main.version=${1:-dev}" \
  -o dist/panaino-bot .
(cd dist && sha256sum panaino-bot > SHA256SUMS)
cat dist/SHA256SUMS
