# task24：四模板升级（饱和/认输/门禁定义/台账/可移植）

## 任务目标
ctx 键由 task23 提供（契约固定如下），模板侧把新信息用起来并消灭硬编码。
**ctx 键契约**：builder=`env_dir/perf_profile_cmd/self_check_cmd/saturation/domain_brief/perf_timeline/budget_status/ledger_brief/gate_defs`；coding=`gate_defs/perf_baseline/workdir_files/ledger_brief/saturation`；debug=`perf_profile_cmd/gate_defs/ledger_brief/saturation/gate_failure`；adjust=`saturation/perf_timeline/budget_status/domain_brief/ledger_brief/own_decisions`。

## 关键结果
- `super/prompts/builder.md`：
  - 输入材料增：`domain_brief`（领域知识摘要——**不必再读文件**）、`perf_timeline`（性能时间线）、`budget_status`（预算余额）、`ledger_brief`（近期实验台账：避免重复已失败方向）、`gate_defs`（门禁定义）、`saturation`
  - 「可用工具」改占位符：`{{perf_profile_cmd}}`、`{{self_check_cmd}}`、领域知识用 `{{env_dir}}/domain_knowledge.md`；**严禁硬编码绝对路径**
  - 饱和纪律：`saturation` 非空 → 优先设计「跳出当前方向」实验（跨族组合/未覆盖做法），idea 里写明「已知饱和，本实验试图打破」
  - 台账纪律：读 `ledger_brief`，不得重复已 reject 的同类方向
- `super/prompts/coding.md`：
  - 输入材料增：`gate_defs`（**关键**：容差/软门禁 band/K 次中位——知道「改善多少才算赢」）、`perf_baseline`（该格精英基线，改善判据）、`workdir_files`（父本文件清单）、`ledger_brief`（同类失败教训）
- `super/prompts/debug.md`：
  - 输入材料增：`gate_failure`（**框架门禁失败详情**：failure_stage/metrics/details——这是新增的修复入口）、`gate_defs`、`ledger_brief`、`perf_profile_cmd`
  - 说明区分两类调试：「自检失败」（attempt 1..M）与「框架门禁失败」（attempt 100+，仅 1 次机会）
- `super/prompts/adjust.md`：
  - 输入材料增：`perf_timeline`、`budget_status`、`domain_brief`、`ledger_brief`、`own_decisions`（**自己的历史决策+效果**——防震荡/防重复）
  - 饱和纪律升级：`saturation` 非空 → 首选 `expand_cell`；若判断空间已榨干（domain_brief 的天花板已触及）→ `terminate_recommend`
  - `own_decisions` 纪律：调权前先看自己上次对同格的决策与效果，避免震荡
- 所有模板：路径一律占位符（`{{env_dir}}`/`{{perf_profile_cmd}}`/`{{self_check_cmd}}`）

# 写域
- super/prompts/builder.md
- super/prompts/coding.md
- super/prompts/debug.md
- super/prompts/adjust.md

# 依赖关系
- task22（键名契约已定；本 task 只写模板，与新 ctx 渲染兼容——缺失键渲染保留原样，不炸）

# 测试方法
`grep -c "/Users/" super/prompts/*.md` 全 0；四模板各含其契约键；`python3 -m unittest super.tests.test_prompts` 绿

# 禁止事项
- 不改 env_8 的 env/prompts（用户资产）；不动框架代码
