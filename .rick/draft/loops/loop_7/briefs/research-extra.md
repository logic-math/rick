# research-extra 简报 — 工程能力测评基线尽调

> 阶段：S 补充调研（loop_7 第 2 轮）| 主题：最具代表性/最难的工程测评基线 + 如何验证 rick 工程能力 | 时间基准：2026-08-24 | 方法：web_search 多角度 × 4 联网叶子（leaf-1 SWE 家族 / leaf-2 其他 agentic benchmark / leaf-3 Frontier-Bench+共识 / leaf-4 harness 级+HITL 评测）+ 本地代码事实
> 硬约束：仅事实+前提+来源；「已验证」= 官方页面/arXiv/本地代码；「未消解」= R7。不给推荐倾向。

## ① benchmark 全景表（任务形态 × 规模 × SOTA × 接受度）

| benchmark | 定位/任务形态 | 规模 | 当前 SOTA(2026) | 生态接受度 |
|---|---|---|---|---|
| SWE-bench(Full) | 真实 GitHub issue→补丁，pass@1 测试补丁判定 | 2,294 issue(Python) | 已饱和，不再区分前沿模型 | 最被引用 SWE 基准（OpenAI 2024-08 起多方采用） |
| SWE-bench Verified | Full 的人工筛选子集（与 OpenAI 合作构建，2024-08-13） | 500 | Claude Opus 5 ≈96%（BenchLM 2026-08） | 曾是前沿发布标准指标；**OpenAI 2026-02 停用**，称污染+设计缺陷 |
| SWE-bench Lite | 快速迭代子集 | 300 | SWE-agent 2025-03 达 65%（当时开源 SOTA） | 论文/消融常用 |
| SWE-bench Multimodal | JS/TS 仓库+图像资产 issue | 517 test/102 dev（17 仓库） | — | 学术扩展 |
| SWE-bench Multilingual | 9 语言 42 仓库 | 300 | — | 学术扩展 |
| SWE-Bench Pro（Scale AI） | long-horizon 企业级多语言 issue，抗污染设计 | 1,865 / 41 仓库（公开 731） | 论文期公开集 GPT-5 23.3%（2025-09）；2026 聚合站 80.3-84.8% 口径不一 | **OpenAI 2026-07 审计 ~30% broken 并撤回推荐**；Artificial Analysis 2026-06 因可作弊移除 |
| Terminal-Bench 2.1（Harbor 团队） | Ubuntu 容器内终端 agent 长程任务（真实工作流启发） | 89 任务（2.1 修正 26 个；v1≈100） | 榜首 ~88% 档（Kimi K3 88.3 / GLM-5.3 88.2，快照口径不一） | 公认测"终端操作/工具恢复"，测 harness 至少同测模型；Anthropic/Cursor/Codex CLI 均报 TB2.0 分 |
| Frontier-Bench v0.1（Harbor/Laude） | TB 继任者（原 Terminal-Bench 3.0 更名）：7 专业领域 74 任务，mean reward | 74 | 发布时最佳 ~34%；Claude Opus 5 自报 43.3%（mini-SWE-agent+GKE，5 次均值）；官方榜 GPT-5.6 Sol 34.4% | 2026-07-23 发布，当前最硬代理基线候选；Opus 5 行为自报无独立复现 |
| LiveCodeBench | 竞赛题（LeetCode/AtCoder/CodeForces）持续更新、抗污染，含自修复/代码执行/测试输出预测 | 400+ | ~89-93.5%（口径未核） | 学术+模型报告常用 |
| RepoBench | 仓库级代码补全（多文件，非 agentic） | Python+Java | — | 学术（ICLR 2024） |
| GAIA（Meta 2023） | 通用助手真实问答（推理/多模态/网页/工具） | 450+ 三难度级 | 人 92% vs GPT-4+插件 15%（原始论文）；2026 SOTA 依赖脚手架（Sonnet 4.5≈74.6% HAL） | ICLR 2024，助手类参考 |
| τ-bench（Sierra） | 模拟用户对话+领域 API 工具+策略指南，比对数据库终态 | Airline/Retail（τ²/τ³ 加 banking/telecom/voice） | Retail 84.7% / Airline 56%；Gemini 3 Pro 90.7%（2026-04，版本未核） | 工具-agent-用户交互基准代表 |
| AgentBench（THUDM, ICLR'24） | 8 环境通用 agent（OS 终端/DB/KG/WebShop 等），非工程专项 | 29 LLM | 已过时 | 通用 agent 参考 |
| ML-Bench（Yale gersteinlab） | 仓库级 ML 任务，LLM-Bench+AGENT-Bench 双设置 | 9,641 例/169 任务/18 仓库 | 无活跃榜单 | 学术；OpenHands 仅集成其评测，非"原 ML-Bench" |
| Aider polyglot | Exercism 225 题（6 语言），真实编辑循环内评分 | 225 | 榜首 ~60%+（口径未核） | 编辑型基准，非终端 agentic |
| H-HEval | 身份无法核实（慎与 IHEval/HD-Eval 混淆） | — | — | R7 上报 |

## ② 「最具代表性/最难」主观共识（2026-08 时点）

- **SWE-bench Verified 已实质退役**（多方验证）：OpenAI 2026-02 宣布因污染+设计缺陷不再报告 Verified（"no longer measures frontier coding capabilities"），改推荐 Pro；2026-07-08 审计 Pro 公开集 731 任务约 30% broken（自动管线 27.4%、5 名工程师人工 34.1%），撤回推荐；Artificial Analysis 2026-06 因可作弊将 Pro 移出 Coding Agent Index。[openai.com 两篇原文(经镜像核实)；the-decoder 2026-07-09；leaf-1]
- **无唯一共识，争论仍在**：backgrind 2026 综述称仅 Frontier-Bench v0.1 与 ARC-AGI-3 仍有区分度（Verified 退役、TB2.1 饱和于 74-84% 带）；ARC-AGI-3（2026-03，交互式，人类 100% 可解 vs 前沿模型发布时 <1%、Opus 5 至 2026-07 仅 30.16%）区分度最强但非纯工程；DeepSWE（113 原创任务/91 仓库/5 语言，判官分歧 1.4% vs Pro 32.4%）被部分社区视为 Pro 抗污染继任者。[backgrind.com；ARC-AGI-3 技术报告；arXiv 2607.07946；leaf-3]
- **定位差异（非难度阶梯）**：社区共识 SWE-bench 测"补丁能力"（issue→patch，pass@1 测试判定），Terminal-Bench/Frontier-Bench 测"终端长程操作+工具恢复"，是两种不同技能；TB 系测 harness 至少同测模型。[dreaming.press；whatllm.org；leaf-2 核实]
- **测"最难"的另一条线**：METR time horizon（人类完成时长标注→logistic 拟合 50% 成功率时间线），TH1.1 套件 228 任务，官方承认近饱和（最新模型几乎全能完成）。[metr.org TH1.1/time-horizons；leaf-3]
- 综上：2026-08 时点"哪个最难/最代表真实工程"**社区仍在争论**；Frontier-Bench（顶分 ~34-43%）与 DeepSWE 是当前最硬候选，ARC-AGI-3 是难度上限参考（非工程）。

## ③ harness/编排级评测（2026 已有量化方法，非空白）

- **harness 方差 ≥ 模型方差（多篇独立论文，已验证）**：Scaffold Effect（arXiv 2607.22585，300 次试验）固定模型跨 harness 通过率差仅 0-8pp，但 tokens/已解任务差 40×，失败指纹与模型无关（Goose=REASON、OpenHands=VERIFY、OpenCode=idle-loop）；Binding Constraint Thesis（arXiv 2605.23950，ICML 2026）受控因子实验 HV=18.48 vs MV=2.37 pp²（7.8×），9 对模型比较 6 次排名反转，SWE-bench Verified Mini 跨 scaffold 差 34-48pp；Harness-Bench（arXiv 2605.27922，5,194 轨迹）同任务同模型池 harness 分差 23.8pp，效应集中于弱模型+工具密集/有状态工作流。
- **评测单元应改为 harness–model 对**：Position 论文（arXiv 2606.17799）指现有分数混淆模型/harness/环境三因素、单一参考解惩罚等价方案、无组件级信号难迭代；处方=harness-aware 元数据+多解形 verifier+组件级指标。必报指标：tokens/solved、no-action turns（"监督税"）、失败向量、恢复率、上下文保持、控制滞后。[leaf-4 各篇]
- **HITL/主观反馈客观化三条路线（均有量化证据）**：①成对偏好+Elo（Computer Agent Arena，ICLR 2026，2,201 票/1,058 用户，正确性为主驱动）；②开放反馈→指标诱导（AutoLibra，ICLR 2026，80 条轨迹即可，3 阶段成功率 +20%）；③模拟用户+rubric 自动评测（AgencyBench，ACL 2026，user-sim 绕开 HITL 扩展性瓶颈；LH-Bench，arXiv 2603.22744，专家 rubric 使 judge κ 0.46→0.60、结构化反馈恢复 70.3% 执行错误）。局限：均依赖 LLM 模拟/判分，真实人类验证规模小、人群偏差存在。[leaf-4]
- **人机参与度是可测设计变量**：HAS-Bench（arXiv 2607.04329，397 任务）5 级 agency×3 通道——A3 平等伙伴 vs 全自动 A1：Pass@1 +8.4、安全率 +26.9，可恢复 65.4% 自主失败；但 A3→A4 边际递减（救 27 破 13），弱模型 ≈0 增益；人类参与收益非单调。[leaf-4]
- **与 rick 的对应**：评测"方法/编排质量"已有成熟范式（固定模型对比 harness，测 tokens/solved、失败指纹、监督税、恢复率）；"监督税/控制滞后"恰是 rick human-loop 的目标变量；人类反馈客观化可用 AutoLibra 式指标诱导处理 draft 记录。

## ④ 对 rick 的落地可行性评估（本地代码事实 + leaf-4 方法论交叉）

前提：rick=引导程序（Go CLI + pi 执行后端），执行维度产出 loops/skills/domain（操作知识），价值维度 domain/draft；已有确定性验证机制：rick-gates hook（门禁/提交/结构校验，确定性脚本）、tdd-red-green-refactor-loop（go test 自评收敛）、testing-conventions（精确包级 go test）、tests/ 含 e2e_v2_test.py（mock_agent 四阶段端到端）+ integration_test.sh。[README.md; .rick/loops/; .rick/domain/testing-conventions.md; tests/]

- rick 能跑契约型/单测型评估：SWE-bench 系（go test/python test 判定 pass@1）、LiveCodeBench（单测判定）、RepoBench（补全比对）——判定机制与 rick 现有"自评命令+exit code 0"模式同构，可离线，成本=算力+环境准备。[前提：评估时 pi runtime 可用]
- 环境类评估可行但更重：Terminal-Bench/Frontier-Bench 需容器（Docker/Ubuntu VM），rick 的 plan/doing 在 workspace 内操作可复用，但需环境编排与隔离（TB 官方有容器化 runner）。[前提：评测方提供容器 runner]
- harness 级对比设计（社区 2026 方法论，见③）：固定模型 × 对比 harness（rick vs 裸 pi vs 其他）在同一任务集上跑——Scaffold Effect 论文已在 50 任务子集上实证此设计；rick 恰好是 harness（非模型），此设计可直接套用。
- **归因陷阱（AgencyBench，ACL 2026）**：模型在原生 scaffold 生态有"主场优势"——对比 rick vs 裸 pi 时必须固定模型+固定任务集+同 attempt 预算，否则差异不可归因于 harness。[leaf-4]
- HITL/主观客观化（rick 的 human-loop 核心关切）：AutoLibra（开放反馈→轨迹指标）、Computer Agent Arena（偏好→结构化反馈）、HAS-Bench（human 参与度可配置，A3 最优）——2026 已有方法把主观反馈转可测指标，但均为研究原型，落地成本高（需采集器+标注）。[leaf-4]
- 量化口径候选：①pass@1 解决率（SWE 类）；②任务完成率+time horizon（METR 式 logistic 拟合，需人类耗时标注）；③mean reward（Frontier-Bench）；④轨迹质量指标（harness 效应论文的组件级信号）。各口径测的东西不同：模型能力 vs 方法质量 vs runtime 差异，需分开。

## ⑤ 未消解项（R7）

1. **H-HEval 身份无法澄清**：leaf-2 三轮检索无权威来源，可能与其他基准混淆（IHEval/HD-Eval）。若 human 坚持要此基准，需另行深挖。
2. **OpenAI 官方页无法直接抓取**（openai.com DNS 不可达）：Verified 退役与 Pro 审计主张经搜索快照+镜像（aetos.ai/ai-tldr/the-decoder）交叉确认，非逐字核读原文；主张多源一致，但权威性打折。
3. **SOTA 数值口径混乱**：Pro 公开集 80.3%~84.8%（聚合站不一）、TB2.1 88.2 vs 88.3、τ-bench 90.7% 版本未核实、Frontier-Bench Opus 5=43.3% 系自报（无独立复现）——跨榜比较前必须先固定 harness/attempt 预算。
4. **"最难"共识不稳定**：格局快速演化（DeepSWE v1.1 于 2026-08-20 更新；Frontier-Bench 由 Terminal-Bench 3.0 更名致命名混用；NVIDIA AVO 专用 harness 已达 ARC-AGI-3 100%），2026 下半年区分度结论可能再变。
5. **跨论文 harness 不可直接比对**：Scaffold Effect 与 Harness-Bench 的 harness 清单互不重叠；n=50 时通过率差异多数不显著；Lite 规模 300 vs 534 文档冲突未最终确认（leaf-1 判为文档笔误）。

叶子产出（同目录）：research-extra-leaf-1.md（SWE 家族）/ leaf-2.md（非 SWE 基准）/ leaf-3.md（Frontier-Bench+共识）/ leaf-4.md（harness+HITL 评测）。
