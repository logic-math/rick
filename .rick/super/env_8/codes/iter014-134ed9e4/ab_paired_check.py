"""Paired A/B: parent (iter007 canonical) vs candidate (iter014 operand-flip).

门禁纪律：交替配对 ≥15 轮各取中位，统计配对差与同向率。
"""
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")


def parent_matmul(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
    blocks = []
    for s, e in edges:
        if e > s:
            blocks.append(np.einsum('ij,jk->ik', a32[s:e], b32, optimize=True))
    if not blocks:
        return np.zeros((0,) + b32.shape[1:], dtype=np.float32)
    return np.concatenate(blocks, axis=0)


def cand_matmul(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    m = a32.shape[0]
    edges = [(i * m // 4, (i + 1) * m // 4) for i in range(4)]
    blocks = []
    for s, e in edges:
        if e > s:
            blocks.append(np.einsum('jk,ij->ik', b32, a32[s:e], optimize=True))
    if not blocks:
        return np.zeros((0,) + b32.shape[1:], dtype=np.float32)
    return np.concatenate(blocks, axis=0)


ROUNDS = 30
rng = np.random.RandomState(42)
a = rng.rand(2048, 2048).astype(np.float32)
b = rng.rand(2048, 2048).astype(np.float32)

# 正确性快查
assert np.array_equal(parent_matmul(a, b), cand_matmul(a, b)), "not bitwise identical"
assert np.array_equal(cand_matmul(a, b), np.matmul(a, b)), "not identical to matmul"
print("bitwise: parent==candidate==matmul @2048 OK")

# 预热
for _ in range(3):
    parent_matmul(a, b)
    cand_matmul(a, b)

p_t, c_t = [], []
for i in range(ROUNDS):
    if i % 2 == 0:
        t0 = time.perf_counter(); parent_matmul(a, b); p_t.append((time.perf_counter() - t0) * 1000)
        t0 = time.perf_counter(); cand_matmul(a, b); c_t.append((time.perf_counter() - t0) * 1000)
    else:
        t0 = time.perf_counter(); cand_matmul(a, b); c_t.append((time.perf_counter() - t0) * 1000)
        t0 = time.perf_counter(); parent_matmul(a, b); p_t.append((time.perf_counter() - t0) * 1000)

p_t = np.array(p_t)
c_t = np.array(c_t)
diffs = c_t - p_t  # 负 = 候选更快
same_dir = int(np.sum(diffs < 0))
print("parent median %.3f ms (min %.3f) | candidate median %.3f ms (min %.3f)" % (
    np.median(p_t), p_t.min(), np.median(c_t), c_t.min()))
print("paired diff median %.3f ms (mean %.3f) | candidate wins %d/%d = %.0f%%" % (
    np.median(diffs), diffs.mean(), same_dir, ROUNDS, 100.0 * same_dir / ROUNDS))
