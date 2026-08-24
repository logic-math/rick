# Rick 项目执行阶段

## 角色定义

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

---

## 先验知识（执行前必读）

## 可用的项目 Loops

- **agent-runtime-bootstrap-loop**："当需要初始化/迁移/重装 rick 的 agent runtime（pi）及其扩展时触发（如 rick tools init-pi、版本升级、runtime 迁移）"
- **protocol-redesign-loop**："当需要重构 AI agent 的多阶段协议(如人机协作流程、任务执行流程),涉及阶段合并/拆分/反向回流/批判层重设计时触发"
- **readme-wiki-sync-loop**："当需要编写或重构 README.md、wiki/（用户面向文档）等描述性资料时触发"
- **tdd-red-green-refactor-loop**："当测试已存在且处于 FAIL 状态，需要通过迭代实现让其变绿时触发（前提：先写测试、当前测试 FAIL、目标是收敛到绿）"


## 可用的项目 Skills

- **check-mechanism**：learning_check / dream_check 命令失败，需要理解失败原因或扩展新检查规则时使用。（doing 门禁已下沉为 rick-gates 确定性脚本，plan_check/doing_check 已删除。）
- **command-registration-verification**：在文档（README、commands.md、学习文档等）中引用项目自身的 CLI 命令、flags、子命令关系时使用
- **dag-task-decomposition**：plan 阶段将复杂需求分解为多个相互依赖的 task 时使用，特别是
- **failure-feedback**：doing 阶段 task 失败重试时，需要理解或调整失败信息如何传递给下一轮 Agent 时使用。
- **fake-binary-script**：当 Go/Python 测试中用**假的可执行脚本**（fake pi、fake node 等）模拟真实二进制时使用
- **global-ref-sync**：修改一个在多个文件中被引用的核心名称/变量时
- **mark-task-success**：doing task 代码已提交（有 commit hash）但 rick-gates 门禁（helper.py）报错
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

**任务 ID**: task11
**任务名称**: 把自然语言 subagent 触发词等价迁移为 pi 显式触发语法

### 任务目标
按 spec（task2）落地 KR3.3：把各命令模板中自然语言 subagent 触发词（「派发 subagent」「SPAWN Sub Agent」「子 Agent」等，共 243 处：root 模板 134 处 + skills/ 109 处，见 RFC §2.1）改写为显式 pi 触发语法（`workflowScript` + `runs.run`/`runs.all` + 真实 agent 名 `agent:'worker'/'reviewer'/'think'/'research'/'exporter'`），并显式化触发权归属（编排权在 parent、普通子 agent 不持 subagent 工具、单写者 one-writer）与 SENSE 特有语义（批判门禁、反向回流、判断记录）。目标：迁移前模板中 `workflowScript`/`runs.run` 零出现（research-report-S-bestpractice.md N3.1），迁移后 >0 且自然语言触发词显著下降。

依据：research-report-S-bestpractice.md BP-1~BP-9 与 D1~D7 差距表；RFC §6 KR3.3。

参考：skill `template_injection_skill`（`{{}}` 陷阱 + dry-run 验证）、`global_ref_sync_skill`（全局替换二次确认）、`verify_go_changes_skill`、`multi_phase_protocol_skill`（批判门禁/反向回流语义的权威定义，确保迁移不丢）。

⚠️ 关键坑：模板经 `PromptBuilder` 用 `{{variable}}` 替换，`extractVariables` 会把模板里任何 `{{...}}` 当变量（job_6 task1 教训）——迁移时**不得在模板中引入非变量的 `{{`**；pi 的 workflowScript 示例用反引号/`${}` 时，若出现在 Go 测试 fixture 里按 bugs.md「Go raw string 反引号截断」处理（raw 段 + 解释型段拼接）。

### 关键结果
1. `internal/prompt/templates/` 顶层 + skills/ 中自然语言触发词改写为显式 pi 语法：`workflowScript` + `runs.run`/`runs.all` + 真实 agent 名；覆盖 sense_loop（think/research/exporter 派发）、plan（六维评审）、doing/easy（Main/Sub Agent）、dream、learning、ctrl
2. 触发权归属显式化：parent 持编排权、单写者、async/context 语义写入相关模板
3. SENSE 特有语义不丢：批判门禁、反向回流（回流上限）、判断记录（judgment.md 只写 human 原话）在 sense_loop.md 迁移后仍完整
4. 验证：`grep -rcE 'workflowScript|runs\.run|runs\.all' internal/prompt/templates/ | grep -v ':0' | wc -l` ≥1；`go build` + `go test ./internal/prompt/... -v` 全绿；`./bin/rick plan --dry-run`/`doing --dry-run`/`human-loop --dry-run` 无 `{{`


### 测试方法
正常路径：前置条件 = task5/9 完成；输入 = 无；操作 = **迁移前先以验收同一正则捕获自然语言触发词基线计数并落盘**（如 `.rick/jobs/job_35/doing/trigger-baseline.txt`）→ 迁移 → `grep -rcE 'workflowScript|runs\.run|runs\.all' internal/prompt/templates/ | grep -v ':0' | wc -l`；预期 = ≥1（至少 1 模板文件含 pi 触发语法，迁移前为 0），且迁移后自然语言触发词计数 < 基线。
边界（真实 agent 名）：前置条件 = 迁移完成；输入 = 无；操作 = `grep -rcE "agent:'worker'|agent:'reviewer'|agent:'think'|agent:'research'|agent:'exporter'" internal/prompt/templates/ | grep -v ':0' | wc -l`；预期 = ≥1（真实内置/自定义 agent 名被显式引用）。
异常（SENSE 语义 + 无变量泄漏）：前置条件 = 迁移完成；输入 = 无；操作 = `grep -cE '批判门禁|反向回流|judgment.md' internal/prompt/templates/sense_loop.md`（预期 ≥1）+ `go build` + `./bin/rick plan --dry-run | grep -c '{{'`（预期 0）；预期 = 语义命中 ≥1 且无 `{{` 泄漏，`go test ./internal/prompt/... -v` 全绿。




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


