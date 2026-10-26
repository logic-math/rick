# RSI 证据：dev-iterations

通过标准：至少记录一条构建指纹（`rick.dev.<sha7>-<ts>` 或 `build_id=<sha7>-<ts>`）。
示例：`DEV_UP bin=... build_id=c539c60-260921192330 port=8414`

<!-- TODO: 填写本文件后再跑 rick tools rsi_check（残留本标记即视为未填写） -->


## 迭代 1（2026-10-17）
- 环境：跨机搬迁（linux→macOS arm64）后 dev 树重建——`dev-web init` 重锚定（tree=/Users/logic/workdir/rick-dev home=/Users/logic/workdir/rick-dev-home）；web/node_modules 平台不符修复（rm -rf + npm i）；dev/self-evolve ff→main HEAD 506fa35
- DEV_UP build_id=506fa35-261017095018 port=8414 health ok
- plan 落盘：job_47（14 task / Kahn 4 层 / gate1-4.py），pipeline_gate 通过；commit 见上
- 关键裁决执行：human 复查计划后裁决 ①执行树走 B（dev 工作区）②INFER_RSI 参照=/Users/logic/workdir/INFER_RSI ③realpi 冒烟可跑
- job 编号说明：main 树 job_45=super RFC 会话锚（grilling 产物）；dev 树 job_45=human-loop v5 已发布（rsi 证据陈旧，防假通过）→ 实现取新号 job_47
- 下一步：L1 七 worker 并行派发（task1/2/3/4/10/12/13）

## 迭代10 成本对比（2026-10-26，task52）

**配置**：`max_parallel_experiments=2`（并发槽池）· `debug_max_turns=0`（不截断）· builder 20 / coding 24 回合
**样本**：env_9 iteration 16/17（真 pi / catpaw-proxy glm-5.3 / 8192 规模），两槽并发运行

| 指标 | 迭代9 基线（12 迭代） | 迭代10（2 迭代） | 变化 |
|---|---|---|---|
| tokens / 迭代 | 2.18M | **1.74M**（it16 2105k / it17 1370k） | **-20%** |
| cost / 迭代 | $1.47 | $1.41（合计 $2.82） | -4% |
| **debug 回合截断** | **12 / 20 次派发** | **0 / 3 次派发** | **截断消除** |
| debug 派发 / 迭代 | 1.67 | 1.50 | -10% |
| 单实验 wall | 25.6 min（串行） | 50-56 min（并发，两实验共享同一窗口） | 单实验变长、总吞吐提升 |

**结局实证**：
- it16：编码后门禁 `soft_gate` 判别失败 → debug 复验（200s/227k）→ **门禁通过**（失败被救回，未被丢弃）
- it17：门禁直接通过 → **入格 601.0ms**（精英）
- 两槽实验窗口重叠（slot0 it16 / slot1 it17 同时 running），事件按 `experiment_id`（格#迭代）可独立还原，`experiment_seq` 1..N 连续

**结论**：human 判断成立——**截断是 debug 浪费的主因**。基线 12 次截断（每次跑满 10 回合仍无最终结论，重启后重跑）对应约 3M+ 有效 token 浪费；本次 0 截断后 debug 单次完成根因定位与结论。token/迭代 -20% 与「不截断但一次做完」并行不矛盾：省下的是**重复重试**，不是单次深度。

**归因 caveat**：本次 2 迭代未触发 adjust（基线 2.18M/迭代里含 ~0.23M adjust 摊薄）；若补齐 adjust，本次对比会更保守地落在 -10~-15%。
