"""archive.py — 精英格档案（MAP-Elites archive，job_47 task2）

单文件 JSON 档案（archive.json）：
  - 格 → 精英映射（descriptor + elite{code_path, perf_value, perf_type, perf_desc,
    score, lineage, confidence, admitted_at, gen}）
  - history：同格被替换的旧精英（上限 HISTORY_LIMIT=20，最旧挤出，防档案膨胀）
  - weight / visit / sample_count / last_sampled_at：采样统计（visit 由 Sampler 经
    回调递增——本模块只承载）
  - pending：延迟入库队列（新格首解 / 容差带内 / 来源格已降权 三判定，L2-Q6）

设计契约（design-tree §2.3 / task2 spec）：
  - 单写者：档案文件由持有方（调度器）独写；本类不做进程锁
  - 事件先于档案：events.jsonl 是唯一真相源，档案是投影——崩溃后可从事件流重建
  - 不删格：set_weight 降 0 = 功能性灭绝，格子与精英永久保留（防信息丢失）
  - 方向感知：perf_desc="low"（越小越好，如 runtime_ms）/ "high"（越大越好，如 score），
    由构造方注入（读 env）并存入 meta——admit 的同格替换条件据此判断
  - 时间戳一律 RFC3339 带时区（domain/bugs.md 坑位：无时区会被 Go time.Parse 拒绝）

本模块与 task1（events）/task3（Sampler）零 import 耦合：cell_id 是不透明字符串，
descriptor 只是数据；采样决策不在本模块（Sampler 职责）。

Python ≥3.9 兼容（禁 PEP604 X|Y 注解，用 Optional[...]）。
"""
import copy
import json
import os
import tempfile
import threading
from datetime import datetime, timezone, timedelta
from typing import Optional

# 东八区（项目惯例，见 domain/bugs.md：tasks.json updated_at 用 +08:00）
_TZ = timezone(timedelta(hours=8))

# pending reason 合法集（L2-Q6 延迟入库三判定）
PENDING_REASONS = frozenset({"new_cell", "tolerance_band", "low_weight_src"})

# 精英字段（六字段基座 + lineage/置信度）
_ELITE_FIELDS = (
    "code_path", "perf_value", "perf_type", "perf_desc",
    "score", "lineage", "confidence", "admitted_at", "gen",
)

HISTORY_LIMIT = 20

# 特征格四要素之「方法描述」（KR2 / 迭代8）：{name, method, source_refs}
# name=短名；method=方法描述（自由文本，可引用论文/调研文档地址）；
# source_refs=引用清单（arxiv id / 文档路径，供 builder 深度检索）。
DESCRIPTION_FIELDS = ("name", "method", "source_refs")


def _now() -> str:
    """RFC3339 带时区时间戳（+08:00）。"""
    return datetime.now(_TZ).isoformat(timespec="seconds")


def _empty_data(perf_desc: str) -> dict:
    return {
        "meta": {
            "env": "",
            "feature_axes": [],
            "perf_desc": perf_desc,
            "updated_at": _now(),
        },
        "cells": {},
        "pending": [],
    }


def _elite_from_entry(entry: dict) -> dict:
    """从 admit entry 提取精英字段。

    entry 为 cell 形态：{"descriptor": {...}, "elite": {code_path, perf_value, perf_type,
    perf_desc, score, lineage, confidence, admitted_at, gen}}。
    兼容顶层扁平形态（entry 直接携带精英字段）——以 "elite" 键优先。
    """
    src = entry.get("elite") if isinstance(entry.get("elite"), dict) else entry
    return {k: copy.deepcopy(src.get(k)) for k in _ELITE_FIELDS}


def _is_better(new_value, old_value, perf_desc: str) -> bool:
    """同格替换条件（方向感知）。严格更优才替换；相等不算。"""
    if new_value is None or old_value is None:
        # 缺值不参与比较：视为不可替换（保守）
        return False
    if perf_desc == "high":
        return new_value > old_value
    return new_value < old_value  # 默认 low（越小越好）


class Archive:
    """精英格档案。

    用法（持有方 = 调度器/测试）：
        a = Archive(path, perf_desc="low")
        a.load()                       # 读档（文件缺失 = 从空档开始）
        a.admit(cell_id, entry)        # 内存变更
        a.set_weight(...); a.add_pending(...)
        a.save()                       # 持久化（tmp + fsync + rename 原子替换）

    变更方法只改内存；save() 才落盘（调用方控制持久化时机）。
    """

    HISTORY_LIMIT = HISTORY_LIMIT

    def __init__(self, path: str, perf_desc: str = "low"):
        if perf_desc not in ("low", "high"):
            raise ValueError("perf_desc must be 'low' or 'high', got %r" % (perf_desc,))
        self.path = path
        self._perf_desc = perf_desc
        self._data = _empty_data(perf_desc)
        # 并发实验（task44/task45）：多 worker 线程读写档案 → 全部公开方法加锁。
        # RLock（可重入）：bump_visit → set_visit 等嵌套调用不会自锁。
        self._lock = threading.RLock()

    # ---- 读档 / 落盘 ----

    def load(self) -> None:
        """读档。文件不存在 → 空档；损坏（非法 JSON）→ 抛错（档案是事件流投影，
        宁可炸也不静默清空——由调用方决定从事件流重建）。"""
        if not os.path.exists(self.path):
            with self._lock:
                self._data = _empty_data(self._perf_desc)
            return
        with open(self.path, encoding="utf-8") as f:
            data = json.load(f)
        # 补齐缺省键（向后兼容旧档）
        data.setdefault("meta", {})
        data["meta"].setdefault("env", "")
        data["meta"].setdefault("feature_axes", [])
        data["meta"].setdefault("perf_desc", self._perf_desc)
        data["meta"].setdefault("updated_at", _now())
        data.setdefault("cells", {})
        data.setdefault("pending", [])
        for cell in data["cells"].values():
            # 迭代8 向后兼容：旧档无 description（四要素之方法描述）→ 补空 dict
            cell.setdefault("description", {})
            cell.setdefault("history", [])
            cell.setdefault("weight", 1.0)
            cell.setdefault("visit", 0)
            cell.setdefault("sample_count", 0)
            cell.setdefault("last_sampled_at", None)
            # 迭代9/五要素兼容：旧档无精英代码路径与实验记录关联 → 补 None
            cell.setdefault("elite_code_path", None)
            cell.setdefault("elite_record_path", None)
        with self._lock:
            self._data = data

    def save(self) -> None:
        """原子落盘：临时文件（同目录）→ fsync → os.replace。
        任一步失败：旧档保持完整（replace 之前不动旧文件）。

        线程安全：写盘全程持锁（并发 worker 的 save 串行化，不会交叉写）。"""
        with self._lock:
            self._data["meta"]["updated_at"] = _now()
            self._data["meta"]["perf_desc"] = self._perf_desc
            d = os.path.dirname(os.path.abspath(self.path)) or "."
            fd, tmp = tempfile.mkstemp(prefix=".archive-", suffix=".tmp", dir=d)
            try:
                with os.fdopen(fd, "w", encoding="utf-8") as f:
                    json.dump(self._data, f, ensure_ascii=False, indent=2)
                    f.flush()
                    os.fsync(f.fileno())
                os.replace(tmp, self.path)
            except BaseException:
                # 清理半写临时文件（不留垃圾；旧档未动）
                try:
                    os.unlink(tmp)
                except OSError:
                    pass
                raise

    # ---- 访问器（副本，防外部直改） ----

    @property
    def cells(self) -> dict:
        with self._lock:
            return copy.deepcopy(self._data["cells"])

    @property
    def meta(self) -> dict:
        with self._lock:
            return copy.deepcopy(self._data["meta"])

    def get_elite(self, cell_id: str) -> Optional[dict]:
        with self._lock:
            cell = self._data["cells"].get(cell_id)
            if not cell or cell.get("elite") is None:
                return None
            return copy.deepcopy(cell["elite"])

    def get_record_path(self, cell_id: str) -> Optional[str]:
        """五要素关联（task44）：该格当前最佳精英对应的实验记录路径。

        返回 `elite_record_path`（admit 时由调度器传入的该迭代
        `experiment_record.md`）；无精英/无记录/旧档 → None。
        """
        with self._lock:
            cell = self._data["cells"].get(cell_id)
            if not cell:
                return None
            p = cell.get("elite_record_path")
            return p if isinstance(p, str) and p else None

    # ---- 变更（内存） ----

    def _descriptor_from_entry(self, entry: dict) -> dict:
        d = entry.get("descriptor")
        return copy.deepcopy(d) if isinstance(d, dict) else {}

    def admit(self, cell_id: str, entry: dict,
              record_path: Optional[str] = None) -> bool:
        """同格替换条件：新 perf_value 严格优于旧（perf_desc 方向感知），或旧格为空。

        entry 形态：{"descriptor": {...}, "elite": {perf_value, code_path, ...}}
        （兼容顶层扁平携带精英字段）。
        record_path：该精英对应迭代的 experiment_record.md（五要素关联，可选）。

        - 新格 / 无精英格 → 入档（True）
        - 更优 → 替换：旧精英入 history（上限 HISTORY_LIMIT，最旧挤出）（True）
        - 更差或相等 → 拒绝（False），档案不变

        入档同时同步五要素字段：`elite_code_path`（当前最佳精英代码）、
        `elite_record_path`（其实验记录）。
        """
        with self._lock:
            elite = _elite_from_entry(entry)
            code_path = elite.get("code_path")
            cell = self._data["cells"].get(cell_id)
            if cell is None:
                cell = {
                    "descriptor": self._descriptor_from_entry(entry),
                    "description": {},
                    "elite": elite,
                    "history": [],
                    "weight": 1.0,
                    "visit": 0,
                    "sample_count": 0,
                    "last_sampled_at": None,
                }
                self._data["cells"][cell_id] = cell
                admitted = True
            else:
                old_elite = cell.get("elite")
                if old_elite is None:
                    cell["elite"] = elite
                    cell.setdefault("descriptor", self._descriptor_from_entry(entry))
                    admitted = True
                elif _is_better(elite.get("perf_value"),
                                old_elite.get("perf_value"),
                                self._data["meta"].get("perf_desc", self._perf_desc)):
                    history = cell.setdefault("history", [])
                    history.append(copy.deepcopy(old_elite))
                    if len(history) > self.HISTORY_LIMIT:
                        del history[: len(history) - self.HISTORY_LIMIT]
                    cell["elite"] = elite
                    admitted = True
                else:
                    admitted = False
            if admitted:
                # 五要素：code_path = 当前最佳精英代码；record_path = 其实验记录
                cell["elite_code_path"] = (
                    code_path if isinstance(code_path, str) and code_path else None)
                cell["elite_record_path"] = (
                    record_path if isinstance(record_path, str) and record_path
                    else None)
            return admitted

    def set_visit(self, cell_id: str, visit: int) -> None:
        """崩溃恢复落点：以事件流投影的 visit 覆盖内存档案（真相在事件；
        job_47 e2e kill-9 场景实测：缺此方法时 _recover 静默跳过，重启后
        档案 visit 回 0）。未知格：创建格骨架（同 set_weight 语义）。"""
        if visit < 0:
            raise ValueError("visit must be >= 0, got %r" % (visit,))
        with self._lock:
            cell = self._data["cells"].get(cell_id)
            if cell is None:
                cell = {
                    "descriptor": {},
                    "description": {},
                    "elite": None,
                    "history": [],
                    "weight": 1.0,
                    "visit": 0,
                    "sample_count": 0,
                    "last_sampled_at": None,
                    "elite_code_path": None,
                    "elite_record_path": None,
                }
                self._data["cells"][cell_id] = cell
            cell["visit"] = int(visit)
            cell["last_sampled_at"] = _now()

    def bump_visit(self, cell_id: str) -> None:
        """visit 递增入口（Sampler.on_visit 回调的档案落地形态：事件之外
        档案也同步计数——投影恢复与运行时计数双链一致）。"""
        with self._lock:
            cell = self._data["cells"].get(cell_id)
            if cell is None:
                self.set_visit(cell_id, 1)
                return
            cell["visit"] = int(cell.get("visit") or 0) + 1
            cell["last_sampled_at"] = _now()

    def set_weight(self, cell_id: str, weight: float) -> None:
        """adjust 软动作落点。weight=0 = 功能性灭绝（**不删格**）。
        未知格：创建格骨架（elite=None）——adjust 可对未入档格预设权重。"""
        if weight < 0:
            raise ValueError("weight must be >= 0, got %r" % (weight,))
        with self._lock:
            cell = self._data["cells"].get(cell_id)
            if cell is None:
                cell = {
                    "descriptor": {},
                    "description": {},
                    "elite": None,
                    "history": [],
                    "weight": 1.0,
                    "visit": 0,
                    "sample_count": 0,
                    "last_sampled_at": None,
                    "elite_code_path": None,
                    "elite_record_path": None,
                }
                self._data["cells"][cell_id] = cell
            cell["weight"] = weight

    def set_description(self, cell_id: str, description: "dict") -> None:
        """特征格四要素之「方法描述」落点（KR2）：{name, method, source_refs}。

        - name：短名（如「M 维四分 + 2-D np.dot」）
        - method：方法描述（自由文本；可引用论文/调研文档地址，供 builder 深度检索）
        - source_refs：引用清单（arxiv id / 文档路径），规范化为 list[str]

        语义：adjust 的 modify_cell 动作（改进格描述）/ new_cell 动作（新建格带描述）。
        未知格：创建格骨架（同 set_weight 语义——adjust 可为未入档格预设描述）。
        非法输入：非 dict → ValueError（脏数据早失败）；未知键丢弃（schema 收敛）。
        """
        if not isinstance(description, dict):
            raise ValueError("description 必须是 dict，收到 %r" % (description,))
        normalized = {}
        for k in DESCRIPTION_FIELDS:
            if k not in description:
                continue
            if k == "source_refs":
                refs = description.get(k)
                if refs is None:
                    normalized[k] = []
                elif isinstance(refs, (list, tuple)):
                    normalized[k] = [str(r) for r in refs if str(r).strip()]
                else:
                    normalized[k] = [str(refs)]
                continue
            normalized[k] = str(description.get(k) or "")
        with self._lock:
            cell = self._data["cells"].get(cell_id)
            if cell is None:
                cell = {
                    "descriptor": {},
                    "description": {},
                    "elite": None,
                    "history": [],
                    "weight": 1.0,
                    "visit": 0,
                    "sample_count": 0,
                    "last_sampled_at": None,
                    "elite_code_path": None,
                    "elite_record_path": None,
                }
                self._data["cells"][cell_id] = cell
            cell["description"] = normalized

    # ---- pending（延迟入库队列） ----

    def add_pending(self, cell_id: str, code_path: str, reason: str) -> None:
        """同格去重（last-wins）：同格再次 pending 时替换旧项——与
        lineage.project_state 的投影语义对齐（一格一个待复测项；旧 code_path
        被新精英覆盖后，复测对象应是新的。e2e 实测：不去重会两侧 pending 集
        合不一致）。"""
        if reason not in PENDING_REASONS:
            raise ValueError(
                "reason must be one of %s, got %r" % (sorted(PENDING_REASONS), reason)
            )
        with self._lock:
            self._data["pending"] = [
                p for p in self._data["pending"] if p.get("cell_id") != cell_id
            ]
            self._data["pending"].append({
                "cell_id": cell_id,
                "code_path": code_path,
                "reason": reason,
                "added_at": _now(),
            })

    def drain_pending(self) -> list:
        """取走全部 pending（重评估消费方）。内存队列即清空；持久化由调用方 save。"""
        with self._lock:
            out = self._data["pending"]
            self._data["pending"] = []
            return out
