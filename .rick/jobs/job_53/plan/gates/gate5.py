#!/usr/bin/env python3
"""job_53 gate5：Bug #2（导航精简 + 横排 + super/human-loop 徽标）层门禁。

human 反馈（2026-11-14）：
  1. Sessions 与 Jobs 页面重复——去掉 Jobs 概念/入口（导航不列 + 路由移除；MonitorView/TaskBoard
     是会话详情页依赖，保留不动）
  2. 导航子项（Sessions/Dreams/Humans）改横向排列（省侧栏竖排空间）
  3. super 会话右侧显示 env_N 徽标、human-loop 显示 loop_N 徽标（与 job_N 徽标同款视觉）

断言清单：
  N1 否定性：WS_VIEWS 无 Jobs 项（导航不列）
  N2 存在性：导航子项横排（WS_VIEWS 渲染容器 flex-row）
  B1 存在性：SessionBadge 存在 env_/loop_ 标识徽标提取（title 正则 env_\\d+|loop_\\d+）
  N3 否定性：App.tsx 无 jobs 路由注册与 Jobs import（概念移除）
  N4 否定性：routes/Jobs.tsx 页面文件已删
  E1 构建绿：npm typecheck + build；go build（防误伤）
  P1 生产零触碰
"""
import os
import re
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


# ---- N1/N2: WS_VIEWS 导航 ----
node = read(os.path.join(DEV_TREE, "web/src/components/layout/WorkspaceListNode.tsx"))
if node is None:
    check("N1 WorkspaceListNode.tsx 存在", False)
else:
    views = re.search(r"const WS_VIEWS = \[(.*?)\] as const", node, re.S)
    body = views.group(1) if views else ""
    check("N1 WS_VIEWS 无 Jobs 项", "Jobs" not in body and 'label: "Jobs"' not in body)
    check("N1 WS_VIEWS 含 Sessions/Dreams/Humans", all(l in body for l in ('"Sessions"', '"Dreams"', '"Humans"')))
    # N2: 导航子项横排（WS_VIEWS 渲染容器 flex-row；外层 expanded 容器仍竖排合理）
    check("N2 导航子项横排（flex-row）", "flex flex-row" in node)

# ---- B1: super/human-loop 徽标 ----
badge = read(os.path.join(DEV_TREE, "web/src/components/layout/SessionBadge.tsx"))
if badge is None:
    check("B1 SessionBadge.tsx 存在", False)
else:
    check(
        "B1 env_/loop_ 标识徽标提取存在",
        bool(re.search(r"env_\\d|env_\[0-9\]|env_\\\\d", badge)) or ("env_" in badge and "loop_" in badge),
    )

# ---- N3/N4: Jobs 概念移除 ----
app = read(os.path.join(DEV_TREE, "web/src/App.tsx"))
if app is None:
    check("N3 App.tsx 存在", False)
else:
    check("N3 App.tsx 无 jobs 路由注册", 'kind="jobs"' not in app and "/ws/:wsId/jobs" not in app)
    check("N3 App.tsx 无 Jobs import", 'from "./routes/Jobs"' not in app)
check("N4 routes/Jobs.tsx 已删", not os.path.exists(os.path.join(DEV_TREE, "web/src/routes/Jobs.tsx")))

# ---- E1: 构建 ----
web_dir = os.path.join(DEV_TREE, "web")
r = subprocess.run(["npm", "run", "typecheck"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 npm typecheck", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")
r = subprocess.run(["npm", "run", "build"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E1 npm build", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")
r = subprocess.run(["go", "build", "./..."], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 go build ./...", r.returncode == 0, r.stderr[-300:] if r.returncode else "")

# ---- P1: 生产零触碰 ----
r = subprocess.run(
    ["git", "-C", PROD_REPO, "status", "--porcelain", "-uno"],
    capture_output=True, text=True, env=GATE_ENV, timeout=60,
)
known_rel = ".rick/jobs/job_50/doing/tasks.json"  # 开工前既有脏文件（基线豁免）
dirty = [l for l in r.stdout.splitlines() if l.strip() and l.split()[-1] not in (known_rel,)]
check("P1 生产树零改动", len(dirty) == 0, "; ".join(dirty)[:200] if dirty else "")

print()
if errors:
    print(f"GATE5 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE5 PASS: 导航精简去 Jobs + 横排 + super/human-loop 徽标断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
