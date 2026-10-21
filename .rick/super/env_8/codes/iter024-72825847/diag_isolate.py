#!/usr/bin/env python3
"""iter024 诊断 3：环境态 fresh-process 下拆解 initializer 两个成分的独立贡献。"""
import json, os, subprocess, sys
HERE = os.path.dirname(os.path.abspath(__file__))
VENV = "/Users/logic/workdir/rick-dev/.rick/super/env_8/env/.venv/bin/python"
BENCH = "/Users/logic/workdir/rick-dev/.rick/super/env_8/env/data/bench.py"
VARIANTS = [
    ("parent", os.path.join(HERE, "probe_v_parent.py")),
    ("emptyinit", os.path.join(HERE, "probe_v_emptyinit.py")),
    ("seterr", os.path.join(HERE, "probe_v_seterr.py")),
    ("qosonly", os.path.join(HERE, "probe_v_qosonly.py")),
    ("ui00", os.path.join(HERE, "solution.py")),
]
def bench(path):
    r = subprocess.run([VENV, BENCH, "--candidate", path, "--k", "5"],
                       capture_output=True, text=True, timeout=300)
    return json.loads(r.stdout.strip().splitlines()[-1])
res = {n: [] for n, _ in VARIANTS}
for i in range(4):
    print("round %d:" % (i + 1))
    for name, path in VARIANTS:
        d = bench(path)
        res[name].append(d["runtime_ms"])
        print("  %-9s median=%.3f score=%.3f timings=%s" % (name, d["runtime_ms"], d["score"], [round(t,2) for t in d["timings"]]))
    sys.stdout.flush()
import statistics
print("\n=== 汇总（环境态 fresh-process, k=5 中位 ×4 轮）===")
base = res["parent"]
for name, _ in VARIANTS:
    diffs = [b - v for b, v in zip(base, res[name])]
    print("%-9s 总中位=%.3f  轮中位=%s  vs parent 配对差中位=%+.3f" %
          (name, statistics.median(res[name]), [round(x,2) for x in res[name]], statistics.median(diffs)))
