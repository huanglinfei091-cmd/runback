#!/usr/bin/env bash
set -euo pipefail

# Installation does not need or consume GitHub credentials.
unset RUNBACK_GITHUB_TOKEN GH_TOKEN GITHUB_TOKEN

repository="huanglinfei091-cmd/runback"
version="${RUNBACK_INSTALL_VERSION:-v0.1.1-alpha}"
install_dir="${RUNBACK_INSTALL_DIR:-$HOME/.local/bin}"

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+-[0-9A-Za-z][0-9A-Za-z.-]*$ ]]; then
  echo "RUNBACK_INSTALL_VERSION must be a release tag such as v0.1.0-alpha" >&2
  exit 2
fi
if [[ "$(uname -s)" != "Linux" ]]; then
  echo "RunBack release binaries currently support Linux only." >&2
  exit 2
fi
case "$(uname -m)" in
  x86_64|amd64) architecture="amd64" ;;
  *)
    echo "RunBack release binaries currently support linux-amd64 only." >&2
    exit 2
    ;;
esac
for tool in awk install tar sha256sum mktemp; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Missing required installation tool: $tool" >&2
    exit 2
  fi
done

archive="runback-${version}-linux-${architecture}.tar.gz"
bundle="${archive%.tar.gz}"
base_url="https://github.com/${repository}/releases/download/${version}"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/runback-release-install.XXXXXX")"
tmp_binary=""
cleanup() {
  rm -rf -- "$tmp_dir"
  if [[ -n "$tmp_binary" ]]; then
    rm -f -- "$tmp_binary"
  fi
}
trap cleanup EXIT

download() {
  url=$1
  destination=$2
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 3 --retry-delay 1 --connect-timeout 20 -o "$destination" "$url"
  elif command -v wget >/dev/null 2>&1; then
    wget -q --tries=3 --timeout=20 -O "$destination" "$url"
  else
    echo "Install curl or wget, then rerun the installer." >&2
    exit 2
  fi
}

download "$base_url/$archive" "$tmp_dir/$archive"
download "$base_url/SHA256SUMS" "$tmp_dir/SHA256SUMS"

expected="$(awk -v name="$archive" '$2 == name {print $1}' "$tmp_dir/SHA256SUMS")"
if [[ ! "$expected" =~ ^[0-9a-fA-F]{64}$ ]]; then
  echo "SHA256SUMS does not contain exactly one valid checksum for $archive" >&2
  exit 1
fi
actual="$(sha256sum "$tmp_dir/$archive" | awk '{print $1}')"
if [[ "$actual" != "$expected" ]]; then
  echo "Checksum verification failed for $archive" >&2
  exit 1
fi

tar -xzf "$tmp_dir/$archive" -C "$tmp_dir" "$bundle/runback"
mkdir -p "$install_dir"
tmp_binary="$(mktemp "$install_dir/.runback-install.XXXXXX")"
install -m 0755 "$tmp_dir/$bundle/runback" "$tmp_binary"
mv -f -- "$tmp_binary" "$install_dir/runback"
tmp_binary=""

echo "Installed RunBack $version to $install_dir/runback"
case ":$PATH:" in
  *":$install_dir:"*) echo "Next: runback doctor" ;;
  *)
    echo "Next: export PATH=\"$install_dir:\$PATH\""
    echo "Then: runback doctor"
    ;;
esac
