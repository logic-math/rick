#!/usr/bin/env python3
"""iter024 诊断 2：fresh-process QoS 剂量矩阵（bench.py 同形态），环境态 vs 合成负载。

变体（各为独立完整解文件，bench.py 起全新进程计时 k=5）：
- parent：默认 QoS worker（iter017 父本）
- ui00  ：USER_INTERACTIVE(0x21, rel 0)——本迭代候选
- uinit ：USER_INITIATED(0x19, rel 0)
- onearg：探针同款单参调用（argtypes=[c_int]）

每轮按 [parent, ui00, uinit, onearg] 顺序各跑一次（k=5 中位），环境态 5 轮；
随后 6 个 default-QoS spinner 下 4 轮。配对差相对 parent。
"""
import json
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
VENV = "/Users/logic/workdir/rick-dev/.rick/super/env_8/env/.venv/bin/python"
BENCH = "/Users/logic/workdir/rick-dev/.rick/super/env_8/env/data/bench.py"

VARIANTS = [
    ("parent", os.path.join(HERE, "probe_v_parent.py")),
    ("ui00", os.path.join(HERE, "solution.py")),
    ("uinit", os.path.join(HERE, "probe_v_uinit.py")),
    ("onearg", os.path.join(HERE, "probe_v_onearg.py")),
]


def bench(path):
    r = subprocess.run([VENV, BENCH, "--candidate", path, "--k", "5"],
                       capture_output=True, text=True, timeout=300)
    return json.loads(r.stdout.strip().splitlines()[-1])


def main():
    rounds_env, rounds_load = 5, 4
    if len(sys.argv) > 1:
        rounds_env = int(sys.argv[1])
    res = {name: [] for name, _ in VARIANTS}

    def sweep(tag):
        for name, path in VARIANTS:
            d = bench(path)
            res[name].append(d["runtime_ms"])
            print("  %-6s median=%.3f score=%.3f timings=%s" %
                  (name, d["runtime_ms"], d["score"], [round(t, 2) for t in d["timings"]]))
        sys.stdout.flush()

    print("--- 环境态（ambient load）---")
    for i in range(rounds_env):
        print("round %d:" % (i + 1))
        sweep("env")

    spinners = [subprocess.Popen([sys.executable, "-c", "while True: pass"]) for _ in range(6)]
    import time
    time.sleep(1.5)
    print("--- 合成负载（6× default-QoS spinner）---")
    for i in range(rounds_load):
        print("round %d:" % (i + 1))
        sweep("load")
    for p in spinners:
        p.terminate()
    for p in spinners:
        p.wait()

    print("\n=== 汇总（各变体全部轮次中位，括号=相对 parent 配对差中位）===")
    import statistics
    base = res["parent"]
    for name, _ in VARIANTS:
        diffs = [b - v for b, v in zip(base, res[name])]
        print("%-8s n=%d 总中位=%.3f  环境态中位=%.3f  负载态中位=%.3f  配对差中位=%+.3f" %
              (name, len(res[name]), statistics.median(res[name]),
               statistics.median(res[name][:rounds_env]),
               statistics.median(res[name][rounds_env:]) if len(res[name]) > rounds_env else float("nan"),
               statistics.median(diffs)))


if __name__ == "__main__":
    main()
