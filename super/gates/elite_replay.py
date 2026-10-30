#!/usr/bin/env python3
"""elite_replay.py — 双回归 harness ①（job_47/task14）

拒历史合法解 = 误杀。读 <ws>/archive.json 的全部精英，逐个用**当前**门
（CascadeGate.verify，真实三级 + 软门禁）重放；任一精英被拒 → killed 非空
→ exit 1。修门提案（adjudicator 的 hard_modify_gate）必须附本 harness 的
通过结果（防误杀证据协议）。

CLI：
    python3 super/gates/elite_replay.py --workdir <ws>

输出：stdout 单行 JSON
    {"harness": "elite_replay", "replayed": N, "killed": [], "passed": true}
退出码：0=零误杀 / 1=存在误杀（或环境错误，error 字段说明）。

stdlib-only，Python >=3.9。
"""
import argparse
import json
import os
import sys

_HERE = os.path.dirname(os.path.abspath(__file__))
_REPO = os.path.dirname(os.path.dirname(_HERE))  # 仓库根
_SUPER = os.path.join(_REPO, "super")
for _p in (_SUPER, _REPO):
    if _p not in sys.path:
        sys.path.insert(0, _p)

from superopt.archive import Archive          # noqa: E402
from superopt.config import load as load_cfg  # noqa: E402
from superopt.gates import CascadeGate        # noqa: E402


def _emit(obj):
    print(json.dumps(obj, ensure_ascii=False))
    sys.stdout.flush()


def main():
    parser = argparse.ArgumentParser(
        prog="elite_replay",
        description="双回归 ①：elite 回放——新门不得拒历史合法解（防误杀）")
    parser.add_argument("--workdir", required=True, help="super 工作区路径")
    args = parser.parse_args()

    ws = os.path.abspath(args.workdir)
    out = {"harness": "elite_replay", "replayed": 0, "killed": [], "passed": False}

    # 环境装配：archive（真相投影）+ env/gates 目录（super.json 是 Go↔py 共享契约）
    archive_path = os.path.join(ws, "archive.json")
    if not os.path.isfile(archive_path):
        out["error"] = "archive.json missing: %s" % archive_path
        _emit(out)
        return 1
    cfg = load_cfg(ws)
    env_dir = cfg.get("env_path") or os.path.join(ws, "env")
    gates_dir = os.path.join(ws, "gates")

    archive = Archive(archive_path,
                      perf_desc=str(cfg.get("config", {}).get("perf_desc") or "low"))
    archive.load()
    gate = CascadeGate(env_dir, gates_dir)

    killed = []
    replayed = 0
    cells = archive.cells
    if callable(cells):          # _RuntimeArchive 形态（方法）
        cells = cells()
    for cell_id, cell in sorted((cells or {}).items()):
        elite = (cell or {}).get("elite")
        if not elite:
            continue  # 出档格（invalidation 后 elite=None）不参与回放
        code_path = elite.get("code_path")
        if not code_path or not os.path.exists(code_path):
            killed.append({"cell_id": cell_id, "reason": "code_path missing: %r" % (code_path,)})
            replayed += 1
            continue
        # 回放语义：验证精英在「当前门」下仍合法（环境健康 + 正确性 + 可测分）。
        # 不传 elite_perf/history 基线——软门禁的 perf 带是「入格比较」语义，
 # 重放时与自史 perf 比对只剩测量噪声（微秒级 bench 噪声 >5% 带宽），
        # 非误杀信号；硬拒（stage1/2/3 测量失败）才是误杀。
        gr = gate.verify(code_path)
        replayed += 1
        if not gr.passed:
            killed.append({"cell_id": cell_id,
                           "failure_stage": gr.failure_stage,
                           "details": (gr.details or "")[-200:]})

    out["replayed"] = replayed
    out["killed"] = killed
    out["passed"] = (replayed > 0) and (not killed)
    _emit(out)
    return 0 if out["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
