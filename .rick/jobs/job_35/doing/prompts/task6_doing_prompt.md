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

**任务 ID**: task6
**任务名称**: 落地 handler 调度聚合层并让 cli 变薄（plan/doing/easy）

### 任务目标
按 spec（task2）落地 KR2.1 第一部分：新建 `internal/handler` 包作为调度聚合层，编排 env→builder→runtime 并持久化 sessionID。将 plan/doing/easy 三大命令的编排逻辑从 `internal/cmd` 迁移到 handler，cli 变薄为「Cobra 命令 + flag 解析 + 调 handler」。

handler 编排模式（目标态，task8 才完全落地）：`env.Ensure()` → `builder.Build()`（产 method+instance 两份）→ 调用 runtime → 持久化 sessionID 到 job 目录（`session_id`）。本 task 中：doing 仍经 executor 执行（不改调度），plan/easy 仍走 `runtime.CallCLI`（交互 TUI，method 经 `--append-system-prompt` 注入）；`runtime.Run` 结构化签名在 task8 切换 doing 时才启用。handler 函数签名以参数接收 flag 值（如 `Options{Verbose, DryRun, JobID}`），**不得 import `internal/cmd` 的 `GetVerbose`/`GetDryRun`/`GetJobID`**（跨包循环依赖），cmd 在调用点透传。

## 逐命令改动明细（调研结论）

### plan（干净命令，不依赖 executor/parser/git/actpath/logging，迁移无编译断裂）
- cli 保留 `NewPlanCmd`（Use/Short/Args + RunE 解析 `GetJobID`/`GetDryRun`/args）；迁出 `executePlanWorkflow`→`handler.Plan`、`reEnterPlanWorkflow`→`handler.ReEnterPlan`、`runPlanDryRun`→`handler.PlanDryRun`
- **删除死代码 `generateJobID()`**（`time.Now().Unix()` 生成，但主流程用 `workspace.NextJobID()`，从未被调用）；`promptForRequirement` 与 easy.go 共享（cmd 包内）→ 迁 handler 或在 cmd 保留共用（否则 easy 侧编译断裂）
- flags：无自有 flag；全局 `--dry-run`/`--job`/`-v`；`--job` 触发重入分支（先校验 plan 目录存在）

### doing（executor 最大消费者，删包同批在 task8）
- cli 保留 `NewDoingCmd`（`--job`/`--easy`/`--ctx` flags + RunE 路由：`--easy`→handler.Easy、`--dry-run`→handler.DoingDryRun、否则 handler.Doing）；迁出 `executeDoingWorkflow`→`handler.Doing`、`runDoingDryRun`→`handler.DoingDryRun`、`loadTasksFromPlan`/`sortTaskFiles`/`extractTaskNumber`（task8 删，pi 直读）、`printExecutionSummary`、`commitDoingResults`/`ensureGitUserConfigured`/`ensureGitInitialized`（task8 下沉 pi 脚本）
- 本 task 只搬编排到 handler（doing 仍经 executor 执行，不改调度），task8 才删包 + 改 `runtime.Run` + workflowScript 编排

### easy（easy.go 无 parser import，仅 easy_prompt.go 的 loadDebugContextLocal 依赖 parser.ExtractBugFrontmatter）
- cli 保留 `NewEasyCmd`（`--ctx`/`-r/--requirement`/`--resume` flags）；迁出 `runEasyMode`→`handler.Easy`、`resumeEasyMode`→`handler.ResumeEasy`、`startEasySession`→`handler.StartEasySession`、`runEasyDryRun`→`handler.EasyDryRun`、`validateCtxInheritance`（--ctx 防覆盖）、`saveSessionID`/`loadSessionID`、`writeEasyTasksJSON`（map 内联，无 executor 依赖，保留「easy_session success」供 dream 发现）、`generateUUID`
- `loadDebugContextLocal` 的 `parser.ExtractBugFrontmatter` 依赖：三层注入后改「注入 `doing/debug/` 路径、pi 自行 read」消除 parser 依赖（task8 一并解耦）

参考：domain/architecture.md「模块划分」；skill `command_registration_verification_skill`、`verify_go_changes_skill`、`global_ref_sync_skill`。

### 关键结果
1. 新建 `internal/handler/`，迁移上述 plan/doing/easy 编排函数为 handler 方法；handler 编排 = env.Ensure → builder.Build（产 method+instance）→ 调用 runtime → 持久化 sessionID
2. `internal/cmd/{plan,doing,easy}.go` 变薄：仅保留 Cobra 命令 + flag 解析 + 调 handler；全部 flag 行为不变；删除死代码 `generateJobID`
3. 命令注册不变（`grep -n AddCommand internal/cmd/root.go` 8 命令）
4. doing 仍经 executor 执行（本 task 不改调度，仅搬编排）；logging/git/parser/actpath/executor 均保留；随函数迁移同步改写 `internal/cmd/{plan,doing,easy}_test.go` 中引用已迁函数的断言（executePlanWorkflow/executeDoingWorkflow/loadTasksFromPlan 等）；另改写 `tools_test.go`（`runDoingDryRun` 引用）与 `dryrun_integration_test.go`（`runPlanDryRun` 引用）
5. `go build` + `go test ./internal/cmd/... ./internal/handler/... -timeout 60s -v` 全绿


### 测试方法
正常路径：前置条件 = task3/4/5 完成；输入 = 无；操作 = `go build -o bin/rick ./cmd/rick && go test ./internal/cmd/... ./internal/handler/... -timeout 60s -v`；预期 = build 成功，测试全绿。
边界（dry-run 变量替换 + 空 requirement）：前置条件 = build 成功；输入 = `plan --dry-run`、`easy --dry-run -r ""`；操作 = `./bin/rick plan --dry-run | grep -c '{{'`（预期 0）+ `./bin/rick easy --dry-run -r ""`（预期报 `requirement cannot be empty`）；预期 = 无 `{{` 且空 requirement 报错。
异常（无效 job + 重入不存在 plan + --ctx 冲突）：前置条件 = build 成功；输入 = `doing job_nonexistent`、`plan --job job_nonexistent`、`easy --ctx <已有 loops 的路径>`；操作 = 依次运行；预期 = 分别报 `job directory not found`、`plan directory does not exist`、`local context already exists`，均 exit 非 0。




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


