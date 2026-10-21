#!/usr/bin/env python3
"""test_correctness.py — 正确性门（stage2，框架终验的一环；agent 自检仅参考）。

env_7 加固版（设计树 §M2 叶子，human 裁决 Q3/Q9——12 项加固清单之⑤⑥⑦⑩）：
- **双规模校验**（⑤）：sizes=[64, 512]（env.json correctness.sizes）——堵「n≤64 正确/n>64 垃圾」
  规模条件捷径。512 规模对 naive 型候选 ~8.4s（< timeout 30s）；比 naive 慢 >3.5× 的候选
  会超时——视为自然淘汰（human 子裁决②）。
- **多 seed 复验**（⑥）：每规模跑固定 seed=42 + 1 个 urandom 派生 seed（防候选特判公开 seed
  的精确输入——Tripwire 场景）；随机 seed 回写输出 JSON 供复现。
- **同输入跑两遍一致性**（⑦）：同一输入两次调用输出逐位一致——抓缓存返回/条件分支特判
  （AlphaEvolve 手册）。
- **候选源静态检查**（⑩）：拒 `sys.modules` / `__main__` 反射 / 对 time 模块属性赋值等
  已知篡改模式（命中即 passed=false，stage=static_check）。

输出协议（gates 框架消费契约）：**stdout 单行 JSON**
    {"passed": true, "max_error": 1.2e-05, "sizes": [64, 512], "seeds": {"64": [42, 123], "512": [42, 456]}}
    {"passed": false, "error": "...", "stage": "static_check|import|run|check"}
退出码：0=passed / 1=未过 / 2=门自身异常（仍输出 JSON）。

用法（gates 以显式解释器调用——优先 env/.venv/bin/python，fallback python3）：
    python3 data/test_correctness.py --candidate <path/to/candidate.py>
"""
import argparse
import importlib.util
import json
import os
import re
import signal
import sys

import numpy as np

# macOS vecLib BLAS 对 float32 matmul 报虚假 FP 标志（SME 进/出 Streaming SVE 模式
# 将 FPSR 全置 1——numpy#28687/PR#29223；数值本身正常，错误由 max_error 显式把关）。
np.seterr(divide="ignore", over="ignore", invalid="ignore")

HERE = os.path.dirname(os.path.abspath(__file__))
ENV_JSON = os.path.join(HERE, "..", "env.json")

DEFAULT_TOLERANCE = 1e-3
DEFAULT_SIZES = [64, 512]
DEFAULT_TIMEOUT_S = 30

# 静态检查：已知篡改/反射模式（human 裁决 Q9-⑩；命中即拒）。
# - sys.modules：候选反射宿主模块（含 sys.modules['__main__'] 变体——主篡改通道）
# - __main__. 属性访问：直接操纵宿主模块对象（注意：不匹配 `if __name__ == "__main__":` 样板——
#   那是候选自检惯例，合法；只匹配 `__main__.xxx` 形态的直接反射访问）
# - time 模块属性赋值（time.perf_counter = ... 等）：篡改计时
# - __builtins__ 赋值：篡改内建
_STATIC_PATTERNS = [
    (r"sys\.modules", "sys.modules reflection"),
    (r"__main__\s*\.", "__main__ attribute access (host module reflection)"),
    (r"\btime\s*\.\s*\w+\s*=", "time attribute assignment"),
    (r"\bsignal\s*\.\s*alarm\s*=", "signal.alarm assignment (disabling gate timeout)"),
    (r"__builtins__", "__builtins__ manipulation"),
]


class _CandidateTimeout(Exception):
    pass


def _alarm_handler(signum, frame):
    raise _CandidateTimeout()


def _load_env_config():
    """读 env.json 的正确性门声明（门禁定义在 env，不在框架）。"""
    cfg = {}
    try:
        with open(ENV_JSON, encoding="utf-8") as f:
            cfg = json.load(f)
    except Exception:
        cfg = {}
    corr = cfg.get("correctness", {})
    tol = cfg.get("tolerance", {}).get("correctness", {}).get("max_error", DEFAULT_TOLERANCE)
    sizes = corr.get("sizes") or [corr.get("size") or 64, 512]
    timeout_s = corr.get("timeout_s", DEFAULT_TIMEOUT_S)
    extra_seeds = int(corr.get("extra_seeds", 1) or 0)
    double_run = bool(corr.get("double_run_consistency", True))
    static_check = bool(cfg.get("gates", {}).get("static_check", True))
    return float(tol), [int(s) for s in sizes], int(timeout_s), extra_seeds, double_run, static_check


def _load_candidate(path):
    spec = importlib.util.spec_from_file_location("candidate", path)
    if spec is None or spec.loader is None:
        raise ImportError("cannot load module from %s" % path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def _static_check_source(path):
    """候选源静态检查（加固⑩）：返回 None（干净）或违规描述。"""
    try:
        with open(path, encoding="utf-8", errors="replace") as f:
            src = f.read()
    except OSError as e:
        return "source unreadable: %s" % e
    for pat, label in _STATIC_PATTERNS:
        if re.search(pat, src):
            return "static check violation: %s (pattern %r)" % (label, pat)
    return None


def _rand_seed():
    """urandom 派生随机 seed（不可预测，回写 JSON 供复现）。"""
    return int.from_bytes(os.urandom(4), "little") % (2 ** 31 - 1)


def _emit(payload, exit_code):
    print(json.dumps(payload, ensure_ascii=False))
    sys.stdout.flush()
    sys.exit(exit_code)


def main():
    parser = argparse.ArgumentParser(description="matmul 正确性门（加固版：双规模+多seed+两遍一致+静态检查）")
    parser.add_argument("--candidate", required=True, help="候选实现路径（暴露 matmul(a,b)）")
    parser.add_argument("--size", type=int, default=None, help="单规模覆盖（兼容旧用法；覆盖双规模）")
    parser.add_argument("--max-error", type=float, default=None, help="容差覆盖（默认读 env.json）")
    parser.add_argument("--timeout", type=int, default=None, help="候选单次执行超时秒")
    parser.add_argument("--seed", type=int, default=42, help="固定 seed（默认 42）")
    args = parser.parse_args()

    tol, sizes, env_timeout, extra_seeds, double_run, static_check = _load_env_config()
    if args.size is not None:
        sizes = [int(args.size)]
    max_error_allowed = args.max_error if args.max_error is not None else tol
    timeout_s = args.timeout if args.timeout is not None else env_timeout

    candidate_abs = os.path.abspath(args.candidate)

    # ---- 静态检查（加固⑩）：先于 import——篡改模式直接拒 ----
    if static_check:
        violation = _static_check_source(candidate_abs)
        if violation is not None:
            _emit({"passed": False, "error": violation, "stage": "static_check"}, 1)

    # ---- 加载候选 ----
    try:
        mod = _load_candidate(candidate_abs)
    except _CandidateTimeout:
        _emit({"passed": False, "error": "timeout", "stage": "import"}, 1)
    except Exception as e:
        _emit({"passed": False, "error": "import: %s" % e, "stage": "import"}, 1)

    fn = getattr(mod, "matmul", None)
    if not callable(fn):
        _emit({"passed": False, "error": "no matmul callable", "stage": "import"}, 1)

    signal.signal(signal.SIGALRM, _alarm_handler)

    def _run_with_timeout(a, b):
        signal.alarm(timeout_s)
        try:
            return fn(a, b)
        finally:
            signal.alarm(0)

    # ---- 双规模 × 多 seed × 两遍一致（加固⑤⑥⑦）----
    overall_max_error = 0.0
    seeds_used = {}
    for n in sizes:
        seeds = [args.seed]
        for _ in range(extra_seeds):
            seeds.append(_rand_seed())
        seeds_used[str(n)] = seeds
        for seed in seeds:
            rng = np.random.RandomState(seed)
            a = rng.rand(n, n).astype(np.float32)
            b = rng.rand(n, n).astype(np.float32)
            expected = np.matmul(a, b)  # 参考实现即锚点（fp32 BLAS 协议）

            try:
                got = _run_with_timeout(a, b)
            except _CandidateTimeout:
                _emit({"passed": False, "error": "timeout", "stage": "run",
                       "n": n, "seed": seed, "timeout_s": timeout_s}, 1)
            except Exception as e:
                _emit({"passed": False, "error": "run: %s" % e, "stage": "run",
                       "n": n, "seed": seed}, 1)

            try:
                got_arr = np.asarray(got, dtype=np.float64)
            except Exception as e:
                _emit({"passed": False, "error": "output not array-convertible: %s" % e,
                       "stage": "check", "n": n, "seed": seed}, 1)
            if got_arr.shape != expected.shape:
                _emit({"passed": False,
                       "error": "shape mismatch: %s vs %s" % (got_arr.shape, expected.shape),
                       "stage": "check", "n": n, "seed": seed}, 1)
            max_error = float(np.max(np.abs(got_arr - expected.astype(np.float64))))
            if max_error > max_error_allowed:
                _emit({"passed": False, "max_error": max_error, "n": n, "seed": seed,
                       "tolerance": max_error_allowed, "stage": "check"}, 1)
            overall_max_error = max(overall_max_error, max_error)

            # 两遍一致性（加固⑦）：同输入再跑一次，输出必须逐位一致（抓缓存/条件分支）
            if double_run:
                try:
                    got2 = _run_with_timeout(a, b)
                except _CandidateTimeout:
                    _emit({"passed": False, "error": "timeout (second run)", "stage": "run",
                           "n": n, "seed": seed}, 1)
                except Exception as e:
                    _emit({"passed": False, "error": "run(second): %s" % e, "stage": "run",
                           "n": n, "seed": seed}, 1)
                try:
                    got2_arr = np.asarray(got2, dtype=np.float64)
                except Exception:
                    _emit({"passed": False, "error": "second output not array-convertible",
                           "stage": "check", "n": n, "seed": seed}, 1)
                if got2_arr.shape != got_arr.shape or not np.array_equal(got2_arr, got_arr):
                    _emit({"passed": False,
                           "error": "inconsistent output on identical input (double-run)",
                           "stage": "check", "n": n, "seed": seed}, 1)

    _emit({"passed": True, "max_error": overall_max_error, "sizes": sizes,
           "seeds": seeds_used, "tolerance": max_error_allowed}, 0)


if __name__ == "__main__":
    main()
