# 当前进度

## Direct URL MVP — 启动

已检查 /root/runback-work/runback；现有 docs/m1/REPORT.md、docs/m2/REPORT.md 和各自 evidence/ 均存在。docs/online、docs/progress 本轮创建。工作副本无 .git，未初始化或清理。启动基线 go test ./... PASS。M1/M2 冻结，仅进行 Direct URL MVP。

## 2026-09-06 — 真实验收认证模式调整

按用户最新要求，最终真实验收改用 Ubuntu 侧手动配置的 RUNBACK_GITHUB_TOKEN / GH_TOKEN。禁止读取 Windows 凭据或通过 SSH 传递凭据。匿名模式及已记录的限流证据继续保留；不再等待匿名 reset 后执行真实验收。当前继续单元/集成测试，Direct URL 尚未验收 PASS。

## 2026-09-06 — Direct URL MVP 验收完成

锁定 Case C `pallets/werkzeug` run 32448268750 attempt 1、job 96671790465，没有更换。Linux 自动 replay workspace 已改为 `/tmp/rb/<12-hex-short-id>`，Online/Bundle 共用 allocator，显式 `--work-dir` 保持原语义。删除长 identity 路径后不再出现 `AF_UNIX path too long`。

Authenticated cold online acquisition + act replay 最终得到 `SAME_FAILURE`：Evidence level TEST，remote 1 / local 1 / matched 1，identity/exception/file/line/message/exit 全部一致。CLI 只有 run URL，无 bundle/lock/work-dir/image/network override。TTFR_ONLINE_COLD 41.564s；总退出 48.131s；自动创建原子 session 并停止 reproduction container。

Flask/Click OnlineSource 与 Bundle canonical parity 均 PASS；Flask/Click bundle plan 回归 PASS；M2 Click Fast Replay 为 STEP_PASSED_UNVERIFIED，full verify 为 FULL_JOB_PASSED。最终 `go test ./...`、`go vet ./...`、`go build ./cmd/runback` 全部退出码 0；M1/M2 SHA256SUMS 均通过，历史 evidence 未修改。

Token 只进入 Ubuntu GitHubOnlineSource。执行进程隔离测试和最终 35345 文件 live scan 均未发现 Token。匿名 GITHUB_RATE_LIMITED 真实证据保留。

Docker 网络实现只做通用 probe 和一个 `io.runback.managed=true` fallback，不含 repo 分支，不配置 subnet/gateway，不修改 daemon。早期曾手工给 docker0 添加地址，已在 online report 披露；收到约束后未再修改宿主 Docker/网络。Case C 成功时使用 GENERIC_DEFAULT，managed fallback 仅完成隔离单测，未宣称真实环境验收。

完整结果：`docs/online/DIRECT_URL_REPORT.md`；最终机器可读核验：`docs/online/evidence/final-verification.json`。

## 2026-09-07 — GitHub Public Alpha Launch 启动

M1、M2、Direct URL 和 Case C 按真实结果冻结为 PASS。本阶段不扩张 Actions 兼容性，
只进行发布安全、README、最小 doctor/version、Linux amd64 artifact、clean HOME smoke、
完整回归、公开仓库与 v0.1.0-alpha Release，以及真实外部用户验证。

初次发布扫描在 214 个文本文件中发现一处 M2 报告私网 SSH 地址；已只改为占位符。
进一步审查发现 Case C 早期 action fetch 失败日志中的两处私网主机地址，已做等价脱敏，
失败类型、远程地址、时间和 Case C 结果链均保留。复扫 214 个文本文件为 0 findings。
Windows 主副本已初始化本地 `main` Git 仓库，尚未提交、push 或创建远程。

## 2026-09-07 — Public Alpha 发布候选验证完成

Linux amd64 release archive 已生成并从 Ubuntu 复制到 Windows 控制副本，SHA256 两端一致。
唯一发布 archive 包含 `runback`、`README.md` 和 `LICENSE`。干净 HOME 安装耗时 59 ms，
`runback version` 正确显示 `RunBack v0.1.0-alpha`，authenticated doctor 为 Ready。

锁定 Case C 在无 `--bundle`、`--lock`、`--work-dir`、`--image`、`--network` 的干净安装中
得到 `SAME_FAILURE`，remote 1 / local 1 / matched 1，退出码 0，并创建 session。该环境的
default bridge 与旧 managed bridge 均无法访问 TCP 443；RunBack 只创建一个带标签、无固定
subnet/gateway 的 `runback-managed`，最终记录 `MANAGED_FALLBACK`。未修改 docker0、daemon、
宿主 IP、路由、防火墙，也未重启 Docker。

Flask/Click bundle plan 回归 PASS。旧 M2 session 的冻结网络失效后，保留其真实 NETWORK
失败，并使用相同 Click bundle、历史镜像和健康 RunBack-owned network 新建 session；原始
失败为 SAME_FAILURE，应用原历史 patch 后 step replay 为 STEP_PASSED_UNVERIFIED，full verify
为 FULL_JOB_PASSED。

最终 `go test ./...`、`go vet ./...`、`go build ./cmd/runback`、gofmt、M1/M2 SHA256 integrity
全部 PASS。发布扫描 218 个文本文件 0 findings；对当前认证 token 的精确扫描覆盖 38,762
个开发与 runtime 文件，0 matches，token 未打印或落盘。当前进入 Windows commit、公开仓库、
tag、GitHub Release 与外部用户验证。

## 2026-09-07 — v0.1.0-alpha 已公开并启动首轮真人验证

公开仓库为 `https://github.com/huanglinfei091-cmd/runback`。tag `v0.1.0-alpha` 锁定在
commit `eb3725213a0892f801ea7a07283bd90abf75b6b3`，GitHub Release 已发布为 prerelease，
不是 draft。Linux amd64 archive 和 SHA256SUMS 可匿名下载；公开下载后的 checksum 与
`a74ff36aa93bc3e3726a8955a009504f412874af5b056cdb85df6e68f74ddd83` 一致。tag commit 的
GitHub Actions run 34076426813 全部 PASS。

Day-one 用户发现从近期 public pull_request failure 建立 100 个候选池，语言分布为 Python
18、JavaScript 13、TypeScript 37、Go 32。逐个核验 PR、run、attempt、jobs、Ubuntu runner 和
失败 step 后，保留 Tier A 30。首轮从 15 个不同仓库选择普通 test/lint/format/build failure，
逐条发送带具体 run/job/step 的 early-alpha 邀请；15 条成功，0 失败，0 跳过，不求 Star，
并明确失败复现也有价值。

首轮刚完成时尚未观察到真人安装或执行，因此当前外部验证状态为
`USER_VALIDATION_PENDING`。用户随后将首次检查从 6 小时提前到 2 小时，约在 2026-09-07
05:42 UTC 检查；只有实际 tester 少于 2 才可
联系 reserve 中最多 15 人；全天直接联系上限保持 30。M1、M2、Direct URL 和 Case C 保持冻结。

## 2026-09-08 — Day-one 第二轮精准邀请

在 2026-09-08 00:08 UTC 重新检查首轮 15 个 PR。首轮评论之后出现 5 条普通开发或审查
评论，但没有任何一条提到 RunBack 安装、命令或 verdict，因此真实 reply/tester 均仍为 0。

reserve 15 个案例逐个重新验证 PR、run、attempt、job 和 Ubuntu runner：6 个 PR 已关闭；
multi-pod、Testplane 汇总结果和 Rust job 3 个案例因超出当前 Alpha 范围未联系；其余 6 个
仍开放且原 failed job 保持 completed/failure，已发送第二轮定制邀请。第二轮 6 条全部发送成功，
累计直接联系 21，未达到全天上限 30。当前仍没有已证实的安装或运行，状态保持
`USER_VALIDATION_PENDING`。24 小时最终检查点为 2026-09-08 03:42 UTC。
