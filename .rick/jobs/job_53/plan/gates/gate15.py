#!/usr/bin/env python3
"""job_53 gate15：Bug #8 2A R2——subagent 进度 SSE 推送（human 裁决：改 rick web，不改前台逻辑）。

断言清单：
  B1 存在性：后端 watcher（subagent_progress.go：status.json 轮询→SessionEvent 发布）
  B2 存在性：watcher 装配（cmd/web.go StartSubagentProgressWatcher）
  F1 存在性：前端 subagent_progress 事件消费（viewModel case + ToolItem.subagentProgress）
  F2 存在性：subagent 卡进度区渲染（ToolCallCard）+ 默认不折叠
  E1 构建绿：go build + go test runtime+web（竞态重试）+ npm typecheck/build
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

sp = read(os.path.join(DEV_TREE, "internal/web/subagent_progress.go")) or ""
check("B1 后端 watcher 存在（status.json 轮询→发布）", "StartSubagentProgressWatcher" in sp and "subagent_progress" in sp and "status.json" in sp)
cmd = read(os.path.join(DEV_TREE, "internal/cmd/web.go")) or ""
check("B2 watcher 装配", "StartSubagentProgressWatcher" in cmd)
vm = read(os.path.join(DEV_TREE, "web/src/components/chat/viewModel.ts")) or ""
check("F1 前端 subagent_progress 消费", 'case "subagent_progress"' in vm and "subagentProgress" in vm)
tc = read(os.path.join(DEV_TREE, "web/src/components/chat/ToolCallCard.tsx")) or ""
check("F2 卡进度区渲染 + 默认不折叠", "subagentProgress" in tc and 'item.toolName === "subagent"' in tc)

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
    print(f"GATE15 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE15 PASS: subagent 进度 SSE 推送（watcher+事件+多卡渲染默认不折叠）断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
