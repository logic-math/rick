#!/usr/bin/env python3
"""gate7.py — job_43 增量迭代 5 第 7 层门禁（task10 后端白名单扩展 + task11 前端工具箱）。

模块级集成测试（human 确认，agent 不得修改）：
  A. Go build + test（internal/web + internal/cmd）；前端 typecheck + build
  B. 静态断言：白名单多根扩展 + 全局工具箱入口
  C. 生产零触碰

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


def main():
    # ---- A. 构建（go test 需净化 rick 专属 env：RICK_STATE_DIR 等会穿透 t.Setenv(HOME)
    #      的隔离——测试读到真实 ~/.rick/web.pid 撞上生产单例（实测））----
    import copy
    clean_env = {k: v for k, v in os.environ.items() if not k.startswith("RICK_")}
    run(["go", "build", "./..."])
    r = subprocess.run(["go", "test", "./internal/web/", "./internal/cmd/"],
                       cwd=ROOT, capture_output=True, text=True, timeout=900, env=clean_env)
    if r.returncode != 0:
        ERRORS.append(f"$ go test ./internal/web/ ./internal/cmd/\n{(r.stdout + r.stderr).strip()[-2000:]}")
    web = os.path.join(ROOT, "web")
    run(["npm", "run", "typecheck"], cwd=web)
    run(["npm", "run", "build"], cwd=web)

    # ---- B1. task10：白名单扩展 ----
    routes = read("internal/web/routes.go")
    # 多 workspace 根（registry 全量路径并入 allowedFileRoots）
    if not re.search(r"allowedFileRoots", routes):
        ERRORS.append("[task10-KR11a] routes.go 未见 allowedFileRoots（被重构掉了？）")
    if not re.search(r"(?s)allowedFileRoots.*?workspaces|registry|ws\.Path", routes):
        ERRORS.append("[task10-KR11a] allowedFileRoots 未见全量 workspace 根注入")
    # dev 树额外根（ExtraFileRoots 或等价注入）
    if not re.search(r"ExtraFileRoots|extraFileRoots|extra_file_roots|RICK_DEV_TREE|rick-dev", routes):
        ERRORS.append("[task10-KR11a] 未见 dev worktree 额外根注入机制")
    # 显式根列表语义保持（拒绝逻辑不动的标志：仍拒绝目录/超限）
    if not re.search(r"readWhitelistedFile", routes):
        ERRORS.append("[task10-KR11c] readWhitelistedFile 硬化读取缺失")
    # 测试覆盖三态
    rt = read("internal/web/routes_test.go")
    if not re.search(r"workspace|cross|other", rt):
        ERRORS.append("[task10-KR11c] routes_test.go 未见跨 workspace 测试")
    webgo = ""
    try:
        webgo = read("internal/cmd/web.go")
    except FileNotFoundError:
        pass
    if webgo and not re.search(r"ExtraFileRoots|rick-dev|devTree|DevTree", webgo):
        ERRORS.append("[task10-KR11a] 组合根未见 dev 根注入")

    # ---- B2. task11：全局工具箱 ----
    app = read("web/src/App.tsx")
    # 顶栏工具箱入口（aria-label/标题含 打开文件/工具箱）
    if not re.search(r"打开文件|工具箱", app):
        ERRORS.append("[task11-KR12a] App.tsx 未见工具箱「打开文件」入口")
    # 全局 Provider（App 壳层）
    if not re.search(r"FileReaderProvider", app):
        ERRORS.append("[task11-KR12b] App.tsx 未见全局 FileReaderProvider")
    # 路径输入 Dialog
    if not re.search(r"Dialog", app):
        ERRORS.append("[task11-KR12a] App.tsx 未见路径输入 Dialog")
    # ChatView 不再持有 Provider（已上提）
    cv = read("web/src/components/chat/ChatView.tsx")
    if re.search(r"FileReaderProvider", cv):
        ERRORS.append("[task11-KR12b] ChatView 仍含 FileReaderProvider（应已提升到 App）")
    if not re.search(r"useFileReader|reader\.open|FileReaderPanel", cv) and not re.search(r"useFileReader", cv):
        ERRORS.append("[task11-KR12e] ChatView 未见 reader 使用（链接点击链路可能断）")

    # ---- C. 生产零触碰 ----
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
            ERRORS.append(f"[RSI] 生产仓库 tracked 文件被改动：\n{r.stdout.strip()[:500]}")
    else:
        ERRORS.append("[RSI] 未能定位生产仓库（worktree list 解析失败）")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)


if __name__ == "__main__":
    main()
