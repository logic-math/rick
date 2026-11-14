#!/usr/bin/env python3
"""job_53 gate12：Bug #7——跨设备草稿失效（SW 旧缓存）+ 侧栏手机端适配。

human 反馈（2026-11-14）：
  A. 跨设备草稿还是不行——实测 drafts.json 无任何会话草稿（只有 ui-state）：
     手机端 service worker 缓存旧 bundle（autoUpdate 不跳等待 → 手机 PWA 长期旧版，
     旧版草稿走 localStorage 从未 PUT 服务端）。
  B. 调整侧边栏后手机端没适配正确（横屏 ≥768px 视口时拖宽值生效占屏——无视口上限）。

断言清单：
  S1 存在性：SW 立即接管（skipWaiting + clientsClaim——新 bundle 不等所有 tab 关闭）
  S2 存在性：侧栏宽度视口 clamp（resize 动态上限——占屏不超过视口 60%）
  V1 存在性：前端版本探针（设置/页脚显示 build_id——两端可确认新旧 bundle）
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


vite = read(os.path.join(DEV_TREE, "web/vite.config.ts")) or ""
check("S1 SW skipWaiting+clientsClaim（立即接管）", "skipWaiting" in vite and "clientsClaim" in vite)

app = read(os.path.join(DEV_TREE, "web/src/App.tsx")) or ""
check("S2 侧栏宽度视口 clamp", "innerWidth" in app and ("clampSidebarW" in app or "0.6" in app or "resize" in app))

settings = read(os.path.join(DEV_TREE, "web/src/routes/Settings.tsx")) or ""
check("V1 版本探针（Settings 显示 build_id）", "build_id" in settings or "buildId" in settings)

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
    print(f"GATE12 FAIL: {len(errors)} 项断言失败")
    sys.exit(1)
print("GATE12 PASS: SW 立即接管 + 侧栏视口 clamp + 版本探针断言通过（构建绿 + 生产零触碰）")
sys.exit(0)
