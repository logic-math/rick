#!/usr/bin/env python3
"""benchmark.py — 评估方法（三要素之「目标」；框架 stage3 与 agent 共用同一脚本）。

职责：
- 固定 seed 生成 n×n float32 矩阵（n 读 env.json bench.size，默认 512；--size 可覆盖）
- 候选预热 1 次 + 计时 K 次（默认 K=3，env.json 可调），runtime_ms = 中位数
- 同协议计时参考实现（numpy.matmul）作基线，score = ref_median / candidate_median
- 候选超时（默认 60s）/ 异常 → passed=false（不产生半截测量）

输出协议（**框架 gates 与 agent 共用契约**）：stdout 最后一行 JSON
    {"runtime_ms": 123.4, "score": 0.87, "timings": [..K..], "ref_runtime_ms": 107.2,
     "n": 512, "k": 3}
    {"passed": false, "error": "timeout", "stage": "warmup"|"timed_run"|"import"}
退出码：0=测量完成 / 1=超时、异常或候选不可用。

用法：
    python3 skills/benchmark.py --candidate <path/to/candidate.py|dir> [--env-dir <env>]
"""
import argparse
import importlib.util
import json
import os
import signal
import sys
import time

import numpy as np

# macOS vecLib BLAS 对 float32 matmul 报虚假 FP 标志（数值正常；见 verify.py 同注释）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_ENV_DIR = os.path.dirname(HERE)

DEFAULT_SIZE = 512
DEFAULT_K = 3
DEFAULT_TIMEOUT_S = 60
_CANDIDATE_NAMES = ("solution.py", "candidate.py", "matmul.py")


class _CandidateTimeout(Exception):
    pass


def _alarm_handler(signum, frame):
    raise _CandidateTimeout()


def _load_env_config(env_dir):
    cfg = {}
    try:
        with open(os.path.join(env_dir, "env.json"), encoding="utf-8") as f:
            cfg = json.load(f)
    except Exception:
        cfg = {}
    bench_cfg = cfg.get("bench", {}) or {}
    size = bench_cfg.get("size", DEFAULT_SIZE)
    k = bench_cfg.get("k", DEFAULT_K)
    timeout_s = bench_cfg.get("timeout_s", DEFAULT_TIMEOUT_S)
    return int(size), int(k), int(timeout_s)


def resolve_candidate(path):
    """候选路径解析：文件原样；目录 → 约定名（solution.py 等）→ 唯一 .py 兜底。"""
    p = os.path.abspath(path)
    if os.path.isfile(p):
        return p
    if os.path.isdir(p):
        for name in _CANDIDATE_NAMES:
            cand = os.path.join(p, name)
            if os.path.isfile(cand):
                return cand
        pys = sorted(
            os.path.join(p, f) for f in os.listdir(p)
            if f.endswith(".py") and not f.startswith("_"))
        if len(pys) == 1:
            return pys[0]
    return p


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
    parser = argparse.ArgumentParser(description="matmul 评估门（K 次计时中位数）")
    parser.add_argument("--candidate", required=True,
                        help="候选实现路径（暴露 matmul(a,b)）")
    parser.add_argument("--env-dir", default=None,
                        help="env 目录（默认取脚本上一级：skills/ 的父目录）")
    parser.add_argument("--size", type=int, default=None,
                        help="矩阵规模（默认读 env.json）")
    parser.add_argument("--k", type=int, default=None,
                        help="计时次数（默认读 env.json）")
    parser.add_argument("--timeout", type=int, default=None,
                        help="候选单次执行超时秒（默认读 env.json）")
    parser.add_argument("--seed", type=int, default=42, help="随机 seed（固定=可复现）")
    args = parser.parse_args()

    env_dir = os.path.abspath(args.env_dir) if args.env_dir else DEFAULT_ENV_DIR
    env_size, env_k, env_timeout = _load_env_config(env_dir)
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
    candidate = resolve_candidate(args.candidate)
    try:
        mod = _load_candidate(candidate)
    except Exception as e:
        _emit({"passed": False, "error": "import: %s" % e, "stage": "import",
               "candidate": candidate})
        sys.exit(1)
    fn = getattr(mod, "matmul", None)
    if not callable(fn):
        _emit({"passed": False, "error": "no matmul callable", "stage": "import",
               "candidate": candidate})
        sys.exit(1)

    # 候选：预热 1 次 + 计时 K 次（任一次超时/异常 → 整体失败，不出半截测量）
    ok, _, err = _time_once(fn, a, b, timeout_s)  # 预热（不计时）
    if not ok:
        _emit({"passed": False,
               "error": "timeout" if err == "timeout" else "warmup failed: %s" % err,
               "stage": "warmup", "timeout_s": timeout_s})
        sys.exit(1)
    timings = []
    for _ in range(k):
        ok, elapsed, err = _time_once(fn, a, b, timeout_s)
        if not ok:
            _emit({"passed": False,
                   "error": "timeout" if err == "timeout" else "run: %s" % err,
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
