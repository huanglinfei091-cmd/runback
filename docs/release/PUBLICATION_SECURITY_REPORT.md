# Publication Security Report

Date: 2026-09-07
Release candidate: `v0.1.0-alpha`

## Scope

The scan covered the repository publication set, release scripts, documentation, test
fixtures and committed evidence. Runtime directories such as `.git`, `.runback`, `bin`,
`cache`, `dist`, `sessions` and `tmp` are excluded from publication and ignored by Git.

The review checked for:

- GitHub token formats and exact presence of the currently authenticated token
- private keys and GitHub CLI credential material
- credentials embedded in URLs
- `.env`, token and runtime state files
- Windows user-profile paths, private SSH targets and private LAN addresses
- accidental inclusion of cache, session, lock, worktree or compiled runtime data

## Findings and changes

The initial scan found one private LAN SSH target in `docs/m2/REPORT.md`. A second manual
review found two copies of the same private host address in the preserved Case C action-fetch
failure log. These values were replaced with descriptive placeholders. The command outcome,
failure classification, timestamps, remote GitHub addresses and Case C result chain were not
changed.

Because `docs/m2/REPORT.md` is part of the frozen M2 evidence set, its entry in
`docs/m2/SHA256SUMS` was updated to the checksum of the redacted report. Every other M1/M2
file and checksum remained unchanged. Both checksum manifests pass after redaction.

The repository also received publication-safe defaults:

- `.gitignore` excludes RunBack runtime state, release output, local binaries, `.env` files
  and token-shaped files.
- `AGENTS.md`, `CONTRIBUTING.md`, `scripts/sync.ps1` and benchmark defaults no longer contain
  developer-machine paths or a private host address.
- `scripts/build.sh`, `scripts/install.sh` and `scripts/release.sh` remove GitHub token
  variables before invoking build tools.
- The issue form explicitly tells reporters never to submit tokens or secrets.

## Final verification

The pre-publication scan covered 218 text files. After adding the two external-user reports,
`scripts/publication-scan.py` scanned the final 220 publication text files and returned:

```text
safe_to_publish: true
findings: 0
```

An exact in-memory comparison against the Ubuntu account's current `gh auth token` scanned
38,762 files across the development checkout and RunBack runtime roots. A second exact scan
against the authenticated Windows control account covered all 220 tracked and pending
publication files. They returned:

```text
exact_token_matches=0
exact_current_token_scan_files=220
exact_current_token_matches=0
```

The token value was never printed, written to a file or passed on a command line. Unit and
process-boundary tests also confirm that GitHub credentials are removed from ActExecutor,
Docker, replay, dev and session execution environments. Signed job-log downloads use a
separate credential-free HTTP client with no cookie jar.

## Integrity results

| Gate | Result |
| --- | --- |
| Publication pattern scan | PASS, 220 text files, 0 findings |
| Exact authenticated-token scans | PASS, Ubuntu 38,762 files and Windows publication set, 0 matches |
| M1 `SHA256SUMS` | PASS |
| M2 `SHA256SUMS` | PASS |
| Release artifact checksum after Windows transfer | PASS |

No GitHub credential, GitHub CLI credential store, private key, RunBack cache, session,
lock, worktree or development binary is part of the Git commit or release archive.

The public repository, tagged source, README and release assets were inspected after
publication. The repository is public, the release is a prerelease rather than a draft, and
the public asset digest matches the locally recorded archive SHA256.
