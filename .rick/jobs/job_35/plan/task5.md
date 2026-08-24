# 依赖关系
task2

# 任务名称
重构 builder 三件（templates + pibuilder + xxxxbuilder），注入路径而非内容

# 任务目标
按 spec（task2）落地 KR2.3：将 `internal/prompt` 重构为 builder 三件——templates（go `embed` 内嵌现有模板）+ pibuilder（pi 统一入口，组合 plan/doing/easy/human-loop 子 builder）+ xxxxbuilder（扩展位）。本 task 只做结构重构，**不改模板内容**（触发语言迁移在 task11，单文件内聚在 task10）。

关键方向：**builder 从「注入内容」改为「注入路径」**——rick 不再把 task.md/debug/OKR/SPEC 的内容解析进提示词，而是把 `job_dir`/`plan_dir`/`loops_dir`/`skills_dir`/`domain_dir` 路径注入模板，让 pi 在运行时自己 read。这使 `internal/parser`（读/校验内容）的消费者**大幅减少**，为 task8 删除 parser 铺路（parser 的 executor/prompt 消费点在 task8 与删 executor 同批解耦）。

**三层注入（方法/技能/实例分离，对齐「方法/实现隔离」+ 上下文熵减）**：每个 cmd 的 builder 产出**两份产物**——`method`（命令特定方法：plan 9 步 SOP / doing 角色+doing_loop / SENSE 5 阶段 → 走 system prompt，pi 的 `--append-system-prompt` 注入，免于被 compaction summarize）+ `instance`（job 上下文/路径 → 走 user prompt 文件）；rick 方法论 skills 走 pi skills 机制加载（不塞 system prompt）。

映射：现有 `internal/prompt/templates/`（顶层 10 个 .md = 9 个 loop + test_python.md，skills/ 19 个，go:embed）→ templates；`PromptBuilder`/`PromptManager` + `plan_prompt.go`/`doing_prompt.go`/`easy_prompt.go`/`human_loop_prompt.go`/`ctrl_prompt.go` 生成器 → 子 builder，由新建 pibuilder 统一入口组合；新增 `xxxxbuilder.go` 定义 `RuntimeBuilder` 接口（扩展位，当前无 pi 之外实现）。

参考：domain/go-patterns.md「embed.FS 目录嵌入」「包内函数共享」；skill `verify_go_changes_skill`、`global_ref_sync_skill`、`template_injection_skill`；RFC §4.2「builder 三件」。

# 关键结果
1. 新增 pibuilder 统一入口（`PIBuilder` 类型 + `BuildPlan/BuildDoing/BuildEasy/BuildHumanLoop/BuildCtrl/BuildDream/BuildLearning` 方法；`BuildPlan(requirement string, params map[string]string) (method string, instance string, err error)`——`method`=命令特定方法(走 system prompt)、`instance`=job 上下文(走 user prompt)，空 requirement 返回 error 含 `requirement cannot be empty`，对齐现 `GeneratePlanPrompt`），组合现有子 builder；`PromptBuilder`/`PromptManager`/模板 embed 保留为底层能力
2. 新增 `xxxxbuilder.go`：定义 `RuntimeBuilder` 接口（`Name() string` / `BuildAgents(method []Method) ([]AgentDef, error)` / `BuildPrompt(cmd string, params map[string]string) (string, error)`）——转义层 seam，说明「新增 runtime 只扩展此 builder，cli/handler/env 不改」；pi 实现 = pibuilder，当前无 pi 之外实现（dsh = dshbuilder 将来新增）；`Method`/`AgentDef` 类型在 pibuilder 落地时定义
3. 注入从内容改为路径 + 三层分离：`task_info_section`/`debug_context`/OKR/SPEC 等「内容注入」改为注入对应路径（`task_info_section` → `plan/taskN.md` 路径、`debug_context` → `doing/debug/` 路径，正文由 pi 自行 read）；**路径注入通过 `SetVariable` 变量值实现（如 `SetVariable("task_info_section", <路径>)`），模板文本零改动**；`{{loops_context}}`/`{{skills_context}}` 保留（frontmatter 摘要，非完整内容）；**method 内容进 system prompt（不参与 compaction，含 rick 全局方法固定前缀 + 命令特定方法）、instance 内容进 user prompt（含路径）、执行期按需技能走 pi skills 机制**
4. cmd 层 import `internal/prompt` 的调用方改为 import builder 包（executor 仍引用 prompt 底层能力，task8 删除 executor 后自然消失）；生成行为（变量替换、prompts/ 目录产物）一致
5. `go build` + `go test ./internal/builder/... ./internal/prompt/... -v` 全绿；`git diff --stat internal/prompt/templates/` 为 0

# 测试方法
（本 task 测试用例设计遵循 skill:tdd + skill:testing-anti-patterns：先写失败测试；测试真实生成行为；断言基于真实方法签名——见 testing-conventions.md。）

1. 正常路径：前置条件 = task2 完成；输入 = 无；操作 = `go build -o bin/rick ./cmd/rick && go test ./internal/builder/... ./internal/prompt/... -v`；预期 = build 成功，builder/prompt 测试全绿。
2. 边界（模板零改动 + 注入路径）：前置条件 = 重构完成；输入 = 无；操作 = `git diff --stat internal/prompt/templates/`（预期无 diff）+ `./bin/rick plan --dry-run | grep -cE 'plan/task|doing/debug|/jobs/|/domain'`（预期 ≥1；`task_info_section`/`debug_context` 变量值已变真实路径片段，非 `plan_dir` 等变量名字面量）；预期 = 模板无 diff 且 task/debug 路径注入命中。
3. 异常（builder 缺参数）：前置条件 = 重构完成；输入 = `PIBuilder.BuildPlan("")`（requirement 为空字符串，其余参数为空）；操作 = 调用检查 error；预期 = 返回 error 含 `requirement cannot be empty`。
