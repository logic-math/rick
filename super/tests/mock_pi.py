#!/usr/bin/env python3
"""mock_pi.py — job_47 task5 fake pi 脚本（测试专用）

照 rick tests/mock_agent.py 范式：MOCK_SCENARIO 环境变量驱动多场景。
模拟 `pi --mode json --append-system-prompt <f> -p <bootstrap>` 的 NDJSON
事件流（stdout 逐行 JSON），供 PiAgentClient（super/superopt/agents.py）单测。

用法：
    MOCK_SCENARIO=ok_builder python3 mock_pi.py --mode json \
        --append-system-prompt <file> -p <bootstrap>

场景（task5 规格）：
    ok_builder     2 条 assistant message_end（usage 非零，第二条含六字段
                   JSON）+ 1 条 user echo message_end（带伪造 usage——验证
                   role 过滤，bugs.md：pi 对 user/assistant 都发 message_end）
    ok_coding      tool_execution_end ×3（read/bash/write）+ final JSON
    zero_usage     message_end 无 usage → 兜底取最后一次非零 message_update
                   顶层 usage
    all_zero       全程零 usage → 客户端降级全 0
    bad_json       final 是废话无 JSON → 客户端 valid=False 优雅降级
    timeout_sleep  无输出长眠 → 客户端超时 kill（TERM→grace→KILL）
    many_turns     连续 N 个 assistant 回合（默认 20；MOCK_TURNS 可调）+ 回合
                   间隔短睡 → 回合预算达上限 kill（truncated=True，部分产出保留）
    ok_debug       debug 修复场景：写 fix_report + three_hypotheses，追加实验记录

产物落盘（迭代8/KR1/KR5）：ok_builder 写 experiment_record.md（六段）+
research_report.md；ok_coding 写 test_solution.py + coding_plan.md 并追加实验
记录第 6 段；ok_debug 追加实验记录（卡点/根因/解决）。落盘一律 try/except
兜底（mock 的 NDJSON 协议不能被文件错误打断；工作目录由调用方 cwd 决定）。

自测：python3 mock_pi.py --self-test（不依赖客户端，直接校验各场景输出可解析）
"""
import json
import os
import sys
import time


def _emit(obj):
    """单行紧凑 JSON + flush（行缓冲，客户端逐行可读）。"""
    sys.stdout.write(json.dumps(obj, ensure_ascii=False, separators=(",", ":")) + "\n")
    sys.stdout.flush()


def _usage(inp, out, cache_read=0, cache_write=0, cost_total=0.0, total=None):
    """构造 pi 的 Usage 结构（pi-ai 接口：input/output/cacheRead/cacheWrite/
    totalTokens/cost{input,output,cacheRead,cacheWrite,total}）。
    缺省字段防御：客户端遇缺键取 0。"""
    if total is None:
        total = inp + out + cache_read + cache_write
    return {
        "input": inp,
        "output": out,
        "cacheRead": cache_read,
        "cacheWrite": cache_write,
        "totalTokens": total,
        "cost": {
            "input": cost_total / 2.0,
            "output": cost_total / 2.0,
            "cacheRead": 0.0,
            "cacheWrite": 0.0,
            "total": cost_total,
        },
    }


def _message_end(role, text, usage=None):
    msg = {"role": role, "content": [{"type": "text", "text": text}]}
    if usage is not None:
        msg["usage"] = usage
    return {"type": "message_end", "message": msg}


def _message_update(role, text, usage):
    """message_update 顶层携带 usage（流式累计）——pi json 模式实测 schema。"""
    return {
        "type": "message_update",
        "message": {"role": role, "content": [{"type": "text", "text": text}]},
        "usage": usage,
    }


def _tool_exec(name, ok=True):
    return {
        "type": "tool_execution_end",
        "toolName": name,
        "toolCallId": "call-" + name,
        "result": {"type": "text", "text": "ok" if ok else "err"},
        "isError": not ok,
    }


# --------------------------------------------------------------------------- #
# 产物落盘（迭代8：实验记录 / 调研报告 / TDD 测试 / 编码计划）
# --------------------------------------------------------------------------- #

EXPERIMENT_RECORD_TEMPLATE = """# 实验记录 iter={iteration_id} · {cell_id}

## 1. 背景与价值
{mock}目标格需要一次可复现的改进实验。

## 2. 实验假设
假设：用库级整阵乘替换逐元素循环可显著降低 runtime_ms。

## 3. 验证逻辑
若成立：gate_check 的 perf 值下降且 verify 通过；若否：perf 持平或正确性失败。

## 4. 具体实验步骤
1) 读父本；2) 实现整阵乘；3) 跑 verify/benchmark；4) 交框架终验。

## 5. 观测现象
预期 runtime_ms 下降、max_error=0。

## 6. 执行过程与结论
（待 coding/debug 追加）
"""

RESEARCH_REPORT_TEMPLATE = """# 调研报告 iter={iteration_id} · {cell_id}

（{mock}）
- 现有代码：父本为逐元素实现，热点在內层循环。
- 业界落地：numpy.matmul / BLAS sgemm 直派发是标准做法。
- 论文实验细节：单次整阵乘在小规模下受调用开销主导，需多次计时取中位。
"""

CODING_PLAN_TEMPLATE = """# 编码执行计划 iter={iteration_id} · {cell_id}

（{mock}）
1) 写单元测试 test_solution.py（正确性 + 形状）；
2) 先看到红（未实现）；
3) 实现 matmul（np.matmul 直派发）；
4) 绿 → 重构 → 保绿。
"""

TEST_SOLUTION_SRC = """import numpy as np


def test_matmul_matches_numpy():
    a = np.arange(64, dtype=np.float32).reshape(8, 8)
    b = np.arange(64, dtype=np.float32).reshape(8, 8)
    from solution import matmul
    out = matmul(a, b)
    assert np.allclose(out, np.matmul(a, b), atol=1e-3)
"""


def _write_text(name, text):
    """尽力落盘（cwd=调用方工作目录）；失败不打断 NDJSON 协议。"""
    try:
        with open(name, "w", encoding="utf-8") as f:
            f.write(text)
        return True
    except OSError:
        return False


def _append_record(title, body):
    """幂等追加实验记录段落（已存在同名段则跳过）。"""
    path = "experiment_record.md"
    try:
        existing = ""
        if os.path.exists(path):
            with open(path, encoding="utf-8") as f:
                existing = f.read()
        if title in existing:
            return True
        with open(path, "a", encoding="utf-8") as f:
            f.write("\n### %s\n%s\n" % (title, body))
        return True
    except OSError:
        return False


def _record_meta():
    """从 cwd/父目录名推断 iteration/cell（mock 尽力而为，失败用占位）。"""
    cwd = os.path.basename(os.getcwd())
    return {"iteration_id": cwd or "?", "cell_id": "mock-cell", "mock": "mock"}


SIX_FIELDS_BUILDER = {
    "valid": True,
    "score": 0.85,
    "perf_value": 123.4,
    "perf_type": "runtime_ms",
    "perf_desc": "low",
    "summary": "建议在 blocked 格尝试转置缓存友好布局",
    # 迭代8/KR4：实验计划与实验记录对齐（假设 + 验证逻辑）
    "experiment_plan": {
        "idea": "整阵乘替换逐元素循环",
        "target_cell": "mock-cell",
        "parents": [],
        "plan_steps": ["读父本", "实现整阵乘", "跑 verify/benchmark"],
        "expected_behavior": "runtime_ms 下降、max_error=0",
        "hypothesis": "库级整阵乘显著快于逐元素循环",
        "validation_logic": "perf 下降且 verify 通过 → 支持；否则否定",
    },
    "papers": [
        {"arxiv_id": "1609.00076", "title": "BLISlab: A Sandbox for Optimizing GEMM",
         "insight": "microkernel 为核心的分块组织", "applicability": "单次整阵 BLAS 派发"}
    ],
    "inspirations": [{"source_cell": "mock-neighbor", "what": "库级直派发"}],
    "reasoning": "mock：父本热点在內层循环，库级整阵乘可消除。",
    # 兼容既有 fixture 契约（test_agents.TestOkBuilder 断言角色增量键在 payload 中；
    # builder 实际不用 proposals，但 mock 载荷同时充当通用样例）
    "proposals": [
        {"type": "soft_weight", "target_cell": "impl=blocked|mem=tiled",
         "action": "weight 0.5", "evidence": "visit 12 零入格", "rule_id": "R1"}
    ],
}

SIX_FIELDS_CODING = {
    "valid": True,
    "score": 0.9,
    "perf_value": 98.7,
    "perf_type": "runtime_ms",
    "perf_desc": "low",
    "summary": "实现分块转置乘法，自检通过",
    "diff_path": "codes/abc123/solution.py",
    "self_check": {"passed": True, "stage": "self_check", "metrics": {"max_error": 1e-06}},
}


def scenario_ok_builder():
    # 迭代8：实验记录（六段）+ 调研报告落盘（cwd=迭代工作目录）
    meta = _record_meta()
    _write_text("experiment_record.md",
                EXPERIMENT_RECORD_TEMPLATE.format(**meta))
    _write_text("research_report.md",
                RESEARCH_REPORT_TEMPLATE.format(**meta))
    # 1) user echo（bugs.md：pi 对 user 轮也发 message_end——带伪造 usage，
    #    客户端必须按 role 过滤、不计数）
    _emit(_message_end("user", "按系统提示词完成任务，输出约定 JSON"))
    # 2) assistant 第 1 条（中间轮，usage 非零）
    _emit(_message_end(
        "assistant", "分析目标格的访问记录……",
        _usage(100, 50, cache_read=1000, cache_write=0, cost_total=0.01)))
    # 3) assistant final（六字段 JSON 在 ```json 围栏里）
    fenced = "```json\n" + json.dumps(SIX_FIELDS_BUILDER, ensure_ascii=False, indent=2) + "\n```"
    _emit(_message_end(
        "assistant", "结论如下：\n" + fenced,
        _usage(200, 80, cache_read=2000, cache_write=10, cost_total=0.02)))
    _emit({"type": "agent_settled", "reason": "end_turn"})


def scenario_ok_coding():
    # 迭代8/KR5：TDD 痕迹 + 编码计划 + 实验记录追加（cwd=迭代工作目录）
    meta = _record_meta()
    _write_text("test_solution.py", TEST_SOLUTION_SRC)
    _write_text("coding_plan.md", CODING_PLAN_TEMPLATE.format(**meta))
    _append_record("coding：执行过程与结论",
                   "先写 test_solution.py 看到红 → 实现 np.matmul → 绿 → 重构保绿；"
                   "自检 passed，max_error≈0。")
    _emit(_message_end("user", "按系统提示词完成任务，输出约定 JSON"))
    _emit(_tool_exec("read"))
    _emit(_tool_exec("bash"))
    _emit(_tool_exec("write"))
    _emit(_message_end(
        "assistant", "已完成实现。\n```json\n" +
        json.dumps(SIX_FIELDS_CODING, ensure_ascii=False, indent=2) + "\n```",
        _usage(300, 150, cache_read=500, cache_write=0, cost_total=0.03)))
    _emit({"type": "agent_settled", "reason": "end_turn"})


def scenario_zero_usage():
    # message_update 流式：先零、再非零、再零、最后非零（取最后一次非零）
    _emit(_message_update("assistant", "流式片段1", _usage(0, 0)))
    _emit(_message_update("assistant", "流式片段2", _usage(100, 20, cost_total=0.005)))
    _emit(_message_update("assistant", "流式片段3", _usage(0, 0)))
    _emit(_message_update("assistant", "流式片段4", _usage(500, 60, cost_total=0.008)))
    # user echo 的 message_update 也带 usage —— 必须被过滤
    _emit(_message_update("user", "echo", _usage(999999, 999999)))
    # message_end：无 usage 字段（触发兜底链）
    _emit(_message_end("assistant", "零上报 provider 的回复。```json\n" +
                       json.dumps(SIX_FIELDS_BUILDER, ensure_ascii=False) + "\n```"))
    _emit({"type": "agent_settled", "reason": "end_turn"})


def scenario_all_zero():
    # message_end 带 usage 但全 0 + message_update 也全 0 → 降级全 0
    _emit(_message_update("assistant", "流式", _usage(0, 0)))
    _emit(_message_end(
        "assistant", "全零场景。```json\n" +
        json.dumps(SIX_FIELDS_BUILDER, ensure_ascii=False) + "\n```",
        _usage(0, 0)))
    _emit({"type": "agent_settled", "reason": "end_turn"})


def scenario_bad_json():
    _emit(_message_end(
        "assistant",
        "我无法完成这个任务：目标格的父本代码文件不存在，且 experiment_plan 里引用的"
        "参考实现路径无法读取。建议检查 codes/{uuid}/ 的快照完整性。这里没有 JSON。",
        _usage(50, 30, cost_total=0.004)))
    _emit({"type": "agent_settled", "reason": "end_turn"})


def scenario_timeout_sleep():
    # 无输出长眠（客户端 SIGTERM 应能终止；python 默认 SIGTERM 处理即退出）
    time.sleep(300)


def scenario_many_turns():
    """回合预算场景（task22）：连续 N 个 assistant 回合（默认 20，> 默认
    builder_max_turns=12），每回合带 tool 调用 + 短睡（保证达上限时进程仍活，
    kill 梯可验证）；最后一回合带六字段 JSON——验证「截断不等于无产出」的
    fallback 解析路径。MOCK_TURNS 可调回合数。"""
    try:
        turns = int(os.environ.get("MOCK_TURNS", "20"))
    except ValueError:
        turns = 20
    _emit(_message_end("user", "按系统提示词完成任务，输出约定 JSON"))
    for i in range(turns):
        _emit(_tool_exec("read" if i % 2 == 0 else "bash"))
        text = "第 %d 回合分析中……" % (i + 1)
        if i == turns - 1:
            text += "\n```json\n" + json.dumps(
                SIX_FIELDS_BUILDER, ensure_ascii=False) + "\n```"
        _emit(_message_end("assistant", text,
                           _usage(100, 20, cost_total=0.001)))
        time.sleep(0.1)  # 回合间隔（达上限时进程仍活 → kill 梯生效）
    _emit({"type": "agent_settled", "reason": "end_turn"})


def scenario_ok_debug():
    """debug 修复场景（迭代8/KR6）：fix_report + three_hypotheses + 追加实验记录。"""
    meta = _record_meta()
    _append_record("debug：卡点/根因/解决",
                   "卡点：门禁 perf 未达预期；三假设逐一验证后根因=未走库级派发；"
                   "解决：改用 np.matmul。")
    payload = {
        "valid": True,
        "score": None, "perf_value": None, "perf_type": None, "perf_desc": None,
        "summary": "已修复：改用 np.matmul",
        "fix_report": {"attempts": 1, "root_cause": "未走库级派发",
                       "changes": ["改用 np.matmul"]},
        "three_hypotheses": [
            {"hypothesis": "未使用库级派发", "if_true": "切到 matmul 后 perf 显著下降",
             "verified": True},
            {"hypothesis": "dtype 转换开销", "if_true": "预转 dtype 后无改善 → 否定",
             "verified": False},
            {"hypothesis": "计时噪声", "if_true": "重复计时方差大 → 否定",
             "verified": False},
        ],
        "verdict": "fixed",
        "final_self_check": {"passed": True, "stage": "self_check",
                             "metrics": {"max_error": 0.0}},
    }
    fenced = "```json\n" + json.dumps(payload, ensure_ascii=False) + "\n```"
    _emit(_message_end("user", "按系统提示词完成任务，输出约定 JSON"))
    _emit(_message_end("assistant", "修复完成：\n" + fenced,
                       _usage(120, 40, cost_total=0.006)))
    _emit({"type": "agent_settled", "reason": "end_turn"})


SCENARIOS = {
    "ok_builder": scenario_ok_builder,
    "ok_debug": scenario_ok_debug,
    "ok_coding": scenario_ok_coding,
    "zero_usage": scenario_zero_usage,
    "all_zero": scenario_all_zero,
    "bad_json": scenario_bad_json,
    "timeout_sleep": scenario_timeout_sleep,
    "many_turns": scenario_many_turns,
}


def self_test():
    """不依赖客户端的自测：各场景（除 timeout_sleep）输出逐行可 JSON 解析。"""
    import io
    import tempfile
    ok = True
    # 产物落盘型场景（ok_builder/ok_coding/ok_debug）会往 cwd 写文件——
    # self-test 统一切到临时目录，避免污染仓库根
    prev_cwd = os.getcwd()
    tmpdir = tempfile.mkdtemp(prefix="mock-pi-selftest-")
    os.chdir(tmpdir)
    for name, fn in SCENARIOS.items():
        if name == "timeout_sleep":
            print("  %-14s skip（长眠场景）" % name)
            continue
        buf = io.StringIO()
        old = sys.stdout
        sys.stdout = buf
        try:
            fn()
        finally:
            sys.stdout = old
        lines = [l for l in buf.getvalue().splitlines() if l.strip()]
        try:
            for line in lines:
                json.loads(line)
            print("  %-14s %d 行 NDJSON 全部可解析 ✓" % (name, len(lines)))
        except ValueError as e:
            ok = False
            print("  %-14s 解析失败：%s" % (name, e))
    os.chdir(prev_cwd)
    import shutil
    shutil.rmtree(tmpdir, ignore_errors=True)
    print("SELF_TEST %s" % ("OK" if ok else "FAIL"))
    return 0 if ok else 1


def main():
    if "--self-test" in sys.argv:
        return self_test()
    scenario = os.environ.get("MOCK_SCENARIO", "ok_builder")
    fn = SCENARIOS.get(scenario)
    if fn is None:
        sys.stderr.write("mock_pi: unknown MOCK_SCENARIO=%r (known: %s)\n"
                         % (scenario, ", ".join(sorted(SCENARIOS))))
        return 2
    # pi 兼容参数（--mode/--append-system-prompt/-p 及位置参数）全部容忍忽略：
    # mock 只按场景输出，不解析语义。
    fn()
    return 0


if __name__ == "__main__":
    sys.exit(main())
