# skill:gen-env（super 进化 env 生成器 · v2 四步构建流程）

> job_56/task2 重写：四步构建流程（设计树追问 → 四 skill 构建 → 演练 → 验证金字塔）
> v2 关键变化：①env 四部分结构（domain_knowledge / skills / **gates 验证器目录** / code）
> ②gate_check.py 移入 `env/gates/`（builder/coding/debug 禁读——防针对验证器优化；
> adjust 可读写维护）③多维优化目标（objective：约束+加权指标→融合分）④验证金字塔
> 分层参数（L1-M0/L{n}/E2E）⑤subagent 超参数配置位（agent_hyperparams）。

## 前置输入（先读再访谈——避免重复劳动）

在开始任何访谈之前，先读工作区里已有的材料：

| 路径（若存在） | 读来做什么 |
|---|---|
| `<ws>/requirement.md` | human 的原始需求（要优化什么） |
| `<ws>/grilling/design-tree.md` | 设计树追问产出：目标/约束/现状候选表述、研究结论、判断节点裁决、实测数据 |
| `<ws>/research-*.md`、`<ws>/research/` 下简报 | 领域调研：权威做法、论文线索、环境事实、已知死族 |
| `<ws>/decisions/`、`<ws>/designs/`（升级路径时） | 上轮 adjust 决策与 builder 设计（判断哪些口径需要改） |

**纪律**：已有素材 → **复述确认**，不重新追问；缺什么补什么。重复访谈是对 human 时间的浪费。

## env 四部分结构（v2 基本结构定义——所有生成的 env 必须遵循）

```
<env>/
├── domain_knowledge.md      # ① 领域专家知识（核心文件——所有实验 agent 的基本准则与文档入口：
│                            #   任务背景 / 多维优化目标定义 / 验证金字塔矩阵 / 资源分时复用规则 /
│                            #   skill 务必调用清单 / 验证器保密纪律 / 特征格实验指引）
├── env.json                 # 任务定义 + v2 键：
│                            #   objective（多维目标：constraints[{name,op,value}] + metrics[{name,weight}]
│                            #             + fusion）→ 融合分（gate_check 加权融合，越高越好）
│                            #   agent_hyperparams（adjust 流水线每步 {model, thinking} 配置位）
│                            #   env_slots（同环境并发实验数声明——分时复用规则由 env 设计者在
│                            #             domain_knowledge.md 描述）
│                            #   verify_levels（验证金字塔定义：E2E/L{n}/L1 各级 cmd+耗时+用途）
│                            #   既有键（task/perf_desc[融合分方向，v2 缺省 "high"]/perf_type/
│                            #          tolerance/bench/correctness/gates/debug_policy）保留
├── feature_space.json       # 初始特征空间：2-4 轴（name/type/取值域），轴间正交
├── skills/                  # ② agent 入口 skill（agent 可读可执行）：
│   ├── profile.py           #   现状：事实获取（热点/环境事实/父本结构）→ 事实 JSON
│   ├── verify.py            #   约束门**薄入口** → 执行引用 gates/verify_impl.py（同一实现）
│   ├── benchmark.py         #   基准门**薄入口** → 执行引用 gates/benchmark_impl.py；支持 --level 分层验证
│   ├── research.py          #   检索指引（方法/信源/关注点，按角色输出；不执行网络请求）
│   └── （可扩展其他 skill——注入 domain_knowledge.md 的调用清单并强调务必调用）
├── gates/                   # ③ 验证器目录（**builder/coding/debug 禁读内容**[superopt-json-guard
│                            #   hook 拦截读取]；adjust 可读写；可执行不可读内容）：
│   ├── gate_check.py        #   v2 验证器入口（**框架必然调用**）：verify → benchmark K 次中位
│   │                        #   → 约束校验 → 加权融合 → 软门禁 → 融合分协议 JSON
│   ├── verify_impl.py       #   验证实现（skills/verify.py 执行引用）
│   ├── benchmark_impl.py    #   基准实现（skills/benchmark.py 执行引用）
│   ├── exploit_registry.json#   已知作弊/坏解样本（门先证红种子）
│   └── golden_examples/     #   金样例（参考实现副本，零误杀证明）
├── code/                    # ④ 基线代码（进化 target）：reference.py（正确性锚点）+ naive 起点
├── data/                    # legacy 兼容薄转发（旧 env 的 test_correctness.py + bench.py）
├── requirements.txt         # 非标准库依赖声明（如 numpy）
└── .venv/                   # 可选：依赖隔离（门禁优先用 .venv/bin/python）
```

**与 v1 的关键差异**：gate_check.py 与验证实现从 `skills/` 移入 `gates/`（验证器隔离）；
skills/ 的 verify/benchmark 变为薄入口（执行引用，不复制实现）；gate_check v2 协议输出
**多维融合分**（见步骤②gate_check 协议）。

## 脚本统一契约（硬性，缺一条 agent 与框架就会漂移）

1. **`--candidate <path>` 唯一必需参数**；其余从 `env.json` 读（不写死规模/容差/超时）。
2. **必须支持 `--env-dir <dir>`**（缺省=脚本上两级）。
3. **多规模读 `correctness.sizes`（数组）**；`--size` 仅覆盖。
4. **stdout 单行 JSON**；失败也输出 JSON（`{"passed": false, "error": ...}`）。
5. **gate_check.py（gates/）v2 协议**（框架侧 superopt/gates.py `_verify_gate_check_v2` 解析）：
   `{"passed": bool, "score": <融合分>, "metrics": {指标名: 值}, "constraints": {约束名: {value, op, target, satisfied}}, "diagnostics": "...", "failure_stage": ...|null, "tolerance_band"?: bool}`
   **score ≥ 0 ⟺ passed**（强一致性，框架侧校验）；score < 0 = 拒绝：
   -1 环境错误 / -2 正确性失败 / -3 约束违反 / -4 软门禁劣化。融合分**越高越好**
   （low 方向指标由 gate_check 派生归一——如 speedup = ref_runtime/candidate_runtime）。
6. **`--baseline <score>`**（软门禁：融合分基线比较，band 读 env.json tolerance.soft_gate.relative_band）。
7. **verify/benchmark 薄入口透传 `--level` 分层验证参数**（实现侧按级别裁剪规模/次数——见步骤④）。
8. **research.py 契约独立**：`--role <adjust|builder|coding|debug>` 必需，stdout 单行 JSON
   （检索指引，不执行网络请求——检索动作由 agent 的 web_search 完成）。
9. **门先证红**：统一实现是框架 `CascadeGate.preverify_red()`——不要手工逐个试 exploit。

## 触发场景

- **env 缺失**：`rick super start` 前目标工作区还没有 `env/`——走四步构建流程从零生成
- **env 升级**：门禁加强/评分调整/特征轴增改/任务域换版/debug_policy 调整/提示词进化（第 7 节）
- 信号词：「建一个 env」「这个域的进化环境」「门禁怎么设计」「加特征轴」「收紧正确性容差」「分析上轮 adjust 的决策轨迹」

matmul 类任务不必从零访谈：`super/env_templates/env_v2/` 是 v2 结构完整示范实例，本 skill 用于定制或从零建设。

## 预期效果

产出一个**可运行的 v2 env**——通过步骤③演练验收（五件可跑 + gate_check 协议闭环 + 门先证红 ok）。

## 核心内容：四步构建流程（不可跳步）

```
步骤① 设计树追问（下钻得到完整 env 设计方案后生成）
   0. 任务定义（reference + naive 起点）→ 0.5 领域实测（跑 reference 拿事实）
   → 1. 三要素定性追问 → 1.5 检索三问 → 2. 门禁定量决策
   → 2.5 debug_policy → 2.6 多维 objective 定义（约束+加权指标）→ 3. 特征空间 → 4. 初始解
步骤② 四 skill 构建方法（必须从 human 主动获得 profile/benchmark/verify/research 的构建方法）
   → 四 skill 组合出 gates/gate_check.py（代码复用：调用 verify/benchmark 实现，不重复实现）
步骤③ 演练（逐 skill 真实执行 + gate_check.py 真实运行——基础设施无问题才交付）
步骤④ 分层验证机制（验证金字塔：分钟级快速验证 → 端到端验证）
—— 跑完一轮后 ——
第 7 节 轨迹分析 → 提示词进化（外层循环，保留）
```

### 步骤① 设计树追问（下钻追问 → 完整 env 设计方案 → 生成）

**先落两个可运行的文件**（reference.py 正确性锚点 + naive_start.py 进化起点），然后按下列环节
**逐层下钻追问 human**（每环节落盘后再进下一环节；已有素材先复述确认）：

1. **任务定义**：任务名 / 输入输出形态 / 正确性锚点（reference）/ naive 起点。
2. **领域实测**：跑 reference 拿基线数据（耗时量级/波动带/死族线索）——访谈有数据依据。
3. **三要素定性追问**（只到「能写成脚本」，不定数值）：
   - **现状**（profile 怎么做）：热点在哪、环境事实、父本结构——事实获取方法
   - **目标**（benchmark 怎么测）：评估方法（K 次计时中位 + score 语义）
   - **约束**（verify 怎么判）：合法性判定（正确性容差/接口/规模）
4. **检索三问**（第四类资产）：搜索方法 / 搜索信源（按角色）/ 关注点（**必须与特征格绑定**）。
5. **门禁定量决策**：容差 / 规模（sizes 数组）/ K / score 上界下界 / 超时 / 软门禁 band 与 baseline。
6. **优化幅度阈值**（debug_policy）：超阈值触发 debug 复验（防错误性能结论进精英地图）。
7. **多维 objective 定义**（v2 新增，human 决策）：
   - **约束列表**（`constraints: [{name, op, value}]`——如 max_error ≤ 1e-3、tp99 ≤ 100ms）
   - **加权指标列表**（`metrics: [{name, weight}]`——如 speedup 0.7 + memory_eff 0.3；
     **目标可以是多维的**：约束之下追求多指标加权和最大，不要求单一指标）
   - **融合函数**（`fusion: "weighted_avg"` 缺省；env 可自定义融合公式）
   - 指标取值链：benchmark 输出多指标 + gate_check 派生归一（low 指标→speedup 类高优派生）
8. **subagent 超参数**（agent_hyperparams）：adjust 流水线各步的 {model, thinking} 档位
   （null=跟随默认；发散步骤[视角生成]可配高 thinking 档+轻模型，收敛步骤[判断/逆转]配强模型）。
9. **资源声明**（env_slots）：同环境并发实验数 + 分时复用规则描述（写进 domain_knowledge.md
   ——env 设计者写清并发实验如何交错使用独占资源；调度器只管资源位分配）。
10. **初始特征空间**：2-4 轴正交（name/type/取值域）。
11. **初始解确认**：naive 起点过约束门（慢但必须对）。

### 步骤② 四 skill 构建方法（**必须从 human 主动获得**）

**硬要求**：profile / benchmark / verify / research 四个基本 skill 的**构建方法必须从 human
主动获得**（human 亲自描述每件 skill 怎么构建——评估逻辑/约束语义/检索口径是 human 的设计
决策，AI 只追问澄清、不代答、不写死）。追问清单（每件 skill 各过一遍）：

| skill | 必问的问题 | 产出 |
|---|---|---|
| `profile.py` | 热点怎么测（cProfile/samples？）/ 环境事实采哪些 / 父本结构怎么呈现 | 现状事实 JSON |
| `benchmark.py`→`gates/benchmark_impl.py` | 计时方法 / K 次与中位 / score 语义与上界下界 / ref 怎么测 / 多指标怎么出 | {多指标, score, timings, k} |
| `verify.py`→`gates/verify_impl.py` | 正确性怎么判（容差/规模/seeds/一致性两遍）/ 已知作弊模式怎么静态拒 | {passed, max_error, stage} |
| `research.py` | 检索方法 / 信源（按角色）/ 关注点（绑定特征格） | 检索指引 JSON |

**gate_check.py 组合复用**（不重复实现）：`gates/gate_check.py` 调用 verify_impl + benchmark_impl
（subprocess 执行引用）→ 约束校验（objective.constraints）→ 加权融合（objective.metrics）
→ 软门禁（--baseline）→ v2 融合分协议 JSON。**verify/benchmark 的实现只写一份**（gates/ 内），
skills/ 薄入口执行引用——框架、agent、gate_check 三方同一实现，禁止两套。

### 步骤③ 演练（基础设施验收——逐件真实执行）

**演练 checklist**（每项真实运行，产物断言后才算通过）：

```bash
cd <env_dir>
PY=.venv/bin/python    # 或 python3

# ① 三要素各自独立可跑（reference 各出合法 JSON）
$PY skills/profile.py    --candidate code/reference.py   # → 事实 JSON（hotspots/env_facts）
$PY skills/verify.py     --candidate code/reference.py   # → {"passed": true, ...}
$PY skills/benchmark.py  --candidate code/reference.py   # → {runtime_ms, score, timings, k}

# ② gate_check 融合分协议闭环（v2 JSON：passed+score+metrics+constraints）
$PY gates/gate_check.py  --candidate code/reference.py [--baseline <score>]
# → {"passed": true, "score": >=0, "metrics": {...}, "constraints": {...}, ...}
# 失败路径演练：naive 起点带 --baseline → score=-4（soft_gate）；作弊样本 → score=-2（correctness）

# ③ naive 起点过约束门（慢但必须对——它是进化起点）
$PY skills/verify.py     --candidate code/naive_start.py # → {"passed": true, ...}

# ④ 检索指引四角色合法（三元组非空）
$PY skills/research.py   --role adjust                   # → 检索指引 JSON（method/sources/focus）

# ⑤ 门先证红自动化（一次跑完）
python3 <repo>/super/superopt/gates.py 的 CascadeGate.preverify_red()
# → {"ok": true, "leaked": [], "false_killed": [], ...}（exploit 全拒 + 金样例零误杀）

# ⑥ env.json 键 ↔ 脚本读取键一致性（human 过一遍：声明的每个键都有消费者）
```

### 步骤④ 分层验证机制（验证金字塔）

**引导 human 对验证分级**——先分钟级快速验证论证改动有效性，再端到端验证（无效解以最低成本排除）：

| 层级 | 参数 | 约定语义 | 典型耗时 | 用途 |
|---|---|---|---|---|
| L1-M0 | `--level L1-M0` | 系统 L1 层的 M0 模块**单独验证** | 秒级 | 单模块单测级快检 |
| L{n} | `--level L{n}` | 第 n 层**整层验证** | 秒~分钟级 | 层级集成验证 |
| E2E | `--level E2E`（或 gate_check 默认） | **端到端验证**（部署+预热+全量测试） | 分钟级 | 框架终验（每实验必跑） |

- **参数协议**：verify/benchmark 薄入口透传 `--level`；实现侧按级别裁剪规模/次数（L1 单规模
  单次、L3 全量 K 次中位、E2E 走 gate_check 全级联）
- **verify_levels 定义**（env.json）：每级 {cmd, budget, caller}——渲染进 domain_knowledge.md
  的验证矩阵表（agent 与 human 可读）
- **验证参数交给领域专家知识**：domain_knowledge.md 的验证金字塔节引导实验 agent 自行小范围
  验证（builder 可选 L1 预检关键假设）——**调度器 gate_check（E2E）与 agent 侧 L1 预检共享
  同一实现入口**，口径不漂移
- **debug 失败跳过 E2E**（调度器协议）：debug agent 反思后主动上报实验失败 → 调度器跳过
  端到端验证（省部署/预热/测试全流程）加速迭代

### env 升级路径（触发场景之二）

既有 env 的演进：门禁加强（新增 exploit 样本/收紧容差）/ 评分调整（objective 加权改版）/
特征轴增改 / 任务域换版本 / debug_policy 调整 / subagent 超参数调档——走步骤①的对应环节
（只问要改的），改动落盘后**必重跑步骤③演练**（基础设施无回归才交付）。

### 7. 轨迹分析 → 提示词进化（《自改进的进化搜索算法》外层循环）

跑完一轮后，进化不该止于解——**搜索算法本身用这轮的轨迹证据迭代**：

- 跑 `python3 super/tools/adjust_trace.py --workdir <ws>` 得决策-效果对照表；读 `decisions/`
  （adjust 决策记录）、`ledger`（实验台账）、`designs/`（builder 决策档案）、实验记录与
  research_report、`profile_history/`
- 诊断坏模式（震荡/过激/迟钝/漏杀/探索不足）→ 处方：改写 `env/prompts/<role>.md`
  （一次只改一个角色）→ 版本管理（延迟具体化 + `_history/` + PROMPTS_LOG）→ 下轮验证回环
- **adjust step7 实验自改进**：adjust 流水线会直接写 domain_knowledge.md 的 bug 解法（追加节）、
  经 humans 提案队列提 skill/验证器修改（human 裁决后应用）——本节是 human 主导的升级通道
- 两级接通：env 级资产在 env 里进化；系统性坏模式升级 rick learning/dream 循环（仓级知识）
