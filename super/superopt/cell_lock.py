"""cell_lock.py — 特征格级互斥锁（一个格同时只有一个实验）

设计（job_47 迭代9/task45，human fix 2）：

- **实验 = 一次特征格上的迭代**。builder/coding/debug 三 agent 在同一格上
  顺序协作；同一格**同时**只允许一个实验；不同格可以并行。
- **双保险**：
  1. 进程内内存集合（``threading.Lock`` 保护）——同一调度器进程内的槽/线程互斥；
  2. 文件锁 ``<ws>/locks/cell_<safe_id>.lock``（``O_CREAT|O_EXCL`` 原子创建，
     内容 ``{pid, ts, cell_id}``）——跨进程互斥（守护进程重启窗口、人工并发
     启两个调度器、槽池未来演进到多进程时仍成立）。
- **陈旧锁回收**：锁文件 mtime 超过 ``ttl_s``（默认 3600s）视为死锁持有者
  （崩溃/被 SIGKILL），清理并重新获取——回收经 ``on_stale`` 回调上报，
  由调用方落 ``cell_lock_stale_recovered`` 事件（可观测）。
- **孤儿锁清理**：``release_orphans()`` 扫描锁目录，把「持有者 pid 已不存在」
  的锁文件清理掉——调度器启动时调用，避免自身上次崩溃留下永久占锁
  （cell_lock_ttl_s 之前不必空等）。
- **只做互斥，不做阻塞等待**：``acquire`` 立即返回 bool；重采样/等待策略
  由调用方（loop）决定（避免锁模块引入超时/睡眠语义）。

并发约定（loop 侧）：worker 线程**不得**直接写 events.jsonl/archive.json
（单写者纪律）；本模块只保护「哪个格归哪个实验」，不涉及事件写入。

stdlib-only，Python >= 3.9。
"""
import errno
import hashlib
import json
import os
import threading
import time
from typing import Any, Callable, Dict, List, Optional, Set

__all__ = ["CellLocks", "safe_id", "pid_alive"]


def safe_id(cell_id: str, length: int = 16) -> str:
    """把任意 cell_id 压成文件系统安全的短 id（sha1 前缀，确定性）。"""
    digest = hashlib.sha1((cell_id or "").encode("utf-8")).hexdigest()
    return digest[:length]


def pid_alive(pid: int) -> bool:
    """pid 是否仍存在（POSIX kill(pid, 0) 语义）。

    - ``ProcessLookupError``（ESRCH）→ False（进程不存在）
    - ``PermissionError``（EPERM）→ True（存在但无权限，如 root 进程）
    - 其它异常 → 保守 True（宁可晚回收，不误删活锁）
    """
    if not isinstance(pid, int) or pid <= 0:
        return False
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return False
    except PermissionError:
        return True
    except OSError:
        return True
    return True


class CellLocks:
    """格级互斥锁（内存 + 文件双保险；线程安全）。

    用法::

        locks = CellLocks(ws, ttl_s=3600, on_stale=cb)
        if locks.acquire(cell_id):
            try:
                ...  # 该格归本实验独占
            finally:
                locks.release(cell_id)

    构造参数：
    - ``ws``：super 工作区根（锁文件落在 ``<ws>/locks/``）
    - ``ttl_s``：陈旧锁回收阈值（秒）；<=0 表示禁用 TTL 回收
    - ``on_stale``：``Callable[[str], None]``，TTL 回收/孤儿清理时逐格回调
      （调用方决定是否落事件——本模块不写事件，保持零依赖）
    - ``pid``：锁文件里记录的持有者 pid（默认当前进程；测试可注入）
    """

    def __init__(self, ws: str, ttl_s: float = 3600.0,
                 on_stale: Optional[Callable[[str], None]] = None,
                 pid: Optional[int] = None) -> None:
        self.ws = os.path.abspath(ws)
        self.locks_dir = os.path.join(self.ws, "locks")
        try:
            self.ttl_s = float(ttl_s or 0.0)
        except (TypeError, ValueError):
            self.ttl_s = 0.0
        if self.ttl_s < 0:
            self.ttl_s = 0.0
        self.on_stale = on_stale
        self.pid = int(pid if pid is not None else os.getpid())
        self._mem = set()  # type: Set[str]
        self._lock = threading.Lock()

    # ---- 路径与文件辅助 ----

    def lock_path(self, cell_id: str) -> str:
        return os.path.join(self.locks_dir, "cell_%s.lock" % safe_id(cell_id))

    def _ensure_dir(self) -> bool:
        try:
            os.makedirs(self.locks_dir, exist_ok=True)
            return True
        except OSError:
            return False

    def _write_lock(self, cell_id: str) -> bool:
        """原子创建锁文件（O_CREAT|O_EXCL）。成功 True / 已存在或失败 False。"""
        path = self.lock_path(cell_id)
        try:
            fd = os.open(path, os.O_CREAT | os.O_EXCL | os.O_WRONLY, 0o644)
        except FileExistsError:
            return False
        except OSError as exc:
            # 目录缺失（首次使用/被清理）→ 建目录后重试一次
            if exc.errno == errno.ENOENT and self._ensure_dir():
                try:
                    fd = os.open(path, os.O_CREAT | os.O_EXCL | os.O_WRONLY,
                                 0o644)
                except OSError:
                    return False
            else:
                return False
        try:
            payload = json.dumps({"pid": self.pid, "ts": _now_ts(),
                                  "cell_id": cell_id},
                                 ensure_ascii=False).encode("utf-8")
            os.write(fd, payload)
        finally:
            try:
                os.close(fd)
            except OSError:
                pass
        return True

    def _read_info(self, path: str) -> Dict[str, Any]:
        try:
            with open(path, "r", encoding="utf-8") as f:
                data = json.loads(f.read() or "{}")
            return data if isinstance(data, dict) else {}
        except (OSError, ValueError):
            return {}

    def _file_age_s(self, path: str) -> Optional[float]:
        try:
            return max(0.0, time.time() - os.stat(path).st_mtime)
        except OSError:
            return None

    def _is_stale(self, path: str) -> bool:
        if self.ttl_s <= 0:
            return False
        age = self._file_age_s(path)
        if age is None:
            return False
        return age > self.ttl_s

    def _notify_stale(self, cell_id: str) -> None:
        cb = self.on_stale
        if callable(cb):
            try:
                cb(cell_id)
            except Exception:  # noqa: BLE001 —— 回调失败不破坏锁语义
                pass

    # ---- 对外 API ----

    def acquire(self, cell_id: str) -> bool:
        """尝试独占该格。成功 True；已被占用（本进程/其它进程/未过期）False。

        同一进程内对同一格重复 acquire 也返回 False——格级互斥不做可重入
        （一个实验一个持有者）。陈旧锁（mtime 超 TTL）会被回收并重新获取，
        回收经 ``on_stale`` 回调上报。
        """
        if not cell_id:
            return False
        with self._lock:
            if cell_id in self._mem:
                return False
            if not self._ensure_dir():
                return False
            path = self.lock_path(cell_id)
            if self._write_lock(cell_id):
                self._mem.add(cell_id)
                return True
            # 文件已存在：判断陈旧（持有者崩溃留下的锁）
            if self._is_stale(path):
                try:
                    os.unlink(path)
                except OSError:
                    pass
                if self._write_lock(cell_id):
                    self._mem.add(cell_id)
                    self._notify_stale(cell_id)
                    return True
                return False
            return False

    def release(self, cell_id: str) -> bool:
        """释放该格（幂等）。返回是否实际释放了本进程持有的锁。"""
        if not cell_id:
            return False
        with self._lock:
            held = cell_id in self._mem
            self._mem.discard(cell_id)
            try:
                os.unlink(self.lock_path(cell_id))
            except OSError:
                pass
            return held

    def locked(self, cell_id: str) -> bool:
        """该格当前是否被锁（本进程内存 或 未过期文件锁）。"""
        if not cell_id:
            return False
        with self._lock:
            if cell_id in self._mem:
                return True
        path = self.lock_path(cell_id)
        if not os.path.exists(path):
            return False
        return not self._is_stale(path)

    def release_all(self) -> int:
        """释放本进程持有的全部格锁（优雅停/异常收尾用）。返回释放数。"""
        with self._lock:
            held = list(self._mem)
            self._mem.clear()
        for cell_id in held:
            try:
                os.unlink(self.lock_path(cell_id))
            except OSError:
                pass
        return len(held)

    def release_orphans(self) -> List[str]:
        """清理「持有者 pid 已不存在」的锁文件（启动时调用）。

        返回被清理的 cell_id 列表（每项同时经 ``on_stale`` 回调上报）。
        只清理**他人已死**的锁；本进程持有的锁、活进程持有的锁、无法解析
        内容的锁一律不动（保守）。
        """
        released = []  # type: List[str]
        try:
            names = os.listdir(self.locks_dir)
        except OSError:
            return released
        for name in names:
            if not name.endswith(".lock"):
                continue
            path = os.path.join(self.locks_dir, name)
            info = self._read_info(path)
            pid = info.get("pid")
            cell_id = info.get("cell_id") or ""
            if not isinstance(pid, int) or pid <= 0:
                continue
            if pid == os.getpid():
                continue  # 自己的锁（正常持有中）
            if pid_alive(pid):
                continue  # 活进程持有
            try:
                os.unlink(path)
            except OSError:
                continue
            released.append(cell_id)
            self._notify_stale(cell_id)
        return released


def _now_ts() -> str:
    """RFC3339 本地时间（带时区——domains bugs.md 坑位）。"""
    return time.strftime("%Y-%m-%dT%H:%M:%S%z")
