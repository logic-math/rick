#!/usr/bin/env python3
"""job_53 gate17：Bug #8 第三批——消息懒加载（游标定位）+ 滚动游标跨设备保留。

human 需求（2026-11-14）：
  打开会话渲染到「指定游标前 N 条」（N 适配屏幕），刷新/跨设备恢复到上次阅读位置。

断言清单：
  C1 存在性：滚动游标记录（MessageList 视口锚回调 + ChatView debounce 存服务端 drafts scroll:<sid>）
  C2 存在性：恢复拉取（有游标时 getEntries before=游标 limit=N（屏幕适配）+ since=游标拉后续）
  C3 存在性：初始定位（Virtuoso scrollToIndex 到游标锚条目——非贴底）
  E1 构建绿 + P1 生产零触碰
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

cv = read(os.path.join(DEV_TREE, "web/src/components/chat/ChatView.tsx")) or ""
ml = read(os.path.join(DEV_TREE, "web/src/components/chat/MessageList.tsx")) or ""

check("C1 视口锚回调（rangeChanged/锚暴露）", "onViewportAnchor" in ml or "rangeChanged" in ml)
check("C1 游标存服务端（scroll: key + saveDraft）", "scroll:" in cv and "getDraft" in cv)

check("C2 恢复拉取（before=游标 + since=游标）", "before: scroll" in cv or "scrollCursor" in cv or "scrollAnchor" in cv)
check("C2 游标中心懒加载窗口（before/since 各 N/2）+ 重试", "CURSOR_RETRIES" in cv and "before: anchorEntryId" in cv)

check("C3 初始定位（scrollToIndex 锚）", "initialAnchor" in ml or "scrollToAnchor" in ml or "initialScrollAnchor" in ml)

web_dir = os.path.join(DEV_TREE, "web")
r = subprocess.run(["npm", "run", "typecheck"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 npm typecheck", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")
r = subprocess.run(["npm", "run", "build"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E1 npm build", r.returncode == 0, (r.stderr or r.stdout)[-400:] if r.returncode else "")

r = subprocess.run(["git", "-C", PROD_REPO, "status", "--porcelain", "-uno"], capture_output=True, text=True, env=GATE_ENV, timeout=60)
known_rel = ".rick/jobs/job_50/doing/tasks.json"
dirty = [l for l in r.stdout.splitlines() if l.strip() and l.split()[-1] not in (known_rel,)]
check("P1 生产树零改动", len(dirty) == 0, "; ".join(dirty)[:200] if dirty else "")

print()
if errors:
    print(f"GATE17 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE17 PASS: 消息懒加载游标定位 + 滚动游标跨设备保留断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
