# task3 — M3 adjust 7 步 subagent 协议：流水线调度 + 7 角色模板 + hook 结构检

> job_56 · DAG L2（单 task 层）· 依赖 task1（humans/events 表）+ task2（env 结构/超参数配置位）
> 源设计：grilling/design-tree.md KR3（含 Q3/Q4/Q5 裁决）

# 任务目标

把 loop.py 的 adjust 段从「单 agent 单次调用」重构为 7 步 subagent 串行流水线（step3 内三视角并行 subagent），每步独立上下文（answer-isolation）、产物落盘传递；humans 表未应用干预注入 step1（human 价值判断）；subagent 超参数（model+thinking）从 env.json 读取派发；json-guard hook 扩展为结构化 schema 检 + gates/ 禁读拦截。

# 关键结果

1. **KR-A【7 步流水线调度】**：`loop.py` 的 `_maybe_start_adjust` 重构为流水线调度器（后台线程内串行执行 7 步；每步一次 `client.run`（step3 为 3 次并行））：
   - **step1 问题还原**（`adjust_fact`）：ctx=区间实验记录+各格改进序列+events 摘要+**humans 表未应用干预（queue_type='intervention', applied=0——human 价值判断）**+饱和信号计算证据 → 产物：事实与期望文档（`facts.md`：现状事实+human 期望+已证伪/已证实假设清单）
   - **step2 视角生成**（`adjust_perspective`）：ctx=facts.md+进化记录摘要 → 产物：3 相关+3 不相关关键词→（用 env 的 research skill 检索配置）外部搜索→**3 个视角描述**（每个=理论解释+基于事实与调研的归纳/演绎证明链+信源）→ 落盘 `perspectives.md`
   - **step3 矛盾生成**（`adjust_contradiction`，**每视角 1 个并行 subagent**：3 线程同模板不同视角参数）：ctx=单视角+facts.md → 产物：该视角下的活跃约束集+影子价格最大的约束=主要矛盾候选+严谨证明 → 3 份产物合并 `contradictions.md`
   - **step4 主要矛盾判断**（`adjust_judge`）：ctx=3 矛盾+事实 → 产物：逻辑判定排序（全局性/根本性/决定性+影子价格最大化论证）→ `judgment.md`
   - **step5 辩证逆转**（`adjust_reversal`）：ctx=主要矛盾+judgment → 产物：辩证逆转方法（矛盾→手段）→ `reversal.md`
   - **step6 解空间自改进**（`adjust_space`）：ctx=judgment+reversal+格现状 → 产物：格动作（未覆盖→新格[正交性+详细优化方法描述]/潜力降→降权/潜力升→升权/迭代饱和→权重0/**2 个次要矛盾多样性配权**）+**结构化 JSON 输出（proposals 契约，过 pi hook 结构检）**；特征格元信息含详细优化方法描述（cells 表 method_desc 写入）
   - **step7 实验自改进**（`adjust_env`）：ctx=区间实验记录+env 状态 → 产物：bug 及解法→**直接写** `env/domain_knowledge.md`（追加节）；skill 修改/验证器反思（机制性错误/噪声/过拟合）→**push_proposal 到 humans 表提案队列**（queue_type='proposal'，待 human 裁决）
   - 步骤产物统一落盘 `codes/adjust/iter_<N>/step<k>_<name>.md|json`；每步失败降级：**跳过+默认值继续+events 记 `adjust_step_degraded`**（不整链重跑——research-L1 结论 9/10）；step6 失败（无 proposals）→ 沿用现状（空提案合法+框架合成 decisions 兜底）
2. **KR-B【humans 表消费】**：step1 ctx 注入未应用干预（human 价值判断——期望锚点）；humans 表 proposal 队列的 push API 消费方=step7（task1 落表+API，本 task 接入）。
3. **KR-C【subagent 超参数派发】**：`env.json` 的 `agent_hyperparams`（每角色/每步 {model, thinking} 配置，Q3 裁决）→ `client.run` 派发时映射为 pi CLI `--model <id>:<thinking>`（或 `--model` + `--thinking` 两参）；默认无配置=跟随 adjust 模型档。config 合并链：env.json `agent_hyperparams` > super.json `agent_models` > 默认。
4. **KR-D【7 角色模板】**：`super/prompts/` 新增 7 个模板：`adjust_fact.md`/`adjust_perspective.md`/`adjust_contradiction.md`/`adjust_judge.md`/`adjust_reversal.md`/`adjust_space.md`/`adjust_env.md`——每个含：角色使命（**尽可能简化，保留最重要的 SENSE 步骤描述**——human 原话：整个协议的 prompt 尽可能简化）/输入材料（上游步骤产物文件路径+ctx 占位符）/输出契约（产物文件+JSON 块）；`prompts.py` ROLES 扩展（模板查找链零改动——新角色自动支持三级覆盖）；原 `adjust.md` 保留为兼容回退（`adjust_mode` 配置：`pipeline`[默认]/`legacy`）。
5. **KR-E【json-guard hook 扩展】**：`.rick/skills/superopt-json-guard/index.ts`（部署源）扩展两类检查（保持既有 JSON 块检查兼容）：
   - **结构化 schema 检**（step6 proposals）：adjust_space 角色的最终回复 JSON 按字段校验（type/action/target_cell/evidence 必填；new_cell 必含 name/method/orthogonality_rationale）→ 缺字段走既有 followUp 修复轮
   - **gates/ 禁读拦截**：builder/coding/debug 角色（按 prompt 文件名或 SUPEROPT_ROLE 环境变量判定）的 read/bash 工具调用命中 `<ws>/env/gates/` 路径 → hook 拒绝该调用（返回错误消息提示禁读纪律）；adjust_* 角色放行
6. **KR-F【decisions schema】**：`decisions/iter{N}.md` 升级为「markdown 人读骨架+内嵌 schema JSON 块」（叠加格式——v1 既定设计）：骨架七节（事实与期望/视角/矛盾/判定/逆转/格动作/实验自改进）+内嵌完整 adjust-protocol schema（primary_contradiction{statement,logic_basis,fact_confidence}/secondary×2/perspectives_used/critique_ranking/cell_actions/saturation_evidence_consumed）；`_ensure_decision_file` 合成兜底同步升级；step6 负责落盘。
7. **KR-G【事后验证注入】**：新增事后验证计算（`_hindsight_evidence`：被升权格后续产出/被杀格事后引用/新格存活——读 DB 实验表+cells 表），结果注入下一轮 step1 ctx（主要矛盾判断对错的证据）。
8. **KR-H【测试】**：test_loop 新增流水线断言（7 步调度顺序/step3 并行三线程/失败降级事件/humans 注入/decisions schema 落盘）；test_prompts 新增 7 角色渲染断言；mock_pi 新增 adjust 流水线场景（由 task5 终验，本 task 先写单测级 mock）。

# 写域（单 task 层，无并行冲突）

```
super/superopt/loop.py（adjust 段：_maybe_start_adjust/_collect_adjust/_join_adjust/_own_decisions/_ensure_decision_file/_decisions_path/ctx 组装[adjust 部分]/新增 _run_adjust_pipeline/_adjust_step_*/_hindsight_evidence）
super/superopt/prompts.py（ROLES 扩展）
super/prompts/{adjust_fact.md, adjust_perspective.md, adjust_contradiction.md, adjust_judge.md, adjust_reversal.md, adjust_space.md, adjust_env.md}（新建）
super/prompts/adjust.md（保留兼容+adjust_mode 文档）
.rick/skills/superopt-json-guard/{index.ts, SKILL.md}
super/tests/{test_loop.py, test_prompts.py}
```

**禁改**：loop.py 实验链函数（`_step_locked/_slot_worker/_dispatch_agent/_apply_proposal/_loop_parallel`——task4 职责；存储行已由 task1 完成）；gates.py（task2）；builder/coding/debug prompts（task4）。

# 实现要点

1. **流水线线程模型**：沿用 `_adjust_thread` 后台线程单飞模式——线程内串行 7 步（每步 client.run 子进程）；step3 内 3 个**并行子线程**（同模板不同视角；收集 join 后合并产物）；`_adjust_lock` 语义不变（同刻至多 1 个流水线在飞）。
2. **每步独立上下文**：步骤产物只经文件传递（step<k> 产物路径注入下一步 prompt ctx）——research-L1 结论 2/5（JSON/结构化锚定+isolation）；叙述类产物（facts/perspectives/contradictions）用 markdown+关键锚定字段（视角列表/矛盾 statement）——混合契约（v1 既定）。
3. **超时预算**：per-step timeout（config `adjust_step_timeout_s` 默认 900s）+ 流水线总预算（`adjust_total_timeout_s` 默认 5400s）；超时中断于当前步——已完成步骤产物保留，step6 未跑则用已完成部分+默认值（保守空提案）合成。
4. **humans 注入格式**：`[{id, content, created_at}, ...]` 注入 step1 ctx（「human 价值判断与期望——尚未应用的干预」）。
5. **step7 写入边界**：直接写 domain_knowledge.md 仅限**追加**（`## adjust 实验知识沉淀 · iter_<N>` 节）——不修改既有内容；验证器/skill 反思**一律提案**（humans 表 push_proposal）——防测量地基污染（Q4 裁决）。
6. **json-guard 角色判定**：superopt PiAgentClient 派发时已注入 `SUPEROPT_JSON_GUARD=1`——增加 `SUPEROPT_ROLE` 环境变量（client.run 派发时按 role 注入）；hook 按 role 白名单（builder/coding/debug）拦截 gates/ 路径。
7. **事件可观测**：每步 `adjust_step_started/ended/degraded`（events 表——task1 已 DB 化）+ 流水线总 `adjust_dispatched/adjust_decision` 事件语义保留。
8. **兼容回退**：`adjust_mode: legacy` 走原单 agent 路径（旧 prompt/旧 ctx 组装函数保留但标记 deprecated 注释）——迁移期可选。

# 测试方法（自测，过程性）

- RED：test_loop 先写流水线断言（mock client 按 step 返回预置产物；断言 7 次派发顺序/step3 三线程/产物文件链/降级路径）先跑红
- GREEN：实现至全绿；test_prompts 断言 7 模板占位符渲染（缺占位不炸——prompts.py 既有语义）
- hook 自测：node/tsx 侧无测试基建——用 mock 场景手动验证（部署到临时 extensions 目录跑一次 pi --mode json 验证拦截）+ SKILL.md 记录验证步骤
- 回归：`python3 -m unittest discover -s super/tests -p 'test_*.py' -v` 全绿（test_loop 既有断言不回归——legacy 路径保留）

# 验收（gate2.py 承担）

`python3 .rick/jobs/job_56/plan/gates/gate2.py`：test_loop（含新增流水线断言）+test_prompts 全绿 + 7 模板存在+输出契约结构断言 + json-guard 扩展结构断言（schema 检+gates 拦截代码存在）+ decisions schema 合成兜底断言。
