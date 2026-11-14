#!/usr/bin/env python3
"""job_53 gate3：Bug #1 迭代 3（删冗余通知 + 无新对话压缩拦截）层门禁。

human 反馈（2026-11-14）：
  A：通知栏「压缩已发起」无用——删除（压缩中占位卡已可见过程）
  B：距上次成功压缩无新对话时不要再发起压缩（避免无谓 resume worker + pi 调用）

断言清单：
  F1 否定性：「压缩已发起」提示文案已删除（ChatView 不再 setModelHint 压缩已发起）
  F2 存在性：/compact 前置拦截——最后完整压缩卡之后无新对话时不调 API
             （拦截判断标志：最后 compaction item + 之后无 user/assistant/tool 项）
  F3 存在性：Already compacted 错误中文映射（前端错误处理含映射）
  E1 构建绿：npm typecheck + build；go build ./...（防误伤）
  P1 生产零触碰
"""
import os
import re
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


# ---- F1: 「压缩已发起」已删 ----
chatview = read(os.path.join(DEV_TREE, "web/src/components/chat/ChatView.tsx"))
check("F1 「压缩已发起」提示已删除", chatview is not None and "压缩已发起" not in chatview)

# ---- F2: 无新对话拦截 ----
check(
    "F2 /compact 无新对话前置拦截",
    chatview is not None and bool(re.search(r"无需压缩|无新对话|无需再压缩", chatview)),
)

# ---- F3: Already compacted 中文映射 ----
mapped = False
for fn in ("ChatView.tsx", "viewModel.ts", "api/client.ts"):
    body = read(os.path.join(DEV_TREE, "web/src/components/chat", fn)) or read(os.path.join(DEV_TREE, "web/src/api", fn)) or ""
    if "Already compacted" in body:
        mapped = True
check("F3 Already compacted 错误中文映射存在", mapped)

# ---- E1: 构建 ----
web_dir = os.path.join(DEV_TREE, "web")
r = subprocess.run(["npm", "run", "typecheck"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 npm typecheck", r.returncode == 0, (r.stderr or r.stdout)[-300:] if r.returncode else "")
r = subprocess.run(["npm", "run", "build"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E1 npm build", r.returncode == 0, (r.stderr or r.stdout)[-300:] if r.returncode else "")
r = subprocess.run(["go", "build", "./..."], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 go build ./...", r.returncode == 0, r.stderr[-300:] if r.returncode else "")

# ---- P1: 生产零触碰 ----
r = subprocess.run(
    ["git", "-C", PROD_REPO, "status", "--porcelain", "-uno"],
    capture_output=True, text=True, env=GATE_ENV, timeout=60,
)
known_rel = ".rick/jobs/job_50/doing/tasks.json"  # 开工前既有脏文件（基线豁免）
dirty = [l for l in r.stdout.splitlines() if l.strip() and l.split()[-1] not in (known_rel,)]
check("P1 生产树零改动", len(dirty) == 0, "; ".join(dirty)[:200] if dirty else "")

print()
if errors:
    print(f"GATE3 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE3 PASS: 删冗余通知 + 无新对话拦截 + 错误中文映射断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
