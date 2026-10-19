#!/usr/bin/env python3
"""gate3.py — job_48 第 3 层门禁（Kahn L3：task3 前端 + task4 提示词增强/e2e）

模块级集成测试（human 确认，agent 不得修改）：
  A. 前端构建：cd web && npm run build（产物落 dev overlay）
  B. 前端契约：NewSessionModal 含 super 选项与 name 输入；Sessions/ChatView 含 super 徽章
  C. 提示词契约：BuildSuperPrompt 含工作区上下文（super_env 路径/观察手段/干预手段/gen_env 引导）
  D. e2e 冒烟：super session 测试（prompt 严格加载 + CLI resume 互通）
  E. 生产零触碰

输出 JSON {"pass": bool, "errors": [...]}，exit 0=绿 / 1=红。
"""
import json
import os
import subprocess
import sys


def get_project_root():
    p = os.path.abspath(__file__)
    for _ in range(6):
        p = os.path.dirname(p)
    return p


ROOT = get_project_root()
ERRORS = []


def run(cmd, cwd=ROOT, timeout=1200):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)


def read(path):
    with open(os.path.join(ROOT, path), encoding="utf-8") as f:
        return f.read()


def exists(path):
    return os.path.exists(os.path.join(ROOT, path))


def main():
    # ---- A. 前端构建 ----
    r = run(["npm", "run", "build"], cwd=os.path.join(ROOT, "web"), timeout=900)
    if r.returncode != 0:
        ERRORS.append(f"[A] npm run build 失败：\n{(r.stdout + r.stderr)[-1500:]}")

    # ---- B. 前端契约 ----
    modal = "web/src/components/sessions/NewSessionModal.tsx"
    if not exists(modal):
        ERRORS.append("[B] 缺 NewSessionModal.tsx")
    else:
        s = read(modal)
        for kw in ["super"]:
            if kw not in s:
                ERRORS.append(f"[B] NewSessionModal 缺 {kw}")
        # name 输入（super 表单）
        if "name" not in s:
            ERRORS.append("[B] NewSessionModal 缺 name 输入字段")
    for p in ["web/src/routes/Sessions.tsx", "web/src/components/chat/ChatView.tsx"]:
        if exists(p):
            s = read(p)
            if "super" not in s:
                ERRORS.append(f"[B] {p} 缺 super 类型支持")

    # ---- C. 提示词契约（session.go 注入工作区上下文） ----
    if not exists("internal/super/session.go"):
        ERRORS.append("[C] 缺 session.go")
    else:
        s = read("internal/super/session.go")
        for kw in ["super_env", "events.jsonl", "archive.json", "codes", "干预"]:
            if kw not in s:
                ERRORS.append(f"[C] session.go 提示词缺上下文关键词 {kw}")

    # ---- D. e2e 冒烟 ----
    if exists("super/tests/super_session_test.py"):
        r = run([sys.executable, "-m", "unittest", "super.tests.super_session_test"],
                timeout=600)
        if r.returncode != 0:
            ERRORS.append(f"[D] super_session_test 失败：\n{(r.stdout + r.stderr)[-1500:]}")
    else:
        # 允许并入 e2e_test.py 的形态：跑 e2e 的 super session 用例（若命名含 SuperSession）
        r = run([sys.executable, "-m", "unittest", "super.tests.e2e_test",
                 "-k", "super_session"], timeout=600)
        if "no tests" not in (r.stdout + r.stderr) and r.returncode != 0:
            ERRORS.append(f"[D] e2e super session 用例失败：\n{(r.stdout + r.stderr)[-1000:]}")

    # ---- E. 生产零触碰 ----
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = None
    for line in wt.stdout.splitlines():
        if line.startswith("worktree ") and "rick-dev" not in line:
            prod = line.split(" ", 1)[1].strip()
            break
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        if r.stdout.strip():
            ERRORS.append(f"[E][RSI] 生产仓库 tracked 文件被改动：\n{r.stdout.strip()[:500]}")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)


if __name__ == "__main__":
    main()
