# orig 调研素材 · 阶段 1（问题还原）· 第 1 批

> 主题：superopt 出论文——「期望 vs 现状」的事实底座
> 方法：superopt 代码原文（一手）+ OpenAlex 元数据 API + arXiv/会议官网原文抓取（一手）+ OpenEvolve README（一手）
> 用途：供用户完成阶段 1 思考（分清期望/现状、事实回溯置信信源、差距成形），不代替其定框架

## A. 「期望」端的事实——中顶会对论文的要求是什么

### A1. ICSE 研究轨（AI4SE 类工程系统论文的主战场）的官方门槛
- 信源（一手）：ICSE 2026 research track CFP（conf.researchr.org/track/icse-2026/icse-2026-research-track，2025-06 抓取）
- 原文要点："high-quality submissions of technical research papers describing **original and unpublished results** of software engineering research"；多领域时作者须选择"brings the **maximum new insights**"的领域；正文 10 页上限；双周期投稿。
- 含义：系统描述本身不构成贡献——需要的是「原创且未发表的**结果**」+「新洞见」。工程系统论文被拒的常见理由即「仅是系统/工具，无研究问题与实证」。
- ICSE/FSE/ASE 技术轨历年接受率约 15–25%（各年份不同，精确数字以官网统计为准——本条为量级事实，非精确引用）。

### A2. 同类系统的"发表形态"先例
- FunSearch（DeepMind）：Romera-Paredes et al., "Mathematical discoveries from program search with large language models", **Nature** 2024（arXiv 2023 末），OpenAlex 引用 **2379**（截至抓取）。贡献形态：LLM+评估器的进化程序搜索框架 + **两个新数学/算法结果**（cap set 问题 n=8 的已知最大可容集合改进；在线装箱启发式改进）。→ 论文的"亮点"是**发现的结果**，不是系统本身。
- AlphaEvolve（DeepMind）：arXiv:2506.13131（2025-06，**白皮书，未经会议同行评审**），OpenAlex 引用 **1729**。一手摘要要点：进化式 coding agent（LLM 流水线 + 评估器反馈改代码）；已验证结果包括：4×4 复矩阵乘 48 次标量乘法（56 年来首次改进 Strassen 1969 该设置）、数据中心调度算法、电路化简、加速自身 LLM 训练。→ 注意：DeepMind 自己都**没有投会议**，而是 arXiv 白皮书 + 惊人结果组合。
- 两个先例共同点：**论文的说服力来自「进化出来的可验证新结果」，系统是手段不是主角**。

### A3. 竞争格局的当下事实（2025–2026）
- 信源（一手）：OpenEvolve 官方 README（github.com/codelion/openevolve，即 algorithmicsuperintelligence/openevolve，抓取于 2026 年初）
- 事实：OpenEvolve = 开源 AlphaEvolve，自称 "**The most advanced open-source evolutionary coding agent**"；架构 = **MAP-Elites + 岛群** + LLM 集成 + 人工反馈（artifact feedback）+ 种子化可复现；**7600+ GitHub stars、1200+ forks、月 PyPI 下载 1.7 万**；README 原文自称 "a **standard tool and baseline** for LLM-driven algorithm discovery"、"cited in hundreds of research papers"。
- README 列出的以 OpenEvolve 为 baseline 的新进化 agent（2025–2026）：SwarmEvolve、HarnessTTT、FrugalEvo、SEDIMA、EvoDuet、LoongFlow、AscendOptimizer 等——**这个子领域已进入「人人互相对比」的拥挤期**。
- 且已有系统性能研究用它："Let the Barbarians In: How AI Can Accelerate Systems Performance Research"（十个系统性能案例研究）。
- 含义：任何新系统论文的**默认对照物**已存在；「MAP-Elites + LLM 代码进化」本身已是商品化组合，不构成原创点。

## B. 「现状」端的事实——superopt 代码原文（一手信源，super/superopt/）

### B1. 系统机制清单（20 个模块，stdlib-only Python ≥3.9 + Go supervisor）
- 四状态循环：adjust（改权重/扩格）→ 加权随机采样（分殖优先）→ builder → coding → debug（≤M 次）→ CascadeGate 框架终验（README + loop.py 文档串）
- 级联门（gates.py）：stage1 环境健康 → stage2 正确性 → stage3 性能+软门禁；**门先证红**（启动时 exploit 全拒 + 金样例全过才放行）；框架零阈值假设（阈值全在 env 侧）
- 事件溯源（events.py/db.py）：sqlite 五表、事件信封 {ts,seq,type,source,payload}、19 类事件、WAL、唯一写者、崩溃游标恢复；「事件先于档案，档案是投影」
- 外置裁决（adjudicator.py）：提案-裁决分离；硬动作（删格/改格/改门）须**双回归证据**（elite_replay + exploit_regression）；human-in-loop（文件驱动）
- epoch 复评（loop.py）：每 10 迭代 pending 重评 M 次，多数过→保留/**复活**（weight 0→1），多数挂→**失效出档**（格保留）
- 档案（archive.py）：MAP-Elites 同格替换、history 上限 20、**延迟入库**（新格首解/容差带内/来源格降权三判定）
- 检索/认知注入（env_templates/matmul）：research.py 固化角色绑定检索三元组（方法/信源/关注点），对抗组内变异局部最优
- 工程完备度：22 个测试文件 + e2e + mock_pi + run_all.sh（tests/ 目录实查）

### B2. 现状中「不存在」的东西（与论文期望的落差核心）
- **仓库内无真实运行结果**：无生产 super.db、无 archive 实档、无任何 benchmark 数字（实查：全仓库仅 env_templates/matmul 与 env_v2 两个模板 + 测试）
- **无任何基线对比数据**：README/文档中未出现 FunSearch/AlphaEvolve/OpenEvolve 对照
- 未声明与任何近邻工作的机制差异表

## C. 近邻学术地图（OpenAlex API 实查，标题|年份|引用数）

### C1. LLM×进化×代码（superopt 的主邻域）
- Mathematical discoveries from program search with LLMs（FunSearch）| 2023/2024 | 2379
- AlphaEvolve: A coding agent for scientific and algorithmic discovery | 2025 | 1729
- Language Model Crossover: Variation through Few-Shot Prompting | 2024 | 244
- Evolving code with a large language model（EvoPrompting, ICML 2024）| 2024 | 131
- Large Language Models As Evolution Strategies | 2024 | 127
- Quality-Diversity through AI Feedback（QDAIF, ICML 2024）| 2024 | 105
- AVO: Agentic Variation Operators for Autonomous Evolutionary Search | 2026 | 47

### C2. 噪声/不确定域的 QD（epoch 复评/复活/失效的直接邻居）
- MAP-Elites for noisy domains by adaptive sampling | 2019 | 39
- Uncertain Quality-Diversity: Evaluation Methodology and New Methods | 2023 | 40
- Extract-QD Framework: A Generic Approach for QD in Noisy/Stochastic/Uncertain Domains | 2025 | 7
- MAP-Elites（源头：Illuminating search spaces by mapping elites, Mouret & Clune）| 2015 | 1348

### C3. 环境/课程共进化（「进化 agent 环境」主张的邻居）
- WebRL: Training LLM Web Agents via Self-Evolving Online Curriculum RL | 2025 | 393
- GenEnv: Difficulty-Aligned Co-Evolution Between LLM Agents and Environment Simulators | 2025 | 63
- EnvGen: Generating and Adapting Environments via LLMs for Training Embodied Agents | 2024 | 76

## D. 张力点（事实 vs 用户原话）

1. 用户方向是「**系统→论文**：这套系统我想要出一个论文……寻找原创亮点」；但 A2/A3 的事实显示领域惯例是「**可陈述的差异 + 可对比的证据→论文**」：OpenEvolve 已是默认 baseline，2025–26 的新进化 agent 论文都以它为对照。
2. 用户说「中顶会，工程系统顶会」但未命名会场；A1 显示 ICSE 类研究轨要求「原创且未发表的**结果**」——纯系统描述在研究轨会被定性为工具论文。
3. 用户要「寻找原创想法」；C1/C2/C3 显示 superopt 的每个机制候选亮点都已有近邻：MAP-Elites+LLM（OpenEvolve/AlphaEvolve）、噪声域复评（noisy-QD 线）、环境进化（GenEnv/WebRL）——原创性只能以「与近邻的差分」存在，而 B2 显示当前没有任何支撑差分的运行数据。

## E. 发表场地图（OpenAlex primary_location 实查——同子域论文实际落点）

| 近邻工作 | 落点（OpenAlex 载体） |
|---|---|
| QDAIF（LLM 当变异算子+行为描述子） | ICLR 2024 |
| EvoPrompting（代码进化） | Genetic Programming and Evolvable Machines（期刊，2024） |
| MAP-Elites for noisy domains（自适应采样） | GECCO Companion 2019 |
| Uncertain Quality-Diversity（评估方法论） | IEEE Trans. on Evolutionary Computation 2023 |
| Extract-QD Framework（噪声/不确定域通用 QD） | GECCO 2025 |
| FunSearch | Nature 2024 |
| AlphaEvolve | arXiv 白皮书 2025（未走会议评审） |

- 含义：这个子域没有唯一「顶会」——ICSE/FSE/ASE（AI4SE）、GECCO/TEVC/GPEM（进化计算）、ICLR/ICML（ML）对「原创」的定义各不相同；会场未定之前，「原创亮点」无法定义。
- 差异示例（供用户自行判断，非结论）：superopt 的 epoch 复评/复活/失效 ≈ noisy-QD 线（2019 GECCO Companion / 2023 TEVC / 2025 GECCO）的直接邻居；「进化 agent 环境」主张 ≈ GenEnv/WebRL 邻居；外置裁决双回归证据/门先证红在近邻清单中未见同名机制——但「未见同名」≠「原创成立」，需差分论证+实验。

## F. 信源清单
- 一手：super/superopt/*.py（20 模块文档串）、super/README.md、super/env_templates/matmul/README.md、super/tests/ 目录清单
- 一手：ICSE 2026 research track CFP（conf.researchr.org，抓取 2026 年初）
- 一手：AlphaEvolve arXiv:2506.13131 摘要；OpenEvolve 官方 README（github）
- 二手（元数据）：OpenAlex API（标题/年份/引用数/载体，抓取于 2026 年初；引用数为时点值）
