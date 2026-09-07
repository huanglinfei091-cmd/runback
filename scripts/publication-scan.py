#!/usr/bin/env python3
"""Scan the public source set without printing any matched secret value."""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path


IGNORED_DIRS = {".git", ".runback", "bin", "cache", "dist", "sessions", "tmp"}
SENSITIVE_NAMES = {
    "id_rsa",
    "id_dsa",
    "id_ecdsa",
    "id_ed25519",
    "hosts.yml",
    "auth.json",
}
RULES = {
    "GITHUB_TOKEN_VALUE": re.compile(rb"(?:gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,})"),
    "PRIVATE_KEY": re.compile(rb"-----BEGIN (?:RSA |OPENSSH |EC |DSA )?PRIVATE KEY-----"),
    "GITHUB_BASIC_AUTH_URL": re.compile(rb"https://[^\s/:@]+:[^\s@]+@github\.com", re.I),
    "GH_HOSTS_CREDENTIAL": re.compile(rb"^\s*(?:oauth_token|user)\s*:\s*[^\s#]+", re.I | re.M),
    "WINDOWS_USER_PATH": re.compile(rb"[A-Za-z]:\\Users\\[^\\\r\n]+", re.I),
    "PRIVATE_SSH_TARGET": re.compile(rb"(?:root|ubuntu|admin)@(?:10\.|192\.168\.|172\.(?:1[6-9]|2[0-9]|3[01])\.)"),
    "PRIVATE_HOST_IPV4": re.compile(rb"192\.168\.(?:[0-9]{1,3})\.(?:[0-9]{1,3})"),
}


def ignored(path: Path, root: Path) -> bool:
    parts = path.relative_to(root).parts
    return any(part in IGNORED_DIRS for part in parts[:-1])


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("root", nargs="?", default=Path(__file__).resolve().parents[1])
    args = parser.parse_args()
    root = Path(args.root).resolve()
    findings: list[dict[str, object]] = []
    scanned = 0
    binary = 0
    ignored_files = 0

    for path in sorted(root.rglob("*")):
        if not path.is_file() or path.is_symlink():
            continue
        if ignored(path, root):
            ignored_files += 1
            continue
        rel = path.relative_to(root).as_posix()
        lower = path.name.lower()
        if lower in SENSITIVE_NAMES or lower == ".env" or lower.startswith(".env.") or lower.endswith(".token"):
            findings.append({"path": rel, "line": 0, "rule": "SENSITIVE_FILE_NAME"})
            continue
        data = path.read_bytes()
        if b"\x00" in data[:8192]:
            binary += 1
            continue
        scanned += 1
        for rule, pattern in RULES.items():
            for match in pattern.finditer(data):
                line = data.count(b"\n", 0, match.start()) + 1
                findings.append({"path": rel, "line": line, "rule": rule})

    report = {
        "root": ".",
        "scanned_text_files": scanned,
        "binary_files_skipped": binary,
        "ignored_runtime_files": ignored_files,
        "findings": findings,
        "safe_to_publish": not findings,
    }
    print(json.dumps(report, indent=2, sort_keys=True))
    return 0 if not findings else 1


if __name__ == "__main__":
    raise SystemExit(main())
