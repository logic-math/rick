"""agents.py — job_47 task5：pi 池客户端（superopt 与 pi agent 的桥）

subprocess 直调 `pi --mode json --append-system-prompt <prompt> -p <bootstrap>`
（J4 裁决：每迭代一进程，env 热更新友好；J4 终裁禁止走 pi RPC）。

关键契约（research-L3-pi-token 简报，信源 A=代码原文）：
  - json 模式无独立 usage 事件：`message_end.message.usage`（该轮最终值，权威）
    与 `message_update` 顶层 `usage`（流式累计）。
  - **pi 对 user 和 assistant 轮都发 message_end**（bugs.md job_30 实证）——
    final_message 与 usage 聚合都必须过滤 `message.role == "assistant"`。
  - Usage 结构（pi-ai）：input/output/cacheRead/cacheWrite/totalTokens +
    cost{input,output,cacheRead,cacheWrite,total}；totalTokens =
    input+output+cacheRead+cacheWrite（恒等式实测验证）。
  - 兜底链：message_end 聚合全零 → 取最后一次非零 message_update 顶层 usage
    → 仍零 → usage 全 0（CostLedger 侧降级标注）。
  - usage 键名大小写/缺键防御：遇缺键取 0，不 raise（faux/test provider 全 0
    路径实证存在）。

六字段基座（继承 v5 schema）：valid/score/perf_value/perf_type/perf_desc/summary；
角色增量（adjust→proposals / builder→experiment_plan / coding→diff_path+self_check /
debug→fix_report+final_self_check）整体保留在 AgentResult.data（不在此处按角色
硬编码 schema——那是 loop.py 各 _run_* 的消费职责）。

stdlib-only，Python ≥3.9（禁 PEP604 X|Y 注解）。
"""
import json
import os
import re
import signal
import subprocess
import threading
import time
from dataclasses import dataclass, field
from typing import Dict, List, Optional

# bootstrap 触发句（v4.4.5 先例：系统提示词承载协议，user 消息只做启动触发）
BOOTSTRAP_MESSAGE = "按系统提示词完成任务，最终回复输出约定 JSON"

# 超时梯（与 Go 侧 PySupervisor 语义一致：TERM → grace → KILL）
TERM_GRACE_S = 5.0

# usage 六键（AgentResult.usage 的契约结构；CostLedger 按 role 消费同名字段）
USAGE_KEYS = ("input_tokens", "output_tokens", "cache_read",
              "cache_write", "total_tokens", "cost_usd")

# 配额/额度报错标记（task57 R1：env_11 实测配额耗尽时 pi 正常退出 0、
# 回复是额度报错文案、usage 全零——识别后标 infra_failure，不再 resume）
_QUOTA_MARKERS = ("额度", "配额", "credit-hub", "credit-mgmt",
                  "quota", "exceeded")

# 候选产物命名契约（task57 R2 产物抢救：solution.*/candidate.* 源文件）
_ARTIFACT_SUFFIXES = (".py", ".cpp", ".cc", ".cxx", ".c")
_ARTIFACT_PREFIXES = ("solution", "candidate")


def _zero_usage():
    return {k: 0 for k in USAGE_KEYS}


@dataclass
class AgentResult:
    """pi 单次运行的结构化结果（六字段基座 + 角色增量 + usage + 元信息）。

    usage 键：input_tokens/output_tokens/cache_read/cache_write/total_tokens/
    cost_usd（映射自 pi usage.input/output/cacheRead/cacheWrite/
    totalTokens/cost.total）。全 0 = provider 未上报（降级路径）。
    data：final message 内嵌 JSON 的完整解析结果（角色增量的载体）。
    model：实际使用的模型 id（per-role 档位可观测；空 = 默认模型）。
    truncated：回合预算耗尽被终止（env_8 实测成本失控治理：pi 无 max-turns
        flag，客户端在 NDJSON 流中计数 assistant message_end；截断不等于无产出——
        final_message/usage/tool_calls/六字段全部保留）。
    turn_warn：回合数达到软告警阈值（task56 A 段：不截断，只告警）。
    json_repair：JSON 输出修复次数（task56 B 段：0=首次即合规；
        1=第一次无 JSON → resume 一次后修复成功）。
    session_id：pi session id（task56 B 段：JSON 修复/失败恢复用
        --session-id 恢复同一会话）。
    infra_failure：基础设施级失败（task57 R1：配额/断连形态——0-token +
        额度报错文案）。真实验失败都烧 token；此标记让调度器把
        infra 空转与实验失败分开计量（iteration_ended.outcome=infra_noop）。
    artifact_path：agent 本轮新落盘的候选产物（task57 R2：workdir 里
        mtime ≥ 运行起点的 solution.*/candidate.* 源文件——「做完了但没说」
        的物证，空串 = 无）。
    """
    role: str
    valid: bool = False
    score: Optional[float] = None
    perf_value: Optional[float] = None
    perf_type: Optional[str] = None
    perf_desc: Optional[str] = None
    summary: Optional[str] = None
    usage: Dict = field(default_factory=_zero_usage)
    data: Dict = field(default_factory=dict)
    final_message: str = ""
    tool_calls: List = field(default_factory=list)
    wall_ms: int = 0
    exit_code: int = 0
    model: str = ""
    truncated: bool = False
    turn_warn: bool = False
    json_repair: int = 0
    session_id: str = ""
    infra_failure: bool = False
    artifact_path: str = ""

    def to_payload(self) -> Dict:
        """事件 payload 形态（agent_ended 事件的载荷）。"""
        return {
            "role": self.role,
            "valid": self.valid,
            "score": self.score,
            "perf_value": self.perf_value,
            "perf_type": self.perf_type,
            "perf_desc": self.perf_desc,
            "summary": self.summary,
            "usage": dict(self.usage),
            "wall_ms": self.wall_ms,
            "exit_code": self.exit_code,
            "tool_calls": list(self.tool_calls),
            "model": self.model,
            "truncated": self.truncated,
            "turn_warn": self.turn_warn,
            "json_repair": self.json_repair,
            "session_id": self.session_id,
            "infra_failure": self.infra_failure,
            "artifact_path": self.artifact_path,
        }


# ─── 六字段提取 ──────────────────────────────────────────────────────────────

_FENCED_JSON_RE = re.compile(r"```(?:json)?\s*(\{.*?\})\s*```", re.DOTALL)


def _extract_six_fields(final_message: str):
    """从 final message 提取六字段基座 + 完整 data。

    载体优先级：```json 围栏 → 裸 JSON 对象（首个 { 起的平衡段）。
    解析失败 → valid=False + summary=前 200 字（不炸调用方循环）。

    返回 (six_fields_dict, data_dict)：
      six_fields: {valid, score, perf_value, perf_type, perf_desc, summary}
      data: 完整解析的 JSON 对象（角色增量载体；失败 = {}）
    """
    text = final_message or ""
    parsed = None

    m = _FENCED_JSON_RE.search(text)
    if m:
        try:
            candidate = json.loads(m.group(1))
            if isinstance(candidate, dict):
                parsed = candidate
        except ValueError:
            parsed = None

    if parsed is None:
        # 裸 JSON：从首个 { 尝试到与之平衡的 }（逐步收缩尾部）
        start = text.find("{")
        if start >= 0:
            depth = 0
            end = -1
            in_str = False
            esc = False
            for i in range(start, len(text)):
                ch = text[i]
                if in_str:
                    if esc:
                        esc = False
                    elif ch == "\\":
                        esc = True
                    elif ch == '"':
                        in_str = False
                    continue
                if ch == '"':
                    in_str = True
                elif ch == "{":
                    depth += 1
                elif ch == "}":
                    depth -= 1
                    if depth == 0:
                        end = i
                        break
            if end > start:
                for stop in (end, len(text) - 1):
                    if stop <= start:
                        continue
                    try:
                        candidate = json.loads(text[start:stop + 1])
                        if isinstance(candidate, dict):
                            parsed = candidate
                            break
                    except ValueError:
                        continue

    six = {
        "valid": False,
        "score": None,
        "perf_value": None,
        "perf_type": None,
        "perf_desc": None,
        "summary": None,
    }
    if parsed is None:
        six["summary"] = text[:200]
        return six, {}

    six["valid"] = bool(parsed.get("valid", False))
    for key in ("score", "perf_value"):
        val = parsed.get(key)
        if isinstance(val, bool):  # 防御：True 不是合法 score
            val = None
        six[key] = float(val) if isinstance(val, (int, float)) else None
    for key in ("perf_type", "perf_desc"):
        val = parsed.get(key)
        six[key] = val if isinstance(val, str) else None
    val = parsed.get("summary")
    six["summary"] = val if isinstance(val, str) else None
    return six, parsed


# ─── usage 解析（防御式：缺键取 0，不 raise） ────────────────────────────────

def _num(value):
    """数字防御：None/str/bool → 0；int/float 原样。"""
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        return 0
    return value


def _usage_from(u) -> Optional[Dict]:
    """pi Usage 结构 → 我们的六键聚合单位。返回 None 表示不可用（无对象/全零）。

    pi 键名大小写防御：优先 research-L3-pi-token 实测映射
    （input/output/cacheRead/cacheWrite/totalTokens/cost.total）。
    """
    if not isinstance(u, dict):
        return None
    cost = u.get("cost")
    cost_total = 0.0
    if isinstance(cost, dict):
        cost_total = _num(cost.get("total"))
    elif isinstance(cost, (int, float)) and not isinstance(cost, bool):
        cost_total = float(cost)  # meta.json 折叠形态兼容
    got = {
        "input_tokens": _num(u.get("input")),
        "output_tokens": _num(u.get("output")),
        "cache_read": _num(u.get("cacheRead")),
        "cache_write": _num(u.get("cacheWrite")),
        "cost_usd": cost_total,
    }
    if (got["input_tokens"] + got["output_tokens"] +
            got["cache_read"] + got["cache_write"]) == 0 and cost_total == 0.0:
        return None
    return got


def _usage_is_zero(u: Dict) -> bool:
    return (u["input_tokens"] + u["output_tokens"] +
            u["cache_read"] + u["cache_write"] == 0 and u["cost_usd"] == 0.0)


# ─── NDJSON 流解析 ───────────────────────────────────────────────────────────

class _StreamState:
    """一次 run 的流解析累积态。"""

    def __init__(self):
        self.final_message = ""
        self.tool_calls = []          # 首现顺序去重
        self._tool_seen = set()
        self.agg = _zero_usage()      # message_end 聚合（权威路径）
        self.last_update = None       # 最后一次非零 message_update（兜底）
        self.bad_lines = 0
        self.assistant_turns = 0      # assistant message_end 回合计数（回合预算）
        self.model_changes = []       # model_change 序列（provider 回退可观测：
                                      # env_11 实测 catpaw→deepseek 静默回退 60 轮无人知）

    def feed(self, line: str):
        line = line.strip()
        if not line:
            return
        try:
            ev = json.loads(line)
        except ValueError:
            self.bad_lines += 1
            return
        if not isinstance(ev, dict):
            return
        etype = ev.get("type")

        if etype == "model_change":
            # provider/model 解析链（回退检测）：记录全部变更
            provider = ev.get("provider")
            model_id = ev.get("modelId")
            if isinstance(provider, str) or isinstance(model_id, str):
                self.model_changes.append(
                    "%s/%s" % (provider or "?", model_id or "?"))

        elif etype == "message_end":
            msg = ev.get("message")
            if not isinstance(msg, dict) or msg.get("role") != "assistant":
                return  # user echo 跳过（bugs.md：两轮都发 message_end）
            self._take_assistant_message(msg)

        elif etype == "message_update":
            # 顶层 usage（流式累计）；role 过滤同上（无 message 字段时容忍）
            msg = ev.get("message")
            if isinstance(msg, dict) and msg.get("role") not in (None, "assistant"):
                return
            u = _usage_from(ev.get("usage"))
            if u is not None:
                self.last_update = u

        elif etype == "tool_execution_end":
            name = ev.get("toolName")
            if isinstance(name, str) and name and name not in self._tool_seen:
                self._tool_seen.add(name)
                self.tool_calls.append(name)
        # agent_settled / 其它事件：无需处理（以 EOF 为终止）

    def _take_assistant_message(self, msg: dict):
        self.assistant_turns += 1
        parts = []
        content = msg.get("content")
        if isinstance(content, list):
            for block in content:
                if isinstance(block, dict) and block.get("type") == "text" \
                        and isinstance(block.get("text"), str):
                    parts.append(block["text"])
        elif isinstance(content, str):  # 防御：极简 provider 直接给字符串
            parts.append(content)
        if parts:
            self.final_message = "\n".join(parts)  # 最后一条 assistant 轮胜出
        u = _usage_from(msg.get("usage"))
        if u is not None:
            for k in ("input_tokens", "output_tokens", "cache_read",
                      "cache_write", "cost_usd"):
                self.agg[k] += u[k]

    def resolved_usage(self) -> Dict:
        """兜底链：聚合 → 最后非零 message_update → 全 0。"""
        if not _usage_is_zero(self.agg):
            out = dict(self.agg)
        elif self.last_update is not None:
            out = dict(self.last_update)
        else:
            out = _zero_usage()
        out["total_tokens"] = (out["input_tokens"] + out["output_tokens"] +
                               out["cache_read"] + out["cache_write"])
        return out


# ─── 客户端 ──────────────────────────────────────────────────────────────────

class PiAgentClient:
    """subprocess 直调 pi 的 agent 池客户端（每迭代一进程）。

    pi_path：pi 可执行文件绝对路径（如
    ~/.rick/pi/agent/runtime/node_modules/.bin/pi；测试指向 mock_pi.py）。
    agent_dir：可选，注入 PI_CODING_AGENT_DIR（继承托管配置）；缺省不设。
    """

    def __init__(self, pi_path: str, agent_dir: Optional[str] = None):
        if not pi_path:
            raise ValueError("pi_path is required")
        self.pi_path = pi_path
        self.agent_dir = agent_dir

    def _build_cmd(self, prompt_path: str, model: Optional[str] = None,
                   session_id: Optional[str] = None,
                   message: Optional[str] = None,
                   resume: bool = False):
        """构建 pi CLI 命令。

        prompt_path：系统提示词文件路径。
        model：per-role 模型档位（空 = 默认）。
        session_id：pi session id（task56 B 段：JSON 修复/失败恢复用
            --session-id 在同一 cwd 恢复会话）。
        message：用户消息（None = BOOTSTRAP_MESSAGE；resume 时传入修复/续跑指令）。
        resume：True = 用 --session-id 恢复已有会话（不传 --append-system-prompt，
            保留原系统提示词的上下文）。
        """
        cmd = [self.pi_path, "--mode", "json"]
        if session_id:
            cmd += ["--session-id", session_id]
        if not resume:
            cmd += ["--append-system-prompt", prompt_path]
        cmd += ["-p", message or BOOTSTRAP_MESSAGE]
        if model:
            cmd += ["--model", model]
        return cmd

    def run(self, role: str, prompt_path: str, workdir: str,
            timeout_s: int = 1800, model: Optional[str] = None,
            max_turns: Optional[int] = None,
            turn_soft_warn: int = 0) -> AgentResult:
        """派发一个 agent 运行并解析其 NDJSON 事件流（task56 B 段含 resume）。

        prompt_path：预渲染的系统提示词文件（绝对化）。
        workdir：agent 的工作目录（须已存在；codes/{uuid}/ 由调用方建好）。
        timeout_s：墙钟超时——超时走 kill 梯 SIGTERM→grace→SIGKILL。
        model：可选模型档位（pi --model 参数；None/空 = 默认模型）。
        max_turns：紧急硬杀（task56 A 段：0/None=不限；非零时真正杀线
            是 max_turns*3——极端防护；软告警由 turn_soft_warn 承担）。
        turn_soft_warn：软告警阈值（达到 → turn_warn=True，不截断）。

        task56 B 段：JSON 修复 + 失败恢复
        - 首次跑完后：JSON 缺失/不合规 → --session-id 恢复同一会话 +
          追加「请只输出约定 JSON」→ 再解析（json_repair=1）。
          task57 R2 修正：不再要求 final_message 非空——截断/空回时
          final_message 可能为空但产物已落盘 workdir，同样要给一次
          补发 JSON 的机会。
        - 首次因网络/超时/异常终止（exit!=0 或 tok=0）→ resume 一次
          （「你被中断了，请继续完成任务」）；仍失败才返回失败结果

        task57 R1：配额/额度报错（0-token + 报错文案）→ infra_failure=True
        直接返回（不 resume——同一凭据重试只会再失败一次，白烧一次派发）。
        """
        import uuid
        session_id = "superopt-%s-%s" % (role, uuid.uuid4().hex[:12])
        t_start = time.time()  # 产物新鲜度基线（含 resume 内的写入）

        result = self._run_once(role, prompt_path, workdir, timeout_s,
                                model, max_turns, session_id, turn_soft_warn)

        # 配额/额度报错（task57 R1）：infra 级故障，不 resume、不修复
        if self._is_quota_error(result):
            result.infra_failure = True
            return result

        # 失败恢复（空回/异常退出 → resume 一次）
        if self._needs_failure_resume(result):
            result = self._resume_session(
                role, prompt_path, workdir, timeout_s, model, max_turns,
                session_id, turn_soft_warn, result,
                "你刚才被中断了。请继续完成任务，最终回复输出约定 JSON 代码块。")

        # JSON 修复（无 JSON 输出 → resume 一次；task57 R2 去掉 final_message
        # 非空的前置——截断时 final_message 可能为 None 但产物在 workdir）
        if not result.valid and not result.data \
                and result.json_repair == 0:
            result = self._resume_session(
                role, prompt_path, workdir, timeout_s, model, max_turns,
                session_id, turn_soft_warn, result,
                "你的最终回复缺少必需的 JSON 输出块。请按系统提示词的"
                "输出契约，只输出那个 JSON 代码块（```json ... ```），"
                "不要任何其它内容。")

        # 产物定位（task57 R2）：本轮新落盘的候选（client 侧可观测物证；
        # 调度器另有 mtime 同规的 _rescue_artifact 扫描，两处同一契约）
        result.artifact_path = self._find_fresh_artifact(workdir, t_start)
        return result

    @staticmethod
    def _is_quota_error(result: "AgentResult") -> bool:
        """task57 R1：配额/额度报错识别（env_11 实测形态）。

        判据：0-token + 回复文本含额度报错标记（额度/credit-hub/quota/
        exceeded）。0-token 是硬门槛——回合截断标记 [turn-budget-exceeded]
        含 exceeded 字样但截断都烧 token，不会误中（再叠加显式排除）。
        """
        usage = result.usage or {}
        if _num(usage.get("total_tokens")) > 0:
            return False
        text = "%s\n%s" % (result.final_message or "", result.summary or "")
        if "[turn-budget-exceeded" in text:
            return False
        low = text.lower()
        return any(m in low for m in _QUOTA_MARKERS)

    @staticmethod
    def _find_fresh_artifact(workdir: str, since_epoch: float) -> str:
        """task57 R2：找本轮新落盘的候选产物（空串 = 无）。

        契约：solution.*/candidate.* 源文件、非空、mtime ≥ 运行起点——
        种子父本（copy2 保留源 mtime）天然被排除，只有 agent 本轮
        写的才算「做完了但没说」的物证。
        """
        try:
            names = sorted(os.listdir(workdir))
        except OSError:
            return ""
        for name in names:
            if not name.startswith(_ARTIFACT_PREFIXES):
                continue
            if not name.endswith(_ARTIFACT_SUFFIXES):
                continue
            p = os.path.join(workdir, name)
            try:
                if not os.path.isfile(p) or os.path.getsize(p) <= 0:
                    continue
                if os.path.getmtime(p) < since_epoch:
                    continue
            except OSError:
                continue
            return p
        return ""

    def _needs_failure_resume(self, result: AgentResult) -> bool:
        """task56 B 段：是否需要失败恢复（空回/异常终止）。"""
        if (result.usage or {}).get("total_tokens", 0) == 0 \
                and result.exit_code == 0 and result.wall_ms < 10000:
            return True
        if result.exit_code not in (0, None) and not result.truncated:
            return True
        return False

    def _resume_session(self, role: str, prompt_path: str, workdir: str,
                        timeout_s: int, model: Optional[str],
                        max_turns: Optional[int], session_id: str,
                        turn_soft_warn: int, prev: AgentResult,
                        message: str) -> AgentResult:
        """用 --session-id 恢复同一会话，追加一条用户消息让 agent 继续完成。"""
        resumed = self._run_once(role, prompt_path, workdir, timeout_s,
                                 model, max_turns, session_id, turn_soft_warn,
                                 resume=True, message=message)
        # 保留首次的 usage/wall（cost 归因到真实总消耗）
        if (resumed.usage or {}).get("total_tokens", 0) == 0:
            resumed.usage = dict(prev.usage)
        resumed.wall_ms = resumed.wall_ms + prev.wall_ms
        resumed.json_repair = prev.json_repair + 1
        return resumed

    def _run_once(self, role: str, prompt_path: str, workdir: str,
                  timeout_s: int, model: Optional[str],
                  max_turns: Optional[int], session_id: str,
                  turn_soft_warn: int = 0,
                  resume: bool = False,
                  message: Optional[str] = None) -> AgentResult:
        """单次 pi 子进程运行（首次 or resume），解析事件流。"""
        prompt_path = os.path.abspath(prompt_path)
        workdir = os.path.abspath(workdir)
        if not os.path.isfile(prompt_path):
            raise ValueError("prompt file not found: %s" % prompt_path)
        if not os.path.isdir(workdir):
            raise ValueError("workdir not found: %s" % workdir)

        env = os.environ.copy()
        if self.agent_dir:
            env["PI_CODING_AGENT_DIR"] = self.agent_dir
        # task57 补遗（pi hook 真实验证版）：激活 superopt-json-guard 扩展
        # （agent_end 时检查最终回复缺 JSON → sendUserMessage(followUp) 续跑
        # 修复轮，同进程同会话；实测见 job_47 迭代13 验证记录）
        env["SUPEROPT_JSON_GUARD"] = "1"

        cmd = self._build_cmd(prompt_path, model, session_id, message, resume)
        state = _StreamState()
        stderr_tail = []
        t0 = time.monotonic()

        # 回合预算（兼容旧行为：max_turns>0 时达上限即 kill；默认 0 = 不限，
        # 软告警由 turn_soft_warn 承担）
        budget = int(max_turns) if isinstance(max_turns, int) and max_turns > 0 else 0
        budget_hit = threading.Event()

        proc = subprocess.Popen(
            cmd, cwd=workdir, env=env,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            text=True,
        )

        def pump_stdout():
            try:
                for line in proc.stdout:
                    state.feed(line)
                    if budget and state.assistant_turns >= budget \
                            and not budget_hit.is_set():
                        budget_hit.set()
            except Exception:  # noqa: BLE001
                pass
            finally:
                try:
                    proc.stdout.close()
                except Exception:  # noqa: BLE001
                    pass

        def pump_stderr():
            try:
                for line in proc.stderr:
                    if len(stderr_tail) < 200:
                        stderr_tail.append(line.rstrip("\n"))
            except Exception:  # noqa: BLE001
                pass
            finally:
                try:
                    proc.stderr.close()
                except Exception:  # noqa: BLE001
                    pass

        reader = threading.Thread(target=pump_stdout, daemon=True)
        err_reader = threading.Thread(target=pump_stderr, daemon=True)
        reader.start()
        err_reader.start()

        timed_out = False
        truncated = False
        deadline = time.monotonic() + max(0.0, float(timeout_s))
        while True:
            if proc.poll() is not None:
                break
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                timed_out = True
                self._kill_ladder(proc)
                break
            try:
                proc.wait(timeout=min(0.1, remaining))
                break
            except subprocess.TimeoutExpired:
                pass
            if budget_hit.is_set():
                truncated = True
                self._kill_ladder(proc)
                break
        reader.join(timeout=10)
        err_reader.join(timeout=10)

        wall_ms = int((time.monotonic() - t0) * 1000)
        usage = state.resolved_usage()
        six, data = _extract_six_fields(state.final_message)

        effective_model = model or ""
        if state.model_changes:
            effective_model = " ".join(state.model_changes[-2:])

        turn_warn = bool(turn_soft_warn and state.assistant_turns >= turn_soft_warn)

        result = AgentResult(
            role=role,
            valid=bool(six["valid"]) and not timed_out
            and (proc.returncode == 0),
            score=six["score"],
            perf_value=six["perf_value"],
            perf_type=six["perf_type"],
            perf_desc=six["perf_desc"],
            summary=six["summary"],
            usage=usage,
            data=data,
            final_message=state.final_message,
            tool_calls=state.tool_calls,
            wall_ms=wall_ms,
            exit_code=proc.returncode if proc.returncode is not None else -1,
            model=effective_model,
            truncated=truncated,
            turn_warn=turn_warn,
            session_id=session_id,
        )
        if timed_out:
            marker = "[timeout after %ss]" % timeout_s
            if not result.summary:
                result.summary = marker
            elif marker not in result.summary:
                result.summary = (result.summary[:180] + " " + marker)[:200]
        elif truncated:
            marker = "[turn-budget-exceeded: %d]" % budget
            result.summary = (marker + " " + (result.summary or "")).strip()[:200]
        elif proc.returncode not in (0, None):
            marker = "[exit %s]" % proc.returncode
            if stderr_tail:
                marker += " stderr_tail: " + " | ".join(stderr_tail[-3:])
            result.summary = ((result.summary or "") + " " + marker).strip()[:200]
        return result

    @staticmethod
    def _kill_ladder(proc: subprocess.Popen):
        """kill 梯：SIGTERM → grace → SIGKILL（整进程组交给调用方语义，这里
        单进程 TERM/KILL 即可——pi 无孙进程）。"""
        try:
            proc.terminate()  # SIGTERM
            try:
                proc.wait(timeout=TERM_GRACE_S)
                return
            except subprocess.TimeoutExpired:
                pass
            proc.kill()  # SIGKILL
            proc.wait(timeout=5)
        except Exception:  # noqa: BLE001 进程已死等竞态——不向上抛
            pass


def _module_selfcheck() -> int:
    """python3 super/superopt/agents.py → 快速自检（六字段提取 + usage 解析）。"""
    six, data = _extract_six_fields('```json\n{"valid": true, "score": 0.5}\n```')
    assert six["valid"] and six["score"] == 0.5, six
    assert data == {"valid": True, "score": 0.5}, data
    u = _usage_from({"input": 10, "output": 5, "cacheRead": 1, "cacheWrite": 2,
                     "cost": {"total": 0.01}})
    assert u == {"input_tokens": 10, "output_tokens": 5, "cache_read": 1,
                 "cache_write": 2, "cost_usd": 0.01}, u
    assert _usage_from({"input": 0, "output": 0}) is None
    print("agents.py selfcheck OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(_module_selfcheck())
