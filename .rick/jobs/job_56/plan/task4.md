# task4 — M4 sub loop 实验执行链强化：三角色协议重写 + 调度确定性 + 失败跳过验证

> job_56 · DAG L3（单 task 层）· 依赖 task3（loop.py adjust 段已定型，避免同文件并发写）
> 源设计：grilling/design-tree.md KR4（含 Q8/Q9 裁决）

# 任务目标

实验执行链（builder→coding→debug）协议重写与调度确定性强化：builder 增加仿真预演（风险推演段必做+可选 L1 验证预检+证伪性论证）；coding 去 TDD（信任一次写对）；debug 完整实验记录协议（四节事实记录+格式过 pi hook 完整性检查+主动上报失败）；调度器 sub loop 读实验记录写回 DB；debug 标记失败→跳过端到端验证加速迭代。

# 关键结果

1. **KR-A【builder 协议】**：`super/prompts/builder.md` 重写——输入材料新增：分配的实验 ID（`<iter>-<时间戳>`）+资源槽位信息（实验槽位号+运行环境槽位）+运行环境信息（env/路径+分时复用规则提示）+特征格优化方法描述（cells 表 method_desc，历史 diff 引用）。工作流：①读 env skill+领域专家知识（domain_knowledge.md 入口）②按特征格 method_desc 调 research skill 查询+读源码事实调研+读该格历史实验 code diff ③设计实验计划 ④**仿真预演段（必做）**：基于运行环境推演实验执行过程可能遇到的风险点+每个风险点的预防/解决措施 ⑤**可选 L1 验证预检**：对关键假设跑 verify skill 分层参数（L1-M0/L1 级）分钟级预检 ⑥**证伪性论证**：无论实验成功失败都可获得足够信息增量（假设+证伪条件写明）。
2. **KR-B【coding 去 TDD】**：`super/prompts/coding.md` 重写——**移除 TDD 强制纪律**（先写测试/看红/最小实现循环+tdd JSON 字段+「作假比失败更糟」的 TDD 语义），改为：信任一次写对——基于实验计划直接实现，实现后**自验**（跑 verify skill 的快速验证级[L1]自查关键断言，非 TDD 流程）；实验记录第 6 节「实现与测试」改为「实现与自验」（做了什么改动+自验结果）。
3. **KR-C【debug 完整实验记录协议】**：`super/prompts/debug.md` 重写——四类触发（自检失败/门禁失败/可疑收益/容差带内）保留；新增职责：①检查实验过程不合理处（是否符合实验计划）②**涨幅过大→检查验证机制问题或实现代码作弊行为**（引用 gates 禁读纪律违规判定）③实验失败→找到根因+反思。**实验记录=本次实验全部事实记录**：完整实验设计（引用 builder 计划+偏差说明）/执行过程（关键操作+中间结果）/实验结论（成功/失败+证据）/反思（根因/教训/信息增量）——四节完整，**格式过 pi hook 完整性检查**（四节标题结构校验）。**主动上报失败协议**：debug 反思后确认实验失败 → 最终 JSON 标记 `experiment_failed: true` + 完整实验记录落盘——**无需再做检查**（调度器跳过端到端验证）。
4. **KR-D【失败跳过验证】**：`loop.py` 实验链调度：debug agent 返回 `experiment_failed: true`（或 coding 阶段就失败且 debug 确认）→ 调度器**跳过 gate_check.py 端到端验证**（省部署/预热/测试全流程），实验表直接记 `score=NULL`+`status='failed'`+实验记录 path 写回——加速迭代（置信：debug 已完成反思确认）。正常路径：调度器执行 gate_check.py（E2E：基于 code 部署→预热→端到端测试→融合分）。
5. **KR-E【sub loop 写回 DB】**：`loop.py` sub loop（槽 worker 内实验链）结束时：读实验记录 path → 实验表 UPDATE（record_path/session_id/score/status）——task1 已落表结构+task3 已定 adjust 消费，本 task 接通实验链写回；`experiment_record.py` 配合四节结构模板（记录仍是 markdown 文档+frontmatter 元信息，DB 存 path 引用）。
6. **KR-F【资源槽位 ctx 注入】**：三角色 ctx 新增：`experiment_id`（新编号）/`slot_info`（实验槽位号）/`env_slot_info`（运行环境槽位+分时复用规则提示）——builder/coding/debug prompt 占位符对接。
7. **KR-G【测试】**：test_loop 新增实验链断言（失败跳过验证路径/DB 写回/ctx 注入占位符）；mock_pi 场景适配（ok_builder 写计划含风险推演段/ok_debug 失败场景标 experiment_failed）。

# 写域（单 task 层，无并行冲突）

```
super/prompts/{builder.md, coding.md, debug.md}
super/superopt/experiment_record.py（四节结构模板+frontmatter）
super/superopt/loop.py（实验链段：_step_locked/_slot_worker/_loop_parallel/_dispatch_agent/gate 调用段/_apply_proposal 仅存储行适配——流程重构限实验链）
super/tests/{test_loop.py, test_experiment_record.py, mock_pi.py}
```

**禁改**：loop.py adjust 流水线函数（task3 已定型——`_maybe_start_adjust/_collect_adjust/_run_adjust_pipeline/_adjust_step_*` 等）；gates.py（task2）；adjust 系 prompts（task3）；db.py/events.py 等（task1 已定型，仅消费 API）。

# 实现要点

1. **prompt 简化原则**（human 原话：prompt 尽可能简化，留最重要的步骤描述）——builder/coding/debug prompt 全文控制在既有量级以内：删除 TDD 长节、删除冗余纪律段，保留角色使命/输入材料/工作流/输出契约四段核心。
2. **experiment_failed 判定链**：coding 自检失败→debug（attempt 语义保留）→debug JSON `experiment_failed:true` → 跳过 gate_check；debug 修复成功→正常 gate_check。门禁失败（gate_check 跑了但 stage 拒绝）→ debug 修复 1 次→重跑门禁（既有语义保留）。
3. **实验记录四节结构**：`# 实验记录 · <experiment_id>` + `## 1. 实验设计`（引用+偏差）+`## 2. 执行过程`+`## 3. 实验结论`+`## 4. 反思`（根因/教训/信息增量）+frontmatter（experiment_id/cell_id/hypothesis/status/score）；json-guard hook 的实验记录完整性检查在 task3 已落（四节标题校验）——本 task 保证模板与 hook 断言一致。
4. **ctx 占位符新增**（prompts.py 的 ctx 键约定注释同步）：`experiment_id`/`slot_info`/`env_slot_info`。
5. **gate_check 调用点**：`_step_locked` 内 gate 调用保持 CascadeGate 接口（task2 兼容设计）——失败跳过仅是条件分支（debug 确认失败→不调 gate）。
6. **作弊检查与 gates 禁读**：debug prompt 的「可疑收益」触发段引用「若发现实现存在针对验证器的定向优化嫌疑（读 gates/ 违规已在 hook 拦截层记录）→ 标记作弊」——事件 `gates_read_blocked`（task3 hook 落）可作 debug 证据输入。

# 测试方法（自测，过程性）

- RED：test_loop 先写断言（debug 上报失败→events 无 gate_check 调用+实验表 status=failed；正常路径 gate_check 调用；DB 写回 record_path）先跑红
- GREEN：实现至全绿；test_experiment_record 四节结构断言
- prompt 结构断言：builder 含「仿真预演/风险点/证伪」节；coding 无「TDD/先写测试/看红」强制；debug 含四节记录契约+experiment_failed 协议
- 回归：`python3 -m unittest discover -s super/tests -p 'test_*.py' -v` 全绿

# 验收（gate3.py 承担）

`python3 .rick/jobs/job_56/plan/gates/gate3.py`：test_loop（实验链断言）+test_experiment_record 全绿 + 三 prompt 结构断言（builder 仿真预演节/coding 无 TDD/debug 四节契约+失败协议）+ 失败跳过验证行为断言。
