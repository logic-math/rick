"""sampler.py — 纯加权随机采样（概率 ∝ adjust 权重）+ 分殖优先。

架构定位（job_47 迭代 3 裁决 J-i3-1/J-i3-2）：**采样决策权集中 adjust agent**。
- 探索 = adjust 加格（expand_cell，走裁决链与 feature_space 热重载——loop 侧）
- 利用 = adjust 权重（权重高 → 采样份额大）
- 淘汰 = 灭绝（weight=0 = 功能性灭绝，R2-2：不删格）
本模块因此退化为**纯执行器**：不做任何统计决策（UCB 数学已退役——迭代 3
裁决；v5 时代的 UCB 实现见 git 历史与本模块的旧文档字符串）。

历史价值保留说明：v5 两大采样缺陷的回归防线**全部保留**：
1. **visit 断链**（db.py:201 零调用）：``increment_visit`` 仍是唯一递增入口
   （经 on_visit 回调写 archive）——visit 现在只服务事件流/统计/自改进分析，
   不再参与采样决策，但断链防御（未注入回调即显式报错）不变。
2. **静默权重接管**：权重语义现在**显式委派**给 adjust（有 weight_changed
   事件审计 + 裁决链），不再是绕过统计层的静默旁路。

采样规则：
- 候选 = 空间内存活格（weight>0；特征空间是候选全集的权威）
- **分殖优先**：存在未访问（visit==0 且 weight>0）的格 → 仅在未访问格中
  按权重加权随机（保证全空间铺开：每格至少 1 次——评估新格用确定性机会，
  新加格由此自动获得首访）
- 全部已访问 → 全部存活格按权重加权随机
- ``cold=True`` 强制参数保留（测试语义：全候选权重随机，跳过分殖优先）

与 Archive 的接口约定（鸭子类型，零 import 耦合）：
- ``archive.cells() -> {cell_id: {visit, weight, elite, ...}}``（返回副本）
- visit 递增经构造注入的 ``on_visit: Callable[[str], None]``（loop 接
  ``archive.bump_visit``）；未注入而调 increment_visit → 显式报错

事件字段契约（⚠️ loop 侧适配归 task19）：``cell_sampled`` 事件的
``ucb_score`` 字段已改名 ``weight_share``（= 该格 weight / 候选集总 weight，
采样前快照的可观测值）——由本模块 ``weight_share()`` 提供，loop 写事件时
应改用该值（本 task 只改 sampler 与其测试，loop.py 的 ucb_score 引用
由 task19 统一适配）。

stdlib-only，Python >= 3.9。
"""
import random
from typing import Any, Callable, Dict, Optional


class Sampler:
    """格子采样器：纯加权随机（概率 ∝ weight）+ 分殖优先。

    权重语义（J-i3-1）：weight 是 **相对采样份额**——允许 >1（如 2.0 =
    平均份额的两倍），0 = 功能性灭绝。adjust agent 是权重的唯一合法作者
    （weight_changed 事件审计）；本模块不做任何修饰（不乘、不加成）。
    """

    def __init__(self,
                 archive: Any,
                 feature_space: Any,
                 on_visit: Optional[Callable[[str], None]] = None,
                 rng: Optional[random.Random] = None,
                 c: Optional[float] = None) -> None:
        """
        :param archive: 鸭子类型 Archive（须有 cells() 方法）
        :param feature_space: 鸭子类型 FeatureSpace（须有 all_cells() 方法）
        :param on_visit: visit 递增回调（loop 接 archive.bump_visit）；None 时
            increment_visit 会显式报错（防 v5 断链复发）
        :param rng: 随机源（random.Random 实例；默认用 random 模块——测试注入固定种子）
        :param c: 【已废弃】UCB 探索系数（迭代 3 退役）。仅为 loop.py 构造点的
            向后兼容保留（接受并忽略）；task19 将清理调用侧并移除此参数。
        """
        if not callable(getattr(archive, "cells", None)):
            raise ValueError("archive 需提供 cells() 方法（鸭子类型接口），收到 %r" % (archive,))
        if not callable(getattr(feature_space, "all_cells", None)):
            raise ValueError("feature_space 需提供 all_cells() 方法（鸭子类型接口），收到 %r"
                             % (feature_space,))
        if on_visit is not None and not callable(on_visit):
            raise ValueError("on_visit 必须是 Callable[[str], None] 或 None")
        self._archive = archive
        self._fs = feature_space
        self._on_visit = on_visit
        self._rng = rng if rng is not None else random
        # c 已废弃：忽略（UCB 数学退役；见类 docstring）

    # ---- 对外 API ----

    def sample_cell(self, cold: Optional[bool] = None) -> str:
        """采样一个目标格（纯计算：不改任何状态）。

        :param cold: None=自动判定（分殖优先 → 全量加权随机）；
            True=强制全候选权重随机（跳过分殖优先——测试语义）
        :return: cell_id
        :raises ValueError: 无可采样格（全部灭绝 / 空间为空）

        分殖优先：自动判定下若存在未访问（visit==0 且 weight>0）的格子 →
        仅在未访问格中按权重加权随机——保证全空间铺开（每格至少 1 次）；
        灭绝格即使未访问也不参与分殖。cold=True 不启用分殖优先（测试语义）。
        """
        snap = self._archive.cells()
        # 候选 = 空间内非灭绝格（weight<=0 过滤；空间是候选全集的权威）
        candidates = [cid for cid in self._fs.all_cells()
                      if self._weight(cid, snap) > 0.0]
        if not candidates:
            raise ValueError("无可采样格子：全部格子 weight<=0（功能性灭绝）或特征空间为空")

        if cold is True:
            return self._weighted_choice(candidates, snap)
        # 自动判定 / cold=False：分殖优先（未访问存活格铺开）
        unvisited = [cid for cid in candidates
                     if self._visit(cid, snap) == 0]
        if unvisited:
            return self._weighted_choice(unvisited, snap)
        return self._weighted_choice(candidates, snap)

    def weight_share(self, cell_id: str) -> float:
        """该格在当前候选集中的采样份额（= weight / 候选集总 weight）。

        供 ``cell_sampled`` 事件 payload 的 ``weight_share`` 字段（可观测性：
        每次采样时的实际概率分布快照）。灭绝格/空间外格返回 0.0。
        """
        snap = self._archive.cells()
        if cell_id not in self._fs.all_cells():
            return 0.0
        candidates = [cid for cid in self._fs.all_cells()
                      if self._weight(cid, snap) > 0.0]
        total = sum(self._weight(cid, snap) for cid in candidates)
        if total <= 0.0:
            return 0.0
        return self._weight(cell_id, snap) / total

    def set_feature_space(self, feature_space: Any) -> None:
        """热重载入口（task19）：替换特征空间引用（采样语义不变）。

        feature_space.json 被 expand_cell 裁决应用或外部修改后，loop 在迭代
        边界重建 FeatureSpace 并经本方法注入——新格（visit=0）自然进入
        分殖优先。仅做鸭子类型校验（与构造一致），不改任何采样行为。
        """
        if not callable(getattr(feature_space, "all_cells", None)):
            raise ValueError("feature_space 需提供 all_cells() 方法（鸭子类型接口），"
                             "收到 %r" % (feature_space,))
        self._fs = feature_space

    def increment_visit(self, cell_id: str) -> None:
        """唯一 visit 递增入口（v5 断链防御保留；写 archive 经 on_visit 回调）。

        visit 现在只服务事件流/统计/自改进分析（不再参与采样决策），
        但递增必须落到 archive 的防线不变。

        :raises ValueError: 未注入回调 / 格不在特征空间内（拒脏格）
        """
        if cell_id not in self._fs.all_cells():
            raise ValueError("格 %r 不在特征空间内（拒绝递增空间外脏格）" % (cell_id,))
        if self._on_visit is None:
            raise ValueError("increment_visit 需要 on_visit 回调（断链防御：递增必须落到 archive）")
        self._on_visit(cell_id)

    # ---- 内部 ----

    def _weighted_choice(self, cell_ids, snap: Dict[str, dict]) -> str:
        """存活格按 weight 比例随机（唯一采样原语）。"""
        weights = [self._weight(cid, snap) for cid in cell_ids]
        return self._rng.choices(cell_ids, weights=weights, k=1)[0]

    @staticmethod
    def _visit(cell_id: str, snap: Dict[str, dict]) -> int:
        entry = snap.get(cell_id) or {}
        try:
            return int(entry.get("visit", 0))
        except (TypeError, ValueError):
            return 0  # 脏数据按未访问处理

    @staticmethod
    def _weight(cell_id: str, snap: Dict[str, dict]) -> float:
        entry = snap.get(cell_id) or {}
        w = entry.get("weight", 1.0)
        if w is None:
            w = 1.0
        try:
            wf = float(w)
        except (TypeError, ValueError):
            return 0.0  # 脏数据保守视为灭绝（不可采），不炸采样循环
        return wf if wf > 0.0 else 0.0
