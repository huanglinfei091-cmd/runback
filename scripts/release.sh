#!/usr/bin/env bash
set -euo pipefail

unset RUNBACK_GITHUB_TOKEN GH_TOKEN GITHUB_TOKEN
cd "$(dirname "$0")/.."
version="${1:-v0.1.1-alpha}"
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-[0-9A-Za-z.-]+$ ]]; then
  echo "release version must look like v0.1.0-alpha" >&2
  exit 2
fi
artifact="runback-${version}-linux-amd64"
rm -rf dist
mkdir -p "dist/$artifact"
go test ./...
go vet ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X main.version=$version" -o "dist/$artifact/runback" ./cmd/runback
cp README.md LICENSE "dist/$artifact/"
tar -C dist -czf "dist/$artifact.tar.gz" "$artifact"
(cd dist && sha256sum "$artifact.tar.gz" > SHA256SUMS)
rm -rf "dist/$artifact"
echo "Built dist/$artifact.tar.gz and dist/SHA256SUMS"
