# 叶子任务 leaf-2：误差吸收 + 节奏触发操作化（grilling L1）

## 背景一行
rick job_50 human-loop v7 设计输入：需区分「一时想不起来（提取失败）」和「真不会」，以及什么信号推进（advance）、什么信号倒退（regress）。

## 环境注意（必读）
本环境 web_search 与 fetch_content 工具已坏（代理 http://10.229.18.27:8412 宕机），不要反复重试这两个工具。改用 bash 直连：
- 抓页面：`env -u http_proxy -u https_proxy curl -sL -m 15 -A "Mozilla/5.0" <url>`（arxiv.org、ai.google.dev、khanacademy.org、github.com 均可直连）
- Bing 搜索：`env -u http_proxy -u https_proxy curl -sL -m 15 -A "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36" "https://www.bing.com/search?q=<查询词>"`，再 grep 提取 `b_algo` 结果块
- arXiv 检索：`https://export.arxiv.org/api/query?search_query=all:<词>&max_results=10`
- PMC 检索：`https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pmc&term=<词>`（teach-back 操作化文献多在医学教育期刊）

## 已验证锚点（parent 已核实，可直接引用）
- LearnLM 论文 = arXiv:2412.16429，全文 HTML：https://arxiv.org/html/2412.16429v3。其中 3.1 节场景设计（49 个 scenario，含 persona、学习目标、误概念剧本）与 3.3 节 pedagogy assessment 是你要挖的重点——scenario 里如何编排「学生不会/半会」剧本，评估里如何评 pacing。
- khanmigo.ai/learners 页面营销级表述：「guides learners to find the answer themselves」「doesn't just give answers」——无操作细节，你需要找 Khan Academy 官方博客/帮助文档中 hint 系统层级（如 hint ladder 几级、每级给什么）。

## 子问题
1. 生产/学术系统怎么区分「提取失败」vs「不会」：二次探针、换措辞重问、延迟复测、部分线索提示——具体规则与出处（Khanmigo hint 系统、teach-back 医学教育操作化、tutee 澄清式追问文献）
2. LearnLM 五原则（公开教学策略文档）中 follow the student / pace 的操作化：什么信号推进、什么信号倒退。可搜 Bing「LearnLM pedagogical principles」找 Google 教学策略一手文档（edu.google 或 storage.googleapis.com 上的 PDF）
3. 其他 LLM 导师的 pacing 规则或 hint ladder（如 MathGPT、EduChat、Khanmigo 同类）

## 要求
- 只报操作化事实，不重复学习科学实证（检索练习/间隔效应已覆盖，勿重复）
- 每条结论附来源 URL 与信源等级：A=一手文档/同行评审；B=权威文档/综述；C=线索
- 查不到就明写无法确认，禁止编造
- 落盘：/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_50/doing/grilling/briefs/research-L1-leaf-2.md（首块 write ≤60 行，超出用 bash cat >> 追加，每批 ≤60 行）
- 产出 ≤1000 字 + 结论 3 条（每条 ≤40 字）
- 最终回复 = 回执（路径 + 结论 3 条）
