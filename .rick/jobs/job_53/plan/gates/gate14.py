#!/usr/bin/env python3
"""job_53 gate14：Bug #8 第二批——2A subagent 独立卡 + 2C token/上下文常驻 + 自动压缩。

human 反馈（2026-11-14）：
  2A. subagent 会话信息流可点击展开，了解其工作进展
  2C. token 消耗/成本（含 subagent）+ 上下文使用率常驻显示（不展开可见）+ 超阈值自动 compact

断言清单：
  A1 存在性：subagent 工具独立成卡（buildFlat 不折叠进组）+ args 摘要（agent · task）
  C1 存在性：stats API（GET /api/sessions/{id}/stats → pi get_session_stats 透传）
  C2 存在性：ChatView 常驻徽标（ctx %/token/cost）+ 80% 警示色
  C3 存在性：自动压缩（阈值监控 + Settings 开关/阈值可调）
  E1 构建绿：npm typecheck + build；go build + go test runtime+web（竞态重试）
  P1 生产零触碰
"""
import os
import subprocess
import sys
import time

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


ml = read(os.path.join(DEV_TREE, "web/src/components/chat/MessageList.tsx")) or ""
vm = read(os.path.join(DEV_TREE, "web/src/components/chat/viewModel.ts")) or ""
cv = read(os.path.join(DEV_TREE, "web/src/components/chat/ChatView.tsx")) or ""
routes = read(os.path.join(DEV_TREE, "internal/web/routes.go")) or ""
sg = read(os.path.join(DEV_TREE, "internal/web/sessions.go")) or ""
st = read(os.path.join(DEV_TREE, "web/src/routes/Settings.tsx")) or ""

check("A1 subagent 独立成卡（不折叠进组）", "subagent" in ml and "flushTools" in ml)
check("A1 subagent 摘要（agent · task 首行）", "subagent" in vm and "toolName" in vm)
check("C1 stats API（路由+handler）", "/api/sessions/{id}/stats" in routes and "SessionStats" in sg)
check("C2 常驻徽标（ctx %/token/cost）", "ctx {" in cv and "tok" in cv and "cost" in cv)
check("C2 阈值警示色（Settings 可调阈值）", "autoCompactThreshold" in cv)
check("C3 自动压缩监控（阈值）", "autoCompact" in cv or "auto-compact" in cv)
check("C3 Settings 开关+阈值", "自动压缩" in st and "threshold" in st)

r = subprocess.run(["go", "build", "./..."], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 go build ./...", r.returncode == 0, r.stderr[-300:] if r.returncode else "")
r = None
for attempt in range(1, 9):
    r = subprocess.run(["go", "test", "-p", "1", "./internal/runtime/", "./internal/web/"], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=1800)
    if r.returncode == 0:
        break
    out = (r.stderr or "") + (r.stdout or "")
    fails = [l for l in out.splitlines() if l.startswith("--- FAIL")]
    if fails and all("RemoveAll" in out.split(l)[0][-800:] or "directory not empty" in out.split(l)[0][-800:] for l in fails):
        time.sleep(2)
        continue
    break
check("E1 go test runtime+web（竞态重试）", r.returncode == 0, ((r.stderr or "") + (r.stdout or ""))[-500:] if r.returncode else "")
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
    print(f"GATE14 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE14 PASS: subagent 独立卡 + stats 常驻/自动压缩断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
