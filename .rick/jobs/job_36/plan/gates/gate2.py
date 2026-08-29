#!/usr/bin/env python3
"""gate2：第 2 层（task4 supervisor / task6 jobs 数据层 / task7 前端 api+stores / task8 SSE+auth）。"""
import json, os, subprocess, sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", "..", ".."))
errors = []

def run(cmd, cwd=ROOT, timeout=600):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

r = run(["go", "build", "./..."])
if r.returncode != 0:
    errors.append(f"go build ./... 失败:\n{r.stderr[-2000:]}")

# task4 supervisor
r = run(["go", "test", "./internal/runtime/", "-run", "TestSupervisor", "-timeout", "180s"])
if r.returncode != 0:
    errors.append(f"task4 supervisor 测试失败:\n{r.stderr[-2000:]}")
if not os.path.exists(os.path.join(ROOT, "internal/runtime/supervisor.go")):
    errors.append("task4: internal/runtime/supervisor.go 不存在")

# task6 jobs 数据层
r = run(["go", "test", "./internal/web/", "-run", "TestJobs", "-timeout", "120s"])
if r.returncode != 0:
    errors.append(f"task6 jobs 测试失败:\n{r.stderr[-2000:]}")
if not os.path.exists(os.path.join(ROOT, "internal/web/jobs.go")):
    errors.append("task6: internal/web/jobs.go 不存在")

# task7 前端 api/stores
for p in ["web/src/api/client.ts", "web/src/api/sse.ts", "web/src/types.ts",
          "web/src/stores/sessions.ts", "web/src/stores/events.ts"]:
    if not os.path.exists(os.path.join(ROOT, p)):
        errors.append(f"task7: {p} 不存在")

# task8 SSE + auth
r = run(["go", "test", "./internal/web/", "-run", "TestSSE", "-timeout", "120s"])
if r.returncode != 0:
    errors.append(f"task8 SSE 测试失败:\n{r.stderr[-2000:]}")
r = run(["go", "test", "./internal/web/", "-run", "TestAuth", "-timeout", "120s"])
if r.returncode != 0:
    errors.append(f"task8 auth 测试失败:\n{r.stderr[-2000:]}")
for p in ["internal/web/sse.go", "internal/web/auth.go"]:
    if not os.path.exists(os.path.join(ROOT, p)):
        errors.append(f"task8: {p} 不存在")

# 前端类型+构建（task7 产出）
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
