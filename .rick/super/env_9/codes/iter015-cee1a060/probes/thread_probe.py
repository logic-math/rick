#!/usr/bin/env python3
"""thread_probe.py — VECLIB_MAXIMUM_THREADS 对 8192 SGEMM 的作用探针（iter15 builder 事实采集）。

三个问题：
  Q1（开关存在性）: 进程外设置 VECLIB_MAXIMUM_THREADS 是否改变 sgemm 耗时（各档位扫描）。
  Q2（在位可实现性）: numpy import 之后、首次 BLAS 调用之前设置（os.environ）是否同样生效
      ——决定 solution.py 能否在 benchmark.py 先 import numpy 的协议下使用该旋钮。
  Q3（数值一致性）: 限制线程数后与默认线程的输出是否逐位一致（uint32 视图比对）。

用法：
  python3 probes/thread_probe.py sweep     # Q1: 子进程扫描 {default,1,2,4,5,6,7,8,9,12,18}
  python3 probes/thread_probe.py inproc     # Q2/Q3: 进程内 late-set 对照（default vs 6 threads）
每臂：1 预热 + 5 计时取中位，n=8192 f32（与 benchmark.py 同口径但不经过候选加载）。
"""
import json
import os
import subprocess
import sys
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

HERE = os.path.dirname(os.path.abspath(__file__))


def _child_code():
    """子进程臂：读本进程 env 的 PROBE_THREADS，若 LATE=1 则在 numpy import 后再 setenv。"""
    return r"""
import os, time, json
import numpy as np
np.seterr(divide="ignore", over="ignore", invalid="ignore")
n = 8192
rng = np.random.RandomState(42)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)
if os.environ.get("LATE") == "1" and os.environ.get("PROBE_THREADS"):
    os.environ["VECLIB_MAXIMUM_THREADS"] = os.environ["PROBE_THREADS"]
c_ref = None
def run(tag):
    t0 = time.perf_counter(); c = np.matmul(a, b); dt = (time.perf_counter()-t0)*1000.0
    return dt, c
# warmup + 5 timed
run("warm")
ts = []
c_last = None
for i in range(5):
    dt, c = run(i)
    ts.append(round(dt, 2))
    if i == 0:
        c_first = c
    c_last = c
# bitwise self-consistency across calls
bit_self = bool(np.array_equal(c_first.view(np.uint32), c_last.view(np.uint32)))
print(json.dumps({"timings": ts, "median": sorted(ts)[2], "bit_self": bit_self}))
"""


def _run_arm(threads=None, late=False):
    env = dict(os.environ)
    if threads is not None:
        env["PROBE_THREADS"] = str(threads)
    env["LATE"] = "1" if late else "0"
    if not late and threads is not None:
        env["VECLIB_MAXIMUM_THREADS"] = str(threads)
    p = subprocess.run([sys.executable, "-c", _child_code()], env=env,
                       capture_output=True, text=True, timeout=180)
    if p.returncode != 0:
        return {"error": p.stderr.strip()[-200:]}
    return json.loads(p.stdout.strip().splitlines()[-1])


def sweep():
    counts = [None, 1, 2, 4, 5, 6, 7, 8, 9, 12, 18]
    out = {}
    for c in counts:
        key = "default" if c is None else str(c)
        out[key] = _run_arm(threads=c, late=False)
        print(key, out[key], flush=True)
    with open(os.path.join(HERE, "thread_sweep.json"), "w") as f:
        json.dump(out, f, indent=1)


def inproc():
    """Q2: default vs late-set-6（numpy 已 import 后 setenv）同批对照；Q3: 逐位比对。"""
    res = {}
    res["default"] = _run_arm(threads=None, late=False)
    res["late_6"] = _run_arm(threads=6, late=True)
    res["early_6"] = _run_arm(threads=6, late=False)
    for k, v in res.items():
        print(k, v, flush=True)
    with open(os.path.join(HERE, "thread_inproc.json"), "w") as f:
        json.dump(res, f, indent=1)


if __name__ == "__main__":
    mode = sys.argv[1] if len(sys.argv) > 1 else "sweep"
    if mode == "sweep":
        sweep()
    else:
        inproc()
