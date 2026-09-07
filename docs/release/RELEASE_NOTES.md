# RunBack v0.1.0-alpha

RunBack turns a completed, failed public GitHub Actions run URL into a local replay and
compares the remote and local failure evidence.

## Validated scope

- Linux amd64 host with Git, Docker Engine and act.
- Public GitHub.com repositories and completed failed runs.
- Ubuntu-hosted jobs with ordinary checkout, setup and run steps.
- Flask and Click bundle cases, plus Werkzeug authenticated Direct URL Case C.
- Case C result: Remote failures 1, Local failures 1, Matched 1, `SAME_FAILURE`.
- `SAME_FAILURE → runback dev → replay --step → verify` debugging loop.

## Install

Download the tarball and `SHA256SUMS`, verify it, then copy the binary to a directory on PATH:

```bash
sha256sum -c SHA256SUMS
tar -xzf runback-v0.1.0-alpha-linux-amd64.tar.gz
mkdir -p "$HOME/.local/bin"
cp runback-v0.1.0-alpha-linux-amd64/runback "$HOME/.local/bin/runback"
runback version
runback doctor
```

## Known limitations

This is an early alpha. Windows, macOS and self-hosted runners, private repositories,
GitHub Enterprise, services, job containers, dynamic matrices, reusable workflows,
repository-local actions, secret-dependent jobs and artifact-heavy workflows are outside
the current scope. act uses a container approximation of GitHub-hosted runners.

`DIFFERENT_FAILURE`, `INSUFFICIENT_EVIDENCE`, `REPLAY_BLOCKED` and
`EVIDENCE_UNAVAILABLE` are truthful outcomes; RunBack does not convert them into success.

Report problems with the repository Bug report form. Include the failed run URL, RunBack,
Docker and act versions, Result, Stage and Cause. Do not include tokens or secrets.
