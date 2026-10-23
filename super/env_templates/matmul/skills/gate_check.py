#!/usr/bin/env python3
"""gate_check.py — 门禁封装（三要素融合：verify → benchmark → 软门禁）。

**框架 gates 与 agent 共用的唯一门禁入口**（消灭「agent 自检一套、框架终验另一套」
的语义漂移——env_8 it19「自检过、框架拒」实证）：
- 内部依次调 `skills/verify.py`（约束）→ `skills/benchmark.py`（评估 K 次中位）
  → 软门禁（与 --baseline 比较，band 读 env.json tolerance.soft_gate.relative_band）
- 框架侧（superopt/gates.py 新布局）只调本脚本一次并消费其 JSON；
  agent 侧（builder/coding/debug）可随时自行调用拿同一结论。

输出协议：stdout 最后一行 JSON
    {"passed": bool,
     "failure_stage": "stage2_correctness"|"stage3_perf"|"soft_gate"|null,
     "metrics": {"correctness": {...},
                 "perf": {"perf_type","perf_desc","value","timings","score","ref_value","k"},
                 "soft": {"relative_band","baseline","tolerance_band","exempt","verdict","ratio"}},
     "details": "..."}
退出码：0=passed / 1=未过 / 2=门自身异常（仍输出 JSON）。

软门禁语义（与框架 _classify_soft 逐字一致）：
- 无 baseline → exempt（首解豁免），passed=True
- |value/baseline - 1| ≤ band → tolerance_band=True（延迟入库信号），passed=True
- 带外更优 → passed=True；带外劣化 → failure_stage=soft_gate，passed=False

用法：
    python3 skills/gate_check.py --candidate <path|dir> [--baseline <perf>]
                                 [--env-dir <env>] [--k N]
"""
import argparse
import json
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_ENV_DIR = os.path.dirname(HERE)
VERIFY_REL = os.path.join("skills", "verify.py")
BENCHMARK_REL = os.path.join("skills", "benchmark.py")

DEFAULT_BAND = 0.05
DEFAULT_CORRECTNESS_TIMEOUT_S = 120
DEFAULT_BENCH_TIMEOUT_S = 600
_CANDIDATE_NAMES = ("solution.py", "candidate.py", "matmul.py")


def _load_env_config(env_dir):
    cfg = {}
    try:
        with open(os.path.join(env_dir, "env.json"), encoding="utf-8") as f:
            cfg = json.load(f)
    except Exception:
        cfg = {}
    gates_cfg = cfg.get("gates", {}) or {}
    tolerance = cfg.get("tolerance", {}) or {}
    soft = tolerance.get("soft_gate", {}) or {}
    return {
        "cfg": cfg,
        "perf_type": cfg.get("perf_type", "runtime_ms"),
        "perf_desc": cfg.get("perf_desc", "low"),
        "band": float(soft.get("relative_band", DEFAULT_BAND)),
        "correctness_timeout_s": int(gates_cfg.get(
            "correctness_timeout_s", DEFAULT_CORRECTNESS_TIMEOUT_S)),
        "bench_timeout_s": int(gates_cfg.get(
            "bench_timeout_s", DEFAULT_BENCH_TIMEOUT_S)),
    }


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


def _classify_soft(value, baseline, band, perf_desc):
    """软门禁带判定——与 superopt/gates.py `_classify_soft` 逐字一致。

    返回 "better" / "band" / "worse"。
    """
    if value == baseline:
        return "band"
    if baseline > 0:
        ratio = value / baseline
        if perf_desc == "high":
            if ratio < 1.0 - band:
                return "worse"
            if ratio > 1.0 + band:
                return "better"
            return "band"
        # low（默认方向）
        if ratio > 1.0 + band:
            return "worse"
        if ratio < 1.0 - band:
            return "better"
        return "band"
    # 退化基线（≤0）：等值已排除
    if perf_desc == "high":
        return "worse" if value < 0 else "better"
    return "worse" if value > 0 else "better"


def _run_json(py, script_path, args, timeout_s):
    """跑一个 env 脚本并解析最后一行 JSON。返回 (parsed|None, error|None)。"""
    cmd = [py, script_path] + args
    try:
        r = subprocess.run(cmd, timeout=timeout_s, capture_output=True, text=True)
    except subprocess.TimeoutExpired:
        return None, "timeout after %ss: %s" % (timeout_s, script_path)
    except OSError as e:
        return None, "failed to spawn %r: %s" % (script_path, e)
    lines = [ln for ln in (r.stdout or "").split("\n") if ln.strip()]
    for raw in reversed(lines):
        try:
            obj = json.loads(raw.strip())
        except ValueError:
            continue
        if isinstance(obj, dict):
            return obj, None
    tail = ((r.stdout or "") + (r.stderr or ""))[-300:]
    return None, "non-JSON stdout (rc=%s): %r" % (r.returncode, tail)


def _emit(payload):
    print(json.dumps(payload, ensure_ascii=False))
    sys.stdout.flush()


def main():
    parser = argparse.ArgumentParser(description="matmul 三要素融合门禁")
    parser.add_argument("--candidate", required=True,
                        help="候选实现路径（文件或目录）")
    parser.add_argument("--baseline", type=float, default=None,
                        help="软门禁基线 perf（缺省=首解豁免）")
    parser.add_argument("--env-dir", default=None,
                        help="env 目录（默认取脚本上一级）")
    parser.add_argument("--k", type=int, default=None,
                        help="计时次数覆盖（默认读 env.json bench.k）")
    args = parser.parse_args()

    env_dir = os.path.abspath(args.env_dir) if args.env_dir else DEFAULT_ENV_DIR
    conf = _load_env_config(env_dir)
    candidate = resolve_candidate(args.candidate)
    py = sys.executable

    metrics = {}

    # ---- ① 约束（verify） ----
    verify_args = ["--candidate", candidate, "--env-dir", env_dir]
    parsed, err = _run_json(py, os.path.join(env_dir, VERIFY_REL), verify_args,
                            conf["correctness_timeout_s"])
    if err is not None:
        metrics["correctness"] = {"passed": False, "error": err}
        _emit({"passed": False, "failure_stage": "stage2_correctness",
               "metrics": metrics, "details": "verify failed: %s" % err})
        sys.exit(1)
    corr = dict(parsed)
    corr["passed"] = bool(parsed.get("passed"))
    metrics["correctness"] = corr
    if not corr["passed"]:
        _emit({"passed": False, "failure_stage": "stage2_correctness",
               "metrics": metrics,
               "details": "verify rejected: %s" % json.dumps(
                   parsed, ensure_ascii=False)[:300]})
        sys.exit(1)

    # ---- ② 评估（benchmark，K 次中位） ----
    bench_args = ["--candidate", candidate, "--env-dir", env_dir]
    if args.k is not None:
        bench_args += ["--k", str(args.k)]
    bparsed, berr = _run_json(py, os.path.join(env_dir, BENCHMARK_REL),
                              bench_args, conf["bench_timeout_s"])
    if berr is not None:
        metrics["perf"] = {"error": berr}
        _emit({"passed": False, "failure_stage": "stage3_perf",
               "metrics": metrics, "details": "benchmark failed: %s" % berr})
        sys.exit(1)
    if bparsed.get("passed") is False:
        msg = json.dumps(bparsed, ensure_ascii=False)
        metrics["perf"] = {"error": msg}
        _emit({"passed": False, "failure_stage": "stage3_perf",
               "metrics": metrics, "details": "benchmark failed: %s" % msg[:300]})
        sys.exit(1)

    perf_type = conf["perf_type"]
    value = bparsed.get(perf_type)
    if not isinstance(value, (int, float)) or isinstance(value, bool):
        msg = "benchmark output missing numeric %r: %r" % (perf_type, bparsed)
        metrics["perf"] = {"error": msg}
        _emit({"passed": False, "failure_stage": "stage3_perf",
               "metrics": metrics, "details": msg[:300]})
        sys.exit(1)
    value = float(value)
    perf = {
        "perf_type": perf_type,
        "perf_desc": conf["perf_desc"],
        "value": value,
        "timings": bparsed.get("timings") or [],
        "score": bparsed.get("score"),
        "ref_value": bparsed.get("ref_" + perf_type),
        "k": bparsed.get("k"),
    }
    metrics["perf"] = perf

    # ---- ③ 软门禁（与 baseline 比较） ----
    baseline = args.baseline
    band = conf["band"]
    soft = {"relative_band": band, "baseline": baseline,
            "tolerance_band": False, "exempt": baseline is None,
            "verdict": "exempt"}
    if baseline is not None:
        verdict = _classify_soft(value, float(baseline), band, conf["perf_desc"])
        soft["verdict"] = verdict
        if float(baseline) > 0:
            soft["ratio"] = round(value / float(baseline), 6)
        soft["tolerance_band"] = verdict == "band"
        metrics["soft"] = soft
        if verdict == "worse":
            _emit({"passed": False, "failure_stage": "soft_gate",
                   "metrics": metrics,
                   "details": "soft gate: %s=%s worse than baseline=%s "
                              "(band=%.3f, ratio=%s)" % (
                                  perf_type, value, baseline, band,
                                  soft.get("ratio"))})
            sys.exit(1)
    metrics["soft"] = soft

    _emit({"passed": True, "failure_stage": None, "metrics": metrics,
           "details": "passed: verify(max_error=%s), perf(%s=%s, k=%s, score=%s), "
                      "soft(%s)" % (
                          corr.get("max_error"), perf_type, value, perf["k"],
                          perf["score"], soft["verdict"])})
    sys.exit(0)


if __name__ == "__main__":
    main()
