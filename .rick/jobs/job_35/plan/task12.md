# 依赖关系
task1, task2, task8, task9, task10, task11

# 任务名称
三个 O 端到端验收 + README/wiki 文档同步

# 任务目标
以用户视角做端到端验收，确认三个 O 全部落地，并同步用户面向文档（README.md + wiki/，不修改 `.rick/domain/`）。

三个 O 验收清单（收敛版）：
- O1（spec 信息内核）：`.rick/domain/spec.md` + `.rick/domain/rick-spec.md` 存在且含四要素 + 验收标准
- O2（三层金字塔 + 做薄）：`internal/{cmd,handler,env,builder,runtime}` 存在且职责与 spec 对应；`rick` 全命令可用；已删除 `internal/{executor,parser,actpath,logging,git,agent}`
- O3（pibuilder pi 对齐）：模板中 `workflowScript`/`runs.run` 出现 >0；think/research/exporter 落盘为 pi agent；门禁由 rick-gates hook + 脚本承载；自然语言触发词下降

参考：loop `readme-wiki-sync-loop`（README/wiki 同步，**禁止修改 .rick/domain/**）；skill `verify_go_changes_skill`、`command_registration_verification_skill`、`check_mechanism_skill`（仅 learning_check 相关）；domain/testing-conventions.md「go test 范围精确性」。

# 关键结果
1. 三 O 验收逐项通过：
   - O1：`test -f .rick/domain/spec.md && test -f .rick/domain/rick-spec.md`；`for w in 模块边界 职责 接口契约 验收标准; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done`；`grep -q '功能等价' .rick/domain/rick-spec.md`；`grep -qE 'dry-run|go test|集成测试' .rick/domain/spec.md`（可操作判据被枚举）
   - O2：`ls internal/cmd internal/handler internal/env internal/builder internal/runtime` 均存在；`for d in executor parser actpath logging git agent; do test ! -d internal/$d || exit 1; done`（6 冗余包全删）
   - O3：`grep -rc workflowScript internal/prompt/templates/ | grep -v ':0' | wc -l` ≥1；显式 `export RICK_PI_AGENT_DIR=<temp>`（或改用 `~/.rick/pi/agent` 默认路径）后，`$RICK_PI_AGENT_DIR/agents/{think,research,exporter}.md` 3 文件存在 + `$RICK_PI_AGENT_DIR/extensions/rick-gates/` 已部署；`test -f .rick/skills/rick-gates/helper.py`；自然语言触发词计数下降（正则口径与 RFC §2.1 基线一致：含 bare `subagent`/`Sub Agent`/`SPAWN`/`子 Agent`，迁移前后用同一正则对比，迁移后 < 迁移前基线）
2. 全命令可用：`./bin/rick --help` + 8 子命令 `--help` 无 panic；`plan --dry-run`/`doing job_35 --dry-run`/`human-loop --dry-run`/`easy --dry-run`/`learning --dry-run`/`dream --dry-run`/`ctrl --dry-run --job job_N` 含 pi 触发语法且无 `{{`
3. 测试全绿：`go build ./...`（整仓编译兜底）+ `go test ./internal/config/... ./internal/env/... ./internal/runtime/... ./internal/builder/... ./internal/prompt/... ./internal/handler/... ./internal/cmd/... ./internal/workspace/... -timeout 60s`；**同步改写 `tests/tools_integration_test.sh`**（task8 已删 plan_check/doing_check，该脚本场景 1,2,3,4,5,6,11 依赖被删命令需删除或改写为 pi 侧门禁脚本验证；场景 7 只删 merge/branch 断言、保留 learning_check 断言）后再跑，并与 mock_agent 对齐
4. README.md + wiki/ 同步三层金字塔 + spec 信息内核 + env 四职责 + 下沉策略（只读引用 `.rick/domain/`，不写）；`git status .rick/domain/` 无本 task 引入的变更
5. 扩展点验收：`grep -q 'type Runtime interface' internal/runtime/*.go`、`grep -q 'type RuntimeEnv interface' internal/env/*.go`、`grep -q 'type RuntimeBuilder interface' internal/builder/*.go` 三接口就位；`grep -rniE 'type .*dsh|dshRuntime|dshEnv|dshBuilder|NewDsh' internal/ cmd/` = 0（无 dsh 类型/实现/构造；代码注释中的「dsh」豁免）；`grep -rn 'piRuntime' internal/handler/` 无命中（handler 依赖接口非具体实现）；`grep -q '"runtime"' internal/config/config.go` 字段就位
6. 真实运行验收（**硬门，功能等价的核心行为验收，不可 skip**）：`rick tools init-pi` 全 ✅ → `rick doing job_N` 按依赖执行 → `doing/tasks.json` 状态机正确、`doing/session_id` 落盘、门禁脚本在 `agent_settled` 后执行；无 pi/API key 环境**须由 supervisor 手动执行并记录结果**（结构 grep 不能替代行为等价）

# 测试方法
（本 task 测试用例设计遵循 skill:tdd + skill:testing-anti-patterns：先写失败测试；测试真实端到端行为不 mock；验证命令基于真实接口。）

1. 正常路径（三 O 验收）：前置条件 = task8~11 完成 + `rick tools init-pi` 已成功；输入 = 无；操作 = 依次执行 KR1 的三组断言（O1 两个 spec 文件 + 四要素关键词 + 功能等价；O2 5 目录存在 + 6 冗余包 `test ! -d`；O3 workflowScript 计数 ≥1 + 3 个 agent 文件 + rick-gates helper.py）；预期 = 全部通过。
2. 边界（命令全可用）：前置条件 = build 成功；输入 = 各子命令 `--help`；操作 = `./bin/rick --help` + 8 个子命令 `--help`；预期 = 均 exit 0、无 panic、含 plan/doing/easy/learning/dream/tools/human-loop/ctrl。
3. 异常（回滚兜底）：前置条件 = 某命令行为异常；输入 = 无；操作 = 记录当前 release commit（`git log --oneline -1`）后，在**副本 worktree** 验证回滚：`git worktree add /tmp/rick-rollback <release-commit>` + `cd /tmp/rick-rollback && ./scripts/build.sh && ./bin/rick --help`（exit 0）；预期 = 可回滚且 rick 仍可编译运行（不污染主工作区；`.rick/domain/` 无本 task 变更）。注意 task8 已删 `plan_check`/`doing_check` 命令，回滚验证改用 `./bin/rick --help`。
