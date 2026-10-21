#!/usr/bin/env python3
"""ramp_ab.py — 预研：import 期预热是否能消除 bench 协议下的首计时调用 ramp。

模拟 bench.py 真实流程（独立进程）：
  load numpy → [可选 import 期预热：3 次 2048 整阵 dot（扔掉结果）] → warmup 1 次 → 计时 3 次 → 中位
对照：无 import 预热（mode=plain，即父本在门禁下的形态）。
用法：python3 ramp_ab.py plain|warm
"""
import json
import sys
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

MODE = sys.argv[1] if len(sys.argv) > 1 else "plain"
N = 2048

if MODE == "warm":
    # import 期预热：3 次全尺寸 dot（不同数组、结果丢弃）——模拟候选在 import 时自热
    rng0 = np.random.RandomState(7)
    wa = rng0.rand(N, N).astype(np.float32)
    wb = rng0.rand(N, N).astype(np.float32)
    for _ in range(3):
        np.dot(wa, wb)
    del wa, wb

rng = np.random.RandomState(42)
a = rng.rand(N, N).astype(np.float32)
b = rng.rand(N, N).astype(np.float32)

# bench 协议：warmup 1 + 计时 3（K=3 取中位）
np.dot(a, b)
ts = []
for _ in range(3):
    t0 = time.perf_counter()
    np.dot(a, b)
    ts.append((time.perf_counter() - t0) * 1000.0)

print(json.dumps({"mode": MODE, "timings": [round(x, 4) for x in ts],
                  "median": round(sorted(ts)[1], 4)}))
