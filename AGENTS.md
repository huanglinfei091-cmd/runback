# RunBack contributor instructions

M1, M2 and Direct URL are frozen after real acceptance. Preserve their core semantics,
tests and authoritative evidence. Change frozen behavior only when a newly registered
case or a real external user proves a general defect. Never add repository-specific
branches or weaken Matcher requirements to make a case pass.

GitHub credentials may enter only online evidence acquisition. They must never be written
to files or passed to Git, act, Docker, workflows, dev/replay containers, sessions, locks,
caches, evidence, logs, or command lines. Keep anonymous public access as the fallback.

`runback doctor` is diagnostic only. Never repair docker0, restart Docker, modify daemon
configuration, change firewalls, or alter host networking. RunBack may create at most one
labeled managed fallback during an actual replay through the existing generic executor path.

Before submitting changes, run `gofmt`, `go test ./...`, `go vet ./...`, and
`go build ./cmd/runback`. Do not modify historical files under `docs/m1/evidence/` or
`docs/m2/evidence/`.
