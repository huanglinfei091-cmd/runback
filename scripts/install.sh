#!/usr/bin/env bash
set -euo pipefail

unset RUNBACK_GITHUB_TOKEN GH_TOKEN GITHUB_TOKEN

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
install_dir="${RUNBACK_INSTALL_DIR:-$HOME/.local/bin}"
install_version="${RUNBACK_INSTALL_VERSION:-dev}"

if [[ ! "$install_version" =~ ^[0-9A-Za-z._+-]+$ ]]; then
  echo "RUNBACK_INSTALL_VERSION contains unsupported characters" >&2
  exit 2
fi
if ! command -v go >/dev/null 2>&1; then
  echo "Go 1.24 or newer is required to build RunBack from source." >&2
  exit 2
fi

mkdir -p "$install_dir"
tmp_binary="$(mktemp "$install_dir/.runback-install.XXXXXX")"
cleanup() {
  rm -f -- "$tmp_binary"
}
trap cleanup EXIT

cd "$repo_root"
go build -trimpath -ldflags="-X main.version=$install_version" -o "$tmp_binary" ./cmd/runback
chmod 0755 "$tmp_binary"
mv -f -- "$tmp_binary" "$install_dir/runback"
trap - EXIT

echo "Installed RunBack $install_version to $install_dir/runback"
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) echo "Add $install_dir to PATH, then run: runback doctor" ;;
esac
