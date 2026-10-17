# 设计树：rick super 模式实现（job_45 · grilling）

> 源需求：学习 INFER_RSI 工作区 + RFC《superopt 重构为 rick-super 模式》，将此 RFC 在 rick 中实现。
> RFC 路径：/workdir/sunquan20/INFER_RSI/.rick/draft/rfc/rfc-superopt-重构为rick-super模式.md（策略层文档，human 2026-10-09 确认；代码级实现方案 = 本 job 的深化对象）
> 本树为活文档：逐层追加（层号 + 模块 + pipeline + 判断节点）。

## 第 1 层（根层）：OKR —— O（目标）+ KR（关键结果）集

**O（目标，可验证表述）**：
在 rick 中实现 RFC 的 super 模式——`rick super` 以三层架构（rick Go 监督树 → py 进化调度器 → pi agent 池）运行一套**新写的 superopt**，具备 RFC 的结构性防御（events.jsonl 单写者真相源、级联评估门+软门禁、提案-裁决分离、软硬淘汰、门先证红），并以 gen_env skill 现场生成的**矩阵乘算子 env** 端到端跑通进化循环（选点→派发→评估→入档→事件可重放），全部系统状态可从事件流恢复。super 模式自带系统级 loop（super-loop，中控中心会话默认加载）与 gen_env skill，env 建设与升级在中控会话内完成。

**L3 第 1 轮裁决回流（2026-10-10，human 逐问回答）**：
- J1=范围 A（完整体系骨架；P1/P2 真实统计与参数定标留首跑）✅
- J2=归属 A（rick 仓内，v5 留 INFER_RSI 只读参照）✅
- J3=**变更**：不移植 demo_env，现场生成 env；新增交付物：① gen_env skill（系统级）② super-loop（系统级，super 会话默认加载，中控中心：env 建设引导+env 升级+后台 super 多 agent 运行的中控）；e2e = gen_env 生成矩阵乘算子 env → super 优化
- J4=派发 A（py subprocess 直调 pi --mode json；human 补充理由：适合 env 更新后实验迭代热更新）✅
- J5=web：本期只 CLI，但**对 web 留足扩展空间**；测试通过后另起 job 做 web ✅
- J6=**细化**（human 给出完整调度策略）：核心调度循环四状态——解空间生成（adjust agent：即 v5 strategy agent 进化版，负责格子增删/正交性保证，后续更多职责）→ 采样（UCB 采样格子→builder agent 选父格→idea→实验计划）→ coding（coding agent 执行代码+自检 env 门禁脚本）→ debug（门禁不符时 debug agent 处理）；**框架入格前独立验证正确性与性能，以框架为准**（测试/编码分离的实现形态）；参考 openevolve 看能否复用
- J7=PySupervisor 新写小模块 ✅
- J8=验收 A（mock e2e+realpi 冒烟≥3 真迭代+双回归+污染注入演练）✅

**KR（关键结果）集（推导：KR1∧…∧KR6 ⟹ O；J1 裁决后更新）**：

| # | KR | 对应 RFC | 充分性角色 |
|---|---|---|---|
| KR1 | 三层架构：`rick super` 会话（中控）+ `rick super start` 后台 Go PySupervisor（Setpgid/心跳/kill 梯/OnDead 游标重启）监督 py 调度器；py 调度器 subprocess 直调 `pi --mode json` 派发 agent（每迭代一进程，env 热更新友好） | §4.1 | 架构骨架 |
| KR2 | py 进化调度器四状态循环：adjust agent（解空间调整：格增删/正交性，v5 strategy agent 进化版）→ UCB 采样（visit 真实递增）→ builder agent（选父格→idea→实验计划）→ coding agent（执行+env 门禁自检）→ debug agent（门禁失败处理）→ 框架独立验证（正确性+性能，以框架为准）→ MAP-Elites 入格 | §5.1/§5.2/§5.3 | 进化算法主体 |
| KR3 | 真相源与档案：events.jsonl append-only 单写者（py 调度器唯一写者；人工干预经队列入事件）+ 精英格 JSON 档案（lineage/置信度标记）+ 崩溃安全（游标重放恢复）| §6 | 唯一真相源 |
| KR4 | c 方向核心机制：agent 自检（env 门禁脚本）与框架终验分离 + 级联门（环境健康→正确性→性能）+ 软门禁（浮动容差）+ 门先证红（exploit 注册表拒识）+ 双回归 harness（elite 回放+exploit 回归）+ 环境失败结论化 | §5.3/§5.3.1-§5.3.3 | 实验正确性 |
| KR5 | 淘汰体系：adjust agent 软动作（采样权/权重调控）自主 + 硬动作（删格/改格定义/修门）提案制走外置裁决；精英重评估+延迟入库+invalidation/复活语义 | §5.4/§4.4 | 防污染闭环 |
| KR6 | 系统级知识资产：super-loop（中控中心会话 loop：env 建设引导/升级/启停监控/干预 SOP）+ gen_env skill（env 生成器）+ matmul env 由 gen_env 现场生成 + e2e 全套验收（mock e2e+realpi ≥3 真迭代+双回归+污染注入演练）+ 北极星可从事件流计算 | §2.1/§9 | 可验证性+资产沉淀 |

**OKR 充分性自检**：KR1 架构 ✓ KR2 循环 ✓ KR3 真相源 ✓ KR4 门 ✓ KR5 淘汰 ✓ KR6 会话/资产/验收 ✓ —— MECE：架构/循环/真相源/门/淘汰/会话资产六面无重叠，覆盖 O 全部主张（三层架构✓核心机制✓真相源✓中控会话✓e2e✓）。

**判断节点（L1 调研后提炼，L3 第 1 轮已裁决 J1-J8；第 1 轮回答打开的新节点 R2-*，待第 2 轮裁决）**：

调研关键事实（详见 research-L1.md）：
- v5 三缺陷全部行号级实证（visit 断链 db.py:201 零调用/UCB 退化 feature_space.py:95-162/StrategyAgent 权重接管 :135-141 且 DB 77 格权重分布与 RFC 逐数吻合/人工修剪=events.bak 比 cur 新 10 事件）；可继承语义清单齐（六字段合约/级联四结局/codes{uuid} 快照/同格替换/加权灵感采样/env 目录规范）；lineage 恒 None、精英无 JSON 导出。
- rick 侧：cobra 注册模式（easy.go 56 行模板）；supervisor.go 六原语+21 测试齐全**但 Spawn 硬编码 pi RPC 协议**（py 不能直接用，OnDead 生产未接线）；py 派 pi 最薄路径=subprocess 直调 `pi --mode json`（realpi 冒烟先例已证）；py 心跳协议无先例需设计；web 有 DoingRunner 注入缝+--daemon re-exec 先例；仓内 py 先例=rick-gates helper.py（git-tracked+python3 直调）+mock_agent.py（MOCK_SCENARIO）；构建=scripts/build.sh、无 make test；loops_context/skills_context 由 LoadLoopsContext/LoadSkillsContext 注入会话提示词（easy_prompt.go:80/127），loop/skill 生成规范=gen-loop/gen-skill skill（Step 0-5 模板/三段式 skill.md）。
- openevolve 复用性（loop_13 既有简报 research-openevolve.md，代码0.4）：A/C/G 可免 fork 满足；B/D/F（失效干预/迭代回调/格级 UCB visit）须 fork——恰是新系统核心诉求；能力互补错位（openevolve 强在 island/级联/进程并行/checkpoint，缺 UCB visit/调权/失效干预/反思钩）；「只借鉴自研」=S 阶段倾向，第三方 fork 实证存在但同步税+bus factor。

| # | 判断节点 | 状态 |
|---|---|---|
| J1-J8 | 见上方 L3 第 1 轮裁决回流 | ✅ 已裁决 |
| R2-1 | super 会话/后台命令面 | ✅ 已裁决=A：`rick super`中控会话（照 easy 模式）；`rick super start [--daemon]`/`stop`/`status`/`event`；干预走队列保单写者 |
| R2-2 | adjust agent 动作权限 | ✅ 已裁决：**不删格**——采样权降 0 即可（功能性灭绝，与 RFC §4.3 同构）；软动作自主、硬动作提案制保留。openevolve 有无 adjust 角色→已核验（research-L1-r2）：**无此角色**，第 3 轮终裁=自研调度器+机制借鉴 |
| R2-3 | 采样机制 | ✅ 已裁决：有 UCB 数据时用 UCB（visit 修复）；冷启动时用特征格先验权重加权概率分布随机 |
| R2-4 | openevolve 复用 | ✅ 已终裁：自研调度器 + 机制借鉴（清单 13 项已呈现，1-5/13 本期落） |
| R2-5 | env 规范 | ✅ 已裁决=A：继承 v5 目录规范+feature_space.json（初始行为描述子）+门禁脚本协议明确化；gen_env skill 引导产出 |

**Human 补充指令（2026-10-10 第 2 轮）**：开工前须详细设计每个模块的数据结构与存储结构，下钻多层设计树，把问题都讨论清楚——设计树下钻须到数据结构/存储结构层。

**L1 调研简报**：research-L1.md（合并版；原文 research-L1a-v5资产盘点.md / research-L1b-rick集成面.md + leaf-1~4 保留）；openevolve 复用性双简报：loop_13 既有（/workdir/sunquan20/INFER_RSI/.rick/draft/loops/loop_13_briefs_stash/research-openevolve.md）+ 本轮专项（research-L1-r2-openevolve专项.md + leaf-1~4；修正 loop_13 的 reseed 误述：main@9196d87 无 reseed 实为迁移制）

**L3 提问记录**：第 1 轮（J1-J8）✅；第 2 轮（R2-1~R2-5）✅（R2-2 删格→降采样权 0；openevolve 开专项调研）；第 3 轮（openevolve 终裁+借鉴清单）✅；第 2 层（L2-Q1~Q7）✅；第 3 层（L3-Q1~Q6）✅（L3-Q2 架构修正：门禁设计归 env 范畴）。全部裁决回流完毕，下钻至叶子层。

---

## 第 2 层：模块 pipeline + 存储布局 + 系统级数据结构

> 本层为设计层：事实底座来自 L1 research-L1.md，无新增事实调研需求（本层全部消解，无未裁决判断节点——L2-Q1~Q7 已全部裁决）。
> 第 1 层根层已达标（O/KR/模块图/pipeline 全清晰，J1-J8+R2-1~R2-5 全部裁决，openevolve 终裁=自研调度器+机制借鉴 1-5/13 本期落）。下钻第 2 层：系统级数据结构与存储结构（human 要求开工前详设）。

### 2.0 模块调用关系（pipeline 总图）

```
human ←→ [M1 super 会话中控] (rick super 交互会话，加载 super-loop + gen_env skill)
              │ CLI 子命令 (start/stop/status/event)
              ▼
[M7 Go PySupervisor daemon] ──监督(Setpgid/心跳/kill梯/游标重启)──▶ [M2 py 进化调度器]
                                                                        │ 单写者
                                                                        ▼
                                                                   [M5 events.jsonl + archive.json]
[M2 py 调度器] ──subprocess pi --mode json──▶ [M3 pi agent 池: adjust/builder/coding/debug]
        │                                           │
        └────框架验证[M4 级联门: 环境健康→正确性→性能+软门禁]──▶ 入格[M5] / 淘汰[M6]
        │
        └────干预摄取[M6 干预队列] ◀── rick super event (human)
```

- M1 中控会话：env 建设引导（gen_env skill）/env 升级/启停监控/干预（调用 CLI 子命令）
- M7 Go 守护：只监督 py 调度器进程组生命周期，不碰进化逻辑；心跳=py 写 heartbeat.json，Go 读
- M2 py 调度器：四状态循环（§2.5）+ 单写者（events.jsonl 唯一写者）+ 成本账本
- M3 agent 池：每迭代 subprocess 直调 `pi --mode json`（热更新友好）
- M4 级联门：框架终验（agent 自检只作参考，入格以框架验证为准）
- M5 真相源：events.jsonl（append-only）+ archive.json（精英格档案）
- M6 淘汰体系：adjust 软动作自主 / 硬动作提案-外置裁决 / 重评估+延迟入库+invalidation

### 2.1 存储布局（super 工作区目录）

```
<SUPER_WS>/                      # rick super start 指定；默认 ~/.rick/super/<name>/
├── super.json                   # 运行元信息：env_path/state(idle|running|stopped)/iteration_cursor/pid/heartbeat_path/budget
├── env/                         # gen_env skill 产出（任务域，详见第 3 层 gen_env 设计）
│   ├── env.json                 # 任务名/perf_desc(高优低优)/评分函数说明
│   ├── feature_space.json       # 初始行为描述子声明（维度/取值/正交性说明）
│   ├── code/                    # 参考实现（每迭代整拷为 codes/{uuid}/ 起点）
│   ├── skills/                  # 可执行验证器（agent 自检脚本，.py 或 .sh）
│   ├── data/                    # bench.py + test_correctness.py（框架终验，JSON 输出协议）
│   └── domain_knowledge.md      # 领域知识 + 「## 性能分析 SOP」「## 性能调优 SOP」两节
├── events.jsonl                 # 唯一真相源（append-only，py 调度器单写者）
├── archive.json                 # 精英格档案（格→精英+lineage+置信度；单文件）
├── interventions/               # 干预队列（rick super event 写入；调度器摄取入 events）
├── codes/{uuid}/                # 每实验隔离快照（code 拷贝+agent 产出+自检日志）
├── gates/                       # exploit 注册表 + 金样例 + 双回归 harness
└── prompts/                     # 4-agent prompt 模板（默认模板，env 可覆盖）
```

### 2.2 events.jsonl 事件类型集（schema 草案）

每条：`{ts, seq, type, source, payload}`；source ∈ {scheduler, agent:adjust|builder|coding|debug, gate, human, adjudicator}

| type | 载荷要点 | 产生者 |
|---|---|---|
| session_started/ended | env/游标 | scheduler |
| iteration_started/ended | iteration_id/cell_id | scheduler |
| cell_sampled | cell_id/visit/ucb_score/冷启动标记 | scheduler |
| agent_started/ended | role/iteration_id/exit_code/token_ms/wall_ms(成本账本) | scheduler |
| candidate_produced | code_path/self_check 摘要 | coding/debug |
| gate_result | stage/failure_stage/passed/metrics(正确性/性能) | gate |
| elite_admitted/replaced | cell_id/code_path/perf_value/lineage/confidence | scheduler |
| weight_changed | cell_id/old_weight/new_weight/reason | adjust(软动作) |
| proposal/adjudicated | proposal_id/type(删格降权/改格定义/修门)/证据引用 | adjust/裁决 |
| invalidation/resurrection | cell_id/code_path/reason | 裁决 |
| intervention | human 输入原文+分类 | scheduler(摄取自队列) |
| heartbeat | ts | scheduler |
| env_failure_concluded | 诊断包(returncode/stderr_tail/死因分类) | scheduler |

### 2.3 archive.json 精英格档案结构（schema 草案）

```json
{
  "meta": {"env": "...", "feature_axes": [...], "updated_at": "..."},
  "cells": {
    "<cell_id>": {
      "descriptor": {"axis1": "v1", "axis2": "v2"},
      "elite": {"code_path": "codes/<uuid>/", "perf_value": 12.3, "perf_type": "runtime_ms",
                "score": 0.87, "lineage": ["<parent_code_path>", ...], "confidence": 0.8,
                "admitted_at": "<ts>", "gen": 3},
      "weight": 1.0, "visit": 5, "sample_count": 5, "last_sampled_at": "<ts>"
    }
  },
  "pending": [{"cell_id": "...", "code_path": "...", "reason": "new_cell|tolerance_band|low_weight_src"}]
}
```
- 精英解代码存 codes/{uuid}/（档案只存 code_path 引用，§L2-Q2）；lineage 从事件流重建（修复 v5 parent_hash 恒 None）
- pending = 延迟入库队列（重评估前不入正式格）

### 2.4 agent 合约（六字段基座 + 角色增量）

- 基座六字段（继承 v5 schema.py）：valid/score/perf_value/perf_type/perf_desc/summary；框架终验覆盖写 perf_*/score 为权威值
- 角色增量：
  - adjust → `{proposals: [{type, target_cell, action, evidence, rule_id}]}`（软动作=weight_changed 直接生效；硬动作=proposal 走外置裁决）
  - builder → `{experiment_plan: {idea, target_cell, parents: [code_path], plan_steps, expected_behavior}}`
  - coding → `{diff_path, self_check: {passed, stage, metrics}}`（自检用 env skills 脚本，结果仅供参考）
  - debug → `{fix_report: {attempts, root_cause, changes}, final_self_check}`（同 worktree 迭代修复）

### 2.5 四状态循环状态机（py 调度器主循环）

```
LOOP (每次迭代):
  每 N 迭代 → [adjust agent] 提案(格调整/采样权) → 硬动作走外置裁决 → 软动作直接应用
  [UCB 采样] target cell：visit 真实递增；冷启动(无 UCB 数据)=先验权重加权随机
  [builder agent] 读 target cell 元信息 + 选父格(跨格重组灵感) → idea → experiment_plan
  [coding agent] codes/{uuid}/ 隔离；按 plan 实现；自检 env skills 脚本
     ├─ 自检不符 → [debug agent] 同 worktree 修复(≤M 次) → 回 coding 或记失败
  [框架验证 M4] 级联门：环境健康→正确性→性能 + 软门禁容差
     ├─ 通过 → MAP-Elites 入格(同格替换条件) + 延迟入库判定(§L2-Q6) → events 追加
     └─ 失败 → 环境失败结论化(诊断包入档) / 失败分类分派
  淘汰：pending 队列重评估 → invalidation(出档) / resurrection(复活，误杀恢复)
```

### 2.6 级联门（M4）三级 + 软门禁

- stage1 环境健康：pi 可用性探活 + env skills 脚本存在性 + 依赖 import 检查 + 参考实现可跑（v5 环境失败 78% 的头部防线）
- stage2 正确性：data/test_correctness.py → {passed, max_error}
- stage3 性能：data/bench.py → {runtime_ms, score} + 软门禁（浮动容差=多计时/统计基线，判杀成本参考 MOFANG 1/2.8）
- 门先证红：gates/exploit_registry.json 冷启动种子 + 8 缺陷模式 checklist；双回归 harness = elite 回放（拒历史合法解=误杀）+ exploit 回归（放行已知坏解=放水）

### 2.7 崩溃安全

- Go PySupervisor：心跳超时→SIGTERM→5s grace→SIGKILL 整树收割→OnDead(游标)重启
- py 调度器：崩溃重启后从 events.jsonl 游标重放投影状态（不承诺确定性回放，RFC §6.3）

### 2.8 预算控制（默认值声明）

- `rick super start`：无迭代上限 + --max-wall-time 默认 24h + 手动 stop；--max-iterations 仅供测试/e2e 传参

### 第 2 层判断节点（L3 批量追问，已裁决 2026-10-10）

| # | 节点 | 裁决 |
|---|---|---|
| L2-Q1 | 工作区位置 | A：默认 `~/.rick/super/<name>/`，与 job 目录独立，多 env 多实例并存 |
| L2-Q2 | 精英解代码存储 | A：档案只存元信息+lineage，代码在 codes/{uuid}/ |
| L2-Q3 | agent 合约角色增量 | A：六字段基座 + 四角色增量 schema |
| L2-Q4 | 级联门三级 | A：保留三级（环境健康→正确性→性能+软门禁） |
| L2-Q5 | 门先证红种子 | A：手工 3-5 exploit 样本 + 8 缺陷模式 checklist |
| L2-Q6 | 延迟入库判定 | A：新格首解/容差带内/来源格已降权 三者任一 |
| L2-Q7 | 预算默认值 | A：无迭代上限+--max-wall-time 24h+手动 stop |

**第 2 层达标**：模块 pipeline + 调用关系 + 存储布局 + 系统级数据结构（events/archive/合约/级联门/崩溃/预算）全部澄清，OKR 充分性自检通过（每个 KR 已有系统级承载结构）。下钻第 3 层。


---

## 第 3 层：每模块内部设计（包结构 + 关键函数签名 + 协议）

> 本层唯一事实调研=token 计量（research-L3-pi-token.md，已消解）；其余为设计层（L3-Q1~Q6 已全部裁决，无遗留判断节点）。
> 目标：开工前把每个模块的数据结构、存储结构、关键接口签名定清楚（human 明确要求）。本层聚焦「结构」，叶子四维度（文件路径/函数签名/工具调用/环境配置）在本层基础上固化。

### 3.1 M2 py 进化调度器（super/ 顶层目录，stdlib-only 零依赖）

```
super/
├── superopt/                     # py 包
│   ├── __main__.py               # python3 -m superopt --workdir <ws> [--max-iterations N] [--adjust-interval N]
│   ├── config.py                 # super.json 读写 + CLI flags 合并
│   ├── events.py                 # EventWriter：append-only 单写者（seq 单调递增 + fsync）
│   ├── archive.py                # Archive：load/save + admit(同格替换) + pending 队列
│   ├── feature_space.py          # FeatureSpace：cell 定义/描述子/正交性校验
│   ├── sampler.py                # Sampler：UCB(visit 递增) + 冷启动先验加权随机
│   ├── agents.py                 # PiAgentClient：subprocess pi --mode json + NDJSON 解析
│   ├── prompts.py                # 预渲染 4-agent prompt（模板 + env 上下文 + cell 元信息）
│   ├── gates.py                  # CascadeGate：stage1/2/3 + 软门禁 + 门先证红
│   ├── loop.py                   # Scheduler：四状态循环 + 崩溃游标恢复
│   ├── adjudicator.py            # 外置裁决（独立进程，硬动作提案裁决）
│   ├── intervention.py           # 干预队列摄取
│   ├── lineage.py                # lineage 从事件流重建（修复 v5 parent_hash 恒 None）
│   └── budget.py                 # 成本账本（时间/token 双轨）
├── prompts/                      # 4-agent 模板：adjust.md / builder.md / coding.md / debug.md
├── env_templates/matmul/         # matmul env 模板（gen_env skill 引用）
├── tests/                        # mock pi agent 测试（照 fake-binary-script 范式）
└── README.md
```

关键签名（草，叶子层固化）：
```python
# events.py
class EventWriter:
    def append(self, type: str, payload: dict, source: str) -> int  # 返回 seq；单写者+fsync
# sampler.py
class Sampler:
    def sample_cell(self, space: FeatureSpace, cold: bool) -> str   # UCB / 先验加权随机
    def increment_visit(self, cell_id: str) -> None
# agents.py
class PiAgentClient:
    def run(self, role: str, prompt_path: str, workdir: str, timeout_s: int) -> AgentResult
# gates.py
class CascadeGate:
    def verify(self, candidate_path: str, env_dir: str) -> GateResult  # stage1→2→3 + 软门禁
# loop.py
class Scheduler:
    def run(self, ws: str, budget: Budget) -> None                    # 主循环 + 游标恢复
# adjudicator.py（独立进程入口）
def main(proposal_path: str, ws: str) -> Decision                     # 硬动作裁决
```

### 3.2 M1+M7 Go 命令面与 PySupervisor（internal/）

```
internal/cmd/super.go          # NewSuperCmd：session/start/stop/status/event 子命令（照 easy.go 56 行模板）
internal/handler/super.go      # handler.SuperCore 编排（cmd→handler 分工照 easy）
internal/super/                # 新 Go 包
├── supervisor.go              # PySupervisor：Spawn(Setpgid)/文件心跳监视/kill 梯/OnDead(游标)重启
├── supervisor_test.go         # fake py 脚本测试（照 supervisor_test.go 21 用例模式）
└── session.go                 # super 会话 prompt 构建（LoadLoopsContext+LoadSkillsContext+super-loop inline+gen_env skill inline）
```

PySupervisor 关键签名：
```go
type PySupervisorConfig struct {
    HeartbeatInterval, HeartbeatTimeout, TermGrace time.Duration
    PythonPath string   // 默认 python3
    OnDead func(ws string, lastSeq int64)          // 死回调：游标重启
}
func NewPySupervisor(cfg PySupervisorConfig) *PySupervisor
func (p *PySupervisor) Start(ws string) (*PyWorker, error)  // spawn python3 -m superopt --workdir ws
type PyWorker struct { /* IsDead/Reason/StderrTail/LastHeartbeat/Close(kill 梯) */ }
```
心跳协议：py 每 HeartbeatInterval 写 `heartbeat.json` {ts, seq, state}；Go 读判超时 → SIGTERM → 5s grace → SIGKILL 整树收割 → OnDead(ws, lastSeq) → 重启（py 从 events 游标重放）。

### 3.3 M4 评估门 + 双回归 harness（gates/ 目录）

```
gates/
├── exploit_registry.json        # 冷启动种子：3-5 个 exploit 样本 {id,name,pattern,sample_path}
├── defect_checklist.json        # 8 缺陷模式（编译/正确性/性能回退/超时/环境/无效输出/作弊/测量噪声）
├── golden_examples/             # 金样例（ref 实现 + 已知好解）
├── elite_replay.py              # 双回归 ①：elite 回放（新门不得拒历史合法解=防误杀）
└── exploit_regression.py        # 双回归 ②：exploit 回归（被拒 exploit 仍须被拒=防放水）
```
级联门实现：stage1 env 健康（pi 探活+skills 脚本存在+依赖 import+参考实现跑通）→ stage2 正确性（data/test_correctness.py → {passed,max_error}）→ stage3 性能（data/bench.py → {runtime_ms,score} + 软门禁）。门先证红：启动时跑 exploit_registry 全部样本，拒识率须 100% 才放行进化。

### 3.4 M3 agent 池派发细节

- py 预渲染 prompt（模板+env 上下文+cell 元信息/experiment_plan）→ 落盘 `codes/{uuid}/prompt_{role}.md`
- 派发：`pi --mode json --append-system-prompt <prompt_file> -p <bootstrap> --cwd codes/{uuid}`（设 PI_CODING_AGENT_DIR 继承托管配置）
- 解析 NDJSON → agent_settled 终止 + tool_calls 收集 + final message → agent_result.json（六字段+角色增量）
- 成本账本：wall-clock（精确）+ tool_call 数 + token 字段占位（rick 的 piEvent 结构不含 token，pi 上游事件 schema 待核实，R7）

### 3.5 M6 淘汰体系细节

- 软动作：adjust agent 的 weight_changed 直接应用（archive.json cells[cell_id].weight，可降 0=功能性灭绝不删格）
- 硬动作：adjust 的 proposal（删格降权/改格定义/修门）→ adjudicator.py 独立进程裁决（读 events+archive+金样例）→ 双回归通过才生效
- 重评估：pending 队列 → 每 epoch 末重评估（M 次复测）→ invalidation（出档）/ resurrection（恢复采样权）

### 3.6 M8 gen_env skill + super-loop 内容设计

**gen_env skill**（.rick/skills/gen_env_skill/skill.md，三段式）：
- 触发：super 会话中 env 缺失或需升级
- 流程：访谈（任务定义/perf_desc 评分方向/参考实现/初始特征轴）→ 产出 env/{env.json, feature_space.json, code/, skills/, data/, domain_knowledge.md}
- matmul env 模板（env_templates/matmul/）：torch.matmul 参考实现 + naive 起点实现 + test_correctness.py + bench.py + 2 轴初始特征空间

**super-loop**（.rick/loops/super-loop.md，Step 0-N 格式）：
- frontmatter：name=super-loop；trigger=「当启动/操作 rick super 模式（env 建设/升级、启停监控、干预后台进化）时触发」；scope=super 会话（全局）
- 内容：env 建设引导（gen_env）/ 启停（start/stop/status）/ 监控（status/events tail）/ 干预（event 队列）/ env 升级（热更新）SOP

### 第 3 层判断节点（L3 批量追问，已裁决 2026-10-10）

| # | 节点 | 裁决 |
|---|---|---|
| L3-Q1 | 4-agent 触发节奏 | A：adjust interval N=5（可配）+ debug max_attempts M=3 |
| L3-Q2 | 软门禁机制 | **架构修正（human）**：门禁/容差/评分函数/阈值**属于 env 设计范畴**，非 rick 系统本身；rick 只提供门禁框架（级联门结构/门先证红/双回归/多计时中位数机制），env 通过 env.json+data/+skills/ 定义具体门禁，gen_env skill 引导用户设计门禁系统 |
| L3-Q3 | 外置裁决进程形态 | A：独立进程 `python3 -m superopt.adjudicator` |
| L3-Q4 | 成本账本 token | A：双轨占位 + **补充要求（human）**：调研 pi，把 token usage 补进来，按 agent role 划分 token |
| L3-Q5 | matmul env 定位 | A：CPU numpy matmul（512×512 float32），初始特征轴 2 维 |
| L3-Q6 | gen_env 产出初始解 | A：env/code/ 含 torch 参考实现+naive 起点，进化从 naive 起步 |

**架构修正落点（L3-Q2 带来的职责重划）**：
- rick super 系统职责 = 门禁**框架**（级联门三段结构 + 门先证红机制 + 双回归 harness + 软门禁机制=多计时 K=3 中位数+与 env 声明的容差比较）；8 缺陷模式 checklist = rick 提供通用框架，env 可补充
- env 职责 = 门禁**定义**（env.json：perf_desc/评分方向/容差 ε/正确性阈值；data/：bench.py+test_correctness.py 实现；skills/：自检脚本）；exploit_registry 具体样本 = env 定义（gen_env 引导）
- gen_env skill 访谈新增门禁系统引导环节（任务定义→评分方向→门禁设计→参考实现→特征轴）

**第 3 层达标**：每模块内部结构+关键签名+协议澄清完毕（L3-Q2 修正后：gates.py=通用框架读 env 门禁定义）。下钻第 4 层（叶子层四维度）。


---

## 第 4 层（叶子层）：四维度固化（文件路径 + 函数签名 + 工具调用 + 环境配置）

> 本层无新增事实调研（事实底座=L1 research-L1.md + L3 research-L3-pi-token.md），全部为设计层——本层全部消解，无判断节点（残留小设计点已在正文标注）。

### 4.0 全量文件清单

**Go 侧（新建/修改）**：
- 新建 `internal/cmd/super.go`、`internal/handler/super.go`、`internal/super/supervisor.go`、`internal/super/supervisor_test.go`、`internal/super/session.go`
- 修改 `internal/cmd/root.go`（AddCommand 一行）、`internal/config/config.go`（SuperConfig 嵌套结构体）

**py 侧（新建 `super/` 顶层目录）**：
- `super/superopt/__init__.py`、`__main__.py`、`config.py`、`events.py`、`archive.py`、`feature_space.py`、`sampler.py`、`agents.py`、`prompts.py`、`gates.py`、`loop.py`、`adjudicator.py`、`intervention.py`、`lineage.py`、`budget.py`
- `super/prompts/{adjust,builder,coding,debug}.md`
- `super/env_templates/matmul/`（env.json/feature_space.json/code/skills/data/domain_knowledge.md）
- `super/tests/`（mock_pi.py + e2e_test.py + test_*.py）
- `super/README.md`

**rick 知识资产（新建）**：
- `.rick/loops/super-loop.md`（系统级 loop）
- `.rick/skills/gen_env_skill/skill.md`（系统级 skill，目录形式）

### 4.1 Go 侧（cmd + handler + super 包）

```go
// internal/cmd/super.go
func NewSuperCmd(version string) *cobra.Command
//   子命令：session（默认）/start/stop/status/event
//   flags: --name, --env, --daemon, --type, --payload, --json

// internal/handler/super.go（cmd→handler 分工照 easy）
func SuperSession(opts Options) error                       // 中控会话（easy.go RunInteractive 同型）
func SuperStart(opts Options, name, env string, daemon bool) error
func SuperStop(opts Options, name string) error
func SuperStatus(opts Options, name string, jsonOut bool) error
func SuperEvent(opts Options, name, etype, payload string) error  // 写干预队列

// internal/super/supervisor.go
type PySupervisorConfig struct {
    HeartbeatInterval, HeartbeatTimeout, TermGrace time.Duration
    PythonPath string                                    // 默认 "python3"
    OnDead     func(ws string, lastSeq int64)            // 死回调：游标重启
}
func NewPySupervisor(cfg PySupervisorConfig) *PySupervisor
func (p *PySupervisor) Start(ws string) (*PyWorker, error)   // spawn: python3 -m superopt --workdir ws
func (p *PySupervisor) Get(ws string) (*PyWorker, bool)
func (p *PySupervisor) Count() int
func (p *PySupervisor) StopAll()
type PyWorker struct{ /* SessionID/IsDead/Reason/StderrTail/LastHeartbeat/Close */ }
func (w *PyWorker) Close() error   // SIGTERM → TermGrace → SIGKILL(-pgid 整树收割)

// internal/super/session.go
func BuildSuperPrompt(rickDir, ws string) (methodFile, promptFile string, err error)
//   复用 LoadLoopsContext/LoadSkillsContext + inline super-loop + inline gen_env skill
```

工具调用：`go build -o bin/rick cmd/rick`；`go test ./internal/super/... ./internal/cmd/... ./internal/handler/...`

### 4.2 py 侧（superopt 包，stdlib-only 零依赖）

```python
# events.py — 单写者真相源
class EventWriter:
    def __init__(self, path: str) -> None
    def append(self, type: str, payload: dict, source: str) -> int   # 返回 seq（fsync）
    def read_all(self) -> list[dict]; def last_seq(self) -> int      # 游标重放

# archive.py — 精英格档案（单文件）
class Archive:
    def load(self) -> dict; def save(self) -> None
    def admit(self, cell_id: str, entry: dict) -> bool               # 同格替换条件
    def get_elite(self, cell_id: str) -> dict | None
    def add_pending(self, cell_id, code_path, reason) -> None; def drain_pending(self) -> list

# feature_space.py
class FeatureSpace:
    def __init__(self, axes: list[dict]) -> None                     # 读 feature_space.json
    def cell_id(self, descriptor: dict) -> str
    def validate_orthogonal(self) -> None                            # 正交性校验

# sampler.py — UCB(visit 递增) + 冷启动先验加权随机
class Sampler:
    def sample_cell(self, cold: bool) -> str
    def increment_visit(self, cell_id: str) -> None

# agents.py — pi 池（含 token 计量）
class PiAgentClient:
    def __init__(self, pi_path: str, agent_dir: str) -> None
    def run(self, role: str, prompt_path: str, workdir: str, timeout_s: int) -> AgentResult
# AgentResult：六字段 + 角色增量 + usage{input_tokens,output_tokens,cache_read,cache_write,total,cost_usd}
# token 计量（research-L3-pi-token）：聚合 message_end(message.role==assistant).message.usage；
# 兜底取最后一次非零 message_update.usage；仍零 → 降级 wall-clock + tool_call 数

# prompts.py
def render(role: str, template_dir: str, ctx: dict) -> str           # 模板+env 上下文+cell 元信息

# gates.py — 级联门框架（读 env 门禁定义，framework 零假设）
class CascadeGate:
    def __init__(self, env_dir: str, gates_dir: str) -> None
    def preverify_red(self) -> bool                                   # 门先证红：exploit 拒识 100%
    def verify(self, candidate_dir: str) -> GateResult                # stage1 环境健康→stage2 正确性→stage3 性能+软门禁

# loop.py — 四状态循环 + 崩溃游标恢复
class Scheduler:
    def __init__(self, ws: str, cfg: dict) -> None
    def run(self) -> None                                             # 主循环 + _recover()
    def _recover(self) -> int                                         # events 游标重放 → last_seq
    def _step(self) -> None                                           # 单迭代
    def _run_adjust(self) -> None                                     # 每 N 迭代（N=5 默认）
    def _run_builder(self, cell_id: str) -> dict                      # → experiment_plan
    def _run_coding(self, plan: dict) -> str                          # → code_path
    def _run_debug(self, code_path: str, max_attempts: int) -> str|None  # M=3 默认
    def _verify_and_admit(self, code_path: str, cell_id: str) -> None

# adjudicator.py — 外置裁决（独立进程）
def main() -> None                                                   # python3 -m superopt.adjudicator --proposal <f> --workdir <ws>
def adjudicate(proposal: dict, ws: str) -> dict                      # 读 events+archive+金样例 → decision

# intervention.py — 干预队列摄取
class InterventionQueue:
    def drain(self) -> list[dict]                                    # 摄取 → 转 events(source=human)

# lineage.py
def rebuild_lineage(events: list[dict], cell_id: str) -> list[str]   # 修复 v5 parent_hash 恒 None

# budget.py — 成本账本（时间/token 双轨）
class CostLedger:
    def record_agent(self, role, wall_ms, tool_calls, usage) -> None
    def summary(self) -> dict                                        # 分 role：wall_ms/tool_calls/tokens/cost_usd

# config.py / __main__.py
def load(ws: str) -> dict; def merge_cli(cfg, args) -> dict
def main() -> None   # argparse: --workdir --max-iterations --adjust-interval --max-wall-time
```

工具调用：`python3 -m superopt --workdir <ws> [--max-iterations N]`；`python3 -m superopt.adjudicator --proposal <f> --workdir <ws>`

### 4.3 资产（prompts + env_templates + loop + skill）

- `super/prompts/adjust.md`（读轨迹→提案：格调整/采样权；软硬动作分界）、`builder.md`（读 cell 元信息→选父格→experiment_plan）、`coding.md`（按 plan 实现+自检 env skills 脚本）、`debug.md`（读 failure_stage+诊断包→修复）
- `super/env_templates/matmul/`：env.json（perf_desc=True 低优/评分方向/容差占位）+ feature_space.json（2 轴：实现策略/内存布局）+ code/（naive 起点 + numpy.matmul 参考——注：L3-Q6 推荐的 torch 参考改 numpy，CPU e2e 免 torch 重依赖，torch 留 CUDA 域）+ skills/（自检脚本）+ data/（test_correctness.py + bench.py，JSON 输出协议）+ domain_knowledge.md（SOP 两节）
- `.rick/loops/super-loop.md`：frontmatter（name=super-loop；trigger=启动/操作 rick super 模式；scope=super 会话全局）+ Step 0-N（env 建设引导 gen_env / start·stop·status / 监控 events tail / event 干预 / env 升级热更新 SOP）
- `.rick/skills/gen_env_skill/skill.md`：三段式（触发：env 缺失/升级；预期效果：产出可运行 env；核心内容：访谈流程=任务定义→评分方向→**门禁系统设计引导**（L3-Q2 修正）→参考实现→初始特征轴→初始解）

### 4.4 测试（自测资产 + e2e + 双回归 harness）

- `super/tests/mock_pi.py`：fake pi 脚本（MOCK_SCENARIO 环境变量驱动，照 rick tests/mock_agent 范式）
- `super/tests/e2e_test.py`：mock 全链（gen_env 产出→调度循环 N 迭代→入格→events 完整→门先证红→监督树 kill/重启恢复→事件流重放一致性→北极星计算）
- `super/tests/test_events.py`/`test_archive.py`/`test_sampler.py`：单元测试（unittest，stdlib）
- `super/gates/elite_replay.py` + `exploit_regression.py`：双回归 harness
- realpi 冒烟：`//go:build realpi` 门控 + `RICK_REAL_PI_ARGS` env（照 rick executor_realpi_smoke_test.go 先例）

### 4.5 环境依赖 + 配置

- Go：无新增依赖（stdlib + 既有 cobra）
- py 调度器：**stdlib-only 零安装**（json/subprocess/argparse/pathlib/math/time/dataclasses）
- matmul env：numpy（bench 用）；无 torch/GPU 依赖
- `~/.rick/config.json` 新增 `super` 段：`{default_super_dir, python_path, heartbeat_interval_ms, heartbeat_timeout_ms, term_grace_ms, adjust_interval(5), debug_max_attempts(3)}`（照 HumanLoopConfig omitempty 先例）

### 4.6 测试方法（自测指引，worker 过程性自测）

1. Go：`go build -o bin/rick cmd/rick && go test ./internal/super/... ./internal/cmd/... ./internal/handler/...`
2. py 单测：`python3 -m unittest discover -s super/tests -p 'test_*.py'`
3. e2e（mock）：`python3 super/tests/e2e_test.py`（mock pi 全链，秒级）
4. 门先证红：启动时 `CascadeGate.preverify_red()` 须 true（exploit 拒识 100%）
5. realpi 冒烟：`go test -tags realpi ./... -run RealPi`（RICK_REAL_PI_ARGS 门控，≥3 真迭代）

