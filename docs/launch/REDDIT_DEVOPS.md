# I built an alpha CLI that replays a failed GitHub Actions run from its URL

RunBack starts from the thing you already have when CI fails: the run URL.

```bash
runback https://github.com/owner/repo/actions/runs/123456789
```

It acquires the exact run attempt, failed commit workflow, jobs and job log, builds a local
replay plan, runs it with `act`, and compares the remote and local failure evidence. It reports
`SAME_FAILURE` only when structured evidence matches; otherwise it keeps the truthful
`DIFFERENT_FAILURE`, `INSUFFICIENT_EVIDENCE` or `REPLAY_BLOCKED` result.

The current `v0.1.0-alpha` is Linux amd64 and focuses on completed failed runs from public
GitHub.com repositories, Ubuntu jobs, and ordinary Python, Go and JavaScript/TypeScript
workflows. Services, job containers, private repositories, secrets and non-Linux runners are
outside the current scope.

One pre-registered Werkzeug case reproduced remote 1 / local 1 / matched 1. Flask and Click
cases cover the bundle and debugging-session paths. The intermediate failures are retained in
the repository rather than hidden.

Project and binary: https://github.com/huanglinfei091-cmd/runback

If you have a public failed Actions run, I would value the actual verdict and installation
friction. Failed reproduction is useful feedback. Please never post credentials or private
logs.
