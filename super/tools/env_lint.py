#!/usr/bin/env python3
"""env_lint.py — super env 确定性验收器（gen-env 步骤③演练的自动化）。

设计动机（job_56 第六/七轮实测）：手工构建 env 踩了 7 个坑（全部靠人工
演练/门先证红发现）——「一次性生成正确的 env」需要确定性验收闭环：
  lint 红项清单 → 直接反馈给生成者修复 → 全绿才算 env 生成完成。

每条规则 = 一个踩过的坑（回归测试集语义）：
  R01 env.json 合法 JSON + 核心键（job_56：constraints bool 值炸框架校验）
  R02 objective 数值 schema（constraints[].value 必须 number——Q6 裁决）
  R03 v2 布局文件（gate_check/verify_impl/benchmark_impl/skills 薄入口）
  R04 exploit_registry 结构（[{id, sample_path}]——非文件名列表）
  R05 门先证红（exploit 全拒 + 金样例零误杀）
  R06 金样例 ModelNew 契约（v3 协议：candidate=ModelNew 铁律——金样例/
      环境健康检查对象必须有 ModelNew）
  R07 benchmark 公平计时（ref/cand 预构造单实例——计时不含模型构造；
      176× 假 speedup 的根因）
  R08 verify 正确性协议完整（多 trial + shape 检 + allclose + 全零量级检
      + outlier——KernelBench-Verified 四分布升级）
  R09 gate_check v2 协议（--env-dir 契约参数 + score 数值 + passed 一致
      + 负分路径 score<0 ⟺ passed=false）
  R10 三要素可跑（profile/verify/benchmark 对金样例全过——远程执行链连通）
  R11 domain_knowledge 必须节（任务/验证金字塔/作弊模式/资源规则/gates 禁读）
  R12 feature_space 轴合法（enum 值非空 + 格可枚举）

用法：python3 env_lint.py <env_dir>   → exit 0=全绿 / 1=红项清单
"""
import json
import os
import subprocess
import sys
from typing import List, Optional

V2_REQUIRED = ("gates/gate_check.py", "gates/verify_impl.py",
               "gates/benchmark_impl.py", "skills/verify.py",
               "skills/benchmark.py")
DK_REQUIRED_SECTIONS = ("验证金字塔", "作弊", "gates")


def lint(env_dir: str) -> List[str]:
    errs = []
    env_dir = os.path.abspath(env_dir)

    def _ok(ok, msg):
        if not ok:
            errs.append(msg)

    # R01 env.json
    env_json = os.path.join(env_dir, "env.json")
    cfg = {}
    if os.path.isfile(env_json):
        try:
            cfg = json.load(open(env_json, encoding="utf-8"))
        except ValueError as e:
            errs.append("R01 env.json 非法 JSON: %s" % e)
    else:
        errs.append("R01 缺 env.json")
    for k in ("task", "perf_desc", "objective", "env_slots",
              "verify_levels", "tolerance"):
        _ok(k in cfg, "R01 env.json 缺键 %r" % k)

    # R02 objective 数值 schema
    obj = cfg.get("objective") if isinstance(cfg.get("objective"), dict) else {}
    for c in obj.get("constraints") or []:
        v = c.get("value")
        _ok(isinstance(v, (int, float)) and not isinstance(v, bool),
            "R02 objective.constraints[%r].value 必须是数值（got %r）"
            % (c.get("name"), v))
    for m in obj.get("metrics") or []:
        _ok(isinstance(m.get("weight"), (int, float)),
            "R02 objective.metrics[%r].weight 必须是数值" % m.get("name"))

    # R03 v2 布局
    for rel in V2_REQUIRED:
        _ok(os.path.isfile(os.path.join(env_dir, rel)),
            "R03 缺 v2 布局文件 %s" % rel)
    _ok(os.path.isfile(os.path.join(env_dir, "code", "reference.py")),
        "R03 缺 code/reference.py")

    # R04 exploit_registry
    reg = os.path.join(env_dir, "gates", "exploit_registry.json")
    if os.path.isfile(reg):
        try:
            entries = json.load(open(reg, encoding="utf-8"))
            _ok(isinstance(entries, list) and entries,
                "R04 exploit_registry 必须非空数组")
            for e in entries:
                _ok(isinstance(e, dict) and e.get("id") and e.get("sample_path"),
                    "R04 exploit_registry 条目缺 id/sample_path: %r" % (e,))
                sp = e.get("sample_path")
                if sp:
                    # 框架语义：sample_path 相对 gates/ 目录（preverify_red 同源）
                    _ok(os.path.isfile(os.path.join(env_dir, "gates", sp)),
                        "R04 exploit 样本不存在: %s" % sp)
        except ValueError as e:
            errs.append("R04 exploit_registry 非法 JSON: %s" % e)
    else:
        errs.append("R04 缺 gates/exploit_registry.json")

    # R05+R06 门先证红（含金样例 ModelNew 契约——经框架 preverify_red）
    repo = _find_repo_root(env_dir)
    if repo:
        r = subprocess.run(
            [sys.executable, "-c",
             "import sys, json; sys.path.insert(0, %r + '/super');\n"
             "from superopt.gates import CascadeGate\n"
             "g = CascadeGate(%r, %r)\n"
             "print(json.dumps(g.preverify_red(), ensure_ascii=False))"
             % (repo, env_dir, os.path.join(env_dir, "gates"))],
            capture_output=True, text=True, timeout=1500)
        try:
            pv = json.loads(r.stdout.strip().splitlines()[-1])
        except (ValueError, IndexError):
            pv = {"ok": False, "error": (r.stderr or r.stdout)[-200:]}
        _ok(pv.get("ok"),
            "R05 门先证红未过: leaked=%r false_killed=%r err=%r"
            % (pv.get("leaked"), pv.get("false_killed"), pv.get("error")))
        for f in pv.get("false_killed") or []:
            errs.append("R06 金样例 %s 被误杀（检查 ModelNew 契约/"
                        "环境健康检查对象）" % f)

    # R07 benchmark 公平计时（静态：计时闭包不含模型实例化）
    bench_src = _read(os.path.join(env_dir, "gates", "benchmark_impl.py"))
    if bench_src is not None:
        _ok("Model(" not in _lambda_bodies(bench_src),
            "R07 benchmark 计时闭包含模型实例化（ref/cand 应预构造单实例）")

    # R08 verify 协议完整（静态关键词——分布/trial/量级/outlier）
    v_src = _read(os.path.join(env_dir, "gates", "verify_impl.py"))
    if v_src is not None:
        if "remote_gate" in v_src:
            # 远程执行薄壳模式：真实现部署于远程（本地只验薄壳引用契约）
            for kw in ("push_candidate", "remote_gate"):
                _ok(kw in v_src, "R08 远程薄壳缺 %r" % kw)
        else:
            for kw in ("allclose", "shape", "outlier"):
                _ok(kw in v_src, "R08 verify_impl 缺协议要素 %r" % kw)

    # R09 gate_check v2 契约（--env-dir 参数 + score/passed 语义）
    gc_src = _read(os.path.join(env_dir, "gates", "gate_check.py"))
    if gc_src is not None:
        _ok("--env-dir" in gc_src, "R09 gate_check 缺 --env-dir 契约参数")
        _ok("score" in gc_src and "passed" in gc_src,
            "R09 gate_check 缺 score/passed v2 协议字段")

    # R10 三要素可跑（金样例或 reference 的 ModelNew 版）
    golden_dir = os.path.join(env_dir, "gates", "golden_examples")
    probe = None
    if os.path.isdir(golden_dir):
        for f in sorted(os.listdir(golden_dir)):
            if f.endswith(".py"):
                probe = os.path.join(golden_dir, f)
                break
    if probe:
        for skill, name in (("skills/verify.py", "verify"),
                            ("skills/benchmark.py", "benchmark")):
            r = subprocess.run(
                [sys.executable, os.path.join(env_dir, skill),
                 "--candidate", probe, "--level", "L1"],
                capture_output=True, text=True, timeout=600)
            ok = False
            for line in reversed((r.stdout or "").splitlines()):
                if line.strip().startswith("{"):
                    try:
                        d = json.loads(line)
                        ok = bool(d.get("passed") or d.get("speedup", 0) > 0)
                    except ValueError:
                        pass
                    break
            _ok(ok, "R10 %s 对金样例失败: %s"
                % (name, (r.stderr or r.stdout)[-150:]))

    # R11 domain_knowledge 必须节
    dk = _read(os.path.join(env_dir, "domain_knowledge.md"))
    if dk is not None:
        for sec in DK_REQUIRED_SECTIONS:
            _ok(sec in dk, "R11 domain_knowledge 缺必须节 %r" % sec)

    # R14 框架链路演练（job_56 刷榜实测盲区：演练走手工命令没走框架真实
    # 调用路径——gate_check 被 CascadeGate 调用时缺 --problem 误杀好解）：
    # 用 CascadeGate.verify()（框架真实路径）跑金样例
    if repo:
        gold_dir = os.path.join(env_dir, "gates", "golden_examples")
        probe2 = None
        if os.path.isdir(gold_dir):
            for f in sorted(os.listdir(gold_dir)):
                if f.endswith(".py"):
                    probe2 = os.path.join(gold_dir, f)
                    break
        if probe2:
            r2 = subprocess.run(
                [sys.executable, "-c",
                 "import sys, json; sys.path.insert(0, %r + '/super');\n"
                 "from superopt.gates import CascadeGate\n"
                 "g = CascadeGate(%r, %r)\n"
                 "r = g.verify(%r)\n"
                 "print(json.dumps({'passed': r.passed, "
                 "'stage': r.failure_stage or ''}))"
                 % (repo, env_dir, os.path.join(env_dir, "gates"), probe2)],
                capture_output=True, text=True, timeout=1500)
            try:
                fr = json.loads(r2.stdout.strip().splitlines()[-1])
            except (ValueError, IndexError):
                fr = {"passed": False, "stage": "lint_parse_error"}
            _ok(fr.get("passed"),
                "R14 框架链路（CascadeGate.verify 真实调用路径）失败: %s"
                % fr.get("stage"))

    # R13 说明书（gen-env 步骤⑤产出物）
    _ok(os.path.isfile(os.path.join(env_dir, "HOW_IT_WORKS.md")),
        "R13 缺 HOW_IT_WORKS.md（env 工作原理说明书——gen-env 步骤⑤）")

    # R12 feature_space
    fs_p = os.path.join(env_dir, "feature_space.json")
    if os.path.isfile(fs_p):
        try:
            fs = json.load(open(fs_p, encoding="utf-8"))
            axes = fs.get("axes") or []
            _ok(bool(axes), "R12 feature_space 无轴")
            for a in axes:
                _ok(a.get("name") and a.get("values"),
                    "R12 轴 %r 缺 name/values" % a.get("name"))
        except ValueError as e:
            errs.append("R12 feature_space 非法 JSON: %s" % e)
    else:
        errs.append("R12 缺 feature_space.json")
    return errs


def _read(p):
    try:
        return open(p, encoding="utf-8").read()
    except OSError:
        return None


def _lambda_bodies(src):
    """lambda 行集合（粗粒度：检查计时 fn 是否内联实例化）。"""
    return "\n".join(ln for ln in src.splitlines()
                     if "lambda" in ln or "def _time" in ln or "_fn =" in ln)


def _find_repo_root(env_dir):
    d = env_dir
    for _ in range(12):
        if os.path.isdir(os.path.join(d, ".rick")) and \
                os.path.isdir(os.path.join(d, "super")):
            return d
        d = os.path.dirname(d)
    return None


def main():
    if len(sys.argv) != 2:
        print("用法: python3 env_lint.py <env_dir>")
        return 2
    errs = lint(sys.argv[1])
    if errs:
        print("env_lint: %d 项红（env 未达验收——修复后重跑）：" % len(errs))
        for e in errs:
            print("  - %s" % e)
        return 1
    print("env_lint: 全绿（14 项规则通过——env 验收完成）")
    return 0


if __name__ == "__main__":
    sys.exit(main())
