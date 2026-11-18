#!/usr/bin/env python3
"""job_53 gate1：Bug #1（compact 全链路修复）层门禁——模块集成测试。

human 裁决的 4 项修复断言（web-feedback-iteration-loop 步骤 4 惯例：
存在性 + 否定性 + 构建绿 + 生产零触碰 + RICK_* 环境净化）。

断言清单：
  A1 存在性：ChatInput.tsx SLASH_COMMANDS 中 /compact 条目恰好 1 条（available:true）
  A2 否定性：不再存在 available:false 的 /compact 占位条目
  B1 存在性：sessions.go 存在「非 active 会话 compact 先 resume worker」分支
             （标志：resume-then-compact 路径函数或显式分支标记 COMPACT_RESUME_MARKER）
  C1 存在性：piMessage/rpc 消息 content 解析兼容 string（executor.go 自定义 UnmarshalJSON 或等价机制，
             标志：UnmarshalJSON / 兼容 string 的解析分支）
  D1 存在性：压缩可视化卡——viewModel/ChatView 存在 compaction 卡渲染分支（kind "compaction"）
  D2 存在性：压缩卡含提示词原文折叠区（SUMMARIZATION_SYSTEM_PROMPT 副本 + 折叠交互标志 collapsed）
  E1 构建绿：go build ./... && go test ./internal/runtime/ ./internal/web/
  E2 构建绿：web npm run typecheck && npm run build
  P1 生产零触碰：git -C <生产树> status --porcelain -uno 为空
  环境净化：go test 子进程剥离 RICK_* 环境变量（RICK_STATE_DIR 穿透 t.Setenv 的 job_43 教训）
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
    except OSError as e:
        return None


# ---- A1/A2: ChatInput 双条目修复 ----
chat_input = read(os.path.join(DEV_TREE, "web/src/components/chat/ChatInput.tsx"))
if chat_input is None:
    check("A1 ChatInput.tsx 存在", False)
else:
    compact_entries = re.findall(r'\{\s*name:\s*"/compact"', chat_input)
    check("A1 /compact 条目恰好 1 条", len(compact_entries) == 1, f"found {len(compact_entries)}")
    bad = re.search(r'name:\s*"/compact"[^}]*available:\s*false', chat_input)
    check("A2 无 available:false 的 /compact 占位", bad is None)

# ---- B1: 非 active 会话 compact 自动 resume ----
sessions = read(os.path.join(DEV_TREE, "internal/web/sessions.go"))
if sessions is None:
    check("B1 sessions.go 存在", False)
else:
    check(
        "B1 compact 前 resume worker 分支存在",
        ("COMPACT_RESUME_MARKER" in sessions)
        or (re.search(r"resume[A-Za-z]*[Ww]orker|resumeSession|resume.*compact", sessions) is not None),
    )

# ---- C1: piMessage content 兼容 string ----
executor = read(os.path.join(DEV_TREE, "internal/runtime/executor.go"))
if executor is None:
    check("C1 executor.go 存在", False)
else:
    check(
        "C1 piMessage content string 兼容（UnmarshalJSON/等价分支）",
        ("UnmarshalJSON" in executor) or ("PiMessage" in executor and "string" in executor and "Content" in executor),
    )

# ---- D1/D2: 压缩可视化卡 ----
vm = read(os.path.join(DEV_TREE, "web/src/components/chat/viewModel.ts"))
chatview = read(os.path.join(DEV_TREE, "web/src/components/chat/ChatView.tsx"))
comp_kind = (vm or "") + (chatview or "")
check(
    "D1 压缩卡渲染分支存在（compaction kind）",
    ('"compaction"' in comp_kind) or ("'compaction'" in comp_kind),
)
# D2: 提示词原文折叠区（模板副本 + 折叠交互）——viewModel/ChatView/新组件任一处
comp_dir = os.path.join(DEV_TREE, "web/src/components/chat")
prompt_copy = False
fold_ui = "collapsed" in comp_kind or "折叠" in comp_kind or "expanded" in comp_kind
for fn in os.listdir(comp_dir) if os.path.isdir(comp_dir) else []:
    if fn.endswith((".tsx", ".ts")):
        body = read(os.path.join(comp_dir, fn)) or ""
        if "SUMMARIZATION_SYSTEM_PROMPT" in body or "You are a context summarization assistant" in body:
            prompt_copy = True
check("D2 压缩卡含提示词原文折叠区", prompt_copy and fold_ui, f"prompt_copy={prompt_copy} fold_ui={fold_ui}")

# ---- E1: go 构建 + 模块测试 ----
r = subprocess.run(["go", "build", "./..."], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 go build ./...", r.returncode == 0, r.stderr[-300:] if r.returncode else "")
r = subprocess.run(
    [
        "go", "test", "./internal/runtime/", "./internal/web/",
    ],
    cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=900,
)
check("E1 go test runtime+web", r.returncode == 0, ((r.stderr or "") + (r.stdout or ""))[-1500:] if r.returncode else "")

# ---- E2: 前端 typecheck + build ----
web_dir = os.path.join(DEV_TREE, "web")
r = subprocess.run(["npm", "run", "typecheck"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E2 npm typecheck", r.returncode == 0, (r.stderr or r.stdout)[-300:] if r.returncode else "")
r = subprocess.run(["npm", "run", "build"], cwd=web_dir, capture_output=True, text=True, env=GATE_ENV, timeout=900)
check("E2 npm build", r.returncode == 0, (r.stderr or r.stdout)[-300:] if r.returncode else "")

# ---- P1: 生产零触碰 ----
r = subprocess.run(
    ["git", "-C", PROD_REPO, "status", "--porcelain", "-uno"],
    capture_output=True, text=True, env=GATE_ENV, timeout=60,
)
known_rel = ".rick/jobs/job_50/doing/tasks.json"  # 开工前既有脏文件（基线豁免，2026-11-12 实测开工前已 M）
dirty = [l for l in r.stdout.splitlines() if l.strip() and l.split()[-1] not in (known_rel,)]
check("P1 生产树零改动", len(dirty) == 0, "; ".join(dirty)[:200] if dirty else "")

print()
if errors:
    print(f"GATE1 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE1 PASS: Bug #1 四项修复全部断言通过（双条目/resume-then-compact/string content/可视化卡+构建绿+生产零触碰）")
sys.exit(0)
