# research-L1b 简报：human-loop 升级的业界方法论基准

> 阶段：grilling L1b（job_44 human-loop 线）｜三子域：深度思考 loop / human 裁决 / RFC 生成｜时点：2026-10-05｜方式：3 叶子外包（researcher）+ 本人直接检索交叉验证

## 事实性结论

**子域1：LLM 深度思考/决策 loop 改进模式**

1. **无外部反馈的内在自我修正会降低推理质量**（强：论文×2，广泛复认）——Huang et al. ICLR 2024（arXiv:2310.01798）：无 ground truth/工具/环境信号时 intrinsic self-correction 会把对的改错；Reflexion（Shinn 2023，arXiv:2303.11366）有效前提恰是环境反馈转语言反馈（HotpotQA 评测有 label leak）。⟹ S-R/EC 纯 LLM 自我批判须以 research 事实或 human 反馈兜底。
2. **Multi-agent debate（MAD）被多篇等算力复现反驳**（强：原始单次+多篇独立反驳）——原始 Du et al.（arXiv:2305.14325，ICML 2024）：3 agent×2 轮，ChatGPT-3.5 六任务 +5~16pp（算术 67→82、MMLU 64→71），但单一弱模型/小样本/无显著性检验；Smit et al. ICML 2024（arXiv:2311.17371）：等算力下多数投票与 MAD 相当或更好且更便宜；arXiv:2502.08788（5 法×9 基准×4 底座）未稳定超标准 prompting；arXiv:2505.22960：等生成预算下数学任务无稳定优势（仅极难题/弱模型占优）；NeurIPS 2025 Debate-or-Vote：投票≥辩论（鞅论证：辩论不改变期望正确率）。⟹ N1/N2 多角色矛盾须对照「多采样+投票」基线。
3. **采样+投票（self-consistency）是最可靠增益**（强：多次复现）——Wang et al. ICLR 2023（arXiv:2203.11171）：GSM8K +17.9pp；More Agents Is All You Need（arXiv:2402.05120）：性能随 agent 数扩展、增益与任务难度相关、与其他方法正交。⟹ E 阶段视角多样性+聚合是低风险升级点。
4. **Tree of Thoughts 搜索型任务收益大、通用性存疑**（中：单次+部分复现）——Yao et al. NeurIPS 2023（arXiv:2305.10601）：Game of 24 GPT-4 CoT 4%→ToT 74%，社区复现 74%→69%；收益集中于需搜索/规划的任务。
5. **单模型多角色（multi-persona/SoM）有条件有效**（中：正效应限强模型+调优）——原始 MAD 即同模型 3 实例；SPP（NAACL 2024，arXiv:2307.05300）：GPT-4 动态 persona +7~18%，但 GPT-3.5/Llama2 无效且任务自建；Liang et al.（arXiv:2305.19118）：过度顺从致「思想退化」，需说服者角色干预；Smit：multi-persona 未调优时低于强基线约 7%。⟹ E 阶段多视角需对抗/说服压力与独立性设计，防视角趋同。
6. **LLM-as-judge 三大偏差有实证数值**（强：两次独立验证）——Zheng et al. NeurIPS 2023（arXiv:2306.05685）：位置/长度偏好/自我增强偏差，GPT-4 judge 与人一致性约 85%（chat 域），缓解靠换序+多 judge；Panickssery et al. NeurIPS 2024（arXiv:2404.13076）：LLM evaluator 偏好自己输出（self-recognition 驱动 self-preference）。⟹ 同底座互评需换位/独立模型/结构化 rubric。

**子域2：human 判断质量增强**

7. **Cognitive forcing functions 可靠降低 over-reliance**（强：多次复现+单次边界）——Buçinca et al. CSCW 2021（arXiv:2102.09692）：人对 AI 建议过度接受（错的也接受）；仅附加解释不降 over-reliance 甚至加重；「先自判再看 AI/延迟展示」显著降低（代价：体验变差）；促使深度参与时解释可降低（Vasconcelos et al. CSCW 2023，单次）。⟹ human 判断点先独立表态再呈现 AI 简报，抑制 rubber-stamping。
8. **对比/排序反馈优于绝对评分（工业标准实践）**（强）——Christiano et al. 2017（arXiv:1706.03741）：对比判断比绝对打分对人更易且信息不损；InstructGPT（Ouyang et al. 2022）：标注员排序 4–9 条输出，信息效率高于二元选择；CAI（Bai et al. 2022）：AI 批量生成批评/对比标签、人只维护原则清单。注意：pairwise 在 LLM-judge 协议上更易受风格/冗长操纵（翻转约 35% vs 绝对分 9%，arXiv:2504.14716）→ 顺序随机+rubric。⟹ 裁决点并排对比+差异标注呈现，而非逐一打分。
9. **长会话判断衰减：实验室稳健、专家自定节奏场景有反驳；自满效应更稳**（混合证据）——vigilance decrement（Mackworth 1948；Warm 2008 综述）高复现但限于强制持续监控：150 万乳腺癌判读自定节奏无衰减（Suleman et al. PNAS 2024）、900 万条 AMT 标注不随月份衰减（Difallah et al. 2016）；automation complacency（Parasuraman & Manzey 2010，Human Factors）高复现：系统越可靠人越松懈；注入真实错误可恢复审阅警觉（Fang et al. 2025，12 周随机实验，单次）。⟹ rick 的 human 裁决属专家自定节奏场景，主要风险是 complacency 而非疲劳——定期注入「应否决」判断点防松懈。

**子域3：RFC/结构化文档生成质量**

10. **outline-then-fill + 验证式 refinement 主流有效，但证据域有限**（中：多次复现、限故事/维基域）——Plan-And-Write（Yao et al. AAAI 2019，arXiv:1811.05701）：先规划后写作优于端到端；Re3/DOC 复现「显著更连贯」；SoT（ICLR 2024）提速 1.9–2.4x 但质量按任务分化（数学/受限写作退化）；Self-Refine（NeurIPS 2023）报告 7 任务 +5–40%，但推理域内在自我修正被结论1反驳；CoVe（ACL 2024）验证式修订降幻觉。缺 RFC/设计文档域对照实验。⟹ RFC 生成 outline→分节填充→验证式修订。
11. **rubric 化评估是工程界标准做法**（强：行业文档级，广泛复用）——OpenAI evals/graders 官方指南：把质量标准拆成明确 criteria/rubric 交模型评分（可多 grader 聚合）比自由打分更稳定；Anthropic cookbook 同类实践。⟹ KR2 的 RFC 质量对比基线可用 rubric 化 LLM 评估+human 抽检。
12. **claim 级引用提升可验证性、但引用质量本身不稳；先取原文再作答收益大**（强：多次复现）——生成式搜索引擎仅 51.5% 句子被引用完全支持、引用 precision 74.5%（Liu et al. EMNLP-F 2023）；ALCE（Gao et al. EMNLP 2023，arXiv:2305.14627）：最佳模型约 50% 生成缺完整引用；claim 级/交错引用优于段落级（RARR ACL 2023；ReClaim NAACL 2025）；Anthropic context retrieval：先抽引文再作答使长文检索 27%→98%（arXiv:2312.11193，复合任务增益衰减）。⟹ RFC 证据链用 claim 级引用+「先证据后结论」生成顺序+引用抽验。

## R7 上报项

1. 置信度/不确定性标注对「文档可用性/human 判断质量」无直接实证：现有证据仅 QA/对话域单次实验（FAccT 2024，N=404 预注册：第一人称不确定表达降 over-reliance；CHI 2024，N=594：权威语气比免责声明更抬升信任）——rick 引入置信度标注需自行 A/B 验证。
2. outline-then-fill/refinement 的证据集中于故事/维基域小规模人工评测，RFC/设计文档域无对照实验——升级效果需自建 rubric 评估。
3. pairwise 风格操纵数值（35% vs 9%）来自 LLM-judge 协议实验；human 裁决场景的等价数值缺失，需自行小样本验证。
4. rick 长会话中 human 判断是否衰减不确定（专家自定节奏场景有 PNAS 2024 反驳）；complacency 风险更确定，但「应否决」判断点的注入频率无数据。
5. Bhatt et al.「Human Attention in AI-Assisted Decision Making」原文未能定位（本人与 leaf-2 均未找到）——AI 监督注意力方向缺一手来源。
6. Du et al. 2023 常被误引为 arXiv:2305.12036，正确编号为 2305.14325——本次任务给定编号有误，已修正，后续引用需注意。
7. 「MAD 仅在极难题/弱模型占优」（arXiv:2505.22960）与 rick S-R 辩证逆转（极难矛盾场景）的条件性交互无直接证据。

## 叶子文件（外包产出，均在 .rick/jobs/job_44/doing/grilling/）

- research-L1b-leaf-1.md：MAD 原始数值/多角色架构/复现谱系（首派 15min 超时未落盘→按协议重派收窄版完成；结论 2/5 主要素材，编号修正出自此叶）
- research-L1b-leaf-2.md：human 判断质量（完成；结论 7-9 主要素材；本人已校正其 2 处过度引申：2504.14716 适用域、Vasconcelos 定性）
- research-L1b-leaf-3.md：文档生成质量（完成；结论 10-12 主要素材）
