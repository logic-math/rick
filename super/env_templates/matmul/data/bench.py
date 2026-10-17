#!/usr/bin/env python3
"""bench.py — 性能门（stage3，框架终验的一环；K 次计时取中位数）。

职责：
- 固定 seed 生成 512×512 float32 矩阵（L3-Q5；规模可 --size 覆盖）
- 候选预热 1 次 + 计时 K 次（默认 K=3，env.json 可调），runtime_ms = 中位数
- 同协议计时参考实现（numpy.matmul）作基线，score = ref_median / candidate_median（加速比）
- 候选超时（默认 60s）/ 异常 → passed=false（v5 环境失败 78% 的防线之一：
  慢到超时就是超时，不产生半截测量）

输出协议（gates 框架消费契约）：**stdout 单行 JSON**
    {"runtime_ms": 123.4, "score": 0.87, "timings": [..K..], "ref_runtime_ms": 107.2, "n": 512, "k": 3}
    {"passed": false, "error": "timeout"}
退出码：0=测量完成 / 1=超时或异常。

用法（gates 以显式解释器调用——优先 env/.venv/bin/python，fallback python3）：
    python3 data/bench.py --candidate <path/to/candidate.py>
"""
import argparse
import importlib.util
import json
import os
import signal
import sys
import time

import numpy as np

# macOS vecLib BLAS 对 float32 matmul 报虚假 FP 标志（数值正常；见 test_correctness.py 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")

HERE = os.path.dirname(os.path.abspath(__file__))
ENV_JSON = os.path.join(HERE, "..", "env.json")

DEFAULT_SIZE = 512
DEFAULT_K = 3
DEFAULT_TIMEOUT_S = 60


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
    size = bench_cfg.get("size", DEFAULT_SIZE)
    k = bench_cfg.get("k", DEFAULT_K)
    timeout_s = bench_cfg.get("timeout_s", DEFAULT_TIMEOUT_S)
    return int(size), int(k), int(timeout_s)


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
    """单次执行 + 超时防护（unix signal.alarm）。返回 (ok, elapsed_ms, error)。"""
    signal.signal(signal.SIGALRM, _alarm_handler)
    signal.alarm(timeout_s)
    try:
        t0 = time.perf_counter()
        fn(a, b)
        return True, (time.perf_counter() - t0) * 1000.0, None
    except _CandidateTimeout:
        return False, None, "timeout"
    except Exception as e:
        return False, None, "%s" % e
    finally:
        signal.alarm(0)


def _emit(payload):
    print(json.dumps(payload, ensure_ascii=False))
    sys.stdout.flush()


def main():
    parser = argparse.ArgumentParser(description="matmul 性能门（K 次计时中位数）")
    parser.add_argument("--candidate", required=True, help="候选实现路径（暴露 matmul(a,b)）")
    parser.add_argument("--size", type=int, default=None, help="矩阵规模（默认读 env.json，512）")
    parser.add_argument("--k", type=int, default=None, help="计时次数（默认读 env.json，3）")
    parser.add_argument("--timeout", type=int, default=None,
                        help="候选单次执行超时秒（默认读 env.json，60）")
    parser.add_argument("--seed", type=int, default=42, help="随机 seed（固定=可复现）")
    args = parser.parse_args()

    env_size, env_k, env_timeout = _load_env_config()
    n = args.size if args.size is not None else env_size
    k = args.k if args.k is not None else env_k
    timeout_s = args.timeout if args.timeout is not None else env_timeout
    if k < 1:
        k = 1

    # 固定 seed 输入
    rng = np.random.RandomState(args.seed)
    a = rng.rand(n, n).astype(np.float32)
    b = rng.rand(n, n).astype(np.float32)

    # 加载候选
    try:
        mod = _load_candidate(os.path.abspath(args.candidate))
    except Exception as e:
        _emit({"passed": False, "error": "import: %s" % e, "stage": "import"})
        sys.exit(1)
    fn = getattr(mod, "matmul", None)
    if not callable(fn):
        _emit({"passed": False, "error": "no matmul callable", "stage": "import"})
        sys.exit(1)

    # 候选：预热 1 次 + 计时 K 次（任一次超时/异常 → 整体失败，不出半截测量）
    ok, _, err = _time_once(fn, a, b, timeout_s)  # 预热（不计时）
    if not ok:
        _emit({"passed": False, "error": "timeout" if err == "timeout" else "warmup failed: %s" % err,
               "stage": "warmup", "timeout_s": timeout_s})
        sys.exit(1)
    timings = []
    for _ in range(k):
        ok, elapsed, err = _time_once(fn, a, b, timeout_s)
        if not ok:
            _emit({"passed": False, "error": "timeout" if err == "timeout" else "run: %s" % err,
                   "stage": "timed_run", "timeout_s": timeout_s})
            sys.exit(1)
        timings.append(elapsed)
    cand_median = _median(timings)

    # 参考实现基线（同协议预热+K 次计时）
    ref = np.matmul
    _time_once(ref, a, b, timeout_s)
    ref_timings = []
    for _ in range(k):
        ok, elapsed, err = _time_once(ref, a, b, timeout_s)
        if ok:
            ref_timings.append(elapsed)
    ref_median = _median(ref_timings) if ref_timings else float("nan")

    score = (ref_median / cand_median) if cand_median > 0 else 0.0

    _emit({
        "runtime_ms": cand_median,
        "score": score,
        "timings": timings,
        "ref_runtime_ms": ref_median,
        "n": n,
        "k": k,
    })
    sys.exit(0)


if __name__ == "__main__":
    main()
