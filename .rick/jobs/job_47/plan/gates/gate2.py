#!/usr/bin/env python3
"""gate2.py — job_47 第 2 层门禁（Kahn L2：task5 agents pi 池 + task6 prompts + task7 gates 级联门 + task11 Go 命令面）

模块级集成测试（human 确认，agent 不得修改）：
  A. agents 单测全绿（mock_pi 六场景：usage 聚合/兜底链/坏 JSON 优雅降级/超时 kill）
  B. prompts 单测全绿 + 四模板文件契约（adjust 含 proposals 软硬分界、builder 含 experiment_plan、
     coding 含 diff_path+以框架为准、debug 含 fix_report）+ 模板覆盖链
  C. gates 单测全绿（级联短路/中位数/软门禁容差带/门先证红 100% 拒识/金样例零误杀/env 定义驱动）
  D. Go：go build + go test ./internal/{super,cmd,handler}/...（命令面 start/stop/status/event +
     BuildSuperPrompt + config 默认值与 py DEFAULTS 一致）
  E. CLI 冒烟：rick super --help 列五子命令
  F. 生产零触碰

输出 JSON {"pass": bool, "errors": [...]}，exit 0=绿 / 1=红。
"""
import json
import os
import subprocess
import sys
import tempfile


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


def exists(path):
    return os.path.exists(os.path.join(ROOT, path))


def read(path):
    with open(os.path.join(ROOT, path), encoding="utf-8") as f:
        return f.read()


def main():
    # ---- A/B/C. py 单测（L2 全部） ----
    r = run([sys.executable, "-m", "unittest", "discover", "-s", "super/tests",
             "-p", "test_*.py", "-v"])
    if r.returncode != 0:
        ERRORS.append(f"[A] py 单测失败：\n{(r.stdout + r.stderr)[-2000:]}")

    # ---- B'. 四模板内容契约 ----
    tpl = {"adjust": ["proposals", "软动作", "硬动作", "正交"],
           "builder": ["experiment_plan", "parents", "idea"],
           "coding": ["diff_path", "self_check", "框架"],
           "debug": ["fix_report", "root_cause", "attempt"]}
    for role, kws in tpl.items():
        p = f"super/prompts/{role}.md"
        if not exists(p):
            ERRORS.append(f"[B] 缺模板 {p}")
            continue
        s = read(p)
        for kw in kws:
            check_kw = kw
            if check_kw not in s:
                ERRORS.append(f"[B] {role}.md 缺关键词 {kw}")
    # coding 模板必须含「作弊」警示（exploit 防线）
    if exists("super/prompts/coding.md"):
        s = read("super/prompts/coding.md")
        if "作弊" not in s and "预存" not in s:
            ERRORS.append("[B] coding.md 缺防作弊警示（预存结果/篡改计时）")

    # ---- C'. 门禁框架零硬编码红线（L3-Q2） ----
    if exists("super/superopt/gates.py"):
        s = read("super/superopt/gates.py")
        # 不得出现写死的 matmul/矩阵细节或绝对阈值常量
        for bad in ["matmul", "numpy", "1e-3", "0.05"]:
            # 0.05/1e-3 只允许作为 env.json 缺省 fallback 的默认值出现一次
            if bad in s:
                ERRORS.append(f"[C][L3-Q2] gates.py 疑似硬编码 env 范畴内容：{bad}")
        for need in ["preverify_red", "stage1", "stage2", "stage3", "soft"]:
            if need not in s:
                ERRORS.append(f"[C] gates.py 缺机制关键词 {need}")

    # ---- D. Go 构建 + 命令面测试 ----
    r = run(["go", "build", "./..."])
    if r.returncode != 0:
        ERRORS.append(f"[D] go build 失败：\n{(r.stdout + r.stderr)[-1500:]}")
    else:
        for pkg in ["./internal/super/...", "./internal/cmd/...", "./internal/handler/..."]:
            r = run(["go", "test", pkg, "-timeout", "180s"])
            if r.returncode != 0:
                ERRORS.append(f"[D] go test {pkg} 失败：\n{(r.stdout + r.stderr)[-1500:]}")

    # ---- E. CLI 冒烟 ----
    with tempfile.TemporaryDirectory() as td:
        bin_path = os.path.join(td, "rick")
        r = run(["go", "build", "-o", bin_path, "./cmd/rick"])
        if r.returncode != 0:
            ERRORS.append(f"[E] 构建临时 rick 失败：\n{(r.stdout + r.stderr)[-800:]}")
        else:
            r = run([bin_path, "super", "--help"])
            for sub in ["session", "start", "stop", "status", "event"]:
                if sub not in r.stdout:
                    ERRORS.append(f"[E] rick super --help 缺子命令 {sub}")
            if r.returncode != 0:
                ERRORS.append(f"[E] rick super --help 退出码 {r.returncode}")

    # ---- F. 生产零触碰 ----
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = None
    for line in wt.stdout.splitlines():
        if line.startswith("worktree ") and "rick-dev" not in line:
            prod = line.split(" ", 1)[1].strip()
            break
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        if r.stdout.strip():
            ERRORS.append(f"[F][RSI] 生产仓库 tracked 文件被改动：\n{r.stdout.strip()[:500]}")
    else:
        ERRORS.append("[F][RSI] 未能定位生产仓库")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)


if __name__ == "__main__":
    main()
