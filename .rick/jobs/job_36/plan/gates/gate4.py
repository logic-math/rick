#!/usr/bin/env python3
"""gate4：第 4 层（task12 前端组装/PWA / task13 server 组装+static+watcher）。"""
import json, os, subprocess, sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", "..", ".."))
errors = []

def run(cmd, cwd=ROOT, timeout=600):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

r = run(["go", "build", "./..."])
if r.returncode != 0:
    errors.append(f"go build ./... 失败:\n{r.stderr[-2000:]}")

# task13: web 包全量测试（含此前各 task 合流回归 + server_test 路由面）
r = run(["go", "test", "./internal/web/...", "-timeout", "600s"])
if r.returncode != 0:
    errors.append(f"task13 internal/web 全量测试失败:\n{r.stderr[-3000:]}")
for p in ["internal/web/server.go", "internal/web/routes.go", "internal/web/static.go", "internal/web/watcher.go"]:
    if not os.path.exists(os.path.join(ROOT, p)):
        errors.append(f"task13: {p} 不存在")
routes = open(os.path.join(ROOT, "internal/web/routes.go")).read()
for endpoint in ["/api/config", "/api/workspaces", "/api/sessions", "/api/events",
                 "/api/health", "/api/web/customize", "/api/web/reset"]:
    if endpoint not in routes:
        errors.append(f"task13: routes.go 缺端点 {endpoint}")

# task12: 前端组装
for p in ["web/src/App.tsx", "web/src/routes/Sessions.tsx", "web/src/routes/Jobs.tsx",
          "web/src/routes/Knowledge.tsx", "web/src/routes/Settings.tsx",
          "web/src/components/sessions/NewSessionModal.tsx"]:
    if not os.path.exists(os.path.join(ROOT, p)):
        errors.append(f"task12: {p} 不存在")

# 前端构建 + PWA 产物
if os.path.exists(os.path.join(ROOT, "web/package.json")):
    r = run(["npx", "tsc", "--noEmit"], cwd=os.path.join(ROOT, "web"), timeout=300)
    if r.returncode != 0:
        errors.append(f"tsc --noEmit 失败:\n{(r.stderr or r.stdout)[-2000:]}")
    r = run(["npm", "run", "build"], cwd=os.path.join(ROOT, "web"), timeout=600)
    if r.returncode != 0:
        errors.append(f"npm run build 失败:\n{(r.stderr or r.stdout)[-2000:]}")
    else:
        dist = os.path.join(ROOT, "web/dist")
        has_manifest = any(("manifest" in f) for f in os.listdir(dist)) if os.path.isdir(dist) else False
        if not has_manifest:
            errors.append("task12: dist 无 manifest（PWA 未配置）")

r = run(["bash", "-c", "pgrep -f '[m]ode rpc' | head -5"])
if r.stdout.strip():
    errors.append(f"残留 pi rpc 进程: {r.stdout.strip()[:200]}")

print(json.dumps({"pass": len(errors) == 0, "errors": errors}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
