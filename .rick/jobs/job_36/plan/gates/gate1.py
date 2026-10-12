#!/usr/bin/env python3
"""gate1：第 1 层（task1 rpc 客户端 / task2 handler core 重构 / task3 前端骨架 / task5 registry）。"""
import json, os, subprocess, sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", "..", ".."))
errors = []

def run(cmd, cwd=ROOT, timeout=600):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

r = run(["go", "build", "./..."])
if r.returncode != 0:
    errors.append(f"go build ./... 失败:\n{r.stderr[-2000:]}")

# task1: rpc 协议客户端
r = run(["go", "test", "./internal/runtime/", "-run", "TestRpc", "-timeout", "120s"])
if r.returncode != 0:
    errors.append(f"task1 rpc 测试失败:\n{r.stderr[-2000:]}")
if not os.path.exists(os.path.join(ROOT, "internal/runtime/rpc.go")):
    errors.append("task1: internal/runtime/rpc.go 不存在")

# task2: handler 重构零行为变化（现有测试全绿 + GetRickDir 收敛）
r = run(["go", "test", "./internal/handler/...", "-timeout", "180s"])
if r.returncode != 0:
    errors.append(f"task2 handler 测试失败:\n{r.stderr[-2000:]}")
r = run(["go", "test", "./internal/cmd/", "-timeout", "120s"])
if r.returncode != 0:
    errors.append(f"task2 cmd 测试失败:\n{r.stderr[-2000:]}")
r = run(["bash", "-c", "grep -c 'GetRickDir()' internal/handler/*.go 2>/dev/null | awk -F: '{s+=$2} END {print s+0}'"])
if int(r.stdout.strip() or 0) > 7:
    errors.append(f"task2: handler 内 GetRickDir 调用未收敛（{r.stdout.strip()} 处，应 ≤7 CLI 包装入口）")

# task3: 前端骨架 + embed
for p in ["web/package.json", "web/vite.config.ts", "web/tsconfig.json", "web/index.html",
          "web/embed.go", "web/src/main.tsx", "web/src/styles/theme.css",
          "web/src/components/starfield/StarfieldBackground.tsx", "web/dist/index.html"]:
    if not os.path.exists(os.path.join(ROOT, p)):
        errors.append(f"task3: {p} 不存在")
r = run(["go", "build", "./web/"])
if r.returncode != 0:
    errors.append(f"task3: go build ./web/ 失败:\n{r.stderr[-1500:]}")
mk = open(os.path.join(ROOT, "Makefile")).read() if os.path.exists(os.path.join(ROOT, "Makefile")) else ""
if "web-dist" not in mk:
    errors.append("task3: Makefile 缺 web-dist 目标")
if os.path.exists(os.path.join(ROOT, "web/package.json")):
    r = run(["npm", "run", "build"], cwd=os.path.join(ROOT, "web"), timeout=600)
    if r.returncode != 0:
        errors.append(f"task3: npm run build 失败:\n{(r.stderr or r.stdout)[-2000:]}")

# task5: registry
r = run(["go", "test", "./internal/web/", "-run", "TestRegistry", "-timeout", "120s"])
if r.returncode != 0:
    errors.append(f"task5 registry 测试失败:\n{r.stderr[-2000:]}")
for p in ["internal/web/registry.go", "internal/web/web.go"]:
    if not os.path.exists(os.path.join(ROOT, p)):
        errors.append(f"task5: {p} 不存在")

print(json.dumps({"pass": len(errors) == 0, "errors": errors}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
