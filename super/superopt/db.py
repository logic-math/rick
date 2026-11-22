"""db.py — super 进化元信息 sqlite 单库（job_56 task1：DB 化地基）

`<ws>/super.db` 承载全部进化元信息（json/jsonl 存储路径废弃——旧工作区
不迁移不兼容；新工作区由 config.init_ws 建 super.db）。

五张核心表（grilling KR1 裁决定稿）：
  - cells        特征格表：cell_id(PK) / name(名称) / method_desc(优化方法描述，
                 详细 TEXT) / best_experiment_id(当前最优实验 ID) / weight(权重值)
                 + state_json（archive 兼容的完整格状态投影：descriptor/elite/
                 history/visit/...——核心列可 SQL 直查，完整状态保投影）
  - experiments  实验表：experiment_id(PK，格式 <iteration>-<YYYYMMDDHHMM>
                 迭代号-年月日时分；同 iter 同分钟并行实验撞号自动加 -2/-3 后缀)
                 / iteration / workdir(=experiment_id，工作目录名) / record_path
                 (实验记录 md 路径) / session_id(pi agent sessionID，行为轨迹
                 还原) / cell_id / score(REAL 可空) / status / created_at
  - humans       人类价值判断表（两类队列）：id(PK 自增) / queue_type
                 ('intervention'=干预队列[human 提交] | 'proposal'=提案队列
                 [runtime 产出待 human 裁决]) / content(JSON TEXT) / applied
                 (0/1 是否已被应用) / created_at / applied_at
  - events       系统级事件日志表：seq(PK 自增) / ts / event_type / source /
                 experiment_id(可空——events 的实验分组语义 cell#iterN) /
                 experiment_seq / payload(JSON TEXT)——查询此表即感知系统状态
                 变化
  - resources    资源表：resource_type('experiment_slot'=实验槽 |
                 'env_slot'=环境槽) / resource_id / description / quota(额度数)
                 ——一条记录代表一个资源位；holder_experiment_id 承载占用

内部表（核心五表之外的分析副产品/投影）：
  - archive_meta   archive 的 meta/pending 投影（key TEXT PK, value_json）
  - ledger_entries 实验台账（id/ts/experiment_id/payload JSON）——ledger.py 后端

架构契约：
  - 单写者：py 调度器是 super.db 唯一写者——由调度器进程锁
    （acquire_scheduler_lock：env/scheduler.pid 存活检测）强保证；
    Go 侧与 tools 只读（SELECT）
  - 并发：进程内多线程共享一连接（check_same_thread=False + RLock）；
    WAL 模式（PRAGMA journal_mode=WAL）提高外部读并发
  - 每工作区单例（db_for_ws）——同 ws 各模块共享连接与锁
  - 事件语义不变：events 表由 events.EventWriter 封装（本模块只提供原语）；
    experiment_id 分组语义（cell#iterN）与 experiments 表 PK（工作目录名）
    是两种 ID（表内自洽，DDL 注释标注）

stdlib-only（标准库 sqlite3），Python >=3.9 兼容（禁 PEP604 注解）。
"""
import json
import os
import sqlite3
import threading
import time
from datetime import datetime, timezone, timedelta
from typing import Any, Dict, List, Optional

__all__ = [
    "DB_FILENAME",
    "SchedulerLockError",
    "SuperDB",
    "db_for_ws",
    "db_path_for_ws",
    "acquire_scheduler_lock",
    "make_experiment_timestamp_id",
]

DB_FILENAME = "super.db"

_TZ = timezone(timedelta(hours=8))  # 项目惯例（+08:00，见 domain/bugs.md）

# ---- DDL（幂等：CREATE TABLE IF NOT EXISTS）----

_DDL = [
    # 特征格表（核心五列 + 状态投影）
    """
    CREATE TABLE IF NOT EXISTS cells (
        cell_id             TEXT PRIMARY KEY,
        name                TEXT NOT NULL DEFAULT '',
        method_desc         TEXT NOT NULL DEFAULT '',
        best_experiment_id  TEXT,
        weight              REAL NOT NULL DEFAULT 1.0,
        state_json          TEXT NOT NULL DEFAULT '{}'
    )
    """,
    # 实验表（experiment_id = <iteration>-<YYYYMMDDHHMM> = 工作目录名）
    """
    CREATE TABLE IF NOT EXISTS experiments (
        experiment_id   TEXT PRIMARY KEY,
        iteration       INTEGER NOT NULL,
        workdir         TEXT,
        record_path     TEXT,
        builder_session_id TEXT,
        coding_session_id  TEXT,
        debug_session_id   TEXT,
        cell_id         TEXT,
        score           REAL,
        status          TEXT NOT NULL DEFAULT 'running',
        created_at      TEXT NOT NULL
    )
    """,
    # adjust 运行表（job_56 第三轮审计：每次 adjust 流水线一行——
    # 各步 pi session + 迭代目录，行为轨迹观察入口）
    """
    CREATE TABLE IF NOT EXISTS adjusts (
        adjust_id       TEXT PRIMARY KEY,
        iteration       INTEGER NOT NULL,
        adjust_dir      TEXT NOT NULL,
        sessions        TEXT NOT NULL,
        decisions_path  TEXT,
        agent_valid     INTEGER,
        created_at      TEXT NOT NULL
    )
    """,
    # 人类价值判断表（两类队列：干预 / 提案）
    """
    CREATE TABLE IF NOT EXISTS humans (
        id          INTEGER PRIMARY KEY AUTOINCREMENT,
        queue_type  TEXT NOT NULL CHECK (queue_type IN ('intervention', 'proposal')),
        content     TEXT NOT NULL,
        applied     INTEGER NOT NULL DEFAULT 0,
        created_at  TEXT NOT NULL,
        applied_at  TEXT
    )
    """,
    # 系统事件日志表（seq 自增 = 事件序；experiment_id 为 events 分组语义）
    """
    CREATE TABLE IF NOT EXISTS events (
        seq            INTEGER PRIMARY KEY AUTOINCREMENT,
        ts             TEXT NOT NULL,
        event_type     TEXT NOT NULL,
        source         TEXT NOT NULL,
        experiment_id  TEXT,
        experiment_seq INTEGER,
        payload        TEXT NOT NULL
    )
    """,
    """
    CREATE INDEX IF NOT EXISTS idx_events_experiment ON events(experiment_id)
    """,
    # 资源表（一条记录 = 一个资源位；holder 承载占用）
    """
    CREATE TABLE IF NOT EXISTS resources (
        resource_type        TEXT NOT NULL,
        resource_id          TEXT NOT NULL,
        description          TEXT,
        quota                INTEGER NOT NULL DEFAULT 1,
        holder_experiment_id TEXT,
        acquired_at          TEXT,
        PRIMARY KEY (resource_type, resource_id)
    )
    """,
    # 内部：archive meta/pending 投影
    """
    CREATE TABLE IF NOT EXISTS archive_meta (
        key        TEXT PRIMARY KEY,
        value_json TEXT NOT NULL
    )
    """,
    # 内部：实验台账（ledger.py 后端）
    """
    CREATE TABLE IF NOT EXISTS ledger_entries (
        id            INTEGER PRIMARY KEY AUTOINCREMENT,
        ts            TEXT NOT NULL,
        experiment_id TEXT,
        payload       TEXT NOT NULL
    )
    """,
]

RESOURCE_EXPERIMENT_SLOT = "experiment_slot"
RESOURCE_ENV_SLOT = "env_slot"


def _now() -> str:
    """RFC3339 带时区时间戳（+08:00，与 events/archive 一致）。"""
    return datetime.now(_TZ).isoformat(timespec="seconds")


def make_experiment_timestamp_id(iteration: int,
                                 minute_ts: Optional[str] = None) -> str:
    """实验表 PK/工作目录名：`<iteration>-<YYYYMMDDHHMM>`（迭代号-年月日时分）。

    minute_ts 可注入（测试确定性）；缺省取当前分钟。
    """
    ts = minute_ts or time.strftime("%Y%m%d%H%M", time.localtime())
    return "%s-%s" % (iteration, ts)


class SchedulerLockError(RuntimeError):
    """调度器进程锁冲突：同 env 已有存活调度器进程。"""


def acquire_scheduler_lock(env_dir: str) -> int:
    """py 调度器进程锁（KR-B 裁决）：env 下 scheduler.pid 存活检测。

    - 无锁文件 / 锁文件指向已死进程 → 写入当前 PID，返回 pid
    - 锁文件指向存活进程 → raise SchedulerLockError（报错含存活 PID 与
      「rick super stop 回收旧进程」建议——一会话一 env 唯一调度器）
    - 正常退出/被杀不清锁文件（下次启动校验进程死活即可）
    """
    path = os.path.join(env_dir, "scheduler.pid")
    if os.path.isfile(path):
        try:
            with open(path, "r", encoding="utf-8") as f:
                raw = f.read().strip()
            old_pid = int(raw)
        except (OSError, ValueError):
            old_pid = 0
        if old_pid > 0:
            try:
                os.kill(old_pid, 0)  # 存活检测（0 信号不投递）
                raise SchedulerLockError(
                    "调度器进程锁冲突：env %s 已有存活调度器进程 pid=%d。"
                    "一个 super 会话对一个 env 只允许一个 py 调度器——"
                    "请先 `rick super stop` 回收旧进程后再重启。" % (env_dir, old_pid))
            except ProcessLookupError:
                pass  # 旧进程已死：锁文件失效，放行覆盖
            except PermissionError:
                # 进程存在但属于他人（权限拒绝≠不存在）：保守拒绝
                raise SchedulerLockError(
                    "调度器进程锁冲突：env %s 的锁指向进程 pid=%d（存活但非本用户"
                    "——PermissionError）。请先 `rick super stop` 回收旧进程。"
                    % (env_dir, old_pid))
    os.makedirs(env_dir, exist_ok=True)
    pid = os.getpid()
    with open(path, "w", encoding="utf-8") as f:
        f.write(str(pid))
    return pid


class SuperDB:
    """super.db 的进程内访问器（每工作区一实例——用 db_for_ws 取单例）。

    线程安全：全部公开方法持 RLock（调度器内槽 worker/adjust 线程共享）。
    崩溃安全：sqlite WAL——进程崩溃不损库（已提交事务保留）。
    """

    def __init__(self, ws: str):
        self.ws = ws
        self.path = os.path.join(ws, DB_FILENAME)
        os.makedirs(ws, exist_ok=True)
        self._lock = threading.RLock()
        self._conn = sqlite3.connect(self.path, check_same_thread=False,
                                     timeout=30.0)
        self._conn.row_factory = sqlite3.Row
        with self._lock:
            self._conn.execute("PRAGMA journal_mode=WAL")
            self._conn.execute("PRAGMA synchronous=NORMAL")
            for ddl in _DDL:
                self._conn.execute(ddl)
            self._migrate()
            self._conn.commit()

    def _migrate(self) -> None:
        """轻量列迁移（job_56 第三轮审计：experiments 三角色 session 列）。

        旧库（session_id 单列）：ALTER 加三列（旧值保留原列——兼容读，
        新写走三列）；新库建表已是三列（ALTER 幂等跳过）。
        """
        try:
            cols = {r["name"] for r in self._conn.execute(
                "PRAGMA table_info(experiments)").fetchall()}
            for col in ("builder_session_id", "coding_session_id",
                        "debug_session_id"):
                if col not in cols:
                    self._conn.execute(
                        "ALTER TABLE experiments ADD COLUMN %s TEXT" % col)
        except sqlite3.Error:
            pass  # 表不存在等竞态：建表循环刚建过，忽略

    # ---- 关闭 ----

    def close(self) -> None:
        with self._lock:
            try:
                self._conn.close()
            except sqlite3.Error:
                pass

    # ---- events 原语（events.EventWriter 的后端） ----

    def append_event(self, event_type: str, source: str, payload: dict,
                     experiment_id: Optional[str] = None,
                     experiment_seq: Optional[int] = None,
                     ts: Optional[str] = None) -> Dict[str, Any]:
        """插入一条事件（返回完整事件 dict：ts/seq/type/source/payload[+分组]）。"""
        ev = {
            "ts": ts or _now(),
            "type": event_type,
            "source": source,
            "payload": payload,
        }
        with self._lock:
            if experiment_id is not None:
                if experiment_seq is None:
                    row = self._conn.execute(
                        "SELECT MAX(experiment_seq) FROM events "
                        "WHERE experiment_id=?", (experiment_id,)).fetchone()
                    base = row[0] if row and row[0] is not None else 0
                    experiment_seq = base + 1
                ev["experiment_id"] = experiment_id
                ev["experiment_seq"] = int(experiment_seq)
            cur = self._conn.execute(
                "INSERT INTO events (ts, event_type, source, experiment_id, "
                "experiment_seq, payload) VALUES (?,?,?,?,?,?)",
                (ev["ts"], ev["type"], ev["source"],
                 ev.get("experiment_id"), ev.get("experiment_seq"),
                 json.dumps(payload, ensure_ascii=False)))
            self._conn.commit()
            ev["seq"] = int(cur.lastrowid)
        return ev

    @staticmethod
    def _row_to_event(row: sqlite3.Row) -> Dict[str, Any]:
        ev = {
            "seq": int(row["seq"]),
            "ts": row["ts"],
            "type": row["event_type"],
            "source": row["source"],
            "payload": json.loads(row["payload"]),
        }
        if row["experiment_id"] is not None:
            ev["experiment_id"] = row["experiment_id"]
            ev["experiment_seq"] = int(row["experiment_seq"] or 0)
        return ev

    def read_events(self) -> List[Dict[str, Any]]:
        """全量读取（seq 升序；payload 反序列化）。"""
        with self._lock:
            rows = self._conn.execute(
                "SELECT * FROM events ORDER BY seq").fetchall()
        return [self._row_to_event(r) for r in rows]

    def read_experiment_events(self, experiment_id: str) -> List[Dict[str, Any]]:
        """单实验事件流（experiment_seq 升序）。"""
        if not isinstance(experiment_id, str) or not experiment_id:
            return []
        with self._lock:
            rows = self._conn.execute(
                "SELECT * FROM events WHERE experiment_id=? "
                "ORDER BY experiment_seq, seq", (experiment_id,)).fetchall()
        return [self._row_to_event(r) for r in rows]

    def experiment_ids(self) -> List[str]:
        """出现过的全部分组 experiment_id（首现顺序）。"""
        with self._lock:
            rows = self._conn.execute(
                "SELECT experiment_id FROM events WHERE experiment_id IS NOT NULL "
                "GROUP BY experiment_id ORDER BY MIN(seq)").fetchall()
        return [r[0] for r in rows]

    def last_seq(self) -> int:
        with self._lock:
            row = self._conn.execute(
                "SELECT MAX(seq) FROM events").fetchone()
        return int(row[0]) if row and row[0] is not None else 0

    # ---- cells（archive 后端：核心列 + 完整状态投影） ----

    def upsert_cell_state(self, cell_id: str, cell: dict,
                          name: Optional[str] = None,
                          method_desc: Optional[str] = None,
                          best_experiment_id: Optional[str] = None,
                          weight: Optional[float] = None) -> None:
        """upsert 一格：核心列（可 SQL 直查）+ state_json（完整状态投影）。

        核心列优先取显式参数，缺省从 cell dict 投影：
          name ← cell.description.name；method_desc ← cell.description.method；
          best_experiment_id ← cell.elite_record_path 关联（不推导，仅显式）；
          weight ← cell.weight。
        """
        desc = cell.get("description") if isinstance(
            cell.get("description"), dict) else {}
        if name is None:
            name = str(desc.get("name") or "")
        if method_desc is None:
            method_desc = str(desc.get("method") or "")
        if weight is None:
            try:
                weight = float(cell.get("weight", 1.0))
            except (TypeError, ValueError):
                weight = 1.0
        state = json.dumps(cell, ensure_ascii=False)
        with self._lock:
            self._conn.execute(
                "INSERT INTO cells (cell_id, name, method_desc, "
                "best_experiment_id, weight, state_json) VALUES (?,?,?,?,?,?) "
                "ON CONFLICT(cell_id) DO UPDATE SET name=excluded.name, "
                "method_desc=excluded.method_desc, "
                "best_experiment_id=COALESCE(excluded.best_experiment_id, "
                "cells.best_experiment_id), weight=excluded.weight, "
                "state_json=excluded.state_json",
                (cell_id, name, method_desc, best_experiment_id, weight, state))
            self._conn.commit()

    def load_cells(self) -> Dict[str, Dict[str, Any]]:
        """全部格的完整状态（state_json 反序列化）。"""
        with self._lock:
            rows = self._conn.execute(
                "SELECT cell_id, state_json FROM cells").fetchall()
        out = {}
        for r in rows:
            try:
                out[r[0]] = json.loads(r[1])
            except (ValueError, TypeError):
                out[r[0]] = {}
        return out

    # ---- archive meta/pending 投影 ----

    def get_meta(self) -> Dict[str, Any]:
        with self._lock:
            row = self._conn.execute(
                "SELECT value_json FROM archive_meta WHERE key='meta'").fetchone()
        return json.loads(row[0]) if row else {}

    def save_meta(self, meta: Dict[str, Any]) -> None:
        with self._lock:
            self._conn.execute(
                "INSERT INTO archive_meta (key, value_json) VALUES ('meta',?) "
                "ON CONFLICT(key) DO UPDATE SET value_json=excluded.value_json",
                (json.dumps(meta, ensure_ascii=False),))
            self._conn.commit()

    def get_pending(self) -> List[Dict[str, Any]]:
        with self._lock:
            row = self._conn.execute(
                "SELECT value_json FROM archive_meta WHERE key='pending'").fetchone()
        return json.loads(row[0]) if row else []

    def save_pending(self, pending: List[Dict[str, Any]]) -> None:
        with self._lock:
            self._conn.execute(
                "INSERT INTO archive_meta (key, value_json) VALUES ('pending',?) "
                "ON CONFLICT(key) DO UPDATE SET value_json=excluded.value_json",
                (json.dumps(pending, ensure_ascii=False),))
            self._conn.commit()

    # ---- humans 两类队列 ----

    def _push_human(self, queue_type: str, content: dict) -> int:
        with self._lock:
            cur = self._conn.execute(
                "INSERT INTO humans (queue_type, content, applied, created_at) "
                "VALUES (?,?,0,?)",
                (queue_type, json.dumps(content, ensure_ascii=False), _now()))
            self._conn.commit()
            return int(cur.lastrowid)

    @staticmethod
    def _row_to_human(row: sqlite3.Row) -> Dict[str, Any]:
        return {
            "id": int(row["id"]),
            "queue_type": row["queue_type"],
            "content": json.loads(row["content"]),
            "applied": bool(row["applied"]),
            "created_at": row["created_at"],
            "applied_at": row["applied_at"],
        }

    def push_intervention(self, etype: str, payload: dict) -> int:
        """干预入队（human 提交的价值判断；drain 摄取时标 applied）。"""
        return self._push_human("intervention",
                                {"type": etype, "payload": payload})

    def pending_interventions(self) -> List[Dict[str, Any]]:
        """未应用干预（不消费——step1 的 human 价值判断注入用）。

        返回展开形态：{id, type, payload, created_at}（content 展开为 type/payload）。
        """
        out = []
        for h in self._query_humans("intervention", applied=0):
            out.append({
                "id": h["id"],
                "type": h["content"].get("type"),
                "payload": h["content"].get("payload") or {},
                "created_at": h["created_at"],
            })
        return out

    def _query_humans(self, queue_type: str, applied: int) -> List[Dict[str, Any]]:
        with self._lock:
            rows = self._conn.execute(
                "SELECT * FROM humans WHERE queue_type=? AND applied=? "
                "ORDER BY id", (queue_type, applied)).fetchall()
        return [self._row_to_human(r) for r in rows]

    def drain_interventions(self) -> List[Dict[str, Any]]:
        """取走全部未应用干预并标 applied=1（at-least-once 摄取语义）。

        返回条目：{id, type, payload, created_at}（与旧 InterventionQueue
        drain 兼容形态——file 字段换为 id 字符串）。
        """
        out = []
        with self._lock:
            rows = self._conn.execute(
                "SELECT * FROM humans WHERE queue_type='intervention' "
                "AND applied=0 ORDER BY id").fetchall()
            now = _now()
            for r in rows:
                self._conn.execute(
                    "UPDATE humans SET applied=1, applied_at=? WHERE id=?",
                    (now, int(r["id"])))
            self._conn.commit()
        for r in rows:
            h = self._row_to_human(r)
            out.append({
                "id": h["id"],
                "type": h["content"].get("type"),
                "payload": h["content"].get("payload") or {},
                "created_at": h["created_at"],
                "file": "humans#%d" % h["id"],  # 兼容旧 drain 条目键
            })
        return out

    def push_proposal(self, content: dict) -> int:
        """提案入队（runtime 产出、待 human 裁决——step7 实验自改进用）。"""
        return self._push_human("proposal", content)

    def pending_proposals(self) -> List[Dict[str, Any]]:
        with self._lock:
            rows = self._conn.execute(
                "SELECT * FROM humans WHERE queue_type='proposal' "
                "AND applied=0 ORDER BY id").fetchall()
        return [self._row_to_human(r) for r in rows]

    def mark_proposal_applied(self, human_id: int) -> None:
        with self._lock:
            self._conn.execute(
                "UPDATE humans SET applied=1, applied_at=? "
                "WHERE id=? AND queue_type='proposal'", (_now(), int(human_id)))
            self._conn.commit()

    # ---- experiments ----

    def register_experiment(self, iteration: int,
                            cell_id: Optional[str] = None,
                            experiment_id: Optional[str] = None) -> str:
        """注册一次实验（INSERT），返回 experiment_id（= 工作目录名）。

        编号：`<iteration>-<YYYYMMDDHHMM>`；同 iter 同分钟并行注册撞 PK 时
        自动加序号后缀（-2/-3/...——并行实验安全）。
        """
        base = experiment_id or make_experiment_timestamp_id(iteration)
        with self._lock:
            eid = base
            suffix = 1
            while True:
                try:
                    self._conn.execute(
                        "INSERT INTO experiments (experiment_id, iteration, "
                        "workdir, cell_id, created_at) VALUES (?,?,?,?,?)",
                        (eid, int(iteration), eid, cell_id, _now()))
                    self._conn.commit()
                    return eid
                except sqlite3.IntegrityError:
                    suffix += 1
                    eid = "%s-%d" % (base, suffix)

    def update_experiment(self, experiment_id: str, **fields: Any) -> None:
        """回写实验行（record_path/三角色 session_id/score/status/cell_id/workdir）。

        三角色 session_id（job_56 第三轮审计）：builder/coding/debug 各自的
        pi session（行为轨迹还原）——experiments 表分三列存（旧 session_id
        单列废弃；传入 session_id 时兼容映射到 coding）。
        """
        if "session_id" in fields and fields.get("session_id") is not None:
            # 旧调用兼容：单 session_id → coding 列（若 coding 未显式给）
            fields.setdefault("coding_session_id", fields.get("session_id"))
        allowed = ("record_path", "builder_session_id", "coding_session_id",
                   "debug_session_id", "score", "status",
                   "cell_id", "workdir", "iteration")
        sets, vals = [], []
        for k in allowed:
            if k in fields and fields[k] is not None:
                sets.append("%s=?" % k)
                vals.append(fields[k])
        if not sets:
            return
        vals.append(experiment_id)
        with self._lock:
            self._conn.execute(
                "UPDATE experiments SET %s WHERE experiment_id=?"
                % ", ".join(sets), vals)
            self._conn.commit()

    def get_experiment(self, experiment_id: str) -> Optional[Dict[str, Any]]:
        with self._lock:
            row = self._conn.execute(
                "SELECT * FROM experiments WHERE experiment_id=?",
                (experiment_id,)).fetchone()
        return dict(row) if row else None

    def recent_experiments(self, n: int = 10) -> List[Dict[str, Any]]:
        with self._lock:
            rows = self._conn.execute(
                "SELECT * FROM experiments ORDER BY created_at DESC, "
                "experiment_id DESC LIMIT ?", (int(n),)).fetchall()
        return [dict(r) for r in rows]

    # ---- adjusts（adjust 运行表：每次流水线一行） ----

    def record_adjust(self, iteration: int, adjust_dir: str,
                      sessions: dict,
                      decisions_path: Optional[str] = None,
                      agent_valid: Optional[bool] = None) -> str:
        """记录一次 adjust 运行（INSERT OR REPLACE——同 iter 幂等）。

        sessions：各步 pi session_id（{"step1": "...", "step3": ["s1","s2","s3"],
        ...}——step3 三视角并行是数组）；adjust_dir=迭代目录
        （codes/adjust/iter_N——行为轨迹观察入口）。
        """
        adjust_id = os.path.basename(adjust_dir.rstrip("/")) or \
            "iter_%d" % int(iteration)
        with self._lock:
            self._conn.execute(
                "INSERT OR REPLACE INTO adjusts "
                "(adjust_id, iteration, adjust_dir, sessions, "
                "decisions_path, agent_valid, created_at) "
                "VALUES (?,?,?,?,?,?,?)",
                (adjust_id, int(iteration), adjust_dir,
                 json.dumps(sessions or {}, ensure_ascii=False),
                 decisions_path, None if agent_valid is None else
                 int(bool(agent_valid)), _now()))
            self._conn.commit()
        return adjust_id

    def interrupt_stale_running(self) -> List[str]:
        """P4（step7 提案4）：把 status=running 且 record_path 为空/不存在的
        死行（会话重启吞噬的在飞实验）标 interrupted——返回被清理的
        experiment_id 列表（启动恢复时调用，一次到位）。"""
        with self._lock:
            rows = self._conn.execute(
                "SELECT experiment_id, record_path FROM experiments "
                "WHERE status='running'").fetchall()
        dead = []
        for r in rows:
            rp = r["record_path"]
            if not rp or not os.path.exists(rp):
                dead.append(r["experiment_id"])
        if dead:
            with self._lock:
                self._conn.executemany(
                    "UPDATE experiments SET status='interrupted' "
                    "WHERE experiment_id=? AND status='running'",
                    [(d,) for d in dead])
                self._conn.commit()
        return dead

    def list_adjusts(self, n: int = 20) -> List[Dict[str, Any]]:
        """最近的 adjust 运行（观察入口：sessions JSON 反序列化）。"""
        with self._lock:
            rows = self._conn.execute(
                "SELECT * FROM adjusts ORDER BY iteration DESC, created_at "
                "DESC LIMIT ?", (int(n),)).fetchall()
        out = []
        for r in rows:
            d = dict(r)
            try:
                d["sessions"] = json.loads(d.get("sessions") or "{}")
            except (TypeError, ValueError):
                d["sessions"] = {}
            out.append(d)
        return out

    # ---- resources（槽位资源调度） ----

    def init_slots(self, n_experiment_slots: int, n_env_slots: int) -> None:
        """初始化资源位（幂等：只增不减——缩减需人工 SQL）。

        experiment_slot：并行实验总上限（exp-slot-0..N-1）
        env_slot：同一 env 允许并发实验数（env-slot-0..M-1，二级限额）
        """
        with self._lock:
            for i in range(int(n_experiment_slots)):
                self._conn.execute(
                    "INSERT OR IGNORE INTO resources (resource_type, "
                    "resource_id, description, quota) VALUES (?,?,?,1)",
                    (RESOURCE_EXPERIMENT_SLOT, "exp-slot-%d" % i,
                     "实验槽 %d（并行实验总上限的一个位）" % i))
            for i in range(int(n_env_slots)):
                self._conn.execute(
                    "INSERT OR IGNORE INTO resources (resource_type, "
                    "resource_id, description, quota) VALUES (?,?,?,1)",
                    (RESOURCE_ENV_SLOT, "env-slot-%d" % i,
                     "环境槽 %d（同一 env 允许并发实验数的一个位）" % i))
            self._conn.commit()

    def acquire_slot(self, resource_type: str,
                     holder_experiment_id: str) -> Optional[str]:
        """占一个空闲资源位（事务性：查空闲→占用→提交，防两线程同抢）。

        无空闲位 → None（调度器据此阻塞等待在飞实验释放）。
        """
        with self._lock:
            self._conn.execute("BEGIN IMMEDIATE")
            try:
                row = self._conn.execute(
                    "SELECT resource_id FROM resources WHERE resource_type=? "
                    "AND holder_experiment_id IS NULL "
                    "ORDER BY resource_id LIMIT 1", (resource_type,)).fetchone()
                if row is None:
                    self._conn.execute("COMMIT")
                    return None
                rid = row[0]
                self._conn.execute(
                    "UPDATE resources SET holder_experiment_id=?, acquired_at=? "
                    "WHERE resource_type=? AND resource_id=?",
                    (holder_experiment_id, _now(), resource_type, rid))
                self._conn.execute("COMMIT")
                return rid
            except Exception:
                try:
                    self._conn.execute("ROLLBACK")
                except sqlite3.Error:
                    pass
                raise

    def rebind_holder(self, resource_id: str,
                      holder_experiment_id: str) -> None:
        """占位换绑（env_9）：资源位持有者从 pending 占位换成真实实验 ID。

        使用方：调度器补槽段先以 ``pending-<cursor>`` 占位（槽满时不泄漏
        实验行/目录），注册真实实验 ID 后换绑。
        """
        with self._lock:
            self._conn.execute(
                "UPDATE resources SET holder_experiment_id=? WHERE resource_id=?",
                (holder_experiment_id, resource_id))
            self._conn.commit()

    def release_holder(self, holder_experiment_id: str) -> None:
        """释放该持有人占用的全部资源位（实验结束）。"""
        with self._lock:
            self._conn.execute(
                "UPDATE resources SET holder_experiment_id=NULL, acquired_at=NULL "
                "WHERE holder_experiment_id=?", (holder_experiment_id,))
            self._conn.commit()

    def slot_stats(self) -> Dict[str, Dict[str, int]]:
        """{resource_type: {total, free}}——观测与调度判定共用。"""
        with self._lock:
            rows = self._conn.execute(
                "SELECT resource_type, COUNT(*) AS total, "
                "SUM(CASE WHEN holder_experiment_id IS NULL THEN 1 ELSE 0 END) "
                "AS free FROM resources GROUP BY resource_type").fetchall()
        return {r[0]: {"total": int(r[1]), "free": int(r[2])} for r in rows}

    # ---- ledger（实验台账后端） ----

    def append_ledger(self, entry: Dict[str, Any]) -> None:
        with self._lock:
            self._conn.execute(
                "INSERT INTO ledger_entries (ts, experiment_id, payload) "
                "VALUES (?,?,?)",
                (str(entry.get("ts") or _now()),
                 entry.get("experiment_id"),
                 json.dumps(entry, ensure_ascii=False, default=str)))
            self._conn.commit()

    def read_ledger(self) -> List[Dict[str, Any]]:
        with self._lock:
            rows = self._conn.execute(
                "SELECT payload FROM ledger_entries ORDER BY id").fetchall()
        out = []
        for r in rows:
            try:
                v = json.loads(r[0])
                if isinstance(v, dict):
                    out.append(v)
            except (ValueError, TypeError):
                continue
        return out


# ---- 每工作区单例 ----

_instances = {}          # type: Dict[str, SuperDB]
_instances_lock = threading.Lock()


def db_path_for_ws(ws: str) -> str:
    return os.path.join(ws, DB_FILENAME)


def db_for_ws(ws: str) -> SuperDB:
    """取（或建）该工作区的 SuperDB 单例——模块间共享连接与锁。"""
    key = os.path.abspath(ws)
    with _instances_lock:
        inst = _instances.get(key)
        if inst is None:
            inst = SuperDB(key)
            _instances[key] = inst
        return inst


def _reset_instances_for_tests() -> None:
    """测试隔离：关闭并清空全部单例（每个用例的临时 ws 互不串连）。"""
    with _instances_lock:
        for inst in list(_instances.values()):
            inst.close()
        _instances.clear()
