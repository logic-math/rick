#!/usr/bin/env python3
"""iter024 coding A/B 复核：父本(iter017, 默认QoS池) vs 候选(iter024, initializer=QoS)。

设计（复刻 probe5 顺序对消）：
- 同进程交替配对调用两个模块的 matmul（各自模块级线程池单例：父本默认 QoS、
  候选 USER_INTERACTIVE），配对差 = 父本 - 候选（正=候选更快）。
- 双顺序各 15 轮（父本先 / 候选先），消除顺序偏差；随后 6 个默认 QoS spinner
  合成负载下单顺序 15 轮复核。
- 预期：环境态（本机常态 load~4-5 背景负载）配对差中位 ≥0 或轻微改善、
  空载纯态 |中位|<0.15ms；合成负载态 ≥+0.5ms 中位改善。
"""
import importlib.util
import os
import subprocess
import sys
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

HERE = os.path.dirname(os.path.abspath(__file__))
PARENT = os.path.join(HERE, "..", "iter017-245a85b7", "solution.py")
CAND = os.path.join(HERE, "solution.py")
N = 2048


def load(path, name):
    spec = importlib.util.spec_from_file_location(name, path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def paired(tag, rounds, parent_first):
    diffs, ts_p, ts_c = [], [], []
    for _ in range(rounds):
        if parent_first:
            t0 = time.perf_counter(); P.matmul(a, b); tp = (time.perf_counter() - t0) * 1e3
            t0 = time.perf_counter(); C.matmul(a, b); tc = (time.perf_counter() - t0) * 1e3
        else:
            t0 = time.perf_counter(); C.matmul(a, b); tc = (time.perf_counter() - t0) * 1e3
            t0 = time.perf_counter(); P.matmul(a, b); tp = (time.perf_counter() - t0) * 1e3
        ts_p.append(tp); ts_c.append(tc); diffs.append(tp - tc)
    ds = sorted(diffs)
    med = ds[len(ds) // 2]
    improved = sum(1 for d in diffs if d > 0)
    print("%s: parent=%7.3fms cand=%7.3fms  配对差中位(parent-cand)=%+.3fms  候选更快轮数=%d/%d  max_parent=%.2f max_cand=%.2f"
          % (tag, sorted(ts_p)[rounds // 2], sorted(ts_c)[rounds // 2], med, improved, rounds,
             max(ts_p), max(ts_c)))
    return med, improved


if __name__ == "__main__":
    P = load(PARENT, "parent_sol")
    C = load(CAND, "cand_sol")

    rng = np.random.RandomState(42)
    a = rng.rand(N, N).astype(np.float32)
    b = rng.rand(N, N).astype(np.float32)
    ref = np.matmul(a, b)
    for name, mod in [("parent", P), ("cand", C)]:
        c = mod.matmul(a, b)
        err = float(np.max(np.abs(c.astype(np.float64) - ref.astype(np.float64))))
        assert err == 0.0, (name, err)
    # 两遍一致性
    assert np.array_equal(C.matmul(a, b), C.matmul(a, b))
    print("正确性：双模块 max_error=0.0、候选两遍逐位一致 — OK")
    P.matmul(a, b); C.matmul(a, b)  # 预热（池创建 + initializer 已落在计时窗口外）

    m1, i1 = paired("环境态 顺序A(父先) ", 15, True)
    m2, i2 = paired("环境态 顺序B(候先) ", 15, False)

    spinners = [subprocess.Popen([sys.executable, "-c", "while True: pass"]) for _ in range(6)]
    time.sleep(1.0)
    m3, i3 = paired("合成负载 顺序A     ", 15, True)
    for p in spinners:
        p.terminate()
    for p in spinners:
        p.wait()
    time.sleep(1.0)
    m4, i4 = paired("环境态2 顺序A      ", 15, True)

    print("\n汇总：环境态中位 %.3f / %.3f ms（双顺序平均 %+.3f）｜负载态 %+.3f（改善 %d/15）"
          % (m1, m2, (m1 + m2) / 2.0, m3, i3))
