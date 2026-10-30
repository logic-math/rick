# task20：adjust_trace.py 决策-效果对照工具 + e2e 断言适配

## 任务目标
自改进的归因工具：读事件流 → 输出「每次 adjust 决策 → 之后 K 迭代效果」对照表；e2e 断言随字段/事件变更适配。

## 关键结果
- `super/tools/adjust_trace.py`（新建目录 super/tools/，stdlib-only，py3.9）：
  - CLI：`python3 super/tools/adjust_trace.py --workdir <ws> [--window 5]`
  - 解析 events.jsonl：adjust_decision 事件（或回退解析 weight_changed/proposal 序列）→ 每次决策一行：迭代号/proposals 摘要（格+旧新权重）/决策时格状态摘要
  - 效果列：该决策后 K 迭代（默认 5）内受影响格的 gate 通过率/perf 变化/是否被采（从 cell_sampled+gate_result+elite 事件推导）
  - 输出 markdown 表到 stdout（中控会话/人可直接消费）
- `super/tests/e2e_test.py` 断言适配：ucb_score→weight_share 字段引用；如有 adjust 快照/事件相关断言更新为新结构；跑通全链 mock
- 附冒烟：临时 ws 手工造几条事件跑 adjust_trace 输出非空（写进测试或自测记录）

# 写域
- super/tools/adjust_trace.py（新建）
- super/tests/e2e_test.py

# 依赖关系
- task19（事件/快照格式）

# 测试方法
`python3 -m unittest super.tests.e2e_test -v` 全绿 + adjust_trace 冒烟输出 markdown 表

# 禁止事项
- 不动 loop/adjudicator（格式以 task19 落地为准，冲突先报告）
