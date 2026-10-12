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

**任务 ID**: task9
**任务名称**: 注册 think/research/exporter 为 pi 自定义 agent（经 env 职责 3 落盘）

### 任务目标
按 spec（task2）落地 KR3.2 的 agent 注册部分：将 think/research/exporter 注册为 pi 自定义 agent——frontmatter 落盘到 `~/.rick/pi/agent/agents/`（user 级），system prompt = 对应源码 skill 的 wiki 内容，tools 声明对齐 pi 工具名。此任务产出的是 env 职责 3（`DeployRickCustomizations`）中「agent」定制物的具体内容。

生成 3 个 agent 文件（YAML frontmatter + system prompt）：
- `think.md`：`name: think`、`tools: read, grep, find, ls`、system prompt = `templates/skills/think.md`（推理识别 + 4 维打分 + 3 启发性问题）
- `research.md`：`name: research`、`tools: read, grep, find, ls, bash, web_search, fetch_content`、system prompt = `templates/skills/research.md`（尽调树 + 信源加权）
- `exporter.md`：`name: exporter`、`tools: read, write, bash`、system prompt = `templates/skills/exporter.md`（RFC 输出，大纲 + 内容两阶段）

由 env 职责 3 幂等写入，frontmatter 含 name/description/tools/defaultContext；不再生成普通 markdown 提示词文件（当前 think/research/exporter 是散落 prompt 文件、无 frontmatter、不在 pi 发现目录——见 research-report-S-reasons-agent.md B4）。

依据：research-report-S-reasons-agent.md B1（frontmatter 定义）/B2（user 级 `~/.rick/pi/agent/agents/**/*.md`）/B3（`runs.run(agent:'think')` 触发）；pi docs/agents.md。

参考：loop `agent-runtime-bootstrap-loop`；skill `pi_extension_install_verification_skill`、`pi_runtime_verification_skill`、`subprocess_env_isolation_skill`、`verify_go_changes_skill`。

### 关键结果
1. `internal/env`（职责 3）产出 3 个 agent 的 frontmatter + system prompt 内容，幂等写到 `runtime.AgentDir()/agents/{think,research,exporter}.md`（尊重 RICK_PI_AGENT_DIR/HOME 隔离）
2. 3 文件 frontmatter 含 `name`/`description`/`tools`/`defaultContext`；system prompt 为对应 skill wiki 正文（非空、非仅 frontmatter）
3. `~/.rick/pi/agent/agents/` 下 3 个 agent 可被 pi 发现——**先按 pi docs/agents.md 确认自定义 agent 的真实发现入口（`pi list` 现用于 extensions、未必列 agent），验收断言用真实入口**（think/research/exporter 真实 agent 名可被 `{action:"list"}` 或等价机制列出）
4. 幂等 + 覆盖语义：rick 自有的占位/旧版内容**必须按内容比对覆盖**（task3 未写 agent 文件，若存在旧散落产物则覆盖）；覆盖判定 = 文件 frontmatter 含 rick 固定标记（如 `rick-managed: true`）才覆盖，无标记文件跳过；重复运行不重复插入


### 测试方法
正常路径：前置条件 = task3/5 完成 + `RICK_PI_AGENT_DIR` 指向 temp；输入 = 运行 `DeployRickCustomizations`（或 init-pi）；操作 = 运行 + `test -f "$RICK_PI_AGENT_DIR/agents/think.md"` + 用 pi docs/agents.md 确认的真实 agent 发现入口（`{action:"list"}` 或等价）列出 think/research/exporter；预期 = 3 文件存在，`head -5 think.md` 含 `name: think`，真实发现入口列出 3 个 agent 名。
边界（幂等 + 覆盖语义）：前置条件 = 已注册一次；输入 = 再次运行；操作 = 再次运行 + `sha256sum` 对比前后；预期 = 文件内容不变（幂等，不重复插入 frontmatter）；另预置一个无 `rick-managed: true` 标记的同名文件 → 运行后内容不被覆盖（仅覆盖有 rick 标记的文件）。
异常（system prompt 非空）：前置条件 = 3 文件已写；输入 = 无；操作 = `awk '/^---$/{n++} n>=2 && /[^[:space:]]/{print}' "$RICK_PI_AGENT_DIR/agents/think.md" | grep -c .`；预期 = ≥1（frontmatter 闭合后存在非空正文，wiki 内容注入成功）。




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


