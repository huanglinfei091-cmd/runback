---
title: From a failed GitHub Actions URL to the same failure locally
published: false
description: How RunBack reconstructs a failed CI run without pretending every local failure is a reproduction.
tags: github, devops, go, opensource
---

When a test passes locally and fails in GitHub Actions, the slow debugging loop is familiar:
change something, push, wait for CI, inspect the new failure, and repeat.

RunBack starts from the failed run URL:

```bash
runback https://github.com/owner/repo/actions/runs/123456789
```

The CLI retrieves the exact workflow-run attempt, all jobs, the workflow file at the failed
commit and the selected job log. It converts those inputs into canonical evidence, sends that
evidence through the same resolver and replay planner used by its offline bundle path, delegates
execution to `act`, and compares structured failure evidence.

## Why exit code 1 is not enough

Two commands can both exit with status 1 for unrelated reasons. A missing dependency, broken
network or invalid local path must not be reported as the remote test failure.

RunBack therefore distinguishes:

- `SAME_FAILURE`: structured evidence matches.
- `DIFFERENT_FAILURE`: the local run failed for another reason.
- `INSUFFICIENT_EVIDENCE`: both runs failed, but the evidence cannot prove identity.
- `REPLAY_BLOCKED`: execution did not reach a trustworthy comparison.
- `EVIDENCE_UNAVAILABLE`: the historical GitHub inputs could not be acquired.

## The real Direct URL path

The pre-registered Werkzeug acceptance case did not pass on the first attempt. Its retained
sequence was:

```text
long workspace path      → DIFFERENT_FAILURE
action download reset    → REPLAY_BLOCKED
unparsed pytest evidence → INSUFFICIENT_EVIDENCE
structured match         → SAME_FAILURE
```

The final result was:

```text
Remote failures: 1
Local failures:  1
Matched:         1
Result: SAME_FAILURE
```

The fixes were general: a shared short Linux workspace allocator and support for a normal
pytest `DID NOT RAISE` failure form. The matcher threshold and remote evidence were not changed.

## Security boundary

Optional GitHub credentials are restricted to evidence acquisition. They are removed from Git,
`act`, Docker, workflow, replay/dev containers, sessions, locks, caches, evidence, logs and
command lines. Signed job-log downloads use a separate client without authorization headers or
cookies.

## Try it

RunBack v0.1.0-alpha currently supports Linux amd64 and focuses on completed failed public
GitHub.com runs, Ubuntu jobs, and common Python, Go and JavaScript/TypeScript workflows.

Source, release and retained evidence:
https://github.com/huanglinfei091-cmd/runback

If you try a public failed run, share the verdict, Stage/Cause and installation friction. A
blocked or different failure is useful evidence. Never share tokens, secrets or private logs.
