---
name: super-loop
trigger: "当启动或操作 rick super 模式（env 建设/升级、启停监控后台进化、干预队列）时触发"
scope: "super 会话（全局）"
---

# Loop: super 中控（env 建设 + 后台进化监督）

`rick super`（默认 `session` 子命令）打开一个**中控中心会话**：本会话负责 env 的建设与升级（gen_env skill）、后台进化引擎的启停与监控、以及 human 干预的投递。进化本身由后台三层架构执行（Go PySupervisor 监督 → py 调度器四 agent loop 调度 → pi agent 池），**中控会话不执行进化逻辑**。

- 本 loop 是普通的 rick loop：super 会话的提示词注入「可用的项目 Loops」目录，当任务属于 super 模式操作时**先完整读取本文件**再动手。
- 架构分工（三层）：中控会话（本 loop）= 人机接口；`rick super start` 起的 Go PySupervisor = 进程监督（心跳/kill 梯/游标重启）；py 调度器 = 唯一真相源写者（events.jsonl）与进化循环——循环由**四个 agent loop** 组成（见「四 Agent Loop」节）。
- 自改进分层（Step 7）：内层迭代进化「解」，中层 adjust 进化「搜索策略」（权重/加格/灭绝），外层（本 loop 的 Step 7 + gen_env skill 第 7 节）用轨迹证据进化「搜索者本身」（四角色 prompt 模板等 env 级资产）。
- **单写者纪律（不可协商，进程级）**：events.jsonl 只有 **py 调度器进程**能写（进程内并发槽的 worker 线程也属于该进程，写入见「事件模型」节的并行写规则）。human 的任何干预（暂停/降权/复活/停机）一律经 `rick super event` 写干预队列，由调度器在迭代边界摄取入事件——**中控会话与 human 永远不直写 events.jsonl / archive.json**。

## 四 Agent Loop（实验驱动的调度架构）

调度器每迭代按固定时序触发四类 agent，**各自有独立的 loop 与 skill 组合**——即「**四 agent loop**」（adjust 四步 / builder 调研+设计 / coding TDD / debug 复验）；四个 loop 共享同一份「**实验记录**」作为恒常上下文（见后节）。

```
每迭代：
  ① [每 adjust_interval 迭代] adjust —— 后台线程（与下一迭代并行，不阻塞）
       loop 四步：全局实验分析 → SENSE(定主要矛盾) → research(找启发式 idea) → 特征格管理
       产物：decisions/iter{N}.md（决策记录）+ proposals/
             + weight_changed / cell_created / cell_modified / proposal 事件
  ② 加权随机采样目标格（概率 ∝ weight；分殖优先覆盖未访问格）
  ③ builder —— 前台串行
       loop：读目标格描述/精英代码/精英实验记录 + 两个随机相邻格（描述/精英代码/实验记录）
             → research（关注点=实验细节/数据/方法）→ research_report.md
             → super_experiment_design → experiment_record.md（六段，含假设与验证逻辑）
  ④ coding —— 前台串行
       loop：读调研报告 + 实验记录 → 细化 research（落地细节/风险卡点）→ coding_plan.md
             → 严格 TDD（先写 test_solution.py → 看到红 → 实现 → 绿 → 重构 → 保绿）
             → 追加实验记录「执行过程与结论」段；过程中卡点按 super_debug_protocol 自查
  ⑤ 框架门禁 gate_check（正确性 → 性能 → 软门禁）——外部权威判定
       失败 / 收益异常(≥ gain_threshold_pct) / 软门禁容差带内 → ⑥
  ⑥ debug —— 独立派发（每候选至多 1 次，attempt 从 100 起编号，与自检失败 1..M 区分）
       loop：复现 → research（关键组件用户手册）→ 三个最可能失败原因逐一验证
             → 修复 或 keep_original（维持原解照常入格）→ 追加实验记录「卡点/根因/解决」段
  ⑦ 入格（MAP-Elites）/ 拒绝 → iteration_ended
```

**触发时机与 IO 表**：

| Agent | 何时触发 | 读取（ctx 注入） | 产出 |
|---|---|---|---|
| **adjust** | 每 `adjust_interval` 迭代边界（后台线程，与下一迭代并行） | 区间内**全部实验记录**（上次 adjust→本次）+ 区间内 agent session 路径（按需 read 补全细节）+ 决策历史（近 3 份）+ 各格描述/权重/得分/visit + perf 时间线 + 领域摘要 + 检索指引(adjust) | `decisions/iter{N}.md`、`proposals/`、`weight_changed`/`cell_created`/`cell_modified` 事件 |
| **builder** | 采样后（前台串行） | 目标格描述/精英/visit/weight + **精英实验记录** + **两个随机相邻格**（共享 ≥1 轴值；各自描述/精英代码/实验记录）+ 历史失败 + 轴级性能对照 + profile 时间序列 + 检索指引(builder) | `research_report.md`、`experiment_record.md`（建）、`experiment_plan.json`、`designs/iter{N}_{cell}.json` |
| **coding** | builder 完成（前台串行） | 实验计划 + **builder 调研报告** + **实验记录** + 三要素命令 + 门禁定义 + 格基线 perf + 检索指引(coding) | `coding_plan.md`、`test_solution.py`、`solution.py`、实验记录追加段 |
| **debug** | ① coding 自检失败（同 agent 内循环 1..M）② **框架门禁失败**（stage2/stage3/soft_gate）③ **收益 ≥ `debug_gain_threshold_pct`**（默认 20，防作弊/误测的异常收益进地图）④ **软门禁容差带内**（`debug_recheck_tolerance_band=true` 时，防无收益噪声解入格） | 失败阶段/metrics/stderr + 实验记录 + 门禁定义 + 检索指引(debug，关注点=用户手册/正确用法) + 复验原因 | `fix_report`（含**三个最可能原因**与逐一验证结论）、`keep_original` 判决、实验记录追加段 |

## skill 加载规范（四 agent 共用）

**加载方式（不可协商）**：agent 的系统提示词（`super/prompts/<role>.md`）**声明**「skill 名称 + 路径 + 触发条件」；agent 执行到对应步骤时用 `read` 读取全文——**不内联全文、不自动全量加载**。
理由：①省 token（每件 skill 百行级，四角色全内联会撑爆 prompt）②skill 可独立进化（env 级 / 仓级覆盖机制现成，见 Step 7）③符合 rick skill 规范（三段式 + 触发场景 → 由触发条件决定何时读）。

| skill | 服务 agent | 触发条件 | 文件 |
|---|---|---|---|
| `super_experiment_analysis` | adjust | 四步第一步（读区间实验记录 → 归纳主要矛盾候选） | `.rick/skills/super_experiment_analysis/skill.md` |
| `super_sense` | adjust | 四步第二步（定主要矛盾 + 辩证逆转；方法论 canon = rick `sense`） | `.rick/skills/super_sense/skill.md` |
| `super_cell_management` | adjust | 四步第四步之前（新建/改进/调权手续与正交性判据） | `.rick/skills/super_cell_management/skill.md` |
| `super_experiment_design` | builder | 调研报告完成后、写实验记录之前 | `.rick/skills/super_experiment_design/skill.md` |
| `super_debug_protocol` | debug（coding 卡点自查亦用） | 进入 debug 后、开始复现之前（方法论 canon = rick `debug-skill`） | `.rick/skills/super_debug_protocol/skill.md` |
| `test-driven-development` | coding | 动手写代码之前（仓内 canon = `internal/prompt/templates/skills/tdd-zh.md`） | 仓级 skill 或上述 canon |
| `sense` / `debug-skill` | adjust / debug | 方法论 canon（由 `super_*` skill **引用路径**，不复制全文） | rick skill canon |

- **skill 不可用时的降级**：声明的 skill 文件不存在 → agent 按该 skill 的核心纪律自行执行（prompt 里写明要点），不中断迭代；缺 skill 属资产缺口，记入 Step 7 复盘。

## 实验记录与决策记录（恒常上下文）

### 实验记录 `codes/iter{N}-{uuid}/experiment_record.md`

**定义**：一次「在某个特征格上的一次迭代」= 一个**实验**；实验记录是该实验的完整上下文文件，**跨 agent 传递**（四个 loop 共享记忆的载体）。

| 阶段 | 谁写 | 写什么 |
|---|---|---|
| 建（六段） | **builder** | 1.背景与价值 2.实验假设（可证伪） 3.验证逻辑（什么结果支持/否定） 4.具体实验步骤（可复现） 5.观测现象（预期指标变化） 6.执行过程与结论（占位） |
| 补 | **coding** | 追加「执行过程与结论」：实现要点、TDD 红绿记录、自检/门禁结果 |
| 补 | **debug** | 追加「卡点/根因/解决」：复现方式、三个候选根因与验证结论、修复或 keep_original |

- **读取方**：coding/debug 每次派发前在 ctx 收到全文；builder 收到目标格精英实验记录 + 两个相邻格的实验记录摘要；**adjust 在全局实验分析时读「上次 adjust 至今」区间内的全部实验记录**（并按需 read 对应 agent 的 pi session 补全细节）。
- **质量信号**：迭代结束时调度器校验实验记录是否被更新（mtime），未更新 → 事件标注 `record_updated: false`——这是「agent 未按 loop 执行」的客观信号，Step 7 复盘要看。

### 决策记录 `decisions/iter{N}.md`

- **谁写**：adjust 每轮决策后写（agent 自写；若 agent 未写，框架按 proposals+evidence 合成最小记录并标注 `framework_synthesized: true`）。
- **内容**：区间实验全景摘要 → **主要矛盾判定** → research 检索到并采用的 idea（论文/文档引用）→ 四类格动作（`new_cell` / `modify_cell` / `weight` / `terminate_recommend`）及依据 → 正交性论证（新建格时）。
- **谁读**：adjust 自己（`decision_history` 近 3 份，防震荡/防重复）；**中控会话 Step 7 轨迹复盘必读**（决策可解释性的主证据）。

### 特征格五要素

`{name, description(方法描述，可引用论文/调研文档地址), weight, score, code_path}`：

- **前四要素由 adjust 维护**：`new_cell` 必填 name/method/source_refs 与**正交性论证**（新维度未被现有格覆盖）；`modify_cell` 改进 method；`weight` 0 = 功能性删除（格与谱系保留，可复活）。
- **第五要素 `code_path` 由框架在入格时同步**：`elite_code_path` = 当前最佳精英代码，`elite_record_path` = 其实验记录（`codes/iter{N}/experiment_record.md`）——agent 读格即得代码与实验档案，不必扫目录。

## 事件模型（实验唯一标记 + 并行写）

**一次实验的唯一标记 = (cell_id, iteration_id) → `experiment_id`**（形如 `impl_a_mem_b#iter7`，非字母数字转 `_`）。

- **事件字段**：每行 `{ts, seq, experiment_id, experiment_seq, type, source, payload}`。实验内事件带 `experiment_id` + `experiment_seq`（**实验内**单调递增）；运维事件（session/heartbeat/intervention/adjust 类）`experiment_id` 为 `null`。
- **并发写安全**（迭代10/task49+task50）：并发槽的 worker 线程与主线程可**并行追加**同一 events.jsonl——O_APPEND 下单行 <4KB 写入原子 + EventWriter 写锁；`seq` 在并行下**仅尽力单调**，**定位与还原一律用 `(experiment_id, experiment_seq)`**，全局时间顺序看 `ts`。跨进程仍然是单写者（只有调度器进程写）。
- **还原单实验**（排障/复盘的标准手法）：
  ```bash
  grep '"experiment_id":"<cell>#iter<N>"' <ws>/events.jsonl | python3 -c 'import sys,json; [print(json.loads(l)["experiment_seq"], json.loads(l)["type"]) for l in sys.stdin]'
  # 或进程内：events.read_experiment("<cell>#iter<N>") -> 按 experiment_seq 排序的事件列表
  ```
- **崩溃恢复**：`lineage.project_state` 按实验分组——有 `cell_sampled` 而无 `iteration_ended` 的实验 → `interrupted_experiments`（重启时据此清格锁/记事件，不重复入档）。
- **`event_fsync`**（`super.json config`，默认 `true`）：关掉可降 I/O 开销，代价是断电时尾部事件可能丢；崩溃恢复依赖事件流时保持 `true`。

### 并发实验（一格一实验，异格并行）

- `max_parallel_experiments`（**默认 2**；1 = 纯串行）：>1 时启用**并发槽池**——主线程负责采样 + 格锁 + 迭代号分配 + `experiment_started{slot}` 事件；worker 线程只跑单次实验（builder→coding→debug→gate→admit），事件并行写（带 experiment_id）。
- **格级互斥**：`<ws>/locks/cell_<safe>.lock`（`O_CREAT|O_EXCL`，跨进程有效）+ 内存集合；同一格同时只有一实验——采样撞锁 → 重采样其它格（`cell_lock_resample`）→ 全占则 `cell_locked_skip`（不消耗迭代号）。
- **全局态操作串行化**：热重载/裁决消费/epoch 复评/adjust 只在「无在飞槽」时执行（避免与 worker 的档案变更交叠）。
- **收尾**：停机时等在飞槽 `slot_stop_grace_s`（默认 120s），超期记 `slot_stop_timeout` 后退出；`experiment_finished` 事件可核对每个实验是否收口。
- **观察口径**：按实验 grep 还原（见上），不要试图从全局 seq 推断实验边界。

## 依赖准备（硬约束，缺失则报错停止）

1. `rick super --help` 列出五子命令（session/start/stop/status/event）——super 命令组已构建进当前 rick。
2. pi runtime 可用：`super.json` 的 `pi_path`（或 `~/.rick/config.json` super 段）指向存在的 pi 二进制（如 `<agent_dir>/runtime/node_modules/.bin/pi`）。缺失 → 按 agent-runtime-bootstrap-loop 处理，不要在缺 pi 时启动进化。
3. env 就绪或准备建设：目标工作区 `~/.rick/super/<name>/env/` 存在且过验收自检（见 Step 1），否则先走 gen_env 现场生成——**没有合格 env 不启动进化**。
4. numpy 等 Python 依赖装在 `env/.venv/`（`python3 -m venv env/.venv && env/.venv/bin/pip install -r env/requirements.txt`）——门禁框架优先用 `env/.venv/bin/python` 跑 env 脚本，不污染系统 Python。

不满足任一条件 → 停在对应步骤修复，不绕过（缺 env 就跑 = 调度器在 stage1 环境健康门反复失败）。

## 目标（Goal）

一次 super 中控会话达成五件事（客观判据）：

| # | 达成条件 | 判据 |
|---|---|---|
| G1 | env 合格（建设或确认既有） | **三要素 + 检索五件齐备**（`skills/{profile,verify,benchmark,gate_check,research}.py`：四件三要素各自独立可跑 + gate_check 对参考实现融合 pass + `research.py` 四角色输出三元组非空） + 门先证红种子拒识 100% |
| G2 | 后台进化按预期运行 | `rick super status` → state=running、心跳新鲜、iteration_cursor 递增、北极星（入格数/各格最优 perf/token 成本）可读 |
| G3 | 干预正确投递且不破坏单写者 | 干预只经 `rick super event` → 队列文件出现在 `interventions/` → 后续事件流出现 intervention 事件（source=human） |
| G4 | 停止干净 | `rick super stop` 后 state=stopped、session_ended 事件落盘、无孤儿进程（supervisor 整树收割） |
| G5 | 自改进闭环走完（长跑后） | Step 7 轨迹复盘有产出：诊断结论 + prompt 迭代决策（改/不改/回滚）落 PROMPTS_LOG，或明确「无需迭代」的判断依据 |

## 上下文管理（Context Management）

- **保留**：env 的任务定义与门禁设计决策（env.json/feature_space.json 内容摘要）、当前迭代游标与北极星快照、human 干预历史、门先证红结论、**prompt 进化状态（PROMPTS_LOG 版本史与最近一次迭代的证据链接）**、**决策记录要点（decisions/ 最近几轮的矛盾判定与格动作）**、**实验记录的关键结论（哪些假设被证伪/证实）**。
- **压缩**：长跑期间的例行心跳/迭代事件（读 status 聚合摘要即可，不逐条复述事件流）；agent 产出的代码细节（在 codes/{uuid}/ 有快照）。
- **遗忘**：已出档精英的历史版本代码内容（archive 里只有引用与谱系）；被拒绝提案的中间稿。
- 落盘位置：工作区 `~/.rick/super/<name>/`（super.json / events.jsonl / archive.json / interventions/ / codes/ / gates/ / prompts/）——速查表见 `super/README.md`。

## 可调用工具（Tool Access）

| 命令 | 用途 | 约束 |
|---|---|---|
| `rick super session` | 进入中控会话（super-loop + gen_env skill 默认加载） | 本 loop 的承载形态 |
| `rick super start --name <n> --env <path> [--daemon]` | 初始化工作区 + 起 PySupervisor + py 调度器（`--daemon` 后台） | 启动内含**门先证红**（exploit 拒识 100% + 金样例零误杀），失败即中止并给出泄漏样本清单 |
| `rick super status --name <n> [--json]` | 心跳/状态/迭代游标/预算/北极星/事件尾摘要 | 只读 |
| `rick super event --name <n> --type <t> --payload '<json>'` | human 干预投递（写 `interventions/` 队列文件） | **唯一干预入口**；禁止直写 events.jsonl/archive.json |
| `rick super stop --name <n>` | 优雅停（SIGTERM 落完当前事件 → grace → SIGKILL 整树） | 停后 state=stopped |
| gen_env skill（加载） | env 建设/升级/提示词进化的访谈式流程 | Step 1/Step 5/Step 7 使用 |

- 权限边界：中控会话不采样、不跑门禁、不改档案——进化决策全部在后台调度器；中控只读状态 + 投递意图。

## 会话工作流（Step 0-7）

**Step 0 环境自检**
- `rick super status --name <n>`：工作区存在性 / state / 心跳；首次没有工作区属正常（走 Step 1）。
- 确认依赖准备四项（命令组 / pi_path / env / .venv）。

**Step 1 env 建设（缺失时；已有 → 跳 Step 2）**
- **三要素追问（强制前置，先于任何生成动作）**：按 gen_env skill 第 0 节逐项追问 human——**目标**（要优化什么、到什么程度算赢、怎么测）、**约束**（什么绝对不能破坏、失败怎么判）、**现状**（当前基线/热点/环境事实）。三者必须由 **human 亲自描述**，AI 只做追问与复述确认，**不得代填**；判据是**每个要素都能转成一个 skill 脚本**（目标→`skills/benchmark.py`、约束→`skills/verify.py`、现状→`skills/profile.py`）。三者不清不进入生成。
- **检索三问（第四类资产，与三要素同级）**：按 gen_env skill 第 0.5 节追问 human——**搜索方法**（web_search 关键词 / 官方文档站内 / GitHub 代码搜索 / arXiv 站内与 API）、**搜索信源**（按角色分：coding/debug=使用手册与代码样例，builder/adjust=论文站）、**关注点**（**必须与特征格绑定**）。判据 = 能写成 `skills/research.py` 的 `ROLE_PROFILES`（四角色三元组非空）；human 亲自描述，AI 只追问确认。**定位**：检索是语义空间进化的「**认知注入**」通道——agent 在同一格以小角度反复微调（局部最优征兆）时，检索能引入外部新原理；关注点绑定到格才能收敛探索、避免泛读无关资料。
- 加载 **gen_env skill** 走访谈式流程：任务定义 → 评分方向 → **门禁系统设计引导**（把三要素落成四件 skill：`profile/verify/benchmark/gate_check`；exploit 注册表 / 金样例——门禁阈值属 env 设计范畴，由 human 决策，本 loop 不代答）→ 初始特征空间（2-4 轴，轴间正交）→ 初始解。
- matmul 类任务可从 `super/env_templates/matmul/` 模板起步，在模板上做门禁定制。
- 验收自检（gen_env 的收尾清单）：五件 skill 各自独立可跑 ✓（三要素四件 + `research.py` 四角色三元组非空）gate_check 对参考实现融合 pass ✓ exploit 样本被拒识 ✓（这是 Step 2 门先证红的地基）。
- **完成后立即推进 Step 2 启动——不得停在 env 生成后**（env_9 实测教训：gen_env 完成后未 start，工作区零轨迹 `events.jsonl`=0 行，等于白做）。若 human 明确要求先停，在会话记录里写明原因与计划启动时间。

**Step 2 启动**
- 启动前确认 env 建设含**检索三问**成果：`skills/research.py` 对四角色（coding/debug/builder/adjust）均能输出非空三元组（缺失 = 回 Step 1 补，别让 agent 无检索能力空跑）。
- `rick super start --name <n> --env <path> --daemon`：启动即跑**门先证红**（全部 exploit 样本拒识 100% + 全部金样例通过）——不绿即中止启动，把泄漏/误杀清单给 human 看，回 Step 1 修 env 门禁。
- 启动成功后用 `rick super status` 确认 state=running、心跳新鲜、iteration_cursor 开始递增。

**Step 3 监控**
- `rick super status --name <n>`（或 `--json` 给机器可读）：心跳/迭代游标/预算墙（--max-wall-time）/北极星（入格数、各格最优 perf、token 分 role 成本）+ 事件尾摘要（最近迭代的 gate_result / elite_admitted / env_failure_concluded）。
- 巡检关注三类信号：① env_failure_concluded 连发 → env 或 pi 环境问题（回依赖准备排查，别让调度器空转）；② 长时间零入格 → 看 gate_result 失败分布（正确性门过严？软门禁容差带？）与 adjust 的 weight_changed 轨迹；③ token 成本异常 → 干预降频（weight）或停机复盘。
- **token / cost 分布（成本可解释性的读法）**：① `status --json` 的北极星含总 token 成本；② 逐迭代归因看 `events.jsonl` 的 `iteration_ended.by_role`（各角色的 tokens/wall_ms/tool_calls）；③ 需要分布视图时跑 `python3 super/tools/cost_report.py --workdir <ws>`（按角色/迭代的成本表 + Top-N 昂贵迭代 + 上下文膨胀 + cache 命中率）——发现「单角色占比异常」或「截断（`truncated=true`）频发」时按 `super_cost_analysis` skill 的浪费模式清单定位。
- **轨迹证据直读**（比 status 聚合更细，看清 agent 到底在干什么）：`decisions/`（**adjust 每轮决策记录**：区间实验摘要 → 主要矛盾判定 → research 引用 → 格动作与依据，决策可解释性的主证据）、`codes/iter{N}-{uuid}/experiment_record.md`（**实验记录全文**：假设/验证逻辑/步骤/执行与结论——四 agent 共享的恒常上下文）、`codes/iter{N}-{uuid}/research_report.md`（builder 调研报告）与 `coding_plan.md`/`test_solution.py`（coding 的编码计划与 TDD 痕迹）、`designs/`（builder 每迭代决策落盘：引用的 arXiv 论文/灵感来源/推理/实验计划）、`profile_history/<cell_id>/`（每格 profile 时间序列——看现状如何变化）、`ledger.jsonl`（跨迭代实验台账）、`prompt_{role}.md` 快照（三要素/检索命令是否被真调用）。
- **单实验维度巡检（迭代10；并发槽下首选口径）**：一次实验 = `(cell_id, iteration_id)` → `experiment_id`。
  - 列出当前在飞/近期实验：`grep '"type":"experiment_started"' events.jsonl | tail -5`（带 `slot`/`experiment_id`）；收口核对 `experiment_finished`。
  - 还原某次实验的完整事件流（四 agent 时序一览）：
    ```bash
    grep '"experiment_id":"<cell>#iter<N>"' <ws>/events.jsonl
    # 或进程内/脚本：events.read_experiment("<cell>#iter<N>")（按 experiment_seq 排序）
    ```
  - 并发下**不要用全局 seq 推断实验边界**（seq 仅尽力单调）；跨实验时序看 `ts`。

**Step 4 干预**
- `rick super event --name <n> --type <t> --payload '<json>'`——类型如 pause / weight（降采样权，0=功能性灭绝不删格）/ resurrect（复活误杀格）/ env_change / stop。
- 纪律：干预写队列文件，调度器在**迭代边界**摄取（延迟 ≤1 迭代）；不承诺即时生效；重复投递按 seq 幂等去重。
- 干预后用 Step 3 验证：事件流出现 intervention 事件（source=human）+ 期望的后续效果（如 weight_changed / 恢复采样）。

**Step 5 env 升级（热更新）**
- 改 env 定义（门禁阈值/评分方向/特征轴/exploit 注册表）直接编辑 `env/` 与 `gates/` 下文件：调度器**每迭代 subprocess 直调 pi**（agent 上下文每迭代重建），env 变更在**下一个迭代自动生效**，无需重启。
- **改门禁（任何正确性门/容差/评分函数变更）必须重跑门先证红**：exploit 注册表全拒 + 金样例全过才继续跑——否则放水门禁会静默放行坏解入格。改门禁后建议顺带跑双回归（elite 回放防误杀 + exploit 回归防放水）。
- 特征轴变更影响格定义：新轴只对后续采样生效，既有格保留（不删格原则；旧格降权由 adjust 提案处理）。**run 内的运行时加格**走 adjust 的 expand_cell 提案（裁决链放行后 feature_space 热重载，下一迭代分殖优先即探索新格）；**跨 run 的轴级改版**走本步 + gen_env 升级路径。

**Step 6 停止与收尾**
- `rick super stop --name <n>`：优雅停（调度器落完当前事件再退）；确认 state=stopped、session_ended 事件、无孤儿进程。
- 长跑不手动停也会撞预算墙（--max-wall-time，默认 24h）或 token 预算墙（max_total_tokens）自然收尾——同样确认 state 与事件完整。
- 收尾报告 human：北极星终值（总迭代/入格数/各格最优 perf/总 token 成本）+ 后续建议（哪些格值得人工深挖、env 门禁哪里值得升级）。

**Step 7 轨迹复盘 → 自改进（《自改进的进化搜索算法》外层循环）**

跑完一轮后，进化不该止于解——**搜索算法本身用这轮的轨迹证据迭代**。本步是标准收尾动作（长跑后必走；短跑可跳过但要在 PROMPTS_LOG 或会话记录里注明「样本不足，不评估」）：

- **轨迹复盘**：加载 **gen_env skill 第 7 节（轨迹分析 → 提示词进化）** 走五步流程——
  ① 取证：跑 `python3 super/tools/adjust_trace.py --workdir <ws>` 得决策-效果对照表；
     并读工作区 `decisions/`（**adjust 决策记录全文**：每轮的主要矛盾判定/research 引用/格动作与
     正交性论证——评估「决策质量」的主证据）、`ledger.jsonl`（跨迭代实验台账：idea→实现→门禁→结论）、
     `designs/`（builder 决策档案：引用了哪些 arXiv 论文/灵感来源/推理/实验计划——评估「实验设计方案」质量的主证据）、
     `codes/iter{N}-{uuid}/experiment_record.md`（**实验记录全文**：假设/验证逻辑/执行与结论，看
     builder 的假设是否可证伪、coding 的 TDD 是否真跑、debug 的三假设是否落到根因）、
     `codes/iter{N}-{uuid}/research_report.md`（builder 调研是否落到可实现动作）、
     `profile_history/`（每格 profile 时间序列——现状演化与热点迁移轨迹）；
  ② 诊断坏模式（震荡/过激/迟钝/漏杀/探索不足，及 builder/coding/debug 各自的模式）；
  ③ 处方：改写 `env/prompts/<role>.md`（四角色均适用；一次只改一个角色）；
  ④ 版本管理：延迟具体化 + `_history/` 存旧版 + PROMPTS_LOG.md 记录；
  ⑤ 下轮验证回环（改善固化/恶化回滚）。
- **prompt 迭代决策**：把诊断结论 + 改/不改/回滚的决策呈给 human 确认（首轮改模板建议 human 在环；跑顺后可放开）。
- **重跑**：模板迭代确认后回 Step 2 启动下一轮——新模板经渲染链（env/prompts 覆盖优先）立即生效于下一轮全部 agent 调用。
- **饱和信号处置**：`saturation` 非空（连续 ≥3 迭代无新纪录 + 全格已充分采样 + 冠军停滞）时——
  adjust 应优先提 `expand_cell`（加格 = 探索唯一入口，经裁决链 + feature_space 热重载，新格下一迭代即被分殖采到）；
  若判断空间已榨干（冠军 perf 已触及 `domain_brief` 的天花板），应提 `terminate_recommend`
  （落 `proposals/`，`needs_human: true`）→ 你与 human 决策：**收手归档**或**换轴**（走 gen_env 轴级改版）。
  复盘时若发现「长时间无新纪录但从未加格/认输」→ 先查 saturation 注入链与模板文案（见 gen_env 第 7 节）。
- **gate→debug 通道（框架内置）**：**框架门禁失败**（级联门的 stage2 正确性 / stage3 性能 / soft_gate）会自动交给
  debug agent 修复 **1 次**（attempt 从 100 起编号，与自检失败的 1..M 区分）；debug 修完重跑门禁，
  仍失败则本迭代记失败（不循环）。复盘时若发现「门禁失败后 debug 修复率低」，属 debug 模板诊断 SOP
  待进化（走第 7 节流程），而不是让 human 手工改候选代码。
- **检索质量复盘（第四类资产专项）**：读 `designs/*.papers` 核对三件事——
  ① **数量与质量**（是否真检索、是否引用可落地机制，还是只抄摘要凑数）；
  ② **是否落到计划**（`applicability` 是否出现在 `plan_steps`/`reasoning`——「只检索不落地」是常见坏模式）；
  ③ **同质 idea 检测**（某格连续多轮 idea 都在同一小角度微调 = 语义空间局部最优征兆）→ 处方优先改 `skills/research.py` 的 `ROLE_PROFILES`
  （放宽关注点 / 换信源 / 换检索方法；属 env 级升级，需 human 确认），而不是只调权重。
- **成本复盘（每个 run 收尾必做——短跑可快速扫一眼并在会话记录注明）**：跑 `python3 super/tools/cost_report.py --workdir <ws>`，读三段：
  ① **按角色占比与中位**（找最大成本项：builder 计划质量 vs coding 实现 vs debug 复验——经验参考：debug 派发次数 / 截断次数是最灵敏的浪费信号）；
  ② **Top-N 昂贵迭代与截断次数**（`truncated=true` 多 = 回合预算不足或任务过大；对照 config 的 `*_max_turns`）；
  ③ **上下文膨胀曲线**（单 agent input 持续 >200k = 上下文在膨胀：相邻格全文注入/实验记录全文注入/重复 benchmark 交叉验证是常见根因）。
  与 `.rick/skills/super_cost_analysis/skill.md` 的浪费模式清单对照（重复 benchmark 验证 / 无效检索 / debug 烧在瞬时波动 / 超长上下文）→ 出优化处方：限长（相邻格只读摘要、调研报告限长）/ 限次（benchmark 复验上限）/ 分工（贵档只给关键格）/ 回合预算校准。
  **纪律**：本步只计量与解释（human 裁决：先保证可解释，再谈优化）；优化动作经 human 确认后落 `env/prompts/` 或 config，不在跑中热改。
- **两级接通**：本步进化的是 env 级资产（该 run 的搜索 DNA：四角色模板/特征空间/门禁）；
  若轨迹分析发现的坏模式是**系统性**的（跨 env 都会出现，如「adjust 普遍不引用事件证据」），
  升级为 rick 既有 **learning/dream 循环**（learning job 沉淀 → 更新 super-loop/gen_env skill 等仓级知识）——
  env 级资产在 env 里进化，仓级知识在 rick 里进化，两层互不越界。

## 产出评估（Output Evaluation）

- 评估类型：客观（命令可验证）。
- 检查表（对应 G1-G5）：
  - [ ] env 验收自检三项全过（gen_env 清单）
  - [ ] `rick super status --json`：state=running（或按预期 stopped）+ iteration_cursor 递增 + 北极星键齐全
  - [ ] 干预经 `rick super event` 投递且事件流可见 intervention（source=human）；全程无任何对 events.jsonl/archive.json 的直接写入
  - [ ] 停止后 state=stopped + session_ended 事件 + 进程树干净
  - [ ] 长跑后 Step 7 走完：诊断结论 + prompt 迭代决策落 PROMPTS_LOG（或注明不评估的理由）；
  - [ ] **实验记录链路完整**：抽样 2-3 个 `codes/iter{N}-{uuid}/experiment_record.md`，六段齐全且 coding/debug 的追加段存在（`record_updated: false` 事件为零或已解释）；`decisions/` 每轮 adjust 有决策记录（或标注 `framework_synthesized`）

## 停止标准（Termination Condition）

- **成功退出**：G1-G5 全达成，human 明示本次中控会话结束（短跑场景 G5 可以为「不评估」结论，但必须有明确记录）。
- **失败退出**：①门先证红修不动（exploit 持续泄漏或金样例持续被误杀）→ 停止并报告 env 门禁缺陷；②env_failure_concluded 连发且排查无解 → 停机留现场（events.jsonl + stderr_tail），交 human 裁决；③pi runtime 不可用 → 停止，走 agent-runtime-bootstrap-loop。
- **优雅退出**：任何时刻 human 要求停 → `rick super stop` + 状态确认后再退出会话；不留 running 孤儿进程，不删任何工作区数据（事件流是唯一真相源，留档可审计）。
