#!/usr/bin/env python3
"""skills/verify.py — agent 侧约束门**薄入口**（v2 布局，执行引用实现）。

v2 验证器隔离（job_56/task2，human 裁决 Q5）：验证**实现**位于 `gates/` 目录
（builder/coding/debug **禁读其内容**——防针对验证器优化/作弊；adjust 可读写维护）。
本入口只做**执行引用**（subprocess 调 `gates/verify_impl.py`，透传 argv/输出/退出码）：
- agent 调用本入口 = 合法（拿同一份约束结论，与框架终验同源）
- 直接读 `gates/verify_impl.py` 内容 = 违规（superopt-json-guard hook 拦截 read/bash 落在
  gates/ 路径的**读取**行为；执行类调用经本入口或 gate_check.py 走）

输出协议与实现侧一致（stdout 最后一行 JSON：{"passed", "max_error", ...}）。
用法：
    python3 skills/verify.py --candidate <path> [--env-dir <env>]
"""
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
IMPL = os.path.join(HERE, "..", "gates", "verify_impl.py")

if __name__ == "__main__":
    r = subprocess.run([sys.executable, IMPL] + sys.argv[1:],
                       capture_output=True, text=True)
    sys.stdout.write(r.stdout)
    sys.stderr.write(r.stderr)
    sys.exit(r.returncode)
