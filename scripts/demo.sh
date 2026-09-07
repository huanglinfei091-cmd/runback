#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
go build -o bin/runback ./cmd/runback
url=https://github.com/pallets/click/actions/runs/33760631267
if [[ "${1:-}" == "--replay" ]]; then
  bin/runback "$url" --bundle testdata/real/click.json
else
  bin/runback "$url" --bundle testdata/real/click.json --dry-run
  echo "Prepared from saved real evidence. Use bash scripts/demo.sh --replay for Docker execution."
fi
