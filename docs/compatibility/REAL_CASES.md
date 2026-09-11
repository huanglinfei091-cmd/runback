# Real Compatibility Cases

Last updated: 2026-09-11

This file records retained executions of real public GitHub Actions failures. It is not a
compatibility percentage: eight selected cases cannot establish a population success rate.
`SAME_FAILURE` appears only when the existing Matcher accepts complete structured evidence.

| Case | Repository | Runtime and workflow | First retained verdict | Final retained verdict | Generic issue and change | Measured TTFR |
| --- | --- | --- | --- | --- | --- | --- |
| M1 A | `pallets/flask` | Python 3.14, setup-python, mypy | `SAME_FAILURE`, 1/1/1 | `SAME_FAILURE`, 1/1/1 | Baseline structured mypy evidence | 85.58 s |
| M1/M2 B | `pallets/click` | Python 3.13 matrix, setup-uv, setup-python, tox/pytest | `DIFFERENT_FAILURE`, remote 1 / local 25 / matched 1 | `SAME_FAILURE`, then `FULL_JOB_PASSED` after a source fix | Missing `less` produced 24 extra failures; frozen validation used an explicit recorded image override and kept the failure-count standard | 42.72 s original M1; warm step replay median 5.459 s |
| Direct URL C | `pallets/werkzeug` | Python 3.9, setup-uv, pytest | `DIFFERENT_FAILURE` | `SAME_FAILURE`, 1/1/1 | Short workspace allocator and a general pytest `DID NOT RAISE` parser; intermediate action-fetch and insufficient-evidence results retained | 41.564 s to cold online result; 48.131 s through session creation |
| Fresh D | `opencitations/ramose` | Python, setup-uv cache, `uv sync`, Pyright | `REPLAY_BLOCKED / PREPARE / NETWORK_DEPENDENCY` | `SAME_FAILURE`, 1/1/1, `STRUCTURED` | Added standard Pyright diagnostics and normalized RunBack-owned short workspace prefixes; no Matcher threshold change | 52.114 s first; 56.252 s final |
| Fresh E | `AidenAI-IO/aiden-firmware` | Go 1.26.7, setup-go, `go test ./...` | `INSUFFICIENT_EVIDENCE`, 0/0/0 | `SAME_FAILURE`, 1/1/1, `TEST` | Added Go test name/source/message parsing; unrelated error-looking test logs remain outside the structured identity | 153.287 s first; 164.047 s final |
| Fresh F | `DeHubToken/dehub-mobile` | Node 20, npm cache, custom i18n check | `INSUFFICIENT_EVIDENCE`, 0/0/0 | `INSUFFICIENT_EVIDENCE`, 0/0/0, `STEP` | The custom script produced the same visible diagnostic, but no repository-specific parser was added | 260.163 s |
| Fresh G | `ClickHouse/click-ui` | Node 24, Yarn 4, TypeScript library build | `INSUFFICIENT_EVIDENCE`, 0/0/0 | `SAME_FAILURE`, 3/3/3, `STRUCTURED` | Added strict standard `tsc` diagnostics with file, line, column, TS code and complete message | 129.154 s first; 117.844 s final |
| Fresh H | `janosh/svelte-widgets` | Node 24, npm, Vitest 5 unit tests | `INSUFFICIENT_EVIDENCE`, 0/0/0 | `SAME_FAILURE`, 4/4/4, `TEST` | Added strict Vitest failure blocks with file, line, column, hierarchical test name, exception and complete message | 184.196 s first; 244.421 s final |

## Fresh Case D

- URL: https://github.com/opencitations/ramose/actions/runs/33990762256
- Attempt/job: `1`, `pyright` (`101372447310`)
- Failed step: `Pyright`; command: `uv run pyright`
- Run and checkout SHA: `d243662f719ebd630625d7f8f767a7f1c84bf4a6`
- Structured identity: `benchmarks/federation/benchmark.py:27:pyright[reportMissingImports]`
- Message: `Import "scipy.stats" could not be resolved`

The case was registered before any RunBack invocation. The first run stopped during Git
preparation after a TLS disconnect. The next full replay reached the intended Pyright step
and produced the same visible diagnostic, but the parser had no Pyright item and the local
line retained `/tmp/rb/<short-id>/workspace/`; the truthful result was
`INSUFFICIENT_EVIDENCE`. A later retry also retained another transient preparation failure.
After the two general parsing changes, the same URL produced one remote item, one local item,
one match and exit code 1 on both sides. A verified session was then created.

Evidence:

- `docs/alpha/CASE_D.json`
- `docs/alpha/evidence/case-d/first-run.log`
- `docs/alpha/evidence/case-d/retry-1.log`
- `docs/alpha/evidence/case-d/retry-2.log`
- `docs/alpha/evidence/case-d/retry-3.log`

## Fresh Case E

- URL: https://github.com/AidenAI-IO/aiden-firmware/actions/runs/34079675283
- Attempt/job: `1`, `go-tests` (`101612452314`)
- Failed step: `Run Go unit tests`; command: `go test ./...` from `src/agent`
- API head SHA: `81983e688175e052c5606d90ed1010693979a9c4`
- Historical checkout SHA extracted from the original job log:
  `e7be449bdaf181dbee75002a0d2ca8fe510a2674`
- Test identity: `TestAudioDialogPublishesStreamingReasoningForVoiceRun`
- Source: `audio_dialog_test.go:1077`
- Message: `published messages = []agent.Message(nil), want one streaming reasoning message`

The case was registered before local replay. It had appeared in the outreach candidate pool,
but no external person reported running RunBack and the repository had not been replayed by
RunBack. The first invocation completed online acquisition and reached the intended Go test,
but standard Go failure output was only step-level evidence, so the result remained
`INSUFFICIENT_EVIDENCE`. The general parser now requires the same test name, source file,
line, message and exit code. Replaying the unchanged URL then produced one remote failure,
one local failure and one exact match. The many expected warning/error strings printed by
other tests did not become structured failure items.

Evidence:

- `docs/alpha/CASE_E.json`
- `docs/alpha/evidence/case-e/first-run.log`
- `docs/alpha/evidence/case-e/retry-1.log`

## Fresh Case F

- URL: https://github.com/DeHubToken/dehub-mobile/actions/runs/34426920635
- Attempt/job: `1`, `Typecheck & Test` (`102714019996`)
- Failed step: `i18n coverage`; command: `npm run i18n:coverage`
- Head SHA: `37d4cd252070a52b105dd8f767c8f6761452dbc9`
- Runtime: Node.js 20 with setup-node and npm caching

Case F was registered before any RunBack invocation. Online acquisition, checkout, dependency
installation and the intended step all ran. The remote and local logs both reported the same
missing `settings.display` translation key in
`components/Settings/AppearancePanel.tsx`. Because this output belongs to a repository-specific
script and has no supported structured identity, RunBack retained
`INSUFFICIENT_EVIDENCE`, `STEP`, `0/0/0`. The case was not replaced and no special parser was
added to make it pass.

Evidence:

- `docs/alpha/CASE_F.json`
- `docs/alpha/evidence/case-f/first-run.log`
- `docs/alpha/evidence/case-f/first-result.json`
- `docs/alpha/evidence/case-f/job.raw.log`

## Fresh Case G

- URL: https://github.com/ClickHouse/click-ui/actions/runs/34430830779
- Attempt/job: `1`, `build` (`102725796825`)
- Failed step: `Build`; command: `yarn build`
- Head SHA: `dff6571abee07d79d9bc1e85117d6ae2d3e79c49`
- Runtime: Node.js 24.x, Corepack and Yarn 4.5.3

Case G was registered and committed before RunBack acquired the run. Its first exact URL
invocation reached the TypeScript build and reproduced all three compiler diagnostics, but
without a `tsc` parser RunBack returned `INSUFFICIENT_EVIDENCE`, `STEP`, `0/0/0` after
129.154 seconds. During setup-node, an unauthenticated tool-download API request hit its own
rate limit; setup-node's ordinary Node.js download fallback succeeded, so this did not block
the target step and acquisition credentials were not passed into execution.

The generic parser requires the same source file, line, column, TypeScript error code, full
message and exit code. It counts incomplete `error TS...` formats as unparsed failures so a
partially parsed set cannot produce `SAME_FAILURE`. The unchanged URL then produced three
remote failures, three local failures and three exact matches in 117.844 seconds. A session was
created, while its debug entry correctly remained unsupported because Node debug environments
are outside the current narrow debug scope.

Evidence:

- `docs/alpha/CASE_G.json`
- `docs/alpha/evidence/case-g/first-run.log`
- `docs/alpha/evidence/case-g/first-result.json`
- `docs/alpha/evidence/case-g/after-parser.log`
- `docs/alpha/evidence/case-g/after-parser-result.json`
- `docs/alpha/evidence/case-g/job.raw.log`

## Fresh Case H

- URL: https://github.com/janosh/svelte-widgets/actions/runs/34449963179
- Attempt/job: `1`, `unit` (`102783221808`)
- Failed step: `Unit tests with coverage`; command: `npm run test:coverage`
- API head SHA: `1015306b0a99ce5772fd483cfc50384ec471b0dd`
- Historical checkout SHA used by the replay: `3893a49b4c2c6e7c471997b4befaf01136045573`
- Runtime: Node.js 24, npm and Vitest 5.0.0

Case H was registered and committed before RunBack acquired or replayed it. The first exact
URL invocation completed authenticated acquisition and reproduced four visible Vitest
failures, but no Vitest parser existed. RunBack therefore returned
`INSUFFICIENT_EVIDENCE`, `STEP`, `0/0/0` after 184.196 seconds. The first retry after the
parser change stopped during dependency installation on a transient npm optional native
binding error; that attempt remains `REPLAY_BLOCKED / EXECUTE / UNKNOWN` and was not
relabelled as a match.

The generic parser accepts only complete Vitest failure blocks. It requires the same source
file, line, column, full hierarchical test name, exception, message and exit code. It counts
incomplete or duplicate identities as unparsed failures, so a partial set cannot produce
`SAME_FAILURE`. The unchanged URL and job then produced four remote failures, four local
failures and four exact matches after 244.421 seconds. A verified session was created, while
the Node debug entry correctly remained outside the current debug scope.

Evidence:

- `docs/alpha/CASE_H.json`
- `docs/alpha/evidence/case-h/first-run.log`
- `docs/alpha/evidence/case-h/first-run.meta`
- `docs/alpha/evidence/case-h/after-parser.log`
- `docs/alpha/evidence/case-h/after-parser.meta`
- `docs/alpha/evidence/case-h/after-parser-retry.log`
- `docs/alpha/evidence/case-h/after-parser-retry.meta`
- `docs/alpha/evidence/case-h/job.raw.log`

## Boundaries

These cases cover Python mypy/pytest/Pyright, one Go test job, one custom JavaScript failure
that remained insufficient, one standard TypeScript compiler failure and one Vitest unit-test
failure on Ubuntu. They do not prove general JavaScript/TypeScript behavior, private repositories, GitHub Enterprise, Windows,
macOS, self-hosted runners, services, job containers, local actions, reusable workflows or
secret-dependent paths. A statically reviewed TypeScript matrix candidate was rejected before
registration because its jobs used a repository-local reusable workflow, which remains outside
the alpha scope.
