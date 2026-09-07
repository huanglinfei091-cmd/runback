# External User Candidates

Discovery completed: 2026-09-07 03:40 UTC

Release under test: `v0.1.0-alpha`

Repository: https://github.com/huanglinfei091-cmd/runback

## Method

The candidate pool was created only after the public repository and downloadable release
existed. Discovery queried active public repositories whose primary language was Python,
JavaScript, TypeScript or Go, with 25–10,000 stars and recent pushes. From those repositories
it fetched recent completed failed `pull_request` workflow runs. Open pull requests with a
current failed status were used to fill the pool when fewer than 100 direct runs were found.

No candidate was run through RunBack before selection. Static qualification required an open
pull request from a human author, a completed failed GitHub Actions run, an attempt-specific
failed job with an Ubuntu runner label, and a concrete failing step. Obvious preview deployment,
AI review, required-label and aggregate required-check gates were excluded from Tier A.

The resulting counts are:

| Measure | Count |
| --- | ---: |
| Candidate pool | 100 |
| Python | 18 |
| JavaScript | 13 |
| TypeScript | 37 |
| Go | 32 |
| Bot-authored PRs rejected | 47 |
| No failed Ubuntu job | 3 |
| Closed PR | 1 |
| Indirect failed-status candidates | 15 |
| Open human PRs with a failed Ubuntu job | 34 |
| Obvious non-reproduction gates removed | 4 |
| Tier A | 30 |
| First outreach wave | 15 |

Tier A is a static acquisition fit, not a claim that RunBack will reproduce the failure. A
candidate can still truthfully produce `DIFFERENT_FAILURE` or `REPLAY_BLOCKED`. Reserve entries
include harder E2E, aggregate-result, local-action or environment-sensitive cases and will not
be contacted in the first wave.

## Tier A

| # | Pull request | Language | Attempt / failed run | Job | Failed step | Plan |
| ---: | --- | --- | --- | --- | --- | --- |
| 1 | [ahmadrosid/nakama#871](https://github.com/ahmadrosid/nakama/pull/871) | TypeScript | [1](https://github.com/ahmadrosid/nakama/actions/runs/34055761725) | knip | Run knip | Wave 1 |
| 2 | [AidenAI-IO/aiden-firmware#639](https://github.com/AidenAI-IO/aiden-firmware/pull/639) | Go | [1](https://github.com/AidenAI-IO/aiden-firmware/actions/runs/34079675283) | go-tests | Run Go unit tests | Wave 1 |
| 3 | [block/schemabot#1317](https://github.com/block/schemabot/pull/1317) | Go | [1](https://github.com/block/schemabot/actions/runs/34079707342) | lint (consumer-module, e2e/consumermodule) | golangci-lint (consumer-module) | Wave 1 |
| 4 | [crbnos/carbon#1583](https://github.com/crbnos/carbon/pull/1583) | TypeScript | [1](https://github.com/crbnos/carbon/actions/runs/34079401006) | Catalog | Run pnpm run check:workflow-catalog | Wave 1 |
| 5 | [DaoCloud/DaoCloud-docs#7349](https://github.com/DaoCloud/DaoCloud-docs/pull/7349) | Python | [1](https://github.com/DaoCloud/DaoCloud-docs/actions/runs/33854186185) | build-test | Run make sync | Wave 1 |
| 6 | [dunglas/mercure#1358](https://github.com/dunglas/mercure/pull/1358) | Go | [1](https://github.com/dunglas/mercure/actions/runs/32027374001) | lint | Lint Code Base | Wave 1 |
| 7 | [dynamical-org/reformatters#1019](https://github.com/dynamical-org/reformatters/pull/1019) | Python | [1](https://github.com/dynamical-org/reformatters/actions/runs/34073560564) | Code Quality (arm64) | Run Pytest | Wave 1 |
| 8 | [fastrepl/anarlog#7356](https://github.com/fastrepl/anarlog/pull/7356) | TypeScript | [1](https://github.com/fastrepl/anarlog/actions/runs/34015472318) | fmt | Check formatting | Wave 1 |
| 9 | [inkstitch/inkstitch#4548](https://github.com/inkstitch/inkstitch/pull/4548) | Python | [1](https://github.com/inkstitch/inkstitch/actions/runs/33997036586) | test | Style check | Wave 1 |
| 10 | [ivanarama/onebase#1232](https://github.com/ivanarama/onebase/pull/1232) | Go | [1](https://github.com/ivanarama/onebase/actions/runs/34003122055) | build | go vet | Wave 1 |
| 11 | [looplj/axonhub#2395](https://github.com/looplj/axonhub/pull/2395) | Go | [1](https://github.com/looplj/axonhub/actions/runs/33902487055) | lint | golangci-lint | Wave 1 |
| 12 | [openclaw/crabbox#1898](https://github.com/openclaw/crabbox/pull/1898) | Go | [1](https://github.com/openclaw/crabbox/actions/runs/34005581617) | Go test | Test | Wave 1 |
| 13 | [ValueCell-ai/ClawX#1283](https://github.com/ValueCell-ai/ClawX/pull/1283) | TypeScript | [1](https://github.com/ValueCell-ai/ClawX/actions/runs/34046172092) | check | Run tests | Wave 1 |
| 14 | [web-infra-dev/rslib#1909](https://github.com/web-infra-dev/rslib/pull/1909) | TypeScript | [1](https://github.com/web-infra-dev/rslib/actions/runs/34075715726) | ut (ubuntu-latest, 24) / test | Unit Test | Wave 1 |
| 15 | [web-infra-dev/rstest#1755](https://github.com/web-infra-dev/rstest/pull/1755) | TypeScript | [1](https://github.com/web-infra-dev/rstest/actions/runs/33865470108) | ut | Run Test | Wave 1 |
| 16 | [aqm857886159/Nomi#580](https://github.com/aqm857886159/Nomi/pull/580) | TypeScript | [1](https://github.com/aqm857886159/Nomi/actions/runs/34078843644) | E2E Walkthroughs (Linux) | MCP L1 handshake journey | Reserve |
| 17 | [cacheplane/angular-agent-framework#1040](https://github.com/cacheplane/angular-agent-framework/pull/1040) | TypeScript | [1](https://github.com/cacheplane/angular-agent-framework/actions/runs/34062155619) | Website — e2e | Run npx nx e2e website --skip-nx-cache | Reserve |
| 18 | [decocms/studio#7022](https://github.com/decocms/studio/pull/7022) | TypeScript | [1](https://github.com/decocms/studio/actions/runs/33926794234) | multi-pod | Run multi-pod scenarios | Reserve |
| 19 | [gemini-testing/html-reporter#803](https://github.com/gemini-testing/html-reporter/pull/803) | TypeScript | [1](https://github.com/gemini-testing/html-reporter/actions/runs/33601451543) | testplane-component | Fail the job if any Testplane job is failed | Reserve |
| 20 | [inclusionAI/Avernet#1953](https://github.com/inclusionAI/Avernet/pull/1953) | Python | [1](https://github.com/inclusionAI/Avernet/actions/runs/34079106157) | Singlebox coverage | Run singlebox coverage | Reserve |
| 21 | [objectstack-ai/objectstack#16470](https://github.com/objectstack-ai/objectstack/pull/16470) | TypeScript | [1](https://github.com/objectstack-ai/objectstack/actions/runs/34078288734) | Part-of PR must not also close its card | PR/card relationship validation | Reserve |
| 22 | [opskat/opskat#262](https://github.com/opskat/opskat/pull/262) | Go | [1](https://github.com/opskat/opskat/actions/runs/30076945745) | Go Lint | golangci-lint action | Reserve |
| 23 | [pwrdrvr/PwrAgent#2001](https://github.com/pwrdrvr/PwrAgent/pull/2001) | TypeScript | [1](https://github.com/pwrdrvr/PwrAgent/actions/runs/34067602810) | Test | Test | Reserve |
| 24 | [raullenchai/Rapid-MLX#3104](https://github.com/raullenchai/Rapid-MLX/pull/3104) | Python | [1](https://github.com/raullenchai/Rapid-MLX/actions/runs/34058117205) | tests | Check test results | Reserve |
| 25 | [Servosity/msp-skills#311](https://github.com/Servosity/msp-skills/pull/311) | Go | [1](https://github.com/Servosity/msp-skills/actions/runs/34077884077) | guards | Install and remote-MCP docs match shipped artifacts | Reserve |
| 26 | [stackrox/stackrox#22609](https://github.com/stackrox/stackrox/pull/22609) | Go | [1](https://github.com/stackrox/stackrox/actions/runs/34074932625) | style-check | make style-slim | Reserve |
| 27 | [strelov1/freehire#2574](https://github.com/strelov1/freehire/pull/2574) | Go | [1](https://github.com/strelov1/freehire/actions/runs/34079082284) | design-system | Check token coverage | Reserve |
| 28 | [The-AI-Republic/pi-dash#338](https://github.com/The-AI-Republic/pi-dash/pull/338) | TypeScript | [1](https://github.com/The-AI-Republic/pi-dash/actions/runs/32886591243) | cargo test + clippy | Test | Reserve |
| 29 | [vm0-ai/vm0#32161](https://github.com/vm0-ai/vm0/pull/32161) | TypeScript | [1](https://github.com/vm0-ai/vm0/actions/runs/34079686456) | Workflow Lint | Run workflow script tests | Reserve |
| 30 | [zouyuxuan122/DSH-Desktop-EAC#290](https://github.com/zouyuxuan122/DSH-Desktop-EAC/pull/290) | JavaScript | [1](https://github.com/zouyuxuan122/DSH-Desktop-EAC/actions/runs/33772213740) | Ubuntu Tauri test/build | Type check and full test suite | Reserve |

## Contact boundary

Wave 1 contains 15 distinct repositories and authors. Each message will reference the exact
failed run and job, identify RunBack as early alpha software, ask for the observed verdict and
installation feedback, and explicitly avoid asking for a star or promotion. Reserve candidates
will only be considered at the user-adjusted two-hour checkpoint if fewer than two people have actually
tested RunBack. Total direct outreach remains capped at 30 for the day.
