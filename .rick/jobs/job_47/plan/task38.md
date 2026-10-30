# task38：特征格四要素模型 + 实验记录模块 + 相邻格抽样

# 任务目标
KR2 数据结构 + KR1 载体模块：特征格增 `description`（名字/方法描述/source_refs）；新增 experiment_record 模块（建/读/追加）；相邻格随机抽样。

# 关键结果
- `super/superopt/archive.py`：
  - 格 schema 增 `description: {name, method, source_refs: []}`（`admit`/`set_weight`/`set_visit` 建格骨架时带默认空 description；新增 `set_description(cell_id, description)`；`cells()` 返回含 description）
  - 向后兼容：旧 archive 无 description → 读时补 `{}`
- 新建 `super/superopt/experiment_record.py`（stdlib-only）：
  - `path_for(iter_dir) -> str`（`<iter_dir>/experiment_record.md`）
  - `create(iter_dir, meta: dict) -> str`：按六段模板写（背景与价值/实验假设/验证逻辑/具体实验步骤/观测现象/执行过程与结论(占位)），meta 含 iteration_id/cell_id/idea/parents
  - `append_section(iter_dir, title: str, body: str) -> bool`：幂等追加（已有同名段则替换内容）
  - `read(iter_dir) -> str`（缺文件返回空串）
  - `digest(iter_dir, limit=1200) -> str`（供 ctx 注入的摘要：前 3 段 + 结论段）
  - `recent(ws, since_iter, until_iter, limit_chars=4000) -> list[dict]`：区间内全部实验记录摘要 [{iteration_id, cell_id, path, digest}]（adjust 全局分析用）
- `super/superopt/feature_space.py`：
  - `neighbor_pool(cell_id) -> list[str]`：共享 ≥1 轴值的其它格（排除自身）
  - `sample_neighbors(cell_id, k=2, rng=None) -> list[str]`：从 pool 随机抽 k（不足则全库随机补；全库也空→[]）
- 测试：`test_archive`（description 往返/set_description/旧档兼容）、新建 `test_experiment_record.py`（六段模板/幂等追加/read/digest/recent 区间）、`test_feature_space`（neighbor_pool 语义/sample_neighbors 随机+补足）

# 写域
- super/superopt/archive.py
- super/superopt/experiment_record.py（新建）
- super/superopt/feature_space.py
- super/tests/test_archive.py
- super/tests/test_experiment_record.py（新建）
- super/tests/test_feature_space.py

# 依赖关系
- 无（L2，与 task39/task40 写域互斥）

# 测试方法
`python3 -m unittest super.tests.test_archive super.tests.test_experiment_record super.tests.test_feature_space -v` 全绿

# 禁止事项
- 不动 loop.py（task39）；不动 gates/prompts
