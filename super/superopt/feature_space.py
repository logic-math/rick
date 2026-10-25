"""feature_space.py — 特征空间（格定义 / 描述子 / 正交性校验）。

MAP-Elites 特征空间底座：轴声明（enum 取值域）→ 全格笛卡尔积 → 确定格 id。
供 sampler（候选格全集）、prompts（格元信息渲染）、loop（入格寻址）共用。

cell_id 规范：轴名排序后 `name=value` 以 `|` 连接（值一律 str 规范化），
同一描述子（键序无关）恒得同一 id。

stdlib-only，Python >= 3.9（禁 PEP604 联合类型注解）。
"""
from itertools import product
import random
from typing import Any, Dict, List, Optional

# 全格数上限：防御特征空间组合爆炸（MAP-Elites 特征空间应为小规模，几十~几百格）
_MAX_CELLS = 50000

_SUPPORTED_TYPES = ("enum",)


class FeatureSpace:
    """特征空间：axes 声明 → 确性格 id 映射。

    axes 结构（来自 feature_space.json，由调用方加载后传入）::

        [
          {"name": "impl_strategy", "type": "enum",
           "values": ["naive_loops", "blocked"], "desc": "实现策略"},
          {"name": "memory_layout", "type": "enum",
           "values": ["row_major", "tiled"], "desc": "内存布局"},
        ]

    - type 缺省视为 enum（当前唯一支持的轴类型；未知 type 构造即报错）
    - values 元素会被 str 规范化（int/bool 统一文本形式；规范化后重复即报错）
    - 正交性校验是独立步骤（validate_orthogonal，env 载入时调用一次）：
      同名轴 / 描述相同（strip 后）的同语义轴 → ValueError
    """

    def __init__(self, axes: List[Dict[str, Any]]) -> None:
        if not isinstance(axes, list) or not axes:
            raise ValueError("特征空间至少需要一轴（axes 为非空 list）")
        normalized = []
        for ax in axes:
            if not isinstance(ax, dict):
                raise ValueError("每轴必须是 dict，收到 %r" % (ax,))
            name = ax.get("name")
            if not isinstance(name, str) or not name.strip():
                raise ValueError("轴缺少合法 name：%r" % (ax,))
            name = name.strip()
            ax_type = ax.get("type", "enum")
            if ax_type not in _SUPPORTED_TYPES:
                raise ValueError("轴 %s 的 type %r 不支持（当前支持：%s）"
                                 % (name, ax_type, "/".join(_SUPPORTED_TYPES)))
            values = ax.get("values")
            if not isinstance(values, list) or not values:
                raise ValueError("enum 轴 %s 需要非空 values" % name)
            norm_values = [self._norm_value(v) for v in values]
            if len(set(norm_values)) != len(norm_values):
                raise ValueError("enum 轴 %s 的 values 规范化后存在重复" % name)
            desc = ax.get("desc", "")
            if not isinstance(desc, str):
                raise ValueError("轴 %s 的 desc 必须是 str" % name)
            normalized.append({"name": name, "type": ax_type,
                               "values": norm_values, "desc": desc})
        # 按轴名排序：cell_id / all_cells 的确定性基础
        self._axes = sorted(normalized, key=lambda a: a["name"])
        self._by_name = {a["name"]: a for a in self._axes}
        self._all_cells = None  # 惰性缓存

    # ---- 校验 ----

    def validate_orthogonal(self) -> None:
        """轴间正交性校验（env 载入时调用一次）。

        - 同名轴 → ValueError（cell_id 寻址歧义）
        - 描述（strip 后非空）完全相同的两轴 → ValueError（同语义冗余的保守代理判定）
        """
        names = [a["name"] for a in self._axes]
        if len(set(names)) != len(names):
            raise ValueError("轴名重复（正交性破坏）：%s" % names)
        seen_desc = {}  # desc(strip) -> axis name
        for a in self._axes:
            key = a["desc"].strip()
            if not key:
                continue  # 空/缺省描述不参与同语义判定
            if key in seen_desc:
                raise ValueError("轴 %s 与 %s 描述相同（同语义轴，正交性破坏）"
                                 % (seen_desc[key], a["name"]))
            seen_desc[key] = a["name"]

    # ---- 寻址 ----

    def cell_id(self, descriptor: Dict[str, Any]) -> str:
        """描述子 → 确定格 id（如 ``impl_strategy=blocked|memory_layout=tiled``）。

        严格校验：未知轴 / 缺轴 / 值不在取值域 → ValueError（脏描述子早失败）。
        """
        if not isinstance(descriptor, dict) or not descriptor:
            raise ValueError("descriptor 必须是非空 dict，收到 %r" % (descriptor,))
        axis_names = self.axis_names
        unknown = [k for k in descriptor if k not in self._by_name]
        if unknown:
            raise ValueError("descriptor 含未声明轴 %s（声明轴：%s）"
                             % (unknown, axis_names))
        missing = [n for n in axis_names if n not in descriptor]
        if missing:
            raise ValueError("descriptor 缺轴 %s（声明轴：%s）" % (missing, axis_names))
        parts = []
        for name in axis_names:  # 已按轴名排序
            norm = self._norm_value(descriptor[name])
            if norm not in self._by_name[name]["values"]:
                raise ValueError("轴 %s 的值 %r 不在取值域 %s"
                                 % (name, descriptor[name],
                                    self._by_name[name]["values"]))
            parts.append("%s=%s" % (name, norm))
        return "|".join(parts)

    def describe_cell(self, cell_id: str) -> Optional[Dict[str, str]]:
        """格 id → 描述子（供 prompt 渲染）。

        非法/未知格 id 返回 None（优雅降级，不 raise）——解析出的值均为
        str 规范化形式（与 cell_id 的值规范一致）。
        """
        if not isinstance(cell_id, str) or not cell_id:
            return None
        descriptor = {}
        for part in cell_id.split("|"):
            name, sep, value = part.partition("=")
            if not sep or name not in self._by_name:
                return None
            if value not in self._by_name[name]["values"]:
                return None
            if name in descriptor:
                return None  # 同轴多次出现
            descriptor[name] = value
        if set(descriptor) != set(self.axis_names):
            return None  # 轴不齐
        # roundtrip 校验：descriptor 必须还原出同一 id（防等价但非规范的形式）
        if self.cell_id(descriptor) != cell_id:
            return None
        return descriptor

    # ---- 全格枚举 ----

    def all_cells(self) -> List[str]:
        """全格笛卡尔积（sampler 冷启动候选全集）。

        顺序确定：轴名序 × 值声明序。全格数超 _MAX_CELLS 时 raise（组合爆炸防御）。
        """
        if self._all_cells is None:
            total = 1
            for a in self._axes:
                total *= len(a["values"])
            if total > _MAX_CELLS:
                raise ValueError("特征空间全格数 %d 超上限 %d（组合爆炸防御，请缩减取值域或轴数）"
                                 % (total, _MAX_CELLS))
            axis_names = self.axis_names
            value_lists = [self._by_name[n]["values"] for n in axis_names]
            self._all_cells = [
                "|".join("%s=%s" % (n, v) for n, v in zip(axis_names, combo))
                for combo in product(*value_lists)
            ]
        return list(self._all_cells)  # 副本：调用方不得改缓存

    # ---- 相邻格抽样（builder 跨格灵感源，K-i8-3 裁决） ----

    def neighbor_pool(self, cell_id: str) -> List[str]:
        """相邻格池：与目标格**共享 ≥1 个轴值**的其它格（排除自身）。

        语义（迭代8）：相邻 = 至少一个维度取值相同——这些格在某个优化维度上
        可比（「换一个策略但保持数据布局」），是跨格重组灵感的自然来源。
        未知/非法 cell_id → []（优雅降级）。
        """
        desc = self.describe_cell(cell_id)
        if desc is None:
            return []
        out = []
        for cid in self.all_cells():
            if cid == cell_id:
                continue
            other = self.describe_cell(cid)
            if other is None:
                continue
            if any(other.get(name) == value for name, value in desc.items()):
                out.append(cid)
        return out

    def sample_neighbors(self, cell_id: str, k: int = 2,
                         rng: Optional[Any] = None) -> List[str]:
        """从相邻格池随机抽 k 个（不足时从全库随机补足；全库也空 → []）。

        :param k: 抽样个数（<=0 → []）
        :param rng: 随机源（提供 shuffle 的对象；默认 random 模块——测试注入固定种子）
        """
        if k <= 0:
            return []
        r = rng if rng is not None else random
        pool = list(self.neighbor_pool(cell_id))
        r.shuffle(pool)
        picked = pool[:k]
        if len(picked) < k:
            others = [c for c in self.all_cells()
                      if c != cell_id and c not in picked]
            r.shuffle(others)
            picked = picked + others[: k - len(picked)]
        return picked

    # ---- 访问器 ----

    @property
    def axis_names(self) -> List[str]:
        """轴名列表（已排序）。"""
        return [a["name"] for a in self._axes]

    @property
    def axes(self) -> List[Dict[str, Any]]:
        """轴声明副本（改副本不影响内部状态）。"""
        return [dict(a) for a in self._axes]

    # ---- 内部 ----

    @staticmethod
    def _norm_value(value: Any) -> str:
        """值规范化：bool → true/false；其余 str()。"""
        if isinstance(value, bool):
            return "true" if value else "false"
        return str(value)
