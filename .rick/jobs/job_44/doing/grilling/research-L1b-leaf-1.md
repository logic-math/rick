# 调研：MAD 原始论文与多角色架构（L1b-leaf-1）

## (1) Du et al. 2023 原始效果与设定
结论：正确编号为 arXiv:2305.14325（ICML 2024；任务所给 2305.12036 有误）。主设定 3 agent×2 轮辩论，主模型 ChatGPT-3.5（附录含 GPT-4、Llama-7B）。单 agent→辩论：算术 67.0→81.8、GSM8K 77.0→85.0、国际象棋 ∆PS 91.4→122.9；事实性：传记 66.0→73.8、MMLU 63.9→71.1、棋步合法性 29.3→45.2。基线含多数投票（算术 75、GSM8K 81），辩论均胜。agent 数与轮数增益单调升，≥4 轮趋饱和。
证据强度：强（一手来源、具体数值）；局限：单一弱模型、小样本、未报告显著性检验。
来源：arXiv:2305.14325（PMLR v235 du24e）。

## (2) 单模型多角色（SoM/multi-persona）独立证据
结论：Du et al. 的 Society of Mind 本身即同一 ChatGPT-3.5 的 3 个实例——原始 MAD 即单模型多 agent，非多模型协作。独立证据：SPP（arXiv:2307.05300，NAACL 2024）让单个 GPT-4 动态扮演多 persona，较 Standard/CoT/Self-Refine 提升 7~18%，但任务为自建，且认知协同仅在 GPT-4 涌现（GPT-3.5/Llama2-13B 无效）。Liang et al.（arXiv:2305.19118）angel/devil 双角色辩论有增益，但发现过度顺从致"思想退化"，需说服者干预。Smit et al. 中 Multi-Persona 原始设定比 Medprompt 低约 7%，调"一致度"后才反超（+15%）。
证据强度：中——正效应限于强模型+自建任务+超参调优；等算力下优于单角色采样投票的稳健证据不足。
来源：arXiv:2307.05300、arXiv:2305.19118、arXiv:2311.17371。

## (3) 2024-2025 MAD 复现谱系（Smit、2502.08788 之外）
结论：多项系统研究支持"debate≈采样投票"等效性：arXiv:2505.22960 在等生成预算（4-16 次）下，数学任务 MAD 无稳定优势，自洽投票 SC 在 MATH500 全模型规模最优；MAD 仅在极难题（AIME）与弱模型占优（Qwen2.5-3B AIME 翻倍）；异构 agent 收敛于调和平均。NeurIPS 2025 "Debate or Vote"：7 基准上多数投票与 MAD 持平或更优，理论证明辩论为鞅过程、期望正确率不变。NeurIPS 2024 "Multi-LLM Debate"：条件性支持，效果取决于 agent 多样性与设计。ACL Findings 2025 "Voting or Consensus?"：聚合策略选择影响结果。
证据强度：强（多组独立等预算对照+理论证明），方向一致：MAD 增益主要来自多样本集成，辩论过程本身贡献小。
来源：arXiv:2505.22960、NeurIPS 2025 Debate-or-Vote、NeurIPS 2024 Multi-LLM Debate、ACL 2025 Findings。

## 结论
1. Du et al. 原始为 3 agent×2 轮，六任务提升约 5~16 个百分点。
2. 单模型多角色有条件有效：强模型+动态 persona+调优，弱模型无效。
3. 等算力复现：MAD≈多数投票，辩论本身期望无增益（鞅论证）。
4. 引用编号应为 arXiv:2305.14325，任务中 2305.12036 有误。
