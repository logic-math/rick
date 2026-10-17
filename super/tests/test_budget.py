"""test_budget.py — superopt.budget 单元测试（job_47/task4）

覆盖：record_agent 聚合（数值精确）、summary 分 role 与总计、
零 token 降级标注（token_degraded）、within_budget 墙钟检查、
record_event 占位行为。
"""
import os
import sys
import time
import unittest

_HERE = os.path.dirname(os.path.abspath(__file__))
_SUPER = os.path.dirname(_HERE)
_REPO = os.path.dirname(_SUPER)
for _p in (_SUPER, _REPO):
    if _p not in sys.path:
        sys.path.insert(0, _p)

from superopt import budget
from superopt.budget import CostLedger


def _usage(inp, out, total, cost):
    return {"input_tokens": inp, "output_tokens": out,
            "cache_read": 0, "cache_write": 0, "total": total,
            "cost_usd": cost}


class TestRecordAndSummary(unittest.TestCase):
    def test_aggregation_per_role(self):
        led = CostLedger(max_wall_time_h=24.0)
        led.record_agent("builder", wall_ms=1000, tool_calls=2,
                         usage=_usage(100, 50, 150, 0.01))
        led.record_agent("builder", wall_ms=500, tool_calls=1,
                         usage=_usage(10, 5, 15, 0.002))
        led.record_agent("coding", wall_ms=2000, tool_calls=4,
                         usage=_usage(200, 100, 300, 0.03))
        s = led.summary()
        b = s["by_role"]["builder"]
        self.assertEqual(b["wall_ms"], 1500)
        self.assertEqual(b["tool_calls"], 3)
        self.assertEqual(b["input_tokens"], 110)
        self.assertEqual(b["output_tokens"], 55)
        self.assertEqual(b["total_tokens"], 165)
        self.assertAlmostEqual(b["cost_usd"], 0.012, places=6)
        c = s["by_role"]["coding"]
        self.assertEqual(c["wall_ms"], 2000)
        self.assertEqual(c["total_tokens"], 300)
        t = s["total"]
        self.assertEqual(t["wall_ms"], 3500)
        self.assertEqual(t["tool_calls"], 7)
        self.assertEqual(t["input_tokens"], 310)
        self.assertEqual(t["total_tokens"], 465)
        self.assertAlmostEqual(t["cost_usd"], 0.042, places=6)
        self.assertFalse(s.get("token_degraded", False))

    def test_usage_none_counts_zero(self):
        led = CostLedger(max_wall_time_h=24.0)
        led.record_agent("debug", wall_ms=10, tool_calls=0, usage=None)
        s = led.summary()
        self.assertEqual(s["by_role"]["debug"]["total_tokens"], 0)
        self.assertEqual(s["by_role"]["debug"]["wall_ms"], 10)

    def test_usage_missing_keys_tolerated(self):
        """usage 缺键（pi 事件 usage 可能 undefined 子字段）按 0 处理。"""
        led = CostLedger(max_wall_time_h=24.0)
        led.record_agent("coding", wall_ms=1, tool_calls=0,
                         usage={"input_tokens": 5})
        s = led.summary()
        self.assertEqual(s["by_role"]["coding"]["input_tokens"], 5)
        self.assertEqual(s["by_role"]["coding"]["total_tokens"], 5)

    def test_token_degraded_when_all_zero(self):
        """所有记录 token 为 0（provider 未上报）→ summary 标注降级。"""
        led = CostLedger(max_wall_time_h=24.0)
        led.record_agent("builder", wall_ms=100, tool_calls=2, usage=None)
        led.record_agent("coding", wall_ms=200, tool_calls=1,
                         usage=_usage(0, 0, 0, 0.0))
        s = led.summary()
        self.assertTrue(s.get("token_degraded", False),
                        "全零 token 须降级标注")

    def test_no_records_no_degradation(self):
        """无任何 agent 记录 → 不算降级（无事发生）。"""
        led = CostLedger(max_wall_time_h=24.0)
        s = led.summary()
        self.assertEqual(s["by_role"], {})
        self.assertFalse(s.get("token_degraded", False))

    def test_partial_tokens_not_degraded(self):
        """部分 role 有非零 token → 不降级（真实计量部分可用）。"""
        led = CostLedger(max_wall_time_h=24.0)
        led.record_agent("builder", wall_ms=100, tool_calls=2, usage=None)
        led.record_agent("coding", wall_ms=200, tool_calls=1,
                         usage=_usage(10, 5, 15, 0.01))
        s = led.summary()
        self.assertFalse(s.get("token_degraded", False))


class TestWithinBudget(unittest.TestCase):
    def test_within_budget_future_start(self):
        led = CostLedger(max_wall_time_h=1.0,
                         wall_start=time.time() + 3600)
        self.assertTrue(led.within_budget())

    def test_exceeded_budget(self):
        led = CostLedger(max_wall_time_h=1.0,
                         wall_start=time.time() - 7200)
        self.assertFalse(led.within_budget())

    def test_zero_means_unlimited(self):
        led = CostLedger(max_wall_time_h=0,
                         wall_start=time.time() - 10**9)
        self.assertTrue(led.within_budget(), "max_wall_time_h=0/负 = 不限时")

    def test_default_wall_start_is_now(self):
        before = time.time()
        led = CostLedger(max_wall_time_h=1.0)
        self.assertGreaterEqual(led.wall_start, before)
        self.assertLessEqual(led.wall_start, time.time())


class TestRecordEventPlaceholder(unittest.TestCase):
    def test_record_event_is_noop(self):
        """record_event 为空实现占位（事件即账，真相源在 events.jsonl）。"""
        led = CostLedger(max_wall_time_h=1.0)
        self.assertIsNone(led.record_event("agent_started", {"role": "x"}))
        self.assertIsNone(led.record_event("heartbeat", {}))
        # 不影响 summary
        self.assertEqual(led.summary()["by_role"], {})


if __name__ == "__main__":
    unittest.main()
