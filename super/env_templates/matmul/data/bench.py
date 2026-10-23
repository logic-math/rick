#!/usr/bin/env python3
"""data/bench.py — legacy 兼容薄转发 → `skills/benchmark.py`（三要素之「目标」）。

保留原因：旧 env（env_7/8/9）与外部引用仍按 `data/bench.py --candidate <path>` 调用。
三要素化后唯一实现位于 `skills/benchmark.py`（框架与 agent 共用同一评估方法）；
本文件**不含任何评估逻辑**，只把 argv 原样转交并透传退出码/输出。
"""
import os
import runpy
import sys


def main():
    here = os.path.dirname(os.path.abspath(__file__))
    target = os.path.join(here, "..", "skills", "benchmark.py")
    sys.argv = [target] + sys.argv[1:]
    runpy.run_path(target, run_name="__main__")


if __name__ == "__main__":
    main()
