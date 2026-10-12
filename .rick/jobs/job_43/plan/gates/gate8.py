#!/usr/bin/env python3
"""gate8.py — job_43 增量迭代 5b 第 8 层门禁（task12 工具箱图标语义化）。

模块级集成测试（human 确认，agent 不得修改）：
  A. 前端 typecheck + build
  B. 静态断言：路径语义图标存在、扳手图标移除
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
    web = os.path.join(ROOT, "web")
    run(["npm", "run", "typecheck"], cwd=web)
    run(["npm", "run", "build"], cwd=web)

    # ---- B. 图标断言 ----
    app = read("web/src/App.tsx")
    # 路径语义图标存在（PathIcon/OpenPath 命名或「路径」注释/aria 标识）
    if not re.search(r"[Pp]athIcon|OpenPath|打开文件（输入路径）|路径", app):
        ERRORS.append("[task12-KR13a] App.tsx 未见路径语义图标标识")
    # 顶栏入口与 Dialog 逻辑保持（task11 交付不回归）
    if not re.search(r"工具箱|打开文件", app):
        ERRORS.append("[task12-KR13c] 工具箱入口文案缺失（task11 回归）")
    if not re.search(r"FileReaderProvider", app):
        ERRORS.append("[task12-KR13c] 全局 FileReaderProvider 缺失（task11 回归）")
    # 扳手语义移除（wrench/扳手 注释与 aria）
    if re.search(r"wrench|扳手", app):
        ERRORS.append("[task12-KR13a] 扳手图标语义仍残留（应已替换为路径图标）")

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
