#!/usr/bin/env python3
"""watchdog.py — super 进化系统巡检采集脚本（skill:super-watchdog 的确定性采集端）。

架构（job_56 中控可用性机制 v3——用户裁决：skill 封装采集脚本+修复指引，
修复动作由中控 loop 自行完成）：
  每 1h pi schedule 向中控投递巡检提示词（指向 skill:super-watchdog）
    → 中控加载 skill → 跑本脚本采集现状
    → 对照预期状态表找差距 → 按修复行为指引（playbook）执行修复

本脚本三种模式：
  默认              纯采集报告（JSON：进程/heartbeat/session/模型/探活/停机统计）
  --restart         修复手柄：重启 super（中控按指引调用）
  --switch-to <m>   修复手柄：切换 runtime agent_models + 重启（中控按指引调用）

退出码（默认模式）：0=健康；1=需处理（中控按 skill playbook 继续）；2=脚本自身故障。
"""
import argparse
import json
import os
import subprocess
import sqlite3
import sys
import time

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

STALE_HEARTBEAT_S = 300  # heartbeat 超过 5 分钟没更新 → 疑似卡死


def _db(ws):
    return sqlite3.connect(os.path.join(ws, "super.db"))


def _now_ts():
    return time.strftime("%Y-%m-%dT%H:%M:%S+08:00")


def _iso_to_epoch(ts):
    try:
        from datetime import datetime
        return datetime.strptime(ts[:19], "%Y-%m-%dT%H:%M:%S").timestamp()
    except Exception:
        return 0


def check_process(ws):
    """① super 进程存活检查。"""
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
    return {"ok": alive, "state": state, "pid": pid, "alive": alive,
            "issue": None if alive else "进程死亡/无 pid"}


def check_heartbeat(ws):
    """② heartbeat 新鲜度。"""
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
    """③ 最近 session_ended 原因分类。"""
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


def probe_api(model=None, timeout=120):
    """④ API 探活：pi 跑一次最小 prompt（能捕捉 0-token 空回/配额类故障）。"""
    from shutil import which
    pi_bin = (os.environ.get("PI_BIN") or which("pi")
              or "/Users/logic/.rick/pi/agent/runtime/node_modules/.bin/pi")
    cmd = [pi_bin, "-p", "--no-session",
           "reply with the single word: pong"]
    if model:
        cmd += ["--model", model]
    try:
        r = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
        out = (r.stdout or "").strip()
        low = out.lower()
        ok = (r.returncode == 0 and len(out) > 0
              and "quota" not in low and "limit" not in low
              and "401" not in out and "429" not in out)
        return ok, (out[:120] or (r.stderr or "")[:120])
    except subprocess.TimeoutExpired:
        return False, "probe timeout %ss" % timeout
    except FileNotFoundError:
        return False, "pi CLI 不可用"


def current_runtime_model(ws):
    """读当前 runtime 模型配置（None=跟随 pi 默认）。"""
    sj = json.load(open(os.path.join(ws, "super.json")))
    am = (sj.get("config") or {}).get("agent_models") or {}
    return am.get("builder") or am.get("adjust") or None


def switch_runtime_model(ws, new_model):
    """动作手柄（中控调用）：切换 runtime 全角色模型。"""
    sj_path = os.path.join(ws, "super.json")
    sj = json.load(open(sj_path))
    sj.setdefault("config", {})["agent_models"] = {
        "builder": new_model, "coding": new_model, "debug": new_model,
        "adjust": new_model,
    }
    with open(sj_path, "w", encoding="utf-8") as f:
        json.dump(sj, f, ensure_ascii=False, indent=2)


def restart_super(ws, max_wall_time_h=None, max_iterations=None):
    """动作手柄（中控调用）：重启 super（续跑章法）。

    章法：①重启前 config 体检（历史残留预算/迭代帽——3h 墙杀 iter_67 判例）
          ②预算与目标匹配（剩余格数 × 单实验耗时 → 预算估算）
          ③带旗标启动（--max-wall-time / --max-iterations 覆写 config 残留）
    """
    rick = os.path.join(ROOT, "bin", "rick")
    name = os.path.basename(ws)
    outs = []
    # ③′ 混战清理（job_56 实测 2 次）：stop→start 竞态致旧 wrapper/py 残活
    # → 多 wrapper 各自拉 py → 互撞进程锁（exit 5）+ 群杀混战。启动前彻底
    # 清理：杀 wrapper + py + 等死透 + 清锁，确保单实例再启动。
    try:
        subprocess.run(["pkill", "-f", "rick super start --name %s" % name],
                       capture_output=True, timeout=30)
        subprocess.run(["pkill", "-f", "superopt --workdir %s" % ws],
                       capture_output=True, timeout=30)
        import time as _t
        deadline = _t.time() + 15
        while _t.time() < deadline:
            r = subprocess.run(
                ["pgrep", "-f", "superopt --workdir %s" % ws],
                capture_output=True, text=True)
            if not r.stdout.strip():
                break
            _t.sleep(1)
        subprocess.run(["pkill", "-9", "-f", "superopt --workdir %s" % ws],
                       capture_output=True, timeout=30)
        try:
            os.remove(os.path.join(ws, "env", "scheduler.pid"))
        except OSError:
            pass
        _t.sleep(2)
        outs.append("混战清理: wrapper/py 全清 + 锁清理完成")
    except Exception as exc:  # noqa: BLE001 清理失败继续 start（best-effort）
        outs.append("混战清理（软失败）: %s" % str(exc)[:60])
    # ① config 体检：残留值上报（不改，重启旗标覆盖它）
    try:
        sj = json.load(open(os.path.join(ws, "super.json")))
        cfg = sj.get("config") or {}
        outs.append("config 体检: wall=%sh(残留) iter帽=%s(残留) cursor=%s"
                    % (cfg.get("max_wall_time_h"), cfg.get("max_iterations"),
                       sj.get("iteration_cursor")))
    except Exception as exc:
        outs.append("config 体检失败: %s" % str(exc)[:60])
    # ② 剩余工作量估算
    try:
        conn = _db(ws)
        cells = conn.execute("SELECT count(*) FROM cells").fetchone()[0]
        adm = conn.execute(
            "SELECT count(DISTINCT cell_id) FROM experiments WHERE status='admitted'"
        ).fetchone()[0]
        conn.close()
        remaining = cells - adm
        est_h = remaining * 0.7 / 8  # 每实验 ~0.7h、8 并发
        outs.append("剩余 %d 格 ≈ %.1fh 纯执行（8 并发 × 0.7h/实验）" % (remaining, est_h))
    except Exception:
        outs.append("工作量估算失败（跳过）")
    # ③ stop → start（带旗标）
    start = [rick, "super", "start", "--name", name, "--daemon"]
    if max_wall_time_h:
        start += ["--max-wall-time", str(max_wall_time_h)]
    if max_iterations:
        start += ["--max-iterations", str(max_iterations)]
    for cmd in ([rick, "super", "stop", "--name", name], start):
        try:
            r = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
            outs.append(((r.stdout or "") + (r.stderr or "")).strip()[:100])
        except Exception as exc:
            outs.append(str(exc)[:100])
    return outs


def downtime_report(ws):
    """⑤ 停机时长统计（ESQ D4 uptime_ratio 同口径）。"""
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
    down = 0.0
    last_end = None
    for etype, ts in rows:
        ep = _iso_to_epoch(ts)
        if etype == "session_ended":
            last_end = ep
        elif etype == "session_started" and last_end is not None:
            down += max(0, ep - last_end)
            last_end = None
    if last_end is not None:
        down += time.time() - last_end
    uptime = 1.0 - (down / total) if total > 0 else 1.0
    return {"evolution_start": first[:19], "total_s": int(total),
            "down_s": int(down), "uptime_ratio": round(uptime, 4)}


def _brief(ws):
    """播报三件套：adjust 决策路径 / 精英排序 / human 待裁决（每小时巡检播报增强）。"""
    try:
        conn = _db(ws)
        # ① 最近 2 条 adjust_decision 的提案摘要（决策路径）
        decisions = []
        for row in conn.execute(
                "SELECT ts, payload FROM events WHERE event_type="
                "'adjust_decision' ORDER BY seq DESC LIMIT 2").fetchall():
            try:
                p = json.loads(row[1])
                props = p.get("proposals") or []
                dsum = "; ".join(
                    "%s:%s" % (str(x.get("type", "?"))[:16],
                               str(x.get("target_cell") or x.get("target")
                                   or x.get("key") or "?")[:30])
                    for x in props[:3]) or "无提案"
                decisions.append("%s [%s]" % (row[0][11:16], dsum))
            except Exception:
                pass
        # ② 精英 top-3
        elites = [("%s=%.1f×" % (r[0][8:38], r[1])) for r in conn.execute(
            "SELECT cell_id, score FROM experiments WHERE status='admitted' "
            "ORDER BY score DESC LIMIT 3").fetchall()]
        # ③ human 待裁决（proposal 队列未 applied）
        pending = conn.execute(
            "SELECT count(*) FROM humans WHERE queue_type='proposal' "
            "AND applied=0").fetchone()[0]
        conn.close()
        return {"adjust_decisions": decisions or ["无"],
                "elites_top3": elites or ["无"],
                "human_pending": pending}
    except Exception:
        return {"adjust_decisions": ["采集失败"], "elites_top3": ["采集失败"],
                "human_pending": "?"}


def collect_report(ws, probe_timeout):
    """默认模式：纯采集（无动作）。"""
    proc = check_process(ws)
    hb = check_heartbeat(ws)
    sess = check_last_session(ws)
    cur_model = current_runtime_model(ws)

    # API 探活只在「系统看起来异常」或「runtime 有专属模型配置」时执行
    # （健康且用默认模型时跳过——省一次探活调用，中控唤醒后可自行补测）
    needs_probe = (not proc["ok"]) or (not hb["ok"]) or cur_model is not None
    api_probe = {"skipped": True, "reason": "系统健康且用默认模型"}
    if needs_probe:
        ok, detail = probe_api(model=cur_model, timeout=probe_timeout)
        api_probe = {"model": cur_model or "(default)", "ok": ok,
                     "detail": detail}

    # 健康判定：进程活 + heartbeat 新鲜（session_ended 是历史记录——
    # 进程已重启后不应继续标记异常；只在进程死时才有当下意义）
    healthy = proc["ok"] and hb["ok"]
    return {
        "ts": _now_ts(),
        "workspace": ws,
        "status": "healthy" if healthy else "needs_attention",
        "checks": {"process": proc, "heartbeat": hb, "last_session": sess},
        "brief": _brief(ws),
        "runtime_model": cur_model or "(pi default)",
        "api_probe": api_probe,
        "downtime": downtime_report(ws),
        "中控提示": ("全部正常，无需动作" if healthy else
                    "异常——中控请深入排查：①api_probe.ok=false → 探活备选模型"
                    "（probe_api(model=...)）→ 活的用 --switch-to 切换；"
                    "②进程死但 API 活 → --restart；③heartbeat 停滞 → 读 super.log"),
    }


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("workspace")
    ap.add_argument("--probe-timeout", type=int, default=120)
    ap.add_argument("--restart", action="store_true",
                    help="动作手柄（中控调用）：重启 super")
    ap.add_argument("--switch-to", metavar="MODEL",
                    help="动作手柄（中控调用）：切换 runtime 模型 + 重启")
    ap.add_argument("--max-wall-time", type=float, metavar="H",
                    help="续跑章法：重启时设墙预算（覆盖 config 残留）")
    ap.add_argument("--max-iterations", type=int, metavar="N",
                    help="续跑章法：重启时设迭代帽（覆盖 config 残留）")
    args = ap.parse_args()
    ws = os.path.abspath(args.workspace)

    # 动作模式（中控执行手柄）
    if args.restart:
        print(json.dumps({"action": "restart",
                          "max_wall_time_h": args.max_wall_time,
                          "max_iterations": args.max_iterations,
                          "output": restart_super(ws, args.max_wall_time,
                                                  args.max_iterations)},
                         ensure_ascii=False, indent=2))
        return 0
    if args.switch_to:
        switch_runtime_model(ws, args.switch_to)
        print(json.dumps({"action": "switch_to", "model": args.switch_to,
                          "restart_output": restart_super(ws)},
                         ensure_ascii=False, indent=2))
        return 0

    # 默认模式：采集报告（无动作）
    try:
        report = collect_report(ws, args.probe_timeout)
    except Exception as exc:
        print(json.dumps({"status": "script_error", "error": str(exc)[:300]},
                         ensure_ascii=False, indent=2))
        return 2
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if report["status"] == "healthy" else 1


if __name__ == "__main__":
    sys.exit(main())
