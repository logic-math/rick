#!/usr/bin/env python3
"""gate4.py — job_43 增量迭代 2 第 4 层门禁（task7 长会话虚拟化 virtuoso）。

模块级集成测试（human 确认，agent 不得修改）：
  A. 前端 typecheck + build；Go build 保险
  B. 静态断言：virtuoso 依赖 + MessageList 虚拟化关键 API
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
    # ---- A. 构建 ----
    run(["go", "build", "./..."])
    web = os.path.join(ROOT, "web")
    run(["npm", "run", "typecheck"], cwd=web)
    run(["npm", "run", "build"], cwd=web)

    # ---- B. 虚拟化断言 ----
    pkg = read("web/package.json")
    if not re.search(r"react-virtuoso", pkg):
        ERRORS.append("[task7-KR8a] web/package.json 未引入 react-virtuoso")
    ml = read("web/src/components/chat/MessageList.tsx")
    if not re.search(r"from\s+[\"']react-virtuoso[\"']|import\s*\{[^}]*Virtuoso", ml):
        ERRORS.append("[task7-KR8a] MessageList 未 import Virtuoso")
    if not re.search(r"<Virtuoso", ml):
        ERRORS.append("[task7-KR8a] MessageList 未使用 <Virtuoso> 组件")
    if not re.search(r"followOutput", ml):
        ERRORS.append("[task7-KR8b] 未见 followOutput（流式贴底兼容）")
    if not re.search(r"computeItemKey", ml):
        ERRORS.append("[task7-KR8e] 未见 computeItemKey（stable key 状态保持）")
    if not re.search(r"firstItemIndex|atTopStateChange|startReached", ml):
        ERRORS.append("[task7-KR8d] 未见 firstItemIndex/atTopStateChange/startReached（历史前置插入兼容）")

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
