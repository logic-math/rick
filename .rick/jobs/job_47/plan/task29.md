# task29：四模板升级（三要素使用纪律 + builder 论文/历史/落盘）

## 任务目标
把三要素、profile 历史、arXiv 论文检索、决策落盘写进模板纪律。
**ctx 键契约**（task28 提供）：`profile_cmd/benchmark_cmd/verify_cmd/gate_check_cmd`（b/c/d）、`target_profile_history/parent_profile_history/inspiration_profile_history`（builder）、`designs_dir/recent_designs`（builder）、`profile_history` 兜底。

## 关键结果
- `super/prompts/builder.md`（大改）：
  - **输入材料增**：`target_profile_history`（目标格 profile 时间序列——看现状如何变化：热点迁移/耗时结构）、`parent_profile_history`、`inspiration_profile_history`（父格与灵感格的现状轨迹）、`recent_designs`（近期决策，避免重复方向）、`designs_dir`
  - **三要素使用纪律**：设计实验前用 `{{profile_cmd}}` 剖析父本拿事实 → 用 `{{benchmark_cmd}}` 理解评估方法（K 次中位/score 语义）→ 用 `{{verify_cmd}}` 确认约束边界；禁止凭想象设计
  - **外部论文检索（强制）**：用 web_search 检索 **arXiv**（关键词=任务+方法+瓶颈，如「matmul optimization BLAS dispatch」「cache blocking GEMM」），最多 3 篇；每篇提取 `{arxiv_id, title, insight(可实现的机制), applicability(在本任务/本格怎么用)}`；论文必须落到可实现动作，不得泛泛引用；工具不可用则如实记录 `papers: []` 并说明原因
  - **决策落盘（强制）**：把 `{profile_facts, papers, inspirations, reasoning, experiment_plan}` 写入 `{{designs_dir}}/iter{N}_{cell}.json`（文件名用 ctx 的 iteration_id 与 cell_id）——这是进化轨迹的核心物证
  - 输出契约 JSON 增 `papers`/`inspirations`/`reasoning` 三个字段
- `super/prompts/coding.md`：
  - 输入材料增 `gate_check_cmd`（**与框架同一套门禁**，可自行调用拿权威判定）、`profile_cmd`（定位热点）、`benchmark_cmd`/`verify_cmd`（分别验证评估/约束）
  - 纪律：实现后依次跑 verify（约束）→ benchmark（评估）→ gate_check（融合判定，含 --baseline）；自检结论与框架同源，避免「自检过框架拒」的漂移
- `super/prompts/debug.md`：输入材料增三者命令 + gate_check_cmd；纪律：先用 profile 定位热点/根因，再用 verify/benchmark 分段定位失败阶段，最后 gate_check 复验
- `super/prompts/adjust.md`：可选给 `gate_check_cmd`（判断某方向是否还有潜力时可让 agent 自行验证）+ 饱和时参考 designs 的新方向

# 写域
- super/prompts/builder.md
- super/prompts/coding.md
- super/prompts/debug.md
- super/prompts/adjust.md

# 依赖关系
- task28（ctx 键契约，键名已写死；缺失键宽松保留不炸）

# 测试方法
`grep 无 /Users/`；四模板含其契约键与三要素命令；`python3 -m unittest super.tests.test_prompts` 绿

# 禁止事项
- 不动框架代码；不改 env_8/env_9 的 env/prompts（用户资产）
