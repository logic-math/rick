# Research: Frontier-Bench v0.1 共识与更硬基线（叶子调研）

## Summary
Frontier-Bench v0.1 全部关键细节经联网核实属实：2026-07-23 发布、74 任务/7 领域、Terminal-Bench 继任者、指标 mean reward、Opus 5 用 mini-SWE-agent+GKE 内部跑分 43.3%、june.kim 审计发现"框架盲区"。METR 时间线方法论确认，TH1.1 套件已近饱和。2026 年社区共识：SWE-bench Verified 与 Pro 双双被 OpenAI 弃用，但"哪个基准最难/最能代表真实工程能力"仍无唯一共识——Frontier-Bench、DeepSWE、ARC-AGI-3 各据一席，争论未息。

## Findings
1. **Frontier-Bench v0.1 细节全部属实**。harbor-framework/frontier-bench v0.1.0 于 2026-07-23 发布（release 标题 "Added initial 74 Tasks"），官方定位为 Terminal-Bench 继任者（原 Terminal-Bench 3.0，PR #1417/#1418 更名），74 任务 7 领域（软件工程/ML/安全/数据科学/运维/硬件/媒体），指标 mean reward，发布时最佳模型约 34%。[GitHub v0.1.0](https://github.com/harbor-framework/frontier-bench/releases/tag/v0.1.0) [frontierbench.ai](https://www.frontierbench.ai/announcement)
2. **Opus 5 跑分细节属实但为自报**。Anthropic 公告脚注：内部运行，mini-SWE-agent harness+GKE 后端，mean reward over 5 attempts，Opus 5=43.3%（约 5% 调用被拒、4% 回退 Opus 4.8 兜底），Opus 4.8=18.7%、Fable 5=33.7%；与官方榜（GPT-5.6 Sol 34.4%、Fable 5 33.8%、Opus 4.8 21.1%）harness/attempt 预算不同，不可直接比较。[Anthropic](https://www.anthropic.com/news/claude-opus-5) [backgrind](https://backgrind.com/blog/agentic-coding-benchmarks-2026/)
3. **june.kim 审计属实且有实质发现（需留意，非阻塞）**。审计覆盖全部 74 任务配置与 grader（Harbor 0.20.0、commit 2d260bc），发现"框架盲区"：所有任务采用独立验证容器（先销毁 agent 容器再评分），未申报状态不可见；9 个 gold-passing 任务中植入 SSH 密钥/第二 git 仓库/客户 CSV 后删除，reward 仍为 1（issue #1429 已上报）。审查管线含静态检查、oracle 验证、/cheat 与 /fortify、35 条 rubric。[june.kim](https://june.kim/auditing-frontier-bench) [审计仓库](https://github.com/kimjune01/frontier-bench-audit) [issue #1429](https://github.com/harbor-framework/frontier-bench/issues/1429)
4. **METR 方法论与长期任务**：任务标注人类完成时间，拟合逻辑斯蒂曲线得 50% 时间线；TH1.1（2026-01-29）套件 170→228 任务、8h+ 长任务 14→31，官方承认套件近饱和（"最新一代模型几乎全能完成"，>16h 测量不可靠），2026-03 修正正则化建模错误使近期模型 50% 时间线最多下调 20%。[TH1.1](https://metr.org/blog/2026-1-29-time-horizon-1-1/) [时间线页](https://metr.org/time-horizons/) [建模假设](https://metr.org/notes/2026-03-20-impact-of-modelling-assumptions-on-time-horizon-results/)
5. **SWE-bench 系出局已成共识**：OpenAI 2026-02 宣布 Verified 因污染/设计缺陷不再衡量前沿能力并推荐 Pro；2026-07-08 审计 Pro 公共集 731 任务（管线标记 200=27.4%、人工 249=34.1% 破损，估算 ~30% broken），正式撤回推荐。[OpenAI-2月](https://openai.com/index/why-we-no-longer-evaluate-swe-bench-verified/) [OpenAI-7月](https://openai.com/index/separating-signal-from-noise-coding-evaluations/)
6. **"最代表真实工程/最难"无唯一共识**：backgrind 认为 Frontier-Bench v0.1 与 ARC-AGI-3 是仅存有区分度者（Verified 实质退役、Terminal-Bench 2.1 饱和于 74-84% 带、ARC-AGI-2 饱和 92.5%）；ARC-AGI-3（2026-03 发布，交互式，人类 100% 可解而前沿模型发布时 <1%，2026-07 止 Opus 5 最高 30.16%）区分度最强但非纯工程；DeepSWE（113 原创任务、91 仓库 5 语言、判官分歧 1.4% vs Pro 32.4%、分差带宽 69.8 点）被部分社区视为 Pro 继任者。结论：工程代理基线选 Frontier-Bench，纯 SWE 抗污染选 DeepSWE，判难度上限看 ARC-AGI-3；争论仍在。[backgrind](https://backgrind.com/blog/agentic-coding-benchmarks-2026/) [ARC-AGI-3 报告](https://arcprize.org/media/ARC_AGI_3_Technical_Report.pdf) [DeepSWE](https://arxiv.org/abs/2607.07946) [TopInsight](https://topinsight.co/ai-coding/swe-bench-replacement-deepswe/)

## Sources
- Kept: GitHub v0.1.0 release（任务清单/时间戳）；frontierbench.ai 公告（34%、7 领域）；Anthropic Opus 5 公告（脚注跑分细节）；june.kim+kimjune01 审计（框架盲区证据）；OpenAI 两篇原文（Verified 退役/Pro 撤回，经 aetos.ai、times.of.ai、ai-tldr.dev 多镜像交叉核实）；backgrind.com 综述（2026-07 四方基准全景）；METR TH1.1/时间线/建模假设三篇；DeepSWE arXiv+官网（v1.1 于 2026-08-20 更新）；ARC-AGI-3 技术报告。
- Dropped: codingfleet.com/llm-stats.com/llmreference.com 聚合页（衍生数据，仅作交叉佐证）；Snorkel 与 PRNewswire（宣传口径）；NVIDIA AVO 100% ARC-AGI-3 报道（专用 harness，与通用模型不可比）。

## Gaps
- openai.com、backgrind.com、june.kim 直接抓取 DNS 失败，内容经搜索快照与镜像站交叉确认，非逐字核读原文。
- Frontier-Bench 官方榜上 Opus 5 一行系自报（0 verified / 1 self-reported），无独立复现。
- NVIDIA AVO 声称 ARC-AGI-3 100%，说明 ARC-AGI-3 对专用系统已有解法，2026 下半年"区分度"结论可能快速变化。
- 社区共识仍在演化（DeepSWE v1.1、Terminal-Bench 3.0/Frontier-Bench 命名混用），建议发布前复查最新榜单。

## 结论（每条 ≤40 字）
1. Frontier-Bench 细节全核实：74任务/7领域、mean reward、Opus5 跑分43.3%，当前最硬代理基线。
2. Verified/Pro 双退役；DeepSWE 抗污染、Frontier-Bench 多维最硬，各据一席，无唯一共识。
3. 选基线看 Frontier-Bench（顶分~34%），勿信自报分；难度上限参考 ARC-AGI-3 与 DeepSWE。
