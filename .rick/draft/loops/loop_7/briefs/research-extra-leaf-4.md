# Research: harness/编排级评测 + HITL 评测方法（leaf-4）

## Summary
9 篇文献全部核实。核心事实：harness 方差 ≥ 模型方差（通过率差 ≤8pp 但成本差 40×、HV/MV=7.8×、同池分差 23.8pp、基础设施差异 6pp）；HITL 客观化有三条可落地路线（偏好+Elo、反馈→指标、模拟用户+rubric），各有量化收益与局限。

## Findings
1. **Scaffold Effect (2607.22585)**：300 次试验（2 模型×Goose/OpenCode/OpenHands-SDK×50 Terminal-Bench Pro 任务）。固定模型跨 harness 通过率差 0–8pp（bootstrap CI 含零），但 tokens/solved task 差 40×（Goose 28K vs OpenCode 1.1–1.5M）；模型升级仅 1.0–1.3×。失败指纹模型无关（Goose=REASON、OpenHands=VERIFY/MAX_TURNS、OpenCode=idle/TIME），no-action turns 差 10×。评测单元应为 harness–model 对，必报 tokens/solved、no-action turns、failure vector。[arXiv](https://arxiv.org/abs/2607.22585)
2. **Stop Comparing (2605.23950, ICML 2026)**：Binding Constraint Thesis——长程任务+可比前沿模型下 harness 方差 ≥ 模型方差。观测证据：Morph 上 6 模型同 scaffold 仅差 4.9pp，固定模型变 harness 差 9.5pp；SWE-bench Verified Mini 跨 scaffold 差 34–48pp。受控因子实验（3 模型×3 配置×100 任务）：HV=18.48pp² vs MV=2.37pp²（7.8×），9 对比较 6 次排名反转。处方：Harness Card 披露 7 层 + locked/factorial 协议 + 轨迹级指标。[arXiv](https://arxiv.org/abs/2605.23950)
3. **Harness-Bench (2605.27922)**：106 任务×8 工作流×6 harness×8 后端=5,194 轨迹。同任务同模型池 harness 分差 23.8pp（NanoBot 76.2 vs OpenClaw 52.4；Codex 80.4 单列）。效应集中于弱模型与 tool 密集/有状态工作流；失败分类：契约违背/恢复失败/grounding 缺口/artifact 失败/状态丢失。[arXiv](https://arxiv.org/abs/2605.27922)
4. **Position: Coding Benchmarks (2606.17799)**：三症状——分数混淆模型/harness/环境、单一参考解惩罚等价方案、无组件级信号难迭代。提出 system harness 五组件（任务/agent harness/环境/上下文/反馈信号）与三环反馈（inner/middle/outer）。处方：harness-aware 元数据、支持多解形的 verifier、组件级评测。旁证（Anthropic）：Terminal-Bench 2.0 资源执行严格↔无上限差 6pp（p<0.01）。[arXiv](https://arxiv.org/abs/2606.17799)
5. **Computer Agent Arena (ICLR 2026)**：双匿名 CUA 同云桌面并行执行、用户盲投，Bradley-Terry/Elo 汇总。2,201 有效票/1,058 用户/12 模型（标注 α 0.68–0.78）。与 OSWorld 等静态基准排名大幅翻转；正确性是偏好主驱动，延迟影响≈0；澄清次数倒 U 形。局限：人群偏差、票数稀疏、模拟桌面。[GitHub](https://github.com/xlang-ai/computer-agent-arena)
6. **AutoLibra (ICLR 2026, 2505.02820)**：开放反馈→(行为,反馈,符号) grounding→聚类成含定义+正反例的指标→LLM-as-a-Judge 打分；coverage/redundancy 元评测。Baba-Is-AI 每阶段仅 18 条标注轨迹，3 阶段成功率 +>20%；WebVoyager 再 +5%。局限：全链路依赖 LLM。[官网](https://autolibra.org/)
7. **HAS-Bench (2607.04329)**：图式框架，5 级 agency（A1 全自动–A5 人主导）×3 通道（澄清/反馈/控制）×persona，397 任务 6 域。A3 平等伙伴 vs A1：Pass@1 +8.4、Task Score +11.5、安全率 +26.9；恢复 65.4% 自主失败任务；弱模型 ≈0 增益；A3→A4 救 27 破 13 边际递减。局限：人类由 LLM 模拟。[arXiv](https://arxiv.org/abs/2607.04329)
8. **AgencyBench (ACL 2026, 2601.11044)**：32 场景/138 任务/6 能力，平均 ~90 次工具调用、~1M tokens、数小时。user simulation agent（按 rubric 缺项反馈）+ Docker 沙箱视觉/功能 rubric 自动评测，绕开 HITL 瓶颈。闭源 48.4% vs 开源 32.1%；模型有原生 scaffold 主场优势。局限：模拟反馈受 rubric 完整性约束、成本高。[ACL](https://aclanthology.org/2026.acl-long.337/)
9. **LH-Bench/Beyond Binary (2603.22744)**：三支柱——专家 SKILL.md rubric 喂 LLM judge + ground-truth artifacts 步级奖励 + 人类成对偏好收敛验证。专家 rubric 使 judge 一致性 κ 0.46→0.60；人类偏好确认排序边界（p<0.05）；结构化 verifier 反馈恢复 70.3% 错误。警示：单次运行级人机 judge 一致性弱（κ 0.06–0.08），细粒度 LLM 分差不可信。[arXiv](https://arxiv.org/abs/2603.22744)

## Sources
- Kept: arXiv 原文（7 篇）、ICLR 2026 论文/OpenReview（CAA、AutoLibra）、ACL Anthology（AgencyBench）、harness-bench.ai/autolibra.org 官网、Anthropic 工程博客（基础设施 6pp 旁证）。
- Dropped: emergentmind/pubdb/alphaxiv 等聚合页（仅交叉验证）、coarena.ai（非论文商业榜单）、二手博客。

## Gaps
- Scaffold Effect 与 Harness-Bench 的 harness 清单不重叠，跨论文不可直接比对；n=50 时通过率差异多不显著。
- HITL 路线均含"模拟人"环节；真实人类规模验证仅 CAA（2,201 票）与 LH-Bench（135/275 票），存在人群/单专家偏差。
- 1M-token 长上下文仅 AgencyBench 一家，成本与可重复性数据未细化。

## 结论（每条 ≤40 字）
1. 评测以 harness–model 对为单位：通过率差 ≤8pp，成本差 40×，失败指纹模型无关。
2. HITL 客观化三范式：人类成对偏好+Elo、开放反馈→指标、模拟用户+rubric 自动评测。
3. 人类参与收益非单调：A3 平等伙伴最优，更高授权边际递减；模拟用户/人群存偏差。
