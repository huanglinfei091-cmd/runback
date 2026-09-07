# Contributing

Start with a public failed-run URL and describe the observed failure.
Do not submit secrets or private logs. Prefer a small regression test that fails before
the change and covers the actual problem, not a test that merely repeats the implementation.

Run go test ./..., go vet ./... and gofmt before submitting changes.
For replay changes, include remote and local failure signatures, the selected step,
commit, run attempt and the act/image versions. Explain missing context explicitly.

Never label an infrastructure failure as a reproduction of an application failure.
Use the benchmark categories and record unsupported and unknown cases.
