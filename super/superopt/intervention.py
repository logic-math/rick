"""intervention.py — 干预队列摄取（job_47/task9 · job_56 task1 DB 化）

人工干预不直写事件表（单写者 = py 调度器），而是入 humans 表的
干预队列（queue_type='intervention'），由调度器在迭代边界 drain 摄取
并转写为 intervention 事件（source=human）——干预与真相源解耦。

DB 化（job_56 裁决 Q4）：humans 表两类队列——
  - intervention（干预队列）：human 提交的干预建议（本模块 push/drain）
  - proposal（提案队列）：runtime 产出、待 human 裁决的提案
    （db.SuperDB.push_proposal——adjust step7 实验自改进用，task3 接入）

drain 语义（at-least-once）：
  - 取走全部未应用干预（applied=0，按 id 序）→ 标 applied=1
    （行保留可审计——「是否被应用」字段承载，替代 json 时代的读后即删）
  - 坏 JSON 概念消失（DB 行总是合法 JSON——sqlite 行级完整性）

push 是 py 侧写入口（测试与 Go 侧 SuperEvent 复用同协议；Go 侧为独立实现，
不 import 本模块）。

stdlib-only，Python >=3.9 兼容（禁 PEP604 X|Y 联合类型注解）。
"""
import os
from typing import List

from .db import db_for_ws

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


class InterventionQueue:
    """humans 表干预队列的 py 侧读写器（签名与 json 时代兼容——ws 定位 DB）。

    用法::

        q = InterventionQueue(ws)
        q.push("stop", {"text": "收工", "category": "stop"})
        for item in q.drain():        # 调度器迭代边界摄取
            ...                        # 转写 intervention 事件（source=human）
    """

    def __init__(self, ws: str):
        self.ws = ws
        self.dir = os.path.join(ws, "interventions")  # 兼容旧属性（Go 侧/测试引用）
        self._db = db_for_ws(ws)

    # ---- 写入口 ----

    def push(self, etype: str, payload: dict) -> str:
        """写一条干预入队列，返回条目标识（humans#<id>）。

        - etype：非空字符串（类型契约见 INTERVENTION_TYPES；不做封闭枚举校验，
          避免与 Go 侧/事件 schema 的演进互相卡死）
        - payload：JSON 对象（事件侧要求 {"text", "category"}，此处只保证可序列化）
        """
        if not isinstance(etype, str) or not etype:
            raise ValueError("etype must be a non-empty string")
        if not isinstance(payload, dict):
            raise ValueError("payload must be a JSON object (dict)")
        import json
        try:
            json.dumps(payload)
        except (TypeError, ValueError) as e:
            raise ValueError("payload is not JSON-serializable: %s" % e)
        hid = self._db.push_intervention(etype, payload)
        return "humans#%d" % hid

    # ---- 摄取 ----

    def drain(self) -> List[dict]:
        """取走全部未应用干预（按 id 序），标 applied=1。

        返回条目结构：`{"type", "payload", "created_at", "file"}`（file=
        "humans#<id>"——与 json 时代 drain 条目键兼容，消费方零改动）。
        """
        return self._db.drain_interventions()

    # ---- 未应用干预查询（不消费——step1 的 human 价值判断注入用，task3 接入） ----

    def pending(self) -> List[dict]:
        """未应用干预（不消费）：[{id, type, payload, created_at}]。"""
        return self._db.pending_interventions()
