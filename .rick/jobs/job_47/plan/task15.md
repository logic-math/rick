# task15：sampler 纯加权随机重构

## 任务目标
采样决策权集中 adjust：sample = 加权随机（概率∝weight）。UCB 数学退役。

## 关键结果
- `super/superopt/sampler.py`：删除 UCB 计算（_ucb_pick/explore bonus）；`sample_cell` = 存活格（weight>0）按 weight 比例 `rng.choices`；**分殖优先保留**（存在 visit==0 且 weight>0 的格 → 仅在其中加权随机）；visit 照常递增（事件/统计用，不参与采样决策）
- `cell_sampled` 事件 payload：`ucb_score` 字段改名 `weight_share`（=该格 weight/总 weight，可观测性）
- `super/tests/test_sampler.py` 重写：权重比例断言（w=3:1 两格 200 次采样比 ~2-3.5:1）；灭绝格不被采；分殖优先（未访问格先覆盖）；weight_share 字段断言；旧 UCB 用例删除
- Python 3.9 兼容

# 写域
- super/superopt/sampler.py
- super/tests/test_sampler.py

# 依赖关系
- 无（L1 首层并行）

# 测试方法
`python3 -m unittest super.tests.test_sampler -v` 全绿；`grep -rn "sqrt\|math.log" super/superopt/sampler.py` 无残留

# 禁止事项
- 不动 loop.py/事件写入侧（字段改名由 task19 在 loop 侧适配——本 task 只改 sampler 与其测试；若 cell_sampled 事件由 loop 写则 loop 侧的 ucb_score 引用留给 task19，你在 sampler 接口注释里标注）
