# Research L1 · leaf-3：跨会话锚存储操作化（tutoring/learning）

## Summary
本子代理环境 web_search/fetch_content 全部故障且无 bash，无法联网取源；仅父会话已核实的 GenMentor 摘要可直接引用，其余为文献先验知识，逐条标注等级与「未复核」，未编造。操作化答案：跨会话锚＝离散掌握状态（ALEKS 型）＋结构化自编辑 profile 块（MemGPT/Letta 型）＋向量 episodic 库（MemGPT archival/Mem0 型）三层组合。

## 验证状态分级
- [A·父核实] 父会话本轮已核实；[B] 权威文献先验，本轮未复核；[C] 线索，未核实。

## Findings

### 1. ALEKS / 知识空间理论（KST）
1. **状态形态**：ALEKS 基于 Falmagne–Doignon 知识空间理论，非 BKT。BKT 按技能存 p(掌握) 概率向量（Corbett & Anderson 1995）；ALEKS 存离散「知识状态」＝学生当前可解题集合（域内题目子集），全体合法状态构成良分级知识空间。[B]
2. **跨会话持久**：知识状态＋派生 outer fringe（可学前沿：覆盖当前状态且差一题的目标集合）与 inner fringe（刚学会前沿）随账号持久于服务端；长期缺席/定期触发「知识检查」重评估，状态可收缩，即以重评估对遗忘建模。[B]
3. **操作化字段**：item 位集 ＋ outer/inner fringe 集合 ＋ 最近评估时间戳 ＋ 重评估调度标记。
4. 来源：https://www.aleks.com/about_aleks/aleks.html ；Falmagne & Doignon《Learning Spaces》Springer 2011 https://link.springer.com/book/10.1007/978-3-642-10966-4 ；Falmagne/Cosyn/Doignon/Thiéry 2006《The Assessment of Knowledge in Theory and in Practice》。[B·未复核]

### 2. GenMentor（LLM 导师多智能体）
5. **摘要级已核实**：goal→skill 映射（微调 LLM＋自建 goal-to-skill 数据集）、skill gap 识别、evolving 学习路径优化、动态多面 learner profile。https://arxiv.org/html/2501.15749v1 [A·父核实]
6. **无法确认**：profile 具体字段表与跨会话更新流程属全文细节，本轮无法抓全文，明写无法确认，不编造；建议补抓该 HTML 的 learner profile/Profiler 章节后再定字段。

### 3. LLM 记忆系统 × tutoring 形态
7. **MemGPT**（arXiv:2310.08560 [A·父核实 ID]，机制 [B]）：主上下文＝系统指令＋core memory（LLM 经 core_memory_append/replace 自编辑）＋FIFO 消息队列；外部＝recall storage（全量对话分页＋conversation_search）＋archival storage（向量库、MiniLM 嵌入、top-k 检索，archival_memory_insert/search）。跨会话＝core memory/archival 持久；论文 multi-session chat 实验即跨会话共享记忆锚。https://arxiv.org/abs/2310.08560
8. **Mem0**（arXiv:2504.19413 [C·父标注待核实，本轮亦未核实]）：两阶段管线：抽取候选记忆→与相似存量比对，LLM 决策 ADD/UPDATE/DELETE/NOOP；存储＝自然语言事实＋元数据＋向量（图变体存实体/关系并动态删除，Neo4j）；检索＝查询向量 top-k＋LLM 重排。https://arxiv.org/abs/2504.19413
9. **Letta**（MemGPT 后续）[C]：结构化 memory blocks（persona＋human；tutoring 中 human 块≈学生画像）＋archival＋sleep-time agent 在会话间隙后台重组记忆。https://www.letta.ai
10. **时间衰减**：MemGPT/Mem0 基线无显式时间衰减；Zep 以时间知识图谱（bi-temporal、边失效）显式建模时间。https://arxiv.org/abs/2501.14156 [C·ID 凭记忆]

### 4. 检索机制→tutoring 组件映射
11. 向量 top-k（MemGPT archival、Mem0）→ 会话/错题 episodic 日志；结构化自编辑块（MemGPT core、Letta blocks）→ learner profile；离散状态＋fringe（ALEKS）→ 掌握度；时间图边失效（Zep）→ misconception 纠错史（旧误解边自动失效）。

## Sources
- Kept：arXiv:2501.15749（GenMentor，摘要父核实 [A]）；arXiv:2310.08560（MemGPT，ID 父核实 [A]）；ALEKS/Springer KST 文献（先验 [B]）；arXiv:2504.19413（Mem0 [C]）；letta.ai、arXiv:2501.14156（[C]）。
- Dropped：本轮无任何在线源可取（web 工具故障），无从做 SEO 剔除。

## Gaps（明写无法确认）
- GenMentor 全文 learner profile 字段表与跨会话更新流程（需抓 https://arxiv.org/html/2501.15749v1 ，含 Profiler agent 细节）。—主要缺口
- Mem0（2504.19413）、Zep（2501.14156）、Letta 的 URL/ID 本轮未联网复核，维持 C 级。
- ALEKS 官方文档对状态持久化/知识检查的官方措辞未核实（KST 机制为文献共识）。
- 所有 [B] 条目本轮未复核；建议网络恢复后由父会话补验。

## 结论（3 条，每条 ≤40 字）
1. ALEKS 存离散知识状态＋可学前沿，非 BKT 概率；随账号持久＋定期重评估。
2. LLM 导师跨会话锚＝自编辑 profile 块＋向量 episodic 库，Mem0 用增删改算子。
3. 本轮网络工具全断且无 bash；GenMentor 字段级细节无法确认，须补抓全文。

---
运行备注：按运行时覆盖指令，本文件为权威输出路径 leaf-3-receipt.md；任务文件内的 research-L1-leaf-3.md 路径已被取代，未另行落盘。
