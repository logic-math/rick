#!/usr/bin/env python3
"""profile.py — agent 侧热点剖析脚本（快速反馈，非终验）。

coding/debug agent 在定位「慢在哪」时使用：对候选实现做 cProfile 剖析 +
关键路径单次计时，输出 top-N 热点函数与占比，帮助定位 Python 侧调度开销
（@2048 的性能故事 = 减少子调用次数，本脚本直接把「次数×单次开销」显形）。

与 self_check.py 的分工：
- self_check.py = 正确性 + 性能「是否合格」（门禁口径预演）
- profile.py    = 性能「损耗在哪」（归因与调优导航）

输出协议：stdout 单行 JSON
    {"ok": true, "n": 2048, "candidate_ms": 141.9, "ref_ms": 6.9,
     "overhead_vs_ref": 20.6, "subcalls": 2049,
     "top": [{"func": "matmul", "ncalls": 2049, "cum_ms": 138.2, "pct": 97.4}, ...],
     "note": "..."}

用法：
    python3 skills/profile.py --candidate <path/to/candidate.py> [--n 2048] [--top 8]
"""
import argparse
import cProfile
import io
import json
import os
import pstats
import sys
import time

import numpy as np

# macOS vecLib BLAS 虚假 FP 标志噪声抑制（数值正常）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")

HERE = os.path.dirname(os.path.abspath(__file__))
ENV_JSON = os.path.join(HERE, "..", "env.json")

DEFAULT_N = 2048
DEFAULT_TOP = 8


def _load_n():
    try:
        with open(ENV_JSON, encoding="utf-8") as f:
            return int(json.load(f).get("bench", {}).get("size", DEFAULT_N))
    except Exception:
        return DEFAULT_N


def _load_candidate(path):
    import importlib.util
    spec = importlib.util.spec_from_file_location("candidate", path)
    if spec is None or spec.loader is None:
        raise ImportError("cannot load module from %s" % path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def main():
    parser = argparse.ArgumentParser(description="候选热点剖析（快速反馈）")
    parser.add_argument("--candidate", required=True, help="候选 .py 路径")
    parser.add_argument("--n", type=int, default=None, help="矩阵规模（默认读 env.json bench.size）")
    parser.add_argument("--top", type=int, default=DEFAULT_TOP, help="热点函数条数")
    args = parser.parse_args()

    n = args.n or _load_n()
    out = {"ok": False, "n": n, "candidate_ms": None, "ref_ms": None,
           "overhead_vs_ref": None, "subcalls": None, "top": [], "note": ""}

    try:
        mod = _load_candidate(args.candidate)
    except Exception as e:
        out["note"] = "load candidate failed: %s" % e
        print(json.dumps(out, ensure_ascii=False))
        return 1
    if not callable(getattr(mod, "matmul", None)):
        out["note"] = "candidate has no callable matmul(a, b)"
        print(json.dumps(out, ensure_ascii=False))
        return 1

    rng = np.random.RandomState(42)
    a = rng.rand(n, n).astype(np.float32)
    b = rng.rand(n, n).astype(np.float32)

    # ---- 参考计时（同 bench.py 口径：预热 1 + 单次计时）----
    np.matmul(a, b)
    t0 = time.perf_counter()
    np.matmul(a, b)
    out["ref_ms"] = round((time.perf_counter() - t0) * 1000.0, 3)

    # ---- 候选计时（预热 1 + 单次计时）----
    try:
        mod.matmul(a, b)
        t0 = time.perf_counter()
        mod.matmul(a, b)
        out["candidate_ms"] = round((time.perf_counter() - t0) * 1000.0, 3)
    except Exception as e:
        out["note"] = "candidate run failed: %s" % e
        print(json.dumps(out, ensure_ascii=False))
        return 1
    if out["ref_ms"] and out["ref_ms"] > 0:
        out["overhead_vs_ref"] = round(out["candidate_ms"] / out["ref_ms"], 2)

    # ---- cProfile 剖析（单独一遍，避免计时污染）----
    prof = cProfile.Profile()
    prof.enable()
    mod.matmul(a, b)
    prof.disable()
    s = io.StringIO()
    st = pstats.Stats(prof, stream=s).sort_stats("cumulative")
    st.print_stats(20)

    total_tt = 0.0
    try:
        total_tt = float(st.total_tt)  # py>=3.9: Stats.total_tt
    except Exception:
        pass

    top = []
    for (filename, lineno, funcname), (cc, nc, tt, ct, callers) in sorted(
            st.stats.items(), key=lambda kv: kv[1][3], reverse=True):
        if len(top) >= args.top:
            break
        where = os.path.basename(str(filename))
        top.append({
            "func": "%s:%s:%s" % (where, lineno, funcname),
            "ncalls": nc,
            "cum_ms": round(ct * 1000.0, 3),
            "pct_of_profile": round((ct / total_tt * 100.0) if total_tt else 0.0, 1),
        })
    out["top"] = top
    out["subcalls"] = sum(int(v[1]) for v in st.stats.values())

    out["ok"] = True
    out["note"] = "cum_ms 为该函数及子调用累计耗时；pct 为占剖析总时间比例"
    print(json.dumps(out, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main())
