#!/usr/bin/env python3
"""gate1.py — job_43 第 1 层门禁（task1 pi 会话持久化 + task2 输入框多行 + task3 导航实时性）。

模块级集成测试（human 确认，agent 不得修改）：
  A. Go 侧：build + test（internal/runtime, internal/config, internal/cmd——含新增
     空闲回收禁用与 MaxActive 配置测试）
  B. 前端：typecheck + build
  C. 静态断言：关键改动落盘（写域内文件）
  D. 生产零触碰：生产仓库 tracked 文件无改动

输出 JSON {"pass": bool, "errors": [...]}，exit 0=绿 / 1=红。
"""
import json
import os
import re
import subprocess
import sys


def get_project_root():
    # .rick/jobs/job_43/plan/gates/gate1.py → 6×dirname 到仓库根
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
    # ---- A. Go build + test ----
    run(["go", "build", "./..."])
    run(["go", "test", "./internal/runtime/", "./internal/config/", "./internal/cmd/"])

    # ---- B. 前端 typecheck + build ----
    web = os.path.join(ROOT, "web")
    run(["npm", "run", "typecheck"], cwd=web)
    run(["npm", "run", "build"], cwd=web)

    # ---- C. 静态断言 ----
    # task1：web.go 组合根禁用空闲回收 + MaxActive 接入 config（负值字面量或 IdleDisabled 哨兵语义 + 64 缺省）
    assert_file_contains("internal/cmd/web.go", [
        (r"IdleTimeout\s*:\s*(-|runtime\.IdleDisabled|IdleDisabled)", "IdleTimeout 负值/IdleDisabled 哨兵禁用空闲回收"),
        (r"MaxActive", "MaxActive 接入 supervisor 配置"),
    ], "task1-KR1a/KR1b")
    assert_file_contains("internal/config/config.go", [
        (r"web_max_active", "config 新增 web_max_active 字段"),
    ], "task1-KR1b")
    # 64 缺省（web.go 或 config.go 或 supervisor.go 任一处出现语义 64 默认）
    combined = ""
    for p in ("internal/cmd/web.go", "internal/config/config.go", "internal/runtime/supervisor.go"):
        try:
            combined += read(p)
        except FileNotFoundError:
            pass
    if not (re.search(r"64", combined) and re.search(r"[Ww]ebMaxActive|web_max_active", combined)):
        ERRORS.append("[task1-KR1b] 未找到 MaxActive 默认 64 语义（web_max_active/WebMaxActive + 64）")
    # task1：supervisor.go 禁用语义注释（中文或英文，哨兵定义处或负值语义处）
    assert_file_contains("internal/runtime/supervisor.go", [
        (r"IdleDisabled|[负禁].{0,40}(空闲回收|[Ii]dle)", "负值/IdleDisabled 禁用空闲回收的哨兵或注释澄清"),
    ], "task1-KR1a")
    # task1：心跳保留（HeartbeatTimeout 仍在 supervisor.go，未被移除）
    assert_file_contains("internal/runtime/supervisor.go", [
        (r"HeartbeatTimeout", "心跳超时语义保留"),
    ], "task1-KR1c")
    # task1：版本 5.0.12
    assert_file_contains("cmd/rick/main.go", [
        (r'VERSION\s*=\s*"5\.0\.12"', "VERSION 提升至 5.0.12"),
    ], "task1-KR1d")
    # task1：新增测试（空闲禁用 + MaxActive 解析）
    sup_test = ""
    for p in ("internal/runtime/supervisor_test.go",):
        try:
            sup_test += read(p)
        except FileNotFoundError:
            pass
    cmd_all = ""
    for p in ("internal/cmd/web_test.go", "internal/cmd/web.go", "internal/config/config_test.go"):
        try:
            cmd_all += read(p)
        except FileNotFoundError:
            pass
    if not re.search(r"[Ii]dle.*(禁|disable|[Dd]isabled|[Kk]eep|存活|[Nn]o.?reap)|负.{0,20}[Ii]dle", sup_test):
        ERRORS.append("[task1-KR1e] internal/runtime 测试未见「空闲回收禁用后 worker 存活」断言")
    if not re.search(r"64", cmd_all):
        ERRORS.append("[task1-KR1e] internal/cmd 或 config 测试未见 MaxActive=64 默认断言")

    # task2：ChatInput 多行
    chat_path = "web/src/components/chat/ChatInput.tsx"
    assert_file_contains(chat_path, [
        (r"[Ee]nterMode", "Enter 语义模式（enterMode）"),
        (r"localStorage", "模式 localStorage 持久化"),
        (r"Shift", "Shift+Enter 换行保持"),
        (r"isComposing", "IME isComposing 守卫保持"),
    ], "task2-KR3a/d")
    try:
        ci = read(chat_path)
        if not (re.search(r"ctrlKey", ci) and re.search(r"\"j\"|'j'", ci)):
            ERRORS.append("[task2-KR3a] ChatInput 未见 Ctrl+J 换行处理（ctrlKey + 'j' 键判定）")
        if not re.search(r"换行", ci):
            ERRORS.append("[task2-KR3c/e] ChatInput 未见换行提示/换行按钮相关文案")
    except FileNotFoundError:
        pass

    # task3：导航
    assert_file_contains("web/src/routes/SessionPage.tsx", [
        (r"key=\{[^}]*session\.id\}|key=\{`[^`]*\$\{session\.id\}`\}", "ChatView/MonitorView key={session.id}"),
    ], "task3-KR2a")
    assert_file_contains("web/src/App.tsx", [
        (r"scrollTop\s*=\s*0|scrollTo\(\s*\{\s*top:\s*0", "路由切换滚动重置"),
        (r"useLocation|location\.pathname", "pathname 依赖"),
    ], "task3-KR2b")
    assert_file_contains("web/src/components/layout/WorkspaceListNode.tsx", [
        (r"useNavigate|navigate\(", "行点击导航"),
        (r"stopPropagation", "chevron 展开/折叠与导航分离"),
    ], "task3-KR2c")

    # ---- D. 生产零触碰 ----
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
