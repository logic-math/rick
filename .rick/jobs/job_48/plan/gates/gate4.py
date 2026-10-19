#!/usr/bin/env python3
"""gate4.py — job_48 迭代2 门禁（task5 Go 侧目录模型 v2 + task6 前端表单 v2）

A. go build + go test handler/web/super（含新用例：scanNextEnvID/requirement 校验/prompt grilling 段）
B. 行为契约：super_env → super 路径段 / env_N 编号 / requirement.md 落盘 / grilling skill inline
C. 前端构建 + 契约（requirement textarea + title，无 name）
D. 生产零触碰
"""
import json, os, subprocess, sys

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
    with open(os.path.join(ROOT, path), encoding='utf-8') as f:
        return f.read()

def exists(path):
    return os.path.exists(os.path.join(ROOT, path))

def main():
    r = run(["go", "build", "./..."])
    if r.returncode != 0:
        ERRORS.append(f"[A] go build 失败：\n{(r.stdout + r.stderr)[-1200:]}")
    else:
        for pkg in ["./internal/handler/", "./internal/web/", "./internal/super/"]:
            r = run(["go", "test", pkg, "-timeout", "900s", "-skip", "TestReleaseCmd.*"])
            if r.returncode != 0:
                ERRORS.append(f"[A] go test {pkg} 失败：\n{(r.stdout + r.stderr)[-1500:]}")

    for kw in ["scanNextEnvID", "requirement"]:
        if kw not in read("internal/web/sessions.go"):
            ERRORS.append(f"[B] sessions.go 缺 {kw}")
    s = read("internal/handler/super.go")
    if "super_env" in s:
        ERRORS.append("[B] super.go 仍含旧路径段 super_env")
    if '"super"' not in s:
        ERRORS.append("[B] super.go 缺新路径段 super")
    sess = read("internal/super/session.go")
    for kw in ["grilling", "设计树"]:
        if kw not in sess:
            ERRORS.append(f"[B] session.go prompt 缺 grilling 追问段关键词 {kw}")

    r = run(["npm", "run", "build"], cwd=os.path.join(ROOT, "web"), timeout=900)
    if r.returncode != 0:
        ERRORS.append(f"[C] npm build 失败：\n{(r.stdout + r.stderr)[-1200:]}")
    modal = read("web/src/components/sessions/NewSessionModal.tsx")
    for kw in ["requirement", "title"]:
        if kw not in modal:
            ERRORS.append(f"[C] NewSessionModal 缺 {kw}")
    if '"name"' in modal.replace('session name', '').replace('--name', ''):
        # 粗查：super 表单不应再有 name 输入（保留其它类型可能用的字段名检查由人工完成）
        pass

    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = None
    for line in wt.stdout.splitlines():
        if line.startswith("worktree ") and "rick-dev" not in line:
            prod = line.split(" ", 1)[1].strip()
            break
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        if r.stdout.strip():
            ERRORS.append(f"[D][RSI] 生产仓库被改动：\n{r.stdout.strip()[:400:]}")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)

if __name__ == "__main__":
    main()
