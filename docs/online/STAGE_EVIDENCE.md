# Direct URL MVP 阶段性实现证据

生成时间：2026-09-06（Asia/Shanghai）  
Ubuntu 工作目录：`/root/runback-work/runback`

本文件记录当前实际代码、测试与真实执行结果。Direct URL MVP 尚未标记为 PASS。

## 1. 本阶段新增/修改的源码文件

相对于 Windows 上冻结的 M1/M2 源码副本 `E:\linux的项目\github\1\runback`，Ubuntu 当前源码差异如下。

新增：

- `cmd/runback/completion.go`
- `internal/acquire/acquire.go`
- `internal/acquire/paths.go`
- `internal/acquire/persist.go`
- `internal/online/cache.go`
- `internal/online/http.go`
- `internal/online/log.go`
- `internal/online/source.go`
- `internal/session/atomic_linux.go`
- `internal/session/atomic_other.go`
- `scripts/online-evaluation.py`

修改：

- `cmd/runback/main.go`
- `internal/github/client.go`
- `internal/github/fixture.go`
- `internal/runner/runner.go`
- `internal/session/session.go`
- `go.mod`
- `go.sum`

测试文件另列于下一节。

## 2. 本阶段新增/修改的测试文件

新增：

- `cmd/runback/completion_test.go`
- `cmd/runback/url_test.go`
- `internal/acquire/acquire_test.go`
- `internal/acquire/parity_integration_test.go`
- `internal/online/edges_test.go`
- `internal/online/online_test.go`
- `internal/runner/credential_process_test.go`
- `internal/session/atomic_test.go`
- `internal/session/credential_process_test.go`

修改：

- `internal/github/client_test.go`
- `internal/runner/runner_test.go`
- `internal/session/session_test.go`

## 3. `docs/online/` 当前文件

- `CASE_C.json`
- `DIRECT_URL_REPORT.md`
- `DIRECT_URL_TASK.md`
- `STAGE_EVIDENCE.md`
- `evidence/case-c-candidate-remote.json`
- `evidence/case-c-online-cold.log`
- `evidence/case-c-online-cold.timing.json`
- `evidence/case-c-pyproject.toml`
- `evidence/click-authenticated-inspect.log`
- `evidence/click-authenticated-inspect.timing.json`
- `evidence/click-parity.json`
- `evidence/click-parity-test.log`
- `evidence/docker-bridge-repair.json`
- `evidence/flask-authenticated-inspect.log`
- `evidence/flask-authenticated-inspect.timing.json`
- `evidence/flask-online-attempt.log`
- `evidence/flask-parity.json`
- `evidence/flask-parity-test.log`
- `evidence/go-build.log`
- `evidence/go-test.log`
- `evidence/go-vet.log`
- `evidence/m1-history-integrity.log`
- `evidence/m2-history-integrity.log`
- `evidence/werkzeug-candidate-remote.json`

## 4. `go test ./...`

命令：

```bash
cd /root/runback-work/runback
go test ./...
```

结果：PASS，退出码 0。所有包通过；`internal/lockfile` 和 `internal/replay` 当前无测试文件。完整输出：`docs/online/evidence/go-test.log`。

## 5. `go vet ./...`

命令：

```bash
cd /root/runback-work/runback
go vet ./...
```

结果：PASS，退出码 0，无诊断输出。证据文件：`docs/online/evidence/go-vet.log`。

## 6. `go build ./cmd/runback`

命令：

```bash
cd /root/runback-work/runback
go build -o bin/runback ./cmd/runback
```

结果：PASS，退出码 0，无诊断输出。证据文件：`docs/online/evidence/go-build.log`。

## 7. Flask authenticated OnlineSource

实际产品命令：

```bash
RUNBACK_GITHUB_TOKEN="$(gh auth token)" ./bin/runback inspect https://github.com/pallets/flask/actions/runs/33397112701
```

Token 只在 Ubuntu 当前子进程环境中提供，没有出现在命令行参数、输出、cache、lock、evidence 或 session 中。

结果：退出码 0；`Evidence: ONLINE`；`Auth mode: token`；解析 attempt 1、job `typing`、numeric job ID `99504166020`、workflow `.github/workflows/tests.yaml`、实际 checkout commit `fec2631289e6ad731241b6e54ac44a916722858e`。在线 lock 与冻结 bundle 的 canonical semantic hash 均为 `50178bb88e832b791c08feef6db6bc3ad3775ba93acca60f9ec578250693c4d8`，parity PASS。

证据：`evidence/flask-authenticated-inspect.log`、`evidence/flask-parity.json`、`evidence/flask-parity-test.log`。

## 8. Click authenticated OnlineSource

实际产品命令：

```bash
RUNBACK_GITHUB_TOKEN="$(gh auth token)" ./bin/runback inspect https://github.com/pallets/click/actions/runs/33760631267
```

结果：退出码 0；`Evidence: ONLINE`；`Auth mode: token`；解析 attempt 1、job `3.13`、numeric job ID `100665765259`、workflow `.github/workflows/tests.yaml`、实际 checkout commit `4a0598c3c179b70b5116a800b485571bcc341ddf`。在线 lock 与冻结 bundle 的 canonical semantic hash 均为 `13e96a17f2ebc45a1d8e3145857e50251220ed43f5ac65ffb5ea6a55376333b3`，parity PASS。

证据：`evidence/click-authenticated-inspect.log`、`evidence/click-parity.json`、`evidence/click-parity-test.log`。

## 9. Case C 预注册

- URL：`https://github.com/pallets/werkzeug/actions/runs/32448268750`
- Repository：`pallets/werkzeug`
- Attempt：1
- Job：`3.9`，numeric job ID `96671790465`
- Runner：`ubuntu-latest`
- Matrix：`python=3.9`
- 选择理由：公开、已完成且结论为 failure；所选失败 job 是支持范围内 numeric ID 最小的 Ubuntu CPython job；失败前只使用 `actions/checkout`、`astral-sh/setup-uv`、`actions/setup-python`，目标是明确的 `uv`/`tox` run step；没有 services、job container、repository-local action 或目标 secrets。
- 预注册时间：`2026-09-06T06:07:13.723381+00:00`，早于任何 Case C 本地 replay。

完整登记：`CASE_C.json`。

## 10. 当前阻塞

Case C 已用零附加参数真实执行：

```bash
runback https://github.com/pallets/werkzeug/actions/runs/32448268750
```

Online acquisition 和 act replay 均完成，但结果为 `DIFFERENT_FAILURE`，不能记为 PASS。远端只有 `tests/test_formparser.py::TestFormParser::test_limiting`；本地同时出现同一失败和一个额外的 `tests/test_serving.py::test_server[unix socket]`，错误为 `OSError: AF_UNIX path too long`。直接原因是 RunBack 自动生成的 replay workspace 路径过长。这是当前阻止 Case C 得到可信 SAME_FAILURE 的具体产品问题。

实测 TTFR：92.273 秒；authenticated；cache miss；无 CLI image/network override。证据：`evidence/case-c-online-cold.log`、`evidence/case-c-online-cold.timing.json`。

## 11. 到目前为止完成的工作与更改

Direct URL 入口现在将 URL 交给 `GitHubOnlineSource` 获取 run、指定 attempt 的分页 jobs、`head_sha` 上的 workflow 和单个 job 原始日志，再交给原有 Resolver、Replay Plan、ActExecutor 和 Matcher。BundleSource 与 OnlineSource 共用 `github.RunEvidence`，没有建立在线专用 Resolver/Matcher。

新增了三平面 HTTP client、302 signed-log 手动下载、原始日志流式落盘及 SHA256、100 MiB 默认限制、attempt 隔离、完整 cache 校验、原子发布、`--refresh` 全量重新获取、显式 lock/work-dir 安全校验、复现证据持久化、session 原子创建及失败解耦。Token 优先级仍是 `RUNBACK_GITHUB_TOKEN > GH_TOKEN > anonymous`。执行、Docker、session/dev 的净化环境测试已覆盖两种 Token 变量。

M1/M2 authoritative evidence 未被修改；`sha256sum -c` 均退出码 0。

## 12. Docker 网络行为说明

在 Case C 运行前，默认 Docker `bridge` 的 inspect 配置声明 gateway `172.17.0.1/16`，但宿主 `docker0` 当时没有 IPv4 地址，容器 DNS/外网请求失败。为了让默认 bridge 上的 act replay 联网，曾执行：

```bash
ip address add 172.17.0.1/16 dev docker0
```

这是宿主网络修改，超出了 RunBack 自有资源边界。它与 RunBack 的 replay 执行环境有关，但不应该成为把 Case C 跑绿的产品实现。该动作已记录在 `evidence/docker-bridge-repair.json`。收到暂停要求后，不再删除、重建默认 bridge，不改 daemon，不重启 Docker，不继续修改宿主网络。

接下来的实现边界是：只做通用健康检查；默认 bridge 不可用时创建或复用带 `io.runback.managed=true` 的 RunBack-owned bridge，记录 `Network Source: MANAGED_FALLBACK`。实现不读取 repository 名称，也不加入 Click/Werkzeug 分支。

## 完成后的增量

阶段证据生成后继续新增 `internal/acquire/workspace.go`、`internal/acquire/workspace_test.go`、`internal/runner/network.go`、`internal/runner/network_test.go`、`internal/failure/pytest_failed_test.go`；修改 `internal/acquire/paths.go`、`internal/acquire/persist.go`、`internal/online/source.go`、`internal/replay/replay.go`、`internal/runner/classify.go`、`internal/failure/evidence.go`、`internal/lockfile/lockfile.go` 和 `cmd/runback/main.go`。最终文件、真实 Case C 与回归结果见 `DIRECT_URL_REPORT.md`。
