# task2：archive.py 精英格档案 + 单元测试

## 任务目标

实现 MAP-Elites 精英格档案：单文件 JSON（archive.json），格→精英映射 + lineage/置信度 + 采样统计 + pending 延迟入库队列。修复 v5「lineage 恒 None、精英无 JSON 导出」缺陷（lineage 由 task4 的 lineage.py 从事件流重建，本模块负责承载与更新）。

## 关键结果

- 新建 `super/superopt/archive.py`（stdlib-only）：
  - `class Archive`（schema 按设计树 §2.3）：
    - `__init__(self, path: str)`；`load(self)` / `save(self)`（写临时文件+rename 原子替换；save 带 fsync）
    - `admit(self, cell_id: str, entry: dict) -> bool`——**同格替换条件**：新 perf_value 优于旧（perf_desc 方向感知：低优=越小越好 / 高优=越大越好，方向读 env 的 perf_desc 并存入 archive meta）或旧格为空；替换时旧精英**不出档**（历史精英保留在 cells[cell_id].history 列表，上限 20，防档案膨胀）——v5「同格替换」语义保留
    - `get_elite(self, cell_id: str) -> Optional[dict]`
    - `add_pending(self, cell_id: str, code_path: str, reason: str)`——延迟入库队列（reason ∈ {new_cell, tolerance_band, low_weight_src}，L2-Q6 三判定）
    - `drain_pending(self) -> list`——取走全部 pending（重评估消费）
    - `set_weight(self, cell_id: str, weight: float)`——adjust 软动作落点（可降 0=功能性灭绝，**不删格**）
    - `cells` 访问器返回副本（防外部直改）
  - 档案 schema：`{meta: {env, feature_axes, perf_desc, updated_at}, cells: {<cell_id>: {descriptor, elite: {code_path, perf_value, perf_type, score, lineage, confidence, admitted_at, gen}, history: [], weight, visit, sample_count, last_sampled_at}}, pending: [{cell_id, code_path, reason, added_at}]}`
- 新建 `super/tests/test_archive.py`（unittest）：
  - admit 新格 → 入档；同格更优 → 替换 + history 追加；同格更差 → 拒绝（返回 False）
  - perf_desc 方向感知（低优：5.0 替换 6.0 ✓；高优：7.0 替换 6.0 ✓）
  - pending add/drain 往返；set_weight 降 0 后格仍在（灭绝不删格）
  - load/save 往返一致；save 原子性（写坏档不破坏旧档——用并发场景简化断言：save 前 load 的内容在 save 失败后仍可 load）

# 写域

- super/superopt/archive.py（新建）
- super/tests/test_archive.py（新建）

## 依赖关系

- 无（Kahn L1；与 task1 等首层任务写域互斥，并行）

## 实现要点

- stdlib-only：json/os/tempfile/dataclasses（可选）
- 与 task1 无 import 依赖（Archive 不写事件）；与 task3（FeatureSpace）解耦：cell_id 是不透明字符串，descriptor 只是数据
- Python ≥3.9 兼容（Optional 语法）

## 测试方法

`python3 -m unittest super.tests.test_archive -v` 全绿

## 禁止事项

- 不得引入第三方依赖；不得在 Archive 内做采样决策（那是 task3 Sampler 的职责）
- 不得删除格（weight=0 语义即灭绝，格子永久保留）
