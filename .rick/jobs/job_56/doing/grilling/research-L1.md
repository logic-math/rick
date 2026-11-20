# research-L1 — 5-subagent 串行流水线工程先例（架构决策依据）

> 2026-11-30 · 调研触发：job_56 grilling L1（adjust 重构为 5-subagent 流水线的架构裁决依据）
> 方法：web_search 3 轮（8+8+6+6 检索）+ 2 篇关键论文 fetch 核实 · 信源等级 A/B/C 见各条

---

## 事实性结论

### Q1 多步 LLM pipeline 的信息传递（JSON vs 自由文本 / 步骤数效应）

1. **JSON 传递的失败模式限定于小模型**：7B 本地模型在多角色流水线中 30-40% 时间生成畸形 JSON（缺括号/截断/夹带评论）→ 解析失败后原始文本级联传递 → GSM8K 从单次调用 75.0% 跌到 45.0%；**仅把 JSON 换成 plaintext（架构/角色/模型不变）恢复到 82.0%**。结论明确限定于本地小模型，未测大模型。（arXiv:2607.26922 "Two Calls Beat Five Agents"，fetch 核实，A 级）

2. **长链传输中 JSON 是漂移最小的信道**：6 跳中继 × 12 条原子事实实测——Json 6 跳仅丢 4.7 个原始点（自由文本丢 21.8、Triples 丢 20.7、约束化 NL 丢 14.5）；**强中继（DeepSeek-v4-flash）下所有格式 hop-6 recall ≥0.973，格式选择几乎无关**。核心表述：结构化=「忠实的、错误局部化的信道」，不是纠错码；弱模型的代价在首次编码步，长链收益在后续传输步。（arXiv:2607.09678，fetch 核实，A 级）

3. **压缩传递的信息瓶颈**：中间态压缩后交下游决策步，小遗漏会破坏严格的数值/类别约束（两 agent 中继实测）。（arXiv:2607.18265，A 级）

4. **「更多 agent ≠ 更好」有 A 级反证**：对多 agent 系统的对照研究——multi-agent 平均精度增益 -0.042、74% 对比不超基线（arXiv:2611.17610）；但投票式并行采样（Agent Forest）下性能确实随 agent 数上升、~15 达饱和（arXiv:2402.05120）。**分歧消解**：并行投票（更多=更好）与串行流水线（更多步=更多累积风险）是不同机制。串行链每步可靠性乘法效应：0.99/步 × 5 步 ≈ 95% 端到端；0.90/步 × 5 步 ≈ 59%（LatentEval 工程分析，B 级）。「3-5 步甜点区」无直接文献——**无强证据，按工程直觉**（见 Gaps）。

### Q2 answer-isolation（每步独立上下文）的证据

5. **CoVe 核心设计 = 独立上下文回答验证问题**：验证问题在与草稿隔离的上下文中回答（不看草稿），消除模型「同样自信地重申原答案」的 anchoring/reaffirmation bias，显著减幻觉；fully-factored（每问题独立处理）效果最好。（CoVe, ACL 2024 Findings，A 级）

6. **反面 trade-off 无系统性反证**：独立上下文的已知代价是丢失上下文线索（与结论 3 的信息瓶颈同向），但未检索到「isolation 导致质量系统性下降」的对照研究；CoVe / Diversity of Thought / 独立 judge 设计均正向。（多源交叉，B 级——负证据的缺失≠证明，见 Gaps）

### Q3 视角/角色多样性生成（预定义 vs LLM 动态生成）

7. **动态生成视角是活跃研究方向且优于固定 prompt**：视角集合级条件化研究把「selecting（从预定义库选）vs generating（LLM 动态生成）」列为正交设计维度（arXiv:2609.30492，B 级）；DoT：prompt 视角变化 + 自一致投票优于固定 prompt（arXiv:2310.07088，A 级）；LLM 动态生成多样视角的能力是成体系的研究问题（arXiv:2311.09799，B 级）。**未见「预定义固定角色列表 vs 动态生成」的直接对照胜负结论——无强证据**。

8. **persona 是双刃剑**：不准确定义的 persona 会拖累推理能力（arXiv:2408.08631，A 级）——**视角定义质量决定效果**，视角生成步的输出契约应约束视角的「可用性论证」。外部检索（RAG/knowledge injection）支撑 persona 生成有先例（arXiv:2504.00241，B 级）；「web search 在视角生成中的用法」无直接对比实验（见 Gaps）。

### Q4 中间步骤失败的降级策略

9. **工程共识：降级是节点契约的一部分**：NodeResult = value + degradations[]（降级与产出同签名返回，orchestrator 无 try/catch）；3-tier degradation（primary → fallback → canned）；safe default 针对管道**最危险**失败模式（而非最常见）保守设计；retry 归 harness 层对模型不可见。（saumyajain.info / agentfootprint.dev / joechirum.com / ACM Queue，均为工程实践，C-B 级）

10. **降级必须记账**：fallback/retry/salvage 事件须与能力分数一起逐 run 记录，否则评估把降级产物当真实能力（degradation accounting，SSRN，B 级）。「跳过该步用默认值继续 vs 整链重跑 vs 降级回单 agent」的直接对比：**无强证据，按工程直觉**——跳过+默认值+记录降级事件在成本与信息保留上优于整链重跑（ACM Queue/Microsoft 可靠 agent 设计指引同向，B 级）。

### Q5 每步 JSON 契约可靠性

11. **强模型 JSON 高可靠**：GPT-4o structured output 模式 0.05% 验证错误、99.97% schema 合规；21 个有效模型中 19 个 parse 失败率 <2%。（arXiv:2604.25359 Structured Output Benchmark，A 级）小模型首次合规率 31-45%，3 步自修复（容错 lexer 修复 + schema 引导约束 + validator 反馈重试）显著提升（Self-Healing JSON / so-agent 实测，B 级）。

12. **validate-and-retry 必须设上限**：部分小模型在 T=1.0 下反复违规到死循环；分类学显示失败可分解（parse 失败/schema 违规/语义违约分层处理）。（KDD 2026 Schema Guardrails + arXiv:2610.33949，B 级）

---

## 对架构决策的直接含义（提炼）

- ①步骤间产物用 JSON 契约传递在强模型下风险低且漂移最小（结论 2/11）——已有 client.run JSON 修复+resume 机制兜底（结论 12 同向：修复上限已有）
- ②小模型/弱档模型是 JSON 契约的主要风险面（结论 1）——per-step 模型分档若配弱模型需格式降级预案
- ③⑤步输出 JSON、中间步（②③④）产物建议 JSON 为主 + 关键叙述字段保留 markdown 原文（结论 2 的「忠实信道」+ 结论 3 的「压缩瓶颈」折中：结构化锚定 + 叙述不压缩）
- ④中间步失败：跳过+默认值继续+降级事件记账（结论 9/10），不整链重跑
- ⑤answer-isolation 有 CoVe 级 A 级证据支持（结论 5），定稿设计可放心执行

## R7 上报项

1. 5 步串行流水线对单 agent 的「决策质量」直接对比实验无文献——三臂对照是原创实证义务（job_54 adjust-protocol §6 已列，本轮检索确认空白）
2. 串行 pipeline 的「步骤数甜点区」无统一定论——只有可靠性乘法数学（结论 4）与反例边界（结论 1）可间接推断
3. 「预定义角色库 vs LLM 动态生成视角」无直接对照胜负结论（结论 7）
4. 「web search 在视角生成环节（而非一般 RAG）中的用法」无直接对比实验（结论 8）

## Gaps

- **步骤数甜点区**：无法回溯到常识层——文献只有两个极端证据（并行采样 agent 数 ~15 饱和 vs 多 agent 系统平均不超基线），串行流水线最优步数无人测过。**处置**：架构裁决留给 human（J1.4），实证留给三臂对照（后续 job）
- **视角生成的外部检索形态**：②号 agent 的 web search 用法（检索→抽象关键词→再检索 vs 一步到位）无文献，按工程直觉设计（检索结果作为视角的 source 字段锚定）
- **answer-isolation 的负面证据缺失**：检索不到≠不存在；CoVe 系全部正向可能存在发表偏倚——缓解：我们的 mock 行为测试不依赖该结论（工程验收不预设质量提升）
