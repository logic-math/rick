# 依赖关系
task5, task8, task9

# 任务名称
把自然语言 subagent 触发词等价迁移为 pi 显式触发语法

# 任务目标
按 spec（task2）落地 KR3.3：把各命令模板中自然语言 subagent 触发词（「派发 subagent」「SPAWN Sub Agent」「子 Agent」等，共 243 处：root 模板 134 处 + skills/ 109 处，见 RFC §2.1）改写为显式 pi 触发语法（`workflowScript` + `runs.run`/`runs.all` + 真实 agent 名 `agent:'worker'/'reviewer'/'think'/'research'/'exporter'`），并显式化触发权归属（编排权在 parent、普通子 agent 不持 subagent 工具、单写者 one-writer）与 SENSE 特有语义（批判门禁、反向回流、判断记录）。目标：迁移前模板中 `workflowScript`/`runs.run` 零出现（research-report-S-bestpractice.md N3.1），迁移后 >0 且自然语言触发词显著下降。

依据：research-report-S-bestpractice.md BP-1~BP-9 与 D1~D7 差距表；RFC §6 KR3.3。

参考：skill `template_injection_skill`（`{{}}` 陷阱 + dry-run 验证）、`global_ref_sync_skill`（全局替换二次确认）、`verify_go_changes_skill`、`multi_phase_protocol_skill`（批判门禁/反向回流语义的权威定义，确保迁移不丢）。

⚠️ 关键坑：模板经 `PromptBuilder` 用 `{{variable}}` 替换，`extractVariables` 会把模板里任何 `{{...}}` 当变量（job_6 task1 教训）——迁移时**不得在模板中引入非变量的 `{{`**；pi 的 workflowScript 示例用反引号/`${}` 时，若出现在 Go 测试 fixture 里按 bugs.md「Go raw string 反引号截断」处理（raw 段 + 解释型段拼接）。

# 关键结果
1. `internal/prompt/templates/` 顶层 + skills/ 中自然语言触发词改写为显式 pi 语法：`workflowScript` + `runs.run`/`runs.all` + 真实 agent 名；覆盖 sense_loop（think/research/exporter 派发）、plan（六维评审）、doing/easy（Main/Sub Agent）、dream、learning、ctrl
2. 触发权归属显式化：parent 持编排权、单写者、async/context 语义写入相关模板
3. SENSE 特有语义不丢：批判门禁、反向回流（回流上限）、判断记录（judgment.md 只写 human 原话）在 sense_loop.md 迁移后仍完整
4. 验证：`grep -rcE 'workflowScript|runs\.run|runs\.all' internal/prompt/templates/ | grep -v ':0' | wc -l` ≥1；`go build` + `go test ./internal/prompt/... -v` 全绿；`./bin/rick plan --dry-run`/`doing --dry-run`/`human-loop --dry-run` 无 `{{`

# 测试方法
（本 task 测试用例设计遵循 skill:tdd + skill:testing-anti-patterns：先写失败测试；断言真实模板内容不 mock；测试基于真实方法签名。）

1. 正常路径：前置条件 = task5/9 完成；输入 = 无；操作 = **迁移前先以验收同一正则捕获自然语言触发词基线计数并落盘**（如 `.rick/jobs/job_35/doing/trigger-baseline.txt`）→ 迁移 → `grep -rcE 'workflowScript|runs\.run|runs\.all' internal/prompt/templates/ | grep -v ':0' | wc -l`；预期 = ≥1（至少 1 模板文件含 pi 触发语法，迁移前为 0），且迁移后自然语言触发词计数 < 基线。
2. 边界（真实 agent 名）：前置条件 = 迁移完成；输入 = 无；操作 = `grep -rcE "agent:'worker'|agent:'reviewer'|agent:'think'|agent:'research'|agent:'exporter'" internal/prompt/templates/ | grep -v ':0' | wc -l`；预期 = ≥1（真实内置/自定义 agent 名被显式引用）。
3. 异常（SENSE 语义 + 无变量泄漏）：前置条件 = 迁移完成；输入 = 无；操作 = `grep -cE '批判门禁|反向回流|judgment.md' internal/prompt/templates/sense_loop.md`（预期 ≥1）+ `go build` + `./bin/rick plan --dry-run | grep -c '{{'`（预期 0）；预期 = 语义命中 ≥1 且无 `{{` 泄漏，`go test ./internal/prompt/... -v` 全绿。
