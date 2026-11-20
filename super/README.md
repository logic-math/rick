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
                                              [super.db sqlite 五表]          [agent 池]
                                                          ▲                       │
                              [级联门 gates.py ◀──────────┴── 框架终验（agent 自检仅参考）]
                              [humans 表两类队列（干预/提案） + 外置裁决 adjudicator]
```

- **Go 层**（`internal/super`、`internal/cmd/super.go`）：监督与命令面，不碰进化逻辑
- **py 层**（`super/superopt/`）：四状态循环（adjust → UCB 采样 → builder → coding/debug）、
  级联门、档案、事件流、裁决——**stdlib-only，Python ≥3.9**
- **agent 层**（pi 池）：每迭代一进程直调 `pi --mode json`（env 热更新友好）

## 目录结构

```
super/
├── superopt/            # py 调度器包（db/events/archive/sampler/agents/gates/loop/...）
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

## super.db（sqlite 五表——job_56 全量 DB 化）

进化元信息的**唯一真相源**（py 调度器单写者 + WAL；Go 侧与 tools 只读 SQL）：

| 表 | 内容 |
|---|---|
| `cells` | 特征格（cell_id/name/method_desc 优化方法/best_experiment_id/weight；格状态经 `state_json` 兼容投影） |
| `experiments` | 实验（experiment_id=`<迭代号>-<YYYYMMDDHHMM>` 时间戳编号=工作目录名/record_path/pi session_id/score/status） |
| `humans` | 人类价值判断**两类队列**：`intervention` 干预队列（human 提交，adjust step1 注入）+ `proposal` 提案队列（runtime 产出待 human 裁决） |
| `events` | 系统级事件日志（seq/ts/type/source/experiment_id/payload JSON）——**查此表即感知系统状态变化** |
| `resources` | 槽位资源（`experiment_slot` 并行实验总上限 / `env_slot` 同环境并发上限——二级限额；调度器仅管资源分配，同环境分时复用是 env 设计责任） |

事件类型沿用 jsonl 时代全集 + v2 新增（`adjust_step_started/ended/degraded`、
`env_proposal_queued`、`experiment_failed_confirmed` 等）。py 调度器进程锁
（`env/scheduler.pid` 存活检测）保证一会话一 env 唯一调度器——重启必先回收旧进程。
旧 json/jsonl 工作区保留不迁移（检测到旧布局 → 明确报错指引新建 env）。

## adjust 7 步 subagent 协议（解空间自改进）

每 `adjust_interval` 迭代触发一次 7 步流水线（每步独立上下文 answer-isolation，
产物落盘传递；`adjust_mode: legacy` 可回退单 agent 旧模式）：

```
①adjust_fact 问题还原   实验记录+humans 未应用干预（human 价值判断）→ facts.md
②adjust_perspective 视角 3 相关+3 不相关关键词→检索→3 视角（理论解释+归纳/演绎证明）→ perspectives.md
③adjust_contradiction 矛盾 每视角 1 个并行 subagent → 活跃约束集→影子价格最大约束 → contradiction_{1..3}.md
④adjust_judge 判定      3 矛盾逻辑审查排序（全局性/根本性/决定性）→ judgment.md
⑤adjust_reversal 逆转   主要矛盾辩证逆转（阻碍→手段）→ reversal.md
⑥adjust_space 解空间    格动作（生 new_cell/灭 weight=0/升权/降权 + 2 次要矛盾多样性配权）
                         → 结构化 proposals JSON（过 pi hook schema 结构检）→ decisions schema 落盘
⑦adjust_env 实验自改进  bug 解法→直接追加 domain_knowledge.md；skill/验证器反思→humans 提案队列
```

## 验证金字塔（分层验证——最低成本排除无效解）

`verify`/`benchmark` skill 带**层次验证参数**：`L1-M0`（系统 L1 层 M0 模块单验）/
`L{n}`（层验证）/`E2E`（端到端）——先分钟级快速验证论证改动有效，再端到端：
- 实验链（builder 风险推演+可选 L1 预检 → coding 一次写对+自验 → debug 完整四节实验记录）
  在 **agent 侧**用 L1 级快检自排除无效解（分钟级成本）
- **框架侧**调度器跑 `env/gates/gate_check.py`（E2E：基于 code 部署→预热→端到端测试）：
  stdout JSON 携带**多维优化目标**具体值（tp99/qps/ttft/tpot 等指标字典+constraints 约束），
  **score<0=未通过拒绝，score≥0=通过且为进化分数**（多目标加权平均/融合公式）
- **debug 主动上报 `experiment_failed: true` → 调度器跳过 E2E 验证**（失败路径加速迭代）

`env/gates/` 目录对 builder/coding/debug **禁读**（superopt-json-guard hook 按
SUPEROPT_ROLE 拦截——防针对验证器优化作弊）；adjust 可读写（反思验证器）；
verify/benchmark skill 直接调用 gates/ 内实现。

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
