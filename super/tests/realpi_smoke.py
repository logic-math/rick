#!/usr/bin/env python3
"""realpi_smoke.py — job_47/task14 realpi 冒烟（≥3 真迭代，J8 验收）

环境门控（照 rick realpi 冒烟先例：executor_realpi_smoke_test.go 的
RICK_REAL_PI_ARGS 模式）：RICK_REAL_PI_ARGS 未设 → skipTest（exit 0，
不阻塞 CI）；gate4 对 realpi 段只验证脚本存在与协议，真跑证据人工贴
doing/rsi/gates.md。

RICK_REAL_PI_ARGS 格式（空格分隔，首 token 必填）：
    "<pi_path> [额外 CLI 参数...]"
例：
    RICK_REAL_PI_ARGS="$HOME/.rick/pi/agent/runtime/node_modules/.bin/pi" \
        python3 super/tests/realpi_smoke.py

流程（真 pi 池 + 真门 + Go 监督，仅 agent 是真的）：matmul 模板生成 env
（venv numpy）→ rick super start --daemon（max-iterations=3）→ 等 ≥3 真迭代
完成 → stop → 断言：
  - iteration_ended ≥ 3
  - 每迭代 builder+coding 的 agent_started 齐（四角色池的真实派发面；
    debug/adjust 为失败/周期路径，不在直线迭代必现——注释留痕）
  - token 聚合非零（pi usage 计量链）
  - ≥1 gate_result（框架终验真跑）
注：max-iterations 到点后调度器 exit 0，Go 监督者视作死亡并重启（完成≠
崩溃不可分辨——已提交代码的已知语义）；重启会话不产生新迭代（恢复后
cursor≥max 即退），本脚本容忍多会话、以事件流为证。

stdlib-only，Python >=3.9。
"""
import json
import os
import subprocess
import sys
import time
import unittest

_HERE = os.path.dirname(os.path.abspath(__file__))
_SUPER = os.path.dirname(_HERE)
_REPO = os.path.dirname(_SUPER)
for _p in (_SUPER, _REPO):
    if _p not in sys.path:
        sys.path.insert(0, _p)


def _read_all(ws):
    out = []
    path = os.path.join(ws, "events.jsonl")
    if not os.path.exists(path):
        return out
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                try:
                    out.append(json.loads(line))
                except ValueError:
                    pass
    return out


def _count(ws, etype, pred=None):
    return sum(1 for e in _read_all(ws)
               if e.get("type") == etype and (pred is None or pred(e)))


def _wait_until(fn, timeout_s, what):
    deadline = time.time() + timeout_s
    while time.time() < deadline:
        v = fn()
        if v:
            return v
        time.sleep(0.5)
    raise AssertionError("等待超时（%ss）：%s" % (timeout_s, what))


class RealPiSmoke(unittest.TestCase):
    """≥3 真迭代（真 pi agent 池 + 真门 + Go 监督链）。"""

    def test_real_pi_smoke(self):
        raw = os.environ.get("RICK_REAL_PI_ARGS", "").strip()
        if not raw:
            self.skipTest("RICK_REAL_PI_ARGS 未设置（realpi 冒烟为环境门控）")
        parts = raw.split()
        pi_path = parts[0]
        extra = parts[1:]
        if not os.path.exists(pi_path):
            self.skipTest("RICK_REAL_PI_ARGS 的 pi_path 不存在: %s" % pi_path)

        # 延迟导入（skip 路径零依赖）
        from e2e_test import (
            _ensure_master_venv, _ensure_rick_bin, _master_root,
            make_env_dir, make_gates_dir, write_super_json,
            write_go_home_config, wait_until,
        )
        import shutil

        venv = _ensure_master_venv()
        if venv is None:
            self.skipTest("numpy venv 不可用（离线？）")
        rick = _ensure_rick_bin()
        self.assertIsNotNone(rick, "go build 失败")

        name = "realpi-smoke"
        home = os.path.join(_master_root(), "home-realpi")
        os.makedirs(home, exist_ok=True)
        write_go_home_config(home)
        ws = os.path.join(home, ".rick", "super", name)
        if os.path.exists(ws):
            shutil.rmtree(ws)
        env_dir = make_env_dir(ws)
        make_gates_dir(ws)
        if extra:
            print("NOTE: RICK_REAL_PI_ARGS 额外参数暂不透传（pi_path=%s）"
                  % pi_path)
        write_super_json(ws, env_dir, pi_path, max_iterations=3)

        env = dict(os.environ)
        env["HOME"] = home
        try:
            r = subprocess.run(
                [rick, "super", "start", "--name", name, "--env", env_dir,
                 "--daemon"], env=env, capture_output=True, text=True,
                timeout=120)
            self.assertEqual(r.returncode, 0,
                             "rick super start 失败: %s%s" % (r.stdout, r.stderr))

            timeout_s = float(os.environ.get("RICK_REAL_PI_TIMEOUT_S", "1800"))
            _wait_until(lambda: _count(ws, "iteration_ended") >= 3 or None,
                        timeout_s, "≥3 真迭代完成（真 pi agent 池）")
        finally:
            try:
                subprocess.run([rick, "super", "stop", "--name", name],
                               env=env, capture_output=True, text=True,
                               timeout=120)
            except Exception as exc:  # noqa: BLE001
                print("WARN: stop 异常: %s" % exc)

        # ---- 断言（事件流为证） ----
        iters = _count(ws, "iteration_ended")
        self.assertGreaterEqual(iters, 3, "真迭代不足：%d < 3" % iters)

        events = _read_all(ws)
        it_ids = sorted({e["payload"].get("iteration_id")
                         for e in events
                         if e["type"] == "iteration_started"})
        for it in it_ids[-3:]:
            roles = {e["payload"].get("role") for e in events
                     if e["type"] == "agent_started"
                     and e["payload"].get("iteration_id") == it}
            self.assertTrue({"builder", "coding"} <= roles,
                            "迭代 %s agent 角色不全：%s" % (it, roles))

        tokens = 0.0
        for e in events:
            if e["type"] == "agent_ended":
                u = (e.get("payload") or {}).get("usage") or {}
                tokens += float(u.get("total_tokens") or 0)
        self.assertGreater(tokens, 0.0, "token 聚合为 0（pi usage 计量链？）")

        self.assertGreaterEqual(_count(ws, "gate_result"), 1,
                                "无 gate_result 事件（框架终验未跑）")
        print("REALPI_SMOKE OK: iterations=%d tokens=%.0f gate_results=%d"
              % (iters, tokens, _count(ws, "gate_result")))


if __name__ == "__main__":
    unittest.main(verbosity=2)
