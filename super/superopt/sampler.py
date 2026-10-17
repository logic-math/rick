"""sampler.py — UCB 采样（visit 真实递增）+ 冷启动先验加权随机。

修复 v5 两大采样缺陷（本模块存在的理由）：
1. **visit 断链**（db.py:201 零调用）：采样不递增 visit → UCB 分母恒不变。
   本实现：``increment_visit`` 是唯一递增入口（经 on_visit 回调写 archive）；
   ``sample_cell`` 为纯计算（调用方显式递增——loop 在采样后接线）。
2. **UCB 退化 / StrategyAgent 权重接管**（feature_space.py:95-162 / :135-141）：
   本实现是纯 UCB1 + 先验权重乘子——无启发式、无 agent 直改权重通道
   （adjust 的软动作经 archive.weight 生效，硬动作走外置裁决，均不经本模块）。

采样决策（设计树 R2-3 裁决）：
- **冷启动**（全格 visit==0，或 ``cold=True`` 强制）：按格子先验权重
  （archive weight；无记录=1.0）加权随机（random.choices）
- **热态/混合态**：``score(cell) = ucb_score(cell) * prior_weight(cell)`` 取
  argmax（R2-3 合并形态的简并实现）。visit==0 的格 ucb=+inf（未探索臂的
  乐观上界）→ 必进 argmax 平局集 → 冷格必被探索；平局集内按先验权重
  加权随机（保住冷启动的权重语义）
- **灭绝格**（weight<=0）：任何态下永不被采样（功能性灭绝，R2-2：不删格）

与 Archive 的接口约定（鸭子类型，零 import 耦合）：
- ``archive.cells() -> {cell_id: {visit, weight, elite, ...}}``（返回副本）
- visit 递增经构造注入的 ``on_visit: Callable[[str], None]``（loop 接
  ``archive.bump_visit``）；未注入而调 increment_visit → 显式报错（断链防御）
- 特征空间是候选全集的权威：archive 中空间外脏格不参与采样

stdlib-only，Python >= 3.9。
"""
import math
import random
from typing import Any, Callable, Dict, Optional

# UCB1 探索系数默认值（可配：构造参数 c）
DEFAULT_C = 0.7


class Sampler:
    """格子采样器：冷启动先验加权随机 + 热/混合态 UCB×weight argmax。"""

    def __init__(self,
                 archive: Any,
                 feature_space: Any,
                 on_visit: Optional[Callable[[str], None]] = None,
                 c: float = DEFAULT_C,
                 rng: Optional[random.Random] = None) -> None:
        """
        :param archive: 鸭子类型 Archive（须有 cells() 方法）
        :param feature_space: 鸭子类型 FeatureSpace（须有 all_cells() 方法）
        :param on_visit: visit 递增回调（loop 接 archive.bump_visit）；None 时
            increment_visit 会显式报错（防 v5 断链复发）
        :param c: UCB 探索系数（>0）
        :param rng: 随机源（random.Random 实例；默认用 random 模块——测试注入固定种子）
        """
        if not callable(getattr(archive, "cells", None)):
            raise ValueError("archive 需提供 cells() 方法（鸭子类型接口），收到 %r" % (archive,))
        if not callable(getattr(feature_space, "all_cells", None)):
            raise ValueError("feature_space 需提供 all_cells() 方法（鸭子类型接口），收到 %r"
                             % (feature_space,))
        if not (isinstance(c, (int, float)) and c > 0):
            raise ValueError("UCB 探索系数 c 必须为正数，收到 %r" % (c,))
        if on_visit is not None and not callable(on_visit):
            raise ValueError("on_visit 必须是 Callable[[str], None] 或 None")
        self._archive = archive
        self._fs = feature_space
        self._on_visit = on_visit
        self._c = float(c)
        self._rng = rng if rng is not None else random

    # ---- 对外 API ----

    def sample_cell(self, cold: Optional[bool] = None) -> str:
        """采样一个目标格（纯计算：不改任何状态）。

        :param cold: None=自动判定（全冷走冷启动，否则热/混合路径）；
            True=强制冷启动路径（无视 visit，纯权重随机）；
            False=强制热/混合路径（无热格时退化为权重随机——无 UCB 数据可用）
        :return: cell_id
        :raises ValueError: 无可采样格（全部灭绝 / 空间为空）
        """
        snap = self._archive.cells()
        # 候选 = 空间内非灭绝格（weight<=0 过滤；空间是候选全集的权威）
        candidates = [cid for cid in self._fs.all_cells()
                      if self._weight(cid, snap) > 0.0]
        if not candidates:
            raise ValueError("无可采样格子：全部格子 weight<=0（功能性灭绝）或特征空间为空")
        hot = [cid for cid in candidates if self._visit(cid, snap) > 0]

        if cold is True or (cold is None and not hot):
            # 冷启动：先验权重加权随机
            return self._weighted_choice(candidates, snap)
        if not hot:
            # cold=False 强制热路径但无热格：退化为权重随机（无 UCB 数据可用）
            return self._weighted_choice(candidates, snap)

        # 热/混合态：score = ucb * prior_weight → argmax；平局集内权重随机
        total_visit = sum(self._visit(cid, snap) for cid in candidates)
        scores = {cid: self._ucb(cid, total_visit, snap) * self._weight(cid, snap)
                  for cid in candidates}
        best = max(scores.values())
        tie = [cid for cid in candidates if scores[cid] == best]
        if len(tie) == 1:
            return tie[0]
        return self._weighted_choice(tie, snap)

    def increment_visit(self, cell_id: str) -> None:
        """唯一 visit 递增入口（修复 v5 断链；写 archive 经 on_visit 回调）。

        :raises ValueError: 未注入回调 / 格不在特征空间内（拒脏格）
        """
        if cell_id not in self._fs.all_cells():
            raise ValueError("格 %r 不在特征空间内（拒绝递增空间外脏格）" % (cell_id,))
        if self._on_visit is None:
            raise ValueError("increment_visit 需要 on_visit 回调（断链防御：递增必须落到 archive）")
        self._on_visit(cell_id)

    def ucb_score(self, cell_id: str, total_visit: int) -> float:
        """单格 UCB 分（不含先验权重；供测试与 cell_sampled 事件 payload）。

        - visit>0：``mean_norm + C*sqrt(ln(total_visit)/visit)``
        - visit==0：``+inf``（未探索臂的乐观上界——序列化进事件 payload 时
          调用方需注意 JSON 无 Infinity，应 cap/标注）
        - mean_norm：格内 elite.score clamp 到 [0,1]；无精英=0.5 中性值
        """
        snap = self._archive.cells()
        return self._ucb(cell_id, total_visit, snap)

    # ---- 内部 ----

    def _ucb(self, cell_id: str, total_visit: int, snap: Dict[str, dict]) -> float:
        visit = self._visit(cell_id, snap)
        if visit <= 0:
            return math.inf
        total = max(int(total_visit), 1)  # 防御：total<1 时 log 非法
        mean = self._mean_norm(cell_id, snap)
        return mean + self._c * math.sqrt(math.log(total) / visit)

    def _weighted_choice(self, cell_ids, snap: Dict[str, dict]) -> str:
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

    @staticmethod
    def _mean_norm(cell_id: str, snap: Dict[str, dict]) -> float:
        entry = snap.get(cell_id) or {}
        elite = entry.get("elite")
        score = elite.get("score") if isinstance(elite, dict) else None
        if score is None:
            return 0.5  # 无精英：中性值
        try:
            sc = float(score)
        except (TypeError, ValueError):
            return 0.5
        return min(1.0, max(0.0, sc))  # clamp 到 [0,1]（防 0-100 制/负值脏数据）
