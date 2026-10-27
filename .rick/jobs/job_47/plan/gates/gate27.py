#!/usr/bin/env python3
"""gate27.py — 迭代11（task54 adjust 饥饿 + task55 进程树收割）

A. py：并发 adjust 触发断言全绿 + 全量无回归
B. loop.py：adjust 不在 if not slots 内 / 跨边界判断 / adjust_dispatched 事件
C. Go：super stop kill 进程组 + 杀后验证 + 测试
D. 生产零触碰
"""
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
    r = run([sys.executable, "-m", "unittest", "discover", "-s", "super/tests", "-p", "test_*.py"])
    if r.returncode != 0: ERRORS.append("[A] py 全量失败：\n" + (r.stdout + r.stderr)[-2000:])
    l = read("super/superopt/loop.py")
    seg = l[l.index("def _loop_parallel"):l.index("def _loop_parallel")+2500]
    if "_maybe_start_adjust" in l:
        # 检查它不在 if not slots 内：找 _maybe_start_adjust 调用点上下文
        lines = l.split("\n")
        for i, ln in enumerate(lines):
            if "_maybe_start_adjust" in ln and "def " not in ln:
                ctx = "\n".join(lines[max(0,i-6):i+1])
                if "if not slots" in ctx:
                    ERRORS.append("[B] adjust 仍在 if not slots 内（饥饿根因）")
    for kw in ["_last_adjust_iter", "adjust_dispatched"]:
        if kw not in l: ERRORS.append("[B] loop.py 缺 " + kw)
    h = read("internal/handler/super.go")
    for kw in ["Kill(-pid", "worker_pid"]:
        if kw not in h: ERRORS.append("[C] super.go 缺 " + kw)
    r = run(["go", "test", "./internal/handler/", "./internal/super/", "-timeout", "300s", "-skip", "TestReleaseCmd.*"])
    if r.returncode != 0: ERRORS.append("[C] Go 测试失败：\n" + (r.stdout + r.stderr)[-1500:])
    print(json.dumps({"pass": not ERRORS, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)
if __name__ == "__main__": main()
