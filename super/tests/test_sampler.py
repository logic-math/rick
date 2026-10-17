"""test_sampler.py — Sampler 单元测试（job_47 / task3）。

跑法（仓库根）：
    python3 -m unittest super.tests.test_sampler -v

这是修复 v5 两大采样缺陷的回归测试集：
1. visit 断链（db.py:201 零调用）→ increment_visit 唯一递增入口（on_visit 回调写 archive）
2. UCB 退化 / StrategyAgent 权重接管 → 纯 UCB1 + 先验权重，无启发式

覆盖：冷启动先验加权随机 / UCB argmax（手工算例）/ visit 真实递增 /
灭绝格永不被采 / 未探索格必被探索（ucb=inf）/ 软降权乘性折扣 /
混合态平局集权重随机 / 空间外脏格忽略 / 构造校验。
"""
import math
import random
import unittest

from super.superopt.feature_space import FeatureSpace
from super.superopt.sampler import Sampler


class FakeArchive:
    """内存假 Archive（鸭子类型：cells() 返回副本 / bump_visit 递增）。

    与真 Archive（task2）接口约定一致：cells() -> {cell_id: {visit, weight,
    elite, ...}} 副本；visit 递增经 bump_visit（loop 集成时由 on_visit 接线）。
    """

    def __init__(self, cells=None):
        self._cells = cells or {}

    def cells(self):
        return {k: dict(v) for k, v in self._cells.items()}

    def bump_visit(self, cell_id):
        entry = self._cells.setdefault(cell_id, {})
        entry["visit"] = entry.get("visit", 0) + 1


def one_axis_fs(values):
    return FeatureSpace([
        {"name": "s", "type": "enum", "values": list(values), "desc": "策略轴"},
    ])


class TestColdStart(unittest.TestCase):
    """冷启动：全格无 visit → 先验权重加权随机（R2-3 裁决）。"""

    def test_weighted_random_ratio(self):
        # 权重 9:1 两格 → 200 次采样选中比 > 3:1（固定 rng 无 flake）
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 0, "weight": 9.0},
            "s=b": {"visit": 0, "weight": 1.0},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(42))
        counts = {"s=a": 0, "s=b": 0}
        for _ in range(200):
            counts[s.sample_cell()] += 1
        self.assertGreater(counts["s=a"], counts["s=b"] * 3,
                           "权重 9:1 下选中比应 > 3:1，实际 %r" % counts)

    def test_unrecorded_cell_default_weight_one(self):
        # archive 无记录 = weight 1.0：均匀可采
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive()
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(11))
        picked = {s.sample_cell() for _ in range(30)}
        self.assertEqual(picked, {"s=a", "s=b"})

    def test_cold_param_forces_cold_path(self):
        # cold=True 强制冷启动路径（无视 visit 数据）：低权格仍有机会被采到
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 5, "weight": 9.0, "elite": {"score": 0.9}},
            "s=b": {"visit": 1, "weight": 1.0},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(7))
        picked = {s.sample_cell(cold=True) for _ in range(50)}
        self.assertIn("s=b", picked,
                      "冷路径下 weight=1.0 的格 50 次内应至少命中一次")


class TestExtinctCells(unittest.TestCase):
    """灭绝格（weight=0，功能性灭绝）：任何态下永不被采。"""

    def test_zero_weight_never_sampled(self):
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 3, "weight": 0.0, "elite": {"score": 0.99}},
            "s=b": {"visit": 1, "weight": 1.0, "elite": {"score": 0.3}},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(1))
        for _ in range(100):
            self.assertNotEqual(s.sample_cell(), "s=a")

    def test_zero_weight_ignored_in_cold_start_too(self):
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 0, "weight": 0.0},
            "s=b": {"visit": 0, "weight": 1.0},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(2))
        for _ in range(50):
            self.assertEqual(s.sample_cell(), "s=b")

    def test_all_extinct_raises(self):
        fs = one_axis_fs(["a"])
        arch = FakeArchive({"s=a": {"visit": 2, "weight": 0.0}})
        s = Sampler(archive=arch, feature_space=fs)
        with self.assertRaises(ValueError):
            s.sample_cell()


class TestUCB(unittest.TestCase):
    """热态 UCB1：argmax(mean_norm + C*sqrt(ln(total)/visit))。"""

    def _three_hot(self):
        fs = one_axis_fs(["a", "b", "c"])
        cells = {
            "s=a": {"visit": 10, "weight": 1.0, "elite": {"score": 0.8}},
            "s=b": {"visit": 2, "weight": 1.0, "elite": {"score": 0.6}},
            "s=c": {"visit": 30, "weight": 1.0, "elite": {"score": 0.9}},
        }
        return fs, FakeArchive(cells)

    def test_picks_ucb_argmax(self):
        # 手工算例（C=0.7, total=42）：
        #   a: 0.8+0.7*sqrt(ln42/10)=1.2280  b: 0.6+0.7*sqrt(ln42/2)=1.5566
        #   c: 0.9+0.7*sqrt(ln42/30)=1.1473  → b 的探索项压倒 c 的均值
        fs, arch = self._three_hot()
        s = Sampler(archive=arch, feature_space=fs)
        self.assertEqual(s.sample_cell(), "s=b")

    def test_ucb_score_formula(self):
        fs, arch = self._three_hot()
        s = Sampler(archive=arch, feature_space=fs)
        self.assertAlmostEqual(
            s.ucb_score("s=b", 42), 0.6 + 0.7 * math.sqrt(math.log(42) / 2),
            places=9)
        self.assertAlmostEqual(
            s.ucb_score("s=a", 42), 0.8 + 0.7 * math.sqrt(math.log(42) / 10),
            places=9)

    def test_visit_zero_ucb_is_inf(self):
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 5, "weight": 1.0, "elite": {"score": 0.95}},
        })
        s = Sampler(archive=arch, feature_space=fs)
        self.assertEqual(s.ucb_score("s=b", 5), math.inf)

    def test_unvisited_cell_must_be_explored(self):
        # 混合态存在 visit=0 格 → 采样必选冷格（ucb=inf 平局集，探索压力真实）
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 5, "weight": 1.0, "elite": {"score": 0.95}},
        })
        s = Sampler(archive=arch, feature_space=fs)
        for _ in range(20):
            self.assertEqual(s.sample_cell(), "s=b")

    def test_sample_is_pure_computation(self):
        # sample_cell 不改 archive（visit 递增是调用方显式 increment_visit）
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 5, "weight": 1.0, "elite": {"score": 0.95}},
        })
        before = arch.cells()
        s = Sampler(archive=arch, feature_space=fs)
        s.sample_cell()
        self.assertEqual(arch.cells(), before)

    def test_no_elite_mean_is_neutral(self):
        # visit>0 但无精英：exploit 项取 0.5 中性值
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 4, "weight": 1.0},  # 无 elite
            "s=b": {"visit": 4, "weight": 1.0, "elite": {"score": 0.5}},
        })
        s = Sampler(archive=arch, feature_space=fs)
        # 两格 UCB 同分（0.5 + 同探索项）→ 平局集 {a,b} 均匀随机
        rng = random.Random(5)
        s = Sampler(archive=arch, feature_space=fs, rng=rng)
        picked = {s.sample_cell() for _ in range(20)}
        self.assertEqual(picked, {"s=a", "s=b"})

    def test_score_clamped_to_unit_interval(self):
        # elite.score 越界（0-100 制 / 负值）→ clamp 到 [0,1]
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 2, "weight": 1.0, "elite": {"score": 87.0}},
            "s=b": {"visit": 2, "weight": 1.0, "elite": {"score": -3.0}},
        })
        s = Sampler(archive=arch, feature_space=fs)
        self.assertEqual(s.ucb_score("s=a", 4), 1.0 + 0.7 * math.sqrt(math.log(4) / 2))
        self.assertEqual(s.ucb_score("s=b", 4), 0.0 + 0.7 * math.sqrt(math.log(4) / 2))


class TestSoftWeight(unittest.TestCase):
    """软降权（R2-2）：weight 乘性折扣采样分，weight=0 即灭绝。"""

    def test_weight_scales_score_deterministically(self):
        # 同 UCB 分、weight 1.0 vs 0.1 → 低权格 score 打折永不被选
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 5, "weight": 1.0, "elite": {"score": 0.9}},
            "s=b": {"visit": 5, "weight": 0.1, "elite": {"score": 0.9}},
        })
        s = Sampler(archive=arch, feature_space=fs)
        for _ in range(10):
            self.assertEqual(s.sample_cell(), "s=a")

    def test_low_weight_cell_can_win_when_ucb_higher(self):
        # 软降权不是硬禁：低权格 UCB 分显著更高时仍可胜出
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 50, "weight": 1.0, "elite": {"score": 0.6}},
            "s=b": {"visit": 1, "weight": 0.5, "elite": {"score": 0.6}},
        })
        s = Sampler(archive=arch, feature_space=fs)
        # total=51: a = 0.6+0.7*sqrt(ln51/50)=0.6+0.197=0.797 → score 0.797
        #          b = 0.6+0.7*sqrt(ln51/1)=0.6+2.745=3.345 → score 1.672 → b 胜
        self.assertEqual(s.sample_cell(), "s=b")


class TestIncrementVisit(unittest.TestCase):
    """visit 断链修复（v5 db.py:201 零调用的回归防线）。"""

    def test_updates_archive_via_callback(self):
        fs = one_axis_fs(["a"])
        arch = FakeArchive()
        s = Sampler(archive=arch, feature_space=fs, on_visit=arch.bump_visit)
        before = arch.cells().get("s=a", {}).get("visit", 0)
        s.increment_visit("s=a")
        self.assertEqual(arch.cells()["s=a"]["visit"], before + 1)

    def test_total_visit_reflects_increment(self):
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 1, "weight": 1.0, "elite": {"score": 0.5}},
        })
        s = Sampler(archive=arch, feature_space=fs, on_visit=arch.bump_visit)
        s.increment_visit("s=b")  # b: 0 → 1（热化）
        self.assertEqual(arch.cells()["s=b"]["visit"], 1)
        # total 现=2；ucb_score(a, 2) 用新快照计算
        self.assertAlmostEqual(
            s.ucb_score("s=a", 2), 0.5 + 0.7 * math.sqrt(math.log(2)),
            places=9)

    def test_requires_callback(self):
        # 未注入 on_visit 而调用 increment_visit → 显式报错（断链防御）
        fs = one_axis_fs(["a"])
        s = Sampler(archive=FakeArchive(), feature_space=fs)
        with self.assertRaises(ValueError):
            s.increment_visit("s=a")

    def test_rejects_cell_outside_space(self):
        fs = one_axis_fs(["a"])
        arch = FakeArchive()
        s = Sampler(archive=arch, feature_space=fs, on_visit=arch.bump_visit)
        with self.assertRaises(ValueError):
            s.increment_visit("s=ghost")

    def test_repeated_increments_accumulate(self):
        fs = one_axis_fs(["a"])
        arch = FakeArchive()
        s = Sampler(archive=arch, feature_space=fs, on_visit=arch.bump_visit)
        for _ in range(7):
            s.increment_visit("s=a")
        self.assertEqual(arch.cells()["s=a"]["visit"], 7)


class TestMixedStateTieBreak(unittest.TestCase):
    """混合态：冷格 ucb=inf 平局集内按先验权重随机。"""

    def test_cold_tie_broken_by_weight(self):
        # 1 热 + 2 冷（5:1）：冷格必被探索（热格永不出线），且高权冷格占多数
        fs = one_axis_fs(["h", "a", "b"])
        arch = FakeArchive({
            "s=h": {"visit": 5, "weight": 1.0, "elite": {"score": 0.9}},
            "s=a": {"visit": 0, "weight": 5.0},
            "s=b": {"visit": 0, "weight": 1.0},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(3))
        counts = {"s=a": 0, "s=b": 0, "s=h": 0}
        for _ in range(200):
            counts[s.sample_cell()] += 1
        self.assertEqual(counts["s=h"], 0, "冷格存在时热格不得出线")
        self.assertGreater(counts["s=a"], counts["s=b"] * 2,
                           "平局集 5:1 加权随机，选中比应 > 2:1，实际 %r" % counts)

    def test_hot_tie_broken_by_weight(self):
        # 热格对称构造（同 visit 同 score）→ 同分平局集，按权重随机
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 4, "weight": 3.0, "elite": {"score": 0.5}},
            "s=b": {"visit": 4, "weight": 1.0, "elite": {"score": 0.5}},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(9))
        counts = {"s=a": 0, "s=b": 0}
        for _ in range(100):
            counts[s.sample_cell()] += 1
        self.assertGreater(counts["s=a"], counts["s=b"])


class TestDirtyArchiveCells(unittest.TestCase):
    """空间是候选全集的权威：archive 中空间外脏格不参与采样。"""

    def test_cells_outside_space_ignored(self):
        fs = one_axis_fs(["a"])
        arch = FakeArchive({
            "s=a": {"visit": 1, "weight": 1.0, "elite": {"score": 0.5}},
            "ghost=v": {"visit": 99, "weight": 99.0, "elite": {"score": 1.0}},
        })
        s = Sampler(archive=arch, feature_space=fs)
        for _ in range(100):
            self.assertEqual(s.sample_cell(), "s=a")

    def test_missing_weight_entry_defaults(self):
        # archive 记录存在但缺 weight/visit 字段 → 默认 1.0/0
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({"s=a": {}})
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(4))
        picked = {s.sample_cell() for _ in range(30)}
        self.assertEqual(picked, {"s=a", "s=b"})

    def test_corrupt_weight_treated_as_extinct(self):
        # weight 非数值（脏数据）→ 保守视为灭绝（不可采），不炸采样循环
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 2, "weight": "broken"},
            "s=b": {"visit": 1, "weight": 1.0, "elite": {"score": 0.4}},
        })
        s = Sampler(archive=arch, feature_space=fs)
        for _ in range(20):
            self.assertEqual(s.sample_cell(), "s=b")


class TestConstruction(unittest.TestCase):
    def test_rejects_non_positive_c(self):
        fs = one_axis_fs(["a"])
        with self.assertRaises(ValueError):
            Sampler(archive=FakeArchive(), feature_space=fs, c=0)
        with self.assertRaises(ValueError):
            Sampler(archive=FakeArchive(), feature_space=fs, c=-0.5)

    def test_rejects_bad_archive_duck(self):
        fs = one_axis_fs(["a"])
        with self.assertRaises(ValueError):
            Sampler(archive=object(), feature_space=fs)

    def test_rejects_bad_feature_space_duck(self):
        with self.assertRaises(ValueError):
            Sampler(archive=FakeArchive(), feature_space=object())

    def test_rejects_non_callable_on_visit(self):
        fs = one_axis_fs(["a"])
        with self.assertRaises(ValueError):
            Sampler(archive=FakeArchive(), feature_space=fs, on_visit="not-callable")

    def test_custom_c_changes_choice(self):
        # C 极大 → 探索主导（visit 最低者胜）；C 极小 → 均值主导（score 最高者胜）
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 2, "weight": 1.0, "elite": {"score": 0.55}},
            "s=b": {"visit": 9, "weight": 1.0, "elite": {"score": 0.9}},
        })
        explore = Sampler(archive=arch, feature_space=fs, c=5.0)
        exploit = Sampler(archive=arch, feature_space=fs, c=0.01)
        # total=11: explore 项 a=5*sqrt(ln11/2)=6.88 → a≈7.43；b=5*sqrt(ln11/9)=3.24 → b≈4.14
        self.assertEqual(explore.sample_cell(), "s=a")
        self.assertEqual(explore.sample_cell(), "s=a")  # 纯计算：可重复
        # exploit 项可忽略：a≈0.564；b≈0.906 → b（均值主导）
        self.assertEqual(exploit.sample_cell(), "s=b")


if __name__ == "__main__":
    unittest.main()
