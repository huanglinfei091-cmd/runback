# RunBack v0.1.3-alpha — try it on a failed GitHub Actions run

RunBack takes the URL of a completed, failed public GitHub Actions run, reconstructs its
historical workflow, job, matrix, runner and failed-step context, replays it locally with
`act`, and compares the remote and local failure evidence.

```bash
runback https://github.com/owner/repo/actions/runs/123456789
```

The goal is to replace repeated edit → push → wait → fail cycles with a local debugging loop:

```text
failed run URL → local replay → SAME_FAILURE → edit → replay --step → verify
```

## Try the alpha

Linux amd64 users can install the current release without Go or sudo:

```bash
curl -fsSL https://raw.githubusercontent.com/huanglinfei091-cmd/runback/main/scripts/install-release.sh | bash
~/.local/bin/runback doctor
~/.local/bin/runback https://github.com/OWNER/REPO/actions/runs/RUN_ID
```

Release and checksums:
https://github.com/huanglinfei091-cmd/runback/releases/tag/v0.1.3-alpha

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

For blocked cases, include `Stage` and `Cause`. Installation friction and unsuccessful reproduction
are useful evidence. Please do **not** post tokens, secrets, cookies, private source, complete
private logs or credentials.

The retained real cases cover Python mypy/pytest/Pyright, Go test, TypeScript compiler and
Vitest failures. The preregistered Vitest Case H initially remained
`INSUFFICIENT_EVIDENCE`; after a generic strict parser was added, the unchanged run produced
remote/local/matched `4/4/4` and `SAME_FAILURE`. These are project-owned evidence chains, not
external-user validation or a claim of complete GitHub Actions support:
https://github.com/huanglinfei091-cmd/runback/blob/main/docs/compatibility/REAL_CASES.md

Repository: https://github.com/huanglinfei091-cmd/runback
