#!/usr/bin/env python3
"""gate2.py — job_43 第 2 层门禁（task4 移动端性能）。

模块级集成测试（human 确认，agent 不得修改）：
  A. 前端：typecheck + build；Go build 保险
  B. 静态断言：移动端静态星空+开关 / dpr 1.5 / 去 blur / 流式节流 / Settings 设置项
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


def assert_file_contains(path, patterns, desc):
    try:
        content = read(path)
    except FileNotFoundError:
        ERRORS.append(f"[{desc}] 文件缺失：{path}")
        return
    for pat, label in patterns:
        if not re.search(pat, content):
            ERRORS.append(f"[{desc}] {path} 未匹配 {label}（pattern: {pat}）")


def main():
    # ---- A. 构建 ----
    run(["go", "build", "./..."])
    web = os.path.join(ROOT, "web")
    run(["npm", "run", "typecheck"], cwd=web)
    run(["npm", "run", "build"], cwd=web)

    # ---- B. 静态断言 ----
    # task4-KR4a：StarfieldBackground 静态判定 + dpr 1.5
    assert_file_contains("web/src/components/starfield/StarfieldBackground.tsx", [
        (r"matchMedia|maxWidth|768|innerWidth", "窄屏判定"),
        (r"[Ss]tatic", "静态渲染路径（复用 reduced-motion 静态帧）"),
        (r"1\.5", "dpr 封顶 1.5"),
        (r"localStorage", "设置项读取"),
    ], "task4-KR4a")
    # task4-KR4b：App.tsx / ChatView.tsx 移动端去 blur（md: 前缀恢复桌面）
    for path in ("web/src/App.tsx", "web/src/components/chat/ChatView.tsx"):
        assert_file_contains(path, [
            (r"md:[^\"']*backdrop-blur|md:bg-[^\"']*/(80|70|60|90)", "移动端不透明底 + md: 恢复毛玻璃"),
        ], "task4-KR4b")
    # task4-KR4c：events.ts 窄屏节流
    assert_file_contains("web/src/stores/events.ts", [
        (r"1[0-2][05]|100|110|120", "窄屏 flush 节流常量（100-120ms）"),
        (r"matchMedia|maxWidth|768|innerWidth", "窄屏判定"),
    ], "task4-KR4c")
    # task4-KR4d：Settings 星空设置项
    assert_file_contains("web/src/routes/Settings.tsx", [
        (r"[Ss]tarfield|星空", "星空模式设置项 UI"),
        (r"localStorage", "localStorage 持久化"),
    ], "task4-KR4d")

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
