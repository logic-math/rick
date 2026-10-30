# Learning Step 1-2 分析：job_47（super RFC 实现 + 13 轮迭代）

> 生成于 learning loop Step 1-2 worker child。
> 数据源：git log --grep job_47（37 提交）、rsi/gates.md（29 gate 全绿）、rsi/dev-iterations.md（13 轮迭代 + 成本对比）、tasks.json（58/58 success，0 重派）、plan/task*.md、env_9/env_11 运行数据、wiki/handover-2026-10-30-web-start-macos.md（release 事故根因链）。

---

## Step 1a：执行轨迹还原（按迭代 1→13）

### 迭代 1：跨机重建 + plan 落盘（2026-10-17）
- 观察：linux→macOS arm64 跨机搬迁后 dev 树重建，dev-web init 重锚定（tree=rick-dev home=rick-dev-home），web/node_modules 平台不符 rm -rf + npm i。
- 关键决策：human 裁决执行树走 B（dev 工作区）、INFER_RSI 参照目录、realpi 冒烟可跑；job 编号取 job_47（main 树 job_45 = super RFC 会话锚 / dev 树 job_45 已被占用，防 rsi 假通过）。
- plan：14 task / Kahn 4 层 / gate1-4。

### 迭代 1-2：RFC 核心实现（task1-14，Kahn L1-L4）
- L1-L2：superopt 包骨架（events/archive/feature_space/sampler/config/budget/lineage）+ Go 侧 PySupervisor（fake-py 心跳/kill 梯/整树收割）。
- 关键教训（gate2 透明化记录）：TestReleaseCmd*（job_43 既有用例）在 macOS 跨机环境 fake-prod 构造死锁——干净 L1 树复现证实与 job_47 无关，gate 断言修正为 -skip 并记录遗留。
- gate 修正模式（首次出现，贯穿全程）：gate 自身的检查缺陷（字符串粗匹配误伤只读代码/冒烟缺 PYTHONPATH）当轮修正并透明化记录，断言语义不变。
- **worker 身份漂移教训**（gate3 记录）：t8 首派失败（worker 身份漂移零产出）→ fresh 重派（任务文本加角色护栏 + 要点内联）成功。
- REALPI_SMOKE 抓到两个真缺陷：①心跳线程缺失 → 重启风暴（修：后台心跳线程 10s 地板 + EventWriter 线程锁）②冒烟 HOME 隔离屏蔽 pi 凭据（修：.config 与 .rick/pi symlink 桥）——realpi 门控的首个价值实证。

### 迭代 3：自改进搜索架构（task15-21，7 task）
- 观察→假设→修复：初版均匀采样 → 加权随机采样 + 格手术 + expand_cell + 决策轨迹 + prompt 进化层。
- 过程修复：e2e flaky 根治（pending 语义统一：档案 add_pending 同格 last-wins 去重，与投影一格一项对齐，e2e 10/10 稳定）；add_pending 移到 elite 事件后（事件先于档案原则——**事件溯源纪律**首次确立）。

### 迭代 4→5：实测驱动的成本治理（task22-26 等）
- **观察（痛点 #1）**：env_8 实测 builder 单次中位 478s/646k token、极值 1575s/2101k（23 回合）——**25M token 失控**，成本失控主因是回合数（每回合重算完整 input）。
- 假设：pi 无 max-turns flag → 框架侧在 NDJSON 流中计数 assistant 回合，超限优雅终止。
- 修复：回合预算（NDJSON 回合计数 + SIGTERM 优雅终止）；saturation 饱和信号注入；adjust terminate_recommend（认输/换轴表达）；ctx 可读兜底（None/空值渲染为人读文案，消灭 prompt 裸露 {{cell_elite}} 占位符）。
- 验证：gate8-10 全绿（336 单测）。

### 迭代 6-7：三要素契约 + research skill（task27-35）
- 观察：端到端真跑发现三要素脚本契约不统一 + env_9 迁移问题（b0bb3cf 修）。
- 修复：三要素（方法/信源/关注点）脚本统一契约；第四类 env 资产 research skill（检索三要素）注入四角色。
- gate 抓到 3 个真缺陷：①debug.md 从未进 git（.gitignore 通用规则命中）→ 加例外强制 add ②e2e fixture 覆写旧键 correctness.size → 继承 8192 规模超时误杀 ③e2e 小规模(96) 与加固门物理下界冲突。——**gate 与 e2e 的「真缺陷捕获」价值反复实证**。

### 迭代 8：实验驱动的四 Agent Loop（task36-42，7 task）
- 架构升级：super 调度从单一循环升级为「adjust（格管理）→ builder（实验设计）→ coding（编码）→ debug（修复）」四 Agent Loop，五件 super_* skill + experiment_record 恒常上下文。
- **回合预算实测校准**（两次 tune 提交）：先按新 loop 实测上调（builder 12→20 / coding 16→24 / debug 10→16 / adjust 8→12），再实测 adjust 12 回合 trunc 无提案 → 12→16。——「预算先给假设值、实测后校准」的模式确立。

### 迭代 9：成本可解释 + 并发安全（task44-48，5 task）
- **观察（痛点 #2）**：env_9 真数据 26.11M tokens / $17.59，**debug 20 次派发 12 次截断**——截断浪费可视化。
- 修复：super_cost_analysis skill + cost_report.py（role 表/迭代表/Top-N）让成本可解释；格级互斥（CellLocks 内存 + 文件 O_EXCL 双保险）+ 并发降级为串行保安全。
- 验证：gate19-22 全绿（413 单测）。

### 迭代 10：成本优化闭环（task49-53，5 task）
- 假设：human 判断「截断是 debug 浪费主因」。
- 修复：debug_max_turns=0 不截断（靠 agent_timeout_s + wall budget 兜底）+ debug 一次性根因定位五步协议 + 并发槽池（max_parallel_experiments=2 + threading.local 线程隔离 + 事件并行化 experiment_id/experiment_seq 可还原）。
- **验证（成本对比表，dev-iterations.md）**：tokens/迭代 2.18M → 1.74M（**-20%**）；debug 截断 12/20 → **0/3**；it16 soft_gate 失败被 debug 救回、it17 入格 601ms 精英；两槽并发窗口重叠实证。归因 caveat 诚实标注（2 迭代未触发 adjust，补齐后保守落在 -10~-15%）。
- 这是全 job 最完整的「观察→假设→修复→量化验证」闭环。

### 迭代 11：并发饥饿 + 进程收割（task54-55，2 task）
- 观察：红黑树 env 实跑发现 adjust 被并发槽饥饿（if not slots 判断导致 adjust 永远抢不到槽）→ 修：移出 if not slots + 跨边界触发 cursor-last>=N。
- 另：9f94d7b 清理——.rick/super/ 5335 个运行时文件误入 git → 加 .gitignore 并从索引移除。——**运行时数据与源码树的卫生纪律**。

### 迭代 12：agent 确定性输出（task56）
- 输入：human 三项裁决（回合软约束/JSON 确定性/session resume）。
- 修复：全部 *_max_turns → 0（不截断）+ turn_soft_warn=20 告警；JSON 修复 resume（--session-id 同会话恢复 + 「请只输出 JSON」一次修复）；失败恢复（超时/网络 → resume 续跑）。
- **但 json_repair 字段在 env_11 实测中 0 触发**——为迭代 13 的分析埋下伏笔。

### 迭代 12.5（穿插）：env_11 轨迹深分析（researcher subagent）
- 派 researcher subagent 全量分析 env_11（18140 事件流）：产出五项根因（R1-R5）——68% 迭代是 API 配额空转 / 回合×完成协议结构性失配（截断 → final_message=None → JSON 修复不触发 → debug 全量代偿 2× token）/ 幻影 elite（阴性结果被当新纪录）/ 处方-基座相位差（30-60min 级空转）/ ref 漂移 ±20.6% 大于头三名差异 4.4%。
- **重校准认知**：原统计「20% 产出率、93-99% invalid」是度量口径被基础设施失败污染——真实工作轮产出率 ≈61%。→ 直接催生迭代 13 的 R5（outcome 五态分类）。

### 迭代 13：R1-R5 数据驱动修复（task57-59，3 task）
- R1 基础设施熔断（三态 outcome + 配额识别 + 连续 3 次 → 暂停 + 60s 探活恢复 + infra_failure_concluded 事件注册——顺带修了调度器崩溃）
- R2 产物落盘抢救（agent 完成但没 JSON → workdir 产物直接 gate 入格；截断也可抢救——「做完了但没说」≠「没做」）
- R3 幻影 elite 拒绝（md5 等价 → elite_measurement_refresh 只刷测量不刷 gen/位次）
- R4 method_stale 自动标记（elite_replaced → [STALE] 注入 builder ctx）
- R5 噪声标定 + 停止判据（top3_spread < noise_band → headroom_below_noise；「何时停」由证据而非预算决定）
- 验证：gate29 全绿 + 521 测试全量 + cost_report 在旧事件流正确 fallback _legacy 桶。

### 迭代 13 后（本次会话）：JSON hook 诚实性事故 + release 跨机事故
- **JSON hook 三 bug**（详见 Step 2）：9667d13 提交「写完+部署+提交」但从未真实跑过 pi 进程；human 质询后补真实测试，发现 3 个 bug（导出形态/api.session 不存在/agent_settled 撞 stale-ctx）→ e2bfd6f 换 agent_end + deliverAs:"followUp"（运行时源码注释明示的官方模式）三测通过。
- **release 跨机事故**：S5 提升连环失败——spawnDetached 相对路径 bug（./bin/rick 在生产树 cwd 解析不到）→ 修后 8413 拓扑误判（真实生产是 6137）→ stopProd 快照断言漏杀 11135 → 自动回滚切到不可执行 ELF → 生产彻底起不来。job_49 接手根治（P0 四项）。

---

## Step 1b：失败/教训总清单

| # | 教训 | 出处 | 后果 |
|---|---|---|---|
| L1 | hook/扩展类改动「写完+部署+提交」≠「验证过」——9667d13 三 bug 全部是真实跑一次就能暴露的 | 迭代 13 JSON hook | 虚假完成感，human 质询才补测 |
| L2 | 度量口径被基础设施失败污染（93-99% invalid 里 86% 是配额空转）——统计先分类再下结论 | env_11 分析 | 早期优化方向可能错置 |
| L3 | 回合预算是「产物复杂度」的函数不是纪律问题——24 回合装不下「1 文件+31 测试+3 脚本」是算术问题 | env_11 R2 | debug 全量代偿 2× 成本 |
| L4 | 停止判据锚定预算跑道而非「剩余空间 < 测量分辨率」——近极限区继续迭代期望收益为负 | env_11 §五 | 尾部迭代纯浪费 |
| L5 | 环境迁移后「带历史注释的配置」不可信（start-web.sh 的 8413 + proxy 经验全部反转） | release 事故 | release 连环失败 |
| L6 | 回滚点必须可执行——自动回滚把「发布失败」升级成「生产起不来」 | release 事故 R3 | 生产死亡 3+ 小时 |
| L7 | gate 自身会有检查缺陷（字符串粗匹配/路径缺 PYTHONPATH）——当轮修正+透明化记录是正确模式 | gate2/3/9 | 无（已消化为惯例） |
| L8 | worker 身份漂移/上下文继承问题 → fresh 重派 + 角色护栏 + 要点内联 | gate3 t8 | 1 次空派发 |
| L9 | 运行时数据误入 git（5335 文件）——运行时目录必须 .gitignore 前置 | 9f94d7b | 仓库膨胀 |
| L10 | rsi 契约字段位置敏感（release.md 首个 version= 行被抓）——头部三行契约 | job_49 复证 | 已知坑第三次复现 |

---

## Step 2：更合理轨迹评估

### 评估 1：迭代 4 的 25M token 失控（最痛成本事故）

**实际路径**：env_8 跑完 → 事后统计发现 builder 极值 2101k/23 回合 → 迭代 4 计划回合预算 → 迭代 5 落地 NDJSON 计数 + 优雅终止。

**可预防性分析**：
- 冗余：无显著冗余工具调用。
- 可预防错误：**是**——失控发生在「跑的当时」而非「跑完之后」。回合数是实时可观测的（NDJSON 流就在那），但初版没有在线告警，只能事后从 usage 统计里发现。
- 更短路径：第一版就该带 turn 软告警（迭代 12 的 turn_soft_warn 其实是 8 个迭代之后才补的「本应在 V1 就有的最低配观测性」）。

**改进建议**：
1. **观测性先于治理**：任何长时 agent 循环 V1 必须带「实时计数 + 软告警事件」（回合/token/时长三维），事后统计是兜底不是主力。落到 superopt：turn_soft_warn、budget 告警事件应该在迭代 1 的骨架里，迭代 4/9/12 的三轮治理会缩短为最多两轮。
2. **预算值用产物复杂度反推**：先数一次典型任务需要多少「文件×测试×脚本」动作数，除以每回合动作数得出预算下界，再给实测余量——而不是先拍 12 再 tune 到 20/24（两次 tune 提交本来可以一次算对）。

### 评估 2：迭代 9→10 的 debug 截断浪费（量化最充分的闭环）

**实际路径**：迭代 9 建 cost_report 让浪费可见（12/20 截断）→ human 判断「截断是主因」→ 迭代 10 改 debug_max_turns=0 + 一次性根因协议 → -20% tokens 验证。

**可预防性分析**：
- 这段做得好：「先可解释再优化」顺序正确，假设由 human 提出、worker 量化验证，归因 caveat 诚实（2 迭代未触发 adjust，保守 -10~-15%）。
- 唯一可缩短处：截断信号在迭代 9 之前就存在于 events.jsonl（truncated 字段），但没人看——**浪费的证据早就有了，缺的是「看」的动作**。cost_report 的价值一半是工具，一半是「强制看」。
- 更短路径：迭代 3 建 events 体系时，把 truncated/invalid 作为默认 dashboard 指标（cost_report 的雏形），迭代 9 可省。

**改进建议**：
3. **每轮迭代收尾固定加一步「事件流健康巡检」**（一条 cost_report 命令）：把「浪费可见化」从专项 task 变成例行动作，预计能把「发现截断浪费」从迭代 9 提前到迭代 5-6。

### 评估 3：迭代 12/13 的 JSON 修复不触发根因（最深刻的链式教训）

**实际路径**：迭代 12 实现 client 层 JSON 修复 resume（触发条件含 `result.final_message` 非空）→ env_11 实测 json_repair=0 触发 → 分析发现 turn_budget 截断时 final_message=None（agent 还在做中间操作就被杀了）→ 迭代 13 R2 产物抢救 + pi hook。pi hook 首版（9667d13）未真实测试就提交 → human 质询 → 补测发现三 bug → e2bfd6f 换 agent_end 模式三测通过。

**可预防性分析**：
- 链条上每一环的修复都是对的，但**链条本可以短很多**：
  - 触发条件 `final_message 非空` 是「想象中的 agent 行为」——如果迭代 12 落地时抓一条真实 env_11 的 agent_ended 事件流看一眼（final_message=None 的比例），当场就能发现条件不成立。
  - 9667d13 的三个 bug（`export default` 导出形态 / `api.session?.messages` 不存在 / agent_settled 撞 stale-ctx）**全部**是「跑一次真实进程」就能暴露的——第一项甚至是 pi 加载器当场报错。
- 根本教训：**「client 层能想到的兜底」和「运行时真实行为」之间隔着一层没有验证的想象**。写 hook 前先读运行时源码（agent-loop.js 的 followUp 队列注释其实一直写着官方模式）+ 写完立刻跑最小真实验证，两步各 10 分钟，省掉一轮「提交→质询→返工」。

**改进建议**：
4. **外部运行时集成的验证协议**（新 skill 候选 `runtime_hook_verify`）：凡 hook/extension/进程间协议类改动，提交前必须过三关：①静态关——对照目标运行时的类型定义/接口签名逐条核（api.session 存在吗？handler 签名对吗？）②加载关——真实进程跑一次看加载日志（导出形态错误当场报）③行为关——事件流断言修复路径真的触发（像 run5.jsonl 的 turn_start→user 注入→assistant JSON 那样的时间序证据）。
5. **写 hook 前先读运行时源码里的官方模式注释**：pi 的 agent-loop.js/_handlePostAgentRun 注释明示了「agent_end handler + followUp 队列」是设计内路径；agent_settled 的重入限制也写在错误信息里。「读源码找官方姿势」比「从 API 表面猜组合」快且稳。

### 评估 4：release 跨机事故（job_47 收尾失败，job_49 根治）

**实际路径**：S3 演练（dry-run）全绿 → S5 正式提升连环失败：spawnDetached 相对路径 → 8413/6137 拓扑误判 → stopProd 快照断言漏杀 → 回滚切到不可执行 ELF → 生产死亡。job_49 接手：P0-1~P4 四项根治 + 真实 promote 实战验证（stopped=[旧pid] skipped=[]）。

**可预防性分析**：
- S3 演练绿但 S5 死——**dry-run 按设计不覆盖 merge/promote/restart/health**（当时已向 human 诚实说明），但「演练绿」带来的心理安全是虚假的：真正被验证过的只有「构建+门禁」。
- 拓扑误判（8413 vs 6137）是信息过时问题：start-web.sh 的 8413 是跨机前的事实，环境换了但「带历史注释的配置」没换。
- stopProd 快照断言漏杀 + 回滚不可执行校验缺失，都是「防御性代码自己没有防御」——与 L1 同构：**发布链路的关键环节从未在真实跨机环境全链路演练过**。

**改进建议**：
6. **S3 演练需要「带暂停点的全链路影子演练」**：对 promote/restart 这类 dry-run 覆盖不到的阶段，在隔离 state-dir + 临时端口跑一次**真提升**（不是 dry-run），验证 stopProd 认领/回滚点可执行/health build_id 校验三个环节。job_49 的 P0-3 复现方法论（临时 state-dir + 6199 端口起实例再 promote）就是标准做法，值得固化为 release 演练 skill 的一节。
7. **环境迁移后对「带历史注释的配置项」做实测校准**（job_49 已沉淀 env_drift_probe skill，本 job 学习应引用而非重复）。

### 评估 5：迭代节奏本身（13 轮迭代的元观察）

- 13 轮迭代构成清晰的三段式：**功能段（1-3）→ 治理段（4-10）→ 数据驱动段（11-13）**。治理段最长，其中 4→5→9→10 四轮都在治成本/截断/预算同一主题——若建议 3（事件流健康巡检例行化）早落地，治理段可压缩 1-2 轮。
- 末期「真实数据驱动」（env_11 researcher 分析 → R1-R5）效率显著高于早中期「跑完看统计再猜」——**把分析前置成 subagent 深读事件流的动作**，是本 job 后半程最重要的方法论升级。
- 迭代 12→13 之间那次「researcher 全量分析 env_11」是转折点：58 个 task 里最后 3 个（task57-59）的单位产出/洞察密度是全 job 最高的。

**改进建议**：
8. **每个 super 环境跑完/跑挂后，固定派一次 researcher 深读事件流**（不只看 cost_report 汇总）：env_11 分析一次就挖出 5 项根因（其中 R1 一项就占 68% 迭代空转）——这个动作的 ROI 是全 job 最高的单一实践，应写进 super-loop 的 Step 7（成本复盘）成为硬性环节。

---

## 给 Step 3-4（skill/loop 提取）的输入建议

**Skill 候选**（按价值排序）：
1. `runtime_hook_verify_skill`——外部运行时 hook 三关验证协议（静态对照/加载/行为事件流断言）；触发场景：写任何 extension/hook/进程间协议。素材：9667d13→e2bfd6f 的三 bug 链 + run5.jsonl 时间序证据法。
2. `super_event_autopsy_skill`——super 环境事件流深读法（重校准度量口径→分类失败→找结构性根因→量化每项的浪费占比）；素材：env_11 分析的 R1-R5 + 61% vs 20% 的重校准。
3. `budget_from_artifact_complexity_skill`——回合预算从产物复杂度反推（动作数×每回合动作数→预算下界→实测余量）；素材：两次 tune 提交 + env_11 R2 的算术论证。

**Loop 升级候选**：
- `super-loop.md` Step 7 补「事件流健康巡检 + researcher 深读」硬性环节（建议 3/8）；补「观测性先于治理」V1 最低配清单（建议 1）。

**Domain 事实候选**（供 Step 5）：
- pi 运行时官方模式：agent_end + deliverAs:"followUp"（agent-loop.js followUp 队列续跑）；agent_settled 内 sendUserMessage 撞 stale-ctx。
- rsi 契约字段位置敏感（release.md 首个 version= 行）——已知坑第三次复现。
- macOS darwin procfs：Environ 不可用（KERN_PROCARGS2 env 部分不可靠）→ 所有权断言退化路径。
- 事件先于档案原则 + add_pending last-wins（迭代 3 e2e flaky 根治）。

---

## 摘要（给 parent 验收）

- **Step 1 完成**：13 轮迭代逐轮「观察→假设→修复→验证」轨迹已还原；关键决策点（human 裁决执行树/job 编号防假通过/预算两次实测校准/human 截断假设量化确认）与 10 条失败/教训清单（L1-L10）已列出。
- **Step 2 完成**：4 个最痛迭代（25M 失控/debug 截断/JSON 不触发/release 跨机）+ 迭代节奏元观察已评估，8 条具体可执行改进建议；3 个 skill 候选、1 个 loop 升级候选、4 条 domain 事实候选已为 Step 3-5 备好输入。
- **核心元教训一句话**：本 job 的浪费大头不在代码而在**验证缺口**——观测性迟到（迭代 4）、浪费证据迟到（迭代 9）、hook 行为未验证（9667d13）、发布链未演练（S5）——四者同根：「跑起来看」这个动作在流程里出现得太晚。
