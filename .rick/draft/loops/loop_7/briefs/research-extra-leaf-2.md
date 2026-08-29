# 调研:非 SWE 的 Agentic/工程基准(leaf-2)

## Summary
终端/仓库级/工具类工程基准已成梯队:Terminal-Bench 系列(容器终端、89 任务、SOTA 约 88%、tbench.ai 归 Harbor 团队、含 Z.ai Verified 变体)是当前 CLI 智能体事实标准;其余各基准定位不同。核实结论:SWE-bench 测补丁能力、Terminal-Bench 测终端操作能力,"不同技能而非难度阶梯"(dreaming.press)成立;而"OpenHands(原 ML-Bench)"说法不成立——ML-Bench 为 Yale 独立基准,OpenHands 仅集成其评测。

## Findings
1. **Terminal-Bench v1**(2025.05,Laude/Stanford):任务=指令+Docker 镜像+测试+时限;任务数官方 beta 口径约 100(80/105 各源不一,未核实)。
2. **TB 2.0**(2025.09,随 Harbor 发布):89 任务(93 贡献者从 229 投稿中选),每任务独有环境+人工解答+验证测试;GPT-5.2+Codex CLI 约 63%,难度定位 hard。
3. **TB 2.1**:2.0 修复版,改 26-28 任务、持续验证,受 Z.ai 的 TB 2.0 Verified 启发;Z.ai Verified(zai-org HF)后并入官方 2.1。SOTA 约 88% 属实:GLM-5.3 88.2%(2026-08 快照)。
4. **tbench.ai 归 Harbor 团队**属实,为 harbor-framework 官方站;Anthropic/Cursor/Codex CLI 均公布 TB2.0 分数,生态接受度高。
5. **OpenHands 非"原 ML-Bench"**:ML-Bench(Yale gersteinlab,2311.09835)9,641 例/169 任务/18 个 ML 仓库,LLM-Bench+AGENT-Bench 双设置;OpenHands(前 OpenDevin)仅将其纳入 eval harness。SOTA 无活跃榜单,未核实。
6. **RepoBench**(ICLR'24):仓库级代码补全(非 agentic),Python+Java,三任务 P(下一行预测)/R(检索)/C(检索增强补全);具体规模数字未核实。
7. **LiveCodeBench**:防污染竞编程基准,从 LeetCode/AtCoder/CodeForces 持续收录,现约 400+ 题,四场景(生成/自修复/执行/测试输出预测);SOTA 约 89-93.5%(2026)。
8. **AgentBench**(THUDM,ICLR'24):8 环境(含 OS 终端/DB/KG/WebShop/Mind2Web),测 29 个 LLM;通用 agent 定位,非工程专项;SOTA 未核实(已过时)。
9. **GAIA**(Meta/HF):450 题(验证集 165)/3 级,测推理/多模态/浏览/工具用;人 92% vs GPT-4 原 15%;SOTA 依赖脚手架:Claude Sonnet 4.5 约 74.6%(HAL),裸模型 Claude Mythos 5 约 52.3%(2026)。
10. **H-HEval**:未核实——三轮检索未见权威来源(慎与 IHEval/HD-Eval 混淆)。
11. **tau-bench**(Sierra,2024.06):模拟 LLM 用户对话+领域 API+策略约束,airline/retail,pass^k 指标;现 τ²(多模态/知识检索)/τ³(语音);SOTA Gemini 3 Pro 90.7%(2026-04,版本口径未核实)。
12. **Aider polyglot**:225 道 Exercism 难题(C++/Go/Java/JS/Python/Rust),在 aider 真实编辑循环内评分,o1(high)首发登顶;近期榜首约 60%+(未核实)。属编辑型编码基准,非终端 agentic。
13. **观点验证**(dreaming.press 2026):原文"They're not the same test at two difficulties... two genuinely different skills"——SWE-bench=补丁能力(issue→patch 过测试),TB=终端操作能力(实机多步任务);dev.to/digitalapplied/掘金同调,成立。

## 结论(每条≤40字)
1. Terminal-Bench 2.1 最贴近 harness 工程测评:容器终端 89 任务、SOTA 88%、tbench.ai 官方权威。(tbench.ai)
2. 更正:ML-Bench 系 Yale 独立基准,OpenHands 仅集成评测,非"原 ML-Bench"。(ml-bench.github.io)
3. 观点成立:SWE-bench 测补丁、TB 测终端操作,是两种技能而非难度阶梯。(dreaming.press)

## Sources
- https://www.tbench.ai/ | https://github.com/harbor-framework/terminal-bench-2-1 | https://huggingface.co/datasets/zai-org/terminal-bench-2-verified | https://benchlm.ai/benchmarks/terminalbench21
- https://ml-bench.github.io/ | https://github.com/OpenDevin/OpenDevin/pull/2015
- https://arxiv.org/abs/2306.03091 | https://livecodebench.github.io/ | https://github.com/THUDM/AgentBench | https://arxiv.org/abs/2311.12983 | https://hal.cs.princeton.edu/gaia | https://taubench.com/ | https://arxiv.org/abs/2406.12045 | https://github.com/aider-ai/polyglot-benchmark/ | https://dreaming.press/posts/terminal-bench-vs-swe-bench.html
