#!/usr/bin/env python3
"""paired_ab.py — 同进程交错配对 A/B：候选(iter025 threaded) vs 父本(iter014 顺序)。

方法论与 builder 预研一致（load~4-5，18 轮交错；每轮交替先手，消顺序偏置）。
"""
import importlib.util
import json
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

CAND = "/Users/logic/workdir/rick-dev/.rick/super/env_8/codes/iter025-47700efa/solution.py"
PARENT = "/Users/logic/workdir/rick-dev/.rick/super/env_8/codes/iter014-134ed9e4/solution.py"


def load(path, name):
    spec = importlib.util.spec_from_file_location(name, path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def main():
    n = 2048
    rounds = 18
    cand = load(CAND, "cand")
    par = load(PARENT, "parent")
    rng = np.random.RandomState(42)
    a = rng.rand(n, n).astype(np.float32)
    b = rng.rand(n, n).astype(np.float32)

    # 正确性三臂快查
    c_c = cand.matmul(a, b)
    c_p = par.matmul(a, b)
    c_r = np.matmul(a, b)
    errs = {
        "cand_vs_ref": float(np.max(np.abs(c_c.astype(np.float64) - c_r.astype(np.float64)))),
        "parent_vs_ref": float(np.max(np.abs(c_p.astype(np.float64) - c_r.astype(np.float64)))),
        "cand_vs_parent": float(np.max(np.abs(c_c.astype(np.float64) - c_p.astype(np.float64)))),
    }

    # 预热（含线程池创建、einsum optimize path 缓存）
    for _ in range(3):
        cand.matmul(a, b)
        par.matmul(a, b)

    # 每轮内部各测 1 次候选/父本（k=1 交错），先手逐轮交替
    pairs = []
    for i in range(rounds):
        if i % 2 == 0:
            t0 = time.perf_counter(); par.matmul(a, b); t_par = (time.perf_counter() - t0) * 1000.0
            t0 = time.perf_counter(); cand.matmul(a, b); t_cand = (time.perf_counter() - t0) * 1000.0
        else:
            t0 = time.perf_counter(); cand.matmul(a, b); t_cand = (time.perf_counter() - t0) * 1000.0
            t0 = time.perf_counter(); par.matmul(a, b); t_par = (time.perf_counter() - t0) * 1000.0
        pairs.append((t_cand, t_par))

    deltas = [c - p for c, p in pairs]  # 负 = 候选更快
    deltas_sorted = sorted(deltas)
    med = deltas_sorted[len(deltas) // 2]
    wins = sum(1 for d in deltas if d < 0)
    cand_med = sorted(c for c, _ in pairs)[rounds // 2]
    par_med = sorted(p for _, p in pairs)[rounds // 2]
    print(json.dumps({
        "n": n, "rounds": rounds,
        "cand_ms_median": round(cand_med, 3),
        "parent_ms_median": round(par_med, 3),
        "paired_delta_median_ms": round(med, 3),
        "win_rate": "%d/%d" % (wins, rounds),
        "deltas_ms": [round(d, 3) for d in deltas],
        "max_errors": errs,
        "load_note": "same-process interleaved, first-mover alternates per round",
    }))


if __name__ == "__main__":
    main()
