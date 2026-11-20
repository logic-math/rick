#!/usr/bin/env python3
"""skills/benchmark.py — agent 侧基准门**薄入口**（v2 布局，执行引用实现）。

v2 验证器隔离（job_56/task2，human 裁决 Q5）：基准**实现**位于 `gates/` 目录
（builder/coding/debug **禁读其内容**；adjust 可读写维护）。本入口只做**执行引用**
（subprocess 调 `gates/benchmark_impl.py`，透传 argv/输出/退出码）。

支持分层验证参数（验证金字塔，job_56/task2 KR-D）：
    python3 skills/benchmark.py --candidate <path> [--level L1|L2|L3|L{n}|L{n}-M{m}|E2E]
`--level` 透传给实现侧（实现按级别裁剪规模/次数——分钟级快速验证到端到端全量）。

输出协议与实现侧一致（stdout 最后一行 JSON：{"runtime_ms", "score", ...}）。
"""
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
IMPL = os.path.join(HERE, "..", "gates", "benchmark_impl.py")

if __name__ == "__main__":
    r = subprocess.run([sys.executable, IMPL] + sys.argv[1:],
                       capture_output=True, text=True)
    sys.stdout.write(r.stdout)
    sys.stderr.write(r.stderr)
    sys.exit(r.returncode)
