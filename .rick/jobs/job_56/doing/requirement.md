/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_54/doing/exploration/anchors.md

学习这里关于 super mod的原创想法，基于此处讨论的设计，重构super mod 提升效果。
---

## Grilling 澄清结论（2026-11-30 · v2 定稿）

**目标升级**：从 v1「adjust 5-subagent 协议重构」升级为 human 原创文档定义的**四项系统性重构**（grilling 两轮追问后定稿，裁决记录见 grilling/design-tree.md 追问记录）：

1. **adjust 7 步 subagent 协议**：问题还原（含 humans 表未应用干预=human 价值判断）→ 视角生成（3 相关+3 不相关关键词→research 搜索→3 视角+归纳演绎证明）→ 矛盾生成（每视角 1 个并行 subagent→活跃约束集→影子价格最大约束）→ 主要矛盾判断（全局性/根本性/决定性）→ 辩证逆转 → 解空间自改进（格动作+次要矛盾多样性配权+结构化 JSON 过 pi hook 结构检查）→ 实验自改进（bug 解法→domain_knowledge 直接写；skill/验证器反思→提案队列）。
2. **sqlite 元信息 DB 化**：五张表（特征格/实验/humans[两类队列：干预+提案]/events/资源[实验槽+环境槽]）；实验 ID=迭代号-年月日时分时间戳（**无纪元**）；py 调度器进程锁（env 下 PID 锁，一会话一 env 唯一调度器）；py/Go/tools 全部 SQL 查询，json/jsonl 废弃，旧工作区保留不迁移。
3. **sub loop 实验执行确定性强化**：builder（风险推演段必做+可选 L1 验证预检+证伪性论证）；coding（**去 TDD**，信任一次写对）；debug（完整实验记录：设计+过程+结论+反思，格式过 pi hook；涨幅过大查作弊/验证机制；**主动上报失败→调度器跳过端到端验证**）。并行实验是核心目标：调度器仅管资源分配（实验槽+环境槽二级限额），同环境分时复用是 ENV 设计责任。
4. **开放 env 构建体系**：env 四部分（domain_knowledge.md 领域专家知识=agent 准则入口 / skills[profile·benchmark·verify·research 四基本+可扩展，注入强调务必调用] / gates/ 目录[builder·coding·debug 禁读，adjust 可读写，verify·benchmark 直接调用引用] / code 基线）；**多维优化目标**（约束+多追求指标：tp99/qps/ttft/tpot 等；分数=加权平均融合公式；stdout JSON；score<0=拒绝）；super loop 四步构建（设计树追问→四 skill 构建方法→演练→验证金字塔分层参数 L1-M0/L{n}/E2E）；subagent 超参数可配（model 后端+thinking 档位承载温度语义）。

**DAG 分层**：L1 = M1(DB 地基) ∥ M2(env 体系) → L2 = M3(7 步协议) ∥ M4(sub loop 执行链) → L3 = M5(集成验收)。

**验收**：全量单测+e2e（mock_pi 7 步场景）+现有测试迁移；gate_check 负分协议、debug 跳过验证、进程锁、gates 禁读拦截均有测试覆盖。
