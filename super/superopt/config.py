"""config.py — super 工作区配置层（job_47/task4）

职责边界（spec）：
- 只读工作区级 `<ws>/super.json`，**不读 ~/.rick**（那是 Go 侧 task11 的职责）。
- super.json 顶层为运行元信息：state/iteration_cursor/env_path/pid/heartbeat_path/budget/config。
- **budget 段是运行时预算状态**（wall_start 进程启动即更新）；**config 段是参数**——两层分离。
- DEFAULTS 与 Go 侧（internal/config SuperConfig）保持一致：
  adjust_interval=5 / debug_max_attempts=3 / soft_gate_k=3 /
  heartbeat_interval_s=30 / heartbeat_timeout_s=90 / term_grace_s=5 /
  max_wall_time_h=24.0 / max_iterations=0（0=无迭代上限）/ python_path="python3" /
  pi_path=""（空=必须由 super.json 显式给出；消费方（agents/loop）对空值报错）/
  builder_max_turns=20 / coding_max_turns=24 / debug_max_turns=0 /
  adjust_max_turns=16（assistant 回合预算；0=不限）/
  event_fsync=True（事件落盘 fsync 开关；高频事件可关，由 close/flush 兜底）/
  debug_gain_threshold_pct=20 / debug_recheck_tolerance_band=True
  （门禁通过后的异常收益/容差带复验；env.json debug_policy 可覆盖）/
  max_parallel_experiments=1（并发实验槽；当前降级为串行——见 loop 事件
  parallel_degraded）/ cell_lock_ttl_s=3600 / cell_lock_max_skips=60 /
  cell_lock_skip_sleep_s=1.0（格级互斥，迭代9）。

stdlib-only，Python >=3.9 兼容（禁 PEP604 联合类型注解）。
"""
import datetime
import json
import os
import tempfile
from typing import Any, Dict

# DEFAULTS：superopt 包的可调参数全集（gate2 断言与 Go 侧默认值一致性）。
DEFAULTS: Dict[str, Any] = {
    "adjust_interval": 5,          # 每 N 迭代跑一次 adjust agent
    "debug_max_attempts": 3,       # debug agent 修复尝试上限 M
    "soft_gate_k": 3,              # 软门禁多计时中位数的计时次数
    "heartbeat_interval_s": 30,    # py 写 heartbeat.json 的周期
    "heartbeat_timeout_s": 90,     # Go 侧判心跳超时的阈值
    "term_grace_s": 5,             # SIGTERM → SIGKILL 的宽限期
    "max_wall_time_h": 24.0,       # 墙钟预算（小时）；0/负 = 不限时
    "max_total_tokens": 0,         # token 预算上限（迭代3/task16）；0 = 不限。
                                   # 例：50_000_000（5000 万——约 demo 两轮的
                                   # 总量级）；超限 within_budget=False →
                                   # loop 优雅停（session_ended 标注）。
    "max_iterations": 0,           # 迭代上限；0 = 无上限（默认仅测试/e2e 传参）
    "python_path": "python3",
    "pi_path": "",                 # pi 二进制路径；空 = 消费方须显式提供
    # per-role 模型档位（role → pi --model 参数；空 dict = 全角色用默认模型）。
    # 优化空间①（job_47 首跑实测：builder 占 token 60%+ 而产出比 coding 简单）：
    # 给便宜角色降档。示例：{"builder": "catpaw-proxy/deepseek-v4-flash"}。
    # agent_ended 事件 payload.model 可观测每个 agent 实际跑的模型。
    "agent_models": {},
    # per-role 回合预算（assistant 回合上限；None/0 = 不限）——env_8 实测：
    # 21 迭代 25M token，builder 单次中位 478s/646k token、极值 1575s/2101k
    # （≈ 23 个 assistant 回合），每回合重算完整 input 是成本主因。pi 无
    # max-turns flag → agents.PiAgentClient 在 NDJSON 流计数 assistant
    # message_end，达上限走 kill 梯优雅终止（truncated 标记，已产出内容保留）。
    # 默认值依据：env_8 实测中位 3-7 回合、极值 23 → 截尾部；
    # coding 实现回合略多，adjust 只需审视一次。
    # **debug_max_turns=0（不设回合上限）——human 裁决（迭代10/task51）**：
    # env_9 实测 debug 20 次派发 12 次截断，而截断后的重试才是浪费主因
    # （截断=根因没定位完就死 → 下次重派从头再来）。改为不限回合，配合
    # 「一次性根因定位」协议（super/prompts/debug.md：复现→三假设→一次
    # 跑完验证→单次修复），兜底改由 agent_timeout_s（默认 900s）+ wall budget。
    "builder_max_turns": 20,
    "coding_max_turns": 24,
    "debug_max_turns": 0,
    "adjust_max_turns": 16,
    # ---- 事件落盘 fsync 开关（迭代10/task49+51）----
    # True（保守）=每事件 fsync；False=仅 flush（高频/并发实验时省 IO，
    # 崩溃可能丢尾行——events 读取有截断容错）。
    "event_fsync": True,
    # ---- 格级互斥与并发实验（迭代9/task45，human fix 2）----
    # 实验 = 一次特征格上的迭代；**同一格同时只有一个实验**（builder/coding/
    # debug 在该格上顺序协作），不同格可并行。互斥由 CellLocks 双保险实现
    # （进程内集合 + <ws>/locks 文件锁，跨进程也成立）。
    # max_parallel_experiments 是并发槽开关：>1 时当前实现会**降级为串行**
    # （每迭代状态是调度器级共享态、worker 不得写 events/archive；拆槽重构
    # 未完成）——降级事实经 `parallel_degraded` 事件显式记录，不静默。
    "max_parallel_experiments": 1,
    "cell_lock_ttl_s": 3600,        # 格锁陈旧回收阈值（秒；<=0 关闭 TTL）
    "cell_lock_max_skips": 60,      # 连续「全格被占」跳过上限 → 优雅停
    "cell_lock_skip_sleep_s": 1.0,  # 每次跳过的等待（秒）
    # debug 触发扩展（迭代8/KR6）：门禁通过后仍可能因「异常收益」或「容差带内
    # 无明显收益」被复验——防止错误的性能结论进入精英地图。
    # env.json 的 `debug_policy: {gain_threshold_pct, recheck_tolerance_band}`
    # 可逐 env 覆盖（gen_env 追问 human 确认）。
    "debug_gain_threshold_pct": 20,      # 收益 ≥ 该百分比 → debug 复验
    "debug_recheck_tolerance_band": True,  # 软门禁容差带内 → debug 复验
}

# merge_cli 允许从 CLI 覆盖的键（与 __main__.py 的 flags 对齐）。
_CLI_OVERRIDABLE = ("max_iterations", "adjust_interval", "max_wall_time")

# --max-wall-time 覆盖 config.max_wall_time_h 时同步 budget.max_wall_time_h
# （budget 段是运行生效值，两层必须一致）。
_BUDGET_SYNC_KEY = "max_wall_time_h"


def now_iso() -> str:
    """当前时间的 RFC3339 完整格式（带本地时区偏移）。

    bugs.md 坑位：每条时间戳必须带时区，否则 rick-gates 校验失败。
    """
    return datetime.datetime.now().astimezone().isoformat(timespec="seconds")


def _empty_super_json() -> Dict[str, Any]:
    """super.json 的最小骨架（load 缺文件时的兜底视图）。"""
    return {
        "state": None,
        "iteration_cursor": 0,
        "env_path": None,
        "pid": None,
        "heartbeat_path": None,
        "budget": {"wall_start": None,
                   "max_wall_time_h": DEFAULTS[_BUDGET_SYNC_KEY]},
        "config": dict(DEFAULTS),
        "updated_at": None,
    }


def load(ws: str) -> Dict[str, Any]:
    """读 `<ws>/super.json`，缺键补 DEFAULTS（仅内存补全，不写回）。

    - 文件不存在 → 返回最小骨架（state=None 等缺席值 + DEFAULTS 全量）。
    - 文件损坏（非法 JSON）→ raise ValueError（上层按环境失败结论化）。
    - 只补 config 段；其余段（state/cursor/pid/budget）按原样返回，缺即缺。
    """
    path = os.path.join(ws, "super.json")
    if not os.path.exists(path):
        return _empty_super_json()
    with open(path, "r", encoding="utf-8") as f:
        try:
            data = json.load(f)
        except json.JSONDecodeError as e:
            raise ValueError("super.json 解析失败（%s）: %s" % (path, e))
    if not isinstance(data, dict):
        raise ValueError("super.json 顶层必须是 JSON 对象: %s" % path)
    cfg = dict(data)
    merged = dict(DEFAULTS)
    merged.update(cfg.get("config") or {})
    cfg["config"] = merged
    return cfg


def merge_cli(cfg: Dict[str, Any], args: Any) -> Dict[str, Any]:
    """argparse Namespace（或 dict）覆盖 config 段——仅非 None 项。

    - 只接受 _CLI_OVERRIDABLE 白名单内的键，未知键忽略（防 CLI 注入）。
    - max_wall_time（flag）→ config.max_wall_time_h + budget.max_wall_time_h
      两层同步（budget 段是运行生效值）。
    - 返回新 dict，不就地修改入参。
    """
    out = dict(cfg)
    conf = dict(out.get("config") or {})
    budget_seg = dict(out.get("budget") or {})

    def _get(key):
        if isinstance(args, dict):
            return args.get(key)
        return getattr(args, key, None)

    for key in _CLI_OVERRIDABLE:
        val = _get(key)
        if val is None:
            continue
        target = _BUDGET_SYNC_KEY if key == "max_wall_time" else key
        conf[target] = val
        if target == _BUDGET_SYNC_KEY:
            budget_seg[_BUDGET_SYNC_KEY] = val
    out["config"] = conf
    if budget_seg or "budget" in out:
        out["budget"] = budget_seg if budget_seg else out.get("budget")
    return out


def _atomic_write_json(path: str, data: Dict[str, Any]) -> None:
    """临时文件 + fsync + rename 原子写（崩溃安全：写坏不破坏旧档）。"""
    fd, tmp = tempfile.mkstemp(dir=os.path.dirname(path),
                               prefix="." + os.path.basename(path) + ".",
                               suffix=".tmp")
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            json.dump(data, f, ensure_ascii=False, indent=2)
            f.flush()
            os.fsync(f.fileno())
        os.replace(tmp, path)
    except BaseException:
        if os.path.exists(tmp):
            os.unlink(tmp)
        raise


def init_ws(ws: str, env_path: str) -> Dict[str, Any]:
    """首次初始化工作区：目录骨架 + super.json。

    幂等语义：已存在的目录/文件不删除；super.json 已存在时只补缺键写回
    （不重置 state/iteration_cursor——运行中重复调用无害）。

    骨架：events.jsonl（空文件）/ archive.json（{"cells": {}, "pending": []}）/
    interventions/ / codes/ / prompts/。
    """
    os.makedirs(ws, exist_ok=True)
    for d in ("interventions", "codes", "prompts"):
        os.makedirs(os.path.join(ws, d), exist_ok=True)

    events_path = os.path.join(ws, "events.jsonl")
    if not os.path.exists(events_path):
        with open(events_path, "w", encoding="utf-8"):
            pass

    archive_path = os.path.join(ws, "archive.json")
    if not os.path.exists(archive_path):
        _atomic_write_json(archive_path, {"cells": {}, "pending": []})

    if os.path.exists(os.path.join(ws, "super.json")):
        # 幂等：读已有内容，仅补全 config 缺键 + 运行元信息
        data = load(ws)
    else:
        data = _empty_super_json()
        data["state"] = "idle"
        data["env_path"] = env_path
        data["budget"] = {"wall_start": None,
                          "max_wall_time_h": DEFAULTS[_BUDGET_SYNC_KEY]}
    data["heartbeat_path"] = os.path.join(ws, "heartbeat.json")
    data["updated_at"] = now_iso()

    _atomic_write_json(os.path.join(ws, "super.json"), data)
    return data
