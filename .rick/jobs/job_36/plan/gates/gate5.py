#!/usr/bin/env python3
"""gate5：第 5 层（task14 env 基座 + handler/cmd web 命令收口）。"""
import json, os, subprocess, sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "..", "..", ".."))
errors = []

def run(cmd, cwd=ROOT, timeout=600):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)

r = run(["go", "build", "./..."])
if r.returncode != 0:
    errors.append(f"go build ./... 失败:\n{r.stderr[-2000:]}")

for pkg in ["./internal/cmd/", "./internal/env/", "./internal/handler/...", "./internal/web/..."]:
    r = run(["go", "test", pkg, "-timeout", "600s"])
    if r.returncode != 0:
        errors.append(f"{pkg} 测试失败:\n{r.stderr[-2000:]}")

for p in ["internal/env/web.go", "internal/handler/web.go", "internal/cmd/web.go"]:
    if not os.path.exists(os.path.join(ROOT, p)):
        errors.append(f"task14: {p} 不存在")
root_go = open(os.path.join(ROOT, "internal/cmd/root.go")).read()
if "NewWebCmd" not in root_go:
    errors.append("task14: root.go 未注册 NewWebCmd")

r = run(["go", "build", "-o", "bin/rick", "./cmd/rick"])
if r.returncode != 0:
    errors.append(f"构建 bin/rick 失败:\n{r.stderr[-1500:]}")
else:
    r = run([os.path.join(ROOT, "bin/rick"), "web", "--help"])
    out = r.stdout + r.stderr
    if r.returncode != 0 or "--port" not in out or "--listen" not in out:
        errors.append(f"rick web --help 异常:\n{out[-800:]}")
    for sub in ["customize", "reset"]:
        r = run([os.path.join(ROOT, "bin/rick"), "web", sub, "--help"])
        if r.returncode != 0:
            errors.append(f"rick web {sub} 子命令缺失")

print(json.dumps({"pass": len(errors) == 0, "errors": errors}, ensure_ascii=False, indent=1))
sys.exit(0 if not errors else 1)
