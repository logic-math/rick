# 设计树 — super mod 四项系统性重构（job_56 · v2 根层重定义）

> v1：adjust 5-subagent 协议重构（基于 job_54 定稿）——已被 human 原创文档升级取代
> v2：human 于 grilling 追问轮提交原创重构定义（四项），本树按 v2 重写并融入九问裁决
> 原料：anchors.md（job_54 理论）+ human 原创文档（本 job 主指令）+ research-L1.md（工程先例）
> 本文档结构：两个设计层（根层 OKR / 模块层 pipeline）+ 执行记录（协议步骤）+ 追问记录

---

# 第 1 层：根层 OKR（O + KR 集 · v2）

## O（全局目标）

super mod 按原创设计完成四项系统性重构——①adjust 协议升级为 7 步 subagent 流水线（含 human 价值判断注入、三视角矛盾并行生成、实验自改进）；②进化元信息全量 sqlite DB 化（5 张核心表 + 槽位资源调度 + 调度器进程锁）；③实验执行 sub loop 确定性强化（builder 仿真预演/去 TDD/debug 协议/验证跳过加速，支撑并行实验）；④开放性 env 构建体系（领域专家知识/skills 注入/gates 目录隔离/多维目标融合分/验证金字塔）——全量测试与 e2e 验证下落地。

## KR 集（KR1 ∧ … ∧ KR7 ⟹ O，已融入九问裁决）

- **KR1【DB 化地基】**：sqlite 单库（`<ws>/super.db`）承载全部进化元信息，五张核心表：特征格表（名称/优化方法描述/当前最优实验ID/权重）、实验表（**迭代号-年月日时分时间戳**=ID=工作目录名/实验记录path/pi sessionID/得分）、humans 表（**队列类型字段**：干预队列=human 提交 / 提案队列=runtime 产出待裁决；干预编号/内容/是否已应用）、events 表（系统级事件日志）、资源表（资源类型=实验槽|环境槽/资源ID/描述/额度数）。py 单写者 + Go 侧 + tools 全部 SQL 查询；json/jsonl 存储废弃；旧工作区保留不迁移。**py 调度器进程锁**：env 下 PID 锁文件，启动时检查——有存活进程则启动失败（一会话一 env 唯一调度器，重启必先回收旧进程）。
- **KR2【槽位资源调度】**：实验槽（并行实验总上限）+ 环境槽（同一 env 允许并发实验数，二级限额）。调度器查询资源表空闲位：总实验槽满→等实验槽；该 env 环境槽满→等环境槽。**调度器仅管资源分配**（两实验可否同环境并发）；实验内部如何交错独占资源（分时复用）是 ENV 的设计责任（gen_env 规则）。
- **KR3【7 步 adjust 协议】**：adjust 每 interval 走 7 步 subagent 流水线（每步独立上下文、产物落盘传递）：①问题还原（实验记录→事实 + humans 表未应用干预作为 human 价值判断→事实与期望文档）②视角生成（3 相关+3 不相关关键词→research skill 搜索→3 个视角描述=理论解释+归纳/演绎证明→视角记录）③矛盾生成（**每视角 1 个并行 subagent**→活跃约束集→影子价格最大约束→矛盾记录）④主要矛盾判断（3 矛盾逻辑判定：全局性/根本性/决定性）⑤辩证逆转（主要矛盾→解决方法）⑥解空间自改进（格动作：新格/降权/升权/权重0/2 次要矛盾多样性配权；结构化 JSON 过 pi hook 结构检查；特征格元信息含详细优化方法）⑦实验自改进（bug 解法→domain_knowledge **直接写**；skill 修改/验证器反思→**提案队列**待 human 裁决）。**subagent 超参数可配**（env 配置每 subagent 的 model 后端+thinking 档位——思维跳跃度）。
- **KR4【sub loop 实验执行强化】**：主调度派 sub loop 驱动三角色：builder（skill+领域知识+实验ID+资源槽位+运行环境→search+读源码+历史 diff→实验计划→**风险推演段（必做）**+可选 L1 验证预检→证伪性论证：成功失败均有信息增量）；coding（**去 TDD**，信任一次写对）；debug（不合理处检查/门禁失败→review 反思/涨幅过大→查验证机制或实现作弊/失败→根因+完整实验记录（设计+过程+结论+反思，格式过 pi hook 完整性检查）；**主动上报失败→调度器跳过端到端验证加速迭代**）。sub loop 读实验记录 path 写回 DB。
- **KR5【开放 env 体系】**：env 四部分：①领域专家知识（domain_knowledge.md=所有 agent 准则与文档入口）②skills（profile/benchmark/verify/research 四基本+可扩展；注入领域知识强调务必调用）③**gates/ 目录**（gate_check.py=框架必然调用的验证器入口；builder/coding/debug 禁读（json-guard hook 拦截）；adjust 可读写；verify/benchmark skill 直接调用 gates/ 内 py 引用实现）④code 基线。**多维优化目标**：目标可多维（如 A>1 前提下 B/C 越大越好）；gate_check.py stdout JSON 携带约束具体值（tp99/qps/ttft/tpot 等指标字典）+融合分（多目标加权平均/融合公式）；score≥0=通过（进化分），<0=拒绝。env 可被 adjust 自改进+human 直接干预。这些规则全部写入 gen_env 的 env 生成规则。
- **KR6【super loop 四步构建流程】**：①设计树追问 skill 下钻→env 设计方案→生成（遵循 env 基本结构）②追问中必须从 human 获得 profile/benchmark/verify/research 四 skill 构建方法→组合出 gate_check.py（代码复用）③演练：逐 skill 真实执行+gate_check.py 真实运行，确保基础设施无问题 ④分层验证机制（验证金字塔）：分钟级快速验证→端到端验证；verify/benchmark skill 带层次参数（L1-M0=系统 L1 层 M0 模块单验/L{n}=层验证/E2E=端到端）；参数进领域专家知识引导 agent 小范围验证、最低成本排除无效解。
- **KR7【测试验收】**：全量单测（DB 层/协议层/执行链层/env 层）+ e2e（mock_pi 支持 7 步流水线与 sub loop）+ 现有测试迁移改造无遗漏；gate_check.py 负分协议、debug 跳过验证路径、进程锁、gates 禁读拦截均有测试覆盖。

## KR 充分性自检

四项改进（KR3=改进1 / KR1+KR2=改进2 / KR4=改进3 / KR5+KR6=改进4）+ KR7 工程验收 ⟹ O。反向检查：无缺项（human 原创文档四节+九问裁决全部入 KR）；改动面经 DAG 分层互斥（见第 2 层）。

---

# 第 2 层：四模块 pipeline 与 DAG 依赖

## 模块划分（改动面互斥）

| 模块 | 内容 | 主要改动面 |
|---|---|---|
| **M1 DB 地基** | sqlite 库+5 表 schema（含 humans 两类队列）+资源表槽位调度+py 调度器进程锁+py/Go/tools 全消费方迁移 | superopt/db.py（新）+events/archive/ledger/intervention 改写+internal/handler+internal/web+tools |
| **M2 env 体系** | env 四部分结构+gates/ 目录隔离规则+gate_check.py 多维目标融合分协议+gen_env skill 重写（四步构建+超参数配置+分时复用规则）+验证金字塔分层参数 | env_templates+gen_env_skill+gates.py 接口+super-loop.md |
| **M3 adjust 7 步协议** | 7 步流水线调度（step3 三视角并行）+新角色 prompt+humans 表注入+subagent 超参数派发+hook 结构检扩展 | loop.py adjust 段+prompts/adjust_*.md+superopt-json-guard 扩展 |
| **M4 sub loop 执行链** | builder/coding/debug prompt 重写（风险推演/去 TDD/完整实验记录）+调度强化（sub loop 写回 DB+失败跳过验证） | prompts/builder\|coding\|debug.md+loop.py 实验链段 |
| **M5 集成验收** | mock_pi 7 步场景+e2e+全量测试迁移 | super/tests |

## DAG 依赖

```
L1:  M1(DB)  ∥  M2(env 体系)      [改动面互斥，可并行]
L2:  M3(7步协议，依赖 M1 humans/events 表 + M2 env 结构)  ∥  M4(sub loop，依赖 M1 资源/实验表 + M2 分层验证)
L3:  M5(集成，依赖 M1-M4 全部)
```

## 本层消解声明

**本层全部消解，无遗留判断节点**——依据：①事实面（F1-F17，见执行记录）已全部自查消解（sqlite 零使用/槽池现状/json-guard 机制/InterventionQueue/env 结构/Go 消费面/pi CLI 超参数面/杀梯现状/gates metrics 三段）；②重量级调研已派 research（research-L1.md：JSON 步间传递/answer-isolation/失败降级/契约可靠性，5 项 A 级结论）；③判断节点（J2.1-J2.9）已全部经 human 裁决（见追问记录 v2 轮）；④叶子层四维度（文件路径+函数签名/文件结构/工具调用/环境配置）在 DAG task 拆解（tasks.json + task*.md）中逐 task 落实。

---

# 执行记录（协议步骤，非设计树层）

## 步骤①调研消解（代码事实，已自查闭环）

| # | 事实 | 来源 |
|---|---|---|
| F1 | sqlite 当前零使用；存储全为 json/jsonl（events.jsonl/archive.json/ledger.jsonl/interventions 目录/heartbeat.json） | super/ grep |
| F2 | 槽池已存在：`_loop_parallel` max_parallel_experiments 槽，每槽 `_slot_worker→_step_locked`（builder→coding→debug 串行链）；但只有「实验槽」概念，无「环境槽」资源类别、不查资源表 | loop.py:580-700 |
| F3 | superopt-json-guard：pi extension（agent_end hook 检查最终回复 JSON 块缺失→followUp 修复轮，MAX_JSON_REPAIR=2）——可扩展为 schema 字段校验+实验记录完整性校验+gates 禁读拦截 | extensions/superopt-json-guard/index.ts |
| F4 | InterventionQueue 已存在（interventions/ 目录队列 push/drain）——humans 表雏形；缺 applied 标记与两类队列语义 | intervention.py |
| F5 | env 结构现状：`<ws>/env/`=code/data/domain_knowledge.md/env.json/feature_space.json/requirements.txt/skills——domain_knowledge.md 已存在 | ~/.rick/super/matmul-verify2/env/ |
| F6 | coding agent 强制 TDD（五步循环+tdd JSON 字段+「作假比失败更糟」纪律）——KR4 要求去掉 | super/prompts/coding.md |
| F7 | debug agent 已有四触发（自检失败/门禁失败/可疑收益/容差带内）+一次性根因定位协议——「涨幅过大查作弊」已有雏形 | super/prompts/debug.md |
| F8 | gates.py=CascadeGate（级联 stage1/2/3+soft_gate+中位数）——gate_check.py 多维融合分为新接口层 | gates.py |
| F9 | Go 侧消费：internal/handler/super.go 读 events.jsonl（aggregateEvents）+archive.json+heartbeat.json；internal/web/sessions.go 同——DB 化需 Go sqlite 读（modernc.org/sqlite 纯 Go 无 cgo） | super.go:219-259, sessions.go:2362 |
| F10 | 实验目录现状：codes/iter{N}-{uuid}/；实验 ID 无时间戳概念——「迭代号-年月日时分」为新编号体系 | loop.py |
| F11 | gen_env skill=.rick/skills/gen_env_skill/skill.md（env 建设访谈流程）——KR6 要求重写为四步构建流程 | skill.md |
| F12 | events.py 单写者架构（py 调度器唯一写者）——DB 化后写者约束延续（单写事务） | events.py:3 |
| F13 | tools 消费面：adjust_trace.py/cost_report.py 读 events.jsonl——DB 化迁移点 | super/tools |
| F14 | rick-gates hook（helper.py）校验 tasks.json——job 执行层门禁与本 job 落地后 super mod 自身测试并存 | .rick/skills/rick-gates |

## 步骤①重量级调研（research-L1.md 已落盘）

JSON 步间传递（强模型下漂移最小，A 级）/answer-isolation（CoVe A 级）/失败降级（跳过+默认值+记账）/JSON 契约可靠性（强模型 99.97%）——支撑 7 步流水线的产物契约与失败降级设计。

## 步骤②提炼判断节点（已全部裁决，见追问记录 v2 轮）

九个判断节点（J2.1-J2.9）：DB 切换方式/实验编号/step3 并行形态/step7 权限分层/验证器保密/分数协议/环境槽语义/范围节奏/仿真语义。

## 步骤④事实回流（裁决打开的新事实，已自查消解）

| # | 事实 | 含义 |
|---|---|---|
| F15 | pi CLI 超参数面：`--model`（provider/id + `:thinking` 后缀）+ `--thinking` level（off~max）；无 temperature CLI 参数 | 裁决 Q3 的「温度系数」用 model+thinking 档位承载（subagent 超参数={model, thinking} 两维；发散=thinking 档+弱模型，收敛=强模型） |
| F16 | Go supervisor 已有 SIGTERM→grace→SIGKILL 杀梯（kill(-pgid) 全树）；py 调度器无进程锁（启动无 PID 检查） | 裁决 Q2 的进程锁=py 调度器启动时 env 下 PID 锁文件检查（新增）；回收链路复用杀梯 |
| F17 | GateResult.metrics 已有三段（correctness/perf/soft）；perf.value=K 次中位数、score 可选（env 定义缺省按 perf_desc 归一） | 裁决 Q6 的多维指标=metrics 扩展为指标字典+融合分（接口层可控扩展） |

## 步骤⑤终止判定（达标）

- 根层：KR1-KR7 全部融入裁决——OKR 充分性重验通过
- 模块层：M1-M5 划分+DAG 三层依赖确认；本层全部消解（见消解声明）
- 叶子层四维度：在 DAG task 拆解（tasks.json + task*.md）中逐 task 落实

---

# 追问记录

## v2 轮（2026-11-30 · J2.1-J2.9 九问全裁决）

| 问 | 裁决 | 结论 |
|---|---|---|
| Q1 DB 切换 | A | 一次切净；旧工作区保留不迁移 |
| Q2 实验编号 | 修正 | **无纪元**；迭代号-分钟级时间戳；**新增 py 调度器进程锁**（env 下 PID 锁，存活进程→启动失败；一会话一 env 一调度器） |
| Q3 step3 并行 | A+扩展 | 每视角并行 subagent；**gen_env 可配 subagent 超参数**（model 后端+thinking 档位→思维跳跃度） |
| Q4 step7 权限 | A+扩展 | domain_knowledge bug 解法直接写；skill/验证器→提案队列；**humans 表加队列类型字段**（干预/提案两类） |
| Q5 验证器保密 | A+细化 | env/gates/ 目录：builder/coding/debug 禁读（hook 拦截）；adjust 可读写；verify/benchmark skill 直接调用 gates/ 内 py；规则入 gen_env |
| Q6 分数协议 | A+扩展 | stdout JSON；**多维优化目标**（约束+多追求指标）；JSON 携带指标具体值（tp99/qps/ttft/tpot）；分数=加权平均/融合公式；score<0=拒 |
| Q7 环境槽 | A | 二级限额（实验槽总上限+env 环境槽并发上限）；调度器仅管资源分配 |
| Q8 范围 | A | 四项全做；**并行实验是本次核心目标**（多环境并行+同环境分时复用，复用机制是 ENV 设计责任） |
| Q9 仿真 | 推荐（A+B） | 风险推演段必做+可选 L1 验证预检 |

## v1 轮（2026-11-30 · J1.1-J1.8 八问）

human 未逐条裁决，提交原创文档重新定义目标（v2 化）——v1 追问作废；其中未被 v2 覆盖且仍有效的裁决（三臂对照不纳入/super-mode-upgrade Job A-E 不并入/JSON+markdown 混合产物/decisions 叠加格式/adjust_interval 默认不变）作为既定设计保留。
