#!/usr/bin/env python3
"""thread_ab.py — 线程旋钮交错 A/B（iter15 builder 事实采集，方法论复刻 iter14 debug_recheck_ab）。

问题：VECLIB_MAXIMUM_THREADS 相对 default 在 8192 SGEMM 上是否有可复现的 floor 收益？
方法：3 轮 × 3 臂交错（每轮换序），每臂 fresh 进程 1 预热 + 7 计时，报 floor(min)/median。
臂：default（不设）/ v6（进程外设 6）/ v12（进程外设 12）/ late6（numpy import 后 setenv 6 = Q2 可实现性）。

判定线：某臂相对 default 的 pooled floor 差 ≥ 10ms（~2.3%）且三轮方向一致 → 旋钮有真实收益。
"""
import json
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
PY = sys.executable

CHILD = r"""
import os, time, json
import numpy as np
np.seterr(divide="ignore", over="ignore", invalid="ignore")
n = 8192
rng = np.random.RandomState(42)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)
if os.environ.get("LATE") == "1":
    os.environ["VECLIB_MAXIMUM_THREADS"] = os.environ["PROBE_THREADS"]
np.matmul(a, b)
ts = []
for i in range(7):
    t0 = time.perf_counter()
    np.matmul(a, b)
    ts.append(round((time.perf_counter() - t0) * 1000.0, 2))
print(json.dumps({"ts": ts, "floor": min(ts), "median": sorted(ts)[3]}))
"""

ARMS = {
    "default": {"PROBE_THREADS": "", "LATE": "0"},   # 不设旋钮
    "v6": {"PROBE_THREADS": "6", "LATE": "0", "VECLIB_MAXIMUM_THREADS": "6"},
    "v12": {"PROBE_THREADS": "12", "LATE": "0", "VECLIB_MAXIMUM_THREADS": "12"},
    "late6": {"PROBE_THREADS": "6", "LATE": "1"},    # numpy import 后 setenv（Q2）
}
ORDERS = [["default", "v6", "v12", "late6"],
          ["v6", "v12", "late6", "default"],
          ["v12", "late6", "default", "v6"]]

results = {k: [] for k in ARMS}
for r, order in enumerate(ORDERS):
    for arm in order:
        env = dict(os.environ)
        for k, v in ARMS[arm].items():
            if v:
                env[k] = v
            else:
                env.pop(k, None)
        p = subprocess.run([PY, "-c", CHILD], env=env, capture_output=True, text=True, timeout=300)
        d = json.loads(p.stdout.strip().splitlines()[-1])
        d["round"] = r
        results[arm].append(d)
        print(f"r{r} {arm:8s} floor={d['floor']:7.2f} median={d['median']:7.2f} ts={d['ts']}", flush=True)

summary = {}
for arm, runs in results.items():
    floors = [r["floor"] for r in runs]
    meds = [r["median"] for r in runs]
    summary[arm] = {"floors": floors, "pooled_floor": min(floors),
                    "pooled_median": sorted(meds)[len(meds) // 2]}
print("\nSUMMARY", json.dumps(summary, indent=1))
with open(os.path.join(HERE, "thread_ab.json"), "w") as f:
    json.dump({"runs": results, "summary": summary}, f, indent=1)
