#!/usr/bin/env python3
"""final_confirm.py — iter15 builder 最终确认探针（一次跑完，读数即记录）。

P1/P2: 跨线程数逐位一致性 —— default vs VECLIB=6（n=64 与 n=8192 各一对子进程，
       sha256(uint32 view) 对比）。科学侧证据（gate 侧同池比对天然同线程数，见 verify/bench 顺序分析）。
P3/P4: benchmark.py 协议全真模拟（fresh 进程、模块加载在 rng.rand 之后、warmup1+K3 中位、
       随后 ref 同协议 K3）—— P3=default 对照臂，P4=late-set 6（与候选 solution.py 的
       实装路径一致：os.environ 在 numpy import 后、首次 BLAS 调用前设置）。
       输出 runtime/score/timings，与 gate 口径同尺度（ref band [380,700]、score≈1 预期）。
"""
import hashlib
import json
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
PY = sys.executable

CHILD = r"""
import hashlib, json, os, time
MODE = os.environ["MODE"]          # "bit" 或 "proto"
N = int(os.environ["N"])
if os.environ.get("SETV") == "1":
    os.environ["VECLIB_MAXIMUM_THREADS"] = "6"   # 模拟候选模块在 numpy import 后设置
import numpy as np
np.seterr(divide="ignore", over="ignore", invalid="ignore")
rng = np.random.RandomState(42)
a = rng.rand(N, N).astype(np.float32)
b = rng.rand(N, N).astype(np.float32)
if MODE == "bit":
    c = np.matmul(a, b)
    h = hashlib.sha256(np.ascontiguousarray(c).view(np.uint32)).hexdigest()
    print(json.dumps({"sha": h}))
else:
    # benchmark.py 全真模拟：rng.rand 已在 setenv 之后（协议：a/b 生成先于候选加载，
    # 但 rand 不走 BLAS——等价）；候选 warmup1+K3，随后 ref warmup1+K3。
    import importlib.util
    # 直接以内联函数模拟候选（= np.matmul 包装 + 已设的 env）
    def matmul(a, b):
        return np.matmul(np.asarray(a, dtype=np.float32), np.asarray(b, dtype=np.float32))
    matmul(a, b)  # warmup（首次 BLAS 调用——线程池在此定型）
    ts = []
    for _ in range(3):
        t0 = time.perf_counter(); matmul(a, b); ts.append(round((time.perf_counter()-t0)*1e3, 2))
    cand_med = sorted(ts)[1]
    np.matmul(a, b)  # ref warmup
    rts = []
    for _ in range(3):
        t0 = time.perf_counter(); np.matmul(a, b); rts.append(round((time.perf_counter()-t0)*1e3, 2))
    ref_med = sorted(rts)[1]
    print(json.dumps({"cand_timings": ts, "cand_median": cand_med,
                      "ref_timings": rts, "ref_median": ref_med,
                      "score": round(ref_med / cand_med, 4)}))
"""


def run(mode, n, setv):
    env = dict(os.environ)
    env["MODE"], env["N"], env["SETV"] = mode, str(n), ("1" if setv else "0")
    env.pop("VECLIB_MAXIMUM_THREADS", None)
    p = subprocess.run([PY, "-c", CHILD], env=env, capture_output=True, text=True, timeout=600)
    if p.returncode != 0:
        return {"error": p.stderr.strip()[-300:]}
    return json.loads(p.stdout.strip().splitlines()[-1])


out = {}
for n in (64, 8192):
    h_def = run("bit", n, False)
    h_v6 = run("bit", n, True)
    out["bit_n%d" % n] = {"default_sha": h_def.get("sha"), "v6_sha": h_v6.get("sha"),
                          "bitwise_equal": h_def.get("sha") == h_v6.get("sha")}
    print("bit n=%d:" % n, out["bit_n%d" % n], flush=True)

for tag, setv in (("proto_default", False), ("proto_v6_late", True)):
    r = run("proto", 8192, setv)
    out[tag] = r
    print(tag, r, flush=True)

with open(os.path.join(HERE, "final_confirm.json"), "w") as f:
    json.dump(out, f, indent=1)
print("\nFINAL", json.dumps(out, indent=1))
