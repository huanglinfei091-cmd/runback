# First External User Protocol

## Material given to the user

- The public RunBack README.
- One install command from the README.
- One public, completed, failed GitHub Actions run URL.

Do not explain Resolver, Replay Plan, ActExecutor, Matcher or internal architecture before
the trial. Help only when the user reaches a concrete blocker, and record that intervention.

## Evidence to record

- User and project identifier, with permission to quote any feedback.
- Start time and time to a working `runback doctor` result.
- Installation blocker, if any: Git, Docker, daemon, act, API quota, or RunBack itself.
- Time from the first URL command to its truthful final result.
- Whether the user understands `SAME_FAILURE` or the reported blocked/different result.
- Whether the user naturally tries `runback dev`, `runback replay --step`, and `runback verify`.
- Whether the workflow reduces a real push-and-wait CI cycle.
- Exact RunBack version, OS, Docker version, act version, URL and final status.

## Acceptance boundary

A local clean-machine test is not an external-user result. A maintainer reply without a
completed attempt is outreach evidence only. Failed installation is retained as product
evidence and is never rewritten as a reproduction result.
