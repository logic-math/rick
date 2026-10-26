#!/usr/bin/env python3
"""test_solution.py — 迭代 15 TDD 测试（先红后绿）。

针对本次改动（BLAS 线程池配置：模块加载期 os.environ.setdefault
VECLIB_MAXIMUM_THREADS=6，置于 import numpy 之前）：

- test_module_sets_veclib_threads_to_6 / test_env_set_before_numpy_import /
  test_fresh_process_import_sets_env 是**区分性测试**：
  在父本（iter003 plain，无 env 设置）上必红，在目标实现上绿——
  证明测试真的在测本次改动（模块加载期 setdefault 本身）。
- test_external_config_not_overridden 验证 setdefault 语义（外部显式配置不被覆盖）。
- 其余为契约保持与泛化边界（父本上即绿，属不变量回归；骨架取自 iter014 测试、
  剔除 iter14 专属的持久缓冲结构断言，保留 fresh 分配语义断言）。

线程数不影响数值结果的依据：probes/final_confirm.json——default 与 v6 在
n=64/8192 的 sha256(uint32 view) 完全相同（BLAS 线程切 M×N 输出域，单元素 K
点积单线程固定序）。

运行：/Users/logic/workdir/rick-dev/.rick/super/env_9/env/.venv/bin/python test_solution.py
"""
import os
import subprocess
import sys

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")

HERE = os.path.dirname(os.path.abspath(__file__))
PY = sys.executable

# 区分性测试前置条件：建立干净基线（VECLIB_MAXIMUM_THREADS 未被外部设置），
# 再 import solution——候选实现应在模块加载期 setdefault("VECLIB_MAXIMUM_THREADS","6")。
os.environ.pop("VECLIB_MAXIMUM_THREADS", None)
import solution  # noqa: E402 （前置 pop 必须先于 import）


def _ref(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.matmul(a32, b32)


def _bitwise(x, y):
    return np.array_equal(np.asarray(x).view(np.uint32), np.asarray(y).view(np.uint32))


def _child_env_probe(env_extra=None):
    """fresh 子进程 import solution 后读 VECLIB_MAXIMUM_THREADS（镜像 benchmark 的
    fresh 进程加载路径）。env_extra 非空时预设该变量（测 setdefault 不覆盖语义）。"""
    env = {k: v for k, v in os.environ.items() if k != "VECLIB_MAXIMUM_THREADS"}
    if env_extra:
        env.update(env_extra)
    code = "import solution, os; print(os.environ.get('VECLIB_MAXIMUM_THREADS'))"
    r = subprocess.run([PY, "-c", code], cwd=HERE, env=env,
                       capture_output=True, text=True, timeout=120)
    return r.returncode, r.stdout.strip(), r.stderr.strip()


# ---------- 区分性测试（父本红 / 目标绿） ----------

def test_module_sets_veclib_threads_to_6():
    """干净基线下 import solution 后 VECLIB_MAXIMUM_THREADS 必须为 "6"——父本必红。"""
    got = os.environ.get("VECLIB_MAXIMUM_THREADS")
    assert got == "6", (
        "solution 模块加载未设置 VECLIB_MAXIMUM_THREADS=6（当前值=%r；"
        "父本 plain 形态无此设置——本测试应红）" % got)


def test_env_set_before_numpy_import():
    """源码时序纪律：setdefault 必须先于 numpy import（先于首次 BLAS 池创建）。"""
    with open(os.path.join(HERE, "solution.py"), encoding="utf-8") as f:
        src = f.read()
    i_set = src.find('os.environ.setdefault("VECLIB_MAXIMUM_THREADS"')
    assert i_set != -1, (
        "solution.py 未出现 VECLIB_MAXIMUM_THREADS setdefault（父本形态——本测试应红）")
    i_np = src.find("import numpy")
    assert i_np != -1, "solution.py 缺 numpy import"
    assert i_set < i_np, "setdefault 必须位于 numpy import 之前"


def test_fresh_process_import_sets_env():
    """fresh 子进程（env 干净）import solution 后读到 "6"——benchmark 加载路径同款。"""
    rc, out, err = _child_env_probe()
    assert rc == 0, "子进程 import solution 失败: %s" % err
    assert out == "6", (
        "fresh 进程加载后 VECLIB_MAXIMUM_THREADS=%r（父本形态应红）" % out)


# ---------- setdefault 语义（防走偏不变量） ----------

def test_external_config_not_overridden():
    """外部显式预设 VECLIB_MAXIMUM_THREADS=12 时，solution 不得覆盖（setdefault 语义）。"""
    rc, out, err = _child_env_probe({"VECLIB_MAXIMUM_THREADS": "12"})
    assert rc == 0, "子进程 import solution 失败: %s" % err
    assert out == "12", "外部显式配置被覆盖（应为 setdefault 语义）: %r" % out


# ---------- 契约保持（不变量回归，骨架取自 iter014） ----------

def test_contract_list_input():
    a = [[1.0, 2.0], [3.0, 4.0]]
    b = [[5.0, 6.0], [7.0, 8.0]]
    got = solution.matmul(a, b)
    ref = _ref(a, b)
    assert isinstance(got, np.ndarray)
    assert got.dtype == np.float32
    assert got.shape == (2, 2)
    assert _bitwise(got, ref)


def test_bitwise_vs_blas_anchor():
    """K 求和序不变：多规模双 seed 与 f32 BLAS 锚逐位一致（正确性核心；跨线程 SHA 证据）。"""
    for n in (64, 300, 512):
        for seed in (42, 123):
            rng = np.random.RandomState(seed)
            a = rng.rand(n, n).astype(np.float32)
            b = rng.rand(n, n).astype(np.float32)
            got = solution.matmul(a, b)
            assert _bitwise(got, _ref(a, b)), "非逐位一致 @ n=%d seed=%d" % (n, seed)


def test_double_run_bitwise():
    """镜像 verify 加固⑦：同输入两遍逐位一致（抓缓存/分支特判）。"""
    rng = np.random.RandomState(99)
    a = rng.rand(128, 128).astype(np.float32)
    b = rng.rand(128, 128).astype(np.float32)
    g1 = solution.matmul(a, b)
    g2 = solution.matmul(a, b)
    assert _bitwise(g1, g2)


def test_recompute_not_cached():
    """每次全量重算：换输入后新调用结果=新锚（红线：预存结果）。"""
    rng = np.random.RandomState(7)
    a = rng.rand(64, 64).astype(np.float32)
    b1 = rng.rand(64, 64).astype(np.float32)
    b2 = rng.rand(64, 64).astype(np.float32)
    c1 = solution.matmul(a, b1)
    assert _bitwise(c1, _ref(a, b1))
    c2 = solution.matmul(a, b2)
    assert _bitwise(c2, _ref(a, b2)), "返回缓存旧结果（红线：预存结果）"
    assert _bitwise(c1, _ref(a, b1)), "先前返回的内容被后续调用篡改"


def test_rectangular_shapes():
    """矩形 (n,k),(k,m) 签名：输出形状精确 (n,m)。"""
    rng = np.random.RandomState(5)
    a = rng.rand(37, 53).astype(np.float32)
    b = rng.rand(53, 29).astype(np.float32)
    got = solution.matmul(a, b)
    assert got.shape == (37, 29)
    assert _bitwise(got, _ref(a, b))


def test_non_2d_fallback():
    """1-D×2-D（matvec）走泛化兜底，结果与 np.matmul 一致。"""
    rng = np.random.RandomState(11)
    v = rng.rand(16).astype(np.float32)
    m = rng.rand(16, 8).astype(np.float32)
    got = solution.matmul(v, m)
    ref = np.matmul(np.asarray(v, np.float32), np.asarray(m, np.float32))
    assert _bitwise(np.asarray(got), ref)


def test_shapes_independent():
    """fresh 分配语义：不同形状输出互不别名，各自正确。"""
    rng = np.random.RandomState(42)
    a64 = rng.rand(64, 64).astype(np.float32)
    a512 = rng.rand(512, 512).astype(np.float32)
    b64 = rng.rand(64, 64).astype(np.float32)
    b512 = rng.rand(512, 512).astype(np.float32)
    c64 = solution.matmul(a64, b64)
    c512 = solution.matmul(a512, b512)
    assert not np.shares_memory(c64, c512)
    assert _bitwise(c64, _ref(a64, b64))
    assert _bitwise(c512, _ref(a512, b512))


def test_single_matmul_dispatch():
    """「唯一改动」纪律：单次调用恰一次 np.matmul 整阵派发、f32 入参、无 out=。"""
    rng = np.random.RandomState(42)
    a = rng.rand(64, 64).astype(np.float32)
    b = rng.rand(64, 64).astype(np.float32)
    calls = []
    orig = np.matmul

    def spy(x1, x2, *args, **kw):
        calls.append((x1.dtype, x2.dtype, args, kw))
        return orig(x1, x2, *args, **kw)

    np.matmul = spy
    try:
        solution.matmul(a, b)
    finally:
        np.matmul = orig
    assert len(calls) == 1, "应恰一次 np.matmul 派发，实际 %d 次" % len(calls)
    d1, d2, extra_args, kw = calls[0]
    assert d1 == np.float32 and d2 == np.float32, "入参须为 f32"
    assert not extra_args, "不应有额外位置参数"
    assert "out" not in kw, "不应使用 out=（父本 plain 形态）"


if __name__ == "__main__":
    fns = [v for k, v in sorted(globals().items()) if k.startswith("test_") and callable(v)]
    failed = 0
    for fn in fns:
        try:
            fn()
            print("PASS %s" % fn.__name__)
        except AssertionError as e:
            failed += 1
            print("FAIL %s: %s" % (fn.__name__, e))
    print("%d/%d passed" % (len(fns) - failed, len(fns)))
    raise SystemExit(1 if failed else 0)
