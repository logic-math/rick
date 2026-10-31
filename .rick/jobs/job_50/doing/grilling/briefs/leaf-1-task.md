# 叶子任务 leaf-1：掌握度判级操作化（grilling L1）

## 背景一行
rick job_50 human-loop v7 设计输入：业界 LLM 导师如何把学生自由文本回答映射到掌握分档。

## 环境注意（必读）
本环境 web_search 与 fetch_content 工具已坏（代理 http://10.229.18.27:8412 宕机），不要反复重试这两个工具。改用 bash 直连：
- 抓页面：`env -u http_proxy -u https_proxy curl -sL -m 15 -A "Mozilla/5.0" <url>`（arxiv.org、ai.google.dev、khanacademy.org、github.com、socratic.com 均可直连）
- Bing 搜索：`env -u http_proxy -u https_proxy curl -sL -m 15 -A "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36" "https://www.bing.com/search?q=<查询词>"`，再从 HTML 里 grep 提取 `b_algo` 结果块
- arXiv 论文检索：`https://export.arxiv.org/api/query?search_query=all:<词>&max_results=10`（Atom XML）
- PMC 医学教育文献：`https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pmc&term=<词>`

## 已验证锚点（parent 已核实，可直接引用）
- LearnLM 论文 = arXiv:2412.16429（LearnLM: Improving Gemini for Learning，全文 HTML：https://arxiv.org/html/2412.16429v3）。核心机制：pedagogical instruction following——用 system-level instructions 描述期望教学行为来训练/评测，29 题 pedagogy rubric 七点量表专家评审。你需要从中挖「判级/评估」相关细节（3.3 Pedagogical assessment 一节）。
- ai.google.dev/gemini-api/docs/learnlm 是转跳存根（LearnLM 已并入 Gemini 2.5）。
- Socratic by Google（socratic.com）= 资源检索型（文本/语音识别→检索解释资源），非对话判级导师。
- GenTutor 在 arXiv 搜不到——若你也找不到，写明「无法确认 GenTutor 存在」。

## 子问题
1. Khanmigo / LearnLM / Socratic by Google 如何判级学生回答——具体 rubric、判据（错误断言数、关键概念覆盖、步骤级评分）、判级 prompt 的公开形态
2. 有无论文给出 free-text 学生回答的掌握度评分 rubric（step-level scoring、error taxonomy、BKT-on-LLM 输出等）
3. Khanmigo for Teachers 的 grading/反馈功能规则形态（khanmigo.ai/teachers 可抓）

## 要求
- 只报操作化事实（规则/rubric/机制形态），不重复学习科学实证
- 每条结论附来源 URL 与信源等级：A=一手文档/同行评审；B=权威文档/综述；C=线索
- 查不到就明写无法确认，禁止编造
- 落盘：/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_50/doing/grilling/briefs/research-L1-leaf-1.md（首块 write ≤60 行，超出用 bash cat >> 追加，每批 ≤60 行）
- 产出 ≤1000 字 + 结论 3 条（每条 ≤40 字）
- 最终回复 = 回执（路径 + 结论 3 条）
