# Architecture

The resolver is the product. The executor is a dependency.

1. `internal/github` validates a GitHub run URL, fetches the selected run attempt,
   paginates its jobs and reads historical workflow contents. The same Source interface
   accepts a copied public evidence bundle.
2. `internal/workflow` enumerates static matrix combinations and maps the remote job name
   to exactly one workflow job and combination. Ambiguity is an error.
3. `internal/resolver` selects the lowest-ID supported failed job, maps its failed step,
   isolates that step's logs, recovers the checkout SHA from checkout's git-log evidence,
   and records blockers and unknowns.
4. `internal/lockfile` persists the frozen context. The lock does not contain tokens.
5. `internal/replay` retains the original selected job, conditions and commands, replacing
   its matrix with a one-row include matrix. No new workflow interpreter is implemented.
6. `internal/runner` prepares a detached checkout, invokes act with explicit paths and
   isolated configuration, saves JSON logs/results and retains the container for shell.
7. `internal/fingerprint` compares distinctive errors from the selected failed step.

The generated event is a partial reconstruction. The original event payload is not
available from the run REST endpoint. Missing context must remain visible.

External action refs and tool version ranges are recorded as requested; mutable refs
are not described as immutable. The actual local image ID is recorded after replay.

Public evidence export happens in Windows. Replay happens in Ubuntu. The exporter
fetches metadata/logs through gh, not through a token copied to Ubuntu.
