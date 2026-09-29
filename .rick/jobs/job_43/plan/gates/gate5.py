#!/usr/bin/env python3
"""gate5.py — job_43 增量迭代 3 第 5 层门禁（task8 贴底间距修复）。

模块级集成测试（human 确认，agent 不得修改）：
  A. 前端 typecheck + build；Go build 保险
  B. 静态断言：Virtuoso Footer spacer 存在（恢复旧 py-3 底部间距）
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

    # ---- B. Footer spacer 断言 ----
    ml = read("web/src/components/chat/MessageList.tsx")
    # Virtuoso Footer/footer prop 存在
    if not re.search(r"Footer", ml):
        ERRORS.append("[task8-KR9a] MessageList 未见 Footer 组件（贴底 spacer）")
    # spacer 高度：12px（h-3）或等价
    if not re.search(r"h-3\b|height:\s*12|height:\s*[\"']12px[\"']", ml):
        ERRORS.append("[task8-KR9a] 未见 ~12px spacer 高度（h-3 / height:12）")
    # Footer 引用稳定（useMemo/模块级，防 P0 同款重挂）
    if re.search(r"Footer", ml) and not re.search(r"useMemo|const .*Footer\s*=|function .*Footer", ml):
        ERRORS.append("[task8-KR9b] Footer 引用未见稳定化（useMemo/模块级组件）")

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
