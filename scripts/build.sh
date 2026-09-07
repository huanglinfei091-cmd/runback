#!/usr/bin/env bash
set -euo pipefail
unset RUNBACK_GITHUB_TOKEN GH_TOKEN GITHUB_TOKEN
cd "$(dirname "$0")/.."
mkdir -p bin
gofmt -l cmd internal
go test ./...
go vet ./...
go build -trimpath -o bin/runback ./cmd/runback
bin/runback --version
