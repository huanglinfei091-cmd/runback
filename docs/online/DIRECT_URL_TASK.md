# Direct URL MVP 执行规格（用户要求冻结）
本文件整理自本轮用户的 0–35 节规格，作为执行依据。唯一任务 Direct URL MVP，不进入下一里程碑。

## 0–2 范围、现场与架构
项目 /root/runback-work/runback。先检查真实文件名；保留 M1/M2 已验证核心语义和 docs/m1/evidence、docs/m2/evidence。无真实通用回归不得重写 Resolver、Replay Plan、ActExecutor、Matcher、Session、dev、replay --step、verify。
创建缺失的 docs/online、docs/progress，维护 DIRECT_URL_TASK.md、DIRECT_URL_REPORT.md、CURRENT.md；已有进度历史只追加，不清空。不得创建重复 M1/M2 文档。
GitHubOnlineSource 与 BundleSource → 同一 canonical RunEvidence → Existing Resolver → Replay Plan → ActExecutor → Matcher。只增加 evidence acquisition，不增加 OnlineResolver/OnlineReplayPlan/OnlineMatcher。

## 3–5 身份和获取
支持 github.com owner/repo/actions/runs/run_id 及明确 attempt。identity=repository+run_id+run_attempt；job identity 加 job_id。明确 attempt 必须遵循，否则读取当前 run_attempt。jobs/log/cache 不能跨 attempt。
仅 public、completed failure。获取 run metadata、完整分页 attempt jobs(per_page=100)、workflow path、workflow@head_sha、Resolver 所选 job 的日志；Source 不选择 job。
Path 来自 run metadata 或 workflow_id metadata，禁止按名称、扫描目录、Git Trees 或默认分支猜测。无法确定或在 head_sha 读取时 EVIDENCE_UNAVAILABLE / WORKFLOW / WORKFLOW_UNAVAILABLE，并提示 --bundle。
Job log API 返回短时 302 Location，最终是 plain text，不是 workflow ZIP。

## 6–8 凭据与原始证据
三个分离 client：apiClient 可带 Authorization；jobLogAPIClient 禁止自动 redirect；logDownloadClient 是独立无 Authorization/Cookie/CookieJar 的 client，手工请求 Location。测试请求边界，不依赖 Go 默认重定向保护。
Token priority RUNBACK_GITHUB_TOKEN > GH_TOKEN > anonymous。只供 online/API 使用，不传执行平面、Docker、dev、Git、session、lock/cache/evidence/log/error/命令行。执行子进程使用显式 sanitized env。
保留下载字节 job.raw.log，记录 size 和 SHA256。HTTP 层只解 Content-Encoding 一次，不猜内容 gunzip。Normalization 另存，不覆盖 raw。

## 9–12 Streaming 与 cache
HTTP Body → tempfile + SHA256 + byte count，成功校验后原子发布。禁止直接 io.ReadAll 大日志。默认 104857600 bytes；RUNBACK_MAX_LOG_SIZE 必须正整数 bytes，非法明确配置错误。超限 EVIDENCE_UNAVAILABLE / JOB_LOG / EVIDENCE_TOO_LARGE，报告 observed/limit，清理 temp，截断证据绝不进入下游。
Cache ~/.runback/cache/github/<owner>/<repo>/<run-id>/attempt-<n>/：
manifest.json、run.json、jobs.json、workflow.yml、job.raw.log。
Manifest 至少 schema_version、repository/run_id/run_attempt/job_id、head_sha、workflow_path、workflow/jobs/job_log SHA256、job_log_size、fetched_at。
Required files 完整、identity 一致、hash 正确才命中。partial=CACHE_INCOMPLETE miss；corrupt hash=corrupt miss。失败不采用部分证据。
os.MkdirTemp 创建 owned unique temp，完整 acquire/validate 后原子发布；失败仅删除本次 temp。启动可 best-effort 清理明确 owned 且 >24h stale temp，失败不阻止启动，禁止删除正式 cache/session/user files。
--refresh 重新获取 run、attempt jobs、workflow path/snapshot、job-log API、新 redirect 和日志，再原子替换。永不持久化 signed URL。

## 13–15 获取错误
顶层统一 EVIDENCE_UNAVAILABLE，具体 Stage/Cause：RUN_METADATA_UNAVAILABLE、RUN_ATTEMPT_UNAVAILABLE、JOBS_UNAVAILABLE、WORKFLOW_UNAVAILABLE、JOB_LOG_UNAVAILABLE、GITHUB_RATE_LIMITED、GITHUB_FORBIDDEN、CACHE_CORRUPT、CACHE_INCOMPLETE、NETWORK、EVIDENCE_TOO_LARGE。
Rate limit 需 headers/明确证据；403 不能直接推断。显示 ACQUIRE、remaining/reset、anonymous/token mode，提示 Set RUNBACK_GITHUB_TOKEN for a higher API limit。禁止无限 retry/长等待。403 无 rate evidence=GITHUB_FORBIDDEN；404 不能断言日志过期。
显式 --bundle 强制 bundle，不触碰 Online；否则默认 Online。失败提示 You may retry using --bundle，不偷偷回退/换 run/换 repo。

## 16–20 Overrides、路径与执行网络
CLI --image/--network > 匹配身份的 recorded lock/session > generic defaults。显式值必须使用并打印来源；禁止按仓库名选环境。历史 Click less image/runback-replay network 不是全局默认。
--lock/--work-dir 覆盖自动路径。已有显式 lock 必须匹配 repository/run/attempt/commit/job，否则 LOCK_IDENTITY_MISMATCH，不执行/覆盖错误 lock。
work-dir 不能覆盖用户数据；可能覆盖非 managed 内容时报 WORKDIR_NOT_SAFE。
Online API 是 ACQUIRE；act 下载 action 是 EXECUTE / ACTION_FETCH_FAILED 或 NETWORK，不能混成 acquisition rate limit。执行不得带上述 token。

## 21–25 Parity、session 与结果
同 run 比较 canonical 语义：repository/run/attempt/commit/event/workflow/job name+id/runner/matrix/runtime/failed step/failure evidence；忽略 HTTP/fetch/cache 元数据。可生成 semantic hash，不直接比较 raw JSON。
只有 FULL_JOB + SAME_FAILURE 创建 verified session。沿用 M2 ID 策略，同 filesystem .tmp-unique 完整创建 original/worktree/state/lock/log metadata、验证、原子 rename、不覆盖 final，最后 active。碰撞换 ID 或报错，不删旧会话。
Reproduction evidence 独立保留在 ~/.runback/reproductions/<owner>/<repo>/<run-id>/attempt-<n>/<job-id>/（可追加唯一执行子目录）：evidence.json、replay-plan.json、act.log。
Session 失败仍保持 Result:SAME_FAILURE + SESSION_CREATE_FAILED + evidence 路径，不改成 replay failure。不做 partial recovery/session repair/evidence show。
成功显示 SAME_FAILURE、session ID、workspace、Next:runback dev；没有残留运行容器，session 文件仍占磁盘。

## 26–28 真实验收与 TTFR
Case C 仅 DIRECT_URL_ACCEPTANCE，必须在任何本地 replay 前根据远程静态信息登记 repo/URL/run/attempt/job/选择时间/理由。优先 public ubuntu-latest completed Python pip/uv pytest/mypy、简单依赖、明确 run command；允许 checkout/setup-python/setup-uv，禁止 local/container/未知 setup action、services、secret-heavy。
登记后不得替换/隐藏 Case C。失败保留，额外成功案例只能另记 C2。不得宣传兼容性普遍证明，Fresh Case D 不属于本轮。
先尝试历史 Flask/Click online parity；历史日志取不到记录 HISTORICAL_ONLINE_LOG_UNAVAILABLE，不改 M1/M2 结论。两者 bundle regression 必须 PASS。
TTFR_ONLINE_COLD=CLI start/cache miss→acquire→replay→结果；cache hit 另记。注明 image、dependency cache、下载、token、CLI overrides。不能混入 M1 42.72s、M2 5.46s 或安装时间。

## 29–30 必须测试与回归
测试覆盖 URL/attempt、online source、attempt jobs/分页、workflow path/head_sha/404/403 rate vs forbidden、302/plain text/raw SHA/streaming/100MiB/envlimit、完整/partial/corrupt/atomic cache、refresh/stale cleanup；
anonymous/token priority/redaction、token 不进 act/Docker/dev、signed download 无 Auth/Cookie；
错误分类、online/bundle parity、lock/workdir安全；
FULL_JOB SAME 创建 session、atomic/collision/session失败保留复现、非SAME不建 verified session。
不得只有 case-specific 测试。最终 go test ./...、go vet ./...、go build ./cmd/runback。
M1 Flask/Click bundle 和 M2 session/debug 回归必须通过；历史 authoritative evidence 不改。

## 31–33 停止与完成标准
不扩展 private、Enterprise、artifacts/services/secrets/复杂Actions/self-hosted/Windows/macOS；如需重写冻结架构则停止并准确记录 limitation。
禁止 Node/Go dev、benchmark项目扩张、Marketplace、AI/Dashboard、Native Executor、通用引擎、persistent dev、repo hacks、为变绿换案例、降 Matcher 标准、大规模无关重构。
至少一例真实 public URL 仅 runback URL（无需 bundle/lock/work-dir/image/network）完成 online acquisition+真实 replay，返回可信结果；SAME_FAILURE 优先。统一证据、attempt/head snapshot、凭据、cache/refresh/parity/回归和 test/vet/build 全部满足后才完成。

## 34–35 交付与自主执行
CURRENT.md 追加阶段状态。REPORT 包含真实架构/API/attempt/head snapshot/redirect/credentials/cache/raw SHA/refresh/override/parity/regressions/Case C/TTFR/tests/命令/结果/失败/limitations/evidence paths。禁止编造数据或把 blocked/likely 写成 same。
普通代码、测试、编译、404、Docker/network 错误自行诊断修复记录。仅登录/OAuth/2FA/缺权限/付费/条款/高风险不可逆或可能破坏用户数据时询问。
