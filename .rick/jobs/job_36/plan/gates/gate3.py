#!/usr/bin/env python3
"""gate3：第 3 层（task9 前端 chat 组件 / task10 前端 monitor+jobs+knowledge / task11 会话 REST+执行桥）。"""
import json, os, subprocess, sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", "..", ".."))
errors = []

def run(cmd, cwd=ROOT, timeout=600):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

r = run(["go", "build", "./..."])
if r.returncode != 0:
    errors.append(f"go build ./... 失败:\n{r.stderr[-2000:]}")

# task9 chat 组件
for p in ["web/src/components/chat/ChatView.tsx", "web/src/components/chat/ToolCallCard.tsx",
          "web/src/components/chat/ChatInput.tsx", "web/src/components/extui/ExtensionUIDialog.tsx"]:
    if not os.path.exists(os.path.join(ROOT, p)):
        errors.append(f"task9: {p} 不存在")

# task10 monitor/jobs/knowledge/common
for p in ["web/src/components/monitor/MonitorView.tsx", "web/src/components/monitor/TaskBoard.tsx",
          "web/src/components/jobs/JobsList.tsx", "web/src/components/knowledge/KnowledgeBrowser.tsx",
          "web/src/components/common/Dialog.tsx"]:
    if not os.path.exists(os.path.join(ROOT, p)):
        errors.append(f"task10: {p} 不存在")

# task11 sessions REST（fake pi 集成测试）
r = run(["go", "test", "./internal/web/", "-run", "TestSession", "-timeout", "300s"])
if r.returncode != 0:
    errors.append(f"task11 sessions 测试失败:\n{r.stderr[-2000:]}")
if not os.path.exists(os.path.join(ROOT, "internal/web/sessions.go")):
    errors.append("task11: internal/web/sessions.go 不存在")

# 前端类型+构建
if os.path.exists(os.path.join(ROOT, "web/package.json")):
    r = run(["npx", "tsc", "--noEmit"], cwd=os.path.join(ROOT, "web"), timeout=300)
    if r.returncode != 0:
        errors.append(f"tsc --noEmit 失败:\n{(r.stderr or r.stdout)[-2000:]}")
    r = run(["npm", "run", "build"], cwd=os.path.join(ROOT, "web"), timeout=600)
    if r.returncode != 0:
        errors.append(f"npm run build 失败:\n{(r.stderr or r.stdout)[-2000:]}")

r = run(["bash", "-c", "pgrep -f '[m]ode rpc' | head -5"])
if r.stdout.strip():
    errors.append(f"残留 pi rpc 进程: {r.stdout.strip()[:200]}")

print(json.dumps({"pass": len(errors) == 0, "errors": errors}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
