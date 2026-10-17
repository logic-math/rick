# super/ — rick super 模式（superopt 进化调度器）

> LLM agent 池在 env 定义的优化任务上跑「采样→实验→门禁→入格」的 MAP-Elites 进化循环。
> RFC：《superopt 重构为 rick-super 模式》；设计树：`.rick/jobs/job_47/doing/grilling/design-tree.md`。

## 三层架构

```
human ←→ [rick super 中控会话]（super-loop + gen_env skill 默认加载）
              │ rick super start/stop/status/event（CLI 子命令）
              ▼
[Go PySupervisor] ──Setpgid/文件心跳/kill梯/游标重启──▶ [py 调度器 superopt]（本目录，stdlib-only）
                                                          │ 唯一写者              │ subprocess
                                                          ▼                       ▼  pi --mode json
                                              [events.jsonl + archive.json]   [agent 池 ×4]
                                                          ▲                       │
                              [级联门 gates.py ◀──────────┴── 框架终验（agent 自检仅参考）]
                              [干预队列 interventions/ + 外置裁决 adjudicator]
```

- **Go 层**（`internal/super`、`internal/cmd/super.go`）：监督与命令面，不碰进化逻辑
- **py 层**（`super/superopt/`）：四状态循环（adjust → UCB 采样 → builder → coding/debug）、
  级联门、档案、事件流、裁决——**stdlib-only，Python ≥3.9**
- **agent 层**（pi 池）：每迭代一进程直调 `pi --mode json`（env 热更新友好）

## 目录结构

```
super/
├── superopt/            # py 调度器包（events/archive/sampler/agents/gates/loop/...）
├── prompts/             # 4-agent prompt 模板（adjust/builder/coding/debug）
├── env_templates/       # env 模板（matmul 为第一个）
│   └── matmul/          #   env.json/feature_space.json/code//data//skills//domain_knowledge.md
├── gates/               # 双回归 harness（elite_replay.py / exploit_regression.py）
├── tests/               # 单测 + mock_pi.py + e2e_test.py + run_all.sh
└── README.md
```

## 快速上手

```bash
# 1. 建工作区 + 生成 env（中控会话里 gen_env skill 引导；matmul 可直接用模板）
cp -R super/env_templates/matmul ~/.rick/super/demo/env
cd ~/.rick/super/demo/env && python3 -m venv .venv && .venv/bin/pip install -r requirements.txt

# 2. 启动（启动前自动门先证红：exploit 全拒 + 金样例全过才放行）
rick super start --name demo --env ~/.rick/super/demo/env --daemon

# 3. 监控与干预
rick super status --name demo          # 心跳/迭代游标/北极星（入格数/最优 perf/token 成本）
rick super event --name demo --type weight --payload '{"cell": "...", "weight": 0.5}'
rick super stop --name demo            # 优雅停（落完当前事件）
```

## env 目录协议（门禁定义属 env——L3-Q2）

```
<ws>/env/
├── env.json              # 任务名/perf_desc(low|high)/perf_type/容差(正确性 ε + 软门禁 band)
├── feature_space.json    # 初始特征轴（正交声明）
├── code/                 # reference.py（正确性锚点+性能基线）+ naive_start.py（进化起点）
├── data/                 # test_correctness.py + bench.py（框架终验脚本）
├── skills/               # self_check.py（agent 自检，仅供参考）
├── domain_knowledge.md   # 领域知识 + 性能分析/调优 SOP 两节
└── requirements.txt      # 依赖（装 .venv；gates 优先用 .venv/bin/python）
```

**gates 消费契约（stdout 单行 JSON）**：

| 脚本 | 输入 | 输出 |
|---|---|---|
| `data/test_correctness.py` | `--candidate <path>` | `{"passed": bool, "max_error": float}`（异常/超时→`{"passed": false, "error": ...}`） |
| `data/bench.py` | `--candidate <path>` | `{"runtime_ms": 中位数, "score": 加速比, "timings": [K次], "ref_runtime_ms": ...}`（超时→`{"passed": false, "error": "timeout"}`） |

级联门三级：stage1 环境健康（解释器/脚本存在/参考实现可跑）→ stage2 正确性 →
stage3 性能 + 软门禁（K=3 次计时中位数；容差带内通过但标记 `tolerance_band` 触发延迟入库）。
门禁**阈值全部来自 env.json**，框架零硬编码。

## events.jsonl 事件类型（单写者：py 调度器）

`session_started/ended`、`iteration_started/ended`、`cell_sampled`、`agent_started/ended`、
`candidate_produced`、`gate_result`、`elite_admitted/elite_replaced`、`weight_changed`、
`proposal`、`adjudicated`、`invalidation`、`resurrection`、`intervention`、`heartbeat`、
`env_failure_concluded`——每条 `{ts, seq, type, source, payload}`，append-only + fsync，
**全系统状态可从事件流重放恢复**（lineage 重建、崩溃游标恢复、北极星计算）。

## archive.json 概览

```
{meta: {env, feature_axes, perf_desc, updated_at},
 cells: {<cell_id>: {descriptor, elite: {code_path, perf_value, perf_type, score,
                lineage, confidence, admitted_at, gen},
          history: [...], weight, visit, sample_count, last_sampled_at}},
 pending: [{cell_id, code_path, reason, added_at}]}   # 延迟入库队列（重评估前不入正式格）
```

精英代码实体在 `codes/{uuid}/`（档案只存引用）；`weight=0` = 功能性灭绝（**不删格**）。

## 测试

```bash
# 单测（全 mock，秒级）
python3 -m unittest discover -s super/tests -p 'test_*.py' -v

# e2e 全链 + 双回归 + 污染演练（mock pi，分钟级）
bash super/tests/run_all.sh

# realpi 冒烟（真 pi 跑 ≥3 真迭代；不设 env 则 skip）
RICK_REAL_PI_ARGS='<pi 路径> <bootstrap 提示>' python3 super/tests/realpi_smoke.py
```

Go 侧：`go build ./... && go test ./internal/super/... ./internal/cmd/... ./internal/handler/...`

## 防作弊与 exploit_registry（gen_env 访谈引导）

设计门禁时引导用户想「这个域里 agent 会怎么作弊」，产出 `gates/exploit_registry.json`
（3-5 个样本）。matmul 域的典型模式：

1. **返回预存结果**：不依赖输入，返回固定/随机矩阵——正确性门拒（输入是固定 seed 的）
2. **篡改计时**：monkeypatch `time.perf_counter` / 后台线程预计算——门在独立进程跑，
   且 K 次中位数 + 参考实现同进程基线对照
3. **降低精度换速度**：float16/截断累加——max_error 超 ε 拒

门先证红：`rick super start` 时全部 exploit 必须被拒（100%）+ 金样例（参考实现）全部通过，
否则不放行进化。双回归（改门后必跑）：elite 回放（新门不得拒历史合法解）+ exploit 回归
（被拒样本仍须被拒）。
