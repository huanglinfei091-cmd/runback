# RunBack

**Reproduce failed GitHub Actions runs locally.**

Stop pushing commits just to debug CI.

```bash
runback https://github.com/owner/repo/actions/runs/123456789
```

```text
GitHub Actions failure
        ↓
runback <URL>
        ↓
SAME_FAILURE locally
        ↓
runback dev
        ↓
edit
        ↓
runback replay --step
        ↓
runback verify
```

[中文说明](README.zh-CN.md) · [Architecture](docs/architecture.md) ·
[Known limitations](docs/limitations.md) · [Direct URL evidence](docs/online/DIRECT_URL_REPORT.md)

## What is RunBack?

RunBack accepts one completed, failed GitHub Actions run URL. It acquires the historical
workflow, commit, failed job, matrix, runner and job log, creates a local replay plan,
delegates execution to [act](https://github.com/nektos/act), and compares the remote and
local failures.

RunBack is not an act replacement. act runs workflows; RunBack determines how to replay
this particular failure and refuses to claim success when the evidence does not match.

## Quick start

Download `runback-v0.1.0-alpha-linux-amd64.tar.gz` and `SHA256SUMS` from the
[v0.1.0-alpha release](https://github.com/huanglinfei091-cmd/runback/releases/tag/v0.1.0-alpha), then:

```bash
sha256sum -c SHA256SUMS
tar -xzf runback-v0.1.0-alpha-linux-amd64.tar.gz
mkdir -p "$HOME/.local/bin"
cp runback-v0.1.0-alpha-linux-amd64/runback "$HOME/.local/bin/runback"
"$HOME/.local/bin/runback" version
"$HOME/.local/bin/runback" doctor
```

Optional system-wide installation:

```bash
sudo install -m 0755 runback-v0.1.0-alpha-linux-amd64/runback /usr/local/bin/runback
```

Run a real public failure:

```bash
runback https://github.com/pallets/werkzeug/actions/runs/32448268750
```

If RunBack reports `SAME_FAILURE`, it creates a debugging session and shows its workspace:

```bash
runback dev
# edit the checked-out source
runback replay --step
runback verify
```

## Requirements

- Linux amd64. Ubuntu-hosted GitHub Actions jobs are the validated target.
- Git.
- Docker Engine with a running daemon.
- [act](https://nektosact.com/installation/index.html). Alpha validation used act 0.2.89.
- Network access to GitHub and action dependencies.

Go 1.24 or newer is needed only to build from source. From a source checkout:

```bash
bash scripts/install.sh
$HOME/.local/bin/runback doctor
```

The installer writes only to a user-owned directory and never installs or repairs Docker,
act, Git, the firewall, Docker daemon, docker0, or host networking.

## `runback doctor`

`doctor` checks Git, Docker, the daemon, act, GitHub API access, the short replay workspace
and Docker networking. It is diagnostic only and does not repair the host.

```text
RunBack Doctor

✓ Git
✓ Docker
✓ act
✓ Docker daemon
✓ GitHub API (authenticated, quota remaining)
✓ Replay workspace (/tmp/rb/<short-id>)
✓ Docker networking

Ready to reproduce public GitHub Actions failures.
```

Public repositories can use anonymous GitHub API access. Without a token, `doctor` warns
that the quota is lower. Optional credential priority is:

```text
RUNBACK_GITHUB_TOKEN > GH_TOKEN > anonymous
```

An existing GitHub CLI login is not read automatically by RunBack.

## Direct URL behavior

```bash
# Acquire evidence and replay the first supported failed Ubuntu job.
runback https://github.com/OWNER/REPO/actions/runs/RUN_ID

# Inspect and resolve without executing repository code.
runback inspect https://github.com/OWNER/REPO/actions/runs/RUN_ID

# Reacquire all online evidence, including a fresh job-log redirect.
runback https://github.com/OWNER/REPO/actions/runs/RUN_ID --refresh

# Force an offline evidence bundle instead of online acquisition.
runback https://github.com/OWNER/REPO/actions/runs/RUN_ID --bundle evidence.json
```

RunBack also supports explicit `--job`, `--image`, `--network`, `--lock` and `--work-dir`
overrides. Normal Direct URL usage does not require them.

## Real validated cases

| Case | Path validated | Truthful result |
| --- | --- | --- |
| Flask | Bundle acquisition → resolver → full replay → matcher | `SAME_FAILURE` |
| Click | Bundle replay → session → dev → step replay → full verify | `SAME_FAILURE`, then `FULL_JOB_PASSED` after the test fix |
| Werkzeug Case C | Authenticated Direct URL → online evidence → full replay → matcher | `Remote 1 / Local 1 / Matched 1`, `SAME_FAILURE` |

Werkzeug Case C was selected before local replay and was not replaced when intermediate
runs produced `DIFFERENT_FAILURE`, `REPLAY_BLOCKED` and `INSUFFICIENT_EVIDENCE`. The
complete chain is retained in [the report](docs/online/DIRECT_URL_REPORT.md).

## Compatibility

The alpha focuses on public GitHub.com repositories, completed failed runs,
`ubuntu-latest`, and ordinary JavaScript/TypeScript, Python and Go workflows. It supports
static matrices and standard checkout/setup/run steps that act can execute.

Windows, macOS and self-hosted runners, private repositories, GitHub Enterprise, reusable
job workflows, dynamic matrices, services, job containers, repository-local actions,
secret-dependent jobs and artifact-heavy workflows are outside the current scope.

## Result meanings

- `SAME_FAILURE`: structured remote and local evidence meets the Matcher standard.
- `DIFFERENT_FAILURE`: the local replay failed differently.
- `INSUFFICIENT_EVIDENCE`: RunBack cannot prove the failures are the same.
- `REPLAY_BLOCKED`: preparation or execution could not reach a trustworthy comparison.
- `EVIDENCE_UNAVAILABLE`: required GitHub evidence could not be acquired.

A nonzero exit code alone is never treated as a reproduction.

## Security model

GitHub credentials are limited to online evidence acquisition. RunBack removes them from
Git, act, Docker, workflow, dev/replay containers, sessions, locks, caches, evidence, logs
and command lines. Signed job-log redirects are downloaded by a separate client without
authorization headers or cookies.

RunBack uses empty secret, variable, environment and input files for act. It does not
recover GitHub Secrets or pretend that missing context exists. Review third-party workflow
code before executing any CI job locally.

## Report bugs

Use the [Bug report form](https://github.com/huanglinfei091-cmd/runback/issues/new?template=bug_report.yml)
and include the RunBack version, OS, Docker and act versions, failed run URL, result,
Stage and Cause. **Do not include tokens, secrets, cookies, private source or credentials.**

## Project status

RunBack is an early alpha with three retained real-case evidence chains. Compatibility is
intentionally narrow, and no 100% GitHub Actions compatibility claim is made. See
[Contributing](CONTRIBUTING.md) before adding a regression case.

```bash
go test ./...
go vet ./...
go build ./cmd/runback
```

MIT licensed. Third-party repositories and action dependencies retain their own licenses.
