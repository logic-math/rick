# Rick 项目执行阶段

## 角色定义

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

---

## 先验知识（执行前必读）

## 可用的项目 Loops

- **agent-runtime-bootstrap-loop**："当需要初始化/迁移/重装 rick 的 agent runtime（pi）及其扩展时触发（如 rick tools init-pi、版本升级、runtime 迁移）"
- **do-check-mark-success-loop**："当 doing_check 报错 task status != success 或 commit_hash 缺失时触发"
- **protocol-redesign-loop**："当需要重构 AI agent 的多阶段协议(如人机协作流程、任务执行流程),涉及阶段合并/拆分/反向回流/批判层重设计时触发"
- **readme-wiki-sync-loop**："当需要编写或重构 README.md、wiki/（用户面向文档）等描述性资料时触发"
- **tdd-red-green-refactor-loop**："当测试已存在且处于 FAIL 状态，需要通过迭代实现让其变绿时触发（前提：先写测试、当前测试 FAIL、目标是收敛到绿）"


## 可用的项目 Skills

- **check-mechanism**：plan/doing/learning_check 命令失败，需要理解失败原因或扩展新检查规则时使用。
- **command-registration-verification**：在文档（README、commands.md、学习文档等）中引用项目自身的 CLI 命令、flags、子命令关系时使用
- **dag-task-decomposition**：plan 阶段将复杂需求分解为多个相互依赖的 task 时使用，特别是
- **failure-feedback**：doing 阶段 task 失败重试时，需要理解或调整失败信息如何传递给下一轮 Agent 时使用。
- **fake-binary-script**：当 Go/Python 测试中用**假的可执行脚本**（fake pi、fake node 等）模拟真实二进制时使用
- **global-ref-sync**：修改一个在多个文件中被引用的核心名称/变量时
- **mark-task-success**：doing task 代码已提交（有 commit hash）但 doing_check 报错
- ****：当通过 `pi install npm:<pkg>` 或 `pi install <local-path>` 安装 pi 扩展（如 subagent、web-access、主题包）后使用。
- ****：当 rick 迁移或更换底层 agent runtime（如 claude code → pi，或 pi 版本大升级），且改动涉及 rick 调用 agent 的 CLI flags / 事件解析 / prompt 落盘路径时使用。
- **pi-theme-verification**：当需要验证/定制 pi 主题时使用
- **subprocess-env-isolation**：当集成测试中通过 subprocess 调用 rick CLI，测试本地通过但行为与预期不符时
- **template-injection**：需要在 `rick plan` 或 `rick easy` 会话中嵌入新的结构化行为时
- **test-script-practices**：在 plan 或 doing 阶段编写/调试任务测试脚本（`.rick/jobs/job_N/doing/tests/taskN.py`）时使用，特别是
- **verify-go-changes**：修改了 Go 源文件后，需要验证编译通过、单元测试和集成测试通过时使用。
- **zero-retry-task-design**：plan 阶段分解需求为多个 task.md 时使用，目标是让每个 task 在 doing 阶段一次性完成，无需重试。


---

## Job 上下文

暂无问题记录

---

## 任务信息

**任务 ID**: task8
**任务名称**: 做薄 cutover：下沉 doing 调度与门禁到 pi，并删除全部冗余 Go 包

### 任务目标
按 spec（task2）落地 KR2.5（rick 做薄，同时覆盖 KR3.2 的 dag/门禁下沉部分）：这是「做薄」的原子切换点，一次性完成调度下沉与冗余包删除（此前 task4/6 刻意保留 executor/agent/actpath/parser/git/logging，避免中间态编译断裂）。

1. **dag 调度 → pi workflowScript（顺序确定性）+ rick 生成期过滤（跳过已完成）**：doing 提示词产出 `workflowScript` + `runs.run` 编排（按 task 依赖拓扑，被依赖 task 先执行，`await` 强制顺序；编排权在 parent、单写者）。**「跳过已完成」= rick 生成期过滤**：rick 在拼提示词时读 `doing/tasks.json`（确定性 Go），过滤掉 `status=success` 的 task，对剩余 pending task 算剩余拓扑，只把 pending task 写进 workflowScript——重试时已完成 task 天然不在编排里（workflowScript 沙箱无文件系统、读不了 tasks.json，这一步只能在生成期做）。**触发假设（基线前提，非风险）**：workflowScript 由 main agent 调 `subagent` 触发——这是「模型遵循明确、无歧义指令」的基线假设，是 AI coding 成立的前提（同 RFC §2.4 核心假设）。若模型连如此明确的触发指令都无法遵循，则模型不足以支撑 AI coding，无需在架构层纠结「是否 100% 触发」。一旦触发，内部顺序 100% 确定（await 强制 + 未 await 报错）；agent_settled 门禁 + rick 薄重试循环仅作为兜底安全网（检测模型偶发失败/部分完成、支持断点续跑），非正确性前提
2. **门禁 → rick 侧确定性脚本（runtime.Run 在 pi 会话结束后调用）+ 可选 pi hook 记录**：tasks.json 可解析 / 无 zombie running / success 有 commit_hash 的门禁语义，由 `runtime.Run` 在解析到 `agent_settled`（pi 会话结束）后**直接调用 `python3 .rick/skills/rick-gates/helper.py <doing_dir>` 校验**（确定性脚本，exit 非 0 = 门禁失败）；不再由 Go 检查——注意 pi 的 extension hook 是「通知」语义而非「拦截」，且 Python 脚本不会被 pi 作为 extension 加载，故门禁判定+重试收敛在 rick 侧，`agent_settled` 只作为「会话结束」的确定性信号；rick-gates hook 扩展（TS 包装）仅作可选的记录/通知
3. **runtime 签名切换**：runtime 的 `Execute` 改为 `Run` 返回 `(sessionID, trace, err)`（返回 sessionID 即成功；未解析出 sessionID 或未收 `agent_settled` 返回 error）；**删除 `internal/agent` 接口**（失去 executor 消费者）与 `internal/actpath`（轨迹由 runtime 的 trace 承载）
4. **删除冗余 Go 包**：`internal/executor`（dag/topological/runner/executor/retry/tasks_json/doing_check/debug_dir）、`internal/parser`、`internal/git`（commit 下沉 pi 脚本）、`internal/logging`（死代码）、`internal/cmd/tools_doing_check.go`、`internal/cmd/tools_plan_check.go`，以及引用它们的测试文件（`tools_test.go`/`doing_test.go`/`learning_test.go` 中相关断言）
5. **调用点迁移**：`handler.Doing` 从 `executor.ExecuteJob` 改为「builder 产 workflowScript 编排 + runtime.Run」；`dream.go` 的 `discoverCompletedJobs`（依赖 `executor.LoadTasksJSON`+`TasksJSON.GetAllTasks`+`TaskState.Status`）与 `learning.go` 的 `collectExecutionData`（依赖 `executor.LoadDebugContext`/`LoadTasksJSON`/`TasksJSON` + `parser.ExtractBugFrontmatter`）迁到 `workspace`（极薄读取器：定义 `TasksJSON`/`TaskState`/`LoadTasksJSON`/`LoadDebugContext` 类型 + `ExtractBugFrontmatter` 极简 frontmatter 提取）或 pi 侧脚本；commit 由 pi 在每个 task 成功后立即执行（复用 mark_task_success_skill 模式）；**prompt（builder 底层）对 parser 的解耦**：`GenerateDoingPromptFile`/`ContextManager`/`loadDebugContextLocal`/`formatTaskInfoSection`/`formatDebugContext` 移除对 `parser.Task`/`parser.DebugInfo`/`parser.ContextInfo` 的 import，改为接收字符串/路径；**同时删除 `context_helpers.go` 的 `formatOKRContent`/`formatSPECContent` 与 `ContextManager` 的 OKR/SPEC 解析方法（生产代码零调用的死代码）及其测试**；**runtime 测试清理**：删除/改写 `internal/runtime/executor_e2e_test.go` 对 `internal/actpath`/`internal/agent` 的依赖及所有 `Execute(...) (agent.AgentSession, error)` 调用点/`agent.ToolCall`/`AgentSession` 断言
6. **act-path 语义由 runtime trace 承接 + doing 新执行流语义等价**：删除 `internal/actpath` 的同时改写模板中所有 `act-path.md` 引用（`learning.md`/`learning_loop.md`/`gen-skill.md`/`gen-loop.md`/`dream.md`/`ctrl.md`）为 runtime trace（raw_session_coding.log + Trace）；清理 `easy_prompt.go`/`doing_prompt.go` 中 `check_command`/`check_step_header` 的死代码设置；`executor_realpi_smoke_test.go` 同步改 `Run` 签名（或声明弃用）；doing 新执行流的 workflowScript 拓扑、per-task commit+commit_hash 时序、tasks.json 写入者与状态机、断点续跑（--resume）、失败汇总输出，逐条对齐原 executor（跳过已完成/running→success/retry/partial/failed）
7. **doing 状态机协议 + trace 产物契约**：明确 parent 单写者、per-task「running→success→commit→回传 commit_hash→parent 写 tasks.json」时序、失败/partial 语义、tasks.json 字段级 schema 与门禁 helper.py 对齐；runtime.Run 落盘 trace 文件（如 `doing/tasks/<id>/trace.md`）供 learning/dream/ctrl 模板引用（替代 act-path.md），6 个模板的 act-path 引用改为该文件；`parser.ExtractBugFrontmatter` 在 workspace（或 prompt）落地极薄实现（删 parser 前 grep 全量 `parser.` 消费点）
8. **skill/loop 维护（删除/下沉后的知识库收敛）**：更新 `.rick/skills/check_mechanism_skill/skill.md`（删除 doing_check/plan_check 段，保留 learning_check 段）、`.rick/skills/mark_task_success_skill/skill.md`（删除 doing_check 段）、`.rick/skills/failure_feedback_skill/skill.md`（删除 internal/executor/retry.go 引用或淘汰）；`.rick/loops/do-check-mark-success-loop.md` 迁 `loops/deprecated/`（或标注失效）；同步更新 `loops/README.md`、`skills/README.md` 索引

依据：RFC §3.2「dag 调度与门禁不再由 rick 维护，利用 pi 能力直接实现」；research-report-S-bestpractice.md BP-1/BP-6/BP-8；pi docs/extensions.md（hook 生命周期）。

参考：loop `tdd-red-green-refactor-loop`；skill `verify_go_changes_skill`、`check_mechanism_skill`（本 job 删除 doing_check/plan_check 后需更新，保留 learning_check 段）、`mark_task_success_skill`（commit 模式复用，需更新删除 doing_check 段）、`dag_task_decomposition_skill`（循环依赖/拓扑排序参考）、`global_ref_sync_skill`、`pi_runtime_verification_skill`、`pi_extension_install_verification_skill`；`failure_feedback_skill`（仅概念参考，本 job 删除 internal/executor/retry.go 后需更新或淘汰）；`do-check-mark-success-loop` 本 job 删除 doing_check 后失效、不再引用。

### 关键结果
1. doing 提示词（builder/doing_prompt.go 产出）含 pi 编排段：rick 先读 `doing/tasks.json` 过滤 `status=success` 的已完成 task，对剩余 pending task 算剩余拓扑，生成只含 pending 的 `workflowScript` + `runs.run` 编排 + 门禁脚本调用（重试天然跳过已完成）；**tasks.json 初始生成者 = builder**（首轮 tasks.json 不存在时由 builder 扫描 `plan/task*.md`，用极简字符串提取「# 依赖关系」章节生成初稿：task 列表 + 依赖 + status=pending，不依赖已删除的 parser 包）；**剩余拓扑的极简算法（Kahn/DFS）内联在 builder**（非已删除的 executor/dag.go）
2. 门禁 = `runtime.Run` 在解析到 `agent_settled`（pi 会话结束）后**直接调用 `python3 .rick/skills/rick-gates/helper.py <doing_dir>`**（rick 侧确定性脚本，exit 非 0 = 门禁失败 → handler 重试）；校验 tasks.json/debug 格式/commit_hash（可解析/无 zombie/success 有 commit_hash 三项语义不丢）。**commit 时序**：每个 task 成功后立即由 pi 执行 commit + 写 commit_hash，再进入下一 task；全部完成后 agent_settled 门禁才校验「success 有 commit_hash」（避免门禁跑到未 commit 的 task 误报）；门禁脚本错误串沿用原语义（`missing commit_hash`、zombie `running`），exit 非 0 = 未过；rick-gates hook 扩展（TS 包装）仅作可选记录/通知，不作门禁拦截
3. runtime 签名切换：`Run(...) (sessionID, trace, err)`，返回 sessionID 即成功；删除 `internal/agent` 接口 + `internal/actpath`
4. 删除 `internal/executor`、`internal/parser`、`internal/git`、`internal/logging` + `tools_doing_check.go`/`tools_plan_check.go` + 相关测试文件；**同步删除/改写 `tools.go` 中对 `NewPlanCheckCmd`/`NewDoingCheckCmd`/`NewEasyCheckCmd` 的注册**（否则 cmd 编译 `undefined`）；`handler.Doing`/`dream`/`learning` 调用点迁移完毕；**同步改写 `templates/plan.md` 第 8 步、`templates/doing.md` 第二步、`skills/doing_loop.md`**（删除 `check_step_header`/`check_command`/`{{check_command}}`/`plan_check`/`doing_check`/`easy_check` 的注入与引用，改为 pi hook 门禁 / workflowScript 编排语义）；**同步改写 `internal/prompt/{doing_prompt_test,plan_prompt_test,context_test,integration_test,integration_rfc001_test}.go` 对 `internal/parser` 的 import**（fixture 改为字符串/路径，或随 task5 迁到 builder 测试）；**`learning_check`/`dream_check` 存活**（`tools_learning_check.go`/`tools_dream_check.go`/`tools_loops_skills_check.go` 只依赖 workspace/标准库，frontmatter 解析是 cmd 包内自实现、不依赖 parser/executor，仅 `--auto-fix` 的 `piagent.FindBinary`/`CallCLI` 改 `runtime`，`tools.go` 注册保留）
5. `rick doing job_N` 端到端可用（pi 按依赖执行 pending task + 门禁 hook 自动校验）；rick 侧薄重试循环（兜底安全网，非正确性前提）：`runtime.Run` 返回门禁失败（workflow 未触发/任务未完成/模型偶发失败）→ 重新 launch（上限 max_retries），重试时重新生成「只含剩余 pending」的编排；`go build` + `go test ./internal/builder/... ./internal/prompt/... ./internal/cmd/... ./internal/runtime/... ./internal/env/... ./internal/handler/... ./internal/workspace/... -timeout 60s` 全绿


### 测试方法
正常路径：前置条件 = task6/7 完成 + 一个含 3 task（task2 依赖 task1）的 job；输入 = `doing job_N --dry-run`；操作 = `./bin/rick doing job_N --dry-run | grep -cE 'workflowScript|runs\.run'`；预期 = ≥1（doing 提示词含 pi 编排语法）。
边界（dag 拓扑 + 跳过已完成 + 门禁脚本 + 冗余包已删）：前置条件 = 同上 job，且 tasks.json 中 task1 已标记 `success`；输入 = `doing job_N --dry-run`；操作 = `test -f .rick/skills/rick-gates/helper.py` + `for d in executor parser actpath logging git agent; do test ! -d internal/$d || exit 1; done` + `grep -oE "runs\.run\('task[0-9]+'" <doing dry-run 输出>` 提取编排的 task 序号序列，断言「task1 已 success → 序列不含 task1；否则 task1 在 task2 之前」；预期 = 门禁脚本已部署、6 冗余包已删、依赖顺序正确、已完成 task 被跳过。
异常（门禁语义不丢 + runtime 签名 + 重试收敛）：前置条件 = 某 task status=success 但 commit_hash 空；输入 = `python3 .rick/skills/rick-gates/helper.py <doing_dir>`；操作 = 跑该脚本；预期 = 报 `missing commit_hash` 退出非 0。另 runtime `Run` 在 fake JSONL 缺 `agent_settled` 时返回 error（未就绪）；门禁检测到「workflow 未触发/未完成」→ handler 重试（重新生成只含剩余 pending 的编排，上限 max_retries）。




---

**你需要一步步执行以下操作，不可跳过任何步骤。**



## 第一步：执行 Doing Loop

# Doing Loop

> ⚠️ 以下是默认 loop 的执行步骤，也是 gen-loop 需要参考的 skill 模板！！

---

## Step 0：Domain 搜索 + Loop 匹配

**必须依次完成以下两项，再进入 Step 1：**

### 0.1 搜索 Domain（强制）

根据澄清的需求，读取 `/workdir/sunquan20/AI_CODING/rick/.rick/domain` 下的相关文件，获取足够的事实信息（环境配置、已知问题、接口约束、构建命令等），建立解决问题的基本视角。

- 由 AI 自行判断读取哪些文件，但**必须完成搜索动作**后再继续
- 遇到任何问题（编译报错 / 测试失败 / 行为异常），**必须优先搜索 `/workdir/sunquan20/AI_CODING/rick/.rick/domain/bugs.md` 和 `/workdir/sunquan20/AI_CODING/rick/.rick/domain/`**，再做其他尝试

### 0.2 匹配 Loop

在 Domain 搜索完毕后，读取 `loops_context`，按 trigger 字段匹配当前任务/需求：

- **有匹配** → 读取对应 Loop 文件，按其定义步骤执行（不再执行以下 Step 1–5）
- **无匹配** → 按以下 Step 1–5 执行默认 Loop

---

## Step 1：Main Agent 确认全局目标

确认以下内容全部清晰后才继续：

- task.md 中 `# 任务目标` 和 `# 关键结果` 已理解
- 成功标准已明确：测试脚本全通过 + check pass + 所有 Key Results 达成

---

## Step 2：Main Agent 读取上下文（压缩策略）

从 `doing/debug/` 目录读取已有信息，按以下方式压缩后传递给 Sub Agent：

- **bug\*.md** → 从每个文件的 frontmatter `summary` 字段提取摘要，避免重复踩坑
- **跨轮核心事实** → 任务目标 + Key Results 达成状态 + debug/ 摘要 + 当前迭代编号 N

---

## Step 3：启动 Sub Agent 执行工作流

**每轮迭代由 Main Agent 启动一个独立 Sub Agent，携带 Step 2 的上下文，执行完整工作流后返回产出摘要。**

```
[Main Agent]
   │
   ├─ SPAWN Sub Agent（携带：任务目标 + debug/摘要 + 迭代编号 N）
   │     │
   │     │  Sub Agent 执行：
   │     │  [ANALYZE] → [RED] → [GREEN] → [REFACTOR] → [COMMIT]
   │     │                 ↑        │
   │     │                 └──[DEBUG]┘
   │     │
   │     └─ Sub Agent 完成，输出产出摘要
   │
   └─ Main Agent 执行 Step 4 产出评估
```

### Sub Agent：ANALYZE（理解需求）
1. 声明：`"I will use skill:sense."`，按 S→E→N 分析（Symptoms / Evidence / Next）
2. 读取 debug/ 摘要，避免重复踩坑

### Sub Agent：RED（先写失败测试）
1. 声明：`"I will use skill:tdd for implementation."`
2. 针对 `# 测试方法` 中每个场景编写测试
3. 运行测试，**必须确认 FAIL**（证明测试有效，进入 GREEN 的前提）

### Sub Agent：GREEN（最小实现）
1. 编写让测试通过的最小实现代码（不超出 task scope）
2. 通过 → REFACTOR；失败 → DEBUG

### Sub Agent：DEBUG（遇红强制触发）

触发条件（任意一条）：测试 FAIL / 编译报错 / 行为与预期不符

1. **优先搜索 `/workdir/sunquan20/AI_CODING/rick/.rick/domain/bugs.md` 和 `/workdir/sunquan20/AI_CODING/rick/.rick/domain/`**，查看是否有精确解决方案
   - 有匹配 → 直接应用，记录引用来源
   - 无匹配 → 继续下方流程
2. 声明：`"I will use skill:debug-skill."`，加载 skill 文件：`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/prompts/skill_debug_skill.md`
3. 在 `doing/debug/` 下创建 `bug{N}-{描述}.md`，按 Phase 1-6 执行
4. Phase 4 上限 3 次，达上限后输出当前状态并升级人工协作
5. 修复后回到 GREEN

### Sub Agent：REFACTOR（代码改善）
1. 测试全绿后改善代码质量（命名、结构、去重）
2. 运行全量测试确认无回归；回归失败 → DEBUG

### Sub Agent：COMMIT（收尾提交）
1. `git add` + `git commit`（commit message 含 task ID）
2. 运行 check 命令（使用 prompt 上下文中的 rick_bin_path 和 job_id）：
   - doing 阶段：`<rick_bin_path> tools doing_check <job_id>`
   - easy 阶段：`<rick_bin_path> tools easy_check <job_id>`
3. check 失败 → 修复后重新运行，循环直到 pass
4. **Sub Agent 完成**：输出本轮产出摘要（完成了哪些 KR、遗留了哪些问题），通知 Main Agent 执行 Step 4

---

## Step 4：Main Agent 产出评估

Sub Agent 完成后，Main Agent 逐项检查：

| 检查项 | 判断方法 |
|--------|----------|
| check pass | 读取 doing_check / easy_check 输出，确认 ✅ |
| 测试全通过 | 确认测试脚本无 FAIL 输出 |
| Key Results 达成 | 逐条比对 task.md `# 关键结果` |

- **全部通过** → 进入 Step 5
- **存在失败** → 将失败原因附加到上下文，返回 Step 3 启动下一轮迭代

---

## Step 5：Main Agent 确认停止标准

**成功退出**：check pass + 测试全通过 + 所有 Key Results 达成

**优雅退出**（任意一条触发）：
- 迭代次数达上限（默认 **3 轮**）
- 连续 2 轮产出相同错误（判断无法自动收敛）
- 人类明确要求停止

**退出时**：Main Agent 输出 Loop 执行摘要（完成了哪些 KR、遗留了哪些问题），等待人类决策。





---

## 第二步：格式检查

`/workdir/sunquan20/AI_CODING/rick/bin/rick tools doing_check job_35`

check pass 后才算完成。


