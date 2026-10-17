学习 INFER _RSI 工作区，/workdir/sunquan20/INFER_RSI/.rick/draft/rfc/rfc-superopt-重构为rick-super模式.md

将此 RFC 在 rick 中实现。
---

## Grilling 澄清结论（2026-10-10，design-tree.md 全量落盘 + grilling_gate 通过）

### 目标
在 rick 中实现 RFC《superopt 重构为 rick-super 模式》：新增 `rick super` 三层架构（Go 监督树→py 进化调度器→pi agent 池），具备 events.jsonl 单写者真相源、级联评估门+软门禁、提案-裁决分离、软硬淘汰、门先证红；以 gen_env skill 现场生成的矩阵乘算子 env 端到端跑通进化循环。

### 关键裁决（human 全部确认）
1. **范围**：完整体系骨架（阶段 0 埋点+阶段 1 c 方向+阶段 2 a/b+阶段 3 最小闭环）；P1/P2 真实统计与参数定标留首跑；教学双环留后续。
2. **代码归属**：rick 仓内新顶层目录 `super/`（py stdlib-only 零依赖）+ Go 侧 internal/cmd/super.go + internal/super/；v5 留 INFER_RSI 只读参照。
3. **会话模型**：`rick super`=中控会话（照 easy 模式，默认加载 super-loop + gen_env skill）；`rick super start [--daemon]`/`stop`/`status`/`event`；干预走队列保单写者。
4. **调度策略（human Q6 原话细化）**：四状态循环——adjust agent（v5 strategy agent 进化版：格增删/正交性；**不删格，采样权降 0=功能性灭绝**）→ UCB 采样格子（visit 真实递增；冷启动=先验权重加权随机）→ builder agent（选父格→idea→实验计划）→ coding agent（执行+env 门禁自检）→ 门禁不符→ debug agent；**框架入格前独立验证，以框架为准**。
5. **openevolve 终裁**：自研调度器 + 机制借鉴（级联评估/评估器协议/TemplateManager/artifacts 侧通道/IslandSelector 快照合约 本期落；岛/并行/checkpoint 后续）。理由：openevolve 无 adjust 角色、主循环零 agent 编排缝、builder/coding/debug 无原生载体、fork 量级 ~1100-1800 行恰集中在我们的核心诉求。
6. **门禁职责（L3-Q2 架构修正）**：门禁/容差/评分函数/阈值属于 **env 设计范畴**；rick 只提供门禁框架（级联门结构/门先证红/双回归/多计时中位数机制）；gen_env skill 引导用户设计门禁系统。
7. **成本账本**：时间（wall-clock）+ token 双轨；token 来源=聚合 pi `message_end.message.usage`（research-L3-pi-token），按 agent role 划分。
8. **验收**：mock e2e 全绿 + realpi 冒烟≥3 真迭代 + 双回归 harness + 污染注入演练（假阳性→软降权→复测→出档/复活全链一例）。

### 数据结构/存储结构（详见 design-tree.md 第 2/3/4 层）
- 存储布局：`~/.rick/super/<name>/`（super.json/env/events.jsonl/archive.json/interventions/codes/gates/prompts）
- events.jsonl 16 类事件类型（append-only 单写者）；archive.json 精英格档案（cells+pending+meta，代码存 codes/{uuid} 引用）
- agent 合约：六字段基座 + 四角色增量（adjust{proposals}/builder{experiment_plan}/coding{diff_path,self_check}/debug{fix_report}）
- 级联门：stage1 环境健康→stage2 正确性→stage3 性能+软门禁（K=3 中位数）；门先证红（exploit 拒识 100%）+ 双回归
- 触发节奏：adjust interval N=5（可配）、debug max_attempts M=3
- 预算：无迭代上限 + --max-wall-time 24h + 手动 stop
