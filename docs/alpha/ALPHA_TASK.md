# Alpha Readiness Task

Direct URL, M1 and M2 are frozen as PASS. Their core semantics and authoritative evidence
must remain unchanged unless Fresh Case D or an external user exposes a general defect.

This stage has three ordered deliverables:

1. Provide a minimal source installer and a read-only `runback doctor` covering Git,
   Docker CLI/daemon, act, GitHub API mode/quota, replay workspace and Docker networking.
2. Select Fresh Case D from remote static evidence, preregister it before local replay,
   execute the exact zero-override Direct URL command and retain every result honestly.
3. Give one unfamiliar user only the README, install command and one failed run URL, then
   record installation time, first-run time, comprehension and whether RunBack reduced CI pushes.

Doctor may inspect existing resources and run disposable probes. It must not install or
repair dependencies, modify docker0, change daemon configuration, restart Docker, change
host networking, or create a managed network. Tokens stay inside `internal/online` and
must never enter subprocesses, workflow execution, Docker, cache, sessions or reports.

Fresh Case D is immutable after preregistration. `SAME_FAILURE`, `DIFFERENT_FAILURE`,
`INSUFFICIENT_EVIDENCE` and `REPLAY_BLOCKED` are all valid experimental outcomes. The case
must not be replaced, the Matcher threshold must not be weakened, and repository-specific
branches are forbidden.

No `v0.1.0-alpha` release is authorized by this task. Release readiness is decided only
after the three deliverables have real evidence.
