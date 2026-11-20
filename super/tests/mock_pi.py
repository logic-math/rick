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
import re
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

EXPERIMENT_RECORD_TEMPLATE = """# 实验记录 {experiment_id} · {cell_id}

---
experiment_id: "{experiment_id}"
cell_id: "{cell_id}"
hypothesis: "用库级整阵乘替换逐元素循环可显著降低 runtime_ms"
status: running
score: 
---

- 迭代：{iteration_id}
- 实验 ID：{experiment_id}
- 特征格：{cell_id}

## 1. 实验设计

### 背景（为什么现在做这个实验：来自现状轨迹/台账/research 的证据）
{mock}目标格需要一次可复现的改进实验。

### 实验假设（一句话，可证伪）
假设：用库级整阵乘替换逐元素循环可显著降低 runtime_ms。

### 验证逻辑（什么结果支持假设、什么结果否定它）
若成立：gate_check 的 perf 值下降且 verify 通过；若否：perf 持平或正确性失败。

### 具体实验步骤（可复现：改哪、怎么改、跑什么命令）
1) 读父本；2) 实现整阵乘；3) 跑 verify/benchmark；4) 交框架终验。

### 观测预期（预期指标与量级，与 gate_defs 同尺度）
预期 runtime_ms 下降、max_error=0。

### 仿真预演（风险点推演：执行过程可能遇到的风险 + 每个风险的预防/解决措施）
风险：小规模下调用开销主导，收益被掩盖 → 多次计时取中位数。

### 证伪性论证（无论实验成功还是失败，各能获得什么信息增量）
成功：整阵乘方向有效，后续可叠加分块；失败：瓶颈不在逐元素循环，转 profile 重新定位。

## 2. 执行过程

（待 coding 填写）

## 3. 实验结论

（待 coding/debug 填写）

## 4. 反思

（待 debug 填写）
"""

RESEARCH_REPORT_TEMPLATE = """# 调研报告 {experiment_id} · {cell_id}

（{mock}）
- 现有代码：父本为逐元素实现，热点在內层循环。
- 业界落地：numpy.matmul / BLAS sgemm 直派发是标准做法。
- 论文实验细节：单次整阵乘在小规模下受调用开销主导，需多次计时取中位。
"""

CODING_PLAN_TEMPLATE = """# 编码执行计划 {experiment_id}

（{mock}）
1) 按 plan_steps 实现整阵乘（np.matmul 直派发，一次写对）；
2) 自验：verify → benchmark → gate_check（L1 快检优先）；
3) 补全实验记录第 2/3 节。
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
    """从 cwd/父目录名推断 experiment_id/iteration/cell（mock 尽力而为，失败用占位）。"""
    cwd = os.path.basename(os.getcwd())
    return {"iteration_id": cwd or "?", "experiment_id": cwd or "?",
            "cell_id": "mock-cell", "mock": "mock"}


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
    # 迭代8/job_56 task4：实验记录（四节+frontmatter）+ 调研报告落盘（cwd=迭代工作目录）
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
    # 迭代8/KR5 + job_56 task4：编码计划 + 实验记录四节补全（cwd=迭代工作目录）
    meta = _record_meta()
    _write_text("coding_plan.md", CODING_PLAN_TEMPLATE.format(**meta))
    _append_record("实现与自验（coding）",
                   "一次写对 np.matmul 直派发实现；自验 verify/benchmark/gate_check"
                   "均通过，max_error≈0。")
    _append_record("实验结论（coding）",
                   "结论：成功。gate_check perf=98.7（基线 123.4），支持实验假设。")
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
    """debug 修复场景（迭代8/KR6）：fix_report + three_hypotheses + 补全四节实验记录。"""
    meta = _record_meta()
    _append_record("反思（debug）",
                   "卡点：门禁 perf 未达预期；三假设逐一验证后根因=未走库级派发；"
                   "解决：改用 np.matmul；教训：小规模下调用开销主导，多次计时取中位。")
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
        "experiment_failed": False,
        "final_self_check": {"passed": True, "stage": "self_check",
                             "metrics": {"max_error": 0.0}},
    }
    fenced = "```json\n" + json.dumps(payload, ensure_ascii=False) + "\n```"
    _emit(_message_end("user", "按系统提示词完成任务，输出约定 JSON"))
    _emit(_message_end("assistant", "修复完成：\n" + fenced,
                       _usage(120, 40, cost_total=0.006)))
    _emit({"type": "agent_settled", "reason": "end_turn"})


def scenario_ok_debug_failed():
    """debug 主动上报失败（job_56 task4 KR-C/D）：experiment_failed=true
    + 完整四节实验记录落盘——调度器据此跳过 gate_check 端到端验证。"""
    meta = _record_meta()
    _append_record("反思（debug）",
                   "复现两次：perf 持平且 verify 失败（max_error 超容差）；"
                   "三假设验证后根因=实验计划的数据类型假设错误（fp16 精度"
                   "不满足容差）；处置：主动上报 experiment_failed（计划方向"
                   "不可行，修复价值低于信息增量）。")
    payload = {
        "valid": True,
        "score": None, "perf_value": None, "perf_type": None, "perf_desc": None,
        "summary": "实验失败（根因=计划数据类型假设错误），已主动上报并补全四节记录",
        "verdict": "experiment_failed",
        "experiment_failed": True,
        "three_hypotheses": [
            {"hypothesis": "实现错误", "if_true": "修 dtype 后 verify 过",
             "verified": False},
            {"hypothesis": "fp16 精度不满足容差（计划假设错误）",
             "if_true": "fp32 下 verify 过、fp16 恒不过 → 计划方向错误",
             "verified": True},
            {"hypothesis": "计时噪声", "if_true": "重复计时方差大",
             "verified": False},
        ],
        "fix_report": {"attempts": 1,
                       "root_cause": "实验计划的 fp16 数据类型假设错误",
                       "changes": "未改动（experiment_failed：跳过端到端验证）"},
        "final_self_check": {"passed": False, "stage": "verify",
                             "metrics": {"max_error": 0.05, "tolerance": 1e-3}},
    }
    fenced = "```json\n" + json.dumps(payload, ensure_ascii=False) + "\n```"
    _emit(_message_end("user", "按系统提示词完成任务，输出约定 JSON"))
    _emit(_message_end("assistant", "实验失败，主动上报：\n" + fenced,
                       _usage(100, 30, cost_total=0.005)))
    _emit({"type": "agent_settled", "reason": "end_turn"})


def _pipeline_prompt_path():
    """argv 解析 --append-system-prompt 的 prompt 文件路径（流水线步骤判定锚）。"""
    args = sys.argv[1:]
    for i, a in enumerate(args):
        if a == "--append-system-prompt" and i + 1 < len(args):
            return args[i + 1]
    return ""


def _extract_rendered_path(prompt_path, pattern):
    """从已渲染 prompt 内容提取真路径（decisions_path / domain_knowledge_path）。

    prompt 是框架渲染后的系统提示词文件：占位符已替换为真路径，
    mock 用正则从内容里提取（比硬编码路径稳——不依赖 ws 布局）。"""
    try:
        with open(prompt_path, encoding="utf-8") as f:
            m = re.search(pattern, f.read())
            return m.group(0) if m else None
    except OSError:
        return None


def _final_payload(step_payload):
    """六字段基座 + 步骤字段（AgentResult 契约 + 各步模板输出契约）。"""
    payload = {
        "valid": True, "score": None, "perf_value": None,
        "perf_type": None, "perf_desc": None,
    }
    payload.update(step_payload)
    return payload


def _emit_final(payload, inp=80, out=25, cost=0.004):
    fenced = "```json\n" + json.dumps(payload, ensure_ascii=False) + "\n```"
    _emit(_message_end("user", "按系统提示词完成任务，输出约定 JSON"))
    _emit(_message_end("assistant", "完成：\n" + fenced,
                       _usage(inp, out, cost_total=cost)))
    _emit({"type": "agent_settled", "reason": "end_turn"})


PIPELINE_FACTS_MD = """# 事实与期望 · 迭代（mock）

## 1. 现状事实（逐实验一行 + 已证伪/已证实假设清单）
- 迭代 1：库级整阵乘替换逐元素循环，gate_check 通过，runtime_ms 下降（已证实）。
- 小规模下调用开销主导（已证实，多次计时取中位）。

## 2. human 期望（未应用干预的意图还原）
- （mock）希望当前格继续深挖分块组织。

## 3. 差距与空白（现状 vs 期望；重复失败模式；空白方向）
- 分块组织方向尚无实验（空白）。
"""

PIPELINE_PERSPECTIVES_MD = """# 视角记录 · 迭代（mock）

## 视角 1：内存布局视角——数据局部性主导小矩阵乘性能
- 关键词：cache blocking（相关）/ 分块（相关）/ 食谱推荐（不相关）
- 逻辑链：小矩阵驻 L1 时循环重排收益大于算法替换（演绎）。
- 信源：事实推理

## 视角 2：调度开销视角——Python 层循环开销主导
- 关键词：python overhead（相关）/ 循环展开（相关）/ 天气预报（不相关）
- 逻辑链：多次计时中位下降且调用次数未变 → 开销在库内（归纳）。
- 信源：事实推理

## 视角 3：数值稳定性视角——精度换速度的边界
- 关键词：float32 精度（相关）/ 容差带（相关）/ 音乐节奏（不相关）
- 逻辑链：float16 截断被门禁拒 → 精度约束是活跃约束（归纳）。
- 信源：事实推理
"""


def _pipeline_contradiction_md(idx):
    return """# 矛盾记录 · 视角 %d（mock）

## 1. 活跃约束集（约束 + 当前状态）
- 精度容差：活跃（float16 被拒）。
- 内存带宽：未测（活跃待证）。

## 2. 影子价格评估（各约束的边际收益与理由）
- 精度容差：低（已满足，放松无收益）。
- 内存布局：高（分块组织未尝试，预期收益最大）。

## 3. 主要矛盾（影子价格最大的约束）
- 陈述：内存布局未分块，数据局部性收益未被释放。
- 证明逻辑：已证实库级派发有效 + 分块组织方向空白（演绎）。
- 事实置信度：库级派发有效（high）/ 分块收益（medium）。
""" % idx


PIPELINE_JUDGMENT_MD = """# 主要矛盾判定 · 迭代（mock）

## 1. 三矛盾逻辑审查
- 视角 1（内存布局）：演绎链完整，确定性高。
- 视角 2（调度开销）：归纳样本单一，确定性中。
- 视角 3（数值稳定性）：约束已满足非瓶颈，确定性中。

## 2. 排序（全局性/根本性/决定性 + 影子价格论证）
1. 内存布局（全局性：适用于所有后续迭代）。
2. 调度开销（局部：依赖规模）。
3. 数值稳定性（已满足）。

## 3. 主要矛盾（选定理由）
内存布局未分块：影子价格最大且全局适用。

## 4. 次要矛盾 ×2（保留理由）
- 调度开销：规模变化时可能上升。
- 数值稳定性：作为硬约束保留监控。
"""

PIPELINE_REVERSAL_MD = """# 辩证逆转 · 迭代（mock）

## 1. 主要矛盾的辩证逆转（阻碍 → 手段的重新定价）
内存未分块是阻碍 → 分块本身是手段：将「布局约束」重新定价为「性能杠杆」。

## 2. 解决方法
1. 分块组织（tile=32）：消解逻辑——局部性提升；证伪条件——分块后 perf 不降。
2. 转置复用：消解逻辑——减少跨步读取；证伪条件——无改善。
"""


PIPELINE_DECISIONS_MD = """# adjust 决策记录 · 迭代（mock 流水线）

## 1. 事实与期望（step1 摘要）
库级派发已证实；human 期望深挖分块组织。

## 2. 视角（step2 三视角）
内存布局 / 调度开销 / 数值稳定性。

## 3. 矛盾（step3 三矛盾）
内存布局未分块（主）/ 调度开销（次）/ 数值稳定性（次）。

## 4. 判定（step4 主要/次要）
主要：内存布局未分块（影子价格最大，全局性）。

## 5. 逆转（step5 方法）
布局约束 → 性能杠杆：分块组织（tile=32）。

## 6. 格动作（step6 动作+理由）
冠军格升权 1.5（证实有效）+ 新格「分块组织」（正交性：tile 尺度维度）。

## 7. 实验自改进（step7 摘要）
沉淀计时中位知识；验证器噪声提案入队列。

```json
{
  "primary_contradiction": {"statement": "内存布局未分块，局部性收益未释放",
    "logic_basis": "库级派发有效 + 分块方向空白（演绎）",
    "fact_confidence": [{"fact": "库级派发有效", "confidence": "high"}]},
  "secondary_contradictions": ["调度开销（规模依赖）", "数值稳定性（硬约束监控）"],
  "perspectives_used": [{"viewpoint": "内存布局（数据局部性）", "source": "事实推理"}],
  "critique_ranking": [{"contradiction": "内存布局", "logical_certainty": "高", "rank": 1}],
  "cell_actions": [
    {"action": "upweight", "cell": "impl=blocked|mem=tiled", "method": "",
     "reason": "iter 1 证实有效", "orthogonality": ""},
    {"action": "create", "cell": "impl=tiled|mem=blocked",
     "method": "tile=32 分块组织", "reason": "主要矛盾解法无格承载",
     "orthogonality": "tile 尺度维度未被现有格覆盖"}],
  "saturation_evidence_consumed": ["（mock）无饱和信号注入"]
}
```
"""


def scenario_ok_adjust_pipeline():
    """7 步 adjust 流水线场景（job_56 task5 KR-A）：按 prompt 文件名分派。

    派发侧每步独立 prompt 文件（prompt_adjust_fact.md /
    prompt_adjust_contradiction_{1..3}.md / …），mock 从 argv 解析
    --append-system-prompt 的文件名判定步骤；cwd=adjust_dir
    （codes/adjust/iter_N/），产物落盘 + 六字段基座 JSON。
    """
    prompt_path = _pipeline_prompt_path()
    base = os.path.basename(prompt_path or "")
    role = base[len("prompt_"):-len(".md")] if (
        base.startswith("prompt_") and base.endswith(".md")) else ""

    if role == "adjust_fact":
        _write_text("facts.md", PIPELINE_FACTS_MD)
        _emit_final(_final_payload({
            "summary": "现状：库级派发已证实；空白：分块组织",
            "facts_path": os.path.abspath("facts.md"),
        }))
    elif role == "adjust_perspective":
        _write_text("perspectives.md", PIPELINE_PERSPECTIVES_MD)
        _emit_final(_final_payload({
            "summary": "三视角：内存布局/调度开销/数值稳定性",
            "perspectives_path": os.path.abspath("perspectives.md"),
            "perspectives": ["内存布局视角（数据局部性）",
                             "调度开销视角（Python 层循环）",
                             "数值稳定性视角（精度换速度边界）"],
        }))
    elif role.startswith("adjust_contradiction"):
        m = re.search(r"_(\d+)$", role)
        idx = int(m.group(1)) if m else 1
        _write_text("contradiction_%d.md" % idx,
                    _pipeline_contradiction_md(idx))
        _emit_final(_final_payload({
            "summary": "视角 %d 主要矛盾：内存布局未分块" % idx,
            "perspective_index": idx,
            "contradiction_path": os.path.abspath(
                "contradiction_%d.md" % idx),
            "primary_contradiction": "内存布局未分块，局部性收益未释放",
            "logic_basis": "库级派发有效（已证实）+ 分块方向空白（演绎）",
            "fact_confidence": [{"fact": "库级派发有效", "confidence": "high"},
                                {"fact": "分块收益", "confidence": "medium"}],
        }))
    elif role == "adjust_judge":
        _write_text("judgment.md", PIPELINE_JUDGMENT_MD)
        _emit_final(_final_payload({
            "summary": "主要矛盾：内存布局未分块",
            "judgment_path": os.path.abspath("judgment.md"),
            "primary_contradiction": "内存布局未分块，局部性收益未释放",
            "secondary_contradictions": ["调度开销（规模依赖）",
                                         "数值稳定性（硬约束监控）"],
            "critique_ranking": [
                {"contradiction": "内存布局", "logical_certainty": "高（演绎完整）", "rank": 1},
                {"contradiction": "调度开销", "logical_certainty": "中（样本单一）", "rank": 2},
                {"contradiction": "数值稳定性", "logical_certainty": "中（已满足）", "rank": 3}],
        }))
    elif role == "adjust_reversal":
        _write_text("reversal.md", PIPELINE_REVERSAL_MD)
        _emit_final(_final_payload({
            "summary": "逆转：布局约束 → 性能杠杆（分块组织）",
            "reversal_path": os.path.abspath("reversal.md"),
            "methods": ["分块组织（tile=32）", "转置复用"],
        }))
    elif role == "adjust_space":
        # decisions 落盘：从已渲染 prompt 提取 decisions_path（占位符已替换）
        decisions_path = _extract_rendered_path(
            prompt_path, r"[\w/.\-]+/decisions/iter\d*(?:_\d+)?\.md")
        if decisions_path:
            try:
                os.makedirs(os.path.dirname(decisions_path) or ".",
                            exist_ok=True)
            except OSError:
                pass
            _write_text(decisions_path, PIPELINE_DECISIONS_MD)
        _emit_final(_final_payload({
            "summary": "主要矛盾（内存布局）→ 新格（分块组织）+ 冠军格升权",
            "decision_path": decisions_path or "",
            "proposals": [
                {"type": "soft_weight", "target_cell": "impl=blocked|mem=tiled",
                 "action": "weight=1.5", "evidence": "iter 1 库级派发证实有效",
                 "rule_id": "pipeline_upweight"},
                {"type": "new_cell", "action": "create",
                 "descriptor": {"impl": "tiled", "mem": "blocked"},
                 "name": "分块组织",
                 "method": "tile=32 分块组织 + 转置复用：将矩阵按 32×32 分块"
                           "逐块 np.matmul（局部性杠杆），块间累积；从主要矛盾"
                           "（内存布局未分块）辩证逆转而来（布局约束→性能杠杆）",
                 "source_refs": ["decisions（mock 流水线）"],
                 "orthogonality_rationale": "现有格均为整阵直派发（无分块维度）；"
                                           "分块组织引入 tile 尺度维度，正交",
                 "evidence": "视角 1 矛盾记录：分块方向空白且影子价格最大",
                 "rule_id": "pipeline_expand"}],
            "cell_actions": [
                {"action": "upweight", "cell": "impl=blocked|mem=tiled",
                 "method": "", "reason": "库级派发已证实（iter 1）",
                 "orthogonality": ""},
                {"action": "create", "cell": "impl=tiled|mem=blocked",
                 "method": "tile=32 分块组织", "reason": "主要矛盾解法无格承载",
                 "orthogonality": "tile 尺度维度未被现有格覆盖"}],
        }))
    elif role == "adjust_env":
        # domain_knowledge 追加：从 prompt 提取目标路径
        dk_path = _extract_rendered_path(
            prompt_path, r"[\w/.\-]+env/domain_knowledge\.md")
        appended = False
        if dk_path:
            try:
                os.makedirs(os.path.dirname(dk_path) or ".", exist_ok=True)
                section = ("\n## adjust 实验知识沉淀 · iter（mock）\n\n"
                           "- bug：小规模下多次计时不取中位 → 结论漂移；"
                           "解法：中位数计时（已被 gates 采纳）。\n")
                mode = "a" if os.path.exists(dk_path) else "w"
                with open(dk_path, mode, encoding="utf-8") as f:
                    f.write(section)
                appended = True
            except OSError:
                appended = False
        _emit_final(_final_payload({
            "summary": "沉淀 1 条计时中位知识 + 1 条验证器噪声提案",
            "domain_knowledge_appended": appended,
            "env_proposals": [
                {"kind": "noise", "title": "多次计时中位数入 domain_knowledge",
                 "finding": "iter 1：单次计时波动 ~5%",
                 "proposal": "benchmark skill 计时次数提到 5 次取中位",
                 "risk": "耗时略增"}],
        }))
    else:
        # 未知角色：诚实无 JSON 终话（客户端降级——不炸协议）
        _emit(_message_end("user", "按系统提示词完成任务"))
        _emit(_message_end("assistant", "（mock）未知步骤角色：%r" % role,
                           _usage(20, 5, cost_total=0.001)))
        _emit({"type": "agent_settled", "reason": "end_turn"})


SCENARIOS = {
    "ok_builder": scenario_ok_builder,
    "ok_debug": scenario_ok_debug,
    "ok_debug_failed": scenario_ok_debug_failed,
    "ok_adjust_pipeline": scenario_ok_adjust_pipeline,
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
    # 可选确定性延迟（迭代10/task52）：并发槽 e2e 用它保证两个实验的时间窗
    # 必然重叠（不依赖真实门禁耗时，避免 flaky）。默认 0 = 不延迟（零影响）。
    try:
        _delay = float(os.environ.get("MOCK_AGENT_SLEEP_S", "0") or 0)
    except ValueError:
        _delay = 0.0
    if _delay > 0:
        time.sleep(_delay)
    # pi 兼容参数（--mode/--append-system-prompt/-p 及位置参数）全部容忍忽略：
    # mock 只按场景输出，不解析语义。
    fn()
    return 0


if __name__ == "__main__":
    sys.exit(main())
