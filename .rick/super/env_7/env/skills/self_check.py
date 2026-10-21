#!/usr/bin/env python3
"""self_check.py — agent 自检脚本（快速反馈，非终验）。

coding agent 在 codes/{uuid}/ 内实现后跑本脚本快速自查：
正确性（test_correctness.py）+ 单次性能测量（bench.py --k 1）。

⚠️ 自检结果仅供参考，入格以框架 gates.py 终验为准（J6 裁决「以框架为准」——
框架会独立跑同一套门，且带 K 次计时中位数 + 软门禁容差判定）。

输出协议：stdout 单行 JSON
    {"correctness": {...}, "perf": {...}, "note": "..."}

用法：
    python3 skills/self_check.py --candidate <path/to/candidate.py>
"""
import argparse
import json
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
TEST_CORRECTNESS = os.path.join(HERE, "..", "data", "test_correctness.py")
BENCH = os.path.join(HERE, "..", "data", "bench.py")


def _run_json(cmd, timeout=300):
    """跑子脚本并解析其单行 JSON 输出；失败返回 {"error": ...}。"""
    try:
        r = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
        lines = [ln for ln in r.stdout.strip().splitlines() if ln.strip()]
        if not lines:
            return {"error": "no output", "exit": r.returncode,
                    "stderr": (r.stderr or "")[-200:]}
        return json.loads(lines[-1])
    except subprocess.TimeoutExpired:
        return {"error": "self_check subprocess timeout"}
    except Exception as e:
        return {"error": "%s" % e}


def main():
    parser = argparse.ArgumentParser(description="matmul agent 自检（仅供参考）")
    parser.add_argument("--candidate", required=True, help="候选实现路径")
    parser.add_argument("--size", type=int, default=None, help="正确性/性能规模覆盖")
    args = parser.parse_args()

    candidate = os.path.abspath(args.candidate)

    corr_cmd = [sys.executable, TEST_CORRECTNESS, "--candidate", candidate]
    perf_cmd = [sys.executable, BENCH, "--candidate", candidate, "--k", "1"]
    if args.size is not None:
        corr_cmd += ["--size", str(args.size)]
        perf_cmd += ["--size", str(args.size)]

    correctness = _run_json(corr_cmd)
    perf = _run_json(perf_cmd)

    payload = {
        "correctness": correctness,
        "perf": perf,
        "note": "自检结果仅供参考，入格以框架 gates.py 终验为准（K 次计时中位数+软门禁容差）",
    }
    print(json.dumps(payload, ensure_ascii=False))
    corr_passed = isinstance(correctness, dict) and correctness.get("passed") is True
    sys.exit(0 if corr_passed else 1)


if __name__ == "__main__":
    main()
