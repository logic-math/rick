#!/usr/bin/env python3
"""gate1.py — job_48 第 1 层门禁（Kahn L1：task1 CLI super 工作区解析切换）

模块级集成测试（human 确认，agent 不得修改）：
  A. go build + go test ./internal/handler/ -run TestSuper（含新增 root 解析用例）
  B. 行为契约：superWsRoot 新语义——config 覆盖优先 / cwd 上溯 .rick / 双失败明确报错
  C. 生产零触碰

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


def run(cmd, cwd=ROOT, timeout=900, env=None):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True,
                          timeout=timeout, env=env)


def main():
    # ---- A. Go 构建与测试 ----
    r = run(["go", "build", "./..."])
    if r.returncode != 0:
        ERRORS.append(f"[A] go build 失败：\n{(r.stdout + r.stderr)[-1500:]}")
    else:
        r = run(["go", "test", "./internal/handler/", "-run", "TestSuper",
                 "-timeout", "300s"])
        if r.returncode != 0:
            ERRORS.append(f"[A] go test handler TestSuper 失败：\n{(r.stdout + r.stderr)[-2000:]}")

    # ---- B. 行为契约（源码静态断言 + 子进程冒烟） ----
    src = os.path.join(ROOT, "internal/handler/super.go")
    with open(src, encoding="utf-8") as f:
        s = f.read()
    # 新语义要点：super_env 路径 + cwd 上溯 + config 覆盖 + 明确报错
    for kw in ["super_env"]:
        if kw not in s:
            ERRORS.append(f"[B] super.go 缺工作区路径关键词 {kw}")
    # 子进程冒烟：在工作区内跑 status 应解析到 .rick/super_env（不存在的 name 应报 not found 而非用旧全局路径）
    import tempfile
    with tempfile.TemporaryDirectory() as td:
        rick = os.path.join(td, "ws", ".rick")
        os.makedirs(os.path.join(rick, "jobs"), exist_ok=True)
        bin_path = os.path.join(td, "rick")
        b = run(["go", "build", "-o", bin_path, "./cmd/rick"])
        if b.returncode != 0:
            ERRORS.append(f"[B] 临时构建失败：{(b.stdout + b.stderr)[-600:]}")
        else:
            # HOME 隔离（屏蔽 ~/.rick/config 干扰）+ cwd=工作区
            env = dict(os.environ, HOME=os.path.join(td, "fakehome"),
                       RICK_STATE_DIR="")
            env.pop("RICK_STATE_DIR", None)
            os.makedirs(env["HOME"], exist_ok=True)
            r = run([bin_path, "super", "status", "--name", "nope", "--json"],
                    cwd=os.path.join(td, "ws"), env=env)
            out = r.stdout + r.stderr
            # 期望：新 root 解析（super_env/nope 不存在）→ not found 类错误，且不引用 ~/.rick/super
            if "super_env" not in out and "not found" not in out and "未初始化" not in out:
                ERRORS.append(f"[B] 工作区内 status 未走 super_env 解析：{out[-300:]}")
            # 旧全局路径不应出现
            if ".rick/super/nope" in out and "super_env" not in out:
                ERRORS.append(f"[B] 仍在用旧全局路径：{out[-200:]}")

    # ---- C. 生产零触碰 ----
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = None
    for line in wt.stdout.splitlines():
        if line.startswith("worktree ") and "rick-dev" not in line:
            prod = line.split(" ", 1)[1].strip()
            break
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        if r.stdout.strip():
            ERRORS.append(f"[C][RSI] 生产仓库 tracked 文件被改动：\n{r.stdout.strip()[:500]}")
    else:
        ERRORS.append("[C][RSI] 未能定位生产仓库")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)


if __name__ == "__main__":
    main()
