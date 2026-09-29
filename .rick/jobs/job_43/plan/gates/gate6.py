#!/usr/bin/env python3
"""gate6.py — job_43 增量迭代 4 第 6 层门禁（task9 会话信息原文 toggle 修复）。

模块级集成测试（human 确认，agent 不得修改）：
  A. 前端 typecheck + build；Go build 保险
  B. 静态断言：toggle 逻辑存在（promptOpen 有置 false/取反路径）
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

    # ---- B. toggle 断言 ----
    cv = read("web/src/components/chat/ChatView.tsx")
    # promptOpen 存在取反/置 false 路径（toggle 语义）
    if not re.search(r"setPromptOpen\(\s*(false|v\s*=>\s*!v|!\s*promptOpen)", cv):
        ERRORS.append("[task9-KR10a] 未见 promptOpen 收起/取反路径（toggle 未实现）")
    # 按钮事件不再单纯 loadPrompt（应经 toggle 分派）
    if re.search(r'onClick=\{\(\)\s*=>\s*void\s+loadPrompt\(\)\}', cv):
        ERRORS.append("[task9-KR10a] 按钮 onClick 仍直连 loadPrompt（无 toggle 分派）")
    # 加载守卫保持（loading/错误重试）
    if not re.search(r"promptLoading", cv):
        ERRORS.append("[task9-KR10b] promptLoading 守卫缺失")

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
