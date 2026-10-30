"""__main__.py — superopt 调度器进程入口（job_47/task9）

`python3 -m superopt --workdir <ws> [--max-iterations N] [--adjust-interval N]
[--max-wall-time H] [--preverify-red]`

运行契约（与 Go 侧 PySupervisor 对齐，task10/task11）：
  - spawn 形态：cwd = rick 仓库根，PYTHONPATH 锚定 <repoRoot>/super
    （`superopt` 包由此解析）；每迭代 subprocess 直调 pi 的热更新友好性
    由 loop.py 实现，本入口只做「配置装配 + 调度器启动」
  - --preverify-red：门先证红（exploit 拒识 100% + 金样例零误杀）先行，
    失败即退出非 0（loop 内部约定 exit 3）——Go 侧由 OnDead/退出码感知
  - 门禁失败/崩溃退出码语义归 loop；本入口传播 run() 的整型返回值

并行纪律（task9 spec）：不 import loop（task8 并行落地中）——Scheduler
延迟到 main() 内 import，保证 `python3 -m superopt --help` 在 loop 未落时
也可冒烟；运行时缺失给出可诊断的错误信息。

stdlib-only，Python >=3.9 兼容（禁 PEP604 X|Y 联合类型注解）。
"""
import argparse
import sys
from typing import List, Optional

from superopt import config


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="superopt",
        description="rick super 进化调度器（四状态循环：adjust → UCB 采样 → "
                    "builder → coding/debug → 级联门验证 → MAP-Elites 入格）。"
                    "唯一写者：events.jsonl。")
    parser.add_argument(
        "--workdir", required=True,
        help="super 工作区路径（含 super.json/events.jsonl/archive.json）")
    parser.add_argument(
        "--max-iterations", type=int, default=None,
        help="迭代上限（默认 0=无上限；测试/e2e 用）")
    parser.add_argument(
        "--adjust-interval", type=int, default=None,
        help="adjust agent 触发间隔（每 N 迭代，默认 5）")
    parser.add_argument(
        "--max-wall-time", type=float, default=None,
        help="墙钟预算（小时，默认 24；0/负 = 不限时）")
    parser.add_argument(
        "--preverify-red", action="store_true",
        help="启动先跑门先证红（exploit 拒识 100%% + 金样例零误杀），失败退出非 0")
    return parser


def main(argv: Optional[List[str]] = None) -> int:
    args = build_parser().parse_args(argv)

    # 配置装配：super.json（缺键补 DEFAULTS）→ CLI 覆盖（白名单键）
    cfg = config.load(args.workdir)
    cfg = config.merge_cli(cfg, args)
    # preverify-red 是行为 flag 而非 config 参数：顶层传递（Scheduler 契约）
    cfg["preverify_red"] = bool(args.preverify_red)

    effective = cfg.get("config") or {}
    print("[superopt] starting: workdir=%s max_iterations=%s adjust_interval=%s "
          "max_wall_time_h=%s preverify_red=%s"
          % (args.workdir, effective.get("max_iterations"),
             effective.get("adjust_interval"), effective.get("max_wall_time_h"),
             cfg["preverify_red"]), flush=True)

    # 延迟 import（并行纪律 + --help 冒烟）：loop.py 由 task8 落地
    try:
        from superopt.loop import Scheduler
    except ImportError as e:
        print("[superopt] FATAL: 调度器模块不可用（super/superopt/loop.py 缺失"
              "或损坏）: %s" % e, file=sys.stderr, flush=True)
        return 4

    scheduler = Scheduler(args.workdir, cfg)
    rc = scheduler.run()
    # run() 的退出码语义归 loop（preverify-red 失败 = 3）；无返回值视为正常
    if isinstance(rc, int) and not isinstance(rc, bool):
        return rc
    return 0


if __name__ == "__main__":
    sys.exit(main())
