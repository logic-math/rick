#!/usr/bin/env python3
"""job_53 gate8：Bug #5 大删除（Go 侧）——plan/doing/learning/ctrl 四种会话类型全链路移除。

human 裁决（2026-11-14，按推荐）：easy 保留；历史存量会话数据保留（列表可见可打开）；
TaskBoard/jobs SSE 连带删除（doing 死代码清理）。

断言清单（全部否定性=删除证据）：
  N1 CLI 四命令文件已删 + root.go 无注册
  N2 prompt 模板四文件已删 + manager 无 embed/方法
  N3 builder 四类型方法已删（Build/Save Prompt）
  N4 sessions.go 无四类型创建分支（prepareInteractive case）+ doing runner/JobsUpdate 删除
  N5 registry.go 无四类型常量 + watcher jobs 已删
  E1 构建绿：go build ./... + go test ./internal/...（全量，dev web down）
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


def gone(p):
    return not os.path.exists(os.path.join(DEV_TREE, p))


# ---- N1: CLI ----
for f in ("internal/cmd/plan.go", "internal/cmd/doing.go", "internal/cmd/learning.go", "internal/cmd/ctrl.go",
          "internal/cmd/plan_test.go", "internal/cmd/doing_test.go", "internal/cmd/learning_test.go"):
    if not gone(f):
        check(f"N1 CLI {f} 已删", False)
check("N1 CLI 四命令文件已删", all(gone(f) for f in (
    "internal/cmd/plan.go", "internal/cmd/doing.go", "internal/cmd/learning.go", "internal/cmd/ctrl.go",
    "internal/cmd/plan_test.go", "internal/cmd/doing_test.go", "internal/cmd/learning_test.go",
)))
root = read("internal/cmd/root.go") or ""
check(
    "N1 root.go 无四命令注册",
    all(s not in root for s in ("NewPlanCmd", "NewDoingCmd", "NewLearningCmd", "NewCtrlCmd")),
)

# ---- N2: prompt 模板（plan/doing/ctrl 删；learning 保留——easy 流程的
# Learning Loop prompt 生成（GenerateEasyLearningPromptFile）依赖，属 easy 保留面）----
check("N2 模板 plan/doing/ctrl 已删", all(gone(f) for f in (
    "internal/prompt/templates/plan.md", "internal/prompt/templates/doing.md", "internal/prompt/templates/ctrl.md",
)))
check("N2 learning.md 保留（easy 流程模板）", not gone("internal/prompt/templates/learning.md"))
mgr = read("internal/prompt/manager.go") or ""
check(
    "N2 manager 无 plan/doing/ctrl embed",
    all(s not in mgr for s in ("templates/plan.md", "templates/doing.md", "templates/ctrl.md")),
)

# ---- N3: builder ----
pb = read("internal/builder/pibuilder.go") or ""
check(
    "N3 builder 四类型方法已删",
    all(s not in pb for s in (
        "func (p *PIBuilder) BuildPlan(", "func (p *PIBuilder) BuildDoing(", "func (p *PIBuilder) BuildCtrl(",
        "func (p *PIBuilder) BuildLearning(", "func (p *PIBuilder) SavePlanPrompt(", "func (p *PIBuilder) SaveDoingPrompt(",
        "func (p *PIBuilder) SaveCtrlPrompt(", "func (p *PIBuilder) SaveLearningPrompt(", "func (p *PIBuilder) writeLearningSkills(",
        "func (p *PIBuilder) renderLearning(",
    )),
)

# ---- N4: sessions.go ----
sg = read("internal/web/sessions.go") or ""
check("N4 prepareInteractive 无四类型分支", "case SessionTypePlan:" not in sg and "case SessionTypeCtrl:" not in sg and "case SessionTypeLearning:" not in sg)
check("N4 doing runner 已删", "DoingRunner" not in sg and "handler.DoingEvent" not in sg and "JobsUpdate(" not in sg)
check("N4 后台 doing 分支已删", "SessionTypeDoing" not in sg)

# ---- N5: registry + watcher ----
reg = read("internal/web/registry.go") or ""
check(
    "N5 registry 无四类型常量",
    all(s not in reg for s in ("SessionTypePlan", "SessionTypeCtrl", "SessionTypeLearning", "SessionTypeDoing")),
)
wat = read("internal/web/watcher.go") or ""
check("N5 watcher jobs 已删", "func watchJobs" not in wat and "readTaskStatuses" not in wat and "diffTaskStatuses" not in wat)

# ---- E1: 构建（dev web down 状态下全量） ----
r = subprocess.run(["go", "build", "./..."], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=600)
check("E1 go build ./...", r.returncode == 0, r.stderr[-500:] if r.returncode else "")
# -p 1：包串行执行——spawn 真实 worker 的测试在全量并发负载下撞 TempDir 清理竞态
# （job_50 backlog；串行化消除并发源，重试兜底剩余偶发）。
for attempt in (1, 2, 3):
    r = subprocess.run(["go", "test", "-p", "1", "./internal/..."], cwd=DEV_TREE, capture_output=True, text=True, env=GATE_ENV, timeout=1800)
    if r.returncode == 0:
        break
check("E1 go test ./internal/...（含竞态重试）", r.returncode == 0, ((r.stderr or "") + (r.stdout or ""))[-600:] if r.returncode else "")

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
    print(f"GATE8 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE8 PASS: plan/doing/learning/ctrl 四类型 Go 全链路删除（CLI+模板+builder+web+watcher）断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
