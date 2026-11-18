#!/usr/bin/env python3
"""job_53 gate7：Bug #4（所有会话类型可重命名会话名）层门禁。

human 反馈（2026-11-14）：所有会话类型都应能重命名任务名（避免会话管理混乱）。
现状仅 job 类会话（params.job 存在）有「任务名」重命名入口。
方案：会话级别名 store（SessionNameStore，键=会话 id）+ PUT /api/sessions/{id}/name
（不依赖 job 概念，与 Bug #2 去 Jobs 方向一致）+ 前端会话设置「会话名」全类型入口。

断言清单：
  S1 存在性：sessionnames.go（SessionNameStore）存在
  S2 存在性：投影带会话名（sessionNames.Get）+ rename 路由注册（PUT /api/sessions/{id}/name）
  F1 存在性：前端 renameSession API + 会话设置会话名块不再 job gating
  F2 存在性：SessionBadge 标题优先 session.name
  E1 构建绿：npm typecheck + build；go build + go test runtime+web（全量）
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


# ---- S1: SessionNameStore ----
sn = read(os.path.join(DEV_TREE, "internal/web/sessionnames.go"))
check("S1 sessionnames.go 存在", sn is not None and "SessionNameStore" in (sn or ""))

# ---- S2: 投影 + 路由 ----
sessions = read(os.path.join(DEV_TREE, "internal/web/sessions.go"))
routes = read(os.path.join(DEV_TREE, "internal/web/routes.go"))
check(
    "S2 投影带会话名 + rename handler",
    "sessionNames" in (sessions or "") and "SessionRename" in (sessions or ""),
)
check("S2 rename 路由注册", 'PUT /api/sessions/{id}/name' in (routes or ""))

# ---- F1: 前端全类型入口 ----
client = read(os.path.join(DEV_TREE, "web/src/api/client.ts"))
dialog = read(os.path.join(DEV_TREE, "web/src/components/sessions/SessionSettingsDialog.tsx"))
check("F1 api.renameSession 存在", "renameSession" in (client or ""))
check(
    "F1 会话名块不再 job gating（所有类型可重命名）",
    dialog is not None and "会话名" in dialog and "{job && (" not in (dialog or "").split("会话名")[0][-2000:],
)

# ---- F2: Badge 标题优先 session.name ----
badge = read(os.path.join(DEV_TREE, "web/src/components/layout/SessionBadge.tsx"))
check(
    "F2 SessionBadge 标题优先 session.name",
    badge is not None and "session.name" in badge,
)

# ---- E1: 构建 ----
r = subprocess.run(["go", "build", "./..."], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 go build ./...", r.returncode == 0, r.stderr[-300:] if r.returncode else "")
# E1 go test：已知 TempDir 清理竞态（job_50 backlog：spawn 真实 worker 的测试
# 全量负载下 worker 异步终止滞后于 t.TempDir 清理，不同测试轮换挂、单次挂概率
# ~1/3；单独跑/重跑稳定绿；debug/bug1 对照验证非回归）。真回归重跑仍挂，竞态
# 单次挂重跑绿——失败重跑一次保断言强度。
for attempt in (1, 2):
    r = subprocess.run(
        ["go", "test", "./internal/runtime/", "./internal/web/"],
        cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=900,
    )
    if r.returncode == 0:
        break
check("E1 go test runtime+web（含竞态重试）", r.returncode == 0, ((r.stderr or "") + (r.stdout or ""))[-600:] if r.returncode else "")
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
    print(f"GATE7 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE7 PASS: 所有会话类型可重命名会话名（store+API+全类型入口+Badge 优先级）断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
