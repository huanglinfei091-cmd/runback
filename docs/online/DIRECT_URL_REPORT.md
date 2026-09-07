# Direct URL MVP 实测报告

状态：**PASS（Direct URL acceptance）**  
完成时间：2026-09-06  
锁定案例：Case C，未替换

RunBack 已用一个真实 public failed run 完成：

```text
failed Run URL
→ authenticated GitHubOnlineSource
→ canonical RunEvidence
→ existing Resolver
→ Replay Plan
→ ActExecutor
→ existing Matcher
→ SAME_FAILURE
→ atomic debugging session
```

实际 CLI 参数只有：

```bash
runback https://github.com/pallets/werkzeug/actions/runs/32448268750
```

没有使用 `--bundle`、`--lock`、`--work-dir`、`--image` 或 `--network`。

## 实际实现架构

`BundleSource` 和 `GitHubOnlineSource` 都输出 `github.RunEvidence`，后续共用原有 Resolver、Replay Plan、ActExecutor 和 Matcher。没有新增 OnlineResolver、OnlineReplayPlan 或 OnlineMatcher。

Direct URL 新增的主要代码是：

- `internal/online/`：GitHub API、job log、cache 与原子发布。
- `internal/acquire/`：统一 source 选择、canonical parity、路径/lock 安全和复现 evidence。
- `cmd/runback/`：URL-first CLI、override precedence、session 结果解耦。
- `internal/session/atomic_*`：session 同文件系统原子发布、碰撞保护。
- `internal/runner/network.go`：非破坏性默认 bridge 探测和一个 RunBack-owned managed fallback。

阶段差异清单见 `STAGE_EVIDENCE.md`。

## GitHub API acquisition 与 attempt

OnlineSource 获取真实 run metadata。URL 显式指定 attempt 时使用 attempt run endpoint；未指定时使用 run metadata 当前 `run_attempt`。Jobs 使用：

```text
GET /repos/{owner}/{repo}/actions/runs/{run_id}/attempts/{attempt}/jobs?per_page=100&page=N
```

持续分页直到不足 100 条，并校验每个 job 的 run/attempt identity；不在 source 内选择目标 job，选择仍由原 Resolver 完成。Private、非 completed、非 failure 和 identity 不一致会在 acquisition 阶段停止。

## Workflow snapshot

Workflow path 只取 run metadata；缺失时通过 `workflow_id` 获取 workflow metadata/path。文件通过 Contents API 按失败 run 的 `head_sha` 获取，不扫描或猜测默认分支 workflow。

原 Resolver 从 job log 得到真实 checkout merge commit 时，会继续读取同一路径的历史 workflow snapshot。OnlineSource 会保留该补充读取的 typed error，`Finish` 不会发布或执行缺少 snapshot 的 cache。

## Job log redirect、raw evidence 与大小限制

Job log API client 禁止自动 redirect，手动读取 302 `Location`；signed URL 使用新建的无 Authorization、无 Cookie、无 CookieJar client 下载。signed URL 不进入 cache。

下载按 64 KiB 流式写临时文件，同时计算 byte count 和 SHA256，成功后原子发布为 `job.raw.log`。HTTP Content-Encoding 只在 HTTP 层处理。默认上限 104857600 bytes；`RUNBACK_MAX_LOG_SIZE` 只接受正整数 bytes，超限返回 `EVIDENCE_TOO_LARGE`，不把截断日志交给 Resolver/Matcher。

Case C raw log：

- size：141084 bytes
- SHA256：`e283b17b1a29ed889f14c63681f723cf8c8a5e68856e58783e64e52490a1a1d1`

## Credential isolation

认证优先级：

```text
RUNBACK_GITHUB_TOKEN > GH_TOKEN > anonymous
```

真实验收在 Ubuntu 本机调用 `gh auth token`，仅将返回值放入当前 RunBack 进程的 `RUNBACK_GITHUB_TOKEN` 环境，不写命令行或文件。Token 只由 GitHubOnlineSource 的 API request 使用。

Runner、Git、act、Docker doctor、session、dev/replay 都使用显式 allowlist/sanitized environment。单元测试覆盖 `RUNBACK_GITHUB_TOKEN`、`GH_TOKEN`、`GITHUB_TOKEN` 不进入执行进程或 Docker 参数。最终 live 扫描 cache、session、全部短 workspace、隔离验证 HOME 和 `docs/online` 共 35345 个文件，Token 匹配数为 0。

## Cache schema 与 refresh

Cache identity：

```text
~/.runback/cache/github/<owner>/<repo>/<run-id>/attempt-<n>/
```

包含 `manifest.json`、`run.json`、`jobs.json`、`files.json`、`workflow.yml`、`job.raw.log`。Manifest 记录 repository/run/attempt/job/head/path、各文件 SHA256、log size 和 fetched_at。只有全部文件存在、identity 一致、普通文件且 hash/size 全部通过才是 HIT。

Acquisition 先写 RunBack-owned `.tmp/runback-acquire-*`，完整校验后发布不可变 generation，再原子切换 attempt symlink。Partial/corrupt cache 当作 miss。`--refresh` 重新获取 run、jobs、workflow metadata/content、新的 log 302 和最终 log，不复用 signed URL；刷新失败保留旧完整 generation。

超过 24 小时且符合 RunBack temp 命名的目录可 best-effort 清理；正式 cache、session 和用户文件不删除。

## Workspace、lock 与 override

Linux 自动 transient replay workspace 统一为：

```text
/tmp/rb/<12-hex-short-id>
```

Online 和 Bundle 走同一个 allocator。物理路径不含 repo、完整 run/job ID 或时间戳。显式 `--work-dir` 保留原语义，不被自动路径覆盖。

Source 优先级是显式 `--bundle` 高于 Online；bundle 模式不初始化 OnlineSource。Execution override 优先级为 CLI image/network，高于 identity-matched lock/session，高于 generic default。显式 lock 必须匹配 repository/run/attempt/commit/job；显式 work-dir 中的非 RunBack 数据会以 `WORKDIR_NOT_SAFE` 拒绝并保留。

## Docker network

RunBack 不删除/重建 default bridge，不改 daemon，不重启 Docker，不指定宿主 IP、subnet 或 gateway。默认 bridge 通过一个无 Token 的短生命周期容器做通用 DNS probe。失败时最多选择一个 fallback：优先复用 `io.runback.managed=true`、driver=bridge、non-internal 的现有网络；没有时只执行一次：

```text
docker network create --driver bridge --label io.runback.managed=true runback-managed
```

不传 subnet/gateway，不读取 repository 名称。选择结果记录为 `Network Source: MANAGED_FALLBACK`。Case C 成功运行时默认 bridge probe 健康，因此真实输出为 `GENERIC_DEFAULT`；managed fallback 分支通过 fake-Docker process test，未在这次成功运行中触发。

早期诊断曾手工为 `docker0` 添加 `172.17.0.1/16`，记录在 `evidence/docker-bridge-repair.json`。这是一次超出 RunBack 自有资源边界的宿主修改；收到约束后未再修改、删除、重建或重启宿主 Docker/网络。本次结果不能作为 managed fallback 的真实环境证明。

## Online / Bundle parity

Flask：

- URL：`https://github.com/pallets/flask/actions/runs/33397112701`
- attempt 1；job `typing` / `99504166020`
- authenticated OnlineSource：PASS
- Online/Bundle semantic hash：`50178bb88e832b791c08feef6db6bc3ad3775ba93acca60f9ec578250693c4d8`
- parity：PASS

Click：

- URL：`https://github.com/pallets/click/actions/runs/33760631267`
- attempt 1；job `3.13` / `100665765259`
- authenticated OnlineSource：PASS
- Online/Bundle semantic hash：`13e96a17f2ebc45a1d8e3145857e50251220ed43f5ac65ffb5ea6a55376333b3`
- parity：PASS

语义比较包括 repository、run/attempt、commit、event、workflow、job、runner、matrix、runtime、failed step 和 failure evidence；fetch timestamp/cache metadata 不参与。

## Case C：Direct URL acceptance

预注册且始终锁定：

- Repository：`pallets/werkzeug`
- URL：`https://github.com/pallets/werkzeug/actions/runs/32448268750`
- Attempt：1
- Job：`3.9` / numeric ID `96671790465`
- Runner：`ubuntu-latest`
- Matrix：`python=3.9`
- 选择依据：public completed failed run；目标 job 只有 checkout/setup-uv/setup-python 和明确的 uv/tox run step；无 services、job container、local action 或目标 secrets。

登记在 `CASE_C.json`，时间早于任何本地 replay。没有因结果不好更换案例。

真实过程全部保留：

1. 原长路径：`DIFFERENT_FAILURE`，remote 1 / local 2 / matched 0；额外失败为 `AF_UNIX path too long`。
2. 短路径第一次：`REPLAY_BLOCKED`；act 宿主侧下载 action 时 connection reset，现分类为 `ACTION_FETCH_FAILED`。
3. 短路径第二次：remote 1 / local 1，但 pytest `DID NOT RAISE` 尚未成为 structured Item，结果 `INSUFFICIENT_EVIDENCE`。
4. 添加通用 pytest `Failed: DID NOT RAISE` 解析后：`SAME_FAILURE`，remote 1 / local 1 / matched 1；所有 Identity、Exception、File、Line、Message、Exit 字段一致。
5. 新隔离 HOME 冷 cache 重试：完整 online acquisition + replay 再次 `SAME_FAILURE`，remote 1 / local 1 / matched 1。

没有修改 Werkzeug 测试、远程 evidence 或 Matcher 阈值。Parser 新增的是通用 pytest failure 类型：

```text
identity: tests/test_formparser.py::TestFormParser::test_limiting
exception: Failed
file: tests/test_formparser.py
line: 61
message: DID NOT RAISE <class 'werkzeug.exceptions.RequestEntityTooLarge'>
exit: 1
```

成功后原子创建 session，并在完整创建后切换 active。复现容器已停止；session 文件保留。

## TTFR

成功的冷 cache 运行：

- `TTFR_ONLINE_COLD`：41.564 s（CLI start 到 `Result: SAME_FAILURE`）
- CLI start 到 session 创建/退出：48.131 s
- evidence cache：miss（输出为 `CACHE_INCOMPLETE` 后完整重新 acquisition）
- image：预先存在，ID `sha256:62d572b92f9f32d3427b6d220ad1f9dca9c7b6ffad37d295425037dbff78abaf`
- dependency cache：新 transient workspace，无项目依赖 cache
- network downloads：发生 action/uv/package 下载
- auth：Ubuntu-local token
- CLI overrides：无
- network source：`GENERIC_DEFAULT`

成功的原用户 HOME cache-hit 运行：

- `TTFR_ONLINE_CACHE_HIT`：53.158 s
- CLI start 到 session 创建/退出：58.230 s
- evidence cache：HIT
- dependency downloads：发生
- auth：Ubuntu-local token
- CLI overrides：无

两次依赖网络状态不同，不能用这两个样本推断 cache hit 更慢或更快；不与 M1/M2 性能数字混算。

## Regression 与最终验证

Flask bundle regression：退出码 0，BundleSource→Resolver→Replay Plan 成功。  
Click bundle regression：退出码 0，BundleSource→Resolver→Replay Plan 成功。  
M2 Click Fast Replay：`STEP_PASSED_UNVERIFIED`，退出码 0。  
M2 Click full verify：`FULL_JOB_PASSED`，退出码 0。  
M1/M2 `SHA256SUMS`：均退出码 0，历史 authoritative evidence 未修改。

最终检查在最后一次 Case C 成功后运行：

```text
go test ./...             PASS (exit 0)
go vet ./...              PASS (exit 0)
go build ./cmd/runback    PASS (exit 0)
```

测试覆盖 URL/attempt、attempt jobs/pagination、workflow metadata/head snapshot、403 rate limit/forbidden、302 plain log、signed request header isolation、stream/hash/size、cache partial/corrupt/atomic refresh、token priority/redaction/process isolation、Bundle/Online parity、lock/workdir safety、short allocator、managed network、atomic session、session failure preserving SAME_FAILURE 和非 SAME session gating。

## 首次匿名限流证据

匿名 Flask 请求真实返回：

```text
EVIDENCE_UNAVAILABLE
Stage: ACQUIRE
Cause: GITHUB_RATE_LIMITED
remaining=0
reset=2026-09-06T06:00:21Z
mode=anonymous
```

执行在 ActExecutor 前停止，没有 silent bundle fallback。证据保留为 `evidence/flask-online-attempt.log`。

## Known Limitations

- P0 仅支持 public GitHub.com、completed failed run、`ubuntu-latest` 和静态可映射 workflow/job/matrix。
- Private、GitHub Enterprise、self-hosted、Windows/macOS runner、services、job/container action、repository-local action、secret-heavy workflow 不在当前范围。
- GitHub 历史 log/workflow 可能不可再获取；无法证明过期时只报告 unavailable，并提示 `--bundle`。
- act 和 GitHub hosted runner 环境不是完全相同；只有严格 Matcher 通过才报告 SAME_FAILURE。
- 自动短 workspace 是 Linux-only；当前 Windows build 可编译，但 Direct URL execution 验收只在 Ubuntu 完成。
- managed Docker fallback 已完成隔离单测，但本次 Case C 成功运行未触发该分支。
- Immutable cache generation 当前保留旧 generation，尚未实现正式 cache GC。

## Evidence paths

- `docs/online/CASE_C.json`
- `docs/online/STAGE_EVIDENCE.md`
- `docs/online/evidence/case-c-online-cold-retry.log`
- `docs/online/evidence/case-c-online-cold-retry.timing.json`
- `docs/online/evidence/case-c-short-workspace-retry-3.log`
- `docs/online/evidence/case-c-short-workspace-retry-3.timing.json`
- `docs/online/evidence/case-c-short-workspace-retry-2.log`
- `docs/online/evidence/case-c-short-workspace.log`
- `docs/online/evidence/case-c-online-cold.log`
- `docs/online/evidence/flask-authenticated-inspect.log`
- `docs/online/evidence/flask-parity.json`
- `docs/online/evidence/click-authenticated-inspect.log`
- `docs/online/evidence/click-parity.json`
- `docs/online/evidence/m1-flask-bundle-regression.log`
- `docs/online/evidence/m1-click-bundle-regression.log`
- `docs/online/evidence/m2-fast-replay-regression.log`
- `docs/online/evidence/m2-full-verify-regression.log`
- `docs/online/evidence/docker-bridge-repair.json`
