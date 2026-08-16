#!/usr/bin/env python3
# Description: rick-gates 门禁 helper（占位骨架）。

"""rick-gates 门禁 helper — 占位骨架（task3 落地，task8 填充真实逻辑）。

rick-gates 是 rick 自有门禁（下沉 pi hook/脚本）。本 helper 是确定性脚本入口，
负责执行 doing 门禁校验。task8 在此填充三类校验：

1. tasks.json 可解析 —— doing/tasks.json 是合法 JSON 且含 tasks 数组。
2. zombie 检测 —— 无遗留的运行中/僵尸任务。
3. commit_hash 存在 —— 成功任务有对应 commit hash。

本占位文件保持最小骨架：只打印使用说明并返回成功（不执行任何真实校验），
避免 task3 引入未完成的门禁逻辑。DeployRickCustomizations 会幂等复制本目录到
pi 托管 agent 目录（extensions/rick-gates/）。
"""

import sys


def main(argv):
    """占位入口：真实门禁校验由 task8 填充。"""
    print("rick-gates helper (placeholder) — gate logic filled in task8", file=sys.stderr)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
