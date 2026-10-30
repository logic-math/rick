# Research: env_11（红黑树范围查询优化）super 模式完整轨迹复盘 —— 系统设计反思与改进建议

> 数据源：`.rick/super/env_11/events.jsonl`（18140 行）、`archive.json`（3564 行）、`decisions/iter161/163/561.md`、`codes/iter001-7378e2f6/iter552-df48c3b4/iter559-fc0cd2f0/experiment_record.md`、`super.json`、`super.log`。
> 说明：本环境未提供 shell 工具，`super/tools/cost_report.py` 无法执行；所有 token/成本数字均取自 `events.jsonl` 的 `usage` 字段与 `iteration_ended` 汇总事件，属抽样统计而非全量聚合（见 §7 Gaps）。

---

## 摘要

env_11 的"562 迭代 / 20% 产出率 / 93-99% invalid"**不是 agent 能力问题，而是度量口径被基础设施失败污染的问题**：约 380 轮（≈68%）迭代是 API 额度耗尽后的零 token 空转，调度器无熔断，把整条 4 角色流水线（adjust + builder + coding + debug×3）反复空跑并照样入账为 invalid。真正发生工作的约 180 轮迭代中，产出率约 61%，且实验方法论（预注册证伪判据、A/B 单变量归因、md5/机器码级核验）质量极高；真正的瓶颈是**回合预算与完成协议的结构性失配**（builder=20 / coding=24 回合必撞墙，final_message 从不产生，json_repair 因此 0 触发，工作靠 debug 全量代偿）与**知识回写滞后**（处方-基座相位差 3 格在案）。最优解 4.734ms 已处于近极限区间：识别出的剩余杠杆合计 2-4ns（4-8%），低于跨会话 ±12-18% 的机器漂移，排序依据 `ref_runtime_ms` 本身摆动 ±20%。

---

## 一、先校正关键前提：80% 无产出的真实构成

### 1.1 主因是"额度饥荒空转"，不是 agent 失败

**证据链（四条独立证据互相印证）：**

1. **直接观测**：`events.jsonl` seq 9494-9538（iter170-172，13:44）、seq 15007-15038（iter430-431，13:55）、seq 18089-18131（iter561-562，16:30）三段，所有 `agent_ended` 的 `usage` 全为 0、`wall_ms` ≈ 820-890ms、summary 为同一条配额报错（"您的2026-10-29额度已使用完毕，当前剩余额度为 0…credit-hub.sankuai.com"）。每次空转仍完整走 adjust→builder→coding→debug×3（`debug_max_attempts=3` 全部重试完），单轮 `iteration_ended.wall_ms` ≈ 9.5s、`tokens=0`。
2. **agent 侧自述**：`codes/iter552-df48c3b4/experiment_record.md` §1 明文："iter168 已为同一杠杆写下完整设计，但 coding/debug **因额度耗尽零执行**…**iter472-547 本格历次访问同样全是额度占位**"。
3. **决策侧台账**：`decisions/iter561.md` §2 人工复核项④："**额度饥荒 382 轮台账污染**（iter548-551）"。
4. **时间线算术**：iter163 结束于 10-29 11:29（seq 9091），iter170 开始于 13:44（seq 9494）——2.25 小时仅推进 7 轮（builder 真实工作、coding/debug 被额度杀死）；13:44→13:55 的 11.8 分钟内却推进了 258 轮（2.7s/轮）——只有零 token 空转才可能达到该速率。13:44→15:00 约 76 分钟 ≈ 377 轮 × 12s，与"iter472-547 全是额度占位"吻合。

**重构后的账目：**

| 项 | 数值 | 说明 |
|---|---|---|
| 总迭代 | 562 | |
| 额度空转轮（infra） | **≈380（68%）** | 零 token、零工具调用、零产物 |
| 真实工作轮 | **≈180** | 含 iter1-163、iter164-170（半程）、iter548-561 |
| 有候选产出 | 110 | **真实轮产出率 ≈ 61%，而非 20%** |
| 无产出 | 451 | ≈380（infra）+ ≈71（真实失败） |

**同样适用于 invalid 派发统计**：debug 1491 次派发中约 3×380 ≈ 1140 次是纯额度重试，占 1334 次 invalid 的 **86%**。adjust 271 次中约 190 次为空转（`proposals:[]`、decision 文件为空壳）。**"builder 93% / coding 99.8% / debug 89% invalid"这三个数字主要度量的是基础设施，不是模型。**

### 1.2 运行的真实终止原因

`super.log` 末行 + seq 18117/18133：
- seq 18117：`experiment_failed … error: "invalid event: type must be one of EVENT_TYPES (33 types), got 'infra_failure_concluded'"` —— **调度器尝试写入一个未注册的事件类型，直接抛异常终止实验循环**（这是一个真 bug：infra 失败处置路径自身没有被测过）。
- seq 18133：`session_ended {iteration_cursor: 562, budget_exhausted: false, infra_failure: true}`。
- 即：**既不是预算耗尽也不是自然收敛，而是"额度耗尽 + 事件类型校验 bug"的组合崩塌**。`super.json` 里 `budget.max_wall_time_h=24` 而 `wall_start=null`，墙钟预算事实上未启用；而 adjust 在 iter561 还在判断"跑道 ≈22.8h，远离收手判据"——**预算模型里根本没有"API 额度"这一维度**。

---

## 二、agent 实际在做什么：工作质量远高于产出率所暗示的水平

抽样 `agent_ended.summary` 与六段式 `experiment_record.md`：

**builder**（如 iter001、iter552）：产出"一手画像 profile.py 实测 → 调研文献（Khuong&Morin 1509.05053 / BP-tree PVLDB'23 / YCSB-E 分档归因）→ 可证伪假设带 + 预注册否定条款 + 对照组设计 + 具体编码清单"。iter552 的六段式记录含 **D0/D6/D7 三路归因设计**（零写折叠 / 零读常数写两个诊断锚），并显式声明"诊断锚仅 ab 口径、永不进门禁"——这是相当高级的实验设计素养。

**debug**（如 iter161、iter558、iter559）：根因分析到**机器码级**——iter161 用反汇编（0x173c/0x174c）证明"父基座 emit 标量循环已被 clang -O2 loop-idiom 自动向量化，手写 8 路 NEON 与之指令级等价（ldp/stp q 对），本格不存在账本所称 1.11ns/元可回收税"，据此**预注册回退**而非硬改代码凑门禁；iter558 用 `objdump` 核验 `umulh×2/prfm×2/零 udiv` 证明插值种子路径真实生效。

**adjust**（decisions/iter561.md、iter163.md）：含区间逐条归因表、主要矛盾判定、被排除的候选矛盾、辩证逆转、良质批判（置信度 + 残余假设 + 止损条件）、**预注册证伪条件 (a)-(e)**、防震荡核查。方法论上无可指摘。

**结论：这条流水线的"科学内核"是好的；坏掉的是承载它的工程外壳。**

---

## 三、invalid 的根因链（分层）

```
L0 基础设施层（占 invalid 的 ~86%）
   └─ API 额度耗尽 → agent 秒退（0 token / 800ms / 配额报错文案）
      → 调度器无熔断 → 4 角色全链空跑 → debug×3 重试 → 全部记 invalid
      → 最终 infra_failure_concluded 未注册 → 调度器崩溃

L1 回合预算层（真实工作轮的主失效）
   └─ builder 预算=20 回合、coding 预算=24 回合
      → C++ 单变量改造（改 solution.cpp + 14-31 个测试 + verify/benchmark/gate）24 回合装不下
      → exit_code=143 / "[turn-budget-exceeded: 24]" / truncated=true → valid=false
      → 但文件已部分落盘（candidate_produced 有 code_path）
      → 每次损耗 1.4-2.2M tokens / $1.4-1.7

L2 完成协议层（让 json_repair 永远为 0 的那一个）
   └─ 被截断的 agent 从未发出结构化 final_message → final_message=None
      → json_repair 的触发条件要求"存在但畸形"的 final_message → 永不触发
      → 自检 metrics 为空 {"passed":false,"stage":"self_check","metrics":{}}
      → 未被截断的 agent 也常以纯文本/markdown 收尾（iter004 builder summary="``"；
         iter002 coding summary="现在"）→ 同样 final_message=None
      → exit_code=0、truncated=false、valid=false —— 系统把这归类为"agent 失败"

L3 角色冗余层（成本放大器）
   └─ coding 被截断 → debug 从头重做编码工作（"按 coding_plan §1 单次补全 11 处编辑"）
      → 一份实现花两份 token；decisions/iter561："编码侧截断 7 连（iter553-559 coding
        全灭于 turn-24/20；全部交付靠 debug 代偿：iter152/553/554/557/558/559 六判例链）"
```

**关于用户给出的 "trunc=0（所有 max_turns=0 生效，不截断了）"**：与事件流直接矛盾——events.jsonl 中大量 `agent_ended` 带 `"truncated":true`、`exit_code:143`、`"[turn-budget-exceeded: 20]"` / `"[turn-budget-exceeded: 24]"`（seq 27、57、142、9012、9082、9091、17986、18063 等）。若 `trunc` 统计的是另一处配置字段，则该字段与实际生效的 per-role 回合预算（builder=20 / coding=24）不一致，**"截断已关闭"的判断不成立**，且回合截断恰是真实工作轮 invalid 的第一主因。

---

## 四、adjust 决策质量：内核优秀，三个工程性失效

### 4.1 失效一：JSON 输出契约不可靠 → 决策丢失，需要人工抢救

seq 9031（10-29 11:04:21，`type:"intervention"`, source:"human"）：人工全量重发 iter161 的 5 项决策，原文说明——"**iter161 轮 adjust 的 5 项动作因输出契约失败（JSON 未发出）丢失**，但其决策手写稿 decisions/iter161.md 完整。中控已逐项核实…请下一轮 adjust 将这 5 项动作原样重发"。`decisions/iter561.md` §2 人工复核项③："adjust 槽 JSON 输出契约失效（**≥6 历史判例**，本槽首字符即围栏对冲）"。

即：adjust 的思维产物（.md 手稿）完整且正确，但**机器可执行的动作通道（JSON 提案）反复断链**，靠人工兜底。271 次 adjust 中 225 次 invalid，大部分是空转，但其中至少 6 次是"想对了却没发出去"。

### 4.2 失效二：处方-基座相位差（知识回写滞后）

`decisions/iter561.md` §2 判定"主要矛盾 = **已验证机制知识与格处方快照之间的相位差已双向化**"，在案 3 格：
- `flat|bg|eytz`：method 基座停在 iter148 5.534 gen4，实际 elite = iter559 4.942 gen6 —— **2 个机制代滞后**；
- `flat|br|btree_wide`：method 基座 iter165 5.874 gen3 + 已被证伪的 RANKCOPY 处方，实际 elite = iter558 5.332 gen4 —— **1 代滞后 + 处方失准**；
- `two|br|rb_soa`：elite = iter156 5.218 gen2 但 method 仍指 iter153 5.584 gen1 —— **跨纪元遗留**。

后果（决策原文）："w24 高份额格的下一访按现文本会重做 KQS 移植（**30-60min 级空转**）"。结构成因：`adjust_interval=2` 与 replace 落点速率同级 + `modify_cell ≤2/轮` 硬上限造成回写积压。**adjust 自己已经给出了正确修法**："建议框架侧在 elite_replaced 时自动标记 method_stale 标志位"——这是应该被采纳的框架级改动，而非每轮靠 adjust 人工核对 elite gen vs method 基座 gen。

### 4.3 失效三：幻影纪录（同码重测被当成机制增益入账）

`codes/iter552-df48c3b4/experiment_record.md` §6 是一份**阴性结果**记录：stnp 非时性写出被三口径 7/7 同向判负（k1024 isolated +7~+11ns、mb 预筛 Δ=+7.2~+7.9ns/8.2KB），处置为 `keep_original`、`KNT_STORE` 缺省置 0，"交付形态 = **elite 等价、零回归**"、"PARENT 4.783/4.785ms ≈ elite 4.747ms"。但 gate_check 跑出 `runtime_ms=4.729`（ref 6028ms）→ **被入账为新的全场 #1（4.734）**。同一现象已被 adjust 在 iter161 亲自诊断过："elite 账面 6.064 gen4 = 同码重测非机制增益——真实水位 6.094-6.123 同码噪声 ±1.5%，勿把 6.064 当增益基线"，并把"同形态入账判据（iter551/552/556 三判例，建议纳入 md5/热路径等价拒绝条件）"列为人工复核项①。**即"4.734 纪录"大概率是 iter159 同一份代码在一次更有利的机器态下的重测值**，而台账把它当成了新纪录。

### 4.4 另一个测量完整性问题：ref 漂移大于被排序的差异

`gate_result` 中 `ref_runtime_ms` 跨全运行摆动 **4999.3 → 6028.0 ms（+20.6%）**（iter559 4999.3 / iter558 5133.9 / iter1 5363.5 / iter161 5503.2 / iter552 6028.0），而头三名候选之间只差 4.4%（4.734/4.879/4.942）。`score = ref_runtime / runtime`，**排序键的方差是它要区分的差异量的 4 倍以上**。

---

## 五、4.73ms 是否接近理论极限？

**负载规格**（`experiment_record.md` §5）：M=100000 元素、Q=100000 探针、`k_mix {0,8,128,1024} @ 10/40/30/20`、K=5 中位、avg_k=248.43、每元素 8B 输出。

**纪录格实测分档**（iter159 基座，iter552 当日实测 +12~18% 漂移已标注）：

| 档 | isolated (当日) | iter159 带估计 | 每查询数据量 |
|---|---|---|---|
| k0 | 9.35 ns | ≈7.65 ns | 0 B（纯双侧搜索） |
| k8 | 14.40 ns | ≈12.5 ns | 64 B |
| k128 | 42.65 ns | ≈37 ns | 1 KB |
| k1024 | 143.50 ns | ≈132 ns | 8 KB |
| mix | 50.01 ns | ≈45 ns | 平均 1987 B |
| gap (mix−加权和) | +1.82 ns | — | 交错税 |

**五个判断：**

1. **搜索侧已经到顶。** k0=7.65ns 完成 100K 键上两次边界定位 + span 分派；手写 branchless+prefetch 的单次下降本就在 4-6ns 量级，两次加分发 ≈8-10ns 是合理下界。iter552 的 H3 归因（D7 零读把 gap 打到负值）进一步确认剩余税在"读侧/emit 工作量"而非可消除的分支。
2. **拷贝侧在 60 GB/s 量级。** 由 k1024 反推：8192B / (143.5−9.35)ns ≈ **61 GB/s** 有效吞吐；整基准 198.7MB / 473.4μs ≈ 42 GB/s 聚合。arm64 单核 L1/L2 驻留拷贝峰值大致在 60-120 GB/s，**已用掉一半以上**。
3. **k8/k128 路径存在结构性低效，但无已知机制能回收。** 若小拷贝也能达到 60 GB/s（k8→8.7ns、k128→24.7ns）且 gap 清零，mix ≈ 38ns ≈ **3.8ms**——这是"带宽理想地板"，比纪录低约 19%。但实测低效来自调用流/emit 结构与尾处理，非带宽；iter552 刚刚证伪了写侧（stnp）这一条路。
4. **收敛签名明显。** 头三名来自三条不同布局线（two_level|bg|eytz / flat|br|eytz / flat|bg|eytz），分别为 4.734 / 4.879 / 4.942ms，**彼此相差 4.4%**——独立机制汇到同一水位是典型的近极限特征。flat|br|eytz gen5 相对 gen4 只 +0.005ms（容差带内），已平台化。
5. **测量分辨率已经低于剩余可识别空间。** 剩余 armed 杠杆合计 2-4ns（decisions/iter561：孪生队列 2-4ns ≈ 覆盖 2.1ns 差距），即 **4-8%**；而同会话噪声 ±1.5%、跨会话漂移 **+12-18%**、ref 摆动 ±20%。官方 benchmark/gate 用绝对值 + 5% 软带，**4-8% 的真实增益在该口径下无法被可靠认证**。

**结论：4.73ms 处于"近极限、强负边际收益"区间。** 它距离一个纯带宽理想地板（≈3.8-4.2ms）还有 10-19%，但没有任何已识别机制能抵达；已识别的 2-4ns 增益空间小于测量分辨率。继续迭代的期望收益为负——正确动作是提交 `terminate_recommend`，而系统从未提交（iter561："terminate_recommend 不提交（saturation 空、跑道 ≈22.8h）"）。**停止判据目前锚定在预算跑道上，而不是"剩余空间 < 测量分辨率"上，这是一个设计缺口。**

---

## 六、优先级排序的系统改进建议（数据驱动）

### R1【最高优先级】基础设施熔断：把"额度耗尽"从"agent invalid"里剥离

- **证据**：382 轮零 token 空转（§1.1 四条证据链）；每轮仍烧完整 4 角色链（adjust+builder+coding+debug×3）；`infra_failure_concluded` 不在 EVENT_TYPES 导致调度器崩溃（seq 18117）；session 以 `infra_failure:true` 而非预算耗尽结束（seq 18133）；预算模型无 API 额度维度（`super.json` budget 只有 wall_time 且 `wall_start:null`）。
- **改法**：① 失败分类三态化：`infra_failure` / `agent_failure` / `no_output`，配额报错（`total_tokens==0 ∧ wall_ms<2000 ∧ summary 匹配配额文案`）一律归 infra；② 触发即熔断：停派发、暂停 `iteration_cursor` 推进、不创建 `codes/iterNNN` 目录、不写空壳 `decisions/iterNNN.md`，按 `00:00:00 自动重置` 定时探活恢复；③ 把 `infra_failure_concluded`（以及同族事件）补进 `EVENT_TYPES`，并为 infra 处置路径写一条单测；④ 把 API 额度纳入 budget 模型（adjust 的"跑道充裕 ⟹ 延续"判断在额度已死 3 小时后仍在做）。
- **预期收益**：消灭 68% 的无效迭代、86% 的 invalid debug 派发，且杜绝结尾的调度器崩溃。

### R2 修"回合预算 × 完成协议"的结构性失配

- **证据**：builder=20 / coding=24 回合必撞（`[turn-budget-exceeded: 20/24]`、exit 143、truncated:true，见 seq 27/57/142/9012/9082/9091/17986/18063）；decisions/iter561 "编码侧截断 7 连（iter553-559 coding 全灭），全部交付靠 debug 代偿（六判例链）"；每次截断损耗 1.4-2.2M tokens / $1.4-1.7（seq 9012=2.04M/$1.70、seq 9082=2.20M/$1.68、seq 9091=2.17M/$1.39、seq 18063=1.74M/$1.25）；`json_repair=0` 的原因正是 `final_message=None` 不满足触发条件；未被截断的 agent 也以纯文本收尾（iter004 builder summary="``"、iter002 coding summary="现在"），exit 0 却 valid=false。
- **改法**：① **把最终结构化输出改成一次工具调用落盘**（写 `final.json` 到 workdir + 一个极短的确认回合），让"做完了但没说"不再等于"没做"；② 回合预算按**产物复杂度**计量（C++ 格的 coding 需要"改 1 个文件 + 14-31 个测试 + 3 个脚本跑通"，24 回合装不下是算术问题不是纪律问题）；③ **截断可抢救**：被 SIGTERM 的 agent 已落盘文件 + md5 直接交给 debug 做"增量续写"，而不是让 debug 从头重做整份实现（L3 冗余目前把成本放大 ~2×）；④ 修 `json_repair` 触发条件，覆盖 `final_message is None` 但 workdir 有产物的情形（从产物反推 fix_report）。
- **预期收益**：消灭真实工作轮的第一大损耗源，估算可省下 7 连截断 × ~$1.5 ≈ 10-20% 的真实工作成本，并把 debug 从"代偿编码"还原为"修缺陷"。

### R3 修"幻影 elite / 同码重测入账"与 ref 漂移

- **证据**：iter552 是 keep_original 阴性结果（KNT_STORE 缺省 0、默认构建 = 父本逐字路径、"交付形态 = elite 等价、零回归"），却被 gate 的 4.729ms 入账为全场 #1（4.734）；iter161 已诊断同型问题并建议"同形态入账判据…纳入 md5/热路径等价拒绝条件"（iter551/552/556 三判例）；`ref_runtime_ms` 摆动 4999→6028（+20.6%）而头三名候选仅差 4.4%。
- **改法**：① elite 准入时比对候选与在位精英的 `solution.cpp` md5（或热路径等价哈希），等价则拒绝 `elite_replaced`、只刷新测量值不刷新 gen/位次；② 排序改为**同会话配对 A/B 比值**（agent 侧已有该纪律："判据一律用同进程差分，禁跨会话比绝对值"——把这条纪律上收到框架层）；③ 每次门禁记录一个机器态锚（当日父本重测值），漂移超阈值时自动降级为 informational。
- **预期收益**：保护台账与位次的可信度，避免后续 adjust 基于幻影纪录做权重决策（flat|br|eytz 的 weight 16→20 之类动作目前就建立在这样的读数上）。

### R4 消除处方-基座相位差：elite_replaced 时自动标记 method_stale

- **证据**：decisions/iter561 §2 在案 3 格相位差（2 代滞后 / 1 代+处方已证伪 / 跨纪元遗留），后果"w24 高份额格的下一访按现文本会重做已入库机制（30-60min 级空转）"；结构成因 `adjust_interval=2` × `modify_cell ≤2/轮` 造成回写积压；adjust 自己提议"框架侧在 elite_replaced 时自动标记 method_stale 标志位"。
- **改法**：`elite_replaced` 事件发生时框架自动把该格 `descriptor.method` 置 stale 并注入新基座路径 + 新精英 md5；stale 未清除前，builder 的 prompt 强制以 `elite_code_path` 的实际内容为第一事实源（decisions/iter561 已把"开工第一步 = 基座逐行读码审计"固化为条款，把它从 prompt 约定升级为框架注入即可）。
- **预期收益**：每次高权重格被访可省 30-60 分钟空转；把知识回写从"每 2 迭代一批、每批 ≤2 格"的节奏解耦出来。

### R5 度量口径分离 + "剩余空间 < 测量分辨率"的停止判据

- **证据**：20% 产出率、93-99% invalid 三个数字全部混合了 infra/回合/完成协议/真实无产出四类成因（§1、§三），导致"80% 无产出"这个结论指向了错误的对象；同时系统没有任何规则在"已识别杠杆 2-4ns < 跨会话漂移 12-18%"时收手，iter561 仍以"跑道 22.8h"为由继续（§五-5）。
- **改法**：① `iteration_ended` 增加 `outcome ∈ {delivered, agent_failed, turn_truncated, infra_noop, no_output}`，所有统计按此分列——这是一次几小时的改动，却能让后续所有 super 运行的诊断从"猜"变成"读"；② 给 adjust 增加 `terminate_recommend` 的硬判据：`已识别剩余杠杆合计(ns) < 同会话噪声带 × 当前纪录(ns)` 或 `top-3 格位次差 < 噪声带`，触发即建议收手；③ 噪声带本身由框架每 N 轮跑一次同码双测自动标定（agent 已经在手工做"当日机器漂移 +4~6% 判读同进程差分"）。
- **预期收益**：让下一次复盘不需要 380 轮空转之后才能发现问题；并让"何时停"由证据而非预算决定。

---

## 七、Gaps（无法完全确证的部分）

1. **无法执行 `cost_report.py`**（本任务环境无 shell 工具）。所有 token/成本数字来自 events.jsonl 的抽样。380 轮空转数是从四条间接证据推得（决策台账明文"382 轮"与我的独立时间线推算"≈380"一致，但非逐条计数）。建议用 `python3 super/tools/cost_report.py --workdir .rick/super/env_11` 复核，并按 `total_tokens==0 ∧ wall_ms<2000` 重新分列统计以确认本报告 §1.1 的重构账目。
2. **`trunc=0` 与事件流的矛盾**未能定位到具体配置字段（我只看到 builder=20 / coding=24 的实际生效预算）。若 `max_turns=0` 指另一处全局配置，说明存在"配置项与实际生效预算不一致"的第二个问题，值得核对 `super/` 的配置读取路径。
3. **4.734 的确切来源迭代**未能逐条定位（iter552 的 gate 读数为 4.729；iter159 为 4.747；台账最终值 4.734）。但"4.734 ≈ iter159 同码重测"这一判断有 iter552 §6 的 keep_original/零回归自述与 iter161 的同型判例双重支撑。
4. 决策文件仅精读了 3 份（iter161 经人工干预转述、iter163、iter561）+ 2 份实验记录全文 + 3 份部分；271 份决策的质量分布未全量评估。
5. 理想地板 3.8ms 是"带宽归一"外推（假设 k8/k128 路径能达到 k1024 的 61 GB/s），不是可实现的工程目标；仅作量级参照。

---

## 附：本次分析直接读取的证据位置（供复核）

| 发现 | 文件:位置 |
|---|---|
| 额度空转（iter170-172） | `events.jsonl` seq 9494-9538 |
| 额度空转（iter430-431） | `events.jsonl` seq 15007-15038 |
| 额度空转（iter561-562）+ 调度器崩溃 | `events.jsonl` seq 18089-18133 |
| `infra_failure_concluded` 事件类型 bug | `events.jsonl` seq 18117 |
| 人工抢救丢失的 adjust 决策（≥6 判例） | `events.jsonl` seq 9031（intervention, source=human） |
| 回合截断实例（builder 20 / coding 24） | `events.jsonl` seq 27/57/142/9012/9082/9091/17986/18063 |
| final_message 协议失效（exit 0 但 valid=false） | `events.jsonl` seq 49（summary="现在"）、seq 129（summary="```"） |
| 382 轮额度台账污染、7 连编码截断、≥6 次 JSON 契约失效 | `decisions/iter561.md` §2 良质批判/人工复核项 |
| 处方-基座相位差 3 格 + method_stale 建议 | `decisions/iter561.md` §2/§4 |
| 幻影纪录（同码重测入账） | `codes/iter552-df48c3b4/experiment_record.md` §6 + `decisions/iter163.md`（iter161 勘误） |
| k_mix 分布、分档读数、+12~18% 跨会话漂移 | `codes/iter552-df48c3b4/experiment_record.md` §1/§5 |
| 三头名收敛 4.734/4.879/4.942、95/41/36 访问 | `events.jsonl` seq 18080 `cells_status_summary` |
| worker 多次 signal:killed | `super.log`（12 次） |
| 墙钟预算未启用 | `super.json`（`budget.wall_start=null`） |

```acceptance-report
{
  "criteriaSatisfied": [
    {
      "id": "criterion-1",
      "status": "satisfied",
      "evidence": "报告含具体文件路径+seq 的证据定位表（§附），并按严重度分级：infra 级（额度空转 382 轮 / EVENT_TYPES bug seq 18117，68% 迭代无效）、协议级（final_message=None → json_repair 永不触发，≥6 次 JSON 契约失效需人工抢救 seq 9031）、预算级（turn-budget 20/24 七连截断，每次 $1.4-1.7）、台账级（同码重测入账为 4.734 纪录，iter552 §6 keep_original 自述 + iter163 勘误）。"
    }
  ],
  "changedFiles": [
    "/Users/logic/workdir/rick-dev/research.md"
  ],
  "testsAddedOrUpdated": [],
  "commandsRun": [
    {
      "command": "python3 super/tools/cost_report.py --workdir .rick/super/env_11",
      "result": "not-run",
      "summary": "任务环境未提供 shell 工具，无法执行；token/成本数字改取自 events.jsonl 的 usage 字段抽样，并在 §7 Gaps-1 显式声明该限制与建议复核命令。"
    }
  ],
  "validationOutput": [
    "已读 events.jsonl 头部(1-170)/中段(9000-9111, 9500-9544, 15000-15055)/尾部(17990-18134)，覆盖额度空转三段直接证据",
    "已读 archive.json 头部(1-100)与尾部(3435-3565)，并从 seq 18080 cells_status_summary 取得全部 80 个 cpp 格的 visit/weight/elite_perf 终值",
    "已读 decisions/iter561.md 全文、decisions/iter163.md 摘要、iter161 人工干预全文(seq 9031)",
    "已读 codes/iter001-7378e2f6、iter552-df48c3b4 两份 experiment_record.md 全部六段",
    "交叉验证：iter552 §5 mix=50.01 与分档加权和 48.19 + gap 1.82 逐项吻合；k_mix 权重 {0,8,128,1024}@{10,40,30,20} 推得 avg_k=246.4 vs 实测 248.43；377 轮×12s 与 13:44-15:00 窗口吻合；决策台账'382 轮'与独立推算'≈380'吻合"
  ],
  "residualRisks": [
    "380/562 空转轮数为四条间接证据推算而非逐条计数，误差估计 ±20 轮；需用 cost_report.py 或 events.jsonl 按 total_tokens==0 重算确认",
    "'trunc=0 不截断'与事件流中大量 truncated:true/[turn-budget-exceeded] 直接矛盾，未定位到用户所指的具体配置字段——可能存在第二处配置不一致问题",
    "4.734 纪录的具体来源迭代未逐条定位（iter552 gate=4.729 / iter159=4.747），'同码重测'判断依赖间接证据（iter552 §6 keep_original 自述 + iter161 同型判例）",
    "3.8ms 理想地板为带宽归一外推，仅作量级参照，非可实现工程目标",
    "271 份决策文件仅精读 3 份，adjust 决策质量分布未全量评估"
  ],
  "noStagedFiles": true,
  "diffSummary": "新建 /Users/logic/workdir/rick-dev/research.md：env_11 完整轨迹的系统设计复盘。核心结论为 80% 无产出中约 68% 是 API 额度空转（infra）而非 agent 失败，真实工作轮产出率约 61%；给出 invalid 四层根因链（infra → turn-budget → final_message 协议 → 角色冗余）、adjust 三个工程性失效（JSON 契约/相位差/幻影纪录）、4.73ms 近极限判定（剩余 4-8% < 测量分辨率 ±12-18%），及 5 条按优先级排序的数据驱动改进建议（配额熔断、完成协议与回合预算、幻影 elite 与 ref 漂移、method_stale 自动回写、度量分列与停止判据）。未改动任何 .rick/super/env_11 下的原始数据文件。",
  "reviewFindings": [
    "blocker（对 super 框架）: events.jsonl seq 18117 - 调度器写入未注册事件类型 'infra_failure_concluded' 抛异常并终止实验循环，session 以 infra_failure=true 结束（budget_exhausted=false）——infra 处置路径自身未经测试",
    "blocker（对 super 框架）: 无 API 额度熔断，额度耗尽后 4 角色流水线（含 debug×3 重试）空转 ≈380 轮并全部记为 invalid；预算模型（super.json budget 仅 wall_time 且 wall_start=null）不含 API 额度维度",
    "high: json_repair 触发条件要求 final_message 存在但畸形，而实际失效模式是 final_message=None（回合截断或纯文本收尾），因此 json_repair=0 且工作只能靠 debug 全量代偿（七连判例，成本放大 ~2×）",
    "high: elite 准入不比对 md5，同码重测被入账为新纪录（4.734 大概率为 iter159 同码重测）；ref_runtime_ms 摆动 4999-6028ms(+20.6%) 而头三名候选仅差 4.4%，排序键方差大于被排序差异",
    "medium: 处方-基座相位差 3 格在案（最高 2 个机制代滞后），高权重格每次被访预计空转 30-60min；建议在 elite_replaced 时框架级自动标记 method_stale（adjust 自己已在 decisions/iter561 提出该建议）",
    "medium: 停止判据锚定预算跑道而非'剩余空间<测量分辨率'——已识别杠杆 2-4ns(4-8%) 低于跨会话漂移 ±12-18%，iter561 仍以跑道 22.8h 为由拒绝 terminate_recommend"
  ],
  "manualNotes": "最需要向父级强调的一点：'562 迭代 20% 产出率、builder 93%/coding 99.8%/debug 89% invalid'这组数字目前度量的是基础设施与完成协议，不是模型能力——在修复 R1/R2 之前，任何基于这些数字的模型/角色调整都会打错靶。反过来，这条流水线的科学内核（预注册证伪、单变量 A/B、md5 与机器码级核验、阴性结果回写）质量很高，值得保留。另：运行真实终止原因是额度耗尽 + EVENT_TYPES bug 崩溃，并非收敛或预算耗尽。"
}
```
