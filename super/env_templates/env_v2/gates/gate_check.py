#!/usr/bin/env python3
"""gate_check.py — v2 验证器入口（多维融合分协议，框架必然调用）。

v2 布局（job_56/task2，human 裁决 Q6）核心语义：
- **多维优化目标**：env.json `objective` 声明多约束 + 多追求指标（如「A>1 前提下
  B/C 越大越好」）——本入口校验约束、加权融合出**单一融合分**
- **融合分协议**：score ≥ 0 = 通过（进化分数，越高越好）；score < 0 = 拒绝：
    -1 环境错误 / -2 正确性失败 / -3 约束违反 / -4 软门禁劣化
  框架侧（superopt/gates.py `_verify_gate_check_v2`）强校验 score≥0 ⟺ passed
- **验证器隔离**：本文件位于 gates/（builder/coding/debug 禁读；adjust 可读写维护；
  verify/benchmark skill 经 skills/ 薄入口**执行引用** gates/verify_impl.py /
  benchmark_impl.py——同一实现，无复制）

内部级联：verify（gates/verify_impl.py）→ benchmark K 次中位（gates/benchmark_impl.py）
→ 约束校验（objective.constraints）→ 加权融合（objective.metrics）
→ 软门禁（--baseline 融合分比较，band 读 env.json tolerance.soft_gate.relative_band）。

指标提取（全部归一为「越高越好」后再加权）：
  派生表（若 benchmark 输出 ref_<x>）：<x> 的 speedup = ref_<x> / <x>，
  派生键名 "speedup"（benchmark 输出 ref_runtime_ms 时 speedup = ref/runtime）
  直接表：benchmark 输出的其余数值键（注意 low 方向的原始指标——如 runtime_ms——
  不要直接当融合指标；用派生 speedup 或 env 在 benchmark 输出自带 score）
  objective.metrics[].name 按序查：派生表 → 直接表；查无该指标 → 环境错误（-1）
约束取值：verify 输出键（如 max_error）→ benchmark 输出键；查无 → 环境错误（-1）

输出协议（stdout 最后一行 JSON，框架侧 v2 解析契约）：
    {"passed": bool,
     "score": <融合分或负失败码>,
     "metrics": {"<指标名>": <K 次中位>, ...},
     "constraints": {"<约束名>": {"value", "op", "target", "satisfied"}, ...},
     "diagnostics": "...",
     "failure_stage": "stage2_correctness"|"stage3_perf"|"soft_gate"|null,
     "tolerance_band"?: bool}
退出码：0=passed / 1=未过 / 2=门自身异常（仍输出 JSON）。

用法：
    python3 gates/gate_check.py --candidate <path|dir> [--baseline <融合分>]
                                [--env-dir <env>] [--k N]
"""
import json
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_ENV_DIR = os.path.dirname(HERE)
VERIFY_IMPL = os.path.join(HERE, "verify_impl.py")
BENCHMARK_IMPL = os.path.join(HERE, "benchmark_impl.py")

DEFAULT_BAND = 0.05
DEFAULT_K = 3
DEFAULT_VERIFY_TIMEOUT_S = 120
DEFAULT_BENCH_TIMEOUT_S = 600

# 负分编码（协议契约：框架侧按 score<0 拒；数值可携带失败类别）
SCORE_ENV_ERROR = -1
SCORE_CORRECTNESS_FAIL = -2
SCORE_CONSTRAINT_VIOLATION = -3
SCORE_SOFT_GATE_WORSE = -4


def _emit(payload, rc):
    print(json.dumps(payload, ensure_ascii=False))
    sys.exit(rc)


def _median(vals):
    vs = sorted(vals)
    m = len(vs)
    if m == 0:
        return None
    if m % 2 == 1:
        return vs[m // 2]
    return (vs[m // 2 - 1] + vs[m // 2]) / 2.0


def _num(v):
    if isinstance(v, bool):
        return None
    if isinstance(v, (int, float)):
        return float(v)
    return None


def _run_json(py, script, args, timeout_s):
    try:
        r = subprocess.run([py, script] + args, timeout=timeout_s,
                           capture_output=True, text=True)
    except subprocess.TimeoutExpired:
        return None, "timeout after %.1fs: %s" % (timeout_s, script)
    except OSError as e:
        return None, "failed to spawn %s: %s" % (script, e)
    for raw in reversed([x for x in (r.stdout or "").split("\n") if x.strip()]):
        try:
            obj = json.loads(raw)
        except ValueError:
            continue
        if isinstance(obj, dict):
            return obj, None
    return None, ("non-JSON stdout (rc=%s); stderr_tail=%r"
                  % (r.returncode, (r.stderr or "")[-200:]))


def _check_constraint(value, op, target):
    if op == "<=":
        return value <= target
    if op == ">=":
        return value >= target
    if op == "<":
        return value < target
    if op == ">":
        return value > target
    if op == "==":
        return value == target
    if op == "!=":
        return value != target
    return False


def main():
    args = sys.argv[1:]
    candidate = None
    baseline = None
    env_dir = DEFAULT_ENV_DIR
    k_override = None
    i = 0
    while i < len(args):
        a = args[i]
        if a == "--candidate" and i + 1 < len(args):
            candidate = args[i + 1]
            i += 2
            continue
        if a == "--baseline" and i + 1 < len(args):
            try:
                baseline = float(args[i + 1])
            except ValueError:
                baseline = None
            i += 2
            continue
        if a == "--env-dir" and i + 1 < len(args):
            env_dir = args[i + 1]
            i += 2
            continue
        if a == "--k" and i + 1 < len(args):
            try:
                k_override = int(args[i + 1])
            except ValueError:
                k_override = None
            i += 2
            continue
        i += 1

    if not candidate:
        _emit({"passed": False, "score": SCORE_ENV_ERROR, "metrics": {},
               "constraints": {}, "diagnostics": "missing --candidate",
               "failure_stage": "stage3_perf"}, 2)

    # env 配置
    cfg = {}
    try:
        with open(os.path.join(env_dir, "env.json"), encoding="utf-8") as f:
            cfg = json.load(f)
    except Exception:
        cfg = {}
    gates_cfg = cfg.get("gates") or {}
    tolerance = cfg.get("tolerance") or {}
    soft_cfg = tolerance.get("soft_gate") or {}
    bench_cfg = cfg.get("bench") or {}
    band_raw = soft_cfg.get("relative_band", DEFAULT_BAND)
    band = (float(band_raw)
            if isinstance(band_raw, (int, float)) and not isinstance(band_raw, bool)
            else DEFAULT_BAND)
    k = int(k_override or bench_cfg.get("k") or DEFAULT_K)
    verify_timeout = int(gates_cfg.get("correctness_timeout_s",
                                       DEFAULT_VERIFY_TIMEOUT_S))
    bench_timeout = int(gates_cfg.get("bench_timeout_s", DEFAULT_BENCH_TIMEOUT_S))
    objective = cfg.get("objective") if isinstance(cfg.get("objective"), dict) else {}

    py = sys.executable
    venv_py = os.path.join(env_dir, ".venv", "bin", "python")
    if os.path.exists(venv_py):
        py = venv_py

    metrics_out = {}
    constraints_out = {}

    # ---- ① verify（正确性） ----
    v, verr = _run_json(py, VERIFY_IMPL, ["--candidate", candidate,
                                          "--env-dir", env_dir], verify_timeout)
    if verr is not None or v is None:
        _emit({"passed": False, "score": SCORE_ENV_ERROR, "metrics": metrics_out,
               "constraints": constraints_out,
               "diagnostics": "verify env error: %s" % verr,
               "failure_stage": "stage2_correctness"}, 2)
    if not v.get("passed"):
        _emit({"passed": False, "score": SCORE_CORRECTNESS_FAIL,
               "metrics": metrics_out, "constraints": constraints_out,
               "diagnostics": "verify rejected candidate: %s"
                              % json.dumps(v, ensure_ascii=False),
               "failure_stage": "stage2_correctness"}, 1)

    # ---- ② benchmark K 次中位 ----
    runs = []
    for _ in range(k):
        b, berr = _run_json(py, BENCHMARK_IMPL, ["--candidate", candidate,
                                                 "--env-dir", env_dir],
                            bench_timeout)
        if berr is not None or b is None:
            _emit({"passed": False, "score": SCORE_ENV_ERROR, "metrics": metrics_out,
                   "constraints": constraints_out,
                   "diagnostics": "benchmark env error: %s" % berr,
                   "failure_stage": "stage3_perf"}, 2)
        if b.get("passed") is False:
            _emit({"passed": False, "score": SCORE_ENV_ERROR, "metrics": metrics_out,
                   "constraints": constraints_out,
                   "diagnostics": "benchmark reported failure: %s"
                                  % json.dumps(b, ensure_ascii=False),
                   "failure_stage": "stage3_perf"}, 1)
        runs.append(b)

    # 指标归一（越高越好）：派生表（ref 存在 → speedup）+ 直接表（benchmark 数值键）
    derived = {}
    direct = {}
    for b in runs:
        for key, val in b.items():
            n = _num(val)
            if n is None:
                continue
            direct.setdefault(key, []).append(n)
            if key.startswith("ref_"):
                base_key = key[4:]
                if base_key in b and _num(b[base_key]) not in (None, 0):
                    derived.setdefault("speedup", []).append(n / _num(b[base_key]))

    # ---- ③ 约束校验（objective.constraints） ----
    unsatisfied = []
    for c in (objective.get("constraints") or []):
        if not isinstance(c, dict):
            continue
        name = c.get("name")
        op = c.get("op", "<=")
        target = _num(c.get("value"))
        if not isinstance(name, str) or target is None:
            continue
        src = v if _num(v.get(name)) is not None else (runs[-1] if runs else {})
        val = _num(src.get(name)) if isinstance(src, dict) else None
        if val is None:
            _emit({"passed": False, "score": SCORE_ENV_ERROR, "metrics": metrics_out,
                   "constraints": constraints_out,
                   "diagnostics": "constraint %r value not found in verify/benchmark "
                                  "output" % name,
                   "failure_stage": "stage3_perf"}, 2)
        ok = _check_constraint(val, op, target)
        constraints_out[name] = {"value": val, "op": op, "target": target,
                                 "satisfied": ok}
        if not ok:
            unsatisfied.append(name)
    if unsatisfied:
        _emit({"passed": False, "score": SCORE_CONSTRAINT_VIOLATION,
               "metrics": metrics_out, "constraints": constraints_out,
               "diagnostics": "constraint(s) violated: %s"
                              % ", ".join(unsatisfied),
               "failure_stage": "stage3_perf"}, 1)

    # ---- ④ 加权融合（objective.metrics，weighted_avg） ----
    mdefs = [m for m in (objective.get("metrics") or [])
             if isinstance(m, dict) and isinstance(m.get("name"), str)]
    if mdefs:
        weights, values = [], []
        for m in mdefs:
            name = m["name"]
            w = _num(m.get("weight", 1.0)) or 1.0
            series = derived.get(name) or direct.get(name)
            if not series:
                _emit({"passed": False, "score": SCORE_ENV_ERROR,
                       "metrics": metrics_out, "constraints": constraints_out,
                       "diagnostics": "objective metric %r not found in benchmark "
                                      "output (derived keys: %s)"
                                      % (name, sorted(derived) + sorted(direct)),
                       "failure_stage": "stage3_perf"}, 2)
            med = _median(series)
            metrics_out[name] = med
            weights.append(w)
            values.append(med)
        score = sum(w * x for w, x in zip(weights, values)) / sum(weights)
    else:
        # 无 objective.metrics：退化用 benchmark 自带 score（K 次中位）；
        # 无 score → 环境配置错误（v2 要求 objective 声明融合指标）
        scores = direct.get("score") or []
        if not scores:
            _emit({"passed": False, "score": SCORE_ENV_ERROR, "metrics": metrics_out,
                   "constraints": constraints_out,
                   "diagnostics": "env.json objective.metrics missing and benchmark "
                                  "output has no score — cannot fuse",
                   "failure_stage": "stage3_perf"}, 2)
        score = _median(scores)
        metrics_out["score"] = score

    # ---- ⑤ 软门禁（融合分基线，越高越好） ----
    payload = {"passed": True, "score": score, "metrics": metrics_out,
               "constraints": constraints_out, "failure_stage": None}
    if baseline is None:
        payload["diagnostics"] = "passed (exempt: no baseline)"
        payload["tolerance_band"] = False
        _emit(payload, 0)
    if baseline > 0 and score < baseline * (1.0 - band):
        payload.update({"passed": False, "score": SCORE_SOFT_GATE_WORSE,
                        "failure_stage": "soft_gate",
                        "diagnostics": "soft gate: fused score %s worse than baseline "
                                       "%s beyond band %.4f" % (score, baseline, band)})
        _emit(payload, 1)
    payload["tolerance_band"] = bool(baseline > 0
                                     and abs(score / baseline - 1.0) <= band)
    payload["diagnostics"] = ("passed via gates/gate_check.py (v2): fused=%s, "
                              "baseline=%s" % (score, baseline))
    _emit(payload, 0)


if __name__ == "__main__":
    main()
