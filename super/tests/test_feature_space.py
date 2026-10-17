"""test_feature_space.py — FeatureSpace 单元测试（job_47 / task3）。

跑法（仓库根）：
    python3 -m unittest super.tests.test_feature_space -v

覆盖：cell_id 确定性（键序无关/轴名排序/格式）、取值域校验、
正交性校验（同名/同语义描述）、describe_cell 往返、all_cells 笛卡尔积、
防御（空轴/未知类型/空 values/组合爆炸上限）。
"""
import unittest

from super.superopt.feature_space import FeatureSpace


def make_axes():
    """两轴 enum 特征空间（matmul 模板同构：3×2 = 6 格）。"""
    return [
        {"name": "impl_strategy", "type": "enum",
         "values": ["naive_loops", "blocked", "vectorized"], "desc": "实现策略"},
        {"name": "memory_layout", "type": "enum",
         "values": ["row_major", "tiled"], "desc": "内存布局"},
    ]


class TestCellIdDeterminism(unittest.TestCase):
    """格 id 确定性：同一描述子（任意键序）恒得同一 id。"""

    def test_same_descriptor_same_id(self):
        fs = FeatureSpace(make_axes())
        d = {"impl_strategy": "blocked", "memory_layout": "tiled"}
        self.assertEqual(fs.cell_id(d), fs.cell_id(dict(d)))

    def test_key_order_independent(self):
        fs = FeatureSpace(make_axes())
        d1 = {"impl_strategy": "blocked", "memory_layout": "tiled"}
        d2 = {"memory_layout": "tiled", "impl_strategy": "blocked"}
        self.assertEqual(fs.cell_id(d1), fs.cell_id(d2))

    def test_id_format_axis_name_sorted(self):
        fs = FeatureSpace(make_axes())
        cid = fs.cell_id({"impl_strategy": "blocked", "memory_layout": "tiled"})
        # 轴名排序：impl_strategy < memory_layout；name=value 以 | 连接
        self.assertEqual(cid, "impl_strategy=blocked|memory_layout=tiled")

    def test_id_different_descriptors_differ(self):
        fs = FeatureSpace(make_axes())
        a = fs.cell_id({"impl_strategy": "blocked", "memory_layout": "tiled"})
        b = fs.cell_id({"impl_strategy": "vectorized", "memory_layout": "tiled"})
        self.assertNotEqual(a, b)

    def test_non_str_values_normalized(self):
        # int 值规范化为 str（取值域与 descriptor 双侧一致）
        axes = [{"name": "k", "type": "enum", "values": [1, 2, 3], "desc": "k 轴"}]
        fs = FeatureSpace(axes)
        self.assertEqual(fs.cell_id({"k": 2}), "k=2")
        self.assertEqual(fs.cell_id({"k": "2"}), "k=2")


class TestCellIdValidation(unittest.TestCase):
    """cell_id 输入校验：脏描述子早失败。"""

    def test_rejects_unknown_axis(self):
        fs = FeatureSpace(make_axes())
        with self.assertRaises(ValueError):
            fs.cell_id({"impl_strategy": "blocked", "memory_layout": "tiled",
                        "ghost": "x"})

    def test_rejects_missing_axis(self):
        fs = FeatureSpace(make_axes())
        with self.assertRaises(ValueError):
            fs.cell_id({"impl_strategy": "blocked"})

    def test_rejects_value_out_of_domain(self):
        fs = FeatureSpace(make_axes())
        with self.assertRaises(ValueError):
            fs.cell_id({"impl_strategy": "quantum", "memory_layout": "tiled"})

    def test_rejects_empty_descriptor(self):
        fs = FeatureSpace(make_axes())
        with self.assertRaises(ValueError):
            fs.cell_id({})


class TestOrthogonality(unittest.TestCase):
    """正交性校验：同名/同语义轴冲突 raise。"""

    def test_duplicate_axis_name_raises(self):
        axes = [
            {"name": "a", "type": "enum", "values": ["x", "y"], "desc": "第一轴"},
            {"name": "a", "type": "enum", "values": ["p", "q"], "desc": "第二轴"},
        ]
        fs = FeatureSpace(axes)
        with self.assertRaises(ValueError):
            fs.validate_orthogonal()

    def test_duplicate_desc_raises(self):
        # 同语义代理判定：描述（strip 后）完全相同的两轴视为冗余
        axes = [
            {"name": "a", "type": "enum", "values": ["x"], "desc": "策略"},
            {"name": "b", "type": "enum", "values": ["y"], "desc": "策略 "},
        ]
        fs = FeatureSpace(axes)
        with self.assertRaises(ValueError):
            fs.validate_orthogonal()

    def test_valid_axes_pass(self):
        fs = FeatureSpace(make_axes())
        fs.validate_orthogonal()  # 不 raise 即通过

    def test_blank_desc_not_treated_as_semantic_dup(self):
        # 空 desc 不参与同语义判定（默认缺省合法）
        axes = [
            {"name": "a", "type": "enum", "values": ["x"], "desc": ""},
            {"name": "b", "type": "enum", "values": ["y"]},
        ]
        fs = FeatureSpace(axes)
        fs.validate_orthogonal()


class TestDescribeCell(unittest.TestCase):
    """describe_cell：格 id → 描述子（prompt 渲染用）。"""

    def test_roundtrip(self):
        fs = FeatureSpace(make_axes())
        d = {"impl_strategy": "vectorized", "memory_layout": "row_major"}
        cid = fs.cell_id(d)
        self.assertEqual(fs.describe_cell(cid), d)

    def test_invalid_id_returns_none(self):
        fs = FeatureSpace(make_axes())
        self.assertIsNone(fs.describe_cell("no_such=cell|at=all"))
        self.assertIsNone(fs.describe_cell("impl_strategy=quantum|memory_layout=tiled"))
        self.assertIsNone(fs.describe_cell(""))
        self.assertIsNone(fs.describe_cell("impl_strategy=blocked"))
        self.assertIsNone(fs.describe_cell("a=1|a=2"))


class TestAllCells(unittest.TestCase):
    """all_cells：全格笛卡尔积（sampler 冷启动候选全集）。"""

    def test_cartesian_product_size(self):
        fs = FeatureSpace(make_axes())
        cells = fs.all_cells()
        self.assertEqual(len(cells), 3 * 2)
        self.assertIn("impl_strategy=blocked|memory_layout=tiled", cells)
        self.assertEqual(len(set(cells)), len(cells))  # 无重复

    def test_deterministic_order(self):
        fs = FeatureSpace(make_axes())
        self.assertEqual(fs.all_cells(), fs.all_cells())
        # 首格 = 每轴第一个值（轴名序 + 值声明序）
        self.assertEqual(fs.all_cells()[0],
                         "impl_strategy=naive_loops|memory_layout=row_major")

    def test_single_axis_single_value(self):
        fs = FeatureSpace([{"name": "s", "type": "enum", "values": ["a"],
                            "desc": "单格空间"}])
        self.assertEqual(fs.all_cells(), ["s=a"])

    def test_huge_space_rejected(self):
        # 组合爆炸防御：3 轴 × 100 值 = 1e6 格 > 上限
        axes = [
            {"name": "ax%d" % i, "type": "enum",
             "values": [str(v) for v in range(100)], "desc": "轴 %d" % i}
            for i in range(3)
        ]
        fs = FeatureSpace(axes)
        with self.assertRaises(ValueError):
            fs.all_cells()


class TestAxisAccessors(unittest.TestCase):
    def test_axis_names_sorted(self):
        fs = FeatureSpace(make_axes())
        self.assertEqual(fs.axis_names, ["impl_strategy", "memory_layout"])

    def test_axes_copy(self):
        fs = FeatureSpace(make_axes())
        got = fs.axes
        self.assertEqual(len(got), 2)
        got[0]["name"] = "mutated"  # 改副本不得影响内部
        self.assertEqual(fs.axis_names, ["impl_strategy", "memory_layout"])


class TestConstructionValidation(unittest.TestCase):
    """构造期结构校验（正交性归 validate_orthogonal）。"""

    def test_empty_axes_rejected(self):
        with self.assertRaises(ValueError):
            FeatureSpace([])

    def test_non_list_rejected(self):
        with self.assertRaises(ValueError):
            FeatureSpace({"name": "a"})

    def test_missing_name_rejected(self):
        with self.assertRaises(ValueError):
            FeatureSpace([{"type": "enum", "values": ["x"], "desc": "d"}])

    def test_unknown_type_rejected(self):
        with self.assertRaises(ValueError):
            FeatureSpace([{"name": "a", "type": "galaxy", "values": ["x"],
                           "desc": "d"}])

    def test_enum_requires_nonempty_values(self):
        with self.assertRaises(ValueError):
            FeatureSpace([{"name": "a", "type": "enum", "values": [],
                           "desc": "d"}])

    def test_duplicate_normalized_values_rejected(self):
        # 规范化后重复（1 与 "1"）→ 取值域坍缩
        with self.assertRaises(ValueError):
            FeatureSpace([{"name": "a", "type": "enum", "values": [1, "1"],
                           "desc": "d"}])

    def test_type_defaults_to_enum(self):
        fs = FeatureSpace([{"name": "a", "values": ["x"], "desc": "d"}])
        self.assertEqual(fs.cell_id({"a": "x"}), "a=x")


if __name__ == "__main__":
    unittest.main()
