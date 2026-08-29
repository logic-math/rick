# Research: SWE-bench 家族 + SWE-Bench Pro 精确事实核实

## Summary
SWE-bench 家族以"真实 GitHub issue → 生成补丁"为核心任务形态，由 Full(2,294)、Lite(300)、Verified(500,与 OpenAI 合作)、Multimodal(517 test)、Multilingual(300) 组成，是业界衡量 AI coding 工程能力的事实标准，swebench.com 官方排行榜被前沿模型系统卡广泛引用。SWE-Bench Pro(Scale)以 1,865 个长时程多语言任务著称，论文期 SOTA 仅 23.3%，但 OpenAI 2026-07 审计确认其公开集约 30% 任务 broken 并撤回推荐——该结论属实，选型时需谨慎对待 Pro 分数。

## Findings
1. **Full**：2,294 个测试实例，12 个热门 Python 仓库的 issue-PR 对；模型输入 issue 文本+base_commit 代码库，输出补丁，由 test_patch 中的 FAIL_TO_PASS/PASS_TO_PASS 测试判定。[swebench.com](https://www.swebench.com/) [HF 数据集](https://huggingface.co/datasets/SWE-bench/SWE-bench)
2. **Verified**：500 实例，2024-08-13 发布，与 OpenAI Preparedness 合作（第二部分）；工程师逐条筛"描述清晰、测试正确、可解"。[OpenAI 公告](https://openai.com/index/introducing-swe-bench-verified/) [swebench verified](https://www.swebench.com/verified.html)
3. **Lite**：300 测试+23 开发实例（11 仓库），2024-03 发布；按"单文件补丁、无图片/链接/commit引用"等 6 条规则筛选降本。[Lite 页](https://www.swebench.com/lite) [make_lite](https://github.com/SWE-bench/SWE-bench/tree/main/swebench/collect/make_lite) 注：新版 datasets 指南页写 534，与官方页及排行榜 300 冲突，判为文档错误。
4. **Multimodal**：共 619 实例（17 个 JS 仓库），test 517、dev 102；每例至少 1 张图（截图/线框图/图表），test split 私有、经 sb-cli API 评测。[论文 arXiv:2410.03859](https://arxiv.org/abs/2410.03859) [页面](https://www.swebench.com/multimodal)
5. **Multilingual**：300 任务、9 语言、42 仓库，定位跨语言泛化评测。[发布页](https://www.swebench.com/multilingual) [排行榜](https://www.swebench.com/multilingual-leaderboard.html)
6. **任务形态与 pass@1**：issue(problem_statement)+base_commit → 模型生成 model_patch；Docker 应用补丁跑测试，FAIL_TO_PASS 全过且 PASS_TO_PASS 无回归即 resolved（pass@1）。[evaluation docs](https://github.com/swe-bench/swe-bench/blob/main/docs/guides/evaluation.md) [FAQ](https://github.com/swe-bench/swe-bench/blob/main/docs/faq.md)
7. **排行榜生态**：swebench.com 官方榜（Verified/Multimodal/Multilingual/Lite/Full，默认 mini-SWE-agent harness）；Claude Opus/Sonnet、GPT-5、Gemini 3 等系统卡普遍报告 Verified/Pro 分数，OpenAI 亦称 Pro 是"多数 coding-agent 实验室发布新模型时引用的基准"。[swebench.com](https://www.swebench.com/) [viewer](https://www.swebench.com/viewer.html)
8. **Pro 基本盘**：1,865 问题、41 个活跃仓库；公开集 11 仓(731 任务)、held-out 12 仓、商业集 18 仓(276 任务)；多语言（Python/Go/JS-TS 等）、long-horizon（人工需数小时至数天、跨多文件补丁）。[arXiv:2509.16941](https://arxiv.org/abs/2509.16941) [Scale Labs](https://labs.scale.com/papers/swe-bench-pro) [HF](https://huggingface.co/datasets/ScaleAI/SWE-bench_Pro)
9. **Pro SOTA**：论文（截止 2025-09-18）公开集 GPT-5 23.3%、Opus 4.1 22.7%，全部<25%；商业集 Opus 4.1 17.8%。OpenAI 审计称 8 个月内前沿模型公开集从 23.3% 升至 80.3%（各聚合站现报 80.3%~84.8%，口径不一）。[arXiv HTML](https://arxiv.org/html/2509.16941) [OpenAI audit](https://openai.com/index/separating-signal-from-noise-coding-evaluations/)
10. **ICML 2026 poster**：确认属实（poster id 61047）。[icml.cc](https://icml.cc/virtual/2026/poster/61047)
11. **OpenAI 审计（2026-07-08/09）属实**：审计 731 公开任务，自动化管道标 200(27.4%)、5 名工程师人工标 249(34.1%) broken，估计约 30%；四类缺陷=测试过严/提示欠规格/测试覆盖不足/提示误导；OpenAI 撤回此前"采用 Pro"推荐。数字未经独立第三方验证，Scale 当时未公开回应；另 Artificial Analysis 已于 2026-06 因可作弊（模型从 commit 历史抄答案）将 Pro 移出其 Coding Agent Index。[OpenAI 原文](https://openai.com/index/separating-signal-from-noise-coding-evaluations/) [AI/TLDR](https://ai-tldr.dev/releases/openai-swe-bench-pro-audit-jul8/) [the-decoder](https://the-decoder.com/openai-finds-roughly-30-percent-of-popular-ai-coding-test-is-broken/)

## Sources
- Kept: OpenAI Verified 公告、swebench.com 官方页/指南、GitHub swe-bench docs、arXiv 2509.16941、labs.scale.com、HF 数据集、icml.cc、OpenAI 审计原文。
- Dropped: BenchLM/AnotherWrapper/morphllm 等聚合站（厂商口径且数据互有出入，仅作旁证）；aiinsiders/beckmann（二手转述，与一手事实一致时仅作佐证）。

## Gaps
- Lite "534 vs 300" 出处未确认（疑为新指南页笔误）；Multilingual 精确发布时间未深挖；Pro 当前榜单最高分各聚合站不一致（80.3%~84.8%），建议以 Scale 官方榜或模型系统卡为准。

## 结论
1. 家族规模：Full2294、Ver500、Lite300、MM517、ML300
2. 任务为真实issue→补丁，pass@1由测试补丁判定
3. Pro审计属实：约30%任务broken，OpenAI撤回推荐，选型慎用
