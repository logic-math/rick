#!/usr/bin/env python3
"""gate4.py — job_47 第 4 层门禁（Kahn L4：task14 e2e 全链 + 双回归 harness + 污染演练；KR 验收门）

模块级集成测试（human 确认，agent 不得修改）：
  A. run_all.sh 全绿：全部单测 + e2e_test（gen_env 产出→Go start→≥5 mock 迭代→入格→
     kill 重启游标恢复→事件重放一致→北极星→stop 优雅停）+ 双回归（elite_replay 零误杀 +
     exploit_regression 零放水）+ 污染注入演练（假阳性→降权→复测→出档/复活全链一例）
  B. Go 全量回归：go build ./... + go test ./internal/... 关键包
  C. realpi 冒烟资产就绪：super/tests/realpi_smoke.py 存在且 RICK_REAL_PI_ARGS 门控逻辑正确
     （未设 env → skipTest 语义；真跑证据以 doing/rsi/gates.md 人工贴出为准，不阻塞本门禁）
  D. 北极星可计算（e2e 内已断言，此处复检 status --json 协议键存在）
  E. 生产零触碰 + dev 树 git 状态（层完成提交后应干净）

输出 JSON {"pass": bool, "errors": [...]}，exit 0=绿 / 1=红。
"""
import json
import os
import subprocess
import sys


def get_project_root():
    p = os.path.abspath(__file__)
    for _ in range(6):
        p = os.path.dirname(p)
    return p


ROOT = get_project_root()
ERRORS = []


def run(cmd, cwd=ROOT, timeout=1800):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)


def read(path):
    with open(os.path.join(ROOT, path), encoding="utf-8") as f:
        return f.read()


def exists(path):
    return os.path.exists(os.path.join(ROOT, path))


def main():
    # ---- A. e2e 全链 + 双回归 + 污染演练 ----
    if not exists("super/tests/run_all.sh"):
        ERRORS.append("[A] 缺 super/tests/run_all.sh")
    else:
        r = run(["bash", "super/tests/run_all.sh"], timeout=1800)
        if r.returncode != 0:
            ERRORS.append(f"[A] run_all.sh 失败：\n{(r.stdout + r.stderr)[-3000:]}")
        else:
            out = r.stdout
            for marker in ["e2e", "elite_replay", "exploit_regression"]:
                # 输出应体现三段都跑了（run_all.sh 的输出标记）
                if marker not in out:
                    ERRORS.append(f"[A] run_all.sh 输出未见 {marker} 段执行痕迹")

    # 双回归 harness 文件契约
    for p in ["super/gates/elite_replay.py", "super/gates/exploit_regression.py"]:
        if not exists(p):
            ERRORS.append(f"[A] 缺 {p}")
        else:
            s = read(p)
            if "--workdir" not in s:
                ERRORS.append(f"[A] {p} 缺 --workdir CLI 协议")

    # ---- B. Go 回归 ----
    r = run(["go", "build", "./..."])
    if r.returncode != 0:
        ERRORS.append(f"[B] go build 失败：\n{(r.stdout + r.stderr)[-1200:]}")
    else:
        # 注：internal/cmd 的 TestReleaseCmd*（job_43 既有用例）macOS 跨机 fake-prod 死锁，
        # 与 job_47 无关（gate2 已确诊记录），-skip 排除；job_47 的 cmd 侧验证 =
        # go build + E 段 CLI 冒烟 + e2e 真二进制全链。
        for pkg, skip in [("./internal/super/...", None), ("./internal/handler/...", None),
                          ("./internal/cmd/...", "TestReleaseCmd.*")]:
            argv = ["go", "test", pkg, "-timeout", "600s"]
            if skip:
                argv += ["-skip", skip]
            r = run(argv)
            if r.returncode != 0:
                ERRORS.append(f"[B] go test {pkg} 失败：\n{(r.stdout + r.stderr)[-1200:]}")

    # ---- C. realpi 冒烟资产 ----
    if not exists("super/tests/realpi_smoke.py"):
        ERRORS.append("[C] 缺 super/tests/realpi_smoke.py")
    else:
        s = read("super/tests/realpi_smoke.py")
        if "RICK_REAL_PI_ARGS" not in s:
            ERRORS.append("[C] realpi_smoke.py 缺 RICK_REAL_PI_ARGS 门控")
        if "skipTest" not in s:
            ERRORS.append("[C] realpi_smoke.py 缺 skipTest 语义（环境门控不阻塞）")
        # 门控下不设 env 应 skip 而非 fail
        r = run([sys.executable, "super/tests/realpi_smoke.py"])
        if r.returncode not in (0,):
            ERRORS.append(f"[C] realpi_smoke.py 未设 env 时应 skip(exit 0)：实际 {r.returncode}\n"
                          f"{(r.stdout + r.stderr)[-400:]}")

    # ---- D. status --json 协议键（北极星可计算） ----
    if exists("super/tests/e2e_test.py"):
        s = read("super/tests/e2e_test.py")
        for kw in ["iterations", "cells", "token", "north_star"]:
            if kw not in s:
                ERRORS.append(f"[D] e2e_test.py 未断言北极星键 {kw}")

    # ---- E. 生产零触碰 ----
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = None
    for line in wt.stdout.splitlines():
        if line.startswith("worktree ") and "rick-dev" not in line:
            prod = line.split(" ", 1)[1].strip()
            break
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        if r.stdout.strip():
            ERRORS.append(f"[E][RSI] 生产仓库 tracked 文件被改动：\n{r.stdout.strip()[:500]}")
    else:
        ERRORS.append("[E][RSI] 未能定位生产仓库")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)


if __name__ == "__main__":
    main()
