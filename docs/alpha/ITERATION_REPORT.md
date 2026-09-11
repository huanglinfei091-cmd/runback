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

## Fresh JavaScript/TypeScript Cases F and G

Case F was registered before local execution from
`DeHubToken/dehub-mobile` run `34426920635`, attempt 1, job `Typecheck & Test`. The exact URL
completed authenticated acquisition, Node 20 setup, npm dependency installation and the target
`npm run i18n:coverage` step. Remote and local logs visibly agreed on the missing
`settings.display` translation key, but that output came from a custom repository script and
provided no supported structured identity. RunBack retained `INSUFFICIENT_EVIDENCE`, `STEP`,
`0/0/0` after 260.163 seconds. No repository-specific parser was added and the case was not
replaced.

Case G was separately preregistered and committed before acquisition:

- Repository: `ClickHouse/click-ui`
- URL: https://github.com/ClickHouse/click-ui/actions/runs/34430830779
- Attempt/job: `1` / `build` (`102725796825`)
- Runner: `ubuntu-latest`
- Failed command: `yarn build`
- Characteristics: checkout, setup-node 24.x, Corepack, immutable Yarn 4 install and a standard
  TypeScript library build; no services, job container, local action or required secret

The first exact URL invocation reproduced three standard TypeScript compiler diagnostics but
returned `INSUFFICIENT_EVIDENCE`, `STEP`, `0/0/0` after 129.154 seconds because no `tsc`
parser existed. A setup-node tool-download request independently reached an anonymous API rate
limit, then setup-node's normal direct Node.js download fallback completed; the target step was
reached and no acquisition token entered execution.

The general TypeScript parser requires an exact source file, line, column, `TS` diagnostic
code, complete message and exit code. Unsupported partial `error TS...` formats are counted as
unparsed failures, so partial parsing cannot become a success. Replaying the unchanged URL then
produced:

```text
Result: SAME_FAILURE
Evidence level: STRUCTURED
Remote failures: 3
Local failures:  3
Matched:         3
```

All three identities were in
`src/components/DatePicker/Common.tsx:1010` (`TS2551`, `TS7031`, `TS7031`). Final TTFR was
117.844 seconds. A verified session was created; the debug entry remained
`DEBUG_UNSUPPORTED` because Node development environments are outside the current narrow M2
debug scope.

## Fresh Vitest Case H

Case H was preregistered and committed before acquisition:

- Repository: `janosh/svelte-widgets`
- URL: https://github.com/janosh/svelte-widgets/actions/runs/34449963179
- Attempt/job: `1` / `unit` (`102783221808`)
- Runner: `ubuntu-latest`
- Failed command: `npm run test:coverage`
- Characteristics: checkout, setup-node 24, npm and Vitest 5; no services, job container,
  local action or required secret

The first exact URL invocation reproduced four visible Vitest failures but returned
`INSUFFICIENT_EVIDENCE`, `STEP`, `0/0/0` after 184.196 seconds because no Vitest parser
existed. After the parser change, one retry stopped during dependency installation on a
transient npm optional native binding error. That attempt is retained as
`REPLAY_BLOCKED / EXECUTE / UNKNOWN`, not presented as a successful reproduction.

The generic Vitest parser requires a complete failure block with the same source file, line,
column, hierarchical test name, exception, complete message and exit code. Incomplete,
unsupported and duplicate identities are counted as unparsed failures. The unchanged URL and
job then produced:

```text
Result: SAME_FAILURE
Evidence level: TEST
Remote failures: 4
Local failures:  4
Matched:         4
```

Final TTFR was 244.421 seconds from a cache hit. A verified session was created; Node debug
remained `DEBUG_UNSUPPORTED`. A scan of all retained Case H evidence found zero exact matches
for the acquisition token.

## Source changes

- `internal/failure/evidence.go`: standard Pyright, Go test, TypeScript compiler and Vitest evidence extraction
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
- `internal/failure/typescript_test.go`
- `internal/failure/vitest_test.go`
- New doctor and workspace-normalization cases in the existing package tests

Mutation tests prove that changed Pyright, Go, TypeScript or Vitest source identity and messages
cannot produce `SAME_FAILURE`. Unparsed Go, partial TypeScript and incomplete or duplicate
Vitest failure sets remain insufficient.

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

After the TypeScript change, the `v0.1.2-alpha` candidate repeated all gates. `gofmt`,
`go test ./...`, `go vet ./...` and `go build ./cmd/runback` passed on the Linux target;
Flask/Click bundle checks and M1/M2 hashes passed; M2 again returned
`STEP_PASSED_UNVERIFIED` then `FULL_JOB_PASSED`. Frozen Case C again used only its exact URL
and returned `SAME_FAILURE`, `TEST`, `1/1/1` in 78.177 seconds. Evidence is retained in
`docs/alpha/evidence/gates-v0.1.2/` and `docs/alpha/evidence/regressions-v0.1.2/`.

Case C used cached online evidence, the generic default bridge path and no CLI image, network,
bundle, lock or work-directory override. No Docker daemon, docker0, host address, route,
firewall or service setting was changed.

After the Vitest change, `gofmt`, `go test ./...`, `go vet ./...` and
`go build ./cmd/runback` passed on the Linux target. The Case H run itself used only its URL
and explicit job selection, with no bundle, lock, work-directory, image or network override.
No frozen acquisition, replay or Matcher semantics changed.

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

### v0.1.2-alpha TypeScript follow-up

The TypeScript compatibility change was published as a separate public prerelease:

- Release: https://github.com/huanglinfei091-cmd/runback/releases/tag/v0.1.2-alpha
- State: public prerelease, not a draft
- Tag commit: `dce0f5d6421920d296ca34de0fbe00f322eba5af`
- Tagged-commit CI: https://github.com/huanglinfei091-cmd/runback/actions/runs/34447304256
- Archive: `runback-v0.1.2-alpha-linux-amd64.tar.gz`
- Archive size: 3,329,606 bytes
- Archive SHA256:
  `d78abad886ebe4bdd0b4338a218d27c1fca3d07ebee2bf5cfcf699e3ba0ecb0e`

The exact archive binary reported `RunBack v0.1.2-alpha`, passed authenticated doctor and
replayed frozen Case C from the exact URL with no replay overrides. It returned
`SAME_FAILURE`, `TEST`, `1/1/1`; total CLI time through session creation was 86 seconds in the
isolated release smoke. A second clean temporary HOME used the public installer URL after
publication; download, public checksum validation, atomic install, version and authenticated
doctor all returned exit code 0. Both temporary homes were deleted after validation.

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
- `docs/alpha/CASE_F.json`
- `docs/alpha/CASE_G.json`
- `docs/alpha/CASE_H.json`
- `docs/alpha/evidence/case-d/`
- `docs/alpha/evidence/case-e/`
- `docs/alpha/evidence/case-f/`
- `docs/alpha/evidence/case-g/`
- `docs/alpha/evidence/case-h/`
- `docs/alpha/evidence/gates-v0.1.2/`
- `docs/alpha/evidence/regressions-v0.1.2/`
- `docs/alpha/evidence/install/`
- `docs/alpha/evidence/gates/`
- `docs/alpha/evidence/regressions/`
- `docs/alpha/evidence/release-candidate/`
- `docs/alpha/evidence/public-release/`
- `docs/alpha/evidence/release-v0.1.2-case-c.log`
- `docs/alpha/evidence/release-v0.1.2-checksum.log`
- `docs/alpha/evidence/release-v0.1.2-doctor.log`
- `docs/alpha/evidence/release-v0.1.2-smoke.meta`
- `docs/alpha/evidence/public-v0.1.2-install.log`
- `docs/alpha/evidence/public-v0.1.2-doctor.log`
- `docs/alpha/evidence/public-v0.1.2-smoke.meta`
- `docs/compatibility/REAL_CASES.md`
- `docs/users/EXTERNAL_USER_REPORT.md`

## Known limitations

The retained cases cover Python mypy, pytest and Pyright, one Go test job, a custom JavaScript
failure that remained insufficient, a standard TypeScript compiler failure and a Vitest unit
test failure that reached strict `SAME_FAILURE`. This does not establish general
JavaScript/TypeScript compatibility.
Private repositories, GitHub Enterprise,
Windows, macOS, self-hosted runners, services, job containers, repository-local actions,
reusable job workflows, dynamic matrices, secret-heavy paths and artifact-heavy workflows
remain outside the current Alpha scope.

The Alpha Feedback Iteration is complete and the implementation is released. External-user
validation remains a separate pending outcome and was not converted into a release claim.
