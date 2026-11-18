#!/usr/bin/env python3
"""job_53 gate11：Bug #6——弹窗草稿跨设备 flush + 单 URL 模式（MemoryRouter + 位置服务端记忆）。

human 反馈（2026-11-14）：
  A. 新建会话对话框草稿不能跨设备共享（防抖 500ms 窗口内关闭弹窗 → 最后输入丢失）
  B. 单 URL 模式：打开只有 IP:port/ 一个 URL；页面切换 URL 不跳转（SPA 内部状态）；
     停留位置存服务端——跨设备打开恢复到上一次操作停留的页面，整体不丢内容。

断言清单：
  A1 存在性：NewSessionModal 关闭路径 flush 草稿（onClose/unmount 即时 PUT——不再依赖防抖）
  U1 存在性：App.tsx 用 MemoryRouter（URL 恒定——BrowserRouter 移除）
  U2 存在性：位置服务端记忆（ui-state draft key 同步：启动 GET 恢复 initialEntries + 路由变化防抖 PUT）
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


# ---- A1: 弹窗草稿 flush ----
modal = read(os.path.join(DEV_TREE, "web/src/components/sessions/NewSessionModal.tsx")) or ""
check("A1 弹窗关闭 flush 草稿", "flushDrafts" in modal or "closeWithFlush" in modal or "unmount" in modal.lower())

# ---- U1: MemoryRouter ----
app = read(os.path.join(DEV_TREE, "web/src/App.tsx")) or ""
check("U1 App 用 MemoryRouter（URL 恒定）", "MemoryRouter" in app and "BrowserRouter" not in app)

# ---- U2: 位置服务端记忆 ----
check(
    "U2 位置服务端记忆（ui-state 同步）",
    "ui-state" in app and ("initialEntries" in app or "initialRoute" in app),
)

# ---- E1: 构建 ----
web_dir = os.path.join(DEV_TREE, "web")
r = subprocess.run(["npm", "run", "typecheck"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 npm typecheck", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")
r = subprocess.run(["npm", "run", "build"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E1 npm build", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")

# ---- P1: 生产零触碰 ----
r = subprocess.run(
    ["git", "-C", PROD_REPO, "status", "--porcelain", "-uno"],
    capture_output=True, text=True, env=GATE_ENV, timeout=60,
)
known_rel = ".rick/jobs/job_50/doing/tasks.json"
dirty = [l for l in r.stdout.splitlines() if l.strip() and l.split()[-1] not in (known_rel,)]
check("P1 生产树零改动", len(dirty) == 0, "; ".join(dirty)[:200] if dirty else "")

print()
if errors:
    print(f"GATE11 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE11 PASS: 弹窗草稿 flush + 单 URL 模式（MemoryRouter + 位置服务端记忆）断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
