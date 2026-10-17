#!/usr/bin/env python3
"""test_correctness.py — 正确性门（stage2，框架终验的一环；agent 自检仅参考）。

职责（L3-Q2：门禁定义属 env 范畴，本脚本即 env 的门禁实现）：
- 固定 seed 生成 64×64 float32 矩阵（快速校验，规模可 --size 覆盖）
- 加载候选模块，调用其 ``matmul(a, b)``
- 与参考实现 np.matmul 的输出对比，max_error ≤ env.json 声明的容差
- 候选异常 / 超时 / 输出形态错误 → passed=false（优雅降级，不崩栈）

输出协议（gates 框架消费契约）：**stdout 单行 JSON**
    {"passed": true, "max_error": 1.2e-05, "n": 64}
    {"passed": false, "error": "timeout" | "no matmul callable" | ...}
退出码：0=passed / 1=未过 / 2=门自身异常（仍输出 JSON）。

用法（gates 以显式解释器调用——优先 env/.venv/bin/python，fallback python3）：
    python3 data/test_correctness.py --candidate <path/to/candidate.py>
"""
import argparse
import importlib.util
import json
import os
import signal
import sys

import numpy as np

# macOS vecLib BLAS 对 float32 matmul 报虚假 FP 标志（divide/overflow/invalid）——
# 数值本身正常（错误由 max_error 显式校验把关）。抑制该噪声保 stderr 诊断通道干净。
np.seterr(divide="ignore", over="ignore", invalid="ignore")

HERE = os.path.dirname(os.path.abspath(__file__))
ENV_JSON = os.path.join(HERE, "..", "env.json")

DEFAULT_TOLERANCE = 1e-3
DEFAULT_SIZE = 64
DEFAULT_TIMEOUT_S = 30


class _CandidateTimeout(Exception):
    pass


def _alarm_handler(signum, frame):
    raise _CandidateTimeout()


def _load_env_config():
    """读 env.json 的容差与规模声明（门禁定义在 env，不在框架——L3-Q2）。"""
    cfg = {}
    try:
        with open(ENV_JSON, encoding="utf-8") as f:
            cfg = json.load(f)
    except Exception:
        cfg = {}
    tol = cfg.get("tolerance", {}).get("correctness", {}).get("max_error", DEFAULT_TOLERANCE)
    size = cfg.get("correctness", {}).get("size", DEFAULT_SIZE)
    timeout_s = cfg.get("correctness", {}).get("timeout_s", DEFAULT_TIMEOUT_S)
    return float(tol), int(size), int(timeout_s)


def _load_candidate(path):
    spec = importlib.util.spec_from_file_location("candidate", path)
    if spec is None or spec.loader is None:
        raise ImportError("cannot load module from %s" % path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def _emit(payload):
    print(json.dumps(payload, ensure_ascii=False))
    sys.stdout.flush()


def main():
    parser = argparse.ArgumentParser(description="matmul 正确性门")
    parser.add_argument("--candidate", required=True, help="候选实现路径（暴露 matmul(a,b)）")
    parser.add_argument("--size", type=int, default=None, help="矩阵规模（默认读 env.json，64）")
    parser.add_argument("--max-error", type=float, default=None,
                        help="容差覆盖（默认读 env.json，1e-3）")
    parser.add_argument("--timeout", type=int, default=None,
                        help="候选单次执行超时秒（默认读 env.json，30）")
    parser.add_argument("--seed", type=int, default=42, help="随机 seed（固定=可复现）")
    args = parser.parse_args()

    tol, env_size, env_timeout = _load_env_config()
    n = args.size if args.size is not None else env_size
    max_error_allowed = args.max_error if args.max_error is not None else tol
    timeout_s = args.timeout if args.timeout is not None else env_timeout

    # 固定 seed 输入（可复现：同一候选每次门结果一致）
    rng = np.random.RandomState(args.seed)
    a = rng.rand(n, n).astype(np.float32)
    b = rng.rand(n, n).astype(np.float32)
    expected = np.matmul(a, b)  # 参考实现即锚点

    # 加载候选
    try:
        mod = _load_candidate(os.path.abspath(args.candidate))
    except _CandidateTimeout:
        _emit({"passed": False, "error": "timeout", "stage": "import"})
        sys.exit(1)
    except Exception as e:
        _emit({"passed": False, "error": "import: %s" % e, "stage": "import"})
        sys.exit(1)

    fn = getattr(mod, "matmul", None)
    if not callable(fn):
        _emit({"passed": False, "error": "no matmul callable", "stage": "import"})
        sys.exit(1)

    # 执行候选（超时防护：unix signal.alarm，macOS/Linux 可用）
    signal.signal(signal.SIGALRM, _alarm_handler)
    signal.alarm(timeout_s)
    try:
        got = fn(a, b)
    except _CandidateTimeout:
        _emit({"passed": False, "error": "timeout", "stage": "run", "timeout_s": timeout_s})
        sys.exit(1)
    except Exception as e:
        _emit({"passed": False, "error": "run: %s" % e, "stage": "run"})
        sys.exit(1)
    finally:
        signal.alarm(0)

    # 形态与数值校验
    try:
        got_arr = np.asarray(got, dtype=np.float64)
    except Exception as e:
        _emit({"passed": False, "error": "output not array-convertible: %s" % e, "stage": "check"})
        sys.exit(1)
    if got_arr.shape != expected.shape:
        _emit({"passed": False, "error": "shape mismatch: %s vs %s" % (got_arr.shape, expected.shape),
               "stage": "check"})
        sys.exit(1)
    max_error = float(np.max(np.abs(got_arr - expected.astype(np.float64))))
    passed = max_error <= max_error_allowed
    _emit({"passed": passed, "max_error": max_error, "n": n, "tolerance": max_error_allowed})
    sys.exit(0 if passed else 1)


if __name__ == "__main__":
    main()
