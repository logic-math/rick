#!/usr/bin/env python3
# Description: rick-gates 门禁 helper — 确定性 doing 门禁校验。

"""rick-gates 门禁 helper — 确定性 doing 门禁校验。

rick 侧确定性脚本（非 pi extension hook，hook 仅作通知/记录）。runtime.Run 在解析到
agent_settled（pi 会话结束）后直接调用本脚本校验 doing 门禁，exit 非 0 = 门禁失败。

校验语义（与原 doing_check 三项语义对齐）：
1. tasks.json 可解析 —— doing/tasks.json 是合法 JSON 且含 tasks 数组。
2. zombie 检测 —— 无遗留 running 状态任务。
3. commit_hash 存在 —— status=success 的任务必须有非空 commit_hash。

用法：python3 helper.py <doing_dir>
"""

import json
import os
import sys


def main(argv):
    if len(argv) < 2:
        print("usage: helper.py <doing_dir>", file=sys.stderr)
        return 2

    doing_dir = argv[1]
    tasks_json_path = os.path.join(doing_dir, "tasks.json")

    # 1. tasks.json 可解析
    try:
        with open(tasks_json_path, "r", encoding="utf-8") as f:
            data = json.load(f)
    except Exception as e:  # noqa: BLE001 - 门禁错误串需兼容任意解析失败
        print("tasks.json 不可解析: %s" % e, file=sys.stderr)
        return 1

    tasks = data.get("tasks")
    if not isinstance(tasks, list):
        print("tasks.json 缺少 tasks 数组", file=sys.stderr)
        return 1

    for task in tasks:
        task_id = task.get("task_id", "?")
        status = task.get("status", "")

        # 2. zombie 检测
        if status == "running":
            print("zombie running: task %s 仍处于 running 状态" % task_id, file=sys.stderr)
            return 1

        # 3. commit_hash 存在
        if status == "success":
            commit_hash = task.get("commit_hash", "")
            if not commit_hash:
                print("missing commit_hash: task %s 状态为 success 但 commit_hash 为空" % task_id, file=sys.stderr)
                return 1

    print("gate passed")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
