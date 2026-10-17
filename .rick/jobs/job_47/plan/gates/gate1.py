#!/usr/bin/env python3
"""gate1.py — job_47 第 1 层门禁（Kahn L1：task1/2/3/4 py 基座 + task10 Go PySupervisor + task12 loop/skill 资产 + task13 matmul env 模板）

模块级集成测试（human 确认，agent 不得修改）：
  A. py 单元测试全绿：super/tests 的 test_events/test_archive/test_feature_space/
     test_sampler/test_config/test_budget/test_lineage（unittest discover）
  B. Go 构建与测试：go build ./... + go test ./internal/super/...（PySupervisor fake-py 用例）
  C. 知识资产契约：super-loop.md frontmatter（name/trigger/scope）+ 四子命令引用 + 干预走队列；
     gen_env skill 三段式 + 门禁系统设计引导环节
  D. matmul env 模板完整性：8 文件齐 + env.json/feature_space.json 可解析 + 2 轴正交命名 +
     venv 中 test_correctness 对参考实现 passed=true、对假解 passed=false；bench 出分
  E. Python ≥3.9 兼容静态检查：super/superopt 无 `X | Y` 联合类型注解（PEP604 语法）
  F. 生产零触碰：生产仓库 tracked 文件无改动

输出 JSON {"pass": bool, "errors": [...]}，exit 0=绿 / 1=红。
"""
import json
import os
import re
import subprocess
import sys
import tempfile


def get_project_root():
    # .rick/jobs/job_47/plan/gates/gate1.py → 6×dirname 到仓库根
    p = os.path.abspath(__file__)
    for _ in range(6):
        p = os.path.dirname(p)
    return p


ROOT = get_project_root()
ERRORS = []


def run(cmd, cwd=ROOT, timeout=900):
    return subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)


def read(path):
    with open(os.path.join(ROOT, path), encoding="utf-8") as f:
        return f.read()


def exists(path):
    return os.path.exists(os.path.join(ROOT, path))


def check(cond, msg):
    if not cond:
        ERRORS.append(msg)


def main():
    # ---- A. py 单元测试 ----
    r = run([sys.executable, "-m", "unittest", "discover", "-s", "super/tests",
             "-p", "test_*.py", "-v"])
    if r.returncode != 0:
        ERRORS.append(f"[A] py 单测失败：\n{(r.stdout + r.stderr)[-2000:]}")
    else:
        for name in ["test_events", "test_archive", "test_feature_space",
                     "test_sampler", "test_config", "test_budget", "test_lineage"]:
            check(exists(f"super/tests/{name}.py"), f"[A] 缺 {name}.py")

    # ---- B. Go 构建与 PySupervisor 测试 ----
    r = run(["go", "build", "./..."])
    if r.returncode != 0:
        ERRORS.append(f"[B] go build 失败：\n{(r.stdout + r.stderr)[-1500:]}")
    else:
        r = run(["go", "test", "./internal/super/...", "-timeout", "120s"])
        if r.returncode != 0:
            ERRORS.append(f"[B] go test internal/super 失败：\n{(r.stdout + r.stderr)[-1500:]}")
        else:
            check("ok" in r.stdout, "[B] go test internal/super 无 ok 输出（异常）")

    # ---- C. 知识资产契约 ----
    if not exists(".rick/loops/super-loop.md"):
        ERRORS.append("[C] 缺 .rick/loops/super-loop.md")
    else:
        head = read(".rick/loops/super-loop.md")
        check("name: super-loop" in head, "[C] super-loop frontmatter 缺 name")
        check("trigger:" in head and "super" in head, "[C] super-loop 缺 trigger")
        check("scope:" in head, "[C] super-loop 缺 scope")
        for kw in ["rick super start", "rick super stop", "rick super status",
                   "rick super event", "干预队列"]:
            check(kw in head, f"[C] super-loop 缺关键词 {kw}")
        check("门先证红" in head, "[C] super-loop 缺门先证红环节")
    if not exists(".rick/skills/gen_env_skill/skill.md"):
        ERRORS.append("[C] 缺 .rick/skills/gen_env_skill/skill.md")
    else:
        s = read(".rick/skills/gen_env_skill/skill.md")
        for kw in ["触发场景", "预期效果", "核心内容", "门禁系统设计引导",
                   "feature_space.json", "test_correctness", "exploit"]:
            check(kw in s, f"[C] gen_env skill 缺关键词 {kw}")

    # ---- D. matmul env 模板 ----
    tpl = "super/env_templates/matmul"
    for f in ["env.json", "feature_space.json", "code/reference.py",
              "code/naive_start.py", "data/test_correctness.py", "data/bench.py",
              "skills/self_check.py", "domain_knowledge.md", "requirements.txt"]:
        check(exists(f"{tpl}/{f}"), f"[D] matmul 模板缺 {f}")
    if exists(f"{tpl}/env.json"):
        try:
            env = json.loads(read(f"{tpl}/env.json"))
            check(env.get("perf_desc") in ("low", "high"), "[D] env.json perf_desc 非法")
            check("runtime_ms" == env.get("perf_type"), "[D] env.json perf_type 应为 runtime_ms")
        except Exception as e:
            ERRORS.append(f"[D] env.json 解析失败：{e}")
    if exists(f"{tpl}/feature_space.json"):
        try:
            fs = json.loads(read(f"{tpl}/feature_space.json"))
            axes = fs.get("axes", [])
            check(len(axes) >= 2, "[D] feature_space 轴数 < 2")
            names = [a.get("name") for a in axes]
            check(len(names) == len(set(names)), "[D] 特征轴重名（正交性破坏）")
        except Exception as e:
            ERRORS.append(f"[D] feature_space.json 解析失败：{e}")
    # 模板自检（venv numpy）：参考实现过正确性门；假解被拒；bench 出分
    with tempfile.TemporaryDirectory() as td:
        venv_py = os.path.join(td, "venv", "bin", "python")
        r = run([sys.executable, "-m", "venv", os.path.join(td, "venv")])
        if r.returncode != 0:
            ERRORS.append(f"[D] venv 创建失败：{r.stderr[-300:]}")
        else:
            r = run([venv_py, "-m", "pip", "install", "-q", "numpy"], timeout=600)
            if r.returncode != 0:
                ERRORS.append(f"[D] numpy 安装失败（网络？）：{r.stderr[-300:]}")
            else:
                # 参考实现过门
                r = run([venv_py, f"{tpl}/data/test_correctness.py",
                         "--candidate", f"{tpl}/code/reference.py"], timeout=120)
                try:
                    out = json.loads(r.stdout.strip().splitlines()[-1])
                    check(out.get("passed") is True,
                          f"[D] 参考实现未过正确性门：{r.stdout[-200:]}{r.stderr[-200:]}")
                except Exception:
                    ERRORS.append(f"[D] test_correctness 输出非 JSON：{(r.stdout + r.stderr)[-300:]}")
                # 假解被拒：naive 也应过（正确但慢），构造假解（返回零矩阵）
                fake = os.path.join(td, "fake.py")
                with open(fake, "w") as f:
                    f.write("import numpy as np\ndef matmul(a,b):\n"
                            "    return np.zeros_like(a)\n")
                r = run([venv_py, f"{tpl}/data/test_correctness.py", "--candidate", fake],
                        timeout=120)
                try:
                    out = json.loads(r.stdout.strip().splitlines()[-1])
                    check(out.get("passed") is False, "[D] 假解（零矩阵）未被正确性门拒绝——门无判别力")
                except Exception:
                    ERRORS.append(f"[D] 假解门输出非 JSON：{(r.stdout + r.stderr)[-300:]}")
                # bench 出分
                r = run([venv_py, f"{tpl}/data/bench.py",
                         "--candidate", f"{tpl}/code/reference.py"], timeout=300)
                try:
                    out = json.loads(r.stdout.strip().splitlines()[-1])
                    check(out.get("runtime_ms", 0) > 0, "[D] bench 无 runtime_ms")
                    check(len(out.get("timings", [])) >= 3, "[D] bench 计时次数 < 3（K 中位数协议）")
                except Exception:
                    ERRORS.append(f"[D] bench 输出非 JSON：{(r.stdout + r.stderr)[-300:]}")

    # ---- E. Python 3.9 兼容静态检查 ----
    import glob
    for p in glob.glob(os.path.join(ROOT, "super/superopt", "*.py")) + \
             glob.glob(os.path.join(ROOT, "super/tests", "*.py")):
        src = open(p, encoding="utf-8").read()
        # PEP 604: "-> list[str] | None" / "x: int | None"
        bad = re.findall(r"->[^=\n]*\|\s*None|:\s*\w+\s*\|\s*None", src)
        if bad:
            ERRORS.append(f"[E] {os.path.relpath(p, ROOT)} 使用 PEP604 联合类型（需 ≥3.10）：{bad[:2]}")

    # ---- F. 生产零触碰 ----
    wt = run(["git", "worktree", "list", "--porcelain"])
    prod = None
    for line in wt.stdout.splitlines():
        if line.startswith("worktree ") and "rick-dev" not in line:
            prod = line.split(" ", 1)[1].strip()
            break
    if prod and os.path.isdir(prod):
        r = run(["git", "status", "--porcelain", "-uno"], cwd=prod)
        if r.stdout.strip():
            ERRORS.append(f"[F][RSI] 生产仓库 tracked 文件被改动：\n{r.stdout.strip()[:500]}")
    else:
        ERRORS.append("[F][RSI] 未能定位生产仓库")

    print(json.dumps({"pass": len(ERRORS) == 0, "errors": ERRORS}, ensure_ascii=False, indent=2))
    sys.exit(0 if not ERRORS else 1)


if __name__ == "__main__":
    main()
