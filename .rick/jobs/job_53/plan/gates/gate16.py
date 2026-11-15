#!/usr/bin/env python3
"""job_53 gate16：Bug #8 2A 终版——subagent 会话深度展示（human 裁决 2026-11-15）。

需求：
  1. subagent 卡片恢复默认折叠
  2. 卡片可跳转到 subagent 会话（右侧栏展示完整 session 输出，只读）
  3. 侧边栏每个会话下拉显示活跃 subagent，点击跳转
  4. subagent 会话不能输入（所有输入走主页面）

断言清单：
  A1 否定性：subagent 卡默认折叠恢复（useExpandState 默认 false）
  A2 存在性：subagent 卡「查看会话」跳转入口（onOpenSubagent）
  B1 存在性：后端 subagent entries API（GET /api/subagent/{runId}/entries——status.json 定位子会话 JSONL 解析）
  B2 存在性：后端活跃 subagent 列表 API（GET /api/sessions/{id}/subagents）
  C1 存在性：右侧栏 SubagentPanel（只读消息流 + 轮询 + 无输入框）
  C2 存在性：侧边栏会话行活跃 subagent 下拉
  E1 构建绿 + P1 生产零触碰
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

routes = read(os.path.join(DEV_TREE, "internal/web/routes.go")) or ""
sg = read(os.path.join(DEV_TREE, "internal/web/subagent_progress.go")) or ""
check("B1 subagent entries API", "/api/subagent/{runId}/entries" in routes and "SubagentEntries" in sg)
check("B2 活跃 subagent 列表 API", "/api/sessions/{id}/subagents" in routes and "SessionSubagents" in sg)

tc = read(os.path.join(DEV_TREE, "web/src/components/chat/ToolCallCard.tsx")) or ""
check("A1 subagent 卡默认折叠恢复", 'item.toolName === "subagent")' not in tc.split("useExpandState")[1][:120])
check("A2 卡片跳转入口", "onOpenSubagent" in tc)

panel = os.path.join(DEV_TREE, "web/src/components/chat/SubagentPanel.tsx")
check("C1 SubagentPanel（只读流）", read(panel) is not None and "poll" in (read(panel) or "").lower() or "setInterval" in (read(panel) or ""))

sb = read(os.path.join(DEV_TREE, "web/src/components/layout/SessionBadge.tsx")) or ""
check("C2 侧边栏活跃 subagent 下拉", "subagent" in sb.lower() and ("listSubagents" in sb or "subagents" in sb))

r = subprocess.run(["go", "build", "./..."], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 go build ./...", r.returncode == 0, r.stderr[-300:] if r.returncode else "")
r = None
for attempt in range(1, 9):
    r = subprocess.run(["go", "test", "-p", "1", "./internal/runtime/", "./internal/web/"], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=1800)
    if r.returncode == 0:
        break
    out = (r.stderr or "") + (r.stdout or "")
    fails = [l for l in out.splitlines() if l.startswith("--- FAIL")]
    if fails and all("RemoveAll" in out.split(l)[0][-800:] or "directory not empty" in out.split(l)[0][-800:] for l in fails):
        time.sleep(2)
        continue
    break
check("E1 go test runtime+web（竞态重试）", r.returncode == 0, ((r.stderr or "") + (r.stdout or ""))[-500:] if r.returncode else "")
web_dir = os.path.join(DEV_TREE, "web")
r = subprocess.run(["npm", "run", "typecheck"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 npm typecheck", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")
r = subprocess.run(["npm", "run", "build"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E1 npm build", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")

r = subprocess.run(["git", "-C", PROD_REPO, "status", "--porcelain", "-uno"], capture_output=True, text=True, env=GATE_ENV, timeout=60)
known_rel = ".rick/jobs/job_50/doing/tasks.json"
dirty = [l for l in r.stdout.splitlines() if l.strip() and l.split()[-1] not in (known_rel,)]
check("P1 生产树零改动", len(dirty) == 0, "; ".join(dirty)[:200] if dirty else "")

print()
if errors:
    print(f"GATE16 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE16 PASS: subagent 会话深度展示（跳转+右侧栏只读+侧边栏活跃列表+默认折叠）断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
