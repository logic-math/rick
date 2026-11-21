#!/usr/bin/env python3
"""gate2 — job_56 L2 层门禁：adjust 7 步 subagent 协议（task3）。

跑法：python3 .rick/jobs/job_56/plan/gates/gate2.py（仓库根）
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

PY_TESTS = ["super.tests.test_loop", "super.tests.test_prompts"]
STEP_ROLES = [
    "adjust_fact",          # step1 问题还原
    "adjust_perspective",   # step2 视角生成
    "adjust_contradiction", # step3 矛盾生成（每视角并行）
    "adjust_judge",         # step4 主要矛盾判断
    "adjust_reversal",      # step5 辩证逆转
    "adjust_space",         # step6 解空间自改进
    "adjust_env",           # step7 实验自改进
]


def sh(cmd, cwd=ROOT, timeout=900):
    r = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)
    return r.returncode, r.stdout, r.stderr


def main():
    errs = []

    def fail(msg):
        errs.append(msg)

    # ① py 模块集成测试（含 7 步流水线断言——test_loop 由 task3 扩展）
    for mod in PY_TESTS:
        rc, out, errtxt = sh([sys.executable, "-m", "unittest", mod, "-v"])
        if rc != 0:
            fail("unittest %s 失败:\n%s\n%s" % (mod, out[-2500:], errtxt[-1500:]))
        else:
            print("[PASS] unittest %s" % mod)

    # ② 7 角色模板存在 + 输出契约
    for role in STEP_ROLES:
        p = os.path.join(ROOT, "super/prompts/%s.md" % role)
        if not os.path.isfile(p):
            fail("模板缺失：super/prompts/%s.md" % role)
            continue
        s = open(p, encoding="utf-8").read()
        if not re.search(r"输出契约|输出 JSON|JSON 代码块", s):
            fail("%s.md 缺输出契约节" % role)
        else:
            print("[PASS] 模板 %s.md 含输出契约" % role)

    # ③ prompts.py ROLES 扩展（7 新角色）
    pp = os.path.join(ROOT, "super/superopt/prompts.py")
    ps = open(pp, encoding="utf-8").read()
    missing = [r for r in STEP_ROLES if '"%s"' % r not in ps]
    if missing:
        fail("prompts.py ROLES 缺角色：%s" % missing)
    else:
        print("[PASS] prompts.py ROLES 含 7 个 adjust 步骤角色")

    # ④ loop.py 流水线结构（7 步串行 + step3 并行 + 失败降级）
    lp = os.path.join(ROOT, "super/superopt/loop.py")
    ls = open(lp, encoding="utf-8").read()
    ok_pipeline = ("adjust_fact" in ls and "adjust_env" in ls)
    check4 = [
        ("7 步角色派发引用（adjust_fact..adjust_env）", ok_pipeline),
        ("step3 并行（线程/ThreadPool + adjust_contradiction）",
         "adjust_contradiction" in ls and
         ("ThreadPool" in ls or "threading.Thread" in ls)),
        ("失败降级（adjust_step_degraded 或等价跳过+默认值记账）",
         re.search(r"adjust_step_degraded|step_degraded|降级", ls) is not None),
        ("humans 表消费（intervention 未应用注入 step1）",
         re.search(r"queue_type|intervention", ls) is not None),
        ("超参数派发（agent_hyperparams/thinking/model 合并链）",
         re.search(r"agent_hyperparams|thinking", ls) is not None),
        ("decisions schema（primary_contradiction 或 schema 内嵌）",
         re.search(r"primary_contradiction|schema", ls) is not None),
        ("事后验证（hindsight/事后验证注入）",
         re.search(r"hindsight|事后验证", ls) is not None),
    ]
    for name, ok in check4:
        if ok:
            print("[PASS] loop.py %s" % name)
        else:
            fail("loop.py 缺 %s" % name)

    # ⑤ 旧路径删除确认（job_56 第二轮：legacy 单 agent 路径已删——旧模板不存在+无 adjust_mode 分支）
    legacy_gone = not os.path.exists(os.path.join(ROOT, "super/prompts/adjust.md"))
    check5 = [
        ("旧 adjust.md 模板已删除（legacy 路径退役）", legacy_gone),
        ("loop.py 无 legacy 单 agent 分支（render(\"adjust\") 调用清零）",
         'render("adjust"' not in ls),
    ]
    for name, ok in check5:
        if ok:
            print("[PASS] %s" % name)
        else:
            fail("缺 %s" % name)

    # ⑥ json-guard hook 扩展（schema 检 + gates 拦截）
    hg = os.path.join(ROOT, ".rick/skills/superopt-json-guard/index.ts")
    if not os.path.isfile(hg):
        fail("json-guard index.ts 缺失（.rick/skills/superopt-json-guard/）")
    else:
        hs = open(hg, encoding="utf-8").read()
        for name, ok in [
            ("schema 字段校验（字段级检查）",
             re.search(r"orthogonality|target_cell|schema|字段", hs) is not None),
            ("gates/ 禁读拦截（builder/coding/debug 角色拦截）",
             re.search(r"gates", hs) is not None and
             re.search(r"builder|coding|debug", hs) is not None),
            ("角色判定（SUPEROPT_ROLE 或等价机制）",
             re.search(r"SUPEROPT_ROLE|role", hs) is not None),
        ]:
            if ok:
                print("[PASS] json-guard %s" % name)
            else:
                fail("json-guard 缺 %s" % name)

    print()
    if errs:
        print("=" * 60)
        print("gate2 失败（%d 项）：" % len(errs))
        for e in errs:
            print(" - %s" % e)
        return 1
    print("gate2 通过：adjust 7 步 subagent 协议（task3）层验收 OK")
    return 0


if __name__ == "__main__":
    sys.exit(main())
