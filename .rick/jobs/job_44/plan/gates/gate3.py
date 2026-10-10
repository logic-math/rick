#!/usr/bin/env python3
"""gate3.py — job_44 第 3 层门禁（task4: dev 实例验证 + RSI 证据 S1-S3）

模块级集成测试（human 确认，agent 不得修改）：
  A. dev 实例健康：GET http://127.0.0.1:8414/api/health → status=ok 且 build_id 非空
  B. 前端产物：dev 树 web/dist/index.html 存在
  C. RSI 证据：doing/rsi/{dev-iterations,gates,release,resume}.md 存在且关键行齐备
     （approval.md 的 APPROVED 行属 S4 human 放行，不在本门禁断言内）
  D. 演练：release --dry-run 记录 prod_untouched=true
  E. 生产零触碰：生产仓库 tracked 文件无改动

输出 JSON {"pass": bool, "errors": [...]}，exit 0=绿 / 1=红。
"""
import json
import os
import re
import subprocess
import sys
import urllib.request


def get_project_root():
    p = os.path.abspath(__file__)
    for _ in range(6):
        p = os.path.dirname(p)
    return p


ROOT = get_project_root()
JOB = os.path.join(ROOT, ".rick", "jobs", "job_44")
ERRORS = []


def read(path):
    with open(path, encoding="utf-8") as f:
        return f.read()


def main():
    # ---- A. dev 实例健康 ----
    health = None
    try:
        with urllib.request.urlopen("http://127.0.0.1:8414/api/health", timeout=5) as resp:
            health = json.loads(resp.read().decode("utf-8"))
    except Exception as e:  # noqa: BLE001
        ERRORS.append(f"[A] dev 实例健康探测失败（8414/api/health）：{e}")
    if health is not None:
        if health.get("status") != "ok":
            ERRORS.append(f"[A] dev 实例 status != ok：{health.get('status')!r}")
        if not health.get("build_id"):
            ERRORS.append("[A] dev 实例 build_id 为空（未换新构建？）")

    # ---- B. 前端产物 ----
    if not os.path.exists(os.path.join(ROOT, "web", "dist", "index.html")):
        ERRORS.append("[B] web/dist/index.html 缺失（前端未构建）")

    # ---- C. RSI 证据 ----
    rsi = os.path.join(JOB, "doing", "rsi")
    ev = {}
    for name in ("dev-iterations.md", "gates.md", "release.md", "resume.md"):
        p = os.path.join(rsi, name)
        if not os.path.exists(p):
            ERRORS.append(f"[C] 证据文件缺失：doing/rsi/{name}")
        else:
            ev[name] = read(p)
    if "dev-iterations.md" in ev and not re.search(r"build_id\s*=\s*\S+|rick\.dev\.[0-9a-f]{7}", ev["dev-iterations.md"]):
        ERRORS.append("[C] dev-iterations.md 未见构建指纹（build_id=... 或 rick.dev.<sha7>-<ts>）")
    if "gates.md" in ev:
        for g in ("gate1", "gate2"):
            if not re.search(rf"GATE\s+{g}\s+pass=true", ev["gates.md"]):
                ERRORS.append(f"[C] gates.md 未见 `GATE {g} pass=true`")
        if re.search(r"pass=false", ev["gates.md"]):
            ERRORS.append("[C] gates.md 含 pass=false（门禁未全绿）")
    if "release.md" in ev and "prod_untouched=true" not in ev["release.md"]:
        ERRORS.append("[C] release.md 未见 prod_untouched=true（dry-run 未记录）")

    # ---- D. 演练数据（dry-run 实际输出，独立于证据文件）----
    r = subprocess.run(
        ["go", "run", "./cmd/rick", "tools", "release", "--dry-run"],
        cwd=ROOT, capture_output=True, text=True, timeout=1200,
    )
    out = r.stdout + r.stderr
    if "prod_untouched=true" not in out:
        ERRORS.append("[D] `rick tools release --dry-run` 输出未见 prod_untouched=true：\n" + out[-1200:])
    for key in ("version=", "rollback_point="):
        if key not in out:
            ERRORS.append(f"[D] dry-run 输出缺 {key}")

    # ---- E. 生产零触碰 ----
    wt = subprocess.run(["git", "worktree", "list", "--porcelain"], cwd=ROOT,
                        capture_output=True, text=True)
    prod = None
    for line in wt.stdout.splitlines():
        if line.startswith("worktree ") and "rick-dev" not in line:
            prod = line.split(" ", 1)[1].strip()
            break
    if prod and os.path.isdir(prod):
        r = subprocess.run(["git", "status", "--porcelain", "-uno"], cwd=prod,
                           capture_output=True, text=True)
        if r.stdout.strip():
            ERRORS.append(f"[E][RSI] 生产仓库 tracked 文件被改动：\n{r.stdout.strip()[:500]}")
    else:
        ERRORS.append("[E][RSI] 未能定位生产仓库（worktree list 解析失败）")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)


if __name__ == "__main__":
    main()
