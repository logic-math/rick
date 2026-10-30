"""test_budget.py — superopt.budget 单元测试（job_47/task4 + 迭代3/task16）

覆盖：record_agent 聚合（数值精确）、summary 分 role 与总计、
零 token 降级标注（token_degraded）、within_budget 墙钟检查、
record_event 占位行为。
迭代3/task16 新增：total_tokens 查询、max_total_tokens 双轨预算
（触发/不触发/降级轨不锁死/两轨独立/向后兼容）。
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


class TestTokenBudget(unittest.TestCase):
    """迭代3/task16：max_total_tokens 双轨预算检查。"""

    def test_total_tokens_query(self):
        """total_tokens() 与 summary()[total][total_tokens] 同源同值。"""
        led = CostLedger(max_wall_time_h=24.0)
        led.record_agent("builder", wall_ms=100, tool_calls=1,
                         usage=_usage(100, 50, 150, 0.01))
        led.record_agent("coding", wall_ms=200, tool_calls=2,
                         usage=_usage(200, 100, 300, 0.03))
        self.assertEqual(led.total_tokens(), 450)
        self.assertEqual(led.summary()["total"]["total_tokens"], 450)

    def test_total_tokens_empty_ledger(self):
        led = CostLedger(max_wall_time_h=24.0)
        self.assertEqual(led.total_tokens(), 0)

    def test_token_limit_triggers_false(self):
        """累计 token 达到上限 → within_budget False（>= 判定：达到即耗尽）。"""
        led = CostLedger(max_wall_time_h=24.0, max_total_tokens=450)
        led.record_agent("builder", wall_ms=100, tool_calls=1,
                         usage=_usage(100, 50, 150, 0.01))
        self.assertTrue(led.within_budget(), "300 < 450 仍在预算内")
        led.record_agent("coding", wall_ms=200, tool_calls=2,
                         usage=_usage(200, 100, 300, 0.03))
        self.assertEqual(led.total_tokens(), 450)
        self.assertFalse(led.within_budget(), "达到 450 上限 → 耗尽")

    def test_token_limit_exceeded(self):
        """单次记录直接超上限 → False。"""
        led = CostLedger(max_wall_time_h=24.0, max_total_tokens=100)
        led.record_agent("builder", wall_ms=100, tool_calls=1,
                         usage=_usage(90, 20, 110, 0.01))
        self.assertFalse(led.within_budget())

    def test_zero_tokens_unlimited(self):
        """max_total_tokens=0/负 = 不限 token（向后兼容默认）。"""
        led = CostLedger(max_wall_time_h=0, max_total_tokens=0)
        led.record_agent("builder", wall_ms=100, tool_calls=1,
                         usage=_usage(10**9, 10**9, 2 * 10**9, 1.0))
        self.assertTrue(led.within_budget())
        led2 = CostLedger(max_wall_time_h=0, max_total_tokens=-5)
        led2.record_agent("builder", wall_ms=100, tool_calls=1,
                          usage=_usage(10, 5, 15, 0.0))
        self.assertTrue(led2.within_budget(), "负值同样视为不限")

    def test_degraded_usage_never_trips_token_limit(self):
        """provider 未上报 usage（全 0 降级轨）→ token 检查恒不触发，
        墙钟轨仍兜底（降级不能把系统锁死）。"""
        led = CostLedger(max_wall_time_h=24.0, max_total_tokens=10)
        for _ in range(5):
            led.record_agent("builder", wall_ms=100, tool_calls=1, usage=None)
        self.assertTrue(led.within_budget(), "全零 token 的降级轨不触发上限")

    def test_wall_and_token_tracks_independent(self):
        """两轨独立：墙钟耗尽 + token 不限 → False；反向亦然。"""
        led = CostLedger(max_wall_time_h=1.0,
                         wall_start=time.time() - 7200,
                         max_total_tokens=0)
        led.record_agent("builder", wall_ms=1, tool_calls=0,
                         usage=_usage(1, 1, 2, 0.0))
        self.assertFalse(led.within_budget(), "墙钟耗尽")
        led2 = CostLedger(max_wall_time_h=0, max_total_tokens=1)
        led2.record_agent("builder", wall_ms=1, tool_calls=0,
                          usage=_usage(1, 1, 2, 0.0))
        self.assertFalse(led2.within_budget(), "token 耗尽")

    def test_default_constructor_unaffected(self):
        """向后兼容：不传 max_total_tokens 的既有构造行为不变。"""
        led = CostLedger(max_wall_time_h=1.0,
                         wall_start=time.time() + 3600)
        led.record_agent("builder", wall_ms=1, tool_calls=0,
                         usage=_usage(10**6, 10**6, 2 * 10**6, 5.0))
        self.assertTrue(led.within_budget(),
                        "默认 0=不限 token，仅墙钟轨生效")
        self.assertEqual(led.max_total_tokens, 0)


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
