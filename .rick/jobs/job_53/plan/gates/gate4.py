#!/usr/bin/env python3
"""job_53 gate4：Bug #1 迭代 4（历史压缩卡显示压缩后 token + 比例）层门禁。

human 反馈（2026-11-14）：刷新后历史压缩卡只显示压缩前 token——应显示压缩后 token + 压缩比例。
方案：后端复刻 pi 的 chars/4 估算（summary + firstKeptEntryId 起保留消息，不含 system prompt），
在 entries 投影对 compaction 条目注入 estimatedTokensAfter（camelCase，与 JSONL/实时响应风格一致）。

断言清单：
  B1 存在性：后端注入逻辑存在（估算函数 + entries 投影注入调用，estimatedTokensAfter 字段）
  F1 存在性：前端历史卡读取 estimatedTokensAfter 并显示压缩比例（↓% 标志）
  E1 构建绿：go build + go test runtime+web（dev web down 状态下全量）+ npm typecheck/build
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


# ---- B1: 后端注入 ----
sessions = read(os.path.join(DEV_TREE, "internal/web/sessions.go"))
if sessions is None:
    check("B1 sessions.go 存在", False)
else:
    check(
        "B1 后端 estimatedTokensAfter 注入存在",
        "estimatedTokensAfter" in sessions
        and bool(re.search(r"func (estimate|inject)\w*(Tokens|Compaction)\w*|func \w*[Ee]stimate\w*", sessions)),
    )

# ---- F1: 前端历史卡显示 ----
vm = read(os.path.join(DEV_TREE, "web/src/components/chat/viewModel.ts"))
card = read(os.path.join(DEV_TREE, "web/src/components/chat/CompactionCard.tsx"))
check(
    "F1 前端读取 estimatedTokensAfter（历史卡）",
    "estimatedTokensAfter" in (vm or "") and "estimatedTokensAfter" in (card or ""),
)
check(
    "F1 历史卡显示压缩比例（↓%）",
    bool(re.search(r"↓|↓%", card or "")) and "historical" in (card or ""),
)

# ---- E1: 构建（dev web down 状态下全量 go test） ----
r = subprocess.run(["go", "build", "./..."], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 go build ./...", r.returncode == 0, r.stderr[-300:] if r.returncode else "")
r = subprocess.run(
    ["go", "test", "./internal/runtime/", "./internal/web/"],
    cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=900,
)
check("E1 go test runtime+web", r.returncode == 0, ((r.stderr or "") + (r.stdout or ""))[-600:] if r.returncode else "")
web_dir = os.path.join(DEV_TREE, "web")
r = subprocess.run(["npm", "run", "typecheck"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 npm typecheck", r.returncode == 0, (r.stderr or r.stdout)[-300:] if r.returncode else "")
r = subprocess.run(["npm", "run", "build"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E1 npm build", r.returncode == 0, (r.stderr or r.stdout)[-300:] if r.returncode else "")

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
    print(f"GATE4 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE4 PASS: 历史压缩卡 estimatedTokensAfter 注入 + 前端显示压缩后 token/比例断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
