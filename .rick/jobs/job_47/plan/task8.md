# task8：loop.py 四状态循环 Scheduler

## 任务目标

superopt 心脏：四状态循环（adjust→UCB 采样→builder→coding→debug→框架验证→MAP-Elites 入格）+ 崩溃游标恢复。修复 v5 的 StrategyAgent 权重接管/UCB 退化路径（采样权只经 Sampler UCB+先验，agent 只能提案）。

## 关键结果

- 新建 `super/superopt/loop.py`（stdlib-only）：
  - `class Scheduler`：
    - `__init__(self, ws: str, cfg: dict)`——组装：EventWriter/Archive/FeatureSpace/Sampler/PiAgentClient/CascadeGate/CostLedger/InterventionQueue（task9）——**以依赖注入**（构造参数可覆盖，测试注 mock）；FeatureSpace 读 ws/env/feature_space.json
    - `run(self) -> None`——主循环：init（--preverify-red flag 时先跑 preverify_red，失败 exit(3)）→ recover（游标重放投影恢复 visit/weight/cursor/pending）→ 循环 while budget.within_budget() 且非 stop 信号（SIGTERM 优雅：signal handler 置标志，落完当前事件退出）：drain 干预队列（task9 摄取入事件）→ 每 adjust_interval 迭代跑 _run_adjust → _step() → heartbeat 写
    - `_recover(self) -> int`——events.read_all → lineage.project_state（task4）恢复 → 返回 last_seq；空事件流=全新开始
    - `_step(self)`——单迭代：
      1. iteration_started 事件
      2. cell_id = sampler.sample_cell() → **increment_visit（唯一递增点）** → cell_sampled 事件（visit/ucb_score/冷启动标记）
      3. plan = _run_builder(cell_id)（agent_started/ended 事件包夹 + token 计量入 CostLedger）→ experiment_plan 存档
      4. code_path = _run_coding(plan)：建 codes/{uuid}/（拷贝父本 code）→ render+write prompt → PiAgentClient.run → 自检结果不符 → _run_debug 循环（M=debug_max_attempts 次，每次 fix_report 事件）
      5. _verify_and_admit(code_path, cell_id)：CascadeGate.verify → gate_result 事件 → 通过 → archive.admit（延迟入库判定：新格首解/容差带内/来源格已降权 → add_pending）→ elite_admitted/replaced 事件 + pending 事件
      6. iteration_ended 事件（本迭代 token/耗时汇总）
    - `_run_adjust(self)`——每 N 迭代：组装 adjust ctx（格现状+近期 gate 失败摘要）→ render → PiAgentClient.run → 解析 proposals：软动作（weight_changed 事件 + archive.set_weight）；硬动作 → proposal 事件 + 投递裁决队列文件（task9 的 adjudicator 消费，本循环不裁决——提案-裁决分离 KR5）
    - 失败分类分派：gate stage1 失败 → env_failure_concluded 事件（诊断包 returncode/stderr_tail/死因分类）暂停迭代 N 次后退出；agent 超时/空产出 → 记失败继续下迭代
    - pending 重评估：每 epoch（=10 迭代，常量）末 drain_pending → 逐个 verify M 次复测 → invalidation（出档：elite 清空+invalidation 事件）/ resurrection（恢复 weight+事件）
- 新建 `super/tests/test_loop.py`（unittest，全 mock：mock_pi 场景 + 内存 Archive/假 gates）：
  - **3 迭代全链**：事件序列断言（iteration_started→cell_sampled→agent_started/ended(builder)→agent_started/ended(coding)→gate_result→elite_admitted→iteration_ended 严格有序；seq 连续）
  - **visit 真实递增**：迭代后 archive visit 比迭代前 +1（修复 v5 断链的回归断言）
  - **崩溃恢复**：跑 2 迭代 → 新 Scheduler 同 ws → _recover 恢复 cursor=2/visit/weight 一致 → 续跑第 3 迭代事件 seq 续号
  - **debug 循环**：coding 自检失败场景 → debug 派发 M 次上限后放弃（fix_report 事件数=M）
  - **adjust 软动作**：mock adjust 提案 weight 0.5 → weight_changed 事件 + archive.weight=0.5；硬动作提案 → proposal 事件 + 裁决队列文件出现（不直改 archive）
  - **延迟入库**：容差带内解 → pending 非空 + elite_admitted 事件
  - **预算墙**：max_iterations=3 → 循环自然退出 + session_ended 事件

# 写域

- super/superopt/loop.py（新建）
- super/tests/test_loop.py（新建）

## 依赖关系

- task5/task6/task7（Kahn L3）+ task1-4（经组合）——import 全部 superopt 模块（真组装）；task9 的 InterventionQueue 以鸭子类型注入（缺省容错：队列目录空则跳过——允许 task8/task9 并行不阻塞）

## 实现要点

- 每迭代一 agent 进程（PiAgentClient.run 独立 subprocess）——env 热更新天然生效（J4）
- 所有状态变更**先事件后档案**（events.jsonl 是唯一真相源；archive 是投影——崩溃后从 events 重建）
- codes/{uuid}/ 完整快照：父本代码拷贝 + prompt + agent final + 自检输出（lineage 重建的物证）
- SIGTERM 优雅退出：当前事件落盘完毕才退（heartbeat 停写 → Go 侧 OnDead 不误判：正常退出 exit 0，Go 读 state）

## 测试方法

`python3 -m unittest super.tests.test_loop -v` 全绿（秒级：全 mock 无真 pi）

## 禁止事项

- 不得让 agent 输出直改 archive/weight（只经事件与提案——KR5）；不得绕过 increment_visit 采样（v5 断链）；不得在 loop 内做门禁阈值判断（CascadeGate 职责）
