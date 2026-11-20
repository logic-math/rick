#!/usr/bin/env python3
"""profile.py — 事实获取（三要素之「现状」；agent 随时调用，框架不做门禁判定）。

泛化语义：**事实信息获取**——回答「现在是什么样」而非「好不好」：
- hotspots：候选实现的耗时热点（cProfile 自耗时排序）——往哪里优化
- env_facts：运行环境事实（python/numpy 版本、BLAS 后端、CPU 核数、内存带宽估算）
- parent_facts：候选自身的结构事实（代码行数、import、顶层函数、单次实测耗时）

输出协议（agent 消费）：stdout 最后一行 JSON
    {"ok": true, "n": 512, "hotspots": [{"function","self_ms","cum_ms","pct"}...],
     "env_facts": {"python","numpy","blas","cpu_count","memcpy_gbps","dtype"},
     "parent_facts": {"candidate","loc","imports","functions","runtime_ms"}}
    {"ok": false, "error": "..."}   # 候选不可用等；事实缺失但协议合法
退出码：0=已产出事实（无论 ok）——profile 不是门禁，不阻断迭代。

用法：
    python3 skills/profile.py --candidate <path/to/candidate.py|dir> [--env-dir <env>]
                              [--size N] [--top N] [--hotspots 0|1]
"""
import argparse
import ast
import contextlib
import io
import json
import os
import signal
import sys
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_ENV_DIR = os.path.dirname(HERE)
DEFAULT_SIZE = 512
DEFAULT_TIMEOUT_S = 60
DEFAULT_TOP = 8
_CANDIDATE_NAMES = ("solution.py", "candidate.py", "matmul.py")


class _CandidateTimeout(Exception):
    pass


def _alarm_handler(signum, frame):
    raise _CandidateTimeout()


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


def _load_env_config(env_dir):
    cfg = {}
    try:
        with open(os.path.join(env_dir, "env.json"), encoding="utf-8") as f:
            cfg = json.load(f)
    except Exception:
        cfg = {}
    bench_cfg = cfg.get("bench", {}) or {}
    size = bench_cfg.get("size", DEFAULT_SIZE)
    timeout_s = bench_cfg.get("timeout_s", DEFAULT_TIMEOUT_S)
    dtype = bench_cfg.get("dtype", "float32")
    return int(size), int(timeout_s), str(dtype)


def _load_candidate(path):
    import importlib.util
    spec = importlib.util.spec_from_file_location("candidate", path)
    if spec is None or spec.loader is None:
        raise ImportError("cannot load module from %s" % path)
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def _emit(payload):
    print(json.dumps(payload, ensure_ascii=False))
    sys.stdout.flush()


# ---- 三块事实 ----

def _env_facts(dtype):
    """运行环境事实：版本 / BLAS 后端 / CPU / 内存带宽估算。"""
    facts = {
        "python": "%d.%d.%d" % sys.version_info[:3],
        "numpy": getattr(np, "__version__", "unknown"),
        "cpu_count": os.cpu_count() or 0,
        "dtype": dtype,
        "blas": "unknown",
        "memcpy_gbps": None,
    }
    # BLAS 后端：捕获 numpy 配置输出（不污染 stdout 协议；屏蔽 pyyaml 提示警告）
    try:
        import warnings
        buf = io.StringIO()
        with contextlib.redirect_stdout(buf), warnings.catch_warnings():
            warnings.simplefilter("ignore")
            try:
                np.show_config()
            except Exception:
                np.__config__.show()
        text = buf.getvalue().lower()
        for key in ("accelerate", "openblas", "mkl", "blis", "veclib"):
            if key in text:
                facts["blas"] = key
                break
    except Exception:
        pass
    # 内存带宽粗估：一次 16MiB 拷贝（读写各一遍）
    try:
        nbytes = 16 * 1024 * 1024
        src = np.ones(nbytes // 4, dtype=np.float32)
        t0 = time.perf_counter()
        _dst = src.copy()
        dt = time.perf_counter() - t0
        if dt > 0:
            facts["memcpy_gbps"] = round((nbytes * 2) / dt / 1e9, 1)
    except Exception:
        pass
    return facts


def _parent_facts(candidate):
    """候选结构事实：行数 / import / 顶层函数。"""
    facts = {"candidate": candidate, "loc": None, "imports": [], "functions": []}
    try:
        with open(candidate, encoding="utf-8") as f:
            src = f.read()
        facts["loc"] = len(src.splitlines())
        tree = ast.parse(src)
        imports = set()
        funcs = []
        for node in tree.body:
            if isinstance(node, ast.Import):
                for a in node.names:
                    imports.add(a.name.split(".")[0])
            elif isinstance(node, ast.ImportFrom):
                if node.module:
                    imports.add(node.module.split(".")[0])
            elif isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                funcs.append(node.name)
        facts["imports"] = sorted(imports)
        facts["functions"] = funcs
    except Exception as e:
        facts["error"] = "%s" % e
    return facts


def _timed_call(fn, a, b, timeout_s):
    """单次执行 + 超时防护；返回 (ok, ms, error)。"""
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


def _sanitize_sys_path():
    """把本脚本所在目录移出 sys.path——脚本名 `profile.py` 会遮蔽 stdlib `profile`，
    cProfile 内部 `import profile` 会命中本文件而 AttributeError（实测）。
    三要素契约要求文件必须叫 profile.py，故在导入 cProfile 前净化路径。"""
    here = os.path.dirname(os.path.abspath(__file__))
    keep = []
    for entry in sys.path:
        try:
            if os.path.abspath(entry or os.getcwd()) == here:
                continue
        except Exception:
            pass
        keep.append(entry)
    sys.path[:] = keep


def _hotspots(fn, a, b, timeout_s, top):
    """cProfile 热点（stdlib）：按 self 时间排序取 top-N。"""
    _sanitize_sys_path()
    try:
        import cProfile
        import pstats
    except Exception as e:  # 绝不因剖析失败而丢事实
        return [], "cProfile unavailable: %s" % e
    prof = cProfile.Profile()
    signal.signal(signal.SIGALRM, _alarm_handler)
    signal.alarm(timeout_s)
    try:
        prof.enable()
        try:
            fn(a, b)
        finally:
            prof.disable()
    except _CandidateTimeout:
        return [], "timeout"
    except Exception as e:
        return [], "%s" % e
    finally:
        signal.alarm(0)
    stats = pstats.Stats(prof)
    rows = []
    total_self = 0.0
    for (_fname, _lineno, func_name), (_cc, _nc, tt, ct, _callers) in stats.stats.items():
        total_self += tt
        rows.append({"function": func_name, "self_ms": tt * 1000.0,
                     "cum_ms": ct * 1000.0})
    rows.sort(key=lambda r: r["self_ms"], reverse=True)
    rows = rows[:top]
    if total_self > 0:
        for r in rows:
            r["pct"] = round(r["self_ms"] / (total_self * 1000.0) * 100.0, 1)
    return rows, None


def main():
    parser = argparse.ArgumentParser(description="matmul 现状剖析（事实，非门禁）")
    parser.add_argument("--candidate", required=True, help="候选实现路径")
    parser.add_argument("--env-dir", default=None,
                        help="env 目录（默认取脚本上一级）")
    parser.add_argument("--size", type=int, default=None,
                        help="矩阵规模（默认读 env.json bench.size）")
    parser.add_argument("--timeout", type=int, default=None,
                        help="单次执行超时秒（默认读 env.json）")
    parser.add_argument("--top", type=int, default=DEFAULT_TOP,
                        help="热点函数个数（默认 %d）" % DEFAULT_TOP)
    parser.add_argument("--hotspots", type=int, default=1,
                        help="1=跑 cProfile 采集热点（默认）/ 0=跳过（省时）")
    args = parser.parse_args()

    env_dir = os.path.abspath(args.env_dir) if args.env_dir else DEFAULT_ENV_DIR
    env_size, env_timeout, dtype = _load_env_config(env_dir)
    n = args.size if args.size is not None else env_size
    timeout_s = args.timeout if args.timeout is not None else env_timeout

    try:
        payload = _gather(args, n, timeout_s, dtype)
    except Exception as e:  # profile 是事实采集，不是门禁——任何异常都出合法 JSON
        payload = {"ok": False, "n": n, "error": "profile crashed: %s" % e,
                   "hotspots": [], "env_facts": {}, "parent_facts": {}}
    _emit(payload)
    sys.exit(0)


def _gather(args, n, timeout_s, dtype):
    """采集三块事实并返回 payload（异常由 main 兜底）。"""
    candidate = resolve_candidate(args.candidate)
    payload = {"ok": False, "n": n, "hotspots": [],
               "env_facts": _env_facts(dtype),
               "parent_facts": _parent_facts(candidate)}

    # 候选加载
    try:
        mod = _load_candidate(candidate)
    except Exception as e:
        payload["error"] = "import: %s" % e
        return payload
    fn = getattr(mod, "matmul", None)
    if not callable(fn):
        payload["error"] = "no matmul callable"
        return payload

    rng = np.random.RandomState(42)
    a = rng.rand(n, n).astype(np.float32)
    b = rng.rand(n, n).astype(np.float32)

    # 父本现状：单次实测耗时
    ok, ms, err = _timed_call(fn, a, b, timeout_s)
    payload["parent_facts"]["runtime_ms"] = round(ms, 3) if ok else None
    if not ok:
        payload["parent_facts"]["runtime_error"] = err

    # 热点剖析（候选可跑才做）
    if ok and args.hotspots:
        rows, herr = _hotspots(fn, a, b, timeout_s, max(1, args.top))
        payload["hotspots"] = rows
        if herr:
            payload["hotspots_error"] = herr

    payload["ok"] = ok
    return payload


if __name__ == "__main__":
    main()
