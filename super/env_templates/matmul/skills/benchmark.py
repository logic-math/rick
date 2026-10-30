#!/usr/bin/env python3
"""bench.py — 性能门（stage3，框架终验的一环；K 次计时取中位数）。

env_7 加固版（设计树 §M2 叶子，human 裁决 Q9——12 项加固清单之①③④⑧⑨）：
- **计时引用快照**（①）：顶部 `from time import perf_counter as _perf`——候选后到的
  `time.perf_counter = fake` 不影响已绑定引用（进程内 monkeypatch 失效）。
- **score 上界拒收**（③）：score > score_max（默认 2.0）→ passed=false（KernelBench 偏执原则：
  本机实测无 >1.0 可达路径，>2.0 必为作弊或测量异常）。
- **计时下界拒收**（④）：runtime_ms < 0.01（512×512 物理不可能）→ 拒（AlphaEvolve「微秒级=硬编码」）。
- **ref 合理带**（⑧）：ref_runtime_ms ∉ [0.10, 0.50]（本机稳态 0.185±3% 的宽容带）→ 拒——
  堵「拖慢 ref 抬 score」通道（候选留后台忙循环抢 CPU）。换机器需重校此带。
- **残留子进程清点**（⑨）：结束前 waitpid(WNOHANG) 循环——有存活子进程（如 fork 的忙循环）
  → 拒（堵通道③的持久化变体）。

输出协议（gates 框架消费契约）：**stdout 单行 JSON**
    {"runtime_ms": 123.4, "score": 0.87, "timings": [..K..], "ref_runtime_ms": 107.2, "n": 512, "k": 3}
    {"passed": false, "error": "score exceeds plausibility bound: 15.3 > 2.0"}
退出码：0=测量完成 / 1=超时或异常或拒收。

用法（gates 以显式解释器调用——优先 env/.venv/bin/python，fallback python3）：
    python3 data/bench.py --candidate <path/to/candidate.py>
"""
import argparse
import importlib.util
import json
import os
import signal
import sys
from time import perf_counter as _perf  # 加固①：引用快照——后到的 monkeypatch 不影响本计时

import numpy as np

# macOS vecLib 对 float32 matmul 报虚假 FP 标志（数值正常；见 test_correctness.py 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")

HERE = os.path.dirname(os.path.abspath(__file__))
ENV_JSON = os.path.join(HERE, "..", "env.json")
_ENV_DIR_OVERRIDE = None  # --env-dir 覆盖（三要素统一契约）

DEFAULT_SIZE = 512
DEFAULT_K = 3
DEFAULT_TIMEOUT_S = 60
DEFAULT_SCORE_MAX = 2.0
DEFAULT_RUNTIME_MS_MIN = 0.01
DEFAULT_REF_BAND = [0.10, 0.50]


class _CandidateTimeout(Exception):
    pass


def _alarm_handler(signum, frame):
    raise _CandidateTimeout()


def _load_env_config():
    cfg = {}
    try:
        with open(ENV_JSON, encoding="utf-8") as f:
            cfg = json.load(f)
    except Exception:
        cfg = {}
    bench_cfg = cfg.get("bench", {})
    gates_cfg = cfg.get("gates", {})
    size = bench_cfg.get("size", DEFAULT_SIZE)
    k = bench_cfg.get("k", DEFAULT_K)
    timeout_s = bench_cfg.get("timeout_s", DEFAULT_TIMEOUT_S)
    score_max = gates_cfg.get("score_max", DEFAULT_SCORE_MAX)
    rt_min = gates_cfg.get("runtime_ms_min", DEFAULT_RUNTIME_MS_MIN)
    ref_band = gates_cfg.get("ref_runtime_band_ms", DEFAULT_REF_BAND)
    return int(size), int(k), float(timeout_s), float(score_max), float(rt_min), list(ref_band)


def _load_candidate(path):
    spec = importlib.util.spec_from_file_location("candidate", path)
    if spec is None or spec.loader is None:
        raise ImportError("cannot load module from %s" % path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def _median(values):
    vs = sorted(values)
    m = len(vs)
    if m == 0:
        return 0.0
    if m % 2 == 1:
        return vs[m // 2]
    return (vs[m // 2 - 1] + vs[m // 2]) / 2.0


def _time_once(fn, a, b, timeout_s):
    """单次执行 + 超时防护。返回 (ok, elapsed_ms, error)。全程用 _perf 引用快照（加固①）。"""
    signal.signal(signal.SIGALRM, _alarm_handler)
    signal.alarm(timeout_s)
    try:
        t0 = _perf()
        fn(a, b)
        return True, (_perf() - t0) * 1000.0, None
    except _CandidateTimeout:
        return False, None, "timeout"
    except Exception as e:
        return False, None, "%s" % e
    finally:
        signal.alarm(0)


def _count_stray_children():
    """加固⑨：收割已退出子进程；返回仍存活子进程数（waitpid 无法精确计数——有存活即 1）。"""
    while True:
        try:
            pid, _ = os.waitpid(-1, os.WNOHANG)
        except ChildProcessError:
            return 0  # 已无任何子进程
        except OSError:
            return 0
        if pid == 0:
            return 1  # 存在未退出的存活子进程（后台进程残留）
        # pid > 0：收割一个已退出子进程，继续循环


def _emit(payload, exit_code=0):
    print(json.dumps(payload, ensure_ascii=False))
    sys.stdout.flush()
    sys.exit(exit_code)


def main():
    parser = argparse.ArgumentParser(description="matmul 性能门（加固版：引用快照+三界拒收+ref带+子进程清点）")
    parser.add_argument("--candidate", required=True, help="候选实现路径（暴露 matmul(a,b)）")
    parser.add_argument("--env-dir", default=None,
                        help="env 目录（缺省=脚本上两级；三要素统一契约）")
    parser.add_argument("--size", type=int, default=None)
    parser.add_argument("--k", type=int, default=None)
    parser.add_argument("--timeout", type=int, default=None)
    parser.add_argument("--seed", type=int, default=42, help="随机 seed（固定=可复现）")
    args = parser.parse_args()
    global _ENV_DIR_OVERRIDE, ENV_JSON
    if getattr(args, "env_dir", None):
        _ENV_DIR_OVERRIDE = args.env_dir
        ENV_JSON = os.path.join(args.env_dir, "env.json")

    n, k, env_timeout, score_max, rt_min, ref_band = _load_env_config()
    n = args.size if args.size is not None else n
    k = args.k if args.k is not None else k
    timeout_s = args.timeout if args.timeout is not None else env_timeout
    timeout_s = max(1, int(timeout_s))  # signal.alarm 需整秒
    if k < 1:
        k = 1

    rng = np.random.RandomState(args.seed)
    a = rng.rand(n, n).astype(np.float32)
    b = rng.rand(n, n).astype(np.float32)

    # 加载候选（引用快照已建立——候选 import 期 monkeypatch time.perf_counter 无效）
    try:
        mod = _load_candidate(os.path.abspath(args.candidate))
    except Exception as e:
        _emit({"passed": False, "error": "import: %s" % e, "stage": "import"}, 1)
    fn = getattr(mod, "matmul", None)
    if not callable(fn):
        _emit({"passed": False, "error": "no matmul callable", "stage": "import"}, 1)

    # 候选：预热 1 次 + 计时 K 次
    ok, _, err = _time_once(fn, a, b, timeout_s)
    if not ok:
        _emit({"passed": False, "error": "timeout" if err == "timeout" else "warmup failed: %s" % err,
               "stage": "warmup", "timeout_s": timeout_s}, 1)
    timings = []
    for _ in range(k):
        ok, elapsed, err = _time_once(fn, a, b, timeout_s)
        if not ok:
            _emit({"passed": False, "error": "timeout" if err == "timeout" else "run: %s" % err,
                   "stage": "timed_run", "timeout_s": timeout_s}, 1)
        timings.append(elapsed)
    cand_median = _median(timings)

    # 参考基线（同进程同协议）
    ref = np.matmul
    _time_once(ref, a, b, timeout_s)
    ref_timings = []
    for _ in range(k):
        ok, elapsed, err = _time_once(ref, a, b, timeout_s)
        if ok:
            ref_timings.append(elapsed)
    ref_median = _median(ref_timings) if ref_timings else float("nan")

    score = (ref_median / cand_median) if cand_median > 0 else 0.0

    # ---- 加固① 身份比对：候选企图篡改计时函数即拒（无论篡改手法是否生效）----
    import time as _time_mod
    if _time_mod.perf_counter is not _perf:
        _emit({"passed": False,
               "error": "timer tampering detected: time.perf_counter replaced by candidate",
               "stage": "tamper", "runtime_ms": cand_median, "score": score}, 1)

    # ---- 加固④：计时下界（物理不可能的快）----
    if cand_median < rt_min:
        _emit({"passed": False,
               "error": "runtime_ms %.6f below physical floor %.4f (hardcoded/short-circuit?)"
                        % (cand_median, rt_min),
               "stage": "plausibility", "runtime_ms": cand_median, "score": score}, 1)

    # ---- 加固③：score 上界（作弊或测量异常）----
    if score > score_max:
        _emit({"passed": False,
               "error": "score %.4f exceeds plausibility bound %.2f (timer tampering or "
                        "measurement anomaly)" % (score, score_max),
               "stage": "plausibility", "runtime_ms": cand_median, "score": score,
               "ref_runtime_ms": ref_median}, 1)

    # ---- 加固⑧：ref 合理带（拖慢 ref 抬 score 的通道）----
    if not (ref_band[0] <= ref_median <= ref_band[1]):
        _emit({"passed": False,
               "error": "ref baseline %.4fms outside plausibility band [%s, %s]ms "
                        "(ref sabotaged? machine recalibration needed?)"
                        % (ref_median, ref_band[0], ref_band[1]),
               "stage": "plausibility", "runtime_ms": cand_median, "score": score,
               "ref_runtime_ms": ref_median}, 1)

    # ---- 加固⑨：残留子进程清点（后台忙循环等持久化篡改）----
    strays = _count_stray_children()
    if strays > 0:
        _emit({"passed": False,
               "error": "%d stray child process(es) alive after bench (background "
                        "sabotage process?)" % strays,
               "stage": "plausibility", "runtime_ms": cand_median, "score": score}, 1)

    _emit({
        "runtime_ms": cand_median,
        "score": score,
        "timings": timings,
        "ref_runtime_ms": ref_median,
        "n": n,
        "k": k,
    }, 0)


if __name__ == "__main__":
    main()
