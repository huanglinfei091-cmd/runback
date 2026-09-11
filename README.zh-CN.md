# RunBack

**把失败的 GitHub Actions run 搬到本地复现。**

不要为了调试 CI 反复提交、推送、等待。

```bash
runback https://github.com/owner/repo/actions/runs/123456789
```

```text
GitHub Actions failure
        ↓
runback <URL>
        ↓
本地 SAME_FAILURE
        ↓
runback dev
        ↓
修改代码
        ↓
runback replay --step
        ↓
runback verify
```

[English](README.md) · [架构](docs/architecture.md) ·
[已知限制](docs/limitations.md) · [Direct URL 证据](docs/online/DIRECT_URL_REPORT.md)

## RunBack 是什么

输入一个已完成且失败的 GitHub Actions run URL，RunBack 会获取失败发生时的
workflow、commit、job、matrix、runner 和日志，生成本地重放计划，调用
[act](https://github.com/nektos/act) 执行，并比较远程与本地失败。

RunBack 不是 act 替代品。act 负责运行 workflow；RunBack 负责还原“这一次失败”应该
怎样运行，并在证据不一致时拒绝宣称成功。

## 快速开始

Linux amd64 用户无需 Go、无需 sudo 即可安装。脚本会下载公开 Release、校验 SHA256，
并且只写入 `~/.local/bin`：

```bash
curl -fsSL https://raw.githubusercontent.com/huanglinfei091-cmd/runback/main/scripts/install-release.sh | bash
export PATH="$HOME/.local/bin:$PATH"
runback doctor
```

如需安装到系统目录，可从
[Release 页面](https://github.com/huanglinfei091-cmd/runback/releases/tag/v0.1.3-alpha)
下载 archive 与 `SHA256SUMS`，校验后安装：

```bash
sha256sum -c SHA256SUMS
tar -xzf runback-v0.1.3-alpha-linux-amd64.tar.gz
sudo install -m 0755 runback-v0.1.3-alpha-linux-amd64/runback /usr/local/bin/runback
```

## 可复制 Demo

下面是 RunBack 项目预注册的 Werkzeug Case C，是已知 Demo，不是外部用户案例：

```bash
runback https://github.com/pallets/werkzeug/actions/runs/32448268750
```

## 复现你自己的 CI 失败

复制一个已完成且失败的 public GitHub Actions run URL：

```bash
runback https://github.com/OWNER/REPOSITORY/actions/runs/RUN_ID
```

得到 `SAME_FAILURE` 后，RunBack 会创建调试 session 并显示工作目录：

```bash
runback dev
# 修改工作目录中的源码
runback replay --step
runback verify
```

## 环境要求

- Linux amd64；已验证目标为 GitHub Actions 的 Ubuntu runner。
- Git。
- 已启动 daemon 的 Docker Engine。
- [act](https://nektosact.com/installation/index.html)；Alpha 验证使用 0.2.89。
- 可以访问 GitHub 和 workflow 的 action 依赖。

只有从源码编译时才需要 Go 1.24+：

```bash
bash scripts/install.sh
$HOME/.local/bin/runback doctor
```

安装脚本只写入用户目录，不会安装或修复 Docker、act、Git、firewall、Docker daemon、
docker0 或宿主网络。

## `runback doctor`

`doctor` 检查 Git、Docker、daemon、act、GitHub API 模式和额度、短 replay workspace、
磁盘空间、PATH 和 Docker 网络。它只诊断，不自动修复宿主机。失败项会给出
`Problem`、`Cause` 和可以继续执行的 `Next`。

公开仓库允许匿名访问，未配置 Token 时只会提示 API 额度较低。可选认证优先级为：

```text
RUNBACK_GITHUB_TOKEN > GH_TOKEN > anonymous
```

RunBack 不会自动读取现有的 GitHub CLI 登录。

## 真实验证案例

| 案例 | 已验证链路 | 真实结果 |
| --- | --- | --- |
| Flask | Bundle 取证 → resolver → full replay → matcher | `SAME_FAILURE` |
| Click | Bundle replay → session → dev → step replay → full verify | 修复前 `SAME_FAILURE`，修复后 `FULL_JOB_PASSED` |
| Werkzeug Case C | 认证 Direct URL → online evidence → full replay → matcher | `Remote 1 / Local 1 / Matched 1`，`SAME_FAILURE` |
| Ramose Case D | 认证 Direct URL → setup-uv/cache → Pyright | 首次 `REPLAY_BLOCKED`，通用解析修复后严格 `SAME_FAILURE` |
| Aiden Firmware Case E | 认证 Direct URL → setup-go → Go test | 首次 `INSUFFICIENT_EVIDENCE`，通用解析修复后严格 `SAME_FAILURE` |
| DeHub Mobile Case F | 认证 Direct URL → setup-node/cache → 自定义 i18n 检查 | 真实保留 `INSUFFICIENT_EVIDENCE`，没有添加仓库专用解析 |
| ClickHouse UI Case G | 认证 Direct URL → Node 24/Yarn → TypeScript build | 首次 `INSUFFICIENT_EVIDENCE`，增加通用 `tsc` 解析后严格 `SAME_FAILURE`，`3/3/3` |
| Svelte Widgets Case H | 认证 Direct URL → Node 24/npm → Vitest | 首次 `INSUFFICIENT_EVIDENCE`，一次安装重试真实保留 `REPLAY_BLOCKED`，增加通用 Vitest 解析后严格 `SAME_FAILURE`，`4/4/4` |

Werkzeug Case C 在本地重放前已经登记。中间出现的 `DIFFERENT_FAILURE`、
`REPLAY_BLOCKED` 和 `INSUFFICIENT_EVIDENCE` 都被保留，详情见
[Direct URL 报告](docs/online/DIRECT_URL_REPORT.md)。
Fresh Case D 到 H 的首次失败、通用问题和最终结果保存在
[真实兼容记录](docs/compatibility/REAL_CASES.md)。

## 兼容范围和结果

Alpha 聚焦 public GitHub.com、已完成的失败 run、`ubuntu-latest`，以及普通
JavaScript/TypeScript、Python、Go workflow。支持静态 matrix 和 act 可以执行的标准
checkout/setup/run step。

Windows、macOS、self-hosted runner、private repo、GitHub Enterprise、动态 matrix、
reusable workflow、services、job container、本地 action、依赖 secrets 和大量 artifacts
的 workflow 暂不支持。

- `SAME_FAILURE`：远程与本地结构化证据满足 Matcher 标准。
- `DIFFERENT_FAILURE`：本地出现了另一种失败。
- `INSUFFICIENT_EVIDENCE`：证据不足，不能证明相同。
- `REPLAY_BLOCKED`：准备或执行阶段无法完成可信比较。
- `EVIDENCE_UNAVAILABLE`：无法取得必要 GitHub 证据。

只有非零退出码绝不算复现成功。

## 安全模型

GitHub 凭据只进入在线取证。它不会进入 Git、act、Docker、workflow、dev/replay
container、session、lock、cache、evidence、log 或命令行。job log 的 signed redirect
由不带 Authorization 和 Cookie 的独立客户端下载。

RunBack 给 act 使用空 secret、variable、environment 和 input 文件，不会伪造 GitHub
Secrets。运行陌生 workflow 前，应先查看其代码。

## 报告问题

使用 [Bug report 表单](https://github.com/huanglinfei091-cmd/runback/issues/new?template=bug_report.yml)，
提供 RunBack version、OS、Docker/act version、失败 URL、Result、Stage 和 Cause。
**不要提交 Token、Secret、Cookie、私有源码或其他凭据。**

## 项目状态

RunBack 当前是 early alpha，已有八条保留的真实案例证据链，但兼容范围仍然有限。

如果你有一个 public GitHub Actions failed run，可以在
[Alpha 测试讨论](https://github.com/huanglinfei091-cmd/runback/discussions/1)中提交 URL 和
真实 verdict。`SAME_FAILURE`、`DIFFERENT_FAILURE`、`INSUFFICIENT_EVIDENCE`、
`REPLAY_BLOCKED` 和 `EVIDENCE_UNAVAILABLE` 都是有价值的 Alpha 结果。不要提交 token、
secret、cookie、私有源码或凭据。
项目不宣称 100% GitHub Actions 兼容。开发检查：

```bash
go test ./...
go vet ./...
go build ./cmd/runback
```

MIT License。第三方仓库和 action 依赖保留各自许可证。
