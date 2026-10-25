# task40：四模板重写（实验驱动 loop + skill 声明）

# 任务目标
四角色提示词按新 loop 重写，并声明各自的 skill 清单（路径 + 触发条件）。
**ctx 键契约**（task39 提供）：adjust=`experiment_records/cell_descriptions/decision_history/research_brief`；builder=`target_record/neighbor_samples/research_report_path/experiment_record_path`；coding=`research_report/experiment_record/coding_plan_path/test_path`；debug=`experiment_record/research_brief(focus=用户手册)/gate_failure/perf_gain_pct`。

# 关键结果
- `super/prompts/adjust.md`：四步 loop——①全局实验分析（读 `experiment_records`，必要时用 read 打开 `experiment_sessions` 里的 pi session）②SENSE（定主要矛盾：全局/根本/决定性的性能成因）③research（按 `research_brief` 的 method/sources/focus/suggested_queries 检索，找启发式 idea；≤3 篇、必须落可实现动作）④格管理（new_cell/modify_cell/权重 0=删除；正交性论证）。**skill 声明**：`super_experiment_analysis`/`super_sense`/`super_cell_management` 路径 + 触发时机（按需 read）。**输出契约**增 new_cell（name/method/source_refs/orthogonality_rationale）、modify_cell（target_cell/method/source_refs）；要求把思考过程写进 `decisions/iter{N}.md`（路径由 ctx 给出）
- `super/prompts/builder.md`：读格 description + 精英实验记录 + **两个随机相邻格**（description/精英 code/实验记录）+ history_failures/axis_perf/profile_history → `research`（brief，关注点=实验细节/数据/方法）→ 写 `research_report.md`（现有代码调研 + 业界落地方法 + 论文实验细节）→ `super_experiment_design` skill → 写 `experiment_record.md` 六段 → 输出 experiment_plan（含 hypothesis/validation_logic）。skill 声明 + 落盘路径（ctx 给）
- `super/prompts/coding.md`：读 builder 调研报告 + 实验记录 → 细化 research（落地细节/风险卡点，role=coding brief）→ 写 `coding_plan.md` → **TDD 纪律**（先写 `test_solution.py` → 看到红 → 实现 → 绿 → 重构 → 保绿；引用 rick tdd canon）→ 卡点则按 `super_debug_protocol` 自查 → 完成后**追加实验记录「执行过程与结论」段**（实现要点/测试红绿/门禁结果）
- `super/prompts/debug.md`：三类触发说明（自检失败/框架门禁失败/异常收益或容差带内复验）+ 流程=复现 → research（brief，focus=用户手册与正确用法）→ **三个最可能失败原因逐一验证**（每个写「若成立应观察到什么」）→ 修复或 `verdict: keep_original` → 追加实验记录「卡点/根因/解决」段
- 四模板统一：`{{...}}` 占位符、无硬编码路径、skill 清单段（名称+路径+触发条件）

# 写域
- super/prompts/adjust.md
- super/prompts/builder.md
- super/prompts/coding.md
- super/prompts/debug.md

# 依赖关系
- 无（L2，写域与 task38/39 互斥；键名按上表契约，缺失键宽松渲染兜底）

# 测试方法
四模板含各自 ctx 键与 skill 声明；`grep 无 /Users/`；`python3 -m unittest super.tests.test_prompts` 绿

# 禁止事项
- 不动框架；不改 env_9 的 env/prompts
