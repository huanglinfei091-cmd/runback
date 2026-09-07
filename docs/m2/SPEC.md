# M2 冻结版：Debug → Fast Replay → Full Verify

## 范围与判定
继续 M1 的 Resolver、Replay Plan 和 ActExecutor。M2 首个实测对象为 Click run 33760631267 的 Python 3.13 job。
新增窄范围的 Python/uv Docker 调试执行器，不实现通用 workflow engine、智能缓存或持久 dev 容器。
未知的前置 setup 动作会明确返回 DEBUG_UNSUPPORTED；不会假装已恢复这些动作的环境。

只有干净源码上的完整 job 得到 SAME_FAILURE，才创建一个新 session 并激活。
已有 session 永不自动重置；再次复现创建新的 ID，保留之前的修改。

## 会话与源码
路径 ~/.runback/sessions/<id>/：
- original/：从已验证的实际失败 commit 初始化的基准源码。
- worktree/：独立 Git checkout，唯一允许用户调试修改的源码。
- cache/：会话工具和下载缓存。
- runback.lock、state.json、reproduction.json：上下文、镜像与原始复现结果。
- logs/、full/、verifications/：调试、基线和隔离验证证据。

这里 worktree 是目录名，使用独立 clone，未使用 Git linked-worktree 的外部 .git 指针；因此在容器里 git status 也能正常运行。

默认 dev、replay --step、verify 使用 active session；均可 --session ID 显式选择。
同一 session 的写操作使用互斥锁，防止 dev 与 verify/fast replay 同时改动。
如果进程被强制 kill，busy 文件会阻止下一次操作；检查 PID 后才清理，不自动猜测旧会话是否仍在使用。

## 调试环境与依赖
dev 和 fast replay 均 bind 同一个 worktree 到 /github/workspace。
容器使用 --rm，退出或被中断后仅清理本次命名的容器。
允许持久化：
- worktree 的 .venv/.tox；
- cache 中的 uv 下载缓存；
- 从已经验证的 act 容器中导出的准确版本 Python/uv 工具文件。

工具仍挂到原容器路径，避免虚拟环境解释器路径变化。不是 docker commit，也不保存整个容器根文件系统。
镜像使用 state 中的 image_override 及已解析的 image_id。Click 继续使用 runback-ubuntu:24.04-less。
宿主的 GitHub、AWS、SSH 凭据不进入 Docker/act/Git 子进程。

manifest/lockfile 内容发生变化时，保守移除该 session worktree 的 .venv/.tox，再运行原命令进行同步安装。
即使 manifest 未变，也执行原 uv/tox 命令，保留它的依赖同步及项目重新安装行为；不缓存测试结果。
环境内容指纹覆盖 pyproject.toml、uv.lock、tox.ini、requirements*.txt、setup.py/setup.cfg、Pipfile*、.python-version。
不声称这是覆盖所有语言或配置文件的通用失效机制。

## 命令与结果
~~~bash
runback <failed-run-url> [--bundle evidence.json] [--image IMAGE] [--network NETWORK]
runback dev [--session ID]
runback replay --step [原失败步骤名称] [--session ID]
runback verify [--session ID]
runback reproduce --session ID
~~~

省略 --step 的名称时，使用已记录的失败步骤。M2 不执行任意未建模步骤。
fast replay 成功只报告 STEP_PASSED_UNVERIFIED；结构化失败匹配仍可报告 SAME_FAILURE，但 mode 为 STEP_ONLY。
修复成功必须由 verify 的 FULL_JOB_PASSED 确认：act 整个 job exit 0，且原目标步骤有明确 Success 事件。
跳过目标步骤不能算验证通过。

## Verify
从 original 在独立目录创建干净源码，然后：
1. 使用 git diff --binary <原始commit>（覆盖 staged、unstaged、提交在本地的修改、删除、重命名和二进制变化）。
2. 使用 git ls-files --others --exclude-standard -z 收集未跟踪文件，逐个复制，完整清单写入 changes.json。
3. 不复制未跟踪且 ignored 的 .venv/.tox 等文件。
4. 保留安全的相对 symlink；拒绝逃出 checkout 的 symlink 和 symlink 父目录。
5. 用原锁定的 workflow、原 job/matrix 和准确工具版本执行 act FULL_JOB。

如果 .github/workflows/** 改变，必须输出 WORKFLOW_MODIFIED。
M2 明确选择 ORIGINAL_WORKFLOW_WITH_LOCAL_WORKFLOW_CHANGES_IGNORED：验证原 CI 配置，不执行修改后的 workflow。
这比自动猜测修改后 workflow 的 job/matrix 更保守；报告绝不隐藏此选择。
Click 的实测修复只修改 src/click/types.py，不修改测试和 workflow。

## 性能与网络
使用同一 session、同一 image ID，先准备 act/uv 下载缓存，再交错测 3 次完整 job 和 3 次 fast replay。
全部由外部脚本从 CLI 进程启动计时到退出，报告中位数与每次原始样本。
完整 job 每次创建新的项目环境；fast replay 复用 worktree 的项目环境，这是比较的唯一主要环境复用差别。
注明缓存状态、安装输出、观察到的下载；没有下载日志不等于网络完全关闭。
15 秒只是 soft target。不使用 M1 的 42.72 秒计算 speedup。

本机 Docker 默认 bridge 丢失 IPv4 网关。为避免改动其他项目，创建带 io.runback.managed=true 标签的 runback-replay 网络。
--network 是记录进 lock 并由 session 继承的普通网络 override，不改默认 Docker/系统配置。

## 验证脚本
- measure.py：对仍未修改源码的 active session 执行同条件性能测量。
- dev_smoke.py：仅用于这个真实 Click 案例，通过真实 CLI PTY 修改源码、检查 bind 持久化和容器删除。
- 这些是 docs 下的案例验证脚本；产品执行器没有按仓库名称分支。

实现参考：[Docker run 的临时容器与挂载](https://docs.docker.com/reference/cli/docker/container/run/)、[uv 的缓存与 Python 路径变量](https://docs.astral.sh/uv/reference/environment/)。
