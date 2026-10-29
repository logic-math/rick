#!/usr/bin/env python3
"""gate29.py — 迭代13（R1-R5 全部落地验证）"""
import json, os, subprocess, sys
def root():
    p = os.path.abspath(__file__)
    for _ in range(6): p = os.path.dirname(p)
    return p
ROOT = root(); ERRORS = []
def run(cmd, cwd=ROOT, timeout=2400):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)
def read(p):
    with open(os.path.join(ROOT, p), encoding="utf-8") as f: return f.read()
def main():
    l = read("super/superopt/loop.py")
    # R1 熔断
    for kw in ["api_outage_detected", "api_outage_recovered", "outcome", "infra_noop"]:
        if kw not in l: ERRORS.append("[R1] loop.py 缺 " + kw)
    # R2 产物抢救
    for kw in ["artifact_rescued", "agent_completed_but_no_json"]:
        if kw not in l: ERRORS.append("[R2] loop.py 缺 " + kw)
    # R3 幻影拒绝
    for kw in ["elite_code_hash", "elite_measurement_refresh"]:
        if kw not in l: ERRORS.append("[R3] loop.py 缺 " + kw)
    a = read("super/superopt/archive.py")
    if "elite_code_hash" not in a: ERRORS.append("[R3] archive 缺 elite_code_hash")
    # R4 method_stale
    for kw in ["method_stale", "set_stale", "[STALE]"]:
        if kw not in l and kw not in a: ERRORS.append("[R4] 缺 " + kw)
    # R5 噪声标定 + 停止判据
    for kw in ["noise_calibrated", "noise_band", "top3_spread", "headroom_below_noise"]:
        if kw not in l: ERRORS.append("[R5] loop.py 缺 " + kw)
    # config
    c = read("super/superopt/config.py")
    for kw in ["noise_calibration_interval", "ref_drift_threshold"]:
        if kw not in c: ERRORS.append("[R5] config 缺 " + kw)
    # events
    e = read("super/superopt/events.py")
    for kw in ["api_outage_recovered", "elite_measurement_refresh", "noise_calibrated", "artifact_rescued"]:
        if kw not in e: ERRORS.append("[events] 缺 " + kw)
    # 测试
    r = run([sys.executable, "-m", "unittest", "discover", "-s", "super/tests", "-p", "test_*.py"])
    if r.returncode != 0: ERRORS.append("[test] 全量失败：\n" + (r.stdout + r.stderr)[-2000:])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)
if __name__ == "__main__": main()
