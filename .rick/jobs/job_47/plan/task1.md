# task1：events.py 单写者真相源 + 单元测试

## 任务目标

实现 super 调度器的唯一真相源：append-only 事件日志 `events.jsonl`。修复 v5 的断链缺陷（events 只写不读、游标无消费方），为崩溃恢复（游标重放）与全系统状态重建（lineage/北极星/审计）提供底座。

## 关键结果

- 新建 `super/superopt/events.py`（stdlib-only）：
  - `class EventWriter`：
    - `__init__(self, path: str)`——打开 append 模式文件句柄；若文件已存在则从尾部续 seq
    - `append(self, type: str, payload: dict, source: str) -> int`——写一行 JSON `{ts, seq, type, source, payload}`；**ts 必须带时区（RFC3339，bugs.md 坑位）**；seq 单调递增（从既有 max seq+1 起）；写后 fsync（崩溃安全）
    - `read_all(self) -> list`——全量读取（重放用；容忍最后一行半写截断：跳过并返回之前完整行——崩溃时 fsync 前挂掉的防线）
    - `last_seq(self) -> int`——最后完整事件 seq（游标）
  - 模块级 `EVENT_TYPES`：16 类事件类型常量集（session_started/session_ended/iteration_started/iteration_ended/cell_sampled/agent_started/agent_ended/candidate_produced/gate_result/elite_admitted/elite_replaced/weight_changed/proposal/adjudicated/invalidation/resurrection/intervention/heartbeat/env_failure_concluded——按设计树 §2.2 定稿，最终清单以设计树为准，需含全部 16+ 类）
  - `validate_event(ev: dict) -> list`——返回 schema 违规列表（type ∈ EVENT_TYPES、必填字段齐全、seq/ts 类型正确）——测试与门禁共用
- 新建 `super/tests/test_events.py`（unittest，stdlib）：
  - 追加 N 条 → seq 连续单调、ts 带时区、每行合法 JSON
  - 重开 EventWriter（模拟重启）→ seq 续号不回退
  - read_all 幂等、last_seq 正确
  - **截断容错**：手工写半行到文件尾 → read_all 跳过坏行不抛异常，last_seq=最后完整行
  - fsync 生效性不直接断言（不 mock fsync），以「写后文件可读」代替

# 写域

- super/superopt/events.py（新建）
- super/tests/test_events.py（新建）

## 依赖关系

- 无（Kahn L1 首层，与 task2/task3/task4/task10/task12/task13 写域互不相交，并行）

## 实现要点

- stdlib-only：json/time/os/io；**Python ≥3.9 兼容**（禁 `X | Y` 联合类型注解，用 `Optional[X]`）
- 单行 JSON紧凑写出（separators=(',',':')），一行一事件
- 不做进程锁（单写者是架构保证：只有 py 调度器进程持有写句柄；人工干预经队列由调度器代写）
- EVENT_TYPES 用 frozenset 常量；payload schema 各事件的必填字段表写成模块级 dict（`EVENT_SCHEMAS`），供 validate_event 与下游消费方引用

## 测试方法

`python3 -m unittest super.tests.test_events -v` 全绿（在 dev 树仓库根执行；super/ 顶层目录加入路径的方式：tests 用 `python3 -m unittest discover -s super/tests` 亦可，见 super/tests/__init__.py 约定——若用包导入，需 super/ 与 super/tests/ 各有 __init__.py，属于本 task 写域外的公共文件由 task4 统一建）

## 禁止事项

- 不得引入第三方依赖
- 不得写 events 以外的文件
- 不得实现「读改写」（append-only 是硬约束）
