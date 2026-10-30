#!/usr/bin/env python3
"""gate1.py — job_49 S2 层门禁 1：代码修复层（P0-1/2/3/4 + P1-2 源码）

模块级集成测试（human 裁决范围 J1=C 的代码面，agent 不得修改本文件）：
  A. dev 树编译：go build ./...
  B. P0 核心用例全绿（判别力核心——修复前红、修复后绿）：
     - TestPromoteRejectsNonExecutableTarget（P0-1）
     - TestRollbackRejectsNonExecutableTarget（P0-2）
     - TestReleaseAutorollbackRefusesBadRollbackTarget（P0-2 自动回滚路径，11:53 事故复现）
     - TestStopProdClaimsBareBuildHoldingLock（P0-3 持锁认领，11135 场景）
     - TestStopProdClaimsPortSquatterRickProcess（P0-3 端口认领）
     - TestRestartRefusesToKillForeignProcess（P0-3 不误杀安全线）
     - TestRestartVerifiesBuildIDAndReplacesProcess（既有回归）
     - TestRestartFailsWhenRunningBuildIsStale（既有回归）
     - TestReleaseAutoRollsBackWhenHealthFails（既有回归）
     - TestRollbackRestoresPreviousVersion / TestRollbackWithoutRollbackPointFails（既有回归）
  C. P0-4 诊断字段：SkippedWhy 字段 + 两态文案存在（grep 源码断言）
  D. P1-2 源码：import.go mergeState 写入前 wrongPlatformBinary 校验 + 用例绿
  E. procfs.Command 接口在位（三级名义认定的兜底）
  F. 全量回归：go test ./...（env 清洗 RICK_*，防 TestWebSingleton 读生产状态假失败）
  G. 生产零触碰：main 树源码（cmd/internal/web）tracked 无改动；6137/8414 状态不变

输出 JSON {"pass": bool, "errors": [...]}，exit 0=绿 / 1=红。
"""
import json
import os
import subprocess
import sys

DEV_TREE = "/Users/logic/workdir/rick-dev"
MAIN_TREE = "/Users/logic/workdir/AI_CODING/rick"
ERRORS = []


def run(cmd, cwd=DEV_TREE, timeout=900, env_extra=None):
    env = dict(os.environ)
    for k in ("RICK_STATE_DIR", "RICK_PI_AGENT_DIR", "PI_CODING_AGENT_DIR"):
        env.pop(k, None)  # env 清洗：防 TestWebSingleton 读生产状态假失败（domain/bugs.md）
    if env_extra:
        env.update(env_extra)
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout, env=env)


def check(cond, msg):
    if not cond:
        ERRORS.append(msg)


def main():
    # ---- A. dev 树编译 ----
    r = run(["go", "build", "./..."])
    check(r.returncode == 0, f"[A] go build 失败:\n{(r.stdout + r.stderr)[-1500:]}")

    # ---- B. P0 核心用例 ----
    r = run(["go", "test", "./internal/env/release/", "-count=1", "-v", "-run",
             "TestPromoteRejectsNonExecutableTarget|"
             "TestRollbackRejectsNonExecutableTarget|"
             "TestReleaseAutorollbackRefusesBadRollbackTarget|"
             "TestStopProdClaimsBareBuildHoldingLock|"
             "TestStopProdClaimsPortSquatterRickProcess|"
             "TestRestartRefusesToKillForeignProcess|"
             "TestRestartVerifiesBuildIDAndReplacesProcess|"
             "TestRestartFailsWhenRunningBuildIsStale|"
             "TestReleaseAutoRollsBackWhenHealthFails|"
             "TestRollbackRestoresPreviousVersion|"
             "TestRollbackWithoutRollbackPointFails"],
            timeout=600)
    out = r.stdout + r.stderr
    check(r.returncode == 0, f"[B] P0 核心用例失败:\n{out[-2500:]}")
    check("FAIL" not in out, "[B] P0 用例输出含 FAIL")

    # ---- C. P0-4 诊断字段 ----
    rel = open(os.path.join(DEV_TREE, "internal/env/release/release.go"), encoding="utf-8").read()
    check("SkippedWhy" in rel, "[C] SkippedWhy 诊断字段缺失（P0-4）")
    check("被 pid" in rel or "占用" in rel, "[C] 端口占用诊断文案缺失（P0-4）")
    check("verifyRunnable" in rel, "[C] verifyRunnable 缺失（P0-1/P0-2）")
    check("claimBySingleton" in rel, "[C] claimBySingleton 缺失（P0-3）")

    # ---- D. P1-2 源码校验 ----
    imp = open(os.path.join(DEV_TREE, "internal/env/relocate/import.go"), encoding="utf-8").read()
    check("wrongPlatformBinary" in imp or "platformOK" in imp or "verifyRunnable" in imp,
          "[D] import.go 写入路径平台校验缺失（P1-2）")
    r = run(["go", "test", "./internal/env/relocate/", "-count=1"])
    check(r.returncode == 0, f"[D] relocate 包红:\n{(r.stdout + r.stderr)[-1200:]}")

    # ---- E. procfs.Command ----
    pf = open(os.path.join(DEV_TREE, "internal/env/procfs/procfs.go"), encoding="utf-8").read()
    check("Command(pid int)" in pf, "[E] procfs.Command 接口缺失（三级名义兜底）")
    r = run(["go", "test", "./internal/env/procfs/", "-count=1"])
    check(r.returncode == 0, f"[E] procfs 包红:\n{(r.stdout + r.stderr)[-800:]}")

    # ---- F. 全量回归 ----
    r = run(["go", "test", "./...", "-count=1"], timeout=1800)
    out = r.stdout + r.stderr
    check(r.returncode == 0, f"[F] 全量回归失败:\n{out[-2500:]}")

    # ---- G. 生产零触碰 ----
    # main 树源码（cmd/internal/web）tracked 无改动
    r = run(["git", "-C", MAIN_TREE, "status", "--porcelain", "--", "cmd", "internal", "web"], cwd=MAIN_TREE)
    check(r.stdout.strip() == "", f"[G] main 树源码被改动:\n{r.stdout}")
    # 6137 演练前状态 = 无监听（生产死亡是修复对象，门禁不拉起它）
    r = run(["bash", "-c", "lsof -nP -iTCP:6137 -sTCP:LISTEN | wc -l"], cwd=MAIN_TREE)
    check(r.stdout.strip() == "0", "[G] 6137 被拉起了（生产零触碰被违反——应保持死亡直到 S5）")
    # 8414 dev 实例在线不变
    r = run(["bash", "-c", "curl -s -m 5 -o /dev/null -w '%{http_code}' http://127.0.0.1:8414/api/health"], cwd=MAIN_TREE)
    check(r.stdout.strip() == "200", f"[G] 8414 dev 实例异常（health={r.stdout.strip()}）")

    ok = len(ERRORS) == 0
    print(json.dumps({"pass": ok, "gate": "gate1", "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
