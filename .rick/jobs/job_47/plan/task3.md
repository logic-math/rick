# task3：feature_space.py + sampler.py + 单元测试

## 任务目标

实现特征空间（格定义/描述子/正交性校验）与采样器（UCB 真实递增 + 冷启动先验加权随机）。**这是修复 v5 两大缺陷的核心**：visit 断链（db.py:201 零调用——采样不递增 visit，UCB 分母恒不变）与 UCB 退化（feature_space.py:95-162 退化为启发式权重、StrategyAgent 权重接管 :135-141）。

## 关键结果

- 新建 `super/superopt/feature_space.py`（stdlib-only）：
  - `class FeatureSpace`：
    - `__init__(self, axes: list)`——读 feature_space.json 的 axes 声明（每轴：name/type/取值域/描述）
    - `cell_id(self, descriptor: dict) -> str`——描述子 → 稳定格 id（轴名排序 + 值规范化的确定性字符串，如 `axis1=v1|axis2=v2`）
    - `validate_orthogonal(self) -> None`——轴间正交性校验（同名/同语义轴冲突即 raise ValueError；env 载入时调用一次）
    - `describe_cell(self, cell_id: str) -> Optional[dict]`——格 id → 描述子（供 prompt 渲染用）
    - `axis_names` / `axes` 访问器
- 新建 `super/superopt/sampler.py`（stdlib-only）：
  - `class Sampler`（构造注入 Archive 引用与 FeatureSpace 引用）：
    - `sample_cell(self, cold: Optional[bool] = None) -> str`——决策逻辑：
      - **热态（有 UCB 数据：格子有 visit 与 perf 记录）**：UCB1 公式 `argmax(mean_perf_normalized + C*sqrt(ln(total_visit)/visit))`；C 可配（默认 0.7）；**采样命中即 visit+1 由 increment_visit 显式做**（调用方负责，本方法纯计算——测试断言往返后 visit 真实变化）
      - **冷启动（格子无 visit 或全零）**：按格子先验权重（archive weight；无记录=1.0）加权随机（random.choices）
      - mixed 态（部分格热部分格冷）：热格参与 UCB、冷格按先验权重参与同一 softmax 归一（简并实现：把 UCB 分数与先验权重相乘后 argmax——设计树 R2-3 裁决「有 UCB 数据时用 UCB、冷启动用先验加权」的合并形态）
    - `increment_visit(self, cell_id: str) -> None`——**唯一 visit 递增入口**（写 archive；修复 v5 断链）
    - `ucb_score(self, cell_id: str, total_visit: int) -> float`——单格 UCB 分（供测试与 cell_sampled 事件 payload）
- 新建 `super/tests/test_sampler.py` + `super/tests/test_feature_space.py`（unittest）：
  - cell_id 确定性（同描述子同 id、不同序键序同 id）
  - 正交性校验：重复轴名 raise
  - **UCB 行为**：构造 3 格（visit/perf 不同）→ 选中的是 UCB argmax；increment_visit 后 archive.visit 变化、total_visit 递增 → 探索压力真实存在（visit=0 的格子下一次必被选/高概率被选——断言冷格被选中）
  - 冷启动：全部新格 → 按权重随机（weight 大的格被选中概率高——统计 200 次采样，权重 9:1 的两格选中比 > 3:1）
  - 灭绝格（weight=0）永不被采（断言 100 次全不命中）

# 写域

- super/superopt/feature_space.py（新建）
- super/superopt/sampler.py（新建）
- super/tests/test_feature_space.py（新建）
- super/tests/test_sampler.py（新建）

## 依赖关系

- 无（Kahn L1；与 task1/task2 **无 import 依赖**——Sampler 以鸭子类型持有 archive 对象（只需 .cells 访问器与 set_visit 类写法），单测用内存假 Archive，真集成在 task8 loop 接线）

## 实现要点

- stdlib-only：math/random
- Sampler 与 Archive 的接口约定：`archive.cells()` 返回 {cell_id: {...visit, weight, elite...}} 副本；visit 递增经 `archive.bump_visit(cell_id)`（若 task2 未提供该方法，在本 task 写域内**不得改 task2 文件**——改为 Sampler.increment_visit 回调注入：构造参数 `on_visit: Callable[[str], None]`，由调用方（loop）接 archive.bump_visit。**取此回调方案**，保持两文件零 import 耦合）
- mean_perf_normalized：格内 elite.score 归一到 [0,1]（无精英=0.5 中性值）

## 测试方法

`python3 -m unittest super.tests.test_feature_space super.tests.test_sampler -v` 全绿

## 禁止事项

- 不得在 Sampler 内直改 archive 文件（经回调）；不得实现删格逻辑；不得用启发式权重替代 UCB（v5 退化路径）
