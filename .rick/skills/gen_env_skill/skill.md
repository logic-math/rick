# skill:gen-env（super 进化 env 生成器）

## 触发场景

super 会话（super-loop）中出现以下情形时加载本 skill：

- **env 缺失**：`rick super start` 前目标工作区还没有 `env/`，需要现场生成一个可运行的进化环境
- **env 升级**：既有 env 需要演进——门禁加强（新增 exploit 样本/收紧容差）、评分方向调整、特征轴增改、任务域换版本、**提示词进化（跑完一轮后用轨迹证据迭代四角色模板，见第 7 节）**
- 信号词：「建一个 env」「这个域的进化环境」「门禁怎么设计」「加特征轴」「收紧正确性容差」「分析上轮 adjust 的决策轨迹」「进化 builder/coding 的 prompt」

matmul 类任务不必从零访谈：`super/env_templates/matmul/` 模板已是完整实例，本 skill 的流程用于**定制**它或从零建设其它任务域。

## 预期效果

产出一个**可运行的 env**——通过验收自检（见核心内容第 6 节）的完整环境，super 调度器可立即在其上跑进化循环：

- 目录协议落地（相对 super 工作区 `~/.rick/super/<name>/`）：

```
env/
├── env.json                 # 任务定义：任务名 / perf_desc（评分方向）/ perf_type（单位）/ 容差声明
├── feature_space.json       # 初始特征空间：2-4 轴（name/type/取值域），轴间正交
├── code/                    # 参考实现（正确性锚点）+ naive 起点（进化起点）
├── skills/                  # agent 自检脚本（快速反馈，非终验）
├── data/                    # 框架终验脚本：test_correctness.py + bench.py（stdout 单行 JSON 协议）
├── domain_knowledge.md      # 领域知识 + 「## 性能分析 SOP」「## 性能调优 SOP」两节
├── requirements.txt         # 非标准库依赖声明（如 numpy）
├── .venv/                   # 可选：依赖隔离（python3 -m venv；门禁优先用 .venv/bin/python）
└── prompts/                # 可选：四角色提示词模板的 env 级定制/进化产物（见第 8 节，延迟具体化）
gates/                       # 工作区级（非 env 内）：
├── exploit_registry.json    # 已知作弊/坏解样本（门先证红的种子）
└── golden_examples/         # 金样例（参考实现即首个金样例）
```

- 关键纪律：**门禁的容差/评分函数/阈值属于 env 设计范畴（human 决策），本 skill 只引导设计，不代答、不写死**；rick 框架（superopt gates.py）只提供级联门结构（环境健康→正确性→性能 + 软门禁多计时中位数）并读 env 的定义执行。

## 核心内容

访谈式流程——**逐环节引导 human 决策**，每环节产出物落盘后再进下一环节：

### 1. 任务定义

- 确定任务名、输入输出（例：两个 512×512 float32 矩阵 → 乘积矩阵）。
- **参考实现**：写 `env/code/reference.py`——用该生态最权威直白的写法（如 numpy.matmul）。它是正确性锚点（门先证红的金样例之首）。
- **naive 起点**：写 `env/code/naive_start.py`——无优化的朴实写法（如纯 Python 三重循环）。进化从这里起步，「慢」是特性不是缺陷。

### 2. 评分方向

- `perf_desc`：**low**（越小越好，如 runtime_ms）或 **high**（越大越好，如 score/吞吐）。
- `perf_type`：单位（runtime_ms / score / …）。
- 写入 `env.json`；同格替换的比较方向由框架按此字段感知——这是 human 的领域判断，不是框架默认。

### 3. 门禁系统设计引导（核心环节）

引导 human 逐项设计门禁（每项给出该域的具体选项，不代答）：

- **正确性门**（`env/data/test_correctness.py`）：输出 stdout 单行 JSON `{"passed": bool, "max_error": float}`。引导决策：
  - 校验规模（小矩阵快速校验，例 64×64）与固定 seed（可复现）
  - **容差 ε**：该域可接受的最大数值误差（1e-3？1e-5？按任务精度要求定）
  - 候选异常/超时如何判（框架把非 JSON/超时按该 stage 失败处理）
- **性能门**（`env/data/bench.py`）：输出 `{"runtime_ms": float, "score": float, "timings": [...]}`。引导决策：
  - bench 规模（例 512×512）、预热次数、计时次数（多计时 K=3 取中位数由框架机制保证，脚本内落 K 次或单次由框架调 K 次均可——**取脚本内 K=3 中位数**）
  - score 语义（相对参考实现的加速比？绝对吞吐？）
  - 超时上限（坏候选不能拖死门禁）
- **环境健康自检**（`env/skills/self_check.py`）：agent 自检脚本（coding agent 快速反馈用）：跑一遍正确性 + 单次 bench，输出 JSON 摘要。**自检仅供参考，入格以框架终验为准**。
- **exploit 注册表**（`<ws>/gates/exploit_registry.json`）：**引导 human 想「这个域里 agent 会怎么作弊」**，构造 3-5 个已知坏解样本（每条 {id, name, pattern, sample_path}）。常见模式参考：
  - 预存结果（返回缓存/硬编码答案）
  - 篡改计时（计时期间偷工/异步返回）
  - 降精度换速度（float64→float16 之类该域不可接受的精度损失）
  - 破坏输入后部分计算（只算可过校验的子集）
  - 样本写成可执行的小脚本放 `gates/` 下，注册表引用路径
- **金样例**（`<ws>/gates/golden_examples/`）：参考实现即首个金样例；可再补 1-2 个风格不同的已知好解。作用：门禁升级时防误杀（历史合法解必须继续通过）。

### 4. 初始特征空间

`env/feature_space.json` 的 axes：**2-4 轴起步**，每轴 {name, type, values/取值域, desc}。
- 引导原则：**轴间正交**——每根轴刻画解的一个独立维度（例：impl_strategy=算法策略轴 / memory_layout=数据排布轴；一个讲「怎么算」、一个讲「数据怎么放」，互不蕴含）。
- 轴取值域是「可枚举的行为描述子」（enum）或「可分档的量」——格 id 由框架从描述子确定性生成。
- 宁少勿滥：先 2 轴跑起来，进化轨迹显示解空间贫瘠再加轴（加轴走 env 升级路径）。

### 5. 初始解

- `env/code/naive_start.py` 即进化起点（第 1 节已落）。
- 框架行为：每迭代把选定父本整拷到 `codes/{uuid}/` 作为工作副本——env 的 code/ 目录保持只读圣洁。

### 6. 验收自检清单（env 交付前逐条过）

```bash
PY=env/.venv/bin/python   # 或系统 python3（无 .venv 时）
# ① 参考实现过正确性门
$PY env/data/test_correctness.py --candidate env/code/reference.py
# → {"passed": true, "max_error": ...}
# ② bench 出分
$PY env/data/bench.py --candidate env/code/reference.py
# → {"runtime_ms": >0, "score": ..., "timings": [3 个]}
# ③ exploit 样本被正确性门拒识（门先证红地基）
$PY env/data/test_correctness.py --candidate <每个 exploit 样本>
# → 全部 {"passed": false, ...}
# ④ naive 起点过正确性门（慢但必须对——它是进化起点）
$PY env/data/test_correctness.py --candidate env/code/naive_start.py
# → {"passed": true, ...}
```

四项全过 → env 合格，回 super-loop Step 2 启动（start 会再跑全量门先证红：exploit 拒识 100% + 金样例零误杀）。
任一不过 → 修对应环节（容差/样本/脚本），不带着病门禁启动进化。

### env 升级路径（触发场景之二）

- 改门禁（容差/评分/exploit 注册表）→ **必须重跑门先证红**（exploit 全拒 + 金样例全过），建议顺带跑双回归（elite 回放 + exploit 回归）；改完下迭代自动生效（调度器每迭代 subprocess 直调 pi，无需重启）。
- 加特征轴 → 新轴只影响后续采样；既有格不删（不删格原则），旧格降权走 adjust 提案（run 内的运行时加格走 adjust 的 expand_cell 提案→裁决链；跨 run 的轴级改版走本升级路径）。
- 升级访谈只需覆盖变更涉及的环节 + 第 6 节清单重过。

### 7. 轨迹分析 → 提示词进化（《自改进的进化搜索算法》外层循环）

搜索的「智能」住在 env 层的可进化资产里（特征空间 + 门禁 + **四角色提示词模板**）。
本环节用上一轮 run 的**执行轨迹证据**迭代这些资产——跑一轮，不仅解在进化，搜索算法本身也在进化。

适用对象：**四角色提示词均适用**（`adjust.md` / `builder.md` / `coding.md` / `debug.md`），
其中 adjust 是首要对象（它是 run 内唯一采样决策者：权重=采样份额、加格=探索、灭绝=淘汰），
builder 是第二优先级（成本大户，计划质量直接决定后续迭代的成功率）。

**渲染机制（改造的对象在哪）**：调度器每迭代按三级查找链解析角色模板——
`<ws>/prompts/<role>.md` → `<ws>/env/prompts/<role>.md`（**env 级覆盖**）→ 仓内默认
`super/prompts/<role>.md`；解析后注入上下文（格元信息/实验计划/失败诊断）落盘为
`codes/{uuid}/prompt_{role}.md` 派 pi 实例。改 env 级模板 = 改变该 env 下每次 agent
调用的行为契约。

**流程（五步，逐环节产出物落盘后再进下一步）**：

1. **取证**：跑 `python3 super/tools/adjust_trace.py --workdir <ws> [--window 5]`
   → 决策-效果对照表（每次 adjust 决策 → 之后 K 迭代受影响格的表现变化，markdown 表输出）。
   证据补充源：`codes/adjust/iter_{N}/`（每次 adjust 的输入 prompt 与输出 proposals 快照）、
   `events.jsonl` 里的 `adjust_decision` / `weight_changed` / `proposal` / `expand_cell` 事件。
2. **诊断坏模式**（对照表 + 快照逐条核对）：
   - **震荡**：同一格反复升降权（A↑B↓ 来回摆）
   - **过激**：单次事件证据就大改权重（如 0.5→0）
   - **迟钝**：连续多迭代的失败证据不动权重
   - **漏杀**：垃圾格（持续差 perf 且无改善信号）长期占采样预算
   - **探索不足**：全程 expand_cell 从未发生（数据饱和后仍不加格）
   - **builder/coding/debug 各自的坏模式**：计划与结果脱节（builder 计划从不被执行）、
     自检通过但框架门拒（coding 自检与门禁口径不一）、debug 修不动同类根因（缺诊断 SOP）
3. **处方：改写 `env/prompts/<role>.md`**——针对诊断出的模式加/改**决策纪律与判断启发**
   （如 adjust：每次权重变动须引用 ≥1 条事件 seq 号 / 同类连续降权须间隔防震荡 /
   何时机值得 expand_cell；如 builder：计划的 plan_steps 必须可被 coding 直接执行）。
   只改出问题的角色的模板——其余角色不动（可归因性：一次迭代一个变量）。
4. **版本管理（三件套，缺一不可）**：
   - **延迟具体化**：默认继承仓内模板（不复制不落盘）；仅当需要定制/进化时才首次落
     `env/prompts/<role>.md`——env 协议保持最小。
   - **旧版存档**：改写前把当前版本拷到 `env/prompts/_history/<role>.{yyyymmdd-HHMMSS}.md`。
   - **PROMPTS_LOG.md**（`env/prompts/` 下）：每行记「时间 | 角色 | 版本要点（改了什么/为什么） |
     证据链接（adjust_trace 输出/事件 seq） | **可移植性检查：无绝对路径**」——进化史可审计、可回滚、可对照。
     可移植性检查判据：`grep -n "/Users/\|/home/\|/workdir/" env/prompts/*.md` 为空才算过。
5. **下轮验证回环**：改完模板 → 回 super-loop 启动下一轮 run → run 完成后重跑第 1 步对照——
   **改善 → 固化（PROMPTS_LOG 追加「已验证」）**；**恶化 → 回滚**（从 `_history/` 恢复上一版并记原因）。
   一次只改一个角色的模板（否则改善/恶化都不可归因）。

**可移植性纪律（硬性）**：`env/prompts/*.md` 里的一切路径必须用渲染注入的占位符——
`{{env_dir}}`、`{{perf_profile_cmd}}`、`{{self_check_cmd}}`（每迭代由调度器渲染填入），
**禁止硬编码绝对路径**。理由（env_8 实测教训）：写死 `<ws>/.rick/super/env_8/...` 的模板
在 env 复制、改名（env_9）、跨机迁移后全部失效，且把该轮进化锁死在单机上。
占位符为空或原样残留时，模板文案应写成「显示占位符原文或为空 = 该能力不可用/未饱和」——
用可读的兜底表述，而不是让 agent 看到 `{{saturation}}` 这种原始记号去猜。

**可用 ctx 字段清单（改模板时优先用它们，别让 agent 花回合去 read 文件——回合即 token）**：

| 角色 | 已注入的上下文键 |
|---|---|
| builder | `env_dir` `perf_profile_cmd` `self_check_cmd` `saturation` `domain_brief` `perf_timeline` `budget_status` `ledger_brief` `gate_defs` `neighbor_elites` `history_failures` `axis_perf` `feature_axes` |
| coding | `gate_defs` `perf_baseline` `workdir_files` `ledger_brief` `saturation` `experiment_plan` `self_check_cmd` `env_dir` `workdir` |
| debug | `gate_failure` `gate_defs` `ledger_brief` `perf_profile_cmd` `saturation` `failure_stage` `stderr_tail` `self_check_result` `attempt` `experiment_plan` |
| adjust | `saturation` `perf_timeline` `budget_status` `domain_brief` `ledger_brief` `own_decisions` `cells_status` `feature_axes` `recent_events` |

设计原则：能进 ctx 的结构化信息就不该让 agent 自己去 read（每读一次约消耗 1 回合上下文）；
模板里引用这些键即可。若发现某角色反复读同一文件，优先考虑把它抽进 ctx（改框架 = 升仓级知识，
走 learning 循环），而不是在模板里硬写路径。

**饱和 / 认输（terminate_recommend）解读**：
- `saturation` 非空 = 框架判定「连续 ≥3 迭代无新纪录 + 全部存活格 visit≥2 + 冠军格近 3 次采样未刷新」。
  此时 adjust 应首选 `expand_cell`（加格 = 探索的唯一入口）；builder 应设计「跳出当前方向」的实验。
- 轨迹分析发现「长时间无新纪录但 adjust 全程未提 expand_cell」→ **先查 saturation 是否真的注入了**
  （prompt 快照里该字段为空/占位符残留 = 注入链或模板有问题，先修这个，别急着改 adjust 的判断纪律）。
- 收到 `terminate_recommend` 提案（`proposals/` 里 `needs_human: true`，不进裁决自动执行）→
  与 human 讨论二选一：**收手归档**（北极星终值 + 结论写收尾报告）或 **换轴**（走 gen_env 的轴级改版升级路径）。
- 反向检查：`saturation` 长期为空但 perf_timeline 明显停滞 → 检查框架的饱和条件阈值是否过严（仓级问题，走 learning）。

**实验台账（`ledger.jsonl`）解读**：工作区根下的跨迭代实验档案——每迭代一条：
`iteration_id / cell_id / idea / parents / gate{passed,failure_stage,perf_value} / verdict(admit|replace|reject|no_candidate) /
perf_delta / reject_reason / 各 agent 成本`。轨迹分析时**优先读它**：比 events.jsonl 密度高一个数量级
（一趟读完一轮实验史）；坏模式（反复 reject 的同类 idea、某格长期 no_candidate、某类失败反复出现）
在台账上一眼可见。agents 侧也已注入 `ledger_brief`（最近若干条摘要）供决策时避坑。

**模型分配引导（prompt 之外的另一个性能杠杆）**：计划/执行分离下两档模型分工是共识——
`builder` 用贵档（实验计划质量优先），`coding` 可配轻档（执行性任务容错高）。
配置走 `super.json` 的 `config.agent_models`（示例：
`{"builder": "catpaw-proxy/glm-5.3", "coding": "catpaw-proxy/deepseek-v4-flash"}`）——
与模板进化独立调优、独立归因；模型分配的调整同样记入 PROMPTS_LOG。

**两级进化路径（接通点）**：本环节进化的是 **env 级资产**（该 run 的搜索 DNA）；
super-loop / gen_env skill 自身等**仓级方法知识**的进化走 rick 既有 learning/dream 循环
（轨迹分析发现系统性坏模式 → learning job 沉淀 → 更新 loop/skill）——两层各司其职、互不越界。
