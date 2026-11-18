# 方向 E 第二轮深挖 · LLM-based 推荐（推理 / 蒸馏 / 语义融合）—— r2-E-deepen

> 调研日期：2026-10-26。输入材料：`research.md`（首轮总览）、`research-genrec/E-llm-rec.md`（本方向首轮）、`research-genrec/F-conference-papers.md`（顶会补充篇）。
> 本轮三件事：**A** 逐条补齐首轮 Gaps（全部抓取 arXiv HTML 全文相应章节 + 核对两个 GitHub 仓库）；**B** 选 SelfDR / Evo-Rec / SARA 三篇深挖到接近可复现（架构、公式与损失、伪代码、训练配方与超参、部署工程细节）；**C** 扫描 2026-07 下旬 ~ 2026-10-26 arXiv 新论文，补充 5 篇首轮与顶会篇均未覆盖的工作（另发现 1 篇已撤稿论文，予以警示）。
> 所有关键数字均出自一手来源（arXiv HTML 全文 / abs 页 / 官方 GitHub 仓库）；抓取不到的逐条标注原因，无编造。

---

## 一、任务 A：首轮 Gaps 逐条补齐

首轮 `E-llm-rec.md` 末尾 5 条 Gaps 的补齐结果如下（编号对应首轮）。

### Gap 1 ✅ SARA §6 离线实验细节（基线名单与分数表）——已补齐

来源：[SARA arXiv HTML v2 §6](https://arxiv.org/html/2609.17639v2)（全文已逐节读取）。

- **SARA-7B 离线对比的通用 MLLM 基线共 2 个**：闭源 **Gemini-3.1-Pro** 与开源 **Qwen3-VL-8B**（另报告 SFT 全量检查点与 QR-DPO 精修版以隔离贡献）。
- **评测集**：从更晚的生产日志构造 **55K 条、与 SFT/DPO 训练作者不相交（author-disjoint）** 的评测集，按极性、类目均衡并按作者流行度分层，防记忆与头部类目偏置。
- **指标**：冻结的离线 Agent Judge 六维分数（coherence/relevance/specificity/safety/polarity consistency/grounding，1–4 分制）+ holistic 分 + **BGE-Sim**（生成理由与原始问卷回答的 BGE 余弦相似度）。
- **完整分数表（Table 4）**：
  | 模型 | Coh. | Rel. | Spec. | Safety | Pol. | Ground. | Holistic | BGE-Sim |
  |---|---|---|---|---|---|---|---|---|
  | Gemini-3.1-Pro | 3.97 | 3.70 | 2.92 | 4.00 | 3.97 | 3.60 | 2.51 | 0.70 |
  | Qwen3-VL-8B | 3.92 | 3.51 | 2.67 | 3.98 | 3.98 | 3.73 | 2.37 | 0.69 |
  | SARA-7B-SFT | 3.99 | 3.99 | 3.07 | 4.00 | 4.00 | 3.63 | 3.15 | 0.82 |
  | SARA-7B-DPO | **4.00** | 3.99 | **3.32** | 4.00 | 4.00 | **3.75** | **3.33** | **0.82** |
  各模型在 coherence/safety/polarity 上接近饱和，差距集中在 relevance/specificity/grounding；QR-DPO 相对 SFT 把 specificity 3.07→3.32、grounding 3.63→3.75、holistic 3.15→3.33。
- **SARA-Ranker 离线指标（此前缺失，已补）**：快手精选直播场景、30 秒滑动窗口采样、最后一天日志为测试集；基线为工业 MMoE 多任务系统（含 SARM）。正理由嵌入路：Click AUC/GAUC +0.06/+0.15pp、Long-view +0.28/+0.24pp、Gift +0.06/+0.12pp；负理由 SID 路：Hate AUC/GAUC +0.35/+0.55pp、Report +0.46/+0.38pp（Report 是最低频事件、逐作者记忆最缺数据处，泛化收益最大）。
- **线上数字精确化**：正路径在线 Click +0.342%、**Watch Time +0.986%**（摘要的 0.99% 即此）、Effective View +0.365%、Follow +0.616%；负路径在线 **Hate −8.164%**、Report −0.4396%、LT-7 留存 +0.027%。A/B 均为 **1% 真实流量独立实验**，30+ 天全量生产、日刷新、在原有延迟预算内。
- **附带补齐**：SID 码本分析（Table 5）——Res-Kmeans、(8192)³ 码本下 SARA 理由表示三级利用率均 100%、两级前缀利用率 7.3%、SID 级碰撞率 1.5%、作者级碰撞 3.3%，优于 TagNex（7.9%/22.5%）与 OneLive MLLM（3.6%/9.5%）；OneLive IA（注入协同监督）仍最低（0.66%/1.76%），为参照上界。SFT 数据量 scaling 曲线：coherence/safety/polarity 早期饱和，relevance/specificity/grounding/holistic 持续受益且未完全饱和。

### Gap 2 ✅ Evo-Rec Stage-2 的 teacher LLM 型号——已补齐

来源：[Evo-Rec arXiv HTML v1 §4.1.3](https://arxiv.org/html/2609.29973v1) + [官方仓库 README](https://github.com/mengdanzhu/evo-rec)。

- **Stage-2 的 CoT 教师是 GPT-5.4**：原文 "we synthesize N=5 candidate CoTs per training instance by **querying the teacher LLM (GPT-5.4)** five times independently with the same prompt"。
- **打分器确认为"冻结的 phase-2 模型"**：在**单样本 CoT 语料**（仓库中为 `sample_1`，见 [README Step 3](https://github.com/mengdanzhu/evo-rec)）上微调得到 p̄_θ，冻结后仅用于计算 Δ_n,k = log p̄_θ(y*|h,z) − log p̄_θ(y*|h)，不参与后续训练。
- 仓库补充细节：教师采样需**非零温度**使 5 条样本有差异，输出仅新增 `reasoning_path` 列（parquet）；仓库自述流程为"6 步"（对齐 SFT → 教师采样 → 训练冻结打分器 → Best-of-N SFT → 排序感知 GRPO → FSDP 合并），且明确 **Step 1 的对齐 SFT 覆盖 8 个互补任务**。

### Gap 3 ✅ SelfDR 是否测过全目录/更大骨干——确认"未测"，原文已核

来源：[SelfDR arXiv HTML v1 §4–§6](https://arxiv.org/html/2609.03313v1)。

- 评测协议为**重排设置**：SASRec 生成 top-20 候选（正样本不保证在候选内）；鲁棒性实验换 LightGCN、ComiRec 作候选生成器，仍为重排。**无全目录 / 端到端生成式评测**。
- 全部 LLM 方法统一用 **LLaMA-3.1-8B-Instruct** 骨干；**无更大骨干实验**。结论节原文："Our work centers on the widely-used LLaMA-8B model… **Future work may extend this paradigm to LLMs of different types and scales** to further validate its adaptability"——即扩展到其他规模明确留作未来工作。
- 补充新信息（首轮未记）：SelfDR 做了**跨域泛化**实验（Clothing↔Home 互测，蒸馏后模型在未见域上仍稳定优于蒸馏前），说明自蒸馏学到的是可迁移知识而非数据集捷径。

### Gap 4 ✅ IMFuse 线上实验与多层嵌入存储成本——确认"无线上实验"，存储成本原文已核

来源：[IMFuse arXiv HTML v2 §4–§6、附录 A](https://arxiv.org/html/2607.27002v2)。

- **无任何线上实验**：全部实验为四个 Amazon 数据集（Toys/Beauty/Clothing/Office）离线 leave-one-out 评测（HR@10/20、NDCG@10/20）。
- **存储/计算成本原文（结论节，逐字）**："**Future work will explore more memory-efficient strategies to reduce the storage and computational costs of multi-layer representations.**"——确认存储成本未解决、留作未来工作。
- 补充新信息（附录 A）：
  - **层数确认**：LLaMA-3-8B 共 **33 层**表示（附录 A.4 "averaged over all 33 LLM layers"），IMFuse 融合 l=0…L 全部 33 层。
  - **换 LLM 泛化**（A.3）：换 **Qwen3-8B** 编码同一物品文本、SASRec 骨干，IMFuse 仍全指标提升（RLMRec 平均 +7.07%、SpecTran +4.46%），证明融合策略可迁移到不同 LLM 层级结构。
  - **单层 vs 多层**（A.1）：单层表示的推荐性能总体随深度上升（深层更强），但简单多层融合在部分设置仍能超过末层——中间层确有互补信息。
  - **物品分组**（A.2）：K-means++ 聚 6 组（elbow + silhouette 定组数），L2 归一化末层表示作聚类输入。

### Gap 5 ✅ 建议项：两个 GitHub 复现仓库核对——已完成

- **Evo-Rec 官方仓库**（[github.com/mengdanzhu/evo-rec](https://github.com/mengdanzhu/evo-rec)，已克隆读取）：见任务 B-2 的"复现级细节"。要点：三数据集统计（Games 3,858 物品 / 49,133 训练 / 6,142 验证 / 6,142 测试；Office 3,459 / 38,924 / 4,866 / 4,866；Industrial 3,686 / 36,259 / 4,533 / 4,533）；数据与 checkpoint 走 Google Drive；`general_reasoning`、`seqrec` 训练/验证、`reasoning`、`cot` 部分标注 "to be released upon acceptance"（复现受限点）；RL 脚本名为 `RL_training_script_no_kl.sh`（暗示 GRPO 无 KL 项，见剩余 Gaps）；仓库基于 SidReasoner 构建。
- **SelfDR 官方仓库**（[github.com/JiangDeccc/SelfDistillation](https://github.com/JiangDeccc/SelfDistillation)）：github.com 页面直接抓取失败（fetch failed），改经 [raw README](https://raw.githubusercontent.com/JiangDeccc/SelfDistillation/main/README.md) 成功。要点：全流程 = ReChorus 跑 SASRec 召回（`candidates.pkl`）→ `explanation_prompt_generate.py` / `train_prompt_generate.py` 生成理由器与重排器提示 → `explan.py` + `explan_reward.py` 做 GRPO 理由器训练 → `joint_all_adaptive.py` 联合自蒸馏（动态加权）→ `test_character.py` 评测；数据经 Dropbox 分发（`item.pkl` / `user.txt` / `user-item.pkl` 三文件制）。仓库未公开逐文件超参（论文称 "Further details…provided in our repo"），具体学习率等需读代码（本轮未逐文件核读，见剩余 Gaps）。

---

## 二、任务 B：方法内部深挖（接近可复现）

选取方向 E 最重要三篇：**SelfDR**（推理内化的蒸馏范式）、**Evo-Rec**（SID 生成式推荐的推理进化）、**SARA**（工业级 MLLM 理由规模化落地，含唯一线上证据）。

### B-1 SelfDR（arXiv:2609.03313，CIKM'26，清华×美团）

来源：[HTML 全文](https://arxiv.org/html/2609.03313v1) ｜ [abs](https://arxiv.org/abs/2609.03313) ｜ [GitHub](https://github.com/JiangDeccc/SelfDistillation)

**架构组成（同一 LLaMA-3.1-8B-Instruct 骨干分饰三角）**
1. **Reasoner**：输入用户历史 H_u + 训练集正样本物品元数据 M_i，输出"该用户为何会喜欢该物品"的针对性理由 r_u,i（Eq.3：r_u,i = LLM_Reasoner(H_u, M_i)）。
2. **教师推荐器**：输入 H_u + 候选集 C_u,i + r_u,i，输出预测（Eq.4）。
3. **学生推荐器（最终交付物）**：输入 H_u + C_u,i，**无理由直出**，输出单字符标识符（每个候选项分配一个字符，top-20 即 A–T），同时用标识符 token 的 log-prob 对全部候选排序得到列表。师生先做同样的指令微调（参数同源），蒸馏期教师冻结（offline distillation）。

**关键公式与损失**
- Reasoner 奖励（Eq.5）：Reward(r_u,i) = Acc_rec(s_u,i)——理由拼入输入后**冻结推荐器**的预测正确性，任意准确率指标均可；用 GRPO（引用 DeepSeekMath 的 Shao et al. 2024）优化，"更稳定收敛 + 更低显存"。
- 防泄漏两道闸：提示词明令不复述元数据；**奖励计算前**对理由做细粒度掩码——与物品标题**连续 ≥3 词完全重叠**的片段被遮掉。
- 蒸馏总损失（Eq.7）：min_Θs [(1−α)·L_ce(Θs) + α·D_kd(Θt, Θs)]；D_kd 取 **softmax 前的 logit 级反向 KL**（Eq.8）：
  D_kd = Σ_{j∈C_u,i} p_Θt(CI_j | H_u, C_u,i, r_u,i) · log[ p_Θt(CI_j|…) / p_Θs(CI_j | H_u, C_u,i) ]，
  反向 KL 使学生集中在教师高概率区（模式寻求）；对齐对象是**候选字符标识符上的输出分布**。
- **动态加权 α（Eq.9，本文核心创新）**：
  α(rank_t_i, rank_s_i) = α_base · f(rank_t_i − k) · σ((rank_s_i − rank_t_i)/τ) · β^𝟙(rank_t_i > rank_s_i)，其中 f(x) = 1 + tanh(x/γ)。
  三因子分工：f 项按"教师把正样本排到的名次相对阈值 k 的距离"软门控（教师排太差→权重趋 0）；σ 项按师生名次差平滑降权（学生已更好→少学教师）；β 惩罚项在"教师劣于学生"时大幅压低 α。阈值 k=5（候选集 20）。

**算法流程（伪代码级）**
```
输入: 训练集 D（按时间切: 前半 → 理由器, 后半 → 蒸馏）; 骨干 LLM θ
1  θ_r ← GRPO(Reasoner 初始化=θ):
     for (H_u, i*) ∈ D_前半:
        r ← Reasoner(H_u, M_i*); 掩码泄漏片段
        s ← FrozenRec(H_u, C_u,i, r)          # 冻结的推荐器
        reward ← Acc(s, i*);  按 GRPO 组相对优势更新 θ_r
2  θ_T, θ_S ← InstructionTune(θ) 各 3 epochs   # 师生同源
3  θ_T ← SFT(θ_T; 输入含 Reasoner 生成的理由)   # 构成教师
4  自蒸馏 (≤3 epochs, early stopping):
     for (H_u, C_u,i, i*) ∈ D_后半:
        (s_t, R_t) ← θ_T(H_u, C_u,i, r_u,i)    # 教师分布 + 正样本名次
        (s_s, R_s) ← θ_S(H_u, C_u,i)
        α ← α_base·f(rank_t−k)·σ((rank_s−rank_t)/τ)·β^[rank_t>rank_s]
        L ← (1−α)·CE(s_s, i*) + α·RKL(p_t ‖ p_s)
        更新 θ_S（θ_T 冻结）
5  部署 θ_S: 单字符输出 + log-prob 排序
```

**训练配方与超参（§4.1.4、§5.1）**
- 数据：Amazon Clothing Shoes and Jewelry、Amazon Home and Kitchen、ML1M；5-core + leave-one-out；3 个随机种子取平均；LLM 基线的外部理由标签（COT4Rec/EXP3RT/RecSAVER 用）由 DeepSeek-V3 生成。
- 指令微调 3 epochs → 蒸馏最多 3 epochs（early stopping）；候选 20，k=5。
- 动态加权超参（Clothing 上敏感性分析）：**α_base 最优 0.9**（过小→退化成 SFT 过拟合；过大→几乎纯蒸馏）、**β≈0.1 最佳**、温度 τ/γ 影响小、最优区间 **[1.0, 2.0]**。
- 硬件：训练与推理均在**单张 NVIDIA A800** 上测；训练时间为全流程 wall-clock（小时），推理在 Clothing 随机抽 2,000 条上测 ms/instance 与 token 用量。

**部署工程要点**
- 输出仅 1 个字符 token：推理开销与无推理的 SOFT 持平（四类方法中最快档），远低于 RecR1（GRPO 后长推理）；ZSRanker 因须同时输出标题防漂移而更慢。
- 训练全程**不调外部 LLM**（对比 EXP3RT/COT4Rec/RecSAVER 需查询闭源模型造标签），训练总时长与多数基线相当。
- 换候选生成器（LightGCN / ComiRec）与跨域迁移（Clothing↔Home）均稳定最优——工程上可即插即用于任意召回器之后。

**关键结果**：Hg@1 = Clothing 0.0118 / Home 0.0136 / ML1M 0.0735（p<0.05，全指标最优；ML1M 上大幅超最强传统基线 0.0640）；自训 8B Reasoner 作理由来源**稳定优于 GPT-4o-mini / Claude-3-Haiku / DeepSeek-V3**（补充观察 GPT-5-Chat 也不敌）——"更强通用教师未必出更强学生"（认知鸿沟 + 领域错配），自蒸馏是更可学的监督源。消融：Sup-Only 过拟合、Dist-Only 不稳、Fixed-α=0.5 劣于动态加权。置信度分析：蒸馏后正样本排 top-2 时模型置信显著升高（训练/测试一致）。

### B-2 Evo-Rec（arXiv:2609.29973，Microsoft×Cornell×Emory）

来源：[HTML 全文](https://arxiv.org/html/2609.29973v1) ｜ [abs](https://arxiv.org/abs/2609.29973) ｜ [GitHub（已克隆核读）](https://github.com/mengdanzhu/evo-rec)

**架构组成**：单骨干 **Qwen3-1.7B，全参数微调**贯穿三阶段；每物品 **L=3 个 SID token**（RQ-VAE 词元化，各数据集独立码本、不跨域共享参数），SID token 追加进词表、随机初始化 embedding；推理时"先解码推理、再 trie 约束 beam search 解码 SID"。

**关键公式与损失**
- 建模：p_θ(z,y|h) = p_θ(z|h)·p_θ(y|h,z)，边际 p_θ(y|h) = Σ_z p_θ(z|h)p_θ(y|h,z)（z 为推理序列，y 为目标 SID）。
- Stage-1 对齐损失（Eq.1）：L_align = −E_x[Σ_t m_t(x)·log p_θ(x_t|x_<t)]；预测/翻译/通用推理样本只对 response 计损（mask prompt），**SID–文本交错样本全序列 LM**（m_t=1，让 SID embedding 从全部语言上下文学习）。
- Stage-2 效用打分（Eq.2）：**Δ_n,k = log p̄_θ(y*_n | h_n, z_n,k) − log p̄_θ(y*_n | h_n)**（p̄_θ 为冻结打分器；两项只差 z_n,k，故 Δ 隔离该 CoT 的增量预测效用）；选择（Eq.3）：k* = argmax_k Δ_n,k，仅当 Δ_n,k* > 0 保留；SFT 损失（Eq.4）：L_BoN = −(1/|A|)Σ[log p_θ(ẑ_n|h_n) + log p_θ(y*_n|h_n, ẑ_n)]（推理与目标联合自回归）。
- Stage-3 约束 beam search 得分（Eq.5）：s_θ(y|h_n,z_n,g) = Σ_ℓ log p̃_θ(y^ℓ | h_n, z_n,g, y_<ℓ)，p̃ 为**掩码到 trie 合法子节点后重归一化**的分布；
- 奖励（Eq.6）：R_EM = 𝟙[r=1]（稀疏）；**R_NDCG = 𝟙[r≤10] · 1/log₂(r+1)**（检索 × 排序双因子，rank-1 时=1，越靠后衰减）。GRPO 组内归一化优化。
- 消融中其他奖励形式：前缀奖励 R_prefix(m) = 1/2^(L−m)（m 为连续匹配层数）；NDCG+Recall 混合 = (1 + 1/log₂(r+1))/2。

**算法流程（伪代码级，对应仓库 6 步）**
```
Step1 对齐 SFT: 8 个互补任务混合训练
  (SID史→SID / SID史→title / title史→title / title史→SID
   双向 title–SID 翻译; 物品级+序列级 SID–文本交错; 通用推理数据)
Step2 教师采样: GPT-5.4, 同一 prompt 非零温度独立采样 K=5 条 CoT/样本
Step3 训练冻结打分器: 在 sample_1 单样本 CoT 语料上 SFT → 冻结
Step4 Best-of-N 拒绝采样 SFT:
  for (h_n, y*_n):
    Δ_k ← log p̄(y*|h,z_k) − log p̄(y*|h), k=1..5
    ẑ_n ← z_{argmax Δ} 若 max Δ>0, 否则丢弃该样本
  SFT on {(h_n, ẑ_n, y*_n)}   # 1 epoch
Step5 排序感知 GRPO (verl + vLLM rollout):
  for h_n: 采样 G=16 条自生成推理
    每条: trie 约束 beam search (B=10) → beam 内正样本名次 r
    reward ← 𝟙[r≤10]/log₂(r+1); 组内归一化; GRPO 更新推理策略
Step6 FSDP 分片 checkpoint 合并 → 单卡 HF 模型评测
推理: 解码 reasoning → trie 约束 beam search (B=10) 全目录出合法 SID
```

**训练配方与超参（§4.1.3 + 仓库）**
- AdamW，80GB A100。Stage-1：lr 2e-5，≤5 epochs。Stage-2：1 epoch，**lr 1e-5，线性衰减，10 步 warmup，batch 72**。Stage-3：verl 框架 + vLLM rollout，**8×A100，batch 256，G=16，B=10，lr 7e-7，≤15 epochs**。评测 R@K/N@K（K∈{5,10}），**全目录排序**（非负采样）。
- 数据：三 Amazon 类目（5-core；滑动窗口最大历史长度 10；时间戳 80/10/10 切分）。规模见 Gap-5。
- 基线数字引自 SIDReasoner（He et al., KDD'26）同一 split/协议：GRU4Rec/Caser/SASRec、TIGER/HSTU/LETTER/LC-Rec、ReaRec/R2ec/SIDReasoner。

**关键结果与机理**
- Games：R@5 0.0710→0.0847（+19.3%）、N@10 0.0563→0.0746（+32.5%）；N@5 +41.1%、R@10 +11.4%——**NDCG 增益大于 Recall**，与排序感知奖励目标一致。
- 消融（Games）：Best-of-N 选择 vs 随机选择 N@10 +27.1%、vs 拒绝采样 +37.1%、R@1 +83.8%；**94.0% 候选 CoT 效用为正**→二元拒绝近乎随机，收益来自"实例内按效用排序"；N=1→3→5 单调升（N@10 0.0571→0.0614→0.0746）。beam vs 无约束采样：N@10 +30.9%。奖励设计：纯 NDCG > NDCG+Recall > 前缀 > 精确匹配。
- 机理分析：三域实例内 Δ 极差 1.22/0.94/1.06（Games/Office/Industrial），Best-of-N 相对随机抽选平均增益 +0.56/+0.43/+0.49；饱和率（log p̄(y*|h)>−0.2）1.9%/11.5%/6.7%——Office 饱和最高、实例内极差最小，故收益最小；拒绝率仅 2.4%/3.2%/2.4%。
- **推理长度在 Stage-3 RL 中自发变短**（无任何长度惩罚），性能反升——有效推理取决于信息量而非链长；thinking 模式 R@1 0.0103→0.0446（+333.0%）。

**部署工程要点**：trie 前缀树从全目录 SID 构建、每步解码仅暴露合法子节点（杜绝非法 SID 的概率浪费）；verl+vLLM 的 rollout 引擎与 FSDP 训练-合并流水线；推理时 beam=10 即可全目录排序。复现注意：部分数据/checkpoint "upon acceptance" 才发布。

### B-3 SARA（arXiv:2609.17639，快手）

来源：[HTML 全文 v2](https://arxiv.org/html/2609.17639v2) ｜ [abs](https://arxiv.org/abs/2609.17639)

**架构组成（collect–scale–integrate 三段）**
1. **数据引擎**：快手直播问卷系统（覆盖 240M 用户）→ 在线 LLM Judge（BitCPM4-0.5B 单趟出六维分 + holistic 1–4）→ 离线 Agent Judge（六维专家 + Senior Reviewer）→ SARA-HQ。
2. **SARA-7B**：Qwen2.5-VL-7B 全参 SFT + QR-DPO，日刷新为 1000 万作者生成正/负理由。
3. **SARA-Ranker**（双路特征接入工业 MMoE+SARM 排序模型）：正理由嵌入路（语义锚）+ 负理由 SID 路（拒绝记忆）。

**关键公式与损失**
- 投放概率动态调整（Eq.1）：P = Min(Max(p₀·r_obs/r_target, p_min), p_max)，p₀=5%（基准，1%–20% 候选经小流量 A/B 定）、r_target=12%、p_min=1%、p_max=20%，r_obs 为过去 5 分钟滑动窗响应率。
- 激励触发（Eq.2）：在线 judge holistic Ĝ>2 → 发 KuaiCoin。
- SFT 损失（Eq.4）：标准自回归 NLL，输入 X_p=(V, T_meta, T_asr, T_comment, I_p)（视频帧+元数据+ASR+评论+极性指令）。
- QR-DPO（Eq.5a/5b）：r_θ(y|X) = β·log[π_θ(y|X)/π_ref(y|X)]，L_DPO = −E log σ(r_θ(y⁺|X) − r_θ(y⁻|X))；偏好对来自 **SFT 模型自采样 K 条 + Agent Judge 排序，rank-1 为 chosen、rank-K 为 rejected**，π_ref=π_SFT，可迭代自提升；数据取 SFT 训练集中 **7K 热门作者**，N=7K 对。
- Agent Judge 聚合规则（Eq.7–9）：六维专家各出 (q_d, e_d)（四级分+证据报告）；Senior Reviewer 非补偿式聚合——∃d∈{relevance, safety, polarity, grounding}, q_d≤2 ⇒ G≤2（硬维度短板不可被语言质量掩盖，如 (4,4,4,4,4,1) 判不合格）。
- 正理由路（Eq.12–17）：三路 token（用户画像 120 token / 作者描述 70 / 正理由 64）各自 64 维稀疏查找表 → 4 层共享编码器（hidden 64、单头、FFN 64）→ 双向残差交叉注意 CA(q,H)=MHA(q,H,H)+q 得 c_u,a（Eq.13–14）→ 与理由 CLS 做互信息式软对比对齐：S = Z^ua(Z^r)ᵀ/τ、Q = ρ(Z^r(Z^r)ᵀ/τ)、τ=0.07，L_MIM = ½[CE(Q,ρ(S)) + CE(Q,ρ(Sᵀ))]（Eq.15–16），总损失 L = L_rank + λ_MIM·L_MIM（Eq.17）；随后进 MMoE，另有门控残差支路把理由特征直通任务塔。
- 负理由路（Eq.18–23）：BGE 编码负理由 R_a⁻ → **三级残差量化**（Res-Kmeans，码本 (8192)³，各级同尺寸 K）→ 累积前缀 ID p₀=s₀、p₁=K·s₀+s₁、p₂=K²s₀+K·s₁+s₂（Eq.20）；作者表示 x_a = E_aid[a]‖E₀[p₀]‖E₁[p₁]‖E₂[p₂]（Eq.21）；目标感知负反馈历史建模 h_u,a⁻ = MHA(x_a, X_u⁻, X_u⁻)（Eq.22，X_u⁻ 为用户 Hate 过的作者序列）；负反馈塔预测 ŷ = f_Htr(b_u,a ‖ h_u,a⁻)（Eq.23）——共享码本让"因激进带货被 Hate"泛化到语义相近的未见作者。

**算法流程（伪代码级）**
```
[数据引擎] 185 天在线收集(延迟触发 T=10s → 极性选择 → 开放式理由)
  → 1,915,718 候选 → Agent Judge(G_agent>2, 硬维度不可补偿)
  → 187,532 HQ(9.8%) → 按作者×极性聚合成 86,564 作者画像 → 语义标签化 → SARA-HQ
[SARA-7B] ① 全参 SFT on 187,532 条 ② QR-DPO: 自采样 K 条/实例
  → Agent Judge 排序 → (rank-1, rank-K) 偏好对 7K 作者 → DPO(π_ref=π_SFT)
  → 日刷新: 为 10M 作者生成正/负理由
[SARA-Ranker] 正路: 理由嵌入作 UA 交互的语义锚(MIM 对齐, 与 L_rank 联训)
             负路: 负理由→BGE→3级残差量化→前缀ID→Hate 历史目标注意力→负反馈塔
[线上] 特征日刷新注入 MMoE+SARM; 1% 流量 A/B; 30+ 天生产
```

**训练配方与数据统计（论文给出的全部粒度）**
- SFT/QR-DPO 的学习率、epoch、GPU 数**论文未披露**（复现需联系作者，见剩余 Gaps）；SFT 数据量效应：质量随 SARA-HQ 规模持续提升且未饱和（relevance/specificity/grounding/holistic 受益最明显）。
- 数据引擎统计：正 1,584,866→107,540（6.8%）、负 330,852→79,992（24.2%，负反馈更"缺陷聚焦"故通过率高）；82% 回答 ≤10 汉字（中位 5、均值 8.25）；UA 矩阵 141,461×86,564，户均 1.33 份问卷、作者均 2.17 条回答；184,142 条（98.2%）获得语义类目（18 一级域 / 141 二级类）；末期日收集 ~10K 正 / 2.5K 负，21 天滚动 HQ 得率 ~5%（正）/ ~21%（负）。
- Judge 校准（Table 3，MAE 越低越好）：零样本 BitCPM4-0.5B holistic 0.58 → 训练后在线 judge 0.35 → 专家分体 0.29 → Agent Judge 0.20，专家间上界 0.12。

**部署工程细节**
- 延迟触发 T=10s 是"感知未成形 vs 负面用户已流失"的工程折中；稀疏投放 + 5 分钟滑窗动态调整防问卷疲劳；阈值激励（KuaiCoin）提升高质量响应意愿。
- 在线 judge 用 0.5B 小模型（BitCPM4-0.5B）保单趟低延迟，仅做激励分配；离线高保真筛选由 Agent Judge 独立执行（不使用在线分作输入，避免偏差传导）。
- MLLM 完全离线日刷新生成特征，线上仍是判别式 MMoE ranker——**LLM 不在在线关键路径**，延迟预算不变。
- 评测防泄漏设计：55K 评测集与训练作者不相交、按作者+直播上下文切分、标注者盲测候选来源。

**关键结果**：见 Gap-1 表（离线分数表 + SARA-Ranker 离线 pp 增益 + 在线 A/B 数字）。核心结论：表述式理由作为行为/内容之外的第三类推荐信号在工业规模可行。

### B-4 附：IMFuse 关键公式速览（配合 Gap-4，便于对照复现）

来源：[IMFuse HTML v2 §3](https://arxiv.org/html/2607.27002v2)。Ê_i = Σ_{l=0}^{L} W^l ⊙ Ẽ_i^l（Eq.8，W∈R^{(L+1)×d} 层×维度权重，Hadamard 积）；深度偏置初始化 w^l ∝ exp(l/(τL))（Eq.9，τ∈{0.2,…,1.0} 步长 0.2）；实例感知 MoE：Ê_i = Σ_m α_i^m·E_i^m，α=softmax(g_i(Ẽ_i))，g 为 2 层 GELU MLP（隐层 h=33），**专家本身即一组逐层权重向量** E_i^m(Ẽ_i)=Σ_l W^{l,m}⊙Ẽ_i^l（Eq.12，非 MLP，规避负载不均/专家饥饿）；结构感知初始化（Eq.13–14）：33 层均分 3 段（K=⌊33/3⌋=11），专家 m 对自己段内 ξ~U(0,1)、段外 ξ~U(0,γ)，γ∈{0.3,0.5}。训练：Adam，lr∈{1e-5,1e-4,1e-3}，weight decay∈{0,1e-6,1e-5,1e-4}，dropout 0–0.5，batch 256，最大历史 10，d=128，≤200 epochs（patience 10，按验证 NDCG@20 早停），4×RTX 5090。参数复杂度：门控 O(dL+L²)、专家 O(MLd)，总时间 O(NMLd)（对比降维网络 O(NDd)，D=4096）。

---

## 三、任务 C：遗漏扫描（2026-07 下旬 ~ 2026-10-26，首轮与顶会篇均未覆盖）

检索方式：多角度 web_search（推理/蒸馏/语义融合×LLM 推荐）+ 逐篇抓取 arXiv abs 页核验日期与内容。以下 5 篇为主选（每篇①–④），另附 2 篇一句话补充与 1 条撤稿警示。

### C-1 WhisperRec / OneLatent：面向基础推荐模型的 latent reasoning（快手）

arXiv [2607.26621](https://arxiv.org/abs/2607.26621)（WhisperRec，v1 2026-07-29，v4 2026-10-22）与 [2610.30675](https://arxiv.org/abs/2610.30675)（OneLatent，v1 2026-10-22；同团队同框架的扩展版，新增生产 serving 系统与线上 A/B）。

- **①思考背景**：LLM 推理能力正被引入基础推荐模型（FRM），但主流"Think-then-Answer"显式 CoT 要生成冗长理由，推理开销巨大；且固定 CoT 模板难以建模多样、动态、上下文依赖的用户兴趣。
- **②核心问题**：如何保留推理对推荐决策的信息贡献，同时摆脱自回归生成理由的延迟瓶颈，让带推理的 FRM 可在线服务。
- **③如何解决**：把教师生成的 CoT 压缩成**少量可学习 latent 推理 token**，走"Latent-Reason-then-Answer"——推理在潜空间完成、不产出文本理由。MV-ACoT（多视角自适应 CoT）从互补视角构造多样高质量监督，并按实例难度自适应推理复杂度（简单实例轻量分析、困难实例多因素推理）；预训练 FRM 之上做三阶段 latent 对齐逐步内化 CoT，再用课程式后训练激活 latent 推理、保持标准推荐能力。
- **④结果亮点**：工业级快手数据集 + 公开 Kuaishou LLM-Rec 基准上全面超显式 CoT 方法：SID@64 较 Think 变体 **+17.44%**、较 No-Think **+9.33%**，在线推理吞吐 **>10×**（WhisperRec v4）/ **>17×**（OneLatent）；OneLatent 版本在快手本地生活广告场景线上 A/B 估计带来 **9.6% 收入提升**（对比含 OneRec、OneReason 在内的强线上基线）。

### C-2 PROVE-REC：可验证的偏好证明（Reasoning with Evidence, Not Merely Rationales）

arXiv [2610.02968](https://arxiv.org/abs/2610.02968)，2026-10-02。

- **①思考背景**：LLM 能从历史与评论推断偏好，但它生成的 rationale 未必反映推荐实际使用的信息——与 SelfDR/Evo-Rec 的"推理质量"关切同源，但把矛头指向**可验证性**：偏好论断可能证据支撑薄弱（grounding 缺陷），也可能对最终排序毫无影响（influence 缺陷），作者合称 **grounding-influence gap**。
- **②核心问题**：如何让 LLM 推荐中的偏好推理"有据可查、确有影响"，而不是看似合理的自圆其说。
- **③如何解决**：两遍（two-pass）架构——Pass A 把完整 pre-target 历史转成紧凑的**偏好证明（preference proof）**：若干正面/回避论断，每条链接到选定的证据条目；Pass B **只用证明及其证据**预测下一物品，杜绝绕过推理路径直接看历史。两个验证目标：证据→证明的 grounding（遮蔽选定证据 vs 遮蔽对照证据，比较效果差）与证明→推荐的影响（移除某论断后目标物品排序边际的下降）；另设排序保持目标留住全历史的有用信息。
- **④结果亮点**：多真实数据集上稳定超过强序列、生成式与 LLM 增强基线，最高 **+7.45%**；受控消融证实两遍架构与两个验证目标各自有效；产出的论断在证据接地性与对推荐的影响力上均显著更强，同时保持排序质量。

### C-3 FLEXRec：紧凑 LLM 的层级出口融合（Empowering Compact LLMs with Fusion of Layer-wise Exits，ICDM'26）

arXiv [2608.17316](https://arxiv.org/abs/2608.17316)，v1 2026-08-18 / v2 08-27；[代码](https://github.com/xurong-liang/FLEXRec)。

- **①思考背景**：LLM-RS 算力上不可持续，紧凑 LLM（≤3B）是出路，但容量不足又催生推理或蒸馏补丁——或增延迟、或依赖大模型，叠加自回归生成后扩展性瓶颈严重；判别式 LLM-RS 可经 embedding 相似度做高效全语料排序，但紧凑骨干表达力与结构自适应性仍受限。
- **②核心问题**：如何在不牺牲高效全语料排序的前提下，把紧凑判别式 LLM-RS 的表达力补上来（与 IMFuse 同问"只用一层够不够"，但目标是效率与容量而非语义坍缩）。
- **③如何解决**：在**多个 transformer 层插入预测头（early exits）**，对各自的得分分布做自适应融合；**AC-Router**（自适应连续路由器）按用户序列动态决定启用出口的数量与身份；新的 **target-k hinge 损失**调节路由稀疏性，控制计算开销。
- **④结果亮点**：Qwen3-1.7B 与 Llama-3.2-3B 两个紧凑骨干、三个真实数据集上取得同类方法 SOTA 精度并保持高效率——用"层间融合 + 动态出口"同时兑现精度与延迟，是 IMFuse 多层语义结论在"判别式 + 紧凑模型 + 推理效率"坐标下的对偶工作。

### C-4 SCoRD：LLM 重排管线的持续蒸馏（Semantic-Assisted Continual Retriever-Reranker Distillation，CIKM'26）

arXiv [2608.19998](https://arxiv.org/abs/2608.19998)，v1 2026-08-20（CIKM'26，DOI 10.1145/3799682.3840906）。

- **①思考背景**：两阶段管线（ID 召回器 + LLM 重排器）常用"重排器→召回器"蒸馏提升召回质量；但真实系统必须持续适应演化的兴趣与新交互——反复更新 LLM 重排器再蒸馏代价不可承受，只更新召回器又因容量有限难以从稀疏数据适应。
- **②核心问题**：非平稳数据流下，LLM 重排管线如何低成本持续进化（continual distillation），而非一次性蒸馏。
- **③如何解决**：引入**语义推理助手**，把 LLM 推断用户底层意图的能力蒸馏成**可复用的意图级指导**；只在**低置信序列**上选择性蒸馏重排器知识；后续**仅更新召回器**（不再反复调 LLM 推理）；并把召回器表示与意图漂移信号回喂重排器，形成检索-重排协同适应闭环。
- **④结果亮点**：真实数据集上实现有效且高效的 retriever-reranker 协同适应——把"蒸馏"从一次性知识搬运改造成持续自进化机制，直接命中方向 E 的"蒸馏"子线在流式场景的缺口。

### C-5 MARI：结构化决策记忆的蒸馏（Beyond Sequences: Distilling Structured Decision Memory for LLM Recommendation）

arXiv [2610.11501](https://arxiv.org/abs/2610.11501)，2026-10-08。

- **①思考背景**：现有方法多建模单一行为类型；即便引入多行为，也把异构动作**压平成同质 token 序列**，抹掉其不同决策角色——无法捕捉"价格 vs 质量"这类权衡中的语义层级与上下文细微差，导致高度相似物品的"困难选择"场景性能退化。
- **②核心问题**：如何在可解释、可扩展的前提下，让 LLM 推荐建立在**显式、结构化的决策证据**之上，而非原始长序列。
- **③如何解决**：维护**决策记忆库（DMB）**，把用户过往 rationale 归档为**结构化决策记忆（SDM）**——目标、约束、权衡的简明记录；SDM 由 **Post-Hoc Decision Distillation** 从异构行为与用户生成内容离线生成；在线检索相关 SDM 增广 LLM 推理，记忆构建与在线推理解耦，免于处理长原始序列的高成本。
- **④结果亮点**：在标准下一物品预测与新提出的 **Difficult Choice Prediction** 任务上显著超 SOTA，延迟开销低，且给出可读的用户决策洞察——"蒸馏"的对象从模型输出扩展到"决策结构"本身，与 SARA 的作者级理由画像互为镜像（SARA 聚合到作者侧，MARI 落到用户决策侧）。

### 补充（一句话级）

- **LARK（Latent-Aligned Reasoning for Multimodal Recommendation）**：arXiv [2609.04645](https://arxiv.org/abs/2609.04645)，2026-09-04。发现多步推理中视觉/文本信号逐步衰减的 **cross-modal dilution**；在单 VLM 内做两阶段潜推理——潜 token 与冻结视觉编码器对齐充当"视觉检查点"，桥接 MLP + 物品间对比学习、中间特征与 CoT 隐状态对齐防推理语义淡出；三个公开基准 + 一个工业数据集 SOTA。
- **Distill Globally, Adapt Locally（trade-up 推荐的推理蒸馏 + 测试时训练）**：arXiv [2609.05363](https://arxiv.org/abs/2609.05363)，2026-09-04，GenAIECommerce@RecSys'26 workshop。RAG 少样本 LLM 教师产出结构化关系标签+理由，监督 15.5M 参数嵌入对分类器（推理蒸馏版 AUC 0.924 vs 纯标签版 0.912）；产品类型测试时训练（PT-TTT）再提至 AUC 0.941；10 万对代理目录上比直接 LLM 推理快约 5000×、成本约低 10000×。

### ⚠️ 撤稿警示：P3Rec

**P3Rec: Distilling Prior–Posterior Preference Reasoning for LLM-based Recommendation**（arXiv [2609.13993](https://arxiv.org/abs/2609.13993)，v1 2026-09-12）**已于 2026-10-17（v4）被作者撤稿**，理由："Due to unresolved disagreements among the authors regarding the content and public dissemination of this manuscript"（作者间就内容与公开发表存在未解决的分歧）。其"先验-后验偏好推理双视角蒸馏 + 兴趣熵自适应校准"的思路虽有参考价值，但**不建议引用或作为依据**，后续调研若再遇到需注明撤稿状态。

---

## 四、来源清单

**保留（一手来源，均抓取成功）**
- SelfDR：[arXiv abs 2609.03313](https://arxiv.org/abs/2609.03313) + [HTML 全文 v1](https://arxiv.org/html/2609.03313v1)（公式 Eq.1–9、Table 3 Hg@1 全量数值、§4.1.4 实现细节、§5.1 超参敏感性、结论节未来工作表述）+ [GitHub raw README](https://raw.githubusercontent.com/JiangDeccc/SelfDistillation/main/README.md)（复现脚本流程；github.com 页面本身 fetch failed，已标注）
- Evo-Rec：[arXiv abs 2609.29973](https://arxiv.org/abs/2609.29973) + [HTML 全文 v1](https://arxiv.org/html/2609.29973v1)（三阶段公式 Eq.1–6、§4.1.3 实现细节含 GPT-5.4/K=5/G=16/B=10、消融与 §5 机理分析）+ [GitHub 仓库](https://github.com/mengdanzhu/evo-rec)（克隆核读：6 步流水线、数据集统计、checkpoint 发布状态）
- SARA：[arXiv abs 2609.17639](https://arxiv.org/abs/2609.17639) + [HTML 全文 v2](https://arxiv.org/html/2609.17639v2)（§2 数据引擎统计、§3 SFT/QR-DPO 公式、§4 Judge 框架与 MAE 表、§5 Ranker 双路公式 Eq.12–23、§6 全部实验表、§8 附录评分细则）
- IMFuse：[arXiv abs 2607.27002](https://arxiv.org/abs/2607.27002) + [HTML 全文 v2](https://arxiv.org/html/2607.27002v2)（Eq.1–14 公式、§4.1.4 超参、Table 2 主结果、§4.4 效率、附录 A.1–A.4）
- 新论文 abs 页（日期与内容核验）：[2607.26621 WhisperRec](https://arxiv.org/abs/2607.26621)、[2610.30675 OneLatent](https://arxiv.org/abs/2610.30675)、[2610.02968 PROVE-REC](https://arxiv.org/abs/2610.02968)、[2608.17316 FLEXRec](https://arxiv.org/abs/2608.17316)、[2608.19998 SCoRD](https://arxiv.org/abs/2608.19998)、[2610.11501 MARI](https://arxiv.org/abs/2610.11501)、[2609.04645 LARK](https://arxiv.org/abs/2609.04645)、[2609.05363 Distill Globally](https://arxiv.org/abs/2609.05363)、[2609.13993 P3Rec（撤稿）](https://arxiv.org/abs/2609.13993)

**剔除**
- mastoxiv.page（cs.IR 机器搬运帖）、searcharxiv.org 与 arxiv.science（arXiv 镜像聚合站）——与 abs 页重复，非一手来源
- doi.org/10.48550 镜像页——与 arXiv abs 重复
- ReasonRec（arXiv 2606.28357，2026-06-08）、MMP-Refer（2604.03666）、AdaM-Rec（2609.38455）、IntentLens（2611.11346）、Fine-grained Semantics Integration（2602.22632）等——窗口外或偏离方向 E 核心（多模态路由/查询理解/早期工作），不纳入主列表

---

## 五、剩余 Gaps（本轮仍未确认，均注明原因）

1. **Evo-Rec 主表 Office/Industrial 逐格数值未获取**：arXiv HTML→文本转换中表格仅渲染为题注（Table 2 等），运行文本只给出 Games 绝对值与全域相对提升；Office/Industrial 的 R@K/N@K 逐格数字需 PDF 版核对。同理 SelfDR Table 2（Hl@K/Nl@K 排序质量逐格值）仅题注，只有 Table 3（Hg@1）完整。
2. **Evo-Rec Stage-3 GRPO 是否带 KL 正则未完全确认**：仓库脚本名 `RL_training_script_no_kl.sh` 暗示无 KL 项，正文只写 "implements GRPO"（未提 KL 系数）；需读脚本源码定论。
3. **SelfDR 代码级超参未逐文件核读**：github.com 仓库页 fetch failed（经 raw README 成功），论文称 "Further details on baseline implementation and hyperparameters are provided in our repo"，但 README 未列学习率/epoch 等具体值，需克隆后读 `explan.py`/`joint_all_adaptive.py`。
4. **Evo-Rec 部分数据与 checkpoint 未发布**：仓库中 `general_reasoning`、`seqrec` 训练/验证、`reasoning`、`cot` 标注 "to be released upon acceptance"，当前仅评测协议与 SID 目录结构可用，完整复现受限。
5. **SARA-7B 的训练超参（学习率/epoch/GPU/温度）论文未披露**：正文只给数据规模（187,532 SFT / 7K QR-DPO 作者）与流程公式；复现需联系作者。
6. **OneLatent v2 时间戳异常**：abs 页显示 v2 为 2026-11-02（晚于今日 2026-10-26），属索引边界现象（与总览文件 G 节同类问题），建议以 v1（2026-10-22）为准复核。
7. **任务 C 五篇新论文为摘要级调研**：PROVE-REC / FLEXRec / SCoRD / MARI / WhisperRec(OneLatent) 的四段式内容基于 abs 摘要（WhisperRec 另有 v4 abs 全文确认），未读各自 HTML 全文，超参、数据集名与逐项数字待下轮展开；FLEXRec 的 GitHub（xurong-liang/FLEXRec）已见链接未核读。
8. **窗口内逐日 arXiv 扫描仍为检索式抽样**：10 月中下旬 cs.IR/cs.LG 或仍有零星遗漏（如 2610.3xxxx 段），建议对 arXiv listing 做逐周补扫。
