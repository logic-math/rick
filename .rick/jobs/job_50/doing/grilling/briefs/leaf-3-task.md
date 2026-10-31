# 叶子任务 leaf-3：跨会话锚存储操作化（grilling L1）

## 背景一行
rick job_50 human-loop v7 设计输入：tutoring/learning 系统在会话间持久化什么供下次 retrieval/复习用。

## 环境注意（必读）
本环境 web_search 与 fetch_content 工具已坏（代理 http://10.229.18.27:8412 宕机），不要反复重试这两个工具。改用 bash 直连：
- 抓页面：`env -u http_proxy -u https_proxy curl -sL -m 15 -A "Mozilla/5.0" <url>`（arxiv.org、github.com、aleks.com、mih信用网一般可直连；wikipedia 被阻）
- Bing 搜索：`env -u http_proxy -u https_proxy curl -sL -m 15 -A "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36" "https://www.bing.com/search?q=<查询词>"`，再 grep 提取 `b_algo` 结果块
- arXiv 检索：`https://export.arxiv.org/api/query?search_query=all:<词>&max_results=10`

## 已验证锚点（parent 已核实，可直接引用）
- GenMentor 论文 = arXiv:2501.15749（LLM-powered Multi-agent Framework for Goal-oriented Learning in ITS）。摘要已核实：goal→skill 映射（微调 LLM + 自建 goal-to-skill 数据集）、skill gap 识别、evolving optimization 排学习路径、动态多面 learner profile。你要挖全文里的「profile 存什么字段、跨会话怎么更新」：https://arxiv.org/html/2501.15749v1
- MemGPT = arXiv:2310.08560；Mem0 = arXiv:2504.19413（待你核实）。
- ALEKS 官网可直连；知识空间理论（Falmagne & Doignon）原始文献可在 arXiv/期刊页找。

## 子问题
1. ALEKS 的 BKT/知识空间状态如何跨会话持久化：掌握度向量形态、知识空间结构（ALEKS 官方文档 + Falmagne 知识空间理论文献）
2. GenMentor / 其他 LLM 导师论文的跨会话记忆机制：misconception 日志、concept index、掌握度快照、learner profile 字段
3. LLM 记忆系统（MemGPT、Mem0、Letta）在 tutoring 场景应用形态：存储字段、检索机制（向量检索/结构化字段/时间衰减）

## 要求
- 只报操作化事实（存储字段/状态形态/检索机制），不重复学习科学实证
- 每条结论附来源 URL 与信源等级：A=一手文档/同行评审；B=权威文档/综述；C=线索
- 查不到就明写无法确认，禁止编造
- 落盘：/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_50/doing/grilling/briefs/research-L1-leaf-3.md（首块 write ≤60 行，超出用 bash cat >> 追加，每批 ≤60 行）
- 产出 ≤1000 字 + 结论 3 条（每条 ≤40 字）
- 最终回复 = 回执（路径 + 结论 3 条）
