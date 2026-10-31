# 叶子任务 leaf-4：真实异质性工程 + 文本绘图教学实践（grilling L1）

## 背景一行
rick job_50 human-loop v7 设计输入：A=如何构造「真实异议」（非角色扮演）；B=LLM 用 ASCII/文本图教学的实操与失败模式。

## 环境注意（必读）
本环境 web_search 与 fetch_content 工具已坏（代理 http://10.229.18.27:8412 宕机），不要反复重试这两个工具。改用 bash 直连：
- 抓页面：`env -u http_proxy -u https_proxy curl -sL -m 15 -A "Mozilla/5.0" <url>`（arxiv.org、github.com 均可直连）
- Bing 搜索：`env -u http_proxy -u https_proxy curl -sL -m 15 -A "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36" "https://www.bing.com/search?q=<查询词>"`，再 grep 提取 `b_algo` 结果块
- arXiv 检索：`https://export.arxiv.org/api/query?search_query=all:<词>&max_results=10`
- GitHub 仓库搜索（无需 token）：`https://api.github.com/search/repositories?q=<词>`

## 已验证锚点（parent 已核实，可直接引用）
- Du et al. 多模型辩论 = arXiv:2305.14325（Improving Factuality and Reasoning in Language Models through Multiagent Debate）。你要挖：辩论轮次结构、答案收敛机制、与单模型自一致性(self-consistency)的对照结论。
- 「Debate Only When Necessary」= arXiv:2504.05047（自适应多智能体协作，省成本）。
- ALEKS 不在本叶子范围。

## 子问题（A：真实异质性）
1. 多模型辩论的实证形态：异构 base model ensemble（不同厂商模型互相 cross-examine）有无论文/实现；与同一模型扮演反方（devil-advocate prompting）效果对照的证据
2. 异构信源 ensemble：不同 RAG 语料立场（如检索自立场不同的媒体/教科书）构造分歧，有无实现
3. 真实人类分歧记录：标注分歧数据集（如 AmbigQA、subjectivity/disagreement NLP 数据集）、RLHF 标注分歧的利用，有无进入导师系统的案例
4. 有无「persona 之外」的异质性工程：如随机温度采样多立场、检索真实论坛分歧帖（Reddit 辩论数据）

## 子问题（B：文本绘图）
5. LLM/教学系统用 ASCII 图/文本图讲概念的实操案例与最佳实践（prompt 工程：图例说明、坐标对齐、逐行绘制）
6. 已知失败模式：LLM 画 ASCII 图的典型缺陷（对齐漂移、比例失真、字符宽度问题——CJK 全角/半角混排）有无文献或社区报告（如 LLM 排版能力评测、Text-to-Diagram 评测基准 like Graphviz-vs-Mermaid 研究或 ASCII art 论文）

## 要求
- 只报操作化事实，不重复学习科学实证
- 每条结论附来源 URL 与信源等级：A=一手文档/同行评审；B=权威文档/综述；C=线索
- 查不到就明写无法确认，禁止编造
- 落盘：/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_50/doing/grilling/briefs/research-L1-leaf-4.md（首块 write ≤60 行，超出用 bash cat >> 追加，每批 ≤60 行）
- 产出 ≤1000 字 + 结论 3 条（每条 ≤40 字）
- 最终回复 = 回执（路径 + 结论 3 条）
