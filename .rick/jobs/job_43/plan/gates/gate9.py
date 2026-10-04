#!/usr/bin/env python3
"""gate9.py — job_43 增量迭代 6 第 9 层门禁（task13 文件读取去白名单）。

模块级集成测试（human 确认，agent 不得修改）：
  A. Go build + test（internal/web + internal/cmd，净化 RICK_* env——gate7 教训）
  B. 静态断言：根目录白名单判定移除 + 基本防御保留
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


def read(path):
    with open(os.path.join(ROOT, path), encoding="utf-8") as f:
        return f.read()


def main():
    # ---- A. 构建（净化 RICK_* env）----
    clean_env = {k: v for k, v in os.environ.items() if not k.startswith("RICK_")}
    for cmd in (["go", "build", "./..."],
                ["go", "test", "./internal/web/", "./internal/cmd/"]):
        r = subprocess.run(cmd, cwd=ROOT, capture_output=True, text=True,
                           timeout=900, env=clean_env)
        if r.returncode != 0:
            ERRORS.append(f"$ {' '.join(cmd)}\n{(r.stdout + r.stderr).strip()[-2000:]}")

    # ---- B. 白名单移除断言 ----
    routes = read("internal/web/routes.go")
    # 根目录判定函数已移除
    if re.search(r"func (deps Deps\.)?allowedFileRoots|func workspaceRelativeFile", routes):
        ERRORS.append("[task13-KR14a] allowedFileRoots/workspaceRelativeFile 仍存在（根判定未移除）")
    # ExtraFileRoots 注入已删（routes.go 与 web.go）
    webgo = read("internal/cmd/web.go")
    if re.search(r"ExtraFileRoots", routes) or re.search(r"ExtraFileRoots|resolveDevTreeRoots", webgo):
        ERRORS.append("[task13-KR14c] ExtraFileRoots/resolveDevTreeRoots 残留（应随白名单删除）")
    # 基本防御保留：目录拒绝 + 大小上限
    if not (re.search(r"IsDir|directory", routes) and re.search(r"maxBytes|MaxBytes|大小|size", routes)):
        ERRORS.append("[task13-KR14b] 目录/大小基本防御缺失")
    # 任意绝对路径测试存在
    wtest = read("internal/web/workspacefile_test.go") + read("internal/web/routes_test.go")
    if not re.search(r"/etc/|/proc/|hostname|绝对|arbitrary|anyPath", wtest):
        ERRORS.append("[task13-KR14d] 测试未见任意绝对路径用例（/etc/ 类）")

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
