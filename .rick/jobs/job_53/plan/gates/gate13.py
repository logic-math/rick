#!/usr/bin/env python3
"""job_53 gate13：Bug #8 第一批——打开文件改目录/文件直接选择 + 会话头部常驻模型名。

human 反馈（2026-11-14）：
  1. 顶栏「打开文件」不要填路径——直接目录/文件选择（对话框式逐级导航）
  2B. 会话运行时常驻显示默认/当前模型名（此前只在折叠的会话信息区可见）

断言清单：
  F1 存在性：打开文件对话框含文件浏览器（fsList 逐级导航 + 文件选定——不再只有路径输入框）
  F2 否定性：打开文件对话框不再以路径输入为主要交互（input 路径输入移除或降级为可选高级模式）
  M1 存在性：会话头部常驻模型名（meta.model 在头部渲染——非折叠区）
  E1 构建绿：npm typecheck + build
  P1 生产零触碰
"""
import os
import subprocess
import sys

DEV_TREE = os.environ.get("DEV_TREE", "/Users/logic/workdir/rick-dev")
PROD_REPO = os.environ.get("PROD_REPO", "/Users/logic/workdir/AI_CODING/rick")
GATE_ENV = {k: v for k, v in os.environ.items() if not k.startswith("RICK_")}

errors = []


def check(label, ok, detail=""):
    print(("PASS" if ok else "FAIL"), label, detail)
    if not ok:
        errors.append(f"{label} {detail}".strip())


def read(p):
    try:
        with open(p, encoding="utf-8") as f:
            return f.read()
    except OSError:
        return None


app = read(os.path.join(DEV_TREE, "web/src/App.tsx")) or ""
check("F1 打开文件含文件浏览器（fsList 导航+选定）", "fsList" in app and ("FilePickBrowser" in app or "filePick" in app or "文件浏览器" in app))
check("F2 路径输入不再是主要交互", 'placeholder="如 .rick/jobs' not in app)

cv = read(os.path.join(DEV_TREE, "web/src/components/chat/ChatView.tsx")) or ""
# M1: 头部（非折叠区）渲染 meta.model——头部段（TYPE_LABEL 附近）含模型名
header = cv.split("MetaRow")[0]
import re
# 头部区域粗定位：TYPE_LABEL 渲染到「已归档」之间
m = re.search(r"\{TYPE_LABEL\[sessionType\]\}(.*?)已归档", cv, re.S)
seg = m.group(1) if m else ""
check("M1 会话头部常驻模型名", "meta.model" in seg or "currentModel" in seg)

web_dir = os.path.join(DEV_TREE, "web")
r = subprocess.run(["npm", "run", "typecheck"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 npm typecheck", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")
r = subprocess.run(["npm", "run", "build"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E1 npm build", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")

r = subprocess.run(
    ["git", "-C", PROD_REPO, "status", "--porcelain", "-uno"],
    capture_output=True, text=True, env=GATE_ENV, timeout=60,
)
known_rel = ".rick/jobs/job_50/doing/tasks.json"
dirty = [l for l in r.stdout.splitlines() if l.strip() and l.split()[-1] not in (known_rel,)]
check("P1 生产树零改动", len(dirty) == 0, "; ".join(dirty)[:200] if dirty else "")

print()
if errors:
    print(f"GATE13 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE13 PASS: 打开文件直接选择 + 头部常驻模型名断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
