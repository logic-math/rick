"""test_cell_lock.py — 格级互斥锁（job_47 迭代9/task45）

覆盖：基本 acquire/release、跨实例（文件锁）、TTL 陈旧回收、未过期不抢占、
多线程竞争唯一赢家、release_all、孤儿锁（死 pid）清理与活 pid 保留、
safe_id 确定性、pid_alive 语义。
"""
import json
import os
import shutil
import sys
import tempfile
import threading
import time
import unittest

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))

from superopt.cell_lock import CellLocks, pid_alive, safe_id


class CellLockTest(unittest.TestCase):
    def setUp(self) -> None:
        self.ws = tempfile.mkdtemp(prefix="superopt-lock-test-")

    def tearDown(self) -> None:
        shutil.rmtree(self.ws, ignore_errors=True)

    # ---- 基本语义 ----

    def test_acquire_release_basic(self):
        locks = CellLocks(self.ws)
        self.assertFalse(locks.locked("a=x"))
        self.assertTrue(locks.acquire("a=x"))
        self.assertTrue(locks.locked("a=x"))
        # 一格一持有者：同进程重复 acquire 也不可重入
        self.assertFalse(locks.acquire("a=x"))
        self.assertTrue(locks.release("a=x"))
        self.assertFalse(locks.locked("a=x"))
        self.assertFalse(locks.release("a=x"))  # 幂等：未持有 → False

    def test_empty_cell_id_rejected(self):
        locks = CellLocks(self.ws)
        self.assertFalse(locks.acquire(""))
        self.assertFalse(locks.release(""))
        self.assertFalse(locks.locked(""))

    def test_file_lock_blocks_other_instance(self):
        """跨实例（不同 CellLocks 对象，= 另一进程语义）互斥。"""
        locks1 = CellLocks(self.ws)
        locks2 = CellLocks(self.ws)
        self.assertTrue(locks1.acquire("c1"))
        self.assertFalse(locks2.acquire("c1"))
        self.assertTrue(locks2.locked("c1"))
        locks1.release("c1")
        self.assertTrue(locks2.acquire("c1"))
        self.assertEqual(locks2.release_all(), 1)

    def test_distinct_cells_independent(self):
        locks = CellLocks(self.ws)
        self.assertTrue(locks.acquire("c1"))
        self.assertTrue(locks.acquire("c2"))
        self.assertTrue(locks.locked("c1"))
        self.assertTrue(locks.locked("c2"))
        locks.release_all()

    # ---- 陈旧 / 孤儿回收 ----

    def test_stale_lock_recovered_by_ttl(self):
        recovered = []
        locks = CellLocks(self.ws, ttl_s=0.01, on_stale=recovered.append)
        self.assertTrue(locks.acquire("c1"))
        path = locks.lock_path("c1")
        old = time.time() - 3600
        os.utime(path, (old, old))  # 模拟持有者崩溃已久
        other = CellLocks(self.ws, ttl_s=0.01, on_stale=recovered.append)
        self.assertTrue(other.acquire("c1"))
        self.assertIn("c1", recovered)
        other.release_all()

    def test_fresh_lock_not_stolen(self):
        locks = CellLocks(self.ws, ttl_s=3600)
        self.assertTrue(locks.acquire("c1"))
        other = CellLocks(self.ws, ttl_s=3600)
        self.assertFalse(other.acquire("c1"))
        locks.release_all()

    def test_ttl_disabled_never_recovers(self):
        locks = CellLocks(self.ws, ttl_s=0)
        self.assertTrue(locks.acquire("c1"))
        path = locks.lock_path("c1")
        old = time.time() - 3600
        os.utime(path, (old, old))
        other = CellLocks(self.ws, ttl_s=0)
        self.assertFalse(other.acquire("c1"))
        locks.release_all()

    def test_release_orphans_dead_pid(self):
        recovered = []
        locks = CellLocks(self.ws, on_stale=recovered.append)
        os.makedirs(locks.locks_dir, exist_ok=True)
        path = locks.lock_path("dead-cell")
        with open(path, "w", encoding="utf-8") as f:
            json.dump({"pid": 999999, "ts": "x", "cell_id": "dead-cell"}, f)
        out = locks.release_orphans()
        self.assertIn("dead-cell", out)
        self.assertIn("dead-cell", recovered)
        self.assertFalse(os.path.exists(path))

    def test_release_orphans_keeps_live_pid_lock(self):
        locks = CellLocks(self.ws)
        os.makedirs(locks.locks_dir, exist_ok=True)
        path = locks.lock_path("live")
        with open(path, "w", encoding="utf-8") as f:
            json.dump({"pid": os.getpid(), "ts": "x", "cell_id": "live"}, f)
        self.assertEqual(locks.release_orphans(), [])
        self.assertTrue(os.path.exists(path))

    def test_release_orphans_ignores_unparsable(self):
        locks = CellLocks(self.ws)
        os.makedirs(locks.locks_dir, exist_ok=True)
        path = locks.lock_path("broken")
        with open(path, "w", encoding="utf-8") as f:
            f.write("not-json")
        self.assertEqual(locks.release_orphans(), [])
        self.assertTrue(os.path.exists(path))  # 保守：无法解析不动

    # ---- 并发 ----

    def test_concurrent_threads_exactly_one_winner(self):
        locks = CellLocks(self.ws)
        results = []
        barrier = threading.Barrier(16)

        def worker():
            barrier.wait()
            results.append(locks.acquire("hot"))

        threads = [threading.Thread(target=worker) for _ in range(16)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()
        self.assertEqual(sum(1 for r in results if r), 1,
                         "同一格只允许一个赢家，实际 %r" % results)
        locks.release_all()

    def test_release_all_clears_memory_and_files(self):
        locks = CellLocks(self.ws)
        for cid in ("a", "b", "c"):
            self.assertTrue(locks.acquire(cid))
        self.assertEqual(locks.release_all(), 3)
        for cid in ("a", "b", "c"):
            self.assertFalse(locks.locked(cid))
            self.assertFalse(os.path.exists(locks.lock_path(cid)))

    # ---- 工具 ----

    def test_safe_id_deterministic_and_short(self):
        self.assertEqual(safe_id("a=x|b=y"), safe_id("a=x|b=y"))
        self.assertNotEqual(safe_id("a=x"), safe_id("a=y"))
        self.assertEqual(len(safe_id("whatever")), 16)

    def test_pid_alive(self):
        self.assertTrue(pid_alive(os.getpid()))
        self.assertFalse(pid_alive(999999))
        self.assertFalse(pid_alive(0))


if __name__ == "__main__":
    unittest.main()
