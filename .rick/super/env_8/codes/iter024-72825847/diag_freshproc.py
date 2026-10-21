#!/usr/bin/env python3
"""iter024 诊断：fresh-process（bench 同形态）下 QoS 候选 vs 父本，双向交替排漂移。

每轮起 2 个全新进程（--k 5），顺序 p→c 与 c→p 交替，共 2×5 轮；每轮间记录
load avg。若候选在两种顺序下都系统性慢 → fresh-process 劣化是真的（非漂移）。
"""
import os
import subprocess
import sys
import time

VENV = "/Users/logic/workdir/rick-dev/.rick/super/env_8/env/.venv/bin/python"
BENCH = "/Users/logic/workdir/rick-dev/.rick/super/env_8/env/data/bench.py"
CAND = os.path.join(os.path.dirname(os.path.abspath(__file__)), "solution.py")
PAR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "iter017-245a85b7", "solution.py")


def run(path):
    r = subprocess.run([VENV, BENCH, "--candidate", path, "--k", "5"],
                       capture_output=True, text=True, timeout=300)
    import json
    d = json.loads(r.stdout.strip().splitlines()[-1])
    return d


def la():
    return float(open("/proc/loadavg").read().split()[0]) if os.path.exists("/proc/loadavg") else \
        float(subprocess.run(["uptime"], capture_output=True, text=True).stdout.split("load averages:")[1].split()[0])


if __name__ == "__main__":
    res = {"c": [], "p": []}
    rounds = [("p", "c"), ("c", "p")] * 5
    for i, (first, second) in enumerate(rounds):
        print("load avg %.2f" % la())
        for tag in (first, second):
            d = run(CAND if tag == "c" else PAR)
            res[tag].append(d["runtime_ms"])
            print("  %s: median=%.3f score=%.3f timings=%s" %
                  (tag, d["runtime_ms"], d["score"], [round(t, 2) for t in d["timings"]]))
        sys.stdout.flush()
    import statistics
    print("\n汇总（fresh-process, k=5 中位, %d 轮/侧）:" % len(res["c"]))
    print("  cand: %s → 总中位 %.3f" % ([round(x, 2) for x in res["c"]], statistics.median(res["c"])))
    print("  par : %s → 总中位 %.3f" % ([round(x, 2) for x in res["p"]], statistics.median(res["p"])))
    print("  配对差(par-cand) 中位 %+.3f" % statistics.median([p - c for p, c in zip(res["p"], res["c"])]))
