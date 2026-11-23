#!/usr/bin/env python3
"""watchdog.py — super 进化系统定时巡检 + API 故障自动切换（中控级）。

设计（job_56 中控可用性机制——防「停机 11h 无人知晓」再演）：
  1. 巡检：super 进程存活 / heartbeat 新鲜度 / DB 事件异常（infra 停机）
  2. 根因分诊：API 探活（pi 一次小 prompt 实测当前模型）
  3. 自愈：API 故障 → 切换 runtime agent_models 到备用链（中控与 runtime
     模型隔离：中控跑默认模型不切换，runtime 切备用供应商）
  4. 上报：全部动作写 super.db watchdog_* 事件（可审计），JSON 输出到 stdout

用法（中控 schedule 每 1h 调一次）：
  python3 super/tools/watchdog.py <workspace> [--probe-timeout 120]

退出码：0=健康或已自愈；2=需人工介入（写明原因）。
"""
import argparse
import json
import os
import subprocess
import sys
import sqlite3
import time

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
sys.path.insert(0, os.path.join(ROOT, "super"))

# runtime 备用模型链（与中控默认 GLM 隔离：优先不同家族）
FALLBACK_CHAIN = [
    "deepseek-v4-pro",    # 1M ctx，DeepSeek 家族（与 GLM 异源）
    "kimi-k3",            # 1M ctx，Moonshot 家族
    "glm-5.2",            # 回退到 GLM 但不同版本（最后的同源保险）
]

STALE_HEARTBEAT_S = 300          # heartbeat 超过 5 分钟没更新 → 疑似卡死
WATCHDOG_INTERVAL_HINT_S = 3600  # 中控巡检周期（写事件用）


def _db(ws):
    return sqlite3.connect(os.path.join(ws, "super.db"))


def _now_ts():
    return time.strftime("%Y-%m-%dT%H:%M:%S+08:00")


def _iso_to_epoch(ts):
    """DB ts（2026-11-22T23:38:46+08:00）→ epoch 秒。"""
    try:
        from datetime import datetime
        return datetime.strptime(ts[:19], "%Y-%m-%dT%H:%M:%S").timestamp()
    except Exception:
        return 0


def check_process(ws):
    """① super 进程存活检查（live_pid）。"""
    sj_path = os.path.join(ws, "super.json")
    if not os.path.isfile(sj_path):
        return {"ok": False, "issue": "super.json 不存在"}
    sj = json.load(open(sj_path))
    state = sj.get("state", "unknown")
    pid = sj.get("pid") or sj.get("worker_pid")
    alive = False
    if pid:
        try:
            os.kill(int(pid), 0)
            alive = True
        except (OSError, ValueError):
            alive = False
    return {"ok": alive and state in ("running", "completed") or alive,
            "state": state, "pid": pid, "alive": alive,
            "issue": None if alive else "进程死亡/无 pid"}


def check_heartbeat(ws):
    """② heartbeat 新鲜度（DB 最后 heartbeat 与当前差）。"""
    conn = _db(ws)
    row = conn.execute(
        "SELECT ts FROM events WHERE event_type='heartbeat' "
        "ORDER BY seq DESC LIMIT 1").fetchone()
    conn.close()
    if not row:
        return {"ok": False, "age_s": None, "issue": "无 heartbeat 记录"}
    age = time.time() - _iso_to_epoch(row[0])
    return {"ok": age < STALE_HEARTBEAT_S, "age_s": int(age),
            "issue": None if age < STALE_HEARTBEAT_S
            else f"heartbeat 停滞 {int(age)}s"}


def check_last_session(ws):
    """③ 最近 session_ended 的原因分类。"""
    conn = _db(ws)
    row = conn.execute(
        "SELECT payload FROM events WHERE event_type='session_ended' "
        "ORDER BY seq DESC LIMIT 1").fetchone()
    conn.close()
    if not row:
        return {"ok": True, "reason": "无结束记录"}
    try:
        p = json.loads(row[0])
    except Exception:
        return {"ok": True, "reason": "unparsable"}
    infra = bool(p.get("infra_failure"))
    budget = bool(p.get("budget_exhausted"))
    return {"ok": not infra, "infra_failure": infra,
            "budget_exhausted": budget,
            "reason": "infra_failure" if infra else
                      ("budget_exhausted" if budget else "正常结束")}


def _pi_bin():
    """pi CLI 路径：env PI_BIN > PATH > rick 部署的 runtime 绝对路径。"""
    if os.environ.get("PI_BIN"):
        return os.environ["PI_BIN"]
    # PATH 里有就用
    from shutil import which
    hit = which("pi")
    if hit:
        return hit
    return "/Users/logic/.rick/pi/agent/runtime/node_modules/.bin/pi"


def probe_api(model=None, timeout=120):
    """④ API 探活：pi 跑一次最小 prompt 实测模型可用性。

    返回 (ok, detail)。探活 prompt 极小（一问一答）——与真实 agent 调用
    同链路（同 pi CLI → 同供应商），能捕捉「0-token 空回」类故障。
    """
    cmd = [_pi_bin(), "-p", "--no-session",
           "reply with the single word: pong"]
    if model:
        cmd += ["--model", model]
    try:
        r = subprocess.run(cmd, capture_output=True, text=True,
                           timeout=timeout)
        out = (r.stdout or "").strip()
        # 0-token 空回/配额报错 = 故障（正常探活必有文本输出）
        ok = r.returncode == 0 and len(out) > 0 and "quota" not in out.lower() \
             and "limit" not in out.lower() and "401" not in out
        return ok, (out[:120] or (r.stderr or "")[:120])
    except subprocess.TimeoutExpired:
        return False, "probe timeout %ss" % timeout
    except FileNotFoundError:
        return False, "pi CLI 不可用"


def current_runtime_model(ws):
    """读当前 runtime 模型配置（super.json config.agent_models 或 None=默认）。"""
    sj = json.load(open(os.path.join(ws, "super.json")))
    am = (sj.get("config") or {}).get("agent_models") or {}
    return am.get("builder") or am.get("adjust") or None


def switch_runtime_model(ws, new_model):
    """切换 runtime 模型（写 super.json config.agent_models 全角色）。"""
    sj_path = os.path.join(ws, "super.json")
    sj = json.load(open(sj_path))
    sj.setdefault("config", {})["agent_models"] = {
        "builder": new_model, "coding": new_model, "debug": new_model,
        "adjust": new_model,
    }
    with open(sj_path, "w", encoding="utf-8") as f:
        json.dump(sj, f, ensure_ascii=False, indent=2)


def write_event(ws, etype, payload):
    """动作写 DB（可审计）。表结构缺失时静默跳过（不阻塞巡检）。"""
    try:
        conn = _db(ws)
        conn.execute(
            "INSERT INTO events (event_type, payload, actor, ts) "
            "VALUES (?, ?, 'watchdog', ?)",
            (etype, json.dumps(payload, ensure_ascii=False), _now_ts()))
        conn.commit()
        conn.close()
    except Exception:
        pass


def restart_super(ws):
    """重启 super（rick super start --daemon，幂等：先 stop）。"""
    rick = os.path.join(ROOT, "bin", "rick")
    for cmd in ([rick, "super", "stop", "--name", os.path.basename(ws)],
                [rick, "super", "start", "--name", os.path.basename(ws),
                 "--daemon"]):
        try:
            subprocess.run(cmd, capture_output=True, text=True, timeout=60)
        except Exception:
            pass


def downtime_report(ws):
    """停机时长统计（中控巡检附带产出，供 ESQ D4 口径）。"""
    conn = _db(ws)
    rows = conn.execute(
        "SELECT event_type, ts FROM events WHERE event_type IN "
        "('session_started','session_ended') ORDER BY seq").fetchall()
    first = conn.execute("SELECT min(ts) FROM events").fetchone()[0]
    conn.close()
    if not rows or not first:
        return {"uptime_ratio": None}
    start_epoch = _iso_to_epoch(first)
    total = time.time() - start_epoch
    # 间隙 = ended→started 差值（空转=调度器死了没人重启）
    down = 0.0
    last_end = None
    for etype, ts in rows:
        ep = _iso_to_epoch(ts)
        if etype == "session_ended":
            last_end = ep
        elif etype == "session_started" and last_end is not None:
            gap = ep - last_end
            if gap > 0:
                down += gap
            last_end = None
    if last_end is not None:  # 当前还停着
        down += time.time() - last_end
    uptime = 1.0 - (down / total) if total > 0 else 1.0
    return {"evolution_start": first[:19], "total_s": int(total),
            "down_s": int(down), "uptime_ratio": round(uptime, 4)}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("workspace")
    ap.add_argument("--probe-timeout", type=int, default=120)
    ap.add_argument("--dry-run", action="store_true",
                    help="只巡检不动作（观察模式）")
    args = ap.parse_args()
    ws = os.path.abspath(args.workspace)

    report = {"ts": _now_ts(), "workspace": ws}
    actions = []

    # ── 巡检三件套 ──
    proc = check_process(ws)
    hb = check_heartbeat(ws)
    sess = check_last_session(ws)
    report["checks"] = {"process": proc, "heartbeat": hb,
                        "last_session": sess}
    report["downtime"] = downtime_report(ws)

    healthy = proc["ok"] and (hb["ok"] or proc["alive"])
    if healthy and not sess.get("infra_failure"):
        report["status"] = "healthy"
        write_event(ws, "watchdog_tick", {
            "status": "healthy", "heartbeat_age_s": hb.get("age_s"),
            "uptime_ratio": report["downtime"].get("uptime_ratio"),
            "interval_hint_s": WATCHDOG_INTERVAL_HINT_S})
        print(json.dumps(report, ensure_ascii=False, indent=2))
        return 0

    # ── 异常路径：分诊 ──
    # 进程死/heartbeat 停滞/上次 infra 结束 → 探当前 runtime 模型
    cur_model = current_runtime_model(ws)
    api_ok, api_detail = probe_api(
        model=cur_model, timeout=args.probe_timeout)
    report["api_probe"] = {"model": cur_model or "(default)",
                           "ok": api_ok, "detail": api_detail}

    if api_ok:
        # API 活着 → 普通崩溃（调度器/其他）：直接重启
        if not args.dry_run:
            restart_super(ws)
            actions.append("restart_super(api_ok)")
            write_event(ws, "watchdog_action", {
                "action": "restart_super", "reason": "进程死而 API 活",
                "model": cur_model or "(default)"})
        report["status"] = "restarted"
    else:
        # API 死 → 沿备用链找活模型（优先与中控 GLM 异源）
        switched = None
        for m in FALLBACK_CHAIN:
            if m == cur_model:
                continue
            ok, _ = probe_api(model=m, timeout=args.probe_timeout)
            if ok:
                switched = m
                break
        if switched:
            if not args.dry_run:
                switch_runtime_model(ws, switched)
                restart_super(ws)
                actions.append(f"switch_model({switched}) + restart")
                write_event(ws, "watchdog_action", {
                    "action": "switch_model_and_restart",
                    "from_model": cur_model or "(default)",
                    "to_model": switched,
                    "reason": "API 探活失败，切换到备用模型链"})
            report["status"] = "model_switched"
            report["switched_to"] = switched
        else:
            # 整个链都死 → 上报人工（中控也不可靠了——供应商级瘫痪）
            report["status"] = "manual_intervention_needed"
            report["issue"] = "主模型与备用链全部探活失败"
            write_event(ws, "watchdog_alert", {
                "status": "all_models_down",
                "tried": [cur_model] + [m for m in FALLBACK_CHAIN
                                        if m != cur_model],
                "hint": "供应商级故障或配额全面耗尽——需人工切换账号/等待恢复"})

    report["actions"] = actions
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 2 if report["status"] == "manual_intervention_needed" else 0


if __name__ == "__main__":
    sys.exit(main())
