# skill:super-cost-analysis（token 成本监控：计量、可解释、可优化）

> 服务对象：**super 中控会话（human 复盘）/ adjust agent（预算决策）/ 调度器运维**
> 加载方式：调用方在提示词中声明本文件路径 + 触发条件；执行到该步骤时用 `read` 读取（不内联）
> 定位（human 裁决 2026-10-26）：**先把「钱花在哪」变成可解释的计量**，优化是后续迭代——本 skill 不擅自改预算，只给证据与处方

## 触发场景

以下任一情形加载本 skill：

1. **run 收尾**：一次进化 run 结束后复盘（super-loop Step 7 轨迹分析的成本维度）
2. **adjust 决策前**：判断「这个格还值不值得继续烧预算」——单格累计 token 已远超其它格、或某格连续多轮高投入无产出
3. **迭代成本异常**：单迭代 tokens > 2× 近期中位数（事件里 `iteration_ended.tokens` 直接可判）
4. **human 要求成本复盘**：「这轮花了多少？花在哪？哪里浪费？」

信号词：「成本」「token 花了多少」「值得吗」「复盘」「太贵了」

## 预期效果

产出一份**成本分布报告**（结构见下），核心回答三个问题：

- **花了多少**：按 role × 迭代 × cell 的 tokens / cost_usd / wall 聚合；单迭代与全 run 合计
- **花在哪**：各 role 占比、上下文膨胀曲线（同一 agent 的 input_tokens 随回合增长）、工具调用密度、cache 命中率
- **哪里浪费 + 怎么省**：命中浪费模式清单的项逐条列出，附最小改动处方（限长/限次/分工），并给「改前 vs 改后」的可验证预期

报告基准（env_9 新 loop 实测，供对照）：单迭代 ≈ **2.1M token**（builder ≈ 0.69M + coding ≈ 1.47M），单迭代墙钟 12–20 分钟。显著高于该基线的迭代/角色值得追问。

## 核心内容

### 1. 数据源（三处，按可信度排序）

| 数据源 | 位置 | 字段 | 用途 |
|---|---|---|---|
| **事件流（权威）** | `<ws>/events.jsonl` | `agent_ended.usage`：`input_tokens` / `output_tokens` / `cache_read` / `cache_write` / `total_tokens` / `cost_usd`；`agent_ended.wall_ms` / `tool_calls` / `truncated` / `model`；`iteration_ended.tokens` / `cost_usd` / **`by_role`**（每角色 {tokens, wall_ms, tool_calls}） | 全部聚合口径的唯一真相源 |
| **pi session（细节）** | `<agent_dir>/sessions/<cwd-slug>/*.jsonl`；`<agent_dir>` 由 `super.json` 的 `pi_path`（`<agent_dir>/runtime/node_modules/.bin/pi`）上推，或环境变量 `PI_CODING_AGENT_DIR` | 每条 `message_end(message.role==assistant).message.usage` + 回合时间戳 | 单轮上下文膨胀曲线、单回合异常（某回合 input 陡增=工具返回超长/重复读大文件） |
| **实验台账** | `<ws>/ledger.jsonl` | `agents`（每角色 tokens/wall_ms）、`verdict` / `perf_delta` | 成本 × 产出 的对照（「花了 X token 换到 Y 改善」） |

工具：`python3 super/tools/cost_report.py --workdir <ws> [--json]` 已把上述聚合做成 markdown 表（含 session 明细自动探测，定位失败自动降级为仅事件流）。**优先跑工具，再按需读原始 session。**

**事件并行写与 `event_fsync`（迭代10）**：事件流现以 `experiment_id`（`<cell>#iter<N>`）标记每次实验、支持并发槽并行写（O_APPEND 单行原子 + 写锁）；还原单实验用 `read_experiment(experiment_id)` 或 `grep '"experiment_id":"<cell>#iter<N>"'`。`super.json config.event_fsync`（默认 `true`）控制每条事件是否 fsync——关掉可降 I/O 开销（高频事件/本机盘），代价是断电时尾部事件可能丢；崩溃恢复依赖事件流时保持 `true`。

### 2. 分析维度

- **按 role**：builder / coding / debug / adjust 的 tokens 占比与单次中位——判断「谁是大头」
- **按迭代**：单迭代 tokens 的分布（中位/尾部）与 Top-5 昂贵迭代——异常迭代逐个看它的实验记录
- **按 cell**：同一格累计 token 与产出（admit/replace 次数）——识别「高投入零产出」的格
- **上下文膨胀**：同一 agent 的 `input_tokens` 序列——线性增长正常；**陡增** = 把大文件反复读进上下文（prompt 侧限长信号）
- **工具调用密度**：`tools_calls / 回合数` 与 `tokens / tool_call`——高的说明在反复试探/验证（如重复 benchmark）
- **cache 命中率**：`cache_read / (input_tokens + cache_read)`——pi 的 `input` 不含缓存部分，分母必须含 cache_read（否则出现 >100% 的荒谬值）；低（<30%）说明前缀不稳定（prompt 抖动/工具输出插入位置频繁变化）
- **成本 × 产出**：结合 ledger 的 `verdict`——同样 token 下 admit 率低的角色/格是优化对象

### 3. 浪费模式清单（命中即记入报告）

1. **重复 benchmark 交叉验证**：coding 在同一候选上多轮 `gate_check` / 多次 `benchmark.py`（session 里连续相同命令）——实测 env_9 的 coding 曾用多轮计时对比 np.dot vs np.matmul 协议
2. **超长上下文**：单 agent 的 `input_tokens` 峰值 > 200k（或超过该角色中位的 3×）——调研报告/相邻格代码全文灌入
3. **无效检索**：`designs/*.json` 的 `papers` 非空但 `plan_steps`/`experiment_plan` 未引用其 `insight`——检索白花
4. **debug 烧在瞬时波动上**：debug 结论是「复测正常/瞬时异常」（`recheck_verdict=keep_original` 或 summary 写「稳定在带内」）却消耗 ≥5 回合
5. **截断后重跑**：`agent_ended.truncated=true` 且同角色在相邻迭代重复同类工作——回合预算不足导致的重复投入。**debug 已默认不设回合上限**（`debug_max_turns=0`）+ 一次性根因协议；builder/coding 仍有回合预算，以「截断率 <20% 且 tokens 中位稳定」为目标调。

### 4. 优化处方（最小改动优先，改前给出可验证预期）

| 处方 | 触发条件 | 做法 | 验证方式 |
|---|---|---|---|
| **限长** | 超长上下文命中 | builder 的相邻格只注入 description + 实验记录摘要（不注入代码全文）；调研报告限 2000 字 | 下次 run 的 `by_role.builder.tokens` 应下降 ≥30% 而 admit 率不降 |
| **限次** | 重复 benchmark 命中 | coding prompt 加纪律「同一候选的 gate_check ≤2 次；benchmark 复验只在参数改动后」 | coding tokens 下降；gate 通过率不变 |
| **分工** | 某角色 token 占比 >50% 且产出低 | `agent_models` 给该角色配轻档；贵档只保留在关键格（champion 格） | 同级质量下降 <5%，成本下降 ≥30% |
| **预算闸门** | 单格高投入零产出 | 用 adjust 的 `soft_weight` 降权该格（0=功能性灭绝）；不是删格 | 后续迭代不再采样该格，token 释放给其它格 |
| **回合预算** | 截断重跑命中 | 调 `*_max_turns`（builder/coding/debug/adjust），以「截断率 <20% 且 tokens 中位稳定」为目标 | 截断率与单迭代 tokens 双指标 |
| **一次性根因（debug）** | debug 多轮试错/重复派发/`truncated=true` | **debug 不设回合上限**（`debug_max_turns=0`）+ 一次性根因协议（复现≥2读数 → 三假设带可证伪判据 → 一次跑完三条验证 → 单次修复 → 复验出结论）——截断→重跑是 debug 浪费主因（env_9 实测：20 次派发 12 次撞上限，问题已定位却无法落结论） | 同格 debug 派发次数下降（目标 ≤1）；debug 的 `truncated=true` 归零；单迭代 tokens 不升（步数↓抵消回合↑） |
| **上下文摘要化** | builder 超长上下文命中 | 相邻格只注入 description + elite perf + 实验记录 digest（400 字）+ code_path，**不内联代码全文**（需更细时 agent 用 `read` 自取） | `by_role.builder.tokens` 下降且 admit 率不降 |

**纪律**：本 skill 只产出分析与处方；改配置/改 prompt 的动作分别走 `env/prompts` 进化流程（PROMPTS_LOG 记录）与 `super.json`/config（可回溯），不直接改运行中的调度器。
