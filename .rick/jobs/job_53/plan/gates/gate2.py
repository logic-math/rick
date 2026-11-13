#!/usr/bin/env python3
"""job_53 gate2：Bug #1 迭代 2（压缩卡持久化 + 整卡折叠）层门禁。

human 反馈（2026-11-13）：压缩卡要一直在会话流中（刷新后也在），且整卡可折叠。
本轮改动面纯前端（viewModel.ts 历史重建 + CompactionCard.tsx 折叠交互），零 Go 改动。

断言清单：
  F1 存在性：buildHistoryItems 的 compaction 分支渲染完整历史卡
             （historical 卡携带 summary + tokensBefore，不再只是边界标记）
  F2 存在性：CompactionCard 整卡折叠交互（折叠/展开切换：collapsed/expanded/setCollapsed 类标志）
  F3 否定性：旧设计「历史回放不重放摘要全文」的轻量分割线不再存在
             （viewModel 历史分支不再以「不重放摘要全文」为由只给 tokensBefore）
  E1 构建绿：npm run typecheck && npm run build
  E2 构建绿：go build ./...（零 Go 改动，防误伤）
  P1 生产零触碰：git -C <生产树> status --porcelain -uno 干净（除基线豁免）
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


# ---- F1/F3: 历史重建完整卡 ----
vm = read(os.path.join(DEV_TREE, "web/src/components/chat/viewModel.ts"))
if vm is None:
    check("F1 viewModel.ts 存在", False)
else:
    hist = re.search(r'if \(entry\.type === "compaction"\) \{(.*?)\n    \}', vm, re.S)
    body = hist.group(1) if hist else ""
    check(
        "F1 历史压缩卡携带 summary",
        bool(hist) and "summary" in body and "entry.summary" in body,
    )
    check(
        "F3 旧轻量分割线设计已移除",
        "不重放摘要全文" not in vm,
    )

# ---- F2: 整卡折叠交互 ----
card = read(os.path.join(DEV_TREE, "web/src/components/chat/CompactionCard.tsx"))
if card is None:
    check("F2 CompactionCard.tsx 存在", False)
else:
    check(
        "F2 整卡折叠交互存在",
        ("setCollapsed" in card or "setExpanded" in card or "setOpen" in card)
        and ("useState" in card)
        and ("collapsed" in card or "expanded" in card or "open" in card),
    )

# ---- E1: 前端构建 ----
web_dir = os.path.join(DEV_TREE, "web")
r = subprocess.run(["npm", "run", "typecheck"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 npm typecheck", r.returncode == 0, (r.stderr or r.stdout)[-300:] if r.returncode else "")
r = subprocess.run(["npm", "run", "build"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E1 npm build", r.returncode == 0, (r.stderr or r.stdout)[-300:] if r.returncode else "")

# ---- E2: go build 防误伤（零 Go 改动，不跑 go test——纯前端轮） ----
r = subprocess.run(["go", "build", "./..."], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E2 go build ./...", r.returncode == 0, r.stderr[-300:] if r.returncode else "")

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
    print(f"GATE2 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE2 PASS: 压缩卡持久化（历史重建完整卡）+ 整卡折叠断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
