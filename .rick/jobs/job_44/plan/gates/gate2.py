#!/usr/bin/env python3
"""gate2.py — job_44 第 2 层门禁（task2 humans 后端 API + task3 humans 前端 + 草稿持久化）

模块级集成测试（human 确认，agent 不得修改）：
  A. Go：go test ./internal/web/...（含 humans tree/file 集成测试与既有回归）
  B. 后端契约：humans.go 存在；routes.go 注册 /humans/tree 与 /humans/file
  C. 前端契约：Humans.tsx 存在；App.tsx 含 humans 路由；humans 浏览组件存在；
     未实现阶段进展/判断卡片（human 明确不要）
  D. 草稿持久化契约：ChatInput.tsx 含 localStorage + rick-web-draft 键 + 发送清除 + 静默降级
  E. 前端 typecheck + build 绿
  F. 生产零触碰

输出 JSON {"pass": bool, "errors": [...]}，exit 0=绿 / 1=红。
"""
import json
import os
import re
import subprocess
import sys


def get_project_root():
    p = os.path.abspath(__file__)
    for _ in range(6):
        p = os.path.dirname(p)
    return p


ROOT = get_project_root()
ERRORS = []


def run(cmd, cwd=ROOT, timeout=900):
    r = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)
    if r.returncode != 0:
        ERRORS.append(f"$ {' '.join(cmd)}\n{(r.stdout + r.stderr).strip()[-2000:]}")
    return r


def read(path):
    with open(os.path.join(ROOT, path), encoding="utf-8") as f:
        return f.read()


def exists(path):
    return os.path.exists(os.path.join(ROOT, path))


def main():
    # ---- A. Go 测试（web 包：humans API + 回归）----
    run(["go", "test", "./internal/web/"])

    # ---- B. 后端契约 ----
    if not exists("internal/web/humans.go"):
        ERRORS.append("[B] internal/web/humans.go 缺失")
    try:
        routes = read("internal/web/routes.go")
        for pat, label in ((r"/humans/tree", "humans tree 路由"),
                           (r"/humans/file", "humans file 路由")):
            if not re.search(pat, routes):
                ERRORS.append(f"[B] routes.go 未注册 {label}")
    except FileNotFoundError:
        ERRORS.append("[B] internal/web/routes.go 缺失")

    # humans handler 必须实现 tree + file 两个 handler
    if exists("internal/web/humans.go"):
        h = read("internal/web/humans.go")
        for pat, label in ((r"handleHumansTree|HumansTree", "tree handler"),
                           (r"handleHumansFile|HumansFile|ReadHumansFile", "file handler")):
            if not re.search(pat, h):
                ERRORS.append(f"[B] humans.go 未见 {label}")

    # ---- C. 前端契约（humans 页面）----
    if not exists("web/src/routes/Humans.tsx"):
        ERRORS.append("[C] web/src/routes/Humans.tsx 缺失")
    try:
        app = read("web/src/App.tsx")
        if "humans" not in app.lower():
            ERRORS.append("[C] App.tsx 未见 humans 路由/导航")
    except FileNotFoundError:
        ERRORS.append("[C] web/src/App.tsx 缺失")
    humans_components = []
    comp_dir = os.path.join(ROOT, "web/src/components/humans")
    if os.path.isdir(comp_dir):
        humans_components = [f for f in os.listdir(comp_dir) if f.endswith((".tsx", ".ts"))]
    if not humans_components:
        ERRORS.append("[C] web/src/components/humans/ 下无浏览组件")
    # human 明确不要：阶段进展条 / 判断点卡片
    joined = ""
    for p in (["web/src/routes/Humans.tsx"] +
              [f"web/src/components/humans/{f}" for f in humans_components]):
        if exists(p):
            joined += read(p)
    for bad, label in ((r"阶段进展|阶段进度|stage.?progress", "阶段进展条"),
                       (r"判断点卡片|judgment.?card", "判断点卡片")):
        if re.search(bad, joined, re.IGNORECASE):
            ERRORS.append(f"[C] humans 页面实现了 human 明确不要的：{label}")

    # ---- D. 草稿持久化契约 ----
    ci_path = "web/src/components/chat/ChatInput.tsx"
    try:
        ci = read(ci_path)
    except FileNotFoundError:
        ci = ""
        ERRORS.append(f"[D] {ci_path} 缺失")
    if ci:
        if "localStorage" not in ci:
            ERRORS.append("[D] ChatInput 未见 localStorage 持久化")
        if not re.search(r"rick-web-draft", ci):
            ERRORS.append("[D] ChatInput 未见草稿键 rick-web-draft")
        if not re.search(r"removeItem", ci):
            ERRORS.append("[D] ChatInput 未见发送后清除草稿（removeItem）")
        if not re.search(r"catch|try\s*\{", ci):
            ERRORS.append("[D] ChatInput 未见 localStorage 不可用时的静默降级（try/catch）")
        if not re.search(r"sessionId|session_id", ci):
            ERRORS.append("[D] ChatInput 未见按会话键控草稿（sessionId）")

    # ---- E. 前端 typecheck + build ----
    web = os.path.join(ROOT, "web")
    run(["npm", "run", "typecheck"], cwd=web)
    run(["npm", "run", "build"], cwd=web)

    # ---- F. 生产零触碰 ----
    wt = subprocess.run(["git", "worktree", "list", "--porcelain"], cwd=ROOT,
                        capture_output=True, text=True)
    prod = None
    for line in wt.stdout.splitlines():
        if line.startswith("worktree ") and "rick-dev" not in line:
            prod = line.split(" ", 1)[1].strip()
            break
    if prod and os.path.isdir(prod):
        r = subprocess.run(["git", "status", "--porcelain", "-uno"], cwd=prod,
                           capture_output=True, text=True)
        if r.stdout.strip():
            ERRORS.append(f"[F][RSI] 生产仓库 tracked 文件被改动：\n{r.stdout.strip()[:500]}")
    else:
        ERRORS.append("[F][RSI] 未能定位生产仓库（worktree list 解析失败）")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)


if __name__ == "__main__":
    main()
