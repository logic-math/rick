#!/usr/bin/env python3
"""job_53 gate9：Bug #5 大删除（前端层）——四类型 UI 移除 + doing 监控视图死代码清理。

human 裁决（2026-11-14）：历史会话数据保留可见（TYPE_LABEL 保留四类型标签用于历史显示）；
TaskBoard/MonitorView/jobs store 连带删除（doing 死代码）。

断言清单：
  N1 否定性：NewSessionModal 无 plan/doing/ctrl/learning 四类型选项（保留 easy/human-loop/dream/super）
  N2 否定性：MonitorView/TaskBoard 组件文件已删 + SessionPage 无监控视图分支（isMonitorType）
  N3 否定性：jobs store（stores/jobs.ts）已删 + main.tsx 无 wireJobs
  P1 存在性：TYPE_LABEL/TYPE_TONE 保留四类型映射（历史会话显示）
  E1 构建绿：npm typecheck + build
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


# ---- N1: NewSessionModal ----
modal = read(os.path.join(DEV_TREE, "web/src/components/sessions/NewSessionModal.tsx")) or ""
for typ in ("plan", "doing", "ctrl", "learning"):
    if f'type: "{typ}",' in modal:
        check(f"N1 模态无 {typ} 选项", False)
check(
    "N1 模态保留 easy/human-loop/dream/super",
    all(f'type: "{t}",' in modal for t in ("easy", "human-loop", "dream", "super")),
)

# ---- N2: 监控视图死代码 ----
check("N2 MonitorView 已删", not os.path.exists(os.path.join(DEV_TREE, "web/src/components/monitor/MonitorView.tsx")))
check("N2 TaskBoard 已删", not os.path.exists(os.path.join(DEV_TREE, "web/src/components/monitor/TaskBoard.tsx")))
sp = read(os.path.join(DEV_TREE, "web/src/routes/SessionPage.tsx")) or ""
check("N2 SessionPage 无监控分支（isMonitorType）", "isMonitorType" not in sp and "MonitorView" not in sp)

# ---- N3: jobs store ----
check("N3 jobs store 已删", not os.path.exists(os.path.join(DEV_TREE, "web/src/stores/jobs.ts")))
main = read(os.path.join(DEV_TREE, "web/src/main.tsx")) or ""
check("N3 main.tsx 无 wireJobs", "wireJobs" not in main)

# ---- P1: TYPE_LABEL 历史显示保留 ----
badge = read(os.path.join(DEV_TREE, "web/src/components/layout/SessionBadge.tsx")) or ""
check(
    "P1 TYPE_LABEL/TONE 保留四类型（历史显示）",
    all(t in badge for t in ('plan: "PLAN"', 'doing: "DOING"', 'ctrl: "CTRL"', 'learning: "LEARNING"')),
)

# ---- E1: 构建 ----
web_dir = os.path.join(DEV_TREE, "web")
r = subprocess.run(["npm", "run", "typecheck"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 npm typecheck", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")
r = subprocess.run(["npm", "run", "build"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E1 npm build", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")

# ---- P1: 生产零触碰 ----
r = subprocess.run(
    ["git", "-C", PROD_REPO, "status", "--porcelain", "-uno"],
    capture_output=True, text=True, env=GATE_ENV, timeout=60,
)
known_rel = ".rick/jobs/job_50/doing/tasks.json"
dirty = [l for l in r.stdout.splitlines() if l.strip() and l.split()[-1] not in (known_rel,)]
check("P1 生产树零改动", len(dirty) == 0, "; ".join(dirty)[:200] if dirty else "")

print()
if errors:
    print(f"GATE9 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE9 PASS: 前端四类型移除 + 监控视图/jobs store 死代码清理断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
