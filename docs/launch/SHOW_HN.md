# Show HN: RunBack – reproduce a failed GitHub Actions run locally from its URL

RunBack is a Linux CLI that accepts a completed, failed public GitHub Actions run URL and tries
to reproduce the same failure locally.

```bash
runback https://github.com/owner/repo/actions/runs/123456789
```

It fetches the run metadata, exact attempt, jobs, workflow at the failed commit and the target
job log. It creates a replay plan, delegates workflow execution to `act`, and compares
structured remote and local failure evidence. A nonzero exit code is not enough: RunBack only
reports `SAME_FAILURE` when its matcher can support that conclusion.

I built it because debugging CI often becomes edit, push, wait, inspect, and repeat. The desired
loop is failed URL, local reproduction, edit, replay the failed step, and verify the full job.

The v0.1.0 alpha currently targets public GitHub.com repositories, Ubuntu jobs, and ordinary
JavaScript/TypeScript, Python and Go workflows. It has retained real-case evidence for Flask,
Click and a pre-registered Werkzeug Direct URL run. It does not claim general Actions
compatibility, and blocked or different failures are treated as valid results.

Linux amd64 release and source:
https://github.com/huanglinfei091-cmd/runback

I am looking for public failed-run URLs and honest results, including failed reproductions.
Please do not share tokens or private logs.
