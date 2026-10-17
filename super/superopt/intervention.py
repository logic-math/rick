"""intervention.py — 干预队列摄取（job_47/task9）

人工干预不直写 events.jsonl（单写者 = py 调度器），而是落
`<ws>/interventions/` 队列文件，由调度器在迭代边界 drain 摄取并转写为
intervention 事件（source=human）——干预与真相源解耦（design-tree §2.0 M6）。

协议（与 Go 侧 handler.SuperEventIn 同构，task11 独立实现）：
  - 文件名：`<RFC3339 紧凑时间戳>-<纳秒余数 6 位>.json`（时间戳命名，字典序
    即时间序——drain 按文件名排序保证摄取顺序）
  - 内容：`{"type": <干预类型>, "payload": {...}, "created_at": <RFC3339 带时区>}`
  - 干预类型（设计树约定分类）：pause / weight / resurrect / env_change / stop

drain 语义（at-least-once）：
  - 扫描 `*.json` → 文件名排序 → 逐个读出 → **读后即删**（转交即清场）；
    崩溃窗口内的重复摄取无害——事件侧由 seq 幂等承载（单写者去重）。
  - 坏 JSON（解析失败 / 顶层非对象）→ 以 `bad_intervention` 条目留痕并同样
    删除（不阻塞队列、不无限重试坏文件）。
  - 非 `.json` 后缀文件不动（目录杂物不是干预）。

push 是 py 侧写入口（测试与 Go 侧 SuperEvent 复用同协议；Go 侧为独立实现，
不 import 本模块）。

stdlib-only，Python >=3.9 兼容（禁 PEP604 X|Y 联合类型注解）。
"""
import json
import os
import tempfile
import time
from typing import List

__all__ = ["InterventionQueue", "INTERVENTION_TYPES"]

# 设计树约定的干预分类（事件侧 payload.category 与此对齐；push 不做封闭校验，
# 新类型经由事件 schema 裁决——这里只是文档化常量，供消费方参考）
INTERVENTION_TYPES = frozenset({
    "pause",         # 暂停采样/观察
    "weight",        # 手动调采样权
    "resurrect",     # 复活被灭绝的格
    "env_change",    # env 定义变更（下迭代热更新生效）
    "stop",          # 停止调度
})


def _now_rfc3339() -> str:
    """RFC3339 带本地时区偏移（domain/bugs.md：无时区会被 Go time.Parse 拒绝）。"""
    return time.strftime("%Y-%m-%dT%H:%M:%S", time.localtime()) + _tz_suffix()


def _tz_suffix() -> str:
    off = time.timezone if time.daylight == 0 else time.altzone
    if off == 0:
        return "Z"
    sign = "-" if off > 0 else "+"   # time.timezone 语义与 ISO 偏移相反
    off = abs(off)
    return "%s%02d:%02d" % (sign, off // 3600, (off % 3600) // 60)


def _queue_name() -> str:
    """队列文件名：紧凑时间戳 + 纳秒余数（与 Go 侧 SuperEventIn 命名同构）。"""
    now = time.time()
    ts = time.strftime("%Y%m%dT%H%M%S", time.localtime(now))
    ns_rem = int((now % 1) * 1_000_000) % 1_000_000
    return "%s-%06d.json" % (ts, ns_rem)


class InterventionQueue:
    """`<ws>/interventions/` 队列的 py 侧读写器。

    用法::

        q = InterventionQueue(ws)
        q.push("stop", {"text": "收工", "category": "stop"})
        for item in q.drain():        # 调度器迭代边界摄取
            ...                        # 转写 intervention 事件（source=human）
    """

    def __init__(self, ws: str):
        self.ws = ws
        self.dir = os.path.join(ws, "interventions")

    # ---- 写入口 ----

    def push(self, etype: str, payload: dict) -> str:
        """写一条干预入队列，返回文件路径。

        - etype：非空字符串（类型契约见 INTERVENTION_TYPES；不做封闭枚举校验，
          避免与 Go 侧/事件 schema 的演进互相卡死）
        - payload：JSON 对象（事件侧要求 {"text", "category"}，此处只保证可序列化）
        - 落盘结构与 Go 侧 SuperEventIn 完全同构（type/payload/created_at）
        """
        if not isinstance(etype, str) or not etype:
            raise ValueError("etype must be a non-empty string")
        if not isinstance(payload, dict):
            raise ValueError("payload must be a JSON object (dict)")
        try:
            json.dumps(payload)
        except (TypeError, ValueError) as e:
            raise ValueError("payload is not JSON-serializable: %s" % e)

        os.makedirs(self.dir, exist_ok=True)
        entry = {
            "type": etype,
            "payload": payload,
            "created_at": _now_rfc3339(),
        }
        # 临时文件 + rename：坏写不污染队列（半写文件 rename 不会发生）
        fd, tmp = tempfile.mkstemp(dir=self.dir, prefix=".intv-", suffix=".tmp")
        try:
            with os.fdopen(fd, "w", encoding="utf-8") as f:
                json.dump(entry, f, ensure_ascii=False, indent=2)
                f.flush()
                os.fsync(f.fileno())
            path = os.path.join(self.dir, _queue_name())
            os.replace(tmp, path)
        except BaseException:
            try:
                os.unlink(tmp)
            except OSError:
                pass
            raise
        return path

    # ---- 摄取 ----

    def drain(self) -> List[dict]:
        """取走全部队列条目（按文件名 = 时间序），读后即删。

        返回条目结构：`{"type", "payload", "created_at", "file"}`；
        坏文件返回 `{"type": "bad_intervention", "file", "error"}`。
        目录不存在 / 为空 / 全是杂物 → 空列表。
        """
        try:
            names = sorted(os.listdir(self.dir))
        except FileNotFoundError:
            return []
        items = []            # type: List[dict]
        for name in names:
            if not name.endswith(".json"):
                continue  # 非 .json 杂物不动（.tmp 中间文件、README 等）
            path = os.path.join(self.dir, name)
            item = self._read_one(path, name)
            items.append(item)
            # 转交即删（含坏文件：不无限重试）
            try:
                os.unlink(path)
            except FileNotFoundError:
                pass  # 并发删除（Go 侧清理）——条目已读出，照常转交
        return items

    # ---- 内部 ----

    @staticmethod
    def _read_one(path: str, name: str) -> dict:
        """读单个队列文件为条目；坏 JSON → bad_intervention 条目。"""
        try:
            with open(path, "r", encoding="utf-8") as f:
                entry = json.load(f)
        except (ValueError, UnicodeDecodeError, OSError) as e:
            return {"type": "bad_intervention", "file": name, "error": str(e)}
        if not isinstance(entry, dict):
            return {"type": "bad_intervention", "file": name,
                    "error": "top-level JSON is not an object"}
        return {
            "type": entry.get("type"),
            "payload": entry.get("payload") or {},
            "created_at": entry.get("created_at"),
            "file": name,
        }
