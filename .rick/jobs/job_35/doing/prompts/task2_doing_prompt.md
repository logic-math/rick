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

**任务 ID**: task2
**任务名称**: 产出 rick 第一份 spec（四层架构 + 5 模块 + env 四职责契约）

### 任务目标
按 task1 定义的 spec 规范，产出 rick 项目第一份 spec（KR1.2），使 rick 拥有这份 spec（信息内核）。spec 覆盖收敛后的最终架构：
- **四层架构（调用逐级往下）**：
  - 第一层 入口：CLI / TUI / WEB-UI（路由命令、解析参数、交互呈现）
  - 第二层 调度聚合：handler（接受入口参数，编排 env/runtime/builder 完成功能）
  - 第三层 执行：env（pi/dsh 及扩展的检查/安装/配置/维护）+ runtime（pi/dsh 调用封装：参数解析+调用）+ builder（按入口拼接 pi/dsh 提示词产物）
  - 第四层 基础设施：pi（当前 runtime）/ dsh（预留 runtime）/ workspace（路径解析）/ config（~/.rick/config.json 加载）
- 调用关系：上层调下层（逐级往下），下层不回调上层；**例外一**：env ↔ dsh 相互调用（dsh 生态交互关系，非纯单向；不单列 dshRuntime/dshBuilder 节点，链接直接连到具体组件 env 与 dsh）；**例外二**：TUI / WEB-UI 跨层直连 pi/dsh（交互界面直接驱动 runtime，绕过 handler/env/runtime/builder）；**例外三**：组合根（cmd 的 RunE 懒加载实例化 piRuntime/piEnv/pibuilder 注入 handler）是 DIP 组合根模式，越级豁免；**例外四**：workspace/config 是跨层叶子基础设施（路径解析/配置加载），可被任意层直接使用，不参与功能调用链的「逐级往下」约束
- 5 模块职责与边界（含 env 四职责、runtime 职责、handler 职责、builder 三件）；`internal/prompt` = builder 三件中 templates 的承载包（L3）；L3 内部（env/runtime/builder）可复用共享路径工具（AgentDir/RuntimeDir/RuntimeBin/AgentEnv 等），不视为越级回调
- builder 三件契约（templates = go embed 内嵌提示词；pibuilder = pi 统一入口组合子 builder；xxxxbuilder = 扩展位）
- runtime 契约：拉起 pi + 内部校验 session 就绪 + 返回 (sessionID, 行为轨迹)
- 删除清单：executor（调度→pi）、parser（读/校验→pi）、actpath（轨迹→runtime）、logging（死代码）、git（→pi 脚本）、agent 接口（失去消费者）
- 验收标准：功能等价 = 通过所有功能验收；rick 做薄（dag 调度与门禁下沉 pi）；**单一 runtime（pi）为当前实现**——为将来 deepseek harness(dsh) 预留三扩展 seam：builder 的 `RuntimeBuilder`（= xxxxbuilder 转义层）、runtime 的 `Runtime`（`Name`/`Run`）、env 的 `RuntimeEnv`（`Ensure`/`DeployCustomizations`/`CheckReady`）+ config `runtime` 字段（默认 `pi`）；当前 pi 是唯一实现，不写 dsh 代码

依据：`.rick/draft/rfc/rfc-rick-三层架构重构与spec信息内核-2026-08-14.md` §4（目标架构）、§6 O1 KR1.2。此 spec 是 task3~11 重构的「契约」。

### 关键结果
1. 新增 `.rick/domain/rick-spec.md`，含四要素结构（模块边界/职责/接口契约/验收标准）且覆盖四层架构 + 5 模块 + 删除清单
2. env 四职责明确写入：①安装/更新 pi agent ②安装/更新 pi 生态扩展/插件/skill ③安装/更新 rick 自有 hook/skill/agent 定制 ④提供 pi 功能点就绪 check 函数（不含 session）
3. runtime 职责明确写入：拉起 pi + 内部校验 session 就绪 + 采集行为轨迹 + 返回 (sessionID, trace)；handler 职责：编排 env→builder→runtime + 持久化 sessionID 到 job 目录；**注入模型明确写入：方法(system prompt) + 技能(skills) + 实例(user prompt) 三层分离——方法走 `--append-system-prompt`（保留 pi 默认骨架）、技能走 pi skills 机制、实例走 prompt 文件**
4. 下沉策略明确写入：dag 调度 → pi workflowScript 编排（await 顺序）、门禁 → pi hook（rick-gates 扩展）+ 确定性脚本；think/research/exporter → pi agent（env 职责 3 落盘）
5. 扩展 seam 明确写入：`RuntimeBuilder`（builder/xxxxbuilder）、`Runtime`（runtime，`Name`/`Run`）、`RuntimeEnv`（env）+ config `runtime` 字段（默认 `pi`）；「新增 dsh 只新增 dshBuilder/dshRuntime/dshEnv 三个实现并注册，cli/handler/方法层 templates 不改」
6. 四层架构图写入 rick-spec.md（含 mermaid/text 图）：第一层 CLI/TUI/WEB-UI → 第二层 handler → 第三层 env/runtime/builder → 第四层 pi/dsh/workspace/config；标注「逐级往下」+ 例外一「env ↔ dsh 相互调用（不单列 dshRuntime/dshBuilder 节点）」+ 例外二「TUI / WEB-UI 跨层直连 pi/dsh」


### 测试方法
正常路径：前置条件 = `.rick/domain/spec.md`（task1）存在；输入 = rick-spec.md 正文；操作 = 写 `.rick/domain/rick-spec.md` + `git add`；预期 = `test -f .rick/domain/rick-spec.md` 返回 0。
边界（5 模块 + env 四职责 + 四层架构覆盖）：前置条件 = rick-spec.md 已写；输入 = 待写入正文；操作 = `for w in cli handler env builder runtime; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done` + `for w in 安装 生态扩展 定制 就绪; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done` + `for w in 第一层 第二层 第三层 第四层 CLI TUI WEB-UI; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done`；预期 = exit 0（5 模块名 + env 四职责 + 四层架构关键词各自命中）。
异常（与 RFC 一致 + 无变量泄漏 + 扩展 seam）：前置条件 = rick-spec.md 已写；操作 = `for w in dag 门禁 sessionID; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done` + `for w in RuntimeBuilder RuntimeEnv runtime; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done` + `grep -c '{{' .rick/domain/rick-spec.md`（=0）；预期 = exit 0（dag/门禁/sessionID + 三 seam 各自命中）且无 `{{`。




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


