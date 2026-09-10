# Real Compatibility Cases

Last updated: 2026-09-10

This file records retained executions of real public GitHub Actions failures. It is not a
compatibility percentage: five selected cases cannot establish a population success rate.
`SAME_FAILURE` appears only when the existing Matcher accepts complete structured evidence.

| Case | Repository | Runtime and workflow | First retained verdict | Final retained verdict | Generic issue and change | Measured TTFR |
| --- | --- | --- | --- | --- | --- | --- |
| M1 A | `pallets/flask` | Python 3.14, setup-python, mypy | `SAME_FAILURE`, 1/1/1 | `SAME_FAILURE`, 1/1/1 | Baseline structured mypy evidence | 85.58 s |
| M1/M2 B | `pallets/click` | Python 3.13 matrix, setup-uv, setup-python, tox/pytest | `DIFFERENT_FAILURE`, remote 1 / local 25 / matched 1 | `SAME_FAILURE`, then `FULL_JOB_PASSED` after a source fix | Missing `less` produced 24 extra failures; frozen validation used an explicit recorded image override and kept the failure-count standard | 42.72 s original M1; warm step replay median 5.459 s |
| Direct URL C | `pallets/werkzeug` | Python 3.9, setup-uv, pytest | `DIFFERENT_FAILURE` | `SAME_FAILURE`, 1/1/1 | Short workspace allocator and a general pytest `DID NOT RAISE` parser; intermediate action-fetch and insufficient-evidence results retained | 41.564 s to cold online result; 48.131 s through session creation |
| Fresh D | `opencitations/ramose` | Python, setup-uv cache, `uv sync`, Pyright | `REPLAY_BLOCKED / PREPARE / NETWORK_DEPENDENCY` | `SAME_FAILURE`, 1/1/1, `STRUCTURED` | Added standard Pyright diagnostics and normalized RunBack-owned short workspace prefixes; no Matcher threshold change | 52.114 s first; 56.252 s final |
| Fresh E | `AidenAI-IO/aiden-firmware` | Go 1.26.7, setup-go, `go test ./...` | `INSUFFICIENT_EVIDENCE`, 0/0/0 | `SAME_FAILURE`, 1/1/1, `TEST` | Added Go test name/source/message parsing; unrelated error-looking test logs remain outside the structured identity | 153.287 s first; 164.047 s final |

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

## Boundaries

These cases cover Python mypy/pytest/Pyright and one Go test job on Ubuntu. They do not prove
general JavaScript/TypeScript behavior, private repositories, GitHub Enterprise, Windows,
macOS, self-hosted runners, services, job containers, local actions, reusable workflows or
secret-dependent paths. A statically reviewed TypeScript matrix candidate was rejected before
registration because its jobs used a repository-local reusable workflow, which remains outside
the alpha scope.
