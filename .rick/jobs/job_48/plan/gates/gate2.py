#!/usr/bin/env python3
"""gate2.py — job_48 第 2 层门禁（Kahn L2：task2 web super 会话类型）

模块级集成测试（human 确认，agent 不得修改）：
  A. go build + go test ./internal/web/（含 super 校验/prep 用例）+ ./internal/handler/ 无回归
  B. 契约：SessionTypeSuper 常量 / ValidateSessionRequest(super) / prepareInteractive 严格加载
  C. spawn 冒烟（若 web 测试覆盖）+ api-contract（若存在）含 super
  D. 生产零触碰

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


def run(cmd, cwd=ROOT, timeout=1200, env=None):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True,
                          timeout=timeout, env=env)


def main():
    # ---- A. Go 构建 + web/handler 测试 ----
    r = run(["go", "build", "./..."])
    if r.returncode != 0:
        ERRORS.append(f"[A] go build 失败：\n{(r.stdout + r.stderr)[-1500:]}")
    else:
        for pkg in ["./internal/web/", "./internal/handler/"]:
            r = run(["go", "test", pkg, "-timeout", "600s",
                     "-skip", "TestReleaseCmd.*"])
            if r.returncode != 0:
                ERRORS.append(f"[A] go test {pkg} 失败：\n{(r.stdout + r.stderr)[-2000:]}")

    # ---- B. 契约断言 ----
    reg = os.path.join(ROOT, "internal/web/registry.go")
    with open(reg, encoding="utf-8") as f:
        s = f.read()
    for kw in ['SessionTypeSuper', 'case SessionTypeSuper']:
        if kw not in s:
            ERRORS.append(f"[B] registry.go 缺 {kw}")
    sess = os.path.join(ROOT, "internal/web/sessions.go")
    with open(sess, encoding="utf-8") as f:
        s2 = f.read()
    if "SessionTypeSuper" not in s2:
        ERRORS.append("[B] sessions.go prepareInteractive 未接 super 分支")
    for kw in ["super_env", "BuildSuperPrompt"]:
        if kw not in s2:
            ERRORS.append(f"[B] sessions.go 缺关键词 {kw}")

    # ---- C. api-contract（存在则须含 super） ----
    contract = os.path.join(ROOT, "internal/api-contract.md")
    if os.path.exists(contract):
        with open(contract, encoding="utf-8") as f:
            c = f.read()
        if "super" not in c:
            ERRORS.append("[C] api-contract.md params 表缺 super 行")

    # ---- D. 生产零触碰 ----
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = None
    for line in wt.stdout.splitlines():
        if line.startswith("worktree ") and "rick-dev" not in line:
            prod = line.split(" ", 1)[1].strip()
            break
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        if r.stdout.strip():
            ERRORS.append(f"[D][RSI] 生产仓库 tracked 文件被改动：\n{r.stdout.strip()[:500]}")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)


if __name__ == "__main__":
    main()
