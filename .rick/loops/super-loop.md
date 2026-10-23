---
name: super-loop
trigger: "当启动或操作 rick super 模式（env 建设/升级、启停监控后台进化、干预队列）时触发"
scope: "super 会话（全局）"
---

# Loop: super 中控（env 建设 + 后台进化监督）

`rick super`（默认 `session` 子命令）打开一个**中控中心会话**：本会话负责 env 的建设与升级（gen_env skill）、后台进化引擎的启停与监控、以及 human 干预的投递。进化本身由后台三层架构执行（Go PySupervisor 监督 → py 调度器四状态循环 → pi agent 池），**中控会话不执行进化逻辑**。

- 本 loop 是普通的 rick loop：super 会话的提示词注入「可用的项目 Loops」目录，当任务属于 super 模式操作时**先完整读取本文件**再动手。
- 架构分工（三层）：中控会话（本 loop）= 人机接口；`rick super start` 起的 Go PySupervisor = 进程监督（心跳/kill 梯/游标重启）；py 调度器 = 唯一真相源写者（events.jsonl）与进化循环（adjust 决策 → 加权采样 → builder 计划 → coding 执行 → debug 修复 → 级联门 → MAP-Elites 入格）。
- 自改进分层（Step 7）：内层迭代进化「解」，中层 adjust 进化「搜索策略」（权重/加格/灭绝），外层（本 loop 的 Step 7 + gen_env skill 第 7 节）用轨迹证据进化「搜索者本身」（四角色 prompt 模板等 env 级资产）。
- **单写者纪律（不可协商）**：events.jsonl 只有 py 调度器一个写者。human 的任何干预（暂停/降权/复活/停机）一律经 `rick super event` 写干预队列，由调度器在迭代边界摄取入事件——中控会话与 human **永远不直写 events.jsonl / archive.json**。

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
| G1 | env 合格（建设或确认既有） | **三要素齐备**（`skills/{profile,verify,benchmark,gate_check}.py` 四件各自独立可跑 + gate_check 对参考实现融合 pass） + 门先证红种子拒识 100% |
| G2 | 后台进化按预期运行 | `rick super status` → state=running、心跳新鲜、iteration_cursor 递增、北极星（入格数/各格最优 perf/token 成本）可读 |
| G3 | 干预正确投递且不破坏单写者 | 干预只经 `rick super event` → 队列文件出现在 `interventions/` → 后续事件流出现 intervention 事件（source=human） |
| G4 | 停止干净 | `rick super stop` 后 state=stopped、session_ended 事件落盘、无孤儿进程（supervisor 整树收割） |
| G5 | 自改进闭环走完（长跑后） | Step 7 轨迹复盘有产出：诊断结论 + prompt 迭代决策（改/不改/回滚）落 PROMPTS_LOG，或明确「无需迭代」的判断依据 |

## 上下文管理（Context Management）

- **保留**：env 的任务定义与门禁设计决策（env.json/feature_space.json 内容摘要）、当前迭代游标与北极星快照、human 干预历史、门先证红结论、**prompt 进化状态（PROMPTS_LOG 版本史与最近一次迭代的证据链接）**。
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
- 加载 **gen_env skill** 走访谈式流程：任务定义 → 评分方向 → **门禁系统设计引导**（把三要素落成四件 skill：`profile/verify/benchmark/gate_check`；exploit 注册表 / 金样例——门禁阈值属 env 设计范畴，由 human 决策，本 loop 不代答）→ 初始特征空间（2-4 轴，轴间正交）→ 初始解。
- matmul 类任务可从 `super/env_templates/matmul/` 模板起步，在模板上做门禁定制。
- 验收自检（gen_env 的收尾清单）：三要素四件 skill 各自独立可跑 ✓ gate_check 对参考实现融合 pass ✓ exploit 样本被拒识 ✓（这是 Step 2 门先证红的地基）。
- **完成后立即推进 Step 2 启动——不得停在 env 生成后**（env_9 实测教训：gen_env 完成后未 start，工作区零轨迹 `events.jsonl`=0 行，等于白做）。若 human 明确要求先停，在会话记录里写明原因与计划启动时间。

**Step 2 启动**
- `rick super start --name <n> --env <path> --daemon`：启动即跑**门先证红**（全部 exploit 样本拒识 100% + 全部金样例通过）——不绿即中止启动，把泄漏/误杀清单给 human 看，回 Step 1 修 env 门禁。
- 启动成功后用 `rick super status` 确认 state=running、心跳新鲜、iteration_cursor 开始递增。

**Step 3 监控**
- `rick super status --name <n>`（或 `--json` 给机器可读）：心跳/迭代游标/预算墙（--max-wall-time）/北极星（入格数、各格最优 perf、token 分 role 成本）+ 事件尾摘要（最近迭代的 gate_result / elite_admitted / env_failure_concluded）。
- 巡检关注三类信号：① env_failure_concluded 连发 → env 或 pi 环境问题（回依赖准备排查，别让调度器空转）；② 长时间零入格 → 看 gate_result 失败分布（正确性门过严？软门禁容差带？）与 adjust 的 weight_changed 轨迹；③ token 成本异常 → 干预降频（weight）或停机复盘。
- **轨迹证据直读**（比 status 聚合更细，看清 agent 到底在干什么）：`designs/`（builder 每迭代决策落盘：引用的 arXiv 论文/灵感来源/推理/实验计划）、`profile_history/<cell_id>/`（每格 profile 时间序列——看现状如何变化）、`ledger.jsonl`（跨迭代实验台账）、`prompt_{role}.md` 快照（三要素命令是否被真调用）。

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
     并读工作区 `ledger.jsonl`（跨迭代实验台账：idea→实现→门禁→结论）、`designs/`（builder 决策档案：
     引用了哪些 arXiv 论文/灵感来源/推理/实验计划——评估「实验设计方案」质量的主证据）、
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
  - [ ] 长跑后 Step 7 走完：诊断结论 + prompt 迭代决策落 PROMPTS_LOG（或注明不评估的理由）

## 停止标准（Termination Condition）

- **成功退出**：G1-G5 全达成，human 明示本次中控会话结束（短跑场景 G5 可以为「不评估」结论，但必须有明确记录）。
- **失败退出**：①门先证红修不动（exploit 持续泄漏或金样例持续被误杀）→ 停止并报告 env 门禁缺陷；②env_failure_concluded 连发且排查无解 → 停机留现场（events.jsonl + stderr_tail），交 human 裁决；③pi runtime 不可用 → 停止，走 agent-runtime-bootstrap-loop。
- **优雅退出**：任何时刻 human 要求停 → `rick super stop` + 状态确认后再退出会话；不留 running 孤儿进程，不删任何工作区数据（事件流是唯一真相源，留档可审计）。
