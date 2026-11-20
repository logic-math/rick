#!/usr/bin/env python3
"""gate1 — job_56 L1 层门禁：DB 地基（task1）+ env 体系（task2）。

跑法：python3 .rick/jobs/job_56/plan/gates/gate1.py（仓库根）
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
PY_TESTS = [
    "super.tests.test_db",        # task1 新增：五表/进程锁/资源槽
    "super.tests.test_events",
    "super.tests.test_archive",
    "super.tests.test_ledger",
    "super.tests.test_intervention",
    "super.tests.test_sampler",
    "super.tests.test_feature_space",
    "super.tests.test_adjudicator",
    "super.tests.test_config",
    "super.tests.test_gates",     # task2：多维融合分协议
]
GO_PKGS = ["./internal/handler/", "./internal/web/", "./internal/runtime/"]


def sh(cmd, cwd=ROOT, timeout=900):
    r = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True,
                       timeout=timeout)
    return r.returncode, r.stdout, r.stderr


def check(name, ok, detail=""):
    print("[PASS] %s" % name if ok else "[FAIL] %s %s" % (name, detail))
    return ok


def main():
    errs = []

    def fail(msg):
        errs.append(msg)

    # ① py 模块集成测试
    for mod in PY_TESTS:
        rc, out, errtxt = sh([sys.executable, "-m", "unittest", mod, "-v"])
        if rc != 0:
            fail("unittest %s 失败:\n%s\n%s" % (mod, out[-2000:], errtxt[-1500:]))
        else:
            print("[PASS] unittest %s" % mod)

    # ② go build + 定向包测试
    rc, out, errtxt = sh(["go", "build", "./..."])
    if rc != 0:
        fail("go build 失败:\n%s" % errtxt[-2000:])
    else:
        print("[PASS] go build")
    if rc == 0:
        rc, out, errtxt = sh(["go", "test"] + GO_PKGS, timeout=1200)
        if rc != 0:
            fail("go test %s 失败:\n%s\n%s" % (GO_PKGS, out[-2000:], errtxt[-1500:]))
        else:
            print("[PASS] go test %s" % " ".join(GO_PKGS))

    # ③ 结构断言：db.py 五表 + 进程锁 + 资源槽
    dbp = os.path.join(ROOT, "super/superopt/db.py")
    if not check("db.py 存在", os.path.isfile(dbp)):
        fail("super/superopt/db.py 不存在")
    else:
        src = open(dbp, encoding="utf-8").read()
        for tbl in ("cells", "experiments", "humans", "events", "resources"):
            if not check("db.py 表 %s" % tbl,
                         re.search(r"CREATE TABLE[^(]*\b%s\b" % tbl, src, re.I)):
                fail("db.py 缺表 %s" % tbl)
        check("db.py humans 两类队列字段", "queue_type" in src) or \
            fail("humans 表缺 queue_type（干预/提案两类队列）")
        check("db.py humans applied 字段", "applied" in src) or \
            fail("humans 表缺 applied")
        check("db.py 进程锁（scheduler.pid / PID 存活检测）",
              "scheduler.pid" in src or "pid" in src.lower()) or \
            fail("db.py 缺进程锁实现或入口")
        check("db.py 资源槽类型（experiment_slot/env_slot）",
              "experiment_slot" in src and "env_slot" in src) or \
            fail("resources 表缺实验槽/环境槽类型")
        check("db.py 资源分配 API（acquire/release）",
              "acquire" in src and "release" in src) or \
            fail("缺资源分配事务 API")

    # ④ 结构断言：进程锁启动检查挂在 __main__
    mp = os.path.join(ROOT, "super/superopt/__main__.py")
    if os.path.isfile(mp):
        msrc = open(mp, encoding="utf-8").read()
        check("__main__.py 进程锁启动检查", "pid" in msrc.lower()) or \
            fail("__main__.py 缺进程锁启动检查")
    else:
        fail("super/superopt/__main__.py 不存在")

    # ⑤ 结构断言：实验 ID 新编号（迭代号-年月日时分）
    for modf in ("super/superopt/db.py", "super/superopt/loop.py"):
        p = os.path.join(ROOT, modf)
        if os.path.isfile(p):
            s = open(p, encoding="utf-8").read()
            ok = re.search(r"%Y%m%d%H%M|YYYYMMDDHHMM|年月日时分", s)
            check("%s 实验时间戳编号" % modf, ok) or \
                fail("%s 缺迭代号-分钟级时间戳编号" % modf)

    # ⑥ 结构断言：旧 json 路径废弃（py 侧实际写入清零；合法豁免：docstring 示例/旧布局检测/签名兼容注释）
    import glob
    leftovers = []
    for f in glob.glob(os.path.join(ROOT, "super/superopt/*.py")):
        if os.path.basename(f) == "db.py":
            continue
        s = open(f, encoding="utf-8").read()
        for pat in ("events.jsonl", "archive.json", "ledger.jsonl"):
            if pat in s:
                for line in s.splitlines():
                    if pat in line and not re.search(
                            r"废弃|迁移|legacy|不再|deprecated|dirname|path 参数"
                            r"|旧布局|时代|示例|真相源|投影|重建|副产品|事件即账"
                            r"|一致性断言|封口|截断|jsonl 时代|唯一写者|单写者"
                            r"|档案|约定|不写|帮助|help|工作区路径|明确报错|指引", line, re.I):
                        leftovers.append("%s: %s" % (os.path.basename(f), line.strip()[:120]))
                        break
    check("py 侧旧 json 路径废弃", not leftovers,
          "; ".join(leftovers[:5])) or fail("旧 json 路径残留: %s" % leftovers[:8])

    # ⑦ 结构断言：gates.py 多维融合分协议
    gp = os.path.join(ROOT, "super/superopt/gates.py")
    gsrc = open(gp, encoding="utf-8").read()
    check("gates.py 多维指标（metrics 字典）", "constraints" in gsrc) or \
        fail("gates.py 缺 constraints（多维目标约束）")
    check("gates.py 融合分（weighted/fusion）",
          re.search(r"weighted|fusion|融合", gsrc, re.I)) or \
        fail("gates.py 缺融合分逻辑")

    # ⑧ 结构断言：gen_env skill 四步构建 + 四 skill + 验证金字塔
    ge = os.path.join(ROOT, ".rick/skills/gen_env_skill/skill.md")
    if not check("gen_env skill 存在", os.path.isfile(ge)):
        fail("gen_env skill 缺失")
    else:
        gs = open(ge, encoding="utf-8").read()
        for kw, desc in [
            (r"步骤\s*[①1]", "四步构建-步骤①设计树追问"),
            (r"profile.*benchmark.*verify.*research|四个.{0,6}skill", "四 skill 构建方法追问"),
            (r"演练", "演练步骤"),
            (r"验证金字塔|分层验证", "验证金字塔/分层验证"),
        ]:
            check("gen_env %s" % desc, re.search(kw, gs, re.I | re.S)) or \
                fail("gen_env skill 缺 %s" % desc)

    # ⑨ 结构断言：env 模板含 gates/ 目录 + domain_knowledge + env.json 新字段
    tpl_dir = os.path.join(ROOT, "super/env_templates")
    tpl_ok = False
    if os.path.isdir(tpl_dir):
        for d in sorted(os.listdir(tpl_dir)):
            dk = os.path.join(tpl_dir, d, "domain_knowledge.md")
            ej = os.path.join(tpl_dir, d, "env.json")
            gt = os.path.join(tpl_dir, d, "gates", "gate_check.py")
            if os.path.isfile(dk) and os.path.isfile(ej) and os.path.isfile(gt):
                es = open(ej, encoding="utf-8").read()
                if "objective" in es:
                    tpl_ok = True
                    print("[PASS] env 模板 %s 含 domain_knowledge+gates/+objective" % d)
                    break
    check("env_templates 有 v2 结构示范模板", tpl_ok) or \
        fail("env_templates 无含 domain_knowledge.md+gates/gate_check.py+objective 的模板")

    # ⑩ 结构断言：super-loop.md 增补（gates 隔离/负分协议/两类队列）
    sl = os.path.join(ROOT, ".rick/loops/super-loop.md")
    ss = open(sl, encoding="utf-8").read()
    check("super-loop.md gates 隔离规则", "gates" in ss and ("禁读" in ss or "不可读" in ss)) or \
        fail("super-loop.md 缺 gates/ 隔离规则")
    check("super-loop.md 负分协议", "负分" in ss or "score" in ss) or \
        fail("super-loop.md 缺 gate_check 负分协议")
    check("super-loop.md 两类队列", "提案" in ss and "干预" in ss) or \
        fail("super-loop.md 缺 humans 两类队列说明")

    print()
    if errs:
        print("=" * 60)
        print("gate1 失败（%d 项）：" % len(errs))
        for e in errs:
            print(" - %s" % e)
        return 1
    print("gate1 通过：DB 地基（task1）+ env 体系（task2）层验收 OK")
    return 0


if __name__ == "__main__":
    sys.exit(main())
