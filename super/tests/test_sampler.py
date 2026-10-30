"""test_sampler.py — Sampler 单元测试（job_47 / task15：纯加权随机）。

跑法（仓库根）：
    python3 -m unittest super.tests.test_sampler -v

迭代 3 裁决（J-i3-1/J-i3-2）后的采样契约：
- 采样 = 纯加权随机（概率 ∝ adjust 权重）；UCB 数学退役
- 分殖优先保留：未访问存活格先铺满（每格至少 1 次）
- 灭绝格（weight=0）永不被采；weight 允许 >1（相对份额语义）
- visit 递增入口/断链防御（v5 回归防线）不变
- weight_share()：采样份额可观测值（cell_sampled 事件字段，loop 侧接线）

覆盖：权重比例随机 / 灭绝不采 / 分殖覆盖 / weight_share / cold 强制路径 /
增量 visit 防线 / 空间外脏格忽略 / 构造校验。
"""
import random
import unittest

from super.superopt.feature_space import FeatureSpace
from super.superopt.sampler import Sampler


class FakeArchive:
    """内存假 Archive（鸭子类型：cells() 返回副本 / bump_visit 递增）。"""

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


class TestWeightedRandom(unittest.TestCase):
    """核心：概率 ∝ 权重的纯加权随机（无 UCB/无加成修饰）。"""

    def test_weight_ratio_3_to_1(self):
        # 权重 3:1 两格 → 200 次采样选中比在 2~3.5:1 之间（固定 rng 无 flake）
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 5, "weight": 3.0, "elite": {"score": 0.9}},
            "s=b": {"visit": 5, "weight": 1.0, "elite": {"score": 0.1}},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(42))
        counts = {"s=a": 0, "s=b": 0}
        for _ in range(200):
            counts[s.sample_cell()] += 1
        ratio = counts["s=a"] / counts["s=b"]
        self.assertGreaterEqual(ratio, 2.0,
                                "3:1 权重选中比应 >= 2:1，实际 %r" % counts)
        self.assertLessEqual(ratio, 3.5,
                              "3:1 权重选中比应 <= 3.5:1（无加成修饰），实际 %r" % counts)

    def test_equal_weights_uniform(self):
        fs = one_axis_fs(["a", "b", "c"])
        arch = FakeArchive({
            "s=a": {"visit": 2, "weight": 1.0},
            "s=b": {"visit": 7, "weight": 1.0, "elite": {"score": 0.99}},
            "s=c": {"visit": 1, "weight": 1.0},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(7))
        picked = {s.sample_cell() for _ in range(60)}
        self.assertEqual(picked, {"s=a", "s=b", "s=c"},
                         "等权（visit/score 不影响采样）三格都应被采到")

    def test_weight_above_one_scales_share(self):
        # J-i3-1：weight 允许 >1（相对份额语义）——2.0 = 平均份额两倍
        fs = one_axis_fs(["a", "b", "c"])
        arch = FakeArchive({
            "s=a": {"visit": 3, "weight": 2.0},
            "s=b": {"visit": 3, "weight": 1.0},
            "s=c": {"visit": 3, "weight": 1.0},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(42))
        counts = {"s=a": 0, "s=b": 0, "s=c": 0}
        for _ in range(300):
            counts[s.sample_cell()] += 1
        self.assertGreater(counts["s=a"], counts["s=b"],
                           "weight=2.0 的格份额应超过 weight=1.0：实际 %r" % counts)
        self.assertGreater(counts["s=a"], counts["s=c"])

    def test_elite_score_does_not_affect_sampling(self):
        # 采样不看精英分数（决策权全在权重）——高精英分低权格 vs 低分高权格
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 5, "weight": 1.0, "elite": {"score": 0.99}},
            "s=b": {"visit": 5, "weight": 9.0, "elite": {"score": 0.01}},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(3))
        counts = {"s=a": 0, "s=b": 0}
        for _ in range(200):
            counts[s.sample_cell()] += 1
        self.assertGreater(counts["s=b"], counts["s=a"] * 3,
                           "9:1 权重应压倒精英分数：实际 %r" % counts)

    def test_sample_is_pure_computation(self):
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 5, "weight": 1.0, "elite": {"score": 0.95}},
            "s=b": {"visit": 3, "weight": 1.0},
        })
        before = arch.cells()
        s = Sampler(archive=arch, feature_space=fs)
        s.sample_cell()
        self.assertEqual(arch.cells(), before)


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

    def test_negative_weight_treated_as_extinct(self):
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 1, "weight": -0.5},
            "s=b": {"visit": 1, "weight": 1.0},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(6))
        for _ in range(20):
            self.assertEqual(s.sample_cell(), "s=b")


class TestColonizeUnvisitedFirst(unittest.TestCase):
    """分殖优先：存在未访问存活格 → 仅在其中加权随机（全空间铺开）。"""

    def test_four_cells_colonized_before_revisit(self):
        # 4 格（含 1 个未访问灭绝格）：存活 3 格——前 3 采必铺满各 1 次
        fs = one_axis_fs(["a", "b", "c", "d"])
        arch = FakeArchive({
            "s=a": {"visit": 0, "weight": 9.0},   # 高权未访问
            "s=b": {"visit": 0, "weight": 0.1},   # 低权未访问（仍必被铺到）
            "s=c": {"visit": 0, "weight": 1.0},
            "s=d": {"visit": 0, "weight": 0.0},   # 灭绝未访问：不参与分殖
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(42),
                    on_visit=arch.bump_visit)
        picked = []
        for i in range(3):
            cid = s.sample_cell()          # 自动判定（cold=None）
            picked.append(cid)
            self.assertNotEqual(cid, "s=d", "灭绝格即使未访问也不可被采")
            s.increment_visit(cid)          # 调用方接线（loop 同型）
        self.assertEqual(sorted(picked), ["s=a", "s=b", "s=c"],
                         "分殖阶段应铺满全部存活格：实际 %r" % picked)
        # 第 4 次：全部存活格已访问 → 全量加权随机（可重访任意存活格）
        fourth = s.sample_cell()
        self.assertIn(fourth, ["s=a", "s=b", "s=c"])

    def test_partial_visited_colonizes_rest_first(self):
        # 混合态：a 已访问（高权），b/c 未访问 → 前两采必落在 b/c（分殖），
        # 不许被 a 的高权重抢走首访机会
        fs = one_axis_fs(["a", "b", "c"])
        arch = FakeArchive({
            "s=a": {"visit": 5, "weight": 9.0, "elite": {"score": 0.99}},
            "s=b": {"visit": 0, "weight": 1.0},
            "s=c": {"visit": 0, "weight": 0.001},   # 极低权未访问
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(3),
                    on_visit=arch.bump_visit)
        first = s.sample_cell()
        self.assertIn(first, ("s=b", "s=c"))
        s.increment_visit(first)
        second = s.sample_cell()
        self.assertIn(second, ("s=b", "s=c"))
        self.assertNotEqual(second, first)

    def test_cold_true_keeps_weighted_random_no_colonize(self):
        # cold=True 强制（测试语义）：全部候选按权重随机（含已访问格），
        # 不启用分殖优先
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 7, "weight": 9.0, "elite": {"score": 0.9}},
            "s=b": {"visit": 0, "weight": 1.0},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(9))
        picked = {s.sample_cell(cold=True) for _ in range(60)}
        self.assertEqual(picked, {"s=a", "s=b"},
                         "强制冷路径下高权已访问格应可被采到")


class TestWeightShare(unittest.TestCase):
    """weight_share()：采样份额可观测值（cell_sampled 事件字段，loop 接线）。"""

    def test_share_proportional_to_weight(self):
        fs = one_axis_fs(["a", "b", "c"])
        arch = FakeArchive({
            "s=a": {"visit": 2, "weight": 3.0},
            "s=b": {"visit": 5, "weight": 1.0},
            "s=c": {"visit": 1, "weight": 0.0},   # 灭绝：不计入分母
        })
        s = Sampler(archive=arch, feature_space=fs)
        self.assertAlmostEqual(s.weight_share("s=a"), 3.0 / 4.0)
        self.assertAlmostEqual(s.weight_share("s=b"), 1.0 / 4.0)
        self.assertEqual(s.weight_share("s=c"), 0.0)

    def test_share_of_unrecorded_cell(self):
        # 无记录格（默认权重 1.0）计入分母
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({"s=a": {"visit": 1, "weight": 1.0}})
        s = Sampler(archive=arch, feature_space=fs)
        self.assertAlmostEqual(s.weight_share("s=a"), 0.5)
        self.assertAlmostEqual(s.weight_share("s=b"), 0.5)

    def test_share_of_cell_outside_space(self):
        fs = one_axis_fs(["a"])
        arch = FakeArchive({"ghost=v": {"visit": 9, "weight": 9.0}})
        s = Sampler(archive=arch, feature_space=fs)
        self.assertEqual(s.weight_share("ghost=v"), 0.0)

    def test_all_extinct_share_zero(self):
        fs = one_axis_fs(["a"])
        arch = FakeArchive({"s=a": {"visit": 2, "weight": 0.0}})
        s = Sampler(archive=arch, feature_space=fs)
        self.assertEqual(s.weight_share("s=a"), 0.0)


class TestIncrementVisit(unittest.TestCase):
    """visit 断链修复（v5 db.py:201 零调用的回归防线）——采样不再用 visit，
    但递增必须落 archive 的防线不变（事件流/统计/自改进分析依赖它）。"""

    def test_updates_archive_via_callback(self):
        fs = one_axis_fs(["a"])
        arch = FakeArchive()
        s = Sampler(archive=arch, feature_space=fs, on_visit=arch.bump_visit)
        before = arch.cells().get("s=a", {}).get("visit", 0)
        s.increment_visit("s=a")
        self.assertEqual(arch.cells()["s=a"]["visit"], before + 1)

    def test_requires_callback(self):
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

    def test_visit_does_not_change_sampling_distribution(self):
        # visit 只是计数（不再参与决策）：递增后分布仍由权重决定
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 0, "weight": 3.0},
            "s=b": {"visit": 0, "weight": 1.0},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(42),
                    on_visit=arch.bump_visit)
        for _ in range(10):
            s.increment_visit("s=b")       # b 访问拉满
        counts = {"s=a": 0, "s=b": 0}
        for _ in range(200):
            counts[s.sample_cell()] += 1
        self.assertGreater(counts["s=a"], counts["s=b"],
                           "visit=10 的低权格不应挤占 3:1 权重分布：实际 %r" % counts)


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
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({"s=a": {}})
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(4))
        picked = {s.sample_cell() for _ in range(30)}
        self.assertEqual(picked, {"s=a", "s=b"})

    def test_corrupt_weight_treated_as_extinct(self):
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 2, "weight": "broken"},
            "s=b": {"visit": 1, "weight": 1.0, "elite": {"score": 0.4}},
        })
        s = Sampler(archive=arch, feature_space=fs)
        for _ in range(20):
            self.assertEqual(s.sample_cell(), "s=b")


class TestConstruction(unittest.TestCase):
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

    def test_deprecated_c_kwarg_accepted_and_ignored(self):
        # 迭代 3 兼容垫片：loop 构造点仍传 c= → 接受且忽略（task19 将清理）
        fs = one_axis_fs(["a", "b"])
        arch = FakeArchive({
            "s=a": {"visit": 3, "weight": 3.0},
            "s=b": {"visit": 3, "weight": 1.0},
        })
        s = Sampler(archive=arch, feature_space=fs, rng=random.Random(42), c=0.7)
        counts = {"s=a": 0, "s=b": 0}
        for _ in range(200):
            counts[s.sample_cell()] += 1
        self.assertGreater(counts["s=a"], counts["s=b"],
                           "c= 被忽略，分布仍由 3:1 权重决定：%r" % counts)

    def test_default_rng_is_module_random(self):
        # 不注入 rng → 用 random 模块（生产形态）；可采样不炸
        fs = one_axis_fs(["a"])
        arch = FakeArchive({"s=a": {"visit": 1, "weight": 1.0}})
        s = Sampler(archive=arch, feature_space=fs)
        self.assertIn(s.sample_cell(), ("s=a",))


if __name__ == "__main__":
    unittest.main()
