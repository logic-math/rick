#!/usr/bin/env python3
"""iter016 builder probe: m_halves np.matmul variants, interleaved round-robin.
Arms:
  A full plain np.matmul (ref floor)
  B parent form: halves np.matmul out= into fresh np.empty
  C sibling form: halves np.dot out= into fresh np.empty (diagnostic only)
  D halves np.matmul no-out + concatenate
  E halves np.matmul out= into persistent prefaulted buffer
  F halves np.matmul out= in two concurrent Python threads
  G stacked gufunc: np.matmul(a.reshape(2,mid,n), b, out=c(2,mid,n)) -> reshape view
"""
import json
import threading
import time

import numpy as np

np.seterr(divide="ignore", over="ignore", invalid="ignore")
n = 8192
rng = np.random.RandomState(42)
a = rng.rand(n, n).astype(np.float32)
b = rng.rand(n, n).astype(np.float32)
mid = n // 2

c_pers = np.empty((n, n), dtype=np.float32)
c_pers[:] = 0  # prefault once


def arm_A():
    return np.matmul(a, b)


def arm_B():
    c = np.empty((n, n), dtype=np.float32)
    np.matmul(a[:mid], b, out=c[:mid])
    np.matmul(a[mid:], b, out=c[mid:])
    return c


def arm_C():
    c = np.empty((n, n), dtype=np.float32)
    np.dot(a[:mid], b, out=c[:mid])
    np.dot(a[mid:], b, out=c[mid:])
    return c


def arm_D():
    c1 = np.matmul(a[:mid], b)
    c2 = np.matmul(a[mid:], b)
    return np.concatenate((c1, c2), axis=0)


def arm_E():
    np.matmul(a[:mid], b, out=c_pers[:mid])
    np.matmul(a[mid:], b, out=c_pers[mid:])
    return c_pers


def arm_F():
    c = np.empty((n, n), dtype=np.float32)

    def lo():
        np.matmul(a[:mid], b, out=c[:mid])

    def hi():
        np.matmul(a[mid:], b, out=c[mid:])

    t1 = threading.Thread(target=lo)
    t2 = threading.Thread(target=hi)
    t1.start()
    t2.start()
    t1.join()
    t2.join()
    return c


def arm_G():
    c = np.empty((2, mid, n), dtype=np.float32)
    np.matmul(a.reshape(2, mid, n), b, out=c)
    return c.reshape(n, n)


arms = [("A_full_plain", arm_A), ("B_parent_out", arm_B), ("C_dot_out", arm_C),
        ("D_noout_concat", arm_D), ("E_pers_out", arm_E), ("F_threaded", arm_F),
        ("G_stacked", arm_G)]

ref = np.matmul(a, b)
for name, fn in arms:
    r = fn()
    print("bitwise_eq", name, bool(np.array_equal(r, ref)), flush=True)

for name, fn in arms:
    fn()

res = {name: [] for name, _ in arms}
for rnd in range(5):
    for name, fn in arms:
        t0 = time.perf_counter()
        fn()
        res[name].append((time.perf_counter() - t0) * 1000.0)

out = {}
for name, _ in arms:
    ts = sorted(res[name])
    out[name] = {"med": round(ts[2], 1), "min": round(ts[0], 1),
                 "all": [round(t, 1) for t in res[name]]}
print(json.dumps(out))
