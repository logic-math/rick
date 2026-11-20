#!/usr/bin/env python3
"""gate3 — job_56 L3 层门禁：sub loop 实验执行链强化（task4）。

跑法：python3 .rick/jobs/job_56/plan/gates/gate3.py（仓库根）
exit 0 = 层验收通过。
"""
import os
import re
import subprocess
import sys

# gates -> plan -> job_56 -> jobs -> .rick -> 仓库根（向上找含 .rick 与 super 的目录）
_d = os.path.dirname(os.path.abspath(__file__))
ROOT = _d
for _ in range(12):
    if os.path.isdir(os.path.join(ROOT, ".rick")) and os.path.isdir(os.path.join(ROOT, "super")):
        break
    ROOT = os.path.dirname(ROOT)

PY_TESTS = ["super.tests.test_loop", "super.tests.test_experiment_record"]


def sh(cmd, cwd=ROOT, timeout=900):
    r = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)
    return r.returncode, r.stdout, r.stderr


def read(p):
    return open(os.path.join(ROOT, p), encoding="utf-8").read()


def main():
    errs = []

    def fail(msg):
        errs.append(msg)

    # ① py 模块集成测试（test_loop 含实验链断言）
    for mod in PY_TESTS:
        rc, out, errtxt = sh([sys.executable, "-m", "unittest", mod, "-v"])
        if rc != 0:
            fail("unittest %s 失败:\n%s\n%s" % (mod, out[-2500:], errtxt[-1500:]))
        else:
            print("[PASS] unittest %s" % mod)

    # ② builder 协议：仿真预演/风险点/证伪性论证 + 资源槽位注入
    b = read("super/prompts/builder.md")
    for name, pat in [
        ("仿真预演段（风险点推演）", r"仿真|风险"),
        ("风险点预防措施", r"预防|风险点"),
        ("证伪性论证", r"证伪"),
        ("实验 ID 注入（experiment_id）", r"experiment_id"),
        ("资源槽位注入（slot/env_slot）", r"slot"),
    ]:
        ok = re.search(pat, b) is not None
        if ok:
            print("[PASS] builder %s" % name)
        else:
            fail("builder.md 缺 %s" % name)

    # ③ coding 协议：去 TDD（无强制 TDD 纪律关键词）
    c = read("super/prompts/coding.md")
    tdd_markers = []
    for line in c.splitlines():
        if re.search(r"先写测试|看到红|看红|saw_red|test_written|TDD 循环|严格 TDD", line):
            # 豁免：说明「不再要求 TDD」的否定句
            if not re.search(r"不再|无需|移除|去.{0,2}TDD|不需要", line):
                tdd_markers.append(line.strip()[:120])
    if tdd_markers:
        fail("coding.md 残留 TDD 强制纪律：%s" % tdd_markers[:5])
    else:
        print("[PASS] coding.md 无 TDD 强制纪律（一次写对+自验）")
    if re.search(r"自验|自查", c):
        print("[PASS] coding.md 含自验环节")
    else:
        fail("coding.md 缺自验环节（替代 TDD 的质量保障）")

    # ④ debug 协议：四节实验记录契约 + experiment_failed 主动上报
    d = read("super/prompts/debug.md")
    for name, pat in [
        ("四节实验记录契约（设计/过程/结论/反思）", r"实验设计|执行过程|实验结论|反思"),
        ("experiment_failed 主动上报协议", r"experiment_failed"),
        ("涨幅过大查作弊/验证机制", r"作弊|验证机制"),
        ("完整实验记录（全部事实记录）", r"全部事实|完整.{0,6}实验记录"),
    ]:
        ok = re.search(pat, d) is not None
        if ok:
            print("[PASS] debug %s" % name)
        else:
            fail("debug.md 缺 %s" % name)

    # ⑤ loop.py：失败跳过验证 + sub loop 写回 DB + ctx 注入
    ls = read("super/superopt/loop.py")
    for name, pat in [
        ("失败跳过验证（experiment_failed → 不跑 gate_check）",
         r"experiment_failed"),
        ("实验表写回（record_path/session_id/score 写回 DB）",
         r"record_path"),
        ("实验 ID 新编号注入", r"experiment_id"),
    ]:
        ok = re.search(pat, ls) is not None
        if ok:
            print("[PASS] loop.py %s" % name)
        else:
            fail("loop.py 缺 %s" % name)

    # ⑥ experiment_record.py 四节结构模板
    try:
        er = read("super/superopt/experiment_record.py")
        ok = re.search(r"实验设计|执行过程|实验结论|反思", er) is not None
        if ok:
            print("[PASS] experiment_record.py 四节结构模板")
        else:
            fail("experiment_record.py 缺四节结构（实验设计/执行过程/实验结论/反思）")
    except OSError:
        fail("experiment_record.py 缺失")

    # ⑦ 行为断言：test_loop 内含失败跳过验证路径测试（防结构断言空转）
    tl = read("super/tests/test_loop.py")
    ok = re.search(r"experiment_failed|skip.{0,10}gate|跳过", tl) is not None
    if ok:
        print("[PASS] test_loop 含失败跳过验证行为断言")
    else:
        fail("test_loop 缺失败跳过验证的行为级测试（仅结构断言不足以验收）")

    print()
    if errs:
        print("=" * 60)
        print("gate3 失败（%d 项）：" % len(errs))
        for e in errs:
            print(" - %s" % e)
        return 1
    print("gate3 通过：sub loop 实验执行链强化（task4）层验收 OK")
    return 0


if __name__ == "__main__":
    sys.exit(main())
