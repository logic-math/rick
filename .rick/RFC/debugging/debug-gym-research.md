# debug-gym 调研报告 — 事实整理

## 项目概述

- **名称**：debug-gym: A Text-Based Environment for Interactive Debugging
- **来源**：Microsoft Research，作者团队包括 Xingdi Yuan、Morgane M Moss、Alessandro Sordoni、Marc-Alexandre Côté 等
- **论文**：arXiv:2503.21557（2025年3月）
- **版本**：当前调研基于 v1.0.0 之后的版本（CHANGELOG 记录最新更新至 2025-10-02）
- **定位**：为 Python 程序的交互式调试设计的文本环境，用于评测 AI Agent 的调试能力
- **核心目标**：提供一个标准化的调试环境，让 LLM-based Agent 通过调试工具（如 pdb）探索代码、收集信息，最终提出 patch 修复 bug

---

## 核心架构

### 模块结构

```
debug_gym/
├── gym/
│   ├── envs/          # 环境类（RepoEnv、LocalEnv、FreeEnv、SWEBench 等）
│   ├── terminals/     # 执行后端（LocalTerminal、DockerTerminal、KubernetesTerminal）
│   └── tools/         # 工具（bash、view、edit、eval、pdb、grep、listdir、submit、MCP）
├── agents/            # Agent 实现（BaseAgent、FroggyAgent、SolutionAgent、SimpleAgent）
└── llms/              # LLM 后端（OpenAI、AzureOpenAI、Anthropic、HuggingFace、Human）
```

### 核心概念

**RepoEnv（环境）**：基于 Gymnasium 范式的交互式环境。
- `env.reset()` 开始一个 episode，返回初始信息
- `env.step(action)` 执行一个工具调用，返回新的观测
- 维护状态：`score`、`resolved`、`terminated`、`last_eval`、`current_breakpoints_state`

**EnvInfo（环境信息结构体）**：每个 step 返回的信息包含：
- `step_observation`：当前 step 的主观测（工具输出）
- `all_observations`：本 step 触发的所有工具的观测
- `eval_observation`：最近一次 eval 的输出
- `current_breakpoints`：当前断点状态
- `action_reasoning`：LLM 的推理过程
- `action_content`：LLM 的回复内容
- `action_tool_call`：具体的工具调用
- `instructions`：任务描述
- `score` / `max_score`：得分
- `terminated` / `resolved`：是否终止/是否解决

**HistoryTracker（历史追踪器）**：
- 存储 `system_message`、`problem_message`、`env_initial_observation`
- 存储每个 step 的 `env_observations`（EnvInfo 列表）和 `llm_responses`（LLMResponse 列表）
- 支持 `json(game_step)` 序列化单个 step 的完整信息

### 数据流

```
LLM → LLMResponse (tool_call + reasoning + content)
    → env.step(action_tool_call, action_content, action_reasoning)
    → EnvironmentTool.use(environment, **kwargs)
    → Observation (source, observation_text)
    → EnvInfo
    → HistoryTracker.step(env_observation, llm_response)
    → build_prompt() → 下一轮 LLM 调用
```

---

## 调试流程设计

### Agent 的基本循环（BaseAgent.run）

```python
# 1. 环境 reset，获取初始观测（包括自动执行 eval）
info = env.reset()

# 2. 主循环，直到 terminated 或 max_steps
while not should_stop:
    agent_response = self.step(info)         # LLM 生成工具调用
    info = self.execute_action(agent_response)  # 执行工具，获取新观测
    should_stop, reason = self.should_stop(step, info)
```

### FroggyAgent 的系统提示结构

每次 LLM 调用时，系统提示包含：
1. **Instructions**：任务描述（例如"调试程序使其通过隐藏测试"）
2. **Current breakpoints**：当前所有断点（可选，由 `show_current_breakpoints` 控制）
3. **Evaluation output of current code**：最近一次 eval 的输出（trim 到 80% 上下文窗口）
4. **Shortcut features**：环境特性说明（如自动恢复断点、auto_list 功能）

### 历史消息构建（build_prompt）

每轮提示 = system_prompt + instance_prompt + history

history 按时序排列：
```
[LLM_response_1] [env_observation_1] [LLM_response_2] [env_observation_2] ...
```

历史截断策略（二选一）：
- `max_history_steps_cutoff`：保留最近 N 步
- `max_history_token_cutoff`：从最新步开始反向累计，不超过 token 限制

### 没有内置假设-验证循环

**调试流程完全依赖 LLM 自主决策**，框架本身不强制任何调试方法论。系统提示中对调试过程的指导（来自 `swebench_debug.yaml` 和 `mini_nightmare.yaml`）：

> "While the code may seem familiar to you from your training, you should not assume you know the code. Instead, you must investigate the code carefully to understand the potential bugs. Once you have gained enough information, propose a patch to fix the bugs."

> "Do not repeat your previous action, especially if it returned tool calling errors or it resulted in information that you already know."

框架没有硬编码的假设-验证循环。LLM 可以自由选择：先用 pdb 设断点→运行→查变量，或者先用 bash 搜索代码→直接 edit。

---

## 信息管理机制

### 上下文信息保留方式

**1. HistoryTracker（完整历史）**

每个 step 都被完整保存在内存中：
- `env_observations`：每个 step 的 EnvInfo 深拷贝
- `llm_responses`：每个 step 的 LLMResponse 深拷贝

不存在"压缩"或"总结"机制。历史信息以原始形式保留，截断策略是**直接丢弃最旧的步骤**（按 token 数或步数）。

**2. Jinja 模板的 `trim_message` 过滤器**

用于在构建 prompt 时截断过长消息：
```jinja
{{ info.eval_observation.observation | trim_message(max_length_percentage=0.1, where="end") }}
```
支持从 `start`、`end` 或 `middle` 截断。FroggyAgent 中 eval 输出截断方式为 `where="middle"`，保留头尾。

**3. 持久化断点（Persistent Breakpoints）**

PDB Tool 维护 `current_breakpoints_state` 字典（存于 RepoEnv）：
```python
{
    "file_path|||line_number": "b file_path:line_number",
    ...
}
```
每次 edit 成功后，PDB 会**自动重启并恢复所有断点**（`on_edit_success` 事件触发）。断点行号会根据编辑位置自动调整（`breakpoint_modify` 方法）。

**4. Trajectory 持久化**

每个 problem 的完整执行轨迹保存为 `trajectory.json`，包含：
- 每个 step 的 `step_id`、`reasoning`、`content`、`action`、`obs`
- 完整的 `prompt_response_pairs`（每轮 LLM 调用的 prompt 和 response）
- `system_message` 和 `problem_message`（只记录一次）

**5. 跨 episode 信息**：没有专门机制。每次 `env.reset()` 会清空所有工具历史和断点。

---

## 终止条件设计

### Agent 级别（BaseAgent.should_stop）

```python
def should_stop(self, step: int, info: EnvInfo):
    if info.terminated:        # 环境触发终止（submit 工具被调用）
        return True, "terminated"
    if step >= self.args.max_steps:  # 步数上限
        return True, "max_steps reached"
    return False, None
```

**终止有两种路径**：
1. Agent 主动调用 `submit` 工具 → `environment.terminated = True`
2. 达到 `max_steps`（默认 100，配置文件中通常设为 50）

### 环境级别（env.resolved）

`resolved` 由 `calculate_resolved` 决定：`score == max_score`

`score` 由最近一次 eval 输出决定（`calculate_score`）。默认实现：`eval_output.success`（bool）。

`submit` 工具默认会**重新运行 eval** 再终止（`eval_on_submit=True`）。

### 重试机制（scripts/run.py）

针对**基础设施故障**（不是调试失败）的重试：
- 仅对 `UnrecoverableTerminalError`（容器/Pod 崩溃）和 `AgentTimeoutException` 触发
- 默认最多重试 3 次（`--max-retries` 参数）
- 重试时**回放上一次的 trajectory**（`replay_actions`），从失败点之后继续

重试机制**不用于**调试失败的情况。如果 Agent 在 max_steps 内未解决问题，直接标记为 `unresolved`，不重试。

### 得分记录

`highscore` 跟踪整个 episode 中出现过的最高得分（`max(highscore, info.score)`）。

状态分类：`pending`、`running`、`resolved`、`unresolved`、`skip-resolved`、`skip-unresolved`、`error`

---

## 对四个追问的回答

### 1. 调试能力定义

debug-gym 对「调试能力」的操作化定义：**在有限步数内，通过交互式工具探索代码并提交正确 patch，使隐藏测试集从 failing 变为 passing**。

评估维度：`score`（通过测试数）和 `resolved`（是否完全通过）。

与「缺陷→感染→失效」状态机的对应：
- 框架提供了工具让 Agent 观察「失效」（eval 输出的 test failure）
- PDB 工具让 Agent 追踪「感染」（变量值、执行路径）
- edit 工具让 Agent 修复「缺陷」（代码修改）

框架**没有明确使用「缺陷→感染→失效」的概念框架**，但工具设计与该模型的调试流程一致。

### 2. 上下文信息保留

debug-gym 的信息保留机制：
- **完整保留**：HistoryTracker 保存每个 step 的完整 EnvInfo 和 LLMResponse
- **截断但不压缩**：超出 token 限制时，直接丢弃最旧的步骤，没有摘要或压缩
- **eval 结果始终可见**：`eval_observation` 独立于 history，每轮都会放入 system prompt
- **断点状态持久化**：`current_breakpoints_state` 在整个 episode 内持续存在
- **没有「关键信息」识别机制**：没有区分重要/不重要信息的逻辑

### 3. 科学调试法内嵌

debug-gym **没有内嵌假设-验证循环**。调试流程完全由 LLM 自主决定。

系统提示只要求：
- 调查代码，不依赖训练记忆
- 不重复已知失败的操作
- 收集足够信息后再提出 patch

没有强制的「先假设→再验证」步骤，没有「记录尝试→防止盲目重试」机制，没有「失败原因分析」的结构化要求。

### 4. review agent / 终止条件

debug-gym **没有独立的 review agent**。

终止由以下机制决定：
- Agent 主动调用 `submit` 工具（自我判断任务完成）
- 达到 `max_steps` 上限
- 基础设施故障（容器崩溃，触发重试或最终标记 error）

`solution_agent` 是唯一具有某种「验证」逻辑的 agent，但它是 oracle agent（知道正确答案），用于环境健全性检查，不是真实调试流程的一部分。

---

## 与当前 rick 方案的关联

### debug-gym 有而 rick 没有的设计

**1. 工具即信息通道**

debug-gym 将每个工具调用的输出作为独立的 `Observation` 返回，并与 LLM 的推理分开记录（`action_reasoning` vs `action_content` vs `step_observation`）。rick 目前没有区分「Agent 的推理」和「Agent 的行动」。

**2. eval 结果的特殊地位**

debug-gym 在每轮系统提示中单独传入最近一次 eval 输出（`eval_observation`），而不是让它淹没在 history 中。这保证了「当前测试状态」始终在 LLM 的视野内。rick 的 debug.md 机制有类似意图，但是基于文件写入/读取，不是 prompt 构建时的结构化注入。

**3. 断点持久化与自动调整**

`persistent_breakpoints` 和 `breakpoint_modify` 实现了编辑后断点行号自动调整。这是调试状态跨 step 保持一致性的具体机制。

**4. Trajectory 完整序列化**

每个 step 的 prompt、response、reasoning、action、observation 全部被序列化为 `trajectory.json`，支持事后分析和失败重试时的回放。rick 的 debug.md 只记录失败信息，不记录完整的 prompt-response 序列。

**5. 得分的细粒度表示**

`score` 和 `max_score` 是数值型（通过测试数/总测试数），而不是二元成功/失败。允许追踪「部分正确」状态。rick 目前的状态只有 done/failed。

**6. 工具失败不终止 episode**

工具调用失败（包括 LLM 幻觉出的无效参数）会返回错误 `Observation` 而不是崩溃。Agent 可以根据错误信息调整下一步行动。rick 的失败处理直接触发重试计数。

### rick 方案有而 debug-gym 没有的设计

**1. 科学调试方法论的显式指导**：rick 方案中的「先猜3次，失败后走假设-验证循环」。debug-gym 没有。

**2. 结构化的「改动了X + 因为Y + 结果Z」记录格式**：debug-gym 记录原始 observation，不强制结构化的调试日志。

**3. review agent（禁止 mock）**：debug-gym 没有独立的代码评审角色。

**4. 跨 Agent 信息传递机制**：debug-gym 是单 Agent 循环，没有多 Agent 协作场景。

---

## 信息来源

- `/Users/sunquan/ai_coding/CODING/debug-gym/README.md`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/agents/base_agent.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/agents/froggy_agent.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/agents/history_tracker.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/agents/solution_agent.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/agents/utils.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/gym/envs/env.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/gym/entities.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/gym/tools/tool.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/gym/tools/pdb.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/gym/tools/edit.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/gym/tools/eval.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/gym/tools/bash.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/gym/tools/submit.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/gym/envs/local.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/debug_gym/experiment.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/scripts/run.py`
- `/Users/sunquan/ai_coding/CODING/debug-gym/configs/swebench_debug.yaml`
- `/Users/sunquan/ai_coding/CODING/debug-gym/configs/mini_nightmare.yaml`
- `/Users/sunquan/ai_coding/CODING/debug-gym/configs/templates/human_friendly_system_prompt.jinja`
- `/Users/sunquan/ai_coding/CODING/debug-gym/data/mini_nightmare/mini_nightmare.md`
- `/Users/sunquan/ai_coding/CODING/debug-gym/CHANGELOG.md`

---

## 待用户补充

1. **论文具体实验数据**：arXiv:2503.21557 中各 benchmark 的具体数字（pass rate、PDB vs no-PDB 对比）无法从代码中获取，需要阅读论文。

2. **`simple_agent.py` 内容**：文件存在但本次未读取，可能包含更简单的 baseline 设计。

3. **MCP Proxy Tool 的具体用途**：MCP_README.md 描述了扩展工具的机制，但具体场景（是否有 review agent 类工具）未调研。

4. **分析脚本的具体指标**：`analysis/` 目录下有生成论文图表的脚本，其中可能包含对「调试轨迹质量」的定量定义，未深入读取。

5. **SWEBench 环境中 `instructions` 的具体格式**：`swe_bench.py` 中任务描述的完整结构（包含 issue 描述、测试信息等），未读取。
