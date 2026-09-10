# Alpha Feedback Iteration Report

Date: 2026-09-10

Status: `PASS`

External validation: `USER_VALIDATION_PENDING`

## Objective

This iteration continued product development without treating external-user replies as a
development blocker. It stayed within the existing Alpha architecture and focused on fresh
public failures, first-run diagnostics, binary installation and a copyable demonstration.

Frozen M1, M2, Direct URL acquisition semantics, Resolver, Replay Plan, ActExecutor, Matcher,
Docker daemon, docker0 and host networking were not redesigned or modified.

## Fresh Case D

Case D was registered from remote metadata and workflow content before any RunBack invocation:

- Repository: `opencitations/ramose`
- URL: https://github.com/opencitations/ramose/actions/runs/33990762256
- Attempt/job: `1` / `pyright` (`101372447310`)
- Runner: `ubuntu-latest`
- Failed command: `uv run pyright`
- Characteristics: setup-uv, cache, locked dependency sync, multiple run steps, no services,
  job container, local action or required secret

The first retained run ended truthfully as `REPLAY_BLOCKED` at `PREPARE` with
`NETWORK_DEPENDENCY` after a Git TLS disconnect. A full retry reached the intended Pyright
step and printed the same visible diagnostic, but neither side had a structured Pyright item,
so the result remained `INSUFFICIENT_EVIDENCE` with `0/0/0`.

The general fix added standard Pyright parsing and normalized RunBack's own short workspace
prefix. The unchanged URL then produced:

```text
Result: SAME_FAILURE
Evidence level: STRUCTURED
Remote failures: 1
Local failures:  1
Matched:         1
```

Identity:
`benchmarks/federation/benchmark.py:27:pyright[reportMissingImports]`

Message: `Import "scipy.stats" could not be resolved`

Measured CLI time was 52.114 seconds for the first retained result and 56.252 seconds for the
final result. The intermediate TLS and insufficient-evidence outcomes remain in the evidence
directory.

## Fresh Case E

Case E was registered before local replay:

- Repository: `AidenAI-IO/aiden-firmware`
- URL: https://github.com/AidenAI-IO/aiden-firmware/actions/runs/34079675283
- Attempt/job: `1` / `go-tests` (`101612452314`)
- Runner: `ubuntu-latest`
- Failed command: `go test ./...` from `src/agent`
- Characteristics: checkout, setup-go, ordinary run step, no services, job container, local
  action or required secret

The first invocation completed online acquisition and reached the intended test, but standard
Go test output had no structured parser. RunBack returned `INSUFFICIENT_EVIDENCE` with
`0/0/0` after 153.287 seconds.

The general Go parser now requires the same test name, `_test.go` source, line, exact message
and exit code. It does not promote unrelated warning or error-looking output. The unchanged URL
then produced:

```text
Result: SAME_FAILURE
Evidence level: TEST
Remote failures: 1
Local failures:  1
Matched:         1
```

Test: `TestAudioDialogPublishesStreamingReasoningForVoiceRun`

Source: `audio_dialog_test.go:1077`

Message: `published messages = []agent.Message(nil), want one streaming reasoning message`

Measured CLI time was 164.047 seconds. Case E's earlier presence in an outreach list was not
treated as external testing; no person reported running RunBack on it.

## Source changes

- `internal/failure/evidence.go`: standard Pyright and Go test evidence extraction
- `internal/fingerprint/fingerprint.go`: RunBack short-workspace prefix normalization
- `internal/doctor/doctor.go`: actionable first-run diagnostics
- `internal/doctor/disk_linux.go` and `disk_other.go`: portable free-space probe
- `scripts/install-release.sh`: checksum-verifying, no-Go, no-sudo release installation
- `README.md` and `README.zh-CN.md`: first-screen command, installation, demo and real-case
  explanation

No repository-name branch was added. No Matcher threshold or success condition changed.

## Tests added

- `internal/failure/pyright_test.go`
- `internal/failure/gotest_test.go`
- New doctor and workspace-normalization cases in the existing package tests

Mutation tests prove that changed Pyright or Go messages cannot produce `SAME_FAILURE`. An
unparsed Go failure remains insufficient.

## First-run and install behavior

The release installer:

- downloads a versioned Linux amd64 archive and `SHA256SUMS`;
- accepts curl or wget;
- verifies the exact archive checksum;
- extracts only the expected RunBack binary;
- installs atomically to `~/.local/bin` by default;
- supports an explicit `RUNBACK_INSTALL_DIR`;
- unsets GitHub token variables and never uses credentials;
- does not install or repair Docker, act, Git, firewall or networking.

`runback doctor` remains diagnostic only. It reports actionable blockers for Git, Docker,
Docker daemon access, act, GitHub API/quota, workspace permissions and disk space; PATH and
network conditions are reported as warnings where RunBack can still proceed. On the validation
host, the authenticated probe was ready with low-disk and PATH warnings. The anonymous probe
truthfully reported the exhausted anonymous GitHub quota and was not ready.

## Demo and discoverability

The README first screen states “Reproduce failed GitHub Actions runs locally” and shows
`runback <failed-run-url>`. It separates the retained Werkzeug Case C copyable demo from a
user's own failed run.

The public repository currently has:

- description: `Reproduce failed GitHub Actions runs locally.`
- topics: `act`, `ci`, `debugging`, `devtools`, `docker`, `github-actions`, `golang`,
  `reproducibility`
- GitHub Discussions enabled
- a bug-report form and public Alpha testing discussion

No screenshot or GIF was fabricated because no real visual capture was available.

## Regression evidence

The release candidate passed:

- `go test ./...`
- `go vet ./...`
- `go build ./cmd/runback`
- Flask bundle inspect
- Click bundle inspect
- M1 and M2 evidence SHA256 verification
- M2 step replay: `STEP_PASSED_UNVERIFIED`
- M2 full verify: `FULL_JOB_PASSED`
- Frozen Case C, exact URL and no replay overrides: `SAME_FAILURE`, TEST, `1/1/1`

Case C used cached online evidence, the generic default bridge path and no CLI image, network,
bundle, lock or work-directory override. No Docker daemon, docker0, host address, route,
firewall or service setting was changed.

## Release-candidate smoke

The exact `v0.1.1-alpha` Linux amd64 archive was extracted into a new temporary HOME and
validated before publication:

- archive checksum: PASS
- `runback version`: `RunBack v0.1.1-alpha`
- authenticated `runback doctor`: ready; low-disk and PATH warnings remained advisory
- exact Case C URL with no replay overrides: `SAME_FAILURE`, TEST, `1/1/1`
- Case C elapsed time through final result: 68.019 seconds
- network: `bridge` from `GENERIC_DEFAULT`

The temporary HOME and its session were removed after the smoke. Evidence is retained in
`docs/alpha/evidence/release-candidate/`.

## Public release

The validated candidate was published as:

- Release: https://github.com/huanglinfei091-cmd/runback/releases/tag/v0.1.1-alpha
- State: public prerelease, not a draft
- Tag commit: `fede635598ecdf2b8fe85fd1d57c64f14c7317df`
- Tagged-commit CI: https://github.com/huanglinfei091-cmd/runback/actions/runs/34426281644
- Archive: `runback-v0.1.1-alpha-linux-amd64.tar.gz`
- Archive SHA256:
  `15708d693c94bd26168dfd2798b41cd401f54ecb120e13fdf3c344e896b223bc`

The public one-command installer was then run in another clean temporary HOME. It downloaded
the released archive without Go or sudo, verified the public checksum, installed atomically,
reported `RunBack v0.1.1-alpha` and completed authenticated `runback doctor` successfully.
The temporary installation was deleted after verification.

## External-user status

The formal outreach audit retained 21 direct invitations, one explicit decline and zero
verified testers. No download, view, clone, reaction or silence was counted as a test. No
external install, invocation, verdict, TTFR, help request, `dev`, `replay --step` or `verify`
evidence has been reported. No feedback was simulated, and no new direct outreach was sent in
this iteration.

Status remains `USER_VALIDATION_PENDING` while development continues.

## Evidence

- `docs/alpha/CASE_D.json`
- `docs/alpha/CASE_E.json`
- `docs/alpha/evidence/case-d/`
- `docs/alpha/evidence/case-e/`
- `docs/alpha/evidence/install/`
- `docs/alpha/evidence/gates/`
- `docs/alpha/evidence/regressions/`
- `docs/alpha/evidence/release-candidate/`
- `docs/alpha/evidence/public-release/`
- `docs/compatibility/REAL_CASES.md`
- `docs/users/EXTERNAL_USER_REPORT.md`

## Known limitations

The retained cases cover Python mypy, pytest and Pyright plus one Go test job. They do not yet
provide a real JavaScript/TypeScript replay case. Private repositories, GitHub Enterprise,
Windows, macOS, self-hosted runners, services, job containers, repository-local actions,
reusable job workflows, dynamic matrices, secret-heavy paths and artifact-heavy workflows
remain outside the current Alpha scope.

The Alpha Feedback Iteration is complete and the implementation is released. External-user
validation remains a separate pending outcome and was not converted into a release claim.
