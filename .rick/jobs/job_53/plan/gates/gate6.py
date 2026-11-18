#!/usr/bin/env python3
"""job_53 gate6：Bug #3（侧栏可拖宽 + 右下角 Rick and Morty 台词）层门禁。

human 反馈（2026-11-14）：
  1. 左侧会话栏可拖动改变宽度（长标题不再被迫截断）
  2. 右下角英文改成最能体现 Rick and Morty 特色的话：「wubba lubba dub dubs」

断言清单：
  R1 存在性：侧栏宽度可拖动（cursor-col-resize 拖动手柄 + 宽度状态/持久化）
  R2 存在性：宽度持久化（localStorage 键）+ clamp 边界（min/max 宽度）
  T1 否定性：旧英文「guardians of the context」已删
  T2 存在性：「wubba lubba dub dubs」已上墙
  E1 构建绿：npm typecheck + build；go build（防误伤）
  P1 生产零触碰
"""
import os
import subprocess
import sys

DEV_TREE = os.environ.get("DEV_TREE", "/Users/logic/workdir/rick-dev")
PROD_REPO = os.environ.get("PROD_REPO", "/Users/logic/workdir/AI_CODING/rick")
GATE_ENV = {k: v for k, v in os.environ.items() if not k.startswith("RICK_")}

errors = []


def check(label, ok, detail=""):
    print(("PASS" if ok else "FAIL"), label, detail)
    if not ok:
        errors.append(f"{label} {detail}".strip())


def read(p):
    try:
        with open(p, encoding="utf-8") as f:
            return f.read()
    except OSError:
        return None


app = read(os.path.join(DEV_TREE, "web/src/App.tsx"))
if app is None:
    check("R1 App.tsx 存在", False)
else:
    check("R1 侧栏拖动手柄（cursor-col-resize）", "cursor-col-resize" in app)
    check(
        "R2 宽度状态 + localStorage 持久化 + clamp",
        "localStorage" in app and ("clamp" in app or "Math.min" in app and "Math.max" in app),
    )
    check("T1 旧英文已删", "guardians of the context" not in app)
    check("T2 wubba lubba dub dubs 已上墙", "wubba lubba dub dubs" in app)

web_dir = os.path.join(DEV_TREE, "web")
r = subprocess.run(["npm", "run", "typecheck"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 npm typecheck", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")
r = subprocess.run(["npm", "run", "build"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E1 npm build", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")
r = subprocess.run(["go", "build", "./..."], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 go build ./...", r.returncode == 0, r.stderr[-300:] if r.returncode else "")

r = subprocess.run(
    ["git", "-C", PROD_REPO, "status", "--porcelain", "-uno"],
    capture_output=True, text=True, env=GATE_ENV, timeout=60,
)
known_rel = ".rick/jobs/job_50/doing/tasks.json"
dirty = [l for l in r.stdout.splitlines() if l.strip() and l.split()[-1] not in (known_rel,)]
check("P1 生产树零改动", len(dirty) == 0, "; ".join(dirty)[:200] if dirty else "")

print()
if errors:
    print(f"GATE6 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE6 PASS: 侧栏可拖宽（持久化+clamp）+ wubba lubba dub dubs 台词断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
