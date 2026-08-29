# research-E 简报 — E 阶段视角候选（loop_7 第 1 轮）

> 阶段：E 视角生成 | 主题：为「rick=human-AI 双循环进化系统 + runtime 选择已决结构」提供跨领域观察透镜 | 基准：2026-08-26 | 方法：4 联网验证叶子（leaf-1 生物学/leaf-2 经济学与预测学/leaf-3 控制论与认知神经科学/leaf-4 设计理论与免疫学）+ parent 映射综合 | 协议：原创性思考=跨领域学习=形成偏见；视角形成=偏见形成=原创
> 硬约束：仅理论忠实转述+映射关系；映射=启发非断言；不给推荐倾向；理论表述忠于原始文献（来源标注于各条）

## ① 视角候选列表（9 个，按映射强度降序；映射锚点均指 judgment.md 已决事项）

### P1 凯利公式 × 校准预测学 × 有效市场悖论（经济学/预测学）→ 度量体系
- 来源：Kelly《A New Interpretation of Information Rate》(Bell System Technical Journal 35(4), 1956)；Brier(1950) 评分+Murphy(1973, J. Appl. Meteor.) 三项分解；Tetlock & Gardner《Superforecasting》(Crown, 2015)；Grossman & Stiglitz(AER 70(3), 1980)。
- 映射：human 拍板的三维向量与博彩三要素同构——正确率=校准概率 p（Brier/Murphy 分解为可靠性/分辨力/不确定度，对应 Y4 按周/月时间窗展开的中位数/max/均值统计值）；赔率=反共识支付比；影响力≈仓位。Kelly 定理：期望为负时任何下注比例长期破产——对应 r3「判断做错价值必为负」（正确性是乘性前提）。G-S 悖论：完全有效的市场使信息获取无利可图——映射 Y5（反共识 q=采样网络主流观点=读取市场价格信号）与 Q4 倾向的内在逻辑（正确且高影响但低赔率=已被定价的共识=无信息租金）。
- 融贯：自洽=三要素在 Kelly/Brier 框架内互相独立可微（对应 X4「独立优化」）；他洽=与 N4 5-job 小样本窗口、N2 subagent 群体概率+human 审核兼容；续洽=度量体系 runtime 无关，层 B 后公式不失效，fork/自研决策本身可用同框架评估赔率。

### P2 互补学习系统 × 睡眠重放（认知神经科学）→ 双循环结构
- 来源：McClelland/McNaughton/O'Reilly《Why There Are Complementary Learning Systems in the Hippocampus and Neocortex》(Psychological Review 102(3), 1995)；CLS 2.0：Kumaran/Hassabis/McClelland(Trends in Cognitive Sciences 20(7), 2016)；海马重放：Wilson & McNaughton(Science 265, 1994)。
- 映射：CLS 核心=快速情景系统（海马：单次编码、易被覆盖）与慢速统计系统（新皮层：交错巩固、抗干扰）必须互补，否则灾难性干扰。同构：draft/loops 逐 job 情景记录=海马；dream（X5 拍板：遍历读取 loop 判断条目→计算复盘→待 job 时机回收）=睡眠期重放→巩固；.rick/skills/ 已沉淀 10+ skill 文件（本地已证）=新皮层语义记忆；5-job 窗口（N4）=交错采样小批量（CLS 防干扰机制本身）。CLS 2.0 补充：重放不止巩固、亦对经验加权以服务规划；与既有结构一致的信息可被新皮层快速吸收——映射 dream 对与既有 skill 一致的判断可快速回收、不一致者走交错巩固。
- 融贯：自洽=速率差是理论内核而非附加；他洽=与 Q2（性能=human 主观稀疏指标，慢系统产出 skill 而非分数）兼容；续洽=预测层 B 后双循环节奏不变、巩固窗口需重校准（X6 实验可检验），并预测 skill 渐进更新优于一次性重写。

### P3 生态位构建 × 共生谱系（进化生物学）→ runtime 选择
- 来源：Odling-Smee/Laland/Feldman《Niche Construction: The Neglected Process in Evolution》(Princeton UP, 2003)；前驱 Lewontin(1983)；共生谱系（de Bary 1879 Symbiose；mutualism/commensalism 术语 van Beneden 1875/76）；Margulis 内共生学说(1970, Yale UP)。
- 映射：pi=宿主（agent loop=稳定代谢；loop_7 源码考古已列 12 钩子全表）；rick=共生体，经扩展/skill 持续改造宿主使用环境=生态位构建——「短期 pi 扩展面自改进」路线的理论名称。事实约束：宿主不按共生体需求进化（pi 社区不会为 rick 改内核 loop）→ 层 B（12 钩子外骨架替换）=生态位构建触及宿主基因型边界，触发全池重评=换宿主事件；Y1 唯一撤销条件（pi 扩展无法满足功能需求）=生态位构建的极限信号。共生谱系词表：pi 第三方生态=开放栖息地；dsh 插件致系统性崩溃（human 问句推测）=向寄生漂移的风险特征。
- 融贯：自洽=生态位构建处理「生物改变环境」的正反馈无内部矛盾；他洽=与 Y1 精确对应；续洽=预测宿主更换后共生体适应重启，fork=「宿主基因组可写」中间态。

### P4 设计规则与模块化期权（设计理论）→ 自改进哲学 + runtime
- 来源：Parnas(CACM 15(12), 1972) 信息隐藏；Baldwin & Clark《Design Rules, Vol.1》(MIT Press, 2000) 可见设计规则 vs 隐藏模块、模块化=实物期权价值（金融期权理论框架）；Conway(Datamation 14(4), 1968)。
- 映射：Baldwin & Clark 区分=架构（可见设计规则）与模块（隐藏部分）：规则冻结后模块可并行实验互不干扰，模块化使实验成为廉价期权，而改规则本身罕见且昂贵。映射：pi 固定 loop+12 钩子=可见设计规则（r2 拍板「loop 本身稳定带来什么」的理论答案=规则稳定性是模块实验的前提）；rick 扩展/skill=隐藏模块（「只关注自己的扩展包」=Parnas 信息隐藏收益）；层 B=设计规则变更事件——理论预测其成本远高于模块变更，应作稀缺事件（与「出现改 loop 需求再权衡」一致）；dsh 全 loop 可定制=把设计规则也降为模块，规则稳定性随之丧失（对应用户担心的插件耦合致系统性混乱）。
- 融贯：自洽；他洽=与 r2 已决「pi/dsh 唯一结构差异=loop」对应：差异恰在设计规则层而非模块层；续洽=层 B 时 fork/自研应按「新设计规则集」定价而非按模块计。

### P5 交易成本经济学（制度经济学）→ runtime 选择（make-vs-buy）
- 来源：Coase《The Nature of the Firm》(Economica, 1937)；Williamson《The Economic Institutions of Capitalism》(Free Press, 1985)：资产专用性/套牢/根本性转化。
- 映射：Coase 边界=市场交易成本 vs 内部组织成本之比较。映射：复用开源生态=市场采购；fork/自研=内部化。Y1 human 表述「组件复用假设可能不成立…定制成本很低则完全可以自己生成扩展」=生成成本下降使内部化边界外移；Y3「验证成本可控（基于已存在扩展语义级对齐）」=验证技术压低交易成本。资产专用性映射：rick 对 12 钩子接口的专用性投资越深，换 runtime 套牢成本越高——层 B 触发=专用性超出市场接口承载力。
- 融贯：自洽；他洽=与 dsh 观望兼容（无明显优势时切换成本本身即交易成本）；续洽=预测层 B 后选择由「AI 生成成本 vs 验证成本」相对价格决定，验证技术进步则自研倾向增强（human 已有此直觉，理论给出成立条件）。

### P6 红皇后 × 适应度景观 × 基因型-表型（群体遗传学）→ 层 B 触发 + spec/runtime 解耦
- 来源：Van Valen(Evolutionary Theory 1(1), 1973) 红皇后；Wright(1932/1970) 适应度景观与 shifting balance 三阶段；Johannsen(Am. Nat. 45, 1911) 基因型/表型术语；Waddington(Nature 150, 1942) 渐成 canalization。
- 映射：红皇后=环境因竞争者进化而持续退化，须不停奔跑才能留在原地——映射 benchmark 生态（本 loop 已证 Verified 2026-02 停用、Frontier-Bench 2026-07 发布）与模型迭代下的 harness：双循环是持续适应机制而非一次性达成。Wright：pi 扩展面=当前局部峰；层 B=峰移（须经谷=fork 期性能下降）；shifting balance 三阶段映射「全池重评→实验→定型」。基因型-表型：spec/语义契约=基因型（稳定可遗传描述）；runtime 实现=表型；r1 Q3 拍板「所有 spec 可通过 runtime 单测器穷举描述」=发育稳态（canalization：基因型→表型映射对扰动不敏感）；度量体系=选择压力（决定 dream 巩固哪些行为）。
- 融贯：自洽；他洽=与 r2「放弃完备 spec、有限穷举+语义契约」兼容（有限穷举=基因型的可行测量）；续洽=runtime 切换验证=表型一致性检验（同 spec 两 runtime 单测穷举输出等价）；层 B 后度量须保持连续否则进化重启。

### P7 二阶控制论（控制论）→ 双循环定位（human 在系统内）
- 来源：von Foerster《Cybernetics of Cybernetics》(1974)/《Observing Systems》(1981)。
- 映射：二阶控制论核心=观察者是所观察系统的组成部分，不存在无观察者的客观性。映射 Q5 拍板「不存在一种 benchmark…benchmark=human+agent…个性化的评价体系」与 Q2「人作为整个系统的核心」：human 非外部评测者而是构成性观察者→完全外置的客观 benchmark 是范畴错误；human-loop 判断记录=观察轨迹显式化；dream 给 human 判断力打分=系统对观察者的二阶观察（控制论的控制论）。
- 融贯：自洽；他洽=与 Y8「纯观察即可」兼容（观察即最小干预）；续洽=预测层 B 后任何 runtime 验收都无法完全外置，human 判断仍是终审环节。

### P8 免疫危险理论 × 克隆选择（免疫学）→ 自改进哲学（隔离性）
- 来源：Burnet《The Clonal Selection Theory of Acquired Immunity》(Vanderbilt UP, 1959)；Matzinger《Tolerance, Danger, and the Extended Family》(Annu. Rev. Immunol. 12, 1994)。
- 映射：克隆选择=针对自身抗原的反应性克隆（Burnet 原文称「禁忌克隆」）在免疫成熟前被清除或抑制（后世称阴性选择），自我/非我识别失败=自身免疫病。映射 r3 拍板「牺牲隔离性的自改进不可取…越来越不可控则无法可持续自改进」：隔离性=自我/非我识别能力；pi 扩展面=免疫隔离边界；差分验证/语义级对齐（Y3）=危险信号检测（Matzinger：应答由危险信号触发而非非我本身——共生微生物/胎儿皆非我却不应答；对应「不禁止一切改动，只拦截有害改动」）。
- 融贯：自洽；他洽=与 dsh 排除理由同构（失隔离→自身免疫式失控；dsh 崩溃为 human 推测，见未消解项）；续洽=层 B 自研 runtime 须先内建验证体系（免疫系统先于扩展存在）。

### P9 必要多样性定律（控制论）→ 12 钩子定制面
- 来源：Ashby《An Introduction to Cybernetics》(1956, 第 11 章 requisite variety)。
- 映射：定律=只有多样性才能吸收多样性；控制器多样性须不低于扰动多样性，不足则需衰减/放大。映射：12 钩子=rick 控制器的多样性预算（loop_7 源码考古已列全表）；dsh 全 loop 可定制=更高多样性上限但多样性本身有协调成本——与 r2 追问「loop 可定制的好处与代价」同构；skill 机制=多样性放大器；steering=实时衰减器。
- 融贯：自洽；他洽=与「短期无改 loop 需求」拍板兼容（当前扰动在预算内）；续洽=层 B 触发判据可表述为「扰动多样性超出钩子预算」，但定律仅给必要性不给定量阈值。

## ② 覆盖矩阵（●强 ◐中 ○弱；列=judgment.md 已决事项；末列=续洽/层 B 后预测力）

| 候选 | runtime 选择 | 双循环 | 度量体系 | 自改进哲学 | 层B后续洽 |
|---|---|---|---|---|---|
| P1 Kelly×校准×G-S | ○ | ◐ | ● | ○ | ◐ |
| P2 CLS×重放 | ○ | ● | ◐ | ◐ | ◐ |
| P3 生态位×共生 | ● | ◐ | ○ | ◐ | ● |
| P4 设计规则×模块化 | ● | ○ | ○ | ● | ● |
| P5 交易成本 | ● | ○ | ○ | ◐ | ● |
| P6 红皇后×景观×基因表型 | ● | ◐ | ◐ | ○ | ● |
| P7 二阶控制论 | ◐ | ● | ◐ | ○ | ◐ |
| P8 免疫×危险理论 | ◐ | ○ | ○ | ● | ◐ |
| P9 Ashby 必要多样性 | ◐ | ○ | ◐ | ◐ | ◐ |

注：无单一候选覆盖全部四事项；四事项各有 ≥2 个 ● 级候选。排序依据=映射密度与事实支撑强度，非视角优劣判断。

## ③ 未消解项（仅上报，不替 human 决策）

1. 「赔率」语义未定：human 语境=反共识独特性（Q4/r2），与博彩赔率（隐含概率倒数）方向相反；P1 按「高赔率=低共识概率」成立，若 human 意指「独特性计数」则 P1 映射强度降级，需 human 澄清。
2. dsh「插件错误致系统性崩溃」在 judgment 中为 human 问句推测，无实测数据；P8 免疫映射为词表级启发，不可证伪性未消解。
3. Ashby 定律无定量形式：12 钩子「多样性预算是否够用」缺少扰动空间的测量定义（定律仅给必要性）。
4. 基因型-表型映射验证缺口：spec 单测穷举=发育稳态目前是规范性目标；语义契约无实现证据（r2 已决方向=有限穷举）。
5. CLS 交错学习预测（5-job 窗口≈防灾难性干扰采样）未经实验；X6 校准实验设计可覆盖。
6. 文献核验完成度：四叶子已全部落盘；残留不确定均为低风险术语/记法级——Kelly 原文用 α 记法（f*=(bp−q)/b 为标准换算）、von Foerster 名句常见措辞系 von Glasersfeld 转述、Burnet 原文用「禁忌克隆」非「阴性选择」、Conway 期号存 14(4)/14(5) 记载分歧、Baldwin & Clark 期权论证宜称「金融/实物期权框架」（详见各 leaf 文件 Gaps）。逐条置信度约 0.8-0.9。

## ④ 叶子文件路径（本轮外包产出）

- research-E-leaf-1.md — 生物学/进化论（P3/P6 来源核验）
- research-E-leaf-2.md — 经济学/预测学（P1/P5 来源核验）
- research-E-leaf-3.md — 控制论/认知神经科学（P2/P7/P9 来源核验）
- research-E-leaf-4.md — 设计理论/免疫学（P4/P8 来源核验）
