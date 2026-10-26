# skill:gen-env（super 进化 env 生成器）

## 前置输入（先读再访谈——避免重复劳动）

在开始任何访谈之前，先读工作区里已有的材料——三要素与检索三问的素材多半已经躺在那里：

| 路径（若存在） | 读来做什么 |
|---|---|
| `<ws>/requirement.md` | human 的原始需求（要优化什么） |
| `<ws>/grilling/design-tree.md` | 设计树追问的产出：目标/约束/现状的候选表述、研究结论、judge 节点裁决、实测数据 |
| `<ws>/research-*.md`、`<ws>/research/` 下简报 | 领域调研：权威做法、论文线索、环境事实、已知死族 |
| `<ws>/decisions/`、`<ws>/designs/`（升级路径时） | 上轮的 adjust 决策与 builder 设计（判断哪些三要素/检索口径需要改） |

**纪律**：已有素材 → **复述确认**（「设计树里写的目标是 X、约束是 Y，我按这个继续，对吗？」），不重新追问；缺什么补什么。**重复访谈是对 human 时间的浪费，也是本 skill 最常见的低效来源。**

## 三要素脚本统一契约（硬性，缺一条 agent 与框架就会漂移）

三个脚本必须满足同一 CLI/IO 契约，框架与 agent 才能共用（env_9 实测：模板版 verify 只读
`correctness.size`（单数）而 env.json 声明 `sizes: [64, 8192]` → 自检只测 64 规模，判断失真、
debug 连烧三轮）：

1. **`--candidate <path>` 是唯一必需参数**；其余全部从 `env.json` 读（不写死规模/容差/超时）。
2. **必须支持 `--env-dir <dir>`**（缺省=脚本上两级）；gate_check 与其他 agent 调用时统一传入。
3. **多规模必须读 `correctness.sizes`（数组）**，`--size` 仅作覆盖；不得只读单数键。
4. **stdout 单行 JSON**；失败也要输出 JSON（`{"passed": false, "error": ...}`），非 JSON = 契约破坏。
5. `gate_check.py` 必须支持 `--baseline <perf>`（软门禁相对带比较）并从 `env.json` 读 `tolerance.soft_gate.relative_band`。
6. **`research.py`（第四类资产）契约独立于上面三条**：`--role <adjust|builder|coding|debug>` 为必需参数（`--cell`/`--env-dir` 可选），同样 stdout 单行 JSON（检索指引）；**不执行网络请求**——检索动作由 agent 的 `web_search` 完成，脚本只约束「怎么搜/去哪搜/关注什么」（见第 1.5 节）。
7. **`--preverify` 语义**：门先证红的统一实现是框架 `CascadeGate.preverify_red()`（调度器 `--preverify-red` 同一实现）——**不要手工逐个试 exploit**（见第 5.3 节）。

## 触发场景

super 会话（super-loop）中出现以下情形时加载本 skill：

- **env 缺失**：`rick super start` 前目标工作区还没有 `env/`，需要现场生成一个可运行的进化环境
- **env 升级**：既有 env 需要演进——门禁加强（新增 exploit 样本/收紧容差）、评分方向调整、特征轴增改、任务域换版本、**优化幅度阈值（debug_policy）调整**、**提示词进化（跑完一轮后用轨迹证据迭代四角色模板，见第 7 节）**
- 信号词：「建一个 env」「这个域的进化环境」「门禁怎么设计」「加特征轴」「收紧正确性容差」「分析上轮 adjust 的决策轨迹」「进化 builder/coding 的 prompt」

matmul 类任务不必从零访谈：`super/env_templates/matmul/` 模板已是完整实例，本 skill 的流程用于**定制**它或从零建设其它任务域。

## 预期效果

产出一个**可运行的 env**——通过验收自检（见核心内容第 5 节）的完整环境，super 调度器可立即在其上跑进化循环：

- 目录协议落地（相对 super 工作区 `~/.rick/super/<name>/`）：

```
env/
├── env.json                 # 任务定义：任务名 / perf_desc（评分方向）/ perf_type（单位）/
│                            #   容差声明 / **debug_policy（优化幅度阈值）**
├── feature_space.json       # 初始特征空间：2-4 轴（name/type/取值域），轴间正交
├── code/                    # 参考实现（正确性锚点）+ naive 起点（进化起点）
├── skills/                  # 三要素 + 检索（进化算法基石；框架与 agent 共用同一份资产；**五件**）：
│   ├── profile.py           #   现状：事实获取（热点/环境事实/父本结构）→ 事实 JSON
│   ├── benchmark.py         #   目标：评估方法（K 次计时中位 + score 语义）→ {perf_type, score, timings, k}
│   ├── verify.py            #   约束：合法性判定（正确性容差/接口/规模）→ {passed, max_error, stage}
│   ├── gate_check.py        #   门禁封装：verify→benchmark→软门禁；框架门禁与 agent 共用的唯一入口
│   └── research.py          #   第四类资产：检索三要素（方法/信源/关注点）按四角色输出检索指引（见第 1.5 节）
├── data/                    # legacy 兼容（旧 env 的 test_correctness.py + bench.py；新 env 不再新增）
├── domain_knowledge.md      # 领域知识 + 「## 现状基线」「## 性能分析 SOP」「## 性能调优 SOP」
├── requirements.txt         # 非标准库依赖声明（如 numpy）
├── .venv/                   # 可选：依赖隔离（python3 -m venv；门禁优先用 .venv/bin/python）
└── prompts/                # 可选：四角色提示词模板的 env 级定制/进化产物（见第 7 节，延迟具体化）
gates/                       # 工作区级（非 env 内）：
├── exploit_registry.json    # 已知作弊/坏解样本（门先证红的种子）
└── golden_examples/         # 金样例（参考实现即首个金样例）
```

- 关键纪律：**门禁的容差/评分函数/阈值属于 env 设计范畴（human 决策），本 skill 只引导设计，不代答、不写死**；rick 框架（superopt gates.py）只提供级联门结构（环境健康→正确性→性能 + 软门禁多计时中位数）并读 env 的定义执行——**新布局下框架门禁只调 `skills/gate_check.py`（三要素封装），agent 侧同一份资产随时可调**（口径统一，禁止两套实现）。

- **与仓级方法 skill 的关系（两层资产，别混淆）**：

| 层 | 资产 | 谁生成/进化 | 内容 |
|---|---|---|---|
| **env 级** | `env/skills/` 五件（profile/benchmark/verify/gate_check/research） | **本 skill 生成**（第 5 节验收） | 该任务域的事实/评估/约束/检索**定义**——每个 env 各不相同 |
| **仓级** | `.rick/skills/super_*`（实验分析 / SENSE / 格管理 / 实验设计 / debug 协议）+ 复用的 `sense`/`tdd`/`debug-skill` canon | rick learning/dream 循环 | **跨 env 通用的方法**——agent 提示词声明路径+触发条件，按需 read |

## 核心内容

访谈式流程——**逐环节引导 human 决策**，每环节产出物落盘后再进下一环节。

**流程顺序（不可跳步）**：

```
0. 任务定义（参考实现 + naive 起点）        ← 先有可运行的对象
0.5 领域研究 / 实测（跑 reference 拿事实）  ← 访谈有数据依据
1. 三要素追问（**定性**收敛：能转成脚本）
1.5 检索三问（方法 / 信源 / 关注点）
2. 门禁参数（**定量**决策：容差 / 规模 / K / score / 超时）
2.5 优化幅度阈值（debug_policy——human 确认）
3. 初始特征空间（2-4 轴正交）
4. 初始解（naive 起点确认）
5. 验收自检（五件齐备 + 门先证红自动化）
—— 跑完一轮后 ——
7. 轨迹分析 → 提示词进化（《自改进的进化搜索算法》外层循环）
```

### 0. 任务定义（参考实现 + naive 起点）

**先落两个可运行的文件，后面才有实测与访谈的对象**：

- 确定任务名、输入输出（例：两个 512×512 float32 矩阵 → 乘积矩阵）。
- **参考实现**：写 `env/code/reference.py`——用该生态最权威直白的写法（如 numpy.matmul）。它是正确性锚点（门先证红的金样例之首）。
- **naive 起点**：写 `env/code/naive_start.py`——无优化的朴实写法（如纯 Python 三重循环）。进化从这里起步，「慢」是特性不是缺陷。
- 依赖声明：`requirements.txt`（如 numpy）+ 可选 `.venv`（`python3 -m venv env/.venv && env/.venv/bin/pip install -r env/requirements.txt`）。

### 0.5 领域研究 / 实测（拿事实——为第 1 节「现状」追问提供数据基础）

**为什么先实测再访谈**：第 1 节要求 human 描述「现状（基线多少/热点在哪）」，但此时若没有实测数据，human 只能凭经验猜。**先用参考实现跑出事实，访谈才有依据**（env_9 实证：设计树里的基线/容差全部来自研究阶段的实测，不是访谈出来的）。

步骤：

1. **读前置输入**：`<ws>/grilling/design-tree.md`、`<ws>/research-*.md`（有则读，已有研究结论不重复做）。
2. **跑参考实现拿基线**：`env/code/reference.py` 在任务自然规模（如 matmul 512 / 8192）下实测——
   耗时（多次取中位）、热点（cProfile top-5 与占比）、环境事实（语言/库版本/BLAS 后端/CPU 核数）、naive 起点耗时（父本基线）。
3. **落盘事实**：写入 `env/domain_knowledge.md` 的 **`## 现状基线`** 段（含「怎么测的」+ 机器指纹/日期），
   作为后续 profile 时间序列的第一份事实、也是第 1 节「现状」访谈与第 2 节参数决策的依据。
4. **对账纪律**：human 在第 1 节给出的数字若与实测不符 → **以实测为准**，并在 domain_knowledge 记差异与原因（负载态/机器差异/测量口径），不得让口头估计覆盖实测。

产出：`domain_knowledge.md` 的现状基线段（必要时在 grilling/design-tree 追加实测附录）。

### 1. 三要素追问（**定性**收敛——只到「能写成脚本」，不定具体数值）

> **职责边界（与第 2 节的分工，不重叠）**：本节只做**定性收敛**——把「目标 / 约束 / 现状」说清楚到**能转成对应脚本**的程度（脚本的输入输出与语义）。**具体数值**（容差 ε 取值、bench 规模、K 次、score 公式、超时上限）一律留到第 2 节「门禁参数」做定量决策；**本节不追问数值**。

**设计原则（human 裁决）**：一个 env 必须具备**三要素**——它们是进化算法的基石，缺一不可：

| 要素 | 泛化语义 | 落成的 skill | 输出协议 |
|---|---|---|---|
| **目标** | 要优化的目标与评估方法（怎么算「变好了」） | `skills/benchmark.py` | `{"<perf_type>": num, "score": num, "timings": [...], "k": 3}` |
| **约束** | 优化过程中绝对不能破坏的东西（合法性边界） | `skills/verify.py` | `{"passed": bool, "max_error": num, "stage": str}` |
| **现状** | 事实信息的获取（当前长什么样、热点在哪、环境如何） | `skills/profile.py` | 事实 JSON（热点/环境事实/父本结构） |

三者由 `skills/gate_check.py` 封装（verify → benchmark → 软门禁）——**框架门禁只调它，agent（builder/coding/debug）也调它**。

**纪律（不可协商）**：

1. **必须由 human 亲自描述三者**——AI 只做追问与复述确认（「我理解你的目标是……对吗？」），**不得代填**（代填的三要素会把 human 的领域判断偷换成 AI 的猜测，进化方向随之偏航）。
2. **判据 = 能写成脚本**：每个要素都必须收敛到「可以写成对应 skill 脚本、输出协议合法」的程度，否则继续追问。
3. 三者都清晰后才进入第 1.5 节（检索三问）与第 2 节（定量参数）。
4. **现状以实测为底**（第 0.5 节）：追问时把实测基线/热点摆出来让 human 确认或修正，不是让 human 凭记忆报数。

**逐要素追问样例（定性口径）**：

- **目标**：「你说的『快/好』是哪个维度——墙钟耗时、吞吐、还是精度保持下的资源占用？」「它能不能被一条命令量出来：输入是什么、输出是哪个数？」「score 该表达什么含义（相对参考的提升？绝对水平？）」→ 数值（K 次、规模、score 公式）→ 第 2 节。
- **约束**：「什么绝对不能破坏——数值结果、输入输出形状/接口、资源上限？」「破坏它们时判定应该是什么形态（布尔通过 / 误差上限 / 违规计数）？」→ 容差与阈值 → 第 2 节。
- **现状**：「要看清现状，必须知道哪些事实——基线水平、瓶颈位置、环境常量、父本形态？」「这些事实能不能用一次剖析拿到？」（第 0.5 节已给出初版，human 补充/修正）

**非 matmul 泛化示例（prompt 优化任务）**：

- 目标 → `benchmark.py`：在固定评测集上跑 N 条样本，输出 `{score, timings, k}`；定性问「评测集是什么？分数表达什么（通过率/平均分）？」→ 规模与次数留第 2 节。
- 约束 → `verify.py`：输出格式合法性（JSON 可解析/字段齐全/长度上限）→ `{passed, max_error(违规计数), stage}`；定性问「什么算非法输出？」
- 现状 → `profile.py`：当前 prompt 的事实画像（长度/结构/指令条数）+ 失败样本分类统计；定性问「失败样本集中在哪里？」

### 1.5 检索三问（第四类资产——「认知注入」，突破局部最优）

**为什么需要它（human 洞察，不可省略）**：进化搜索在**语义空间**里做组内变异，容易陷局部最优——agent 在某个特征格里会以很小的角度反复微调同一种写法（如只调分块大小），长期出不来；这种状态很难说实现了有价值的搜索。**组内变异无法自己跳出，必须从外部注入新认知**。检索就是这条通道：复用前人的经验（使用手册/代码样例/论文），把搜索从「自己猜」变成「站在已有知识上改」——这是我们能越来越强、突破局部最优的核心方法。

第四类资产 `skills/research.py` 承载检索的**三要素（三元组）**，按 agent 角色分别固化：

| 参数 | 含义 | 决策要点 |
|---|---|---|
| **搜索方法** | 用什么方式检索 | `web_search` 关键词组合 / 官方文档站内搜索 / GitHub 代码搜索 / arXiv 站内与 API |
| **搜索信源** | 去哪检索（按角色分） | coding/debug：官方文档、代码仓库优秀实现、技术问答；builder/adjust：arXiv、OpenReview、PapersWithCode |
| **关注点** | 检索什么（**必须与特征格绑定**） | 见下方角色分工——同一域里不同角色关心的完全不同 |

**角色分工（默认口径，human 可按域调整）**：

| 角色 | 检索类型 | 关注点 |
|---|---|---|
| `coding` / `debug` | **使用手册类**（事实/API 用法） | 该格实现细节：API 用法与限制（参数约束/dtype 行为/已知坑）、可参考的代码样例 |
| `builder` | **idea 类**（实验设计） | 实验细节、数据与方法：benchmark 配置、规模影响、硬件差异对比、可复现做法 |
| `adjust` | **idea 类**（创新点） | 创新点与研究趋势：尚未覆盖的方法族、近期新方向（加格的判断依据） |

**纪律（不可协商）**：

1. **必须由 human 亲自描述三问**——AI 只做追问与复述确认（「我理解 coding/debug 查官方文档与代码样例、builder/adjust 查论文，对吗？」），**不得代填**（检索信源与关注点是领域判断，代填会让 agent 查错方向、把回合烧在无关资料上）。
2. **关注点必须与特征格绑定**：不能只写「查矩阵乘优化」，要写「分块策略族的格关注 cache 分块大小与数据布局的实验数据」——**绑定到格，检索才能真正收敛探索空间**（这是三元组里最重要的一项）。
3. **判据 = 能写成 `skills/research.py` 的 `ROLE_PROFILES`**：四角色各一组 `{method, sources, focus, suggested_queries}`，脚本按 `--role`（+`--cell`）输出单行 JSON 指引；缺任一角色、或任一参数为空 = 未达标。
4. **research.py 只产出检索指引，不执行网络请求**：检索动作由 agent 自己的 `web_search` 完成；脚本负责约束「怎么搜/去哪搜/关注什么」，避免 agent 广泛乱搜。

**追问样例**：

- 「你的域里，开发者查什么文档最有效？有没有权威手册/源码仓库值得让 coding agent 直接查？」
- 「论文该关注创新点还是实验细节？builder 设计实验时需要哪些数据（配置/规模/硬件）？」
- 「这个域的主要解法有哪几个流派？哪些流派当前 env 没覆盖（adjust 需要知道去查什么新方向）？」
- 「哪些信源不可靠或过时，要明确排除？」

**非 matmul 泛化示例（prompt 优化任务）**：coding/debug 查「该框架的 prompt 最佳实践文档与官方示例」；builder 查「A/B 实验设计与评测方法论文」；adjust 查「prompt 优化新范式（self-refine / reflexion 等）」。

### 2. 门禁参数（**定量**决策——承接第 1 节三要素）

> **本节承接第 1 节的三要素（已定性收敛），只做定量决策**：把具体数值、规模、公式定下来——**不重复追问「目标是什么 / 约束是什么 / 现状是什么」**，直接问「取多少 / 用哪个公式 / 上限设多少」。每项都给出该域的具体选项，不代答。

**① 评分方向（env.json 顶层）**

- `perf_desc`：**low**（越小越好，如 runtime_ms）或 **high**（越大越好，如 score/吞吐）。
- `perf_type`：单位（runtime_ms / score / …）。
- 同格替换的比较方向由框架按此字段感知——这是 human 的领域判断，不是框架默认。

**② 约束参数 → `env/skills/verify.py`**

输出 stdout 单行 JSON `{"passed": bool, "max_error": float, "stage": "..."}`。逐项定量：

- **校验规模**：`correctness.sizes`（**数组**，如 `[64, 8192]`；必须显式声明——脚本优先读数组，缺省才回退单数 `size`）+ 固定 seed（如 42）+ `extra_seeds`（额外随机 seed 数）+ `double_run_consistency`（同输入跑两遍一致性）
- **容差 ε**：`tolerance.correctness.max_error`（1e-3？1e-5？按任务精度要求定）
- **超时**：`correctness.timeout_s`（单候选单规模执行上限）
- 候选异常/超时如何判（框架把非 JSON/超时按该 stage 失败处理）
- `gates.static_check`：是否做候选源静态检查（拒已知篡改模式）
- 框架与 agent **共用同一份脚本**（agent 实现后自行调用；框架 stage2 调它）——口径唯一

**③ 目标参数 → `env/skills/benchmark.py`**

输出 `{"<perf_type>": float, "score": float, "timings": [...], "k": 3}`。逐项定量：

- `bench.size`（如 512 / 8192）、`bench.k`（计时次数，**取脚本内 K 次中位数**，如 3）、预热次数、`bench.timeout_s`
- **score 公式**：`score = ref_runtime_ms / runtime_ms`（低优）或等价换算——它就是「到什么程度算赢」的量化定义；上界 `gates.score_max`（如 2.0，超出视为作弊/异常）与下界 `gates.runtime_ms_min`（如 0.01ms，物理不可能=硬编码）
- `gates.ref_runtime_band_ms`：参考实现的合理耗时带（如 `[380, 700]`）——堵「拖慢 reference 抬 score」通道；**换机器需重校**
- 超时上限（坏候选不能拖死门禁）

**④ 现状参数 → `env/skills/profile.py`**

输出事实 JSON（热点函数与占比 / 环境事实：BLAS 后端、numpy 版本、CPU 核数、内存带宽估算 / 父本结构摘要）。逐项定量：

- 剖析规模与超时（`bench.size`/`bench.dtype`/`bench.timeout_s` 复用）
- top-N 热点条数、是否含环境常量、父本结构摘要粒度
- 该脚本同时提供**现状变化的时间维度**：调度器每迭代采一次并归档 `profile_history/<cell_id>/iter{N}.json`，进化轨迹即现状演化史

**⑤ 门禁封装参数 → `env/skills/gate_check.py`**

依次 verify（约束）→ benchmark（目标，K 次中位）→ 软门禁（与 `--baseline` 比较，`tolerance.soft_gate.relative_band` 读 env.json）→ 输出 `{passed, failure_stage, metrics, details}` 单行 JSON。

- `tolerance.soft_gate.relative_band`（如 0.05）：性能劣化在带内 = 通过但标记 `tolerance_band`（延迟入库）；带外劣化 = `soft_gate` 失败
- `gates.correctness_timeout_s` / `gates.bench_timeout_s`：框架侧子进程超时（≥ 脚本内超时，留余量）
- 框架 stage2/stage3 调它一次完成级联判定；**agent（builder/coding/debug）也调它**——门禁口径从框架到 agent 完全统一（消除「自检过、框架拒」的语义漂移）
- **不再生成 `self_check.py`**：AI 按需自行调用 profile / benchmark / verify / gate_check 四件，不做混合自检。另加 `skills/research.py`（第 1.5 节）——**五件齐备**。

**⑥ exploit 注册表 + 金样例**

- **exploit 注册表**（`<ws>/gates/exploit_registry.json`）：**引导 human 想「这个域里 agent 会怎么作弊」**，构造 3-5 个已知坏解样本（每条 {id, name, pattern, sample_path}）。常见模式参考：
  - 预存结果（返回缓存/硬编码答案）
  - 篡改计时（计时期间偷工/异步返回）
  - 降精度换速度（float64→float16 之类该域不可接受的精度损失）
  - 破坏输入后部分计算（只算可过校验的子集）
  - 样本写成可执行的小脚本放 `gates/` 下，注册表引用路径
- **金样例**（`<ws>/gates/golden_examples/`）：参考实现即首个金样例；可再补 1-2 个风格不同的已知好解。作用：门禁升级时防误杀（历史合法解必须继续通过）。

### 2.5 优化幅度阈值（debug_policy——human 确认，防错误性能结论进精英地图）

**为什么需要它**：门禁只判「过/不过」与容差带，但**异常大的收益**本身也可疑（可能是作弊或测量异常），**无明显收益的带内解**也可能引错方向；两类都要交给 debug agent 复验，避免错误结论被遗传。

**追问 human（定量）**：

1. 「多大比例的收益才算可疑？——性能提升超过 **X%** 视为异常，需 debug 复验」（默认 **20%**）
2. 「软门禁容差带内的解（劣化但在 band 内）是否也要复验？」（默认 **是**）

**落盘 `env.json`**：

```json
"debug_policy": {
  "gain_threshold_pct": 20,
  "recheck_tolerance_band": true
}
```

- `gain_threshold_pct`：新候选相对该格精英（或全局最优）的性能提升百分比 ≥ 该值 → 门禁通过后仍派 debug 复验（`reason=suspicious_gain`）
- `recheck_tolerance_band`：`gate_check` 判定 `tolerance_band`（带内劣化）时 → 派 debug 复验（`reason=tolerance_band_recheck`）
- human 未明确时用默认值并在 `domain_knowledge.md` 记「用默认，待首跑后校准」；调整该阈值属 **env 升级路径**（改后重跑门先证红）。

### 2.6 成本监控与预算（token 可解释性——建 env 时就要定口径）

一次实验 = builder/coding/debug 多次 LLM 调用，成本是主要运行约束。**先保证计量与可解释，再谈优化**（human 裁决：暂时只做计量与解释）。

- **预算**：`env.json` 的 `max_total_tokens`（0=不限）与 `--max-wall-time` 双轨，命中任一 → 调度器优雅停（`session_ended.budget_exhausted=true`）。首轮建议设 token 上限：先观察一轮的「单迭代平均成本」再定。
- **归因口径（框架写入，权威，勿另建账本）**：`events.jsonl` 的 `agent_ended.usage`（input/output/cache_read/cache_write/total_tokens/cost_usd）与 `iteration_ended.by_role`（本迭代按角色的 tokens/wall_ms/tool_calls 聚合）——**每次迭代的成本可归因到具体 agent**。
- **复盘工具**：`python3 super/tools/cost_report.py --workdir <ws> [--json] [--top N]` → 按角色/迭代的成本表、Top-N 昂贵迭代、上下文膨胀曲线、cache 命中率；能定位 pi session 时补单轮明细（定位失败自动降级为仅事件流）。
- **配套 skill**：`.rick/skills/super_cost_analysis/skill.md`（七维分析 + 浪费模式清单 + 优化处方）——run 收尾与 super-loop Step 7 复盘时加载。

### 3. 初始特征空间

`env/feature_space.json` 的 axes：**2-4 轴起步**，每轴 {name, type, values/取值域, desc}。
- 引导原则：**轴间正交**——每根轴刻画解的一个独立维度（例：impl_strategy=算法策略轴 / memory_layout=数据排布轴；一个讲「怎么算」、一个讲「数据怎么放」，互不蕴含）。
- 轴取值域是「可枚举的行为描述子」（enum）或「可分档的量」——格 id 由框架从描述子确定性生成。
- 宁少勿滥：先 2 轴跑起来，进化轨迹显示解空间贫瘠再加轴（加轴走 env 升级路径）。**每个格后续由 adjust 逐步补 `description`（名字/方法描述/来源引用）**，新建时框架给空描述。

**特征格五要素**（进化算法的核心数据结构；框架落盘在 `archive.json`）：

| # | 要素 | 含义 | 谁维护 |
|---|---|---|---|
| 1 | `name` | 短名（一句话指代这个优化方向） | adjust（`new_cell` 必填；`modify_cell` 可改） |
| 2 | `method` | 方法描述（这个方向具体怎么做；**可引用论文/调研地址** `source_refs`，供 builder 设计实验） | adjust（`new_cell` / `modify_cell`） |
| 3 | `weight` | 采样权重（概率 ∝ weight；0 = 功能性灭绝——格与谱系保留，可复活） | adjust（每轮决策） |
| 4 | `score` | 该格当前最佳精英的归一化得分（框架权威口径） | 框架（admit 时覆盖写） |
| 5 | `code_path` | **该格当前最佳精英代码**（`elite_code_path`）+ **其实验记录**（`elite_record_path` → `codes/iter{N}/experiment_record.md`） | 框架（admit 时同步） |

- 前四要素回答「这个方向是什么 / 值多少预算 / 现在多好」；第五要素把**可复用代码与完整实验档案直接挂在格上**——agent（builder/coding/debug）读格即得精英代码与实验记录，不必扫目录猜。
- 正交性：`new_cell` 必须引入**未被现有格覆盖的优化维度**（新轴值或新描述维度），正交性论证写进决策记录（`decisions/iter{N}.md`）。

### 3.5 并发约束（一格同时只有一个实验；不同格可并行）

**为什么**：一个「实验」= 在某个特征格上的一次迭代（builder→coding→debug→门禁）。同一格上并存两个实验会互相踩精英、污染 profile 历史与实验记录；不同格之间无共享状态，天然可并行。

- **格级互斥（已实现）**：调度器每个实验持格锁（内存 + 文件 `<ws>/locks/cell_<safe_id>.lock`，`O_CREAT|O_EXCL` 原子创建）。采样撞锁 → 重采样到其它格（事件 `cell_lock_resample`）；全部格被占 → 短暂等待并记 `cell_locked_skip`（**不消耗迭代号**：跳过的不是实验），连续超 `cell_lock_max_skips`（默认 60）→ `cell_lock_overflow` 优雅停（防死等空转）。
- **陈旧锁回收**：锁文件 mtime 超 `cell_lock_ttl_s`（默认 3600s）视为崩溃遗留 → 清理并记 `cell_lock_stale_recovered`；启动时清理孤儿锁记 `cell_lock_orphans_released`。
- **并发度**：`max_parallel_experiments`（默认 1）。**当前实现为保守串行**——设为 >1 时调度器记 `parallel_degraded` 事件并在 stderr 说明，仍按串行执行（并发槽要求事件流/档案单写者纪律，正确性保障仍在演进）。**建 env 时不要依赖并发**；需要吞吐优先加迭代预算或降低单迭代成本。
- **对 env 的要求**：三要素脚本必须**可被并发调用**（无共享可变状态、不写固定临时文件名——用 `tempfile` 或带 candidate 哈希）；benchmark 计时对并发负载敏感，放宽 `gates.ref_runtime_band_ms` 前先确认是否真要点并发。

### 4. 初始解

- `env/code/naive_start.py` 即进化起点（第 0 节已落）。
- 框架行为：每迭代把选定父本整拷到 `codes/{uuid}/` 作为工作副本——env 的 code/ 目录保持只读圣洁。

### 5. 验收自检（env 交付前逐条过）

#### 5.1 五件 skill 可用

```bash
PY=env/.venv/bin/python   # 或系统 python3（无 .venv 时）
# ① 三要素各自独立可跑（跑参考实现各出合法 JSON）
$PY env/skills/profile.py   --candidate env/code/reference.py   # → 事实 JSON（热点/环境/父本）
$PY env/skills/verify.py    --candidate env/code/reference.py   # → {"passed": true, "max_error": ...}
$PY env/skills/benchmark.py --candidate env/code/reference.py   # → {"<perf_type>": >0, "score": ..., "timings": [3 个]}
# ② 门禁封装融合 pass（框架同一入口）
$PY env/skills/gate_check.py --candidate env/code/reference.py
# → {"passed": true, "failure_stage": null, "metrics": {...}}
# ③ naive 起点过约束（慢但必须对——它是进化起点）
$PY env/skills/verify.py --candidate env/code/naive_start.py
# → {"passed": true, ...}
# ④ 检索指引四角色合法（三元组非空——第四类资产）
for role in coding debug builder adjust; do
  $PY env/skills/research.py --role $role
  # → {"role": ..., "method": ..., "sources": [...], "focus": ..., "suggested_queries": [...]}
done
```

#### 5.2 env.json 键 ↔ 脚本读取键一致性

**为什么必须查**（env_9 实测教训）：配置只写了 `correctness.size`（单数）而 `verify.py` 读 `correctness.sizes`（数组）→ 自检只测小规模、判断失真，debug 连烧三轮。**脚本读取而 env.json 未声明的治理性键 = 真失真，必须显式声明**（有默认值的可容忍，但治理性键必须写）。

| 脚本 | 读取的 env.json 键 |
|---|---|
| `skills/benchmark.py` | `bench.size`、`bench.k`、`bench.timeout_s`、`gates.score_max`、`gates.runtime_ms_min`、`gates.ref_runtime_band_ms` |
| `skills/verify.py` | `correctness.sizes`（数组优先，缺省回退 `correctness.size`）、`correctness.timeout_s`、`correctness.extra_seeds`、`correctness.double_run_consistency`、`tolerance.correctness.max_error`、`gates.static_check` |
| `skills/gate_check.py` | `tolerance.soft_gate.relative_band`、`gates.correctness_timeout_s`、`gates.bench_timeout_s`、`bench.k`（`--k` 覆盖） |
| `skills/profile.py` | `bench.size`、`bench.dtype`、`bench.timeout_s` |
| `skills/research.py` | 自身 `ROLE_PROFILES`（可选 `env.json` 的 `research.<role>` 覆盖） |
| 框架调度器 | `task`、`perf_desc`、`perf_type`、`debug_policy.gain_threshold_pct`、`debug_policy.recheck_tolerance_band` |

一致性自检（缺失即红）：

```bash
PY=env/.venv/bin/python
$PY - <<'PYEOF'
import json, sys
cfg = json.load(open("env/env.json"))
need = ["bench.size", "bench.k", "bench.timeout_s", "bench.dtype",
        "correctness.sizes", "correctness.timeout_s", "correctness.extra_seeds",
        "correctness.double_run_consistency", "tolerance.correctness.max_error",
        "tolerance.soft_gate.relative_band", "gates.correctness_timeout_s",
        "gates.bench_timeout_s", "gates.score_max", "gates.runtime_ms_min",
        "gates.ref_runtime_band_ms", "debug_policy.gain_threshold_pct",
        "debug_policy.recheck_tolerance_band", "perf_desc", "perf_type"]
def get(path):
    cur = cfg
    for part in path.split("."):
        if not isinstance(cur, dict) or part not in cur:
            return None
        cur = cur[part]
    return cur
missing = [k for k in need if get(k) is None]
print("MISSING:", missing)
sys.exit(1 if missing else 0)
PYEOF
```

#### 5.3 门先证红自动化（一次跑完——不要手工逐个试 exploit）

```bash
# 框架入口（与调度器 start 时 --preverify-red 同一实现）：CascadeGate.preverify_red
cd <仓库根>
$PY - <<'PYEOF'
import sys
sys.path.insert(0, "super")
from superopt.gates import CascadeGate
r = CascadeGate("<ws>/env", "<ws>/gates").preverify_red()
print(r)
sys.exit(0 if r.get("ok") else 1)
PYEOF
# 判据：ok=true ⇔ leaked=[]（exploit 全部被拒，拒识率 100%）
#                 且 false_killed=[]（金样例零误杀）
# 输出键：{"ok", "leaked", "false_killed", "exploits_checked", "goldens_checked"}
```

调度器 `python3 -m superopt --workdir <ws> --preverify-red` 启动时跑同一检查（不通过以非 0 退出，进化不启动）。

以上 5.1–5.3 全过 → env 合格，回 super-loop Step 2 启动。
任一不过 → 修对应环节（容差/样本/脚本/键声明），**不带着病门禁启动进化**。

### env 升级路径（触发场景之二）

- 改门禁（容差/评分/exploit 注册表）或 **`debug_policy`（优化幅度阈值）** → **必须重跑门先证红**（exploit 全拒 + 金样例全过，见 5.3），建议顺带跑双回归（elite 回放 + exploit 回归）；改完下迭代自动生效（调度器每迭代 subprocess 直调 pi，无需重启）。
- 加特征轴 → 新轴只影响后续采样；既有格不删（不删格原则），旧格降权走 adjust 提案（run 内的运行时加格走 adjust 的 `new_cell`/`expand_cell` 提案→裁决链；跨 run 的轴级改版走本升级路径）。
- 升级访谈只需覆盖变更涉及的环节 + 第 5 节清单重过（**前置输入里的 `decisions/`、`designs/`、`ledger.jsonl` 是升级判断的主证据**）。

### 7. 轨迹分析 → 提示词进化（《自改进的进化搜索算法》外层循环）

搜索的「智能」住在 env 层的可进化资产里（特征空间 + 门禁 + **四角色提示词模板** + **`research.py` 检索三元组**）。

> **检索的定位（第四类资产为何值得进化）**：进化搜索在语义空间做组合变异，**容易陷局部最优**——agent 在某一特征格内以小角度反复微调同一写法，此时搜索的「价值」已枯竭，但组内变异无法自救。**检索是外部认知注入通道**：当轨迹显示某格 idea 同质多轮，正确的处方往往不是再调权重，而是**改 research 三元组**（放宽关注点/换信源/换检索方法）——让 agent 先看看这个领域里别人怎么做的。三元组的「关注点」与特征格绑定，是让检索真正收敛探索空间（而不是泛读废文献）的关键。
本环节用上一轮 run 的**执行轨迹证据**迭代这些资产——跑一轮，不仅解在进化，搜索算法本身也在进化。

适用对象：**四角色提示词均适用**（`adjust.md` / `builder.md` / `coding.md` / `debug.md`），
以及 **`skills/research.py` 的 `ROLE_PROFILES`（检索三元组：方法/信源/关注点）**——当轨迹显示某格 idea 同质多轮（局部最优征兆）时，改三元组比改模板更治本；
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
   `events.jsonl` 里的 `adjust_decision` / `weight_changed` / `proposal` / `expand_cell` / `new_cell` / `cell_modified` 事件、
   **`<ws>/designs/`**（builder 每迭代的决策落盘：profile 事实 / 参考论文（arxiv_id+insight+applicability）/ 灵感来源 / 推理 / 实验计划——进化「意图层」的完整档案，评估实验设计质量的主证据）、
   **`<ws>/decisions/`**（adjust 每轮的决策记录：主要矛盾判定、research 引用、动作与依据——决策可解释性的主证据）、
   **`<ws>/profile_history/<cell_id>/iter{N}.json`**（每格 profile 时间序列——现状如何演化：热点迁移、耗时结构变化、环境事实漂移）、
   **`env/skills/`** 五件（三要素四件 + `research.py` 检索三元组的实际实现——口径是否被 candidate 特定 hack、容差与评估是否匹配目标、检索关注的与格是否对得上）。
2. **诊断坏模式**（对照表 + 快照逐条核对）：
   - **震荡**：同一格反复升降权（A↑B↓ 来回摆）
   - **过激**：单次事件证据就大改权重（如 0.5→0）
   - **迟钝**：连续多迭代的失败证据不动权重
   - **漏杀**：垃圾格（持续差 perf 且无改善信号）长期占采样预算
   - **探索不足**：全程 new_cell/expand_cell 从未发生（数据饱和后仍不加格）
   - **builder/coding/debug 各自的坏模式**：计划与结果脱节（builder 计划从不被执行）、
     自检结论与框架门拒（coding 未调 `gate_check`——口径应是同一份脚本，出现即模板/注入链问题）、debug 修不动同类根因（缺诊断 SOP）
   - **三要素未被使用**（新布局专项）：`designs/` 里 `papers` 全空或泛泛引用（未真正检索 arXiv 或检索了不落到可实现机制）、
     `profile_facts` 与 `profile_history` 对不上（builder 没真看现状变化）、coding/debug 的 prompt 快照里无
     `verify`/`benchmark`/`gate_check` 调用痕迹（四件 skill 白给了）——处方是改对应角色模板的三要素使用纪律。
   - **检索类坏模式**（第四类资产专项）：
     - **只检索不落地**：`designs/*.papers` 有论文，但 `plan_steps`/`reasoning` 里没有任何来自论文的可实现机制（典型：抄 arXiv 摘要凑数）→ 处方 = 改 builder 模板的「落地纪律」（每篇必须给出 applicability 并落到某一步）。
     - **完全不检索**：`papers` 长期为空且无说明（未用 `research_brief`）→ 查注入链（research.py 是否存在/命令是否注入）。
     - **检索过窄 / idea 同质**（局部最优征兆）：某格连续多轮 idea 都在同一小角度微调（如反复只调分块大小）、没有任何外来思路——**这正是语义空间进化的局部最优陷阱**，说明 research 三元组的 **focus 太窄或信源太单一**；处方 = 改 `skills/research.py` 的 `ROLE_PROFILES`（放宽关注点/换信源，属 env 级升级，需 human 确认）。
3. **处方：改写 `env/prompts/<role>.md`**——针对诊断出的模式加/改**决策纪律与判断启发**
   （如 adjust：每次权重变动须引用 ≥1 条事件 seq 号 / 同类连续降权须间隔防震荡 /
   何时机值得 new_cell/expand_cell；如 builder：计划的 plan_steps 必须可被 coding 直接执行）。
   只改出问题的角色的模板——其余角色不动（可归因性：一次迭代一个变量）。
4. **版本管理（三件套，缺一不可）**：
   - **延迟具体化**：默认继承仓内模板（不复制不落盘）；仅当需要定制/进化时才首次落
     `env/prompts/<role>.md`——env 协议保持最小。
   - **旧版存档**：改写前把当前版本拷到 `env/prompts/_history/<role>.<yyyymmdd-HHMMSS>.md`。
   - **PROMPTS_LOG.md**（`env/prompts/` 下）：每行记「时间 | 角色 | 版本要点（改了什么/为什么） |
     证据链接（adjust_trace 输出/事件 seq） | **可移植性检查：无绝对路径**」——进化史可审计、可回滚、可对照。
     可移植性检查判据：`grep -n "/Users/\|/home/\|/workdir/" env/prompts/*.md` 为空才算过。
5. **下轮验证回环**：改完模板 → 回 super-loop 启动下一轮 run → run 完成后重跑第 1 步对照——
   **改善 → 固化（PROMPTS_LOG 追加「已验证」）**；**恶化 → 回滚**（从 `_history/` 恢复上一版并记原因）。
   一次只改一个角色的模板（否则改善/恶化都不可归因）。

**可移植性纪律（硬性）**：`env/prompts/*.md` 里的一切路径必须用渲染注入的占位符——
`{{env_dir}}`、`{{profile_cmd}}`、`{{benchmark_cmd}}`、`{{verify_cmd}}`、`{{gate_check_cmd}}`（每迭代由调度器渲染填入），
**禁止硬编码绝对路径**。理由（env_8 实测教训）：写死 `<ws>/.rick/super/env_8/...` 的模板
在 env 复制、改名（env_9）、跨机迁移后全部失效，且把该轮进化锁死在单机上。
占位符为空或原样残留时，模板文案应写成「显示占位符原文或为空 = 该能力不可用/未饱和」——
用可读的兜底表述，而不是让 agent 看到 `{{saturation}}` 这种原始记号去猜。

**可用 ctx 字段清单（改模板时优先用它们，别让 agent 花回合去 read 文件——回合即 token）**：

| 角色 | 已注入的上下文键 |
|---|---|
| builder | `env_dir` `profile_cmd` `benchmark_cmd` `verify_cmd` `gate_check_cmd` `target_profile_history` `parent_profile_history` `inspiration_profile_history` `designs_dir` `recent_designs` `saturation` `domain_brief` `perf_timeline` `budget_status` `ledger_brief` `gate_defs` `neighbor_elites` `history_failures` `axis_perf` `feature_axes` `target_record` `neighbor_samples` `research_report_path` `experiment_record_path` `research_brief` |
| coding | `profile_cmd` `verify_cmd` `benchmark_cmd` `gate_check_cmd` `gate_defs` `perf_baseline` `workdir_files` `ledger_brief` `saturation` `experiment_plan` `env_dir` `workdir` `research_report` `experiment_record` `coding_plan_path` `test_path` `research_brief` |
| debug | `profile_cmd` `verify_cmd` `benchmark_cmd` `gate_check_cmd` `gate_failure` `gate_defs` `ledger_brief` `saturation` `failure_stage` `stderr_tail` `self_check_result` `attempt` `experiment_plan` `experiment_record` `perf_gain_pct` `recheck_reason` `research_brief` |
| adjust | `gate_check_cmd` `saturation` `perf_timeline` `budget_status` `domain_brief` `ledger_brief` `own_decisions` `cells_status` `feature_axes` `recent_events` `experiment_records` `experiment_sessions` `decision_history` `cell_descriptions` `decisions_path` `research_brief` |

设计原则：能进 ctx 的结构化信息就不该让 agent 自己去 read（每读一次约消耗 1 回合上下文）；
模板里引用这些键即可。若发现某角色反复读同一文件，优先考虑把它抽进 ctx（改框架 = 升仓级知识，
走 learning 循环），而不是在模板里硬写路径。

**饱和 / 认输（terminate_recommend）解读**：
- `saturation` 非空 = 框架判定「连续 ≥3 迭代无新纪录 + 全部存活格 visit≥2 + 冠军格近 3 次采样未刷新」。
  此时 adjust 应首选 `new_cell`/`expand_cell`（加格 = 探索的唯一入口）；builder 应设计「跳出当前方向」的实验。
- 轨迹分析发现「长时间无新纪录但 adjust 全程未提 new_cell/expand_cell」→ **先查 saturation 是否真的注入了**
  （prompt 快照里该字段为空/占位符残留 = 注入链或模板有问题，先修这个，别急着改 adjust 的判断纪律）。
- 收到 `terminate_recommend` 提案（`proposals/` 里 `needs_human: true`，不进裁决自动执行）→
  与 human 讨论二选一：**收手归档**（北极星终值 + 结论写收尾报告）或 **换轴**（走 gen_env 的轴级改版升级路径）。
- 反向检查：`saturation` 长期为空但 perf_timeline 明显停滞 → 检查框架的饱和条件阈值是否过严（仓级问题，走 learning）。

**实验台账（`ledger.jsonl`）解读**：工作区根下的跨迭代实验档案——每迭代一条：
`iteration_id / cell_id / idea / parents / gate{passed,failure_stage,perf_value} / verdict(admit|replace|reject|no_candidate) /
perf_delta / reject_reason / 各 agent 成本`。轨迹分析时**优先读它**：比 events.jsonl 密度高一个数量级
（一趟读完一轮实验史）；坏模式（反复 reject 的同类 idea、某格长期 no_candidate、某类失败反复出现）
在台账上一眼可见。agents 侧也已注入 `ledger_brief`（最近若干条摘要）供决策时避坑。

**实验记录（`codes/iter{N}/experiment_record.md`）解读**：一次迭代的完整上下文文件——
builder 建（背景与价值 / 实验假设 / 验证逻辑 / 具体步骤 / 观测现象），coding 补（编码计划 / TDD 红绿 / 实现要点），
debug 补（卡点 / 根因 / 解决）。诊断「计划与结果脱节」「同一根因反复出现」时逐份读它。

**模型分配引导（prompt 之外的另一个性能杠杆）**：计划/执行分离下两档模型分工是共识——
`builder` 用贵档（实验计划质量优先），`coding` 可配轻档（执行性任务容错高）。
配置走 `super.json` 的 `config.agent_models`（示例：
`{"builder": "catpaw-proxy/glm-5.3", "coding": "catpaw-proxy/deepseek-v4-flash"}`）——
与模板进化独立调优、独立归因；模型分配的调整同样记入 PROMPTS_LOG。

**两级进化路径（接通点）**：本环节进化的是 **env 级资产**（该 run 的搜索 DNA）；
super-loop / gen_env skill 自身等**仓级方法知识**的进化走 rick 既有 learning/dream 循环
（轨迹分析发现系统性坏模式 → learning job 沉淀 → 更新 loop/skill 与 `super_*` 方法 skill）——两层各司其职、互不越界。
