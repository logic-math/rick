"""budget.py — 成本账本（时间/token 双轨）（job_47/task4）

设计（research-L3-pi-token）：
- token 来源 = agents 侧聚合 pi `message_end`(role==assistant) 的 message.usage
  （input/output/cacheRead/cacheWrite/totalTokens/cost.total），按本模块的
  snake_case 契约传入：{input_tokens, output_tokens, cache_read, cache_write,
  total, cost_usd}。缺键按 0 容错（pi 事件 usage 子字段可能 undefined）。
- **事件即账**：budget 的持久真相源是 events.jsonl 的 agent_started/ended 对
  （CostLedger 不持久化，仅内存聚合）；周期性把 summary 写成事件由 loop 负责。
- 降级链：usage=None 或全 0（provider 不上报）→ token 记 0，summary 附
  token_degraded=true（账本降级为 wall-clock + tool_call 数双轨的墙钟轨）。

stdlib-only，Python >=3.9 兼容。
"""
import time
from typing import Any, Dict, List, Optional

# usage 的六个契约字段（缺键按 0 计）。
_USAGE_FIELDS = ("input_tokens", "output_tokens", "cache_read",
                 "cache_write", "total", "cost_usd")

# summary 每个 role 的输出键（tool_calls 与 wall_ms 是墙钟轨；其余 token 轨）。
_SUMMARY_KEYS = ("wall_ms", "tool_calls", "input_tokens", "output_tokens",
                 "total_tokens", "cost_usd")


def _zero_usage() -> Dict[str, float]:
    return {k: 0.0 for k in _USAGE_FIELDS}


def _norm_usage(usage: Optional[Dict[str, Any]]) -> Dict[str, float]:
    """usage → 六字段全量 dict；None/缺键/非数值按 0 容错。"""
    out = _zero_usage()
    if not usage:
        return out
    for k in _USAGE_FIELDS:
        v = usage.get(k)
        if isinstance(v, (int, float)) and not isinstance(v, bool):
            out[k] = float(v)
    # total 缺失时按字段和兜底（totalTokens 恒等式：
    # input + output + cacheRead + cacheWrite，research-L3-pi-token #3）
    if out["total"] == 0:
        out["total"] = (out["input_tokens"] + out["output_tokens"] +
                        out["cache_read"] + out["cache_write"])
    return out


class CostLedger(object):
    """内存成本账本：分 role 聚合 wall_ms / tool_calls / token / cost。

    构造参数：
        max_wall_time_h —— 墙钟预算（小时）；0/负 = 不限时。
        wall_start     —— 预算计时起点（unix 秒）；None = 当前时刻。
    """

    def __init__(self, max_wall_time_h: float,
                 wall_start: Optional[float] = None) -> None:
        self.max_wall_time_h = float(max_wall_time_h)
        self.wall_start = time.time() if wall_start is None else float(wall_start)
        # per-role 累加器：{role: {wall_ms, tool_calls, <六 usage 字段>}}
        self._by_role = {}  # type: Dict[str, Dict[str, float]]
        self._agent_runs = 0  # 有过 record_agent 的次数（降级判定用）

    # ---- 记账 ----

    def record_agent(self, role: str, wall_ms: int, tool_calls: int,
                     usage: Optional[Dict[str, Any]] = None) -> None:
        """记一次 agent 运行（一个 role 的一次独立 subprocess 会话）。

        usage 结构见模块 docstring；None = provider 未上报（全 0 记账）。
        """
        acc = self._by_role.setdefault(
            role, {"wall_ms": 0.0, "tool_calls": 0.0, **_zero_usage()})
        acc["wall_ms"] += float(wall_ms)
        acc["tool_calls"] += float(tool_calls)
        u = _norm_usage(usage)
        for k in _USAGE_FIELDS:
            acc[k] += u[k]
        self._agent_runs += 1

    def record_event(self, type: str, payload: Dict[str, Any]) -> None:
        """空实现占位（spec）：事件即账，真相源在 events.jsonl。

        loop 周期性把 summary 写成事件时直接调 summary()，不经本方法。
        保留签名是为了给 loop 一个「事件钩子」的稳定挂点，不引入任何状态。
        """
        return None

    # ---- 查询 ----

    def summary(self) -> Dict[str, Any]:
        """分 role 汇总 + 总计；全零 token（且有记录）→ token_degraded=true。"""
        by_role = {}
        tot = {"wall_ms": 0.0, "tool_calls": 0.0, **_zero_usage()}
        for role, acc in sorted(self._by_role.items()):
            row = {
                "wall_ms": acc["wall_ms"],
                "tool_calls": int(acc["tool_calls"]),
                "input_tokens": acc["input_tokens"],
                "output_tokens": acc["output_tokens"],
                "total_tokens": acc["total"],
                "cost_usd": acc["cost_usd"],
            }
            by_role[role] = row
            tot["wall_ms"] += acc["wall_ms"]
            tot["tool_calls"] += acc["tool_calls"]
            tot["input_tokens"] += acc["input_tokens"]
            tot["output_tokens"] += acc["output_tokens"]
            tot["total"] += acc["total"]
            tot["cost_usd"] += acc["cost_usd"]
        total_row = {
            "wall_ms": tot["wall_ms"],
            "tool_calls": int(tot["tool_calls"]),
            "input_tokens": tot["input_tokens"],
            "output_tokens": tot["output_tokens"],
            "total_tokens": tot["total"],
            "cost_usd": tot["cost_usd"],
        }
        out = {"by_role": by_role, "total": total_row}
        # 降级标注：有 agent 记录且全部 token 为 0（provider 未上报 usage）
        if self._agent_runs > 0 and tot["total"] == 0:
            out["token_degraded"] = True
        return out

    def within_budget(self) -> bool:
        """墙钟预算检查（无迭代上限——迭代上限由 loop 的 max_iterations 控制）。

        max_wall_time_h <= 0 → 不限时（True）。
        """
        if self.max_wall_time_h <= 0:
            return True
        elapsed = time.time() - self.wall_start
        return elapsed < self.max_wall_time_h * 3600.0
