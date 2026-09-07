# M2 实测报告

M2 已跑通同一个真实 Click failure 的完整调试闭环：

失败 URL → act FULL_JOB SAME_FAILURE → 自动激活 session → dev 中源码修改 → Fast Replay 通过 → 独立 act FULL_JOB 验证通过。

## 固定案例与环境

- URL: https://github.com/pallets/click/actions/runs/33760631267
- Attempt: 1；job: tests / 3.13；numeric ID: 100665765259
- Actual PR merge commit: 4a0598c3c179b70b5116a800b485571bcc341ddf
- Python: 3.13.15；uv: 0.12.9（从原日志提取并固定）
- Image override: runback-ubuntu:24.04-less
- Image ID: sha256:cc542dcefdd558c658863b1f9542ba7bbe6e28a37466222ed8fc92a0596ab1c7
- Network: runback-replay
- Session: 33760631267-100665765259-1788655911544661446
- Worktree: /root/.runback/sessions/33760631267-100665765259-1788655911544661446/worktree

M1 保留为历史证据。M2 仍使用 Windows 导出的真实 public API bundle，Ubuntu 执行真正的源码拉取、容器和测试。

## 实际结果

| 阶段 | 结果 |
|---|---|
| URL 重新复现并创建会话 | SAME_FAILURE；remote 1 / local 1 / matched 1 |
| 未修复的 Fast Replay | SAME_FAILURE；原 test_light_imports / AssertionError |
| 真实 dev CLI PTY | 进入 /github/workspace；容器修改同步到 host；退出后容器不存在、改动仍在 |
| 修复后 Fast Replay | STEP_PASSED_UNVERIFIED |
| 修复后完整 Verify | FULL_JOB_PASSED；原目标 step 明确成功；整个 job exit 0 |
| 测试结果 | 1991 passed, 24 skipped, 31000 deselected, 1 xfailed |
| 原失败与修复隔离 | 修复后再次 reproduce --session，原副本仍 SAME_FAILURE，worktree patch 校验值不变 |
| 依赖文件内容变化 | 检测到变化并重建 .venv/.tox；恢复文件后重新验证通过 |

修复是将 src/click/types.py 中仅用于类型注解的 import builtins 移入已有 TYPE_CHECKING 分支。
没有放宽测试断言，没有修改 workflow。最终 verify 复制的 tracked changes 仅 src/click/types.py。
这是对该失败 job 的验证；没有声称已验证仓库所有其他 job。

## 同条件性能结果

时间为实际 CLI launch-to-exit，单位秒：

| 组 | warm FULL_JOB | warm Fast Replay |
|---|---:|---:|
| 1 | 9.521 | 5.459 |
| 2 | 9.453 | 5.507 |
| 3 | 9.464 | 5.406 |
| 中位数 | **9.464** | **5.459** |

speedup_ratio = **1.734**，每轮节省约 **4.00 秒**。Fast Replay 达到了 15 秒 soft target。

第一次 Fast Replay（初始化项目依赖）约 17.30 秒，不属于热缓存样本。
完整 job 在此次热缓存环境下也只有约 9.46 秒，因此不能宣称从 M1 的 42.72 秒缩短到 5.46 秒。

两者使用同一台主机、同一 session、同一 image ID，共享 session uv 下载缓存。
FULL_JOB 每次使用新 checkout 并重建项目环境，Fast Replay 使用同一 worktree 内的 .venv/.tox。
uv/tox 的同步和项目安装仍执行，安装输出已记录。热样本未观察到 Downloading/Downloaded 输出，但未做抓包，不能声称完全离线或零网络访问。
原始样本和条件见 evidence/performance.json；每次完整日志一并保留。

## 过程中修复的问题

1. 默认 Docker bridge 丢失网关地址，造成容器 DNS 超时。主机联网正常。创建 RunBack 专用 bridge，并通过记录在 lock 中的 --network 使用；未修改默认 bridge 或重启其他服务。
2. M1 的 setup-uv 跟随 latest，原 run 为 0.12.9，M1 重放曾使用 0.12.10。M2 增加准确 uv 版本解析和 pin；保留 M1 原始记录。
3. Verify 需要明确目标 step 的成功事件。增加 target_succeeded，跳过步骤不能算修复成功。
4. 二进制 diff 和 NUL 分隔路径必须保留原始字节，不能对其 TrimSpace。变更复制测试覆盖 staged、删除、重命名、二进制、带空格文件名、untracked 与 ignored 文件。
5. 依赖清单变化不能继续使用旧项目环境。实测证明 dependency_reset 与必要安装会发生。

## 直接使用现有会话

~~~bash
ssh <ubuntu-host>
cd /root/runback-work/runback
./bin/runback dev
./bin/runback replay --step
./bin/runback verify
~~~

当前 active worktree 已保留上述源码修复。运行 reproduce --session ID 会在另一个干净副本中重现原失败，不覆盖 worktree 的修复。

要从 URL 创建新会话：

~~~bash
./bin/runback https://github.com/pallets/click/actions/runs/33760631267 --bundle docs/m1/evidence/case-b-bundle.json --lock /root/runback-work/cases/case-b.lock --work-dir /root/runback-work/temp/m2-create --image runback-ubuntu:24.04-less --network runback-replay
~~~

## 检查与边界

go test ./...、go vet ./...、CLI 编译通过。新增测试覆盖 active/explicit session、互斥、共用挂载和镜像、凭据隔离、完整变化复制、路径安全、依赖失效、准确 uv pin、跳过目标步骤不算验证通过。

M2 首先支持能由 checkout + setup-uv + setup-python 建立环境的窄范围 Python/uv job。遇到其他前置 setup，明确 DEBUG_UNSUPPORTED；没有宣称泛化到任意 workflow 或 JS/Go 的调试环境。
方案定义见 SPEC.md。没有发布 GitHub 仓库或进入其他里程碑。

完整日志、状态快照、源码 patch、最终 verify.json、性能数据、测试输出均在 evidence/。宿主凭据和 .venv/.tox 不在交付包内。
