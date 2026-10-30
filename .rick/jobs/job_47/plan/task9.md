# task9：adjudicator.py + intervention.py + 入口

## 任务目标

实现外置裁决（硬动作提案的独立裁决进程——提案-裁决分离 KR5）与干预队列摄取（human 干预经队列入事件，保单写者），以及包入口（__main__）。

## 关键结果

- 新建 `super/superopt/adjudicator.py`：
  - `def adjudicate(proposal: dict, ws: str) -> dict`——读 proposal + 证据（events.jsonl 相关事件摘要 + archive 现状 + gates/exploit_registry 金样例比对）→ 裁决 {decision: approve|reject, reason, conditions}；裁决规则（内置，硬动作保守）：删格提案 → 检查该格 visit/weight/精英历史（weight 已 0 且连续 20 迭代未采样 → 可批；否则 reject「先降权观察」）；改格定义/修门提案 → **强制条件**：改门必须附双回归结果（elite_replay+exploit_regression 通过——gate4 harness 由 task14 提供，本 task 定义调用协议：proposal.evidence.regression 字段缺失 → reject「缺双回归证据」）
  - `def main() -> None`——独立进程入口：argparse `--proposal <path> --workdir <ws>`；读裁决队列文件（interventions/../proposals/ 或 loop 投递的裁决目录 `<ws>/proposals/`——**取 `<ws>/proposals/`**）→ adjudicate → 写回 `<ws>/proposals/<id>.decision.json` + **返回码**：approve=0/reject=1；调度器（task8）在下一迭代 drain decisions 应用（adjudicated 事件）
  - 裁决进程独立于调度器（human 可手动跑或调度器 spawn——契约：文件驱动，无锁）
- 新建 `super/superopt/intervention.py`：
  - `class InterventionQueue`：
    - `__init__(self, ws: str)`；`drain(self) -> list`——扫描 `<ws>/interventions/*.json`（时间戳命名）→ 按文件名排序读出 → **转交后删除**（at-least-once：先读后删，崩溃重复无害——事件幂等由 seq 保证）；坏 JSON 跳过 + 记 bad_intervention
    - 摄取方（loop）把每条干预转 intervention 事件（source=human 原文+分类：pause/weight/resurrect/env_change/stop）
    - `push(self, etype: str, payload: dict) -> str`——测试与 Go 侧 SuperEvent 复用的写入口（写文件返回路径；Go 侧独立实现同协议）
- 新建 `super/superopt/__main__.py`：argparse——`--workdir <ws>` 必填；`--max-iterations/--adjust-interval/--max-wall-time/--preverify-red` 可选；config.load + merge_cli → Scheduler(ws, cfg).run()；入口含 **subprocess 兼容 cwd**：以 `-m superopt` 从仓库根跑（`python3 -m superopt --workdir ...`，cwd=仓库根或 PYTHONPATH 含仓库根）
- 新建 `super/tests/test_adjudicator.py` + `test_intervention.py`（unittest）：
  - adjudicate：删格（weight=0+20 迭代未采样）→ approve；活跃格删格 → reject；修门无 regression 证据 → reject；带证据 → 按证据
  - main 文件往返：proposal 文件 → decision 文件 + 返回码
  - InterventionQueue：push→drain 往返（排序正确）；坏 JSON 容错；drain 后文件删除
  - __main__：`python3 -m superopt --help` exit 0（子进程冒烟）

# 写域

- super/superopt/adjudicator.py（新建）
- super/superopt/intervention.py（新建）
- super/superopt/__main__.py（新建）
- super/tests/test_adjudicator.py、test_intervention.py（新建）

## 依赖关系

- task5/task6/task7（Kahn L3 同层；与 task8 并行——**契约对齐点**：proposals/ 与 interventions/ 目录协议 + decision 文件 schema，两 task spec 已互相写明；本 task 不 import loop.py）

## 实现要点

- adjudicator 独立进程的裁决**建议性输出**：decision 文件由调度器消费应用（还是人不直接干预？——KR5：外置裁决=human 或脚本裁决，调度器只应用已裁决决策；adjudicate() 是**自动裁决规则**，human 也可以手改 decision 文件覆盖——文件驱动天然支持 human-in-loop）
- Python ≥3.9 兼容

## 测试方法

`python3 -m unittest super.tests.test_adjudicator super.tests.test_intervention -v` 全绿 + `cd <repo根> && python3 -m superopt --help` 冒烟

## 禁止事项

- 不得 import loop.py（并行层）；不得直写 events.jsonl（adjudicator 的裁决经 decision 文件，事件由调度器写——单写者不破）
