#!/usr/bin/env python3
"""gate3.py — job_43 增量迭代 2 第 3 层门禁（task5 输入框极简 + task6 移动端性能二轮）。

模块级集成测试（human 确认，agent 不得修改）：
  A. 前端 typecheck + build；Go build（VERSION bump）
  B. 静态断言：task5 输入框极简 + task6 动画/blur/节流降级
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


def assert_file_not_contains(path, patterns, desc):
    try:
        content = read(path)
    except FileNotFoundError:
        ERRORS.append(f"[{desc}] 文件缺失：{path}")
        return
    for pat, label in patterns:
        if re.search(pat, content):
            ERRORS.append(f"[{desc}] {path} 不应再包含 {label}（pattern: {pat}）")


def main():
    # ---- A. 构建 ----
    run(["go", "build", "./..."])
    web = os.path.join(ROOT, "web")
    run(["npm", "run", "typecheck"], cwd=web)
    run(["npm", "run", "build"], cwd=web)

    # ---- B. task5：输入框极简 ----
    ci = "web/src/components/chat/ChatInput.tsx"
    assert_file_not_contains(ci, [
        (r"enterMode|ENTER_MODE_KEY|readEnterMode|toggleEnterMode", "enterMode 模式逻辑（应已删除）"),
        (r"↵", "换行按钮（应已删除）"),
    ], "task5-KR6a/b")
    assert_file_contains(ci, [
        (r"scrollHeight", "textarea scrollHeight 自适应"),
        (r"isComposing", "IME 守卫保持"),
    ], "task5-KR6c")
    ci_content = read(ci)
    # Enter 固定换行：Enter 分支走 insertNewline（不再直接 send）
    if not re.search(r"insertNewline", ci_content):
        ERRORS.append("[task5-KR6b] ChatInput 未见 insertNewline（Enter=换行依赖它）")
    # Ctrl/Cmd+Enter 发送保留
    if not (re.search(r"ctrlKey|metaKey", ci_content) and re.search(r"[Ee]nter", ci_content)):
        ERRORS.append("[task5-KR6b] 未见 Ctrl/Cmd+Enter 发送处理")
    # max-h-36 硬顶应放宽
    if re.search(r"max-h-36\b", ci_content):
        ERRORS.append("[task5-KR6c] ChatInput 仍含 max-h-36 硬顶（应放宽）")
    assert_file_contains("cmd/rick/main.go", [
        (r'VERSION\s*=\s*"5\.0\.13"', "VERSION 提升至 5.0.13"),
    ], "task5-KR6d")

    # ---- B. task6：动画降级 + blur 补漏 + 节流 ----
    ml = "web/src/components/chat/MessageList.tsx"
    ml_content = read(ml)
    # 空态飞船静态（flying 不再为 true 于空态）
    if re.search(r"<Saucer[^>]*flying(?:\s|>|=\{true\})", ml_content):
        ERRORS.append("[task6-KR7a] MessageList 空态 Saucer 仍是 flying 动画（应静态）")
    # 展开工具条 blur 移动端降级（md: 前缀出现于该文件的 backdrop-blur 类）
    if re.search(r"(?<!md:)backdrop-blur", ml_content):
        ERRORS.append("[task6-KR7c] MessageList 存在无 md: 前缀的 backdrop-blur（展开条补漏未完成）")
    # theme.css：窄屏动画降级媒体查询
    theme = "web/src/styles/theme.css"
    theme_content = read(theme)
    if not re.search(r"@media\s*\(max-width\s*:\s*767px\)", theme_content):
        ERRORS.append("[task6-KR7b/d] theme.css 未见 767px 窄屏动画降级媒体查询")
    else:
        # 媒体查询块内应关闭装饰动画（rm-saucer-hover / rm-beam / rm-pulse 至少其一）
        m = re.search(r"@media\s*\(max-width\s*:\s*767px\)\s*\{(.*?)\n\}", theme_content, re.DOTALL)
        block = m.group(1) if m else ""
        if not re.search(r"rm-saucer-hover|rm-beam|rm-pulse|animation\s*:\s*none", block):
            ERRORS.append("[task6-KR7b/d] 767px 媒体查询块未关闭装饰动画")
    # ToolElapsed 窄屏降频
    assert_file_contains("web/src/components/chat/ToolCallCard.tsx", [
        (r"767|isNarrow|matchMedia|innerWidth", "窄屏判定"),
        (r"5000|5_000|5\s*\*\s*1000", "5s 降频常量"),
    ], "task6-KR7e")
    # App.tsx 断线横幅 + FileReaderPanel blur 降级
    for path in ("web/src/App.tsx", "web/src/components/files/FileReaderPanel.tsx"):
        assert_file_not_contains(path, [
            (r"(?<!md:)backdrop-blur", "无 md: 前缀的 backdrop-blur"),
        ], "task6-KR7c")

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
