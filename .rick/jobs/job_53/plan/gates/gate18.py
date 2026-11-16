#!/usr/bin/env python3
"""job_53 gate18：Bug #9——刷新弹「平台升级」通知移除。"""
import os, subprocess, sys

DEV_TREE = os.environ.get("DEV_TREE", "/Users/logic/workdir/rick-dev")
PROD_REPO = os.environ.get("PROD_REPO", "/Users/logic/workdir/AI_CODING/rick")
GATE_ENV = {k: v for k, v in os.environ.items() if not k.startswith("RICK_")}
errors = []

def check(label, ok, detail=""):
    print(("PASS" if ok else "FAIL"), label, detail)
    if not ok: errors.append(label)

def read(p):
    try:
        with open(p) as f: return f.read()
    except OSError: return None

app = read(os.path.join(DEV_TREE, "web/src/App.tsx")) or ""
check("N1 RecoveryBanner 已移除", "<RecoveryBanner" not in app and "RecoveryBanner" not in app)

r = subprocess.run(["npm", "run", "typecheck"], cwd=os.path.join(DEV_TREE, "web"), capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 npm typecheck", r.returncode == 0)
r = subprocess.run(["npm", "run", "build"], cwd=os.path.join(DEV_TREE, "web"), capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E1 npm build", r.returncode == 0)

r = subprocess.run(["git", "-C", PROD_REPO, "status", "--porcelain", "-uno"], capture_output=True, text=True, env=GATE_ENV, timeout=60)
dirty = [l for l in r.stdout.splitlines() if l.strip() and l.split()[-1] != ".rick/jobs/job_50/doing/tasks.json"]
check("P1 生产树零改动", len(dirty) == 0)

if errors:
    print(f"GATE18 FAIL: {len(errors)} 项"); sys.exit(1)
print("GATE18 PASS: RecoveryBanner 移除"); sys.exit(0)
