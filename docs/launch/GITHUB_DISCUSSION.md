# RunBack v0.1.0-alpha — try it on a failed GitHub Actions run

RunBack takes the URL of a completed, failed public GitHub Actions run, reconstructs its
workflow/job/matrix/runner context, replays it locally with `act`, and compares the remote and
local failure evidence.

```bash
runback https://github.com/owner/repo/actions/runs/123456789
```

The goal is to replace repeated edit → push → wait → fail cycles with a local debugging loop:

```text
failed run URL → local replay → SAME_FAILURE → edit → replay --step → verify
```

## Try the alpha

Download the Linux amd64 binary and `SHA256SUMS` from the
[v0.1.0-alpha release](https://github.com/huanglinfei091-cmd/runback/releases/tag/v0.1.0-alpha),
then run:

```bash
sha256sum -c SHA256SUMS
tar -xzf runback-v0.1.0-alpha-linux-amd64.tar.gz
mkdir -p "$HOME/.local/bin"
cp runback-v0.1.0-alpha-linux-amd64/runback "$HOME/.local/bin/runback"

runback doctor
runback https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

The current alpha targets public GitHub.com repositories, completed failed runs, Ubuntu jobs,
and ordinary JavaScript/TypeScript, Python and Go workflows. Private repositories, services,
job containers, repository-local actions, secret-dependent jobs, Windows/macOS runners and
GitHub Enterprise are outside the validated scope.

## What useful feedback looks like

Reply with a public failed run URL and the observed RunBack result:

- `SAME_FAILURE`
- `DIFFERENT_FAILURE`
- `INSUFFICIENT_EVIDENCE`
- `REPLAY_BLOCKED`
- `EVIDENCE_UNAVAILABLE`

For blocked cases, include `Stage` and `Cause`. Installation friction and failed reproduction
are useful evidence. Please do **not** post tokens, secrets, cookies, private source, complete
private logs or credentials.

The current evidence includes Flask, Click and one pre-registered Werkzeug Direct URL case.
For Werkzeug, RunBack reported remote 1 / local 1 / matched 1 and `SAME_FAILURE`. This is an
early alpha with deliberately narrow compatibility, not a claim of complete GitHub Actions
support.

Repository: https://github.com/huanglinfei091-cmd/runback
