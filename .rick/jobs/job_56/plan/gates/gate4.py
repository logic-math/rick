#!/usr/bin/env python3
"""gate4 — job_56 L4 层门禁：集成验收（task5）——全量回归 + e2e + 文档。

跑法：python3 .rick/jobs/job_56/plan/gates/gate4.py（仓库根）
exit 0 = job 层验收通过（最终门禁）。
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



def sh(cmd, cwd=ROOT, timeout=2400):
    r = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)
    return r.returncode, r.stdout, r.stderr


def read(p):
    return open(os.path.join(ROOT, p), encoding="utf-8").read()


def main():
    errs = []

    def fail(msg):
        errs.append(msg)

    # ① py 全量测试（含 e2e/mock/单测——discover 全跑）
    rc, out, errtxt = sh([sys.executable, "-m", "unittest",
                          "discover", "-s", "super/tests", "-p", "test_*.py", "-v"])
    if rc != 0:
        # 尝试定位失败用例（unittest 输出 FAIL/ERROR 行）
        bad = [ln for ln in (out + errtxt).splitlines()
               if re.match(r"(FAIL|ERROR):", ln)]
        fail("py 全量 unittest 失败（%d 个用例）:\n%s\n%s"
             % (len(bad), "\n".join(bad[:15]), out[-1500:]))
    else:
        print("[PASS] py 全量 unittest（super/tests discover）")

    # ② go build + 全量 test
    rc, out, errtxt = sh(["go", "build", "./..."])
    if rc != 0:
        fail("go build 失败:\n%s" % errtxt[-2000:])
    else:
        print("[PASS] go build ./...")
    if rc == 0:
        rc, out, errtxt = sh(["go", "test", "./..."], timeout=2400)
        if rc != 0:
            bad = [ln for ln in (out + errtxt).splitlines() if "--- FAIL" in ln]
            fail("go test ./... 失败:\n%s\n%s" % ("\n".join(bad[:10]), errtxt[-1500:]))
        else:
            print("[PASS] go test ./...")

    # ③ mock_pi 流水线场景
    try:
        mp = read("super/tests/mock_pi.py")
        for name, pat in [
            ("ok_adjust_pipeline 场景（7 步流水线）", r"ok_adjust_pipeline"),
            ("ok_debug_failed 场景（失败上报）", r"ok_debug_failed"),
        ]:
            ok = re.search(pat, mp) is not None
            if ok:
                print("[PASS] mock_pi %s" % name)
            else:
                fail("mock_pi.py 缺 %s" % name)
    except OSError:
        fail("mock_pi.py 缺失")

    # ④ e2e 断言（e2e_test 含 7 步流水线+DB 断言——discover 已跑通即存在）
    try:
        et = read("super/tests/e2e_test.py")
        for name, pat in [
            ("e2e 覆盖 7 步流水线", r"adjust_fact|adjust_pipeline|7\s*步"),
            ("e2e 覆盖 DB 状态断言（humans/cells/experiments）",
             r"humans|cells|experiments"),
        ]:
            ok = re.search(pat, et) is not None
            if ok:
                print("[PASS] e2e %s" % name)
            else:
                fail("e2e_test.py 缺 %s" % name)
    except OSError:
        fail("e2e_test.py 缺失")

    # ⑤ 进程锁 e2e 场景（et 在 ④ 中赋值；缺文件时兜底空串）
    try:
        et = et  # noqa: PLW0127
    except NameError:
        et = ""
    lock_e2e = ("scheduler.pid" in et) or ("pid" in read("super/tests/test_db.py").lower()
                                           if os.path.exists(os.path.join(ROOT, "super/tests/test_db.py")) else False)
    if lock_e2e:
        print("[PASS] 进程锁 e2e/单测覆盖")
    else:
        fail("进程锁缺 e2e 或单测覆盖（scheduler.pid 场景）")

    # ⑥ README 收口
    try:
        rm = read("super/README.md")
        for name, pat in [
            ("DB 架构说明", r"sqlite|super\.db|DB"),
            ("7 步协议说明", r"7\s*步|adjust_fact"),
            ("验证金字塔/分层验证", r"验证金字塔|分层验证|L1-M0"),
        ]:
            ok = re.search(pat, rm) is not None
            if ok:
                print("[PASS] README %s" % name)
            else:
                fail("super/README.md 缺 %s" % name)
    except OSError:
        fail("super/README.md 缺失")

    print()
    if errs:
        print("=" * 60)
        print("gate4 失败（%d 项）：" % len(errs))
        for e in errs:
            print(" - %s" % e)
        return 1
    print("gate4 通过：集成验收（task5）——job_56 四项重构全部落地")
    return 0


if __name__ == "__main__":
    sys.exit(main())
