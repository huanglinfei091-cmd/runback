# M1: Real Failure Reproduction

两例预先固定的真实 public GitHub Actions 失败，均已走完 URL → Resolver → Replay Plan → act FULL_JOB → 本地失败 → 结构化证据比较，结果为 SAME_FAILURE。以下是两次实际执行的证据，不是总体 reproduction rate。

## 实测结果

| 项目 | Case A: Flask | Case B: Click |
|---|---|---|
| Run URL | https://github.com/pallets/flask/actions/runs/33397112701 | https://github.com/pallets/click/actions/runs/33760631267 |
| Job / numeric ID | typing / 99504166020 | tests (3.13) / 100665765259 |
| Actual checkout | fec2631289e6ad731241b6e54ac44a916722858e | 4a0598c3c179b70b5116a800b485571bcc341ddf |
| Workflow | .github/workflows/tests.yaml | .github/workflows/tests.yaml |
| Matrix | 无 | python=3.13 |
| Actual Python | 3.14.7 | 3.13.15 |
| Runner | ubuntu-latest | ubuntu-latest |
| Executor / mode | act / FULL_JOB | act / FULL_JOB |
| Target reached | true | true |
| Evidence | STRUCTURED / mypy | TEST / pytest |
| Remote / local / matched failures | 1 / 1 / 1 | 1 / 1 / 1 |
| Remote / local exit | 1 / 1 | 1 / 1 |
| Result | SAME_FAILURE | SAME_FAILURE |
| CLI measured time | 85.58 s | 42.72 s |
| Workspace dirty before execution | false | false |

Case A 从已收集的案例中选取，是 Python typing failure，不是 pytest assertion。Case B 验证了 pytest、matrix、uv、tox 和历史 PR merge commit。两例均恢复实际 checkout SHA，而非仅使用 PR head SHA。上述耗时不含首次下载镜像，不能作为冷启动性能指标。

## Remote / Local Evidence

Case A 两侧完全相同：

- Identity: src/flask/json/provider.py:117:mypy[unused-ignore]
- Exception/diagnostic: mypy[unused-ignore]
- Source: src/flask/json/provider.py:117
- Message: Unused "type: ignore" comment
- Exit: 1

Case B 两侧完全相同：

- Test: tests/test_imports.py::test_light_imports
- Exception: AssertionError
- Source: tests/test_imports.py:80
- Message: assert 'builtins' in {'__future__', 'abc', 'codecs', 'collections', 'collections.abc', 'configparser', ...}
- Exit: 1

两个 result.json 都包含 remote/local 原文摘录、结构化 items、失败总数和 matched_fields。Identity、Exception、File、Line、Message、Exit 六项均为 true。原日志中 message 本身有省略号；比较的是可观察的断言文本，不声称恢复省略集合的全部成员。

## Act compatibility 与环境修复

先在独立目录对 Case A 原始失败 commit、原始 workflow 做真实 act smoke test，执行到原 typing 步骤并捕获 exit 1，结构化证据相同。记录在 evidence/case-a-original-act*，耗时约 207.22 s。

前期基础镜像下载中断，续传后校验 SHA256 并完成导入。原 act action cache 下载完整 Git 历史耗时过长，改用 act 自带 --use-new-action-cache。没有开发新的 cache 或 executor。

Click 首次完整执行重现了原 test_light_imports，同时出现额外 24 个分页器失败：remote 1 / local 25 / matched 1，正确结论是 DIFFERENT_FAILURE。容器缺少 less，实际测试源码使用 less。归因为 ENVIRONMENT_MISMATCH，不属于 ACT_INCOMPATIBILITY。

修复仅在固定基础镜像上安装 Ubuntu less，使用通用 --image override 记录进 lock，再重新执行整个失败 job。没有修改 Click 源码、测试、失败命令，代码中没有按仓库名处理的分支。新的完整执行仅余原失败。

- Base digest: sha256:62d572b92f9f32d3427b6d220ad1f9dca9c7b6ffad37d295425037dbff78abaf
- Custom image: runback-ubuntu:24.04-less
- Custom image ID: sha256:cc542dcefdd558c658863b1f9542ba7bbe6e28a37466222ed8fc92a0596ab1c7
- less: 590-2ubuntu2.1 amd64
- less.deb SHA256: f26481c7f5e492e7536f9e75a9aa237f4772b637cdc933f356bc567e180b8a39

evidence/case-b-initial-result.json 是修复计数前的历史产物，不能作为初次失败数量的依据。对应原始完整日志不变；evidence/case-b-initial-evidence.json 是最终代码重新分析的结果，准确统计 25 个失败，未能解析的 24 个也逐一列出。

## 重放命令

在 Ubuntu 现有目录 /root/runback-work/runback 中执行：

~~~bash
go build -o bin/runback ./cmd/runback

./bin/runback https://github.com/pallets/flask/actions/runs/33397112701 --bundle docs/m1/evidence/case-a-bundle.json --lock /root/runback-work/cases/case-a.lock --work-dir /root/runback-work/temp/m1-case-a --timeout 8m

./bin/runback https://github.com/pallets/click/actions/runs/33760631267 --bundle docs/m1/evidence/case-b-bundle.json --lock /root/runback-work/cases/case-b.lock --work-dir /root/runback-work/temp/m1-case-b --image runback-ubuntu:24.04-less --timeout 8m
~~~

--bundle 是 Windows 通过 GitHub API 导出的真实 public 元数据与原始日志副本，没有模拟本地执行。Ubuntu 从 URL 校验 bundle 身份并运行同一个 Resolver，随后真实拉取代码和执行 act。采用复制证据的方式避免向 Ubuntu/Replay 传递 GitHub token。普通在线入口仍为 runback URL；可用现有 scripts/export-evidence.ps1 从有读取日志权限的 Windows 环境导出证据。

本次验证的是 copied-evidence 模式下的真实重放，不声称 GitHub SSH key 能读取 Actions HTTP 日志。依赖下载仍需网络，历史日志也可能过期。

在其他 Linux 主机重建 Click 基础镜像：

~~~bash
mkdir -p /tmp/runback-runner-image
cp docs/m1/Dockerfile.runner /tmp/runback-runner-image/Dockerfile
cd /tmp/runback-runner-image
curl --fail --location -o less.deb https://mirrors.ustc.edu.cn/ubuntu/pool/main/l/less/less_590-2ubuntu2.1_amd64.deb
echo 'f26481c7f5e492e7536f9e75a9aa237f4772b637cdc933f356bc567e180b8a39  less.deb' | sha256sum --check
docker build -t runback-ubuntu:24.04-less .
~~~

## 验证及交付范围

go test ./...、go vet ./...、go build 均通过。真实捕获集成测试另行验证 Case A、Case B 为 SAME_FAILURE，Click 首次执行为 DIFFERENT_FAILURE，输出存于 evidence/validation.log。普通单元测试不会访问 GitHub 或运行 Docker。

本轮修复覆盖：act 日志的 job/step/stage 归属、实际版本锁定、完整 job 执行、两级证据、额外未解析失败计数、pytest 参数空格保留、过滤包含 Error 字样的通过测试行。STEP-only 最多 LIKELY_MATCH；结构化证据不完整不升级 SAME_FAILURE。

结果仅证明这两例在记录环境中的重放。runner 镜像并非 GitHub VM 的完全副本；没有推广为 JS/TS/Go 真实复现验证，也没有发布仓库或扩展 M1 之外功能。

代码主副本：

- Windows: E:\linux的项目\github\1\runback
- Ubuntu: /root/runback-work/runback

完整证据见同目录 evidence/：原始远程日志、完整 act JSONL、本次 result.json、CLI 输出、lock、replay.yml、真实 API bundle 及原始 act smoke test。最终代码重新判定的结果另存 case-a-verified-evidence.json、case-b-verified-evidence.json。SHA256SUMS 可校验报告与证据文件。
