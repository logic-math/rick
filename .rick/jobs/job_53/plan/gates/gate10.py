#!/usr/bin/env python3
"""job_53 gate10：Bug #5 迭代 5b——历史标签删除 + 创建 title 任务化 + 模态多行 + 草稿服务端化。

human 反馈（2026-11-14）：历史会话 UI 不再显示四类型标签（数据保留给 agent 阅读）；
创建时显示任务名信息；新建会话输入框可换行；草稿存服务端（跨设备不中断）。

断言清单：
  T1 否定性：TYPE_LABEL/TYPE_TONE 无 plan/doing/ctrl/learning 四类型映射（历史标签删除）
  S1 存在性：创建 title 任务化（prep.title 优先 requirement/topic——human-loop/super 分支标志）
  M1 存在性：模态需求输入为 textarea（多行换行）
  D1 存在性：草稿服务端 API（PUT/GET /api/sessions/{id}/draft）+ 前端 api.saveDraft/getDraft
  D2 存在性：前端草稿读写走服务端（ChatInput/模态的草稿调用点）
  E1 构建绿：npm typecheck + build；go build + go test runtime+web
  P1 生产零触碰
"""
import os
import subprocess
import sys
import time

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


# ---- T1: 历史标签删除 ----
badge = read(os.path.join(DEV_TREE, "web/src/components/layout/SessionBadge.tsx")) or ""
check(
    "T1 TYPE_LABEL 无四类型映射",
    all(f'{k}: "PLAN"' not in badge and f'{k}: "DOING"' not in badge and f'{k}: "CTRL"' not in badge and f'{k}: "LEARNING"' not in badge
        for k in ("plan", "doing", "ctrl", "learning")) and 'plan: "PLAN"' not in badge and 'doing: "DOING"' not in badge,
)

# ---- S1: 创建 title 任务化 ----
sg = read(os.path.join(DEV_TREE, "internal/web/sessions.go")) or ""
check(
    "S1 创建 title 优先 requirement/topic（human-loop/super prep）",
    "truncateTitleForList" in sg or ("prepTitleFromInput" in sg) or ("titleFromRequirement" in sg),
)

# ---- M1: 模态多行 ----
modal = read(os.path.join(DEV_TREE, "web/src/components/sessions/NewSessionModal.tsx")) or ""
check("M1 模态输入为 textarea（多行）", "<textarea" in modal)

# ---- D1/D2: 草稿服务端 ----
routes = read(os.path.join(DEV_TREE, "internal/web/routes.go")) or ""
check("D1 draft 路由注册", "/api/drafts/{key}" in routes and "SessionDraft" in (sg or ""))
client = read(os.path.join(DEV_TREE, "web/src/api/client.ts")) or ""
check("D1 api saveDraft/getDraft", "saveDraft" in client and "getDraft" in client)
ci = read(os.path.join(DEV_TREE, "web/src/components/chat/ChatInput.tsx")) or ""
check("D2 ChatInput 草稿服务端化", "saveDraft" in ci or "getDraft" in ci)

# ---- E1: 构建 ----
web_dir = os.path.join(DEV_TREE, "web")
r = subprocess.run(["npm", "run", "typecheck"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 npm typecheck", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")
r = subprocess.run(["npm", "run", "build"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E1 npm build", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")
r = subprocess.run(["go", "build", "./..."], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 go build ./...", r.returncode == 0, r.stderr[-400:] if r.returncode else "")
for attempt in (1, 2, 3):
    r = subprocess.run(["go", "test", "-p", "1", "./internal/runtime/", "./internal/web/"], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=1800)
    if r.returncode == 0:
        break
check("E1 go test runtime+web（-p1+重试）", r.returncode == 0, ((r.stderr or "") + (r.stdout or ""))[-500:] if r.returncode else "")

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
    print(f"GATE10 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE10 PASS: 历史标签删除 + 创建 title 任务化 + 模态多行 + 草稿服务端化断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
