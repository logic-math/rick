# env_7 设计树（grilling 活文档）

> 使命锚点：`requirement.md` = 「优化矩阵乘法」
> 协议：逐层追加；每层 = 模块 + pipeline + 判断节点；L1 调研记录随层落盘。

---

## 第 0 层（根）：OKR

O（Objective）：交付 env_7「优化矩阵乘法」super 进化使命的**完整闭环**。本使命是全链第 7 次尝试——env_1~6 均未真正启动进化（env_1~5 零事件空跑，env_6 止步 grilling 第 1 层、判断节点从未被 human 裁决）——本次以「**真实跑通全链**」为核心目标：需求理清 → env 建设合格 → 门先证红 → 后台进化真实运行（迭代递增 + 精英入格 + 北极星可读）→ 干预正确投递 → 干净停止 + 收尾报告。

**KR 集（演绎：KR1 ∧ KR2 ∧ KR3 ∧ KR4 ∧ KR5 ⟹ O）**：
- **KR1 需求理清**：design-tree.md 落盘且过 grilling_gate——本层判断节点全部经 human 裁决（不吞判断节点）。
- **KR2 env 合格**：gen_env 产出 `env_7/env/` + `env_7/gates/`（exploit 注册表 + 金样例），验收自检四项全过（参考过正确性门 / bench 出分 / exploit 全拒识 / naive 过门）。
- **KR3 启动成功**：super.json 配置正确（pi_path 显式等）→ `rick super start` 门先证红 100%（exploit 拒识 + 金样例零误杀）→ state=running + 心跳新鲜 + iteration_cursor 递增。
- **KR4 进化产出**：≥1 精英入格（elite_admitted / elite_replaced 事件）+ 北极星可读（入格数 / 各格最优 perf / token 分 role 成本）+ 干预投递且事件流可见 intervention（source=human）。
- **KR5 干净收尾**：`rick super stop` → state=stopped + session_ended 事件 + 无孤儿进程 + 北极星终值收尾报告（落盘 env_7 工作区）。

**OKR 充分性自检**：KR1 是前提（需求不清不建设 env）；KR2 是硬前置（无合格 env 不启动——super-loop 依赖准备纪律）；KR3 打通启动链路（含门先证红防线）；KR4 是使命主体产出（真实进化而非空跑）；KR5 是工程收尾（不留孤儿、可审计、可复盘）。五者联合 ⟹ 使命完整交付。✅

---

## 第 1 层：使命 pipeline（四大模块）

```
[M1 grilling 需求追问] → [M2 gen_env 环境建设] → [M3 启动与运行监督] → [M4 停止与收尾报告]
     （现在）              （KR2）                （KR3+KR4）           （KR5）
```

| 模块 | 职责 | 输入 | 输出 | 边界 |
|---|---|---|---|---|
| M1 grilling | 设计树追问循环，判断节点全部经 human 裁决 | requirement + 仓库事实 + 本机基准校准 | design-tree.md（过 grilling_gate） | 不建设 env、不启动进化 |
| M2 gen_env | 访谈式 env 建设（模板定制 / venv / exploit 注册表 / 金样例） | KR1 裁决 + matmul 模板 | `env_7/env/` + `env_7/gates/`，验收自检四项全过 | 门禁阈值属 human 决策，skill 只引导不代答 |
| M3 启动与运行监督 | super.json 配置 → start（门先证红）→ status 监控 → event 干预 | KR2 的 env + KR1 的预算/干预预案 | events.jsonl（唯一真相源，只读）+ archive 精英 + intervention 事件 | 中控不执行进化逻辑；不直写 events/archive |
| M4 停止与收尾 | stop 干净收尾 + 北极星终值报告 | M3 运行痕迹 | state=stopped + session_ended + 收尾报告落盘 | 报告形态待 L1 裁决 |

**层内 pipeline 依赖**：M1 → M2 → M3 → M4 严格串行（后者的输入是前者的产出）；M3 的预算/干预预案在 M1 就要裁决（影响 start 参数与监控阈值）。

### 调研记录（第 1 层 · L1/L2 提炼）

**已自查消解的事实**（信源：仓库文件直读 + 运行时验证，等级 A）：

1. **历史失败面**：env_1~6 全部 state=idle、events.jsonl 0 行——进化全链从未真实跑通。env_1 需求与 env_7 完全相同（「优化矩阵乘法」）。env_6 止步 grilling（设计树只有第 0/1 层，判断节点 J1-J5 提炼后从未向 human 批量追问）。
2. **matmul 模板完整可用**：`super/env_templates/matmul/`——env.json（512×512 float32 / 正确性容差 1e-3 / 软门禁带 5% / bench 预热 1 + K=3 中位数 / 单次 timeout 60s / 正确性门 64×64 timeout 30s / score=ref_median/cand_median 加速比）、feature_space 2 轴（impl_strategy×4 值 × memory_layout×3 值 = 12 格）、reference.py（np.matmul 锚点 + 性能基线）、naive_start.py（纯 Python 三重循环，512 单次 15s+）、test_correctness.py / bench.py（stdout 单行 JSON 协议）、self_check.py（agent 自检）、domain_knowledge.md（分析 SOP + 调优 SOP + 作弊模式）。
3. **框架机制**（super/superopt + internal/super + internal/handler/super.go 直读）：
   - gates_dir 固定 = `<ws>/gates`；门先证红要求 `exploit_registry.json`（JSON 数组，每条 {id, sample_path}，≥1 条）全拒识 + `golden_examples/` 目录全过；注册表缺失/空/malformed → ok=False → 启动中止。
   - **pi_path 必须显式写入 super.json config 段**（默认空 → PiAgentClient ValueError → 调度器 exit 4「env 不可用」）；Go start 的 initSuperWorkspace 幂等保留已有 config——需在 start 前预写。
   - `agent_models` config 可 per-role 降档（job_47 首跑实测：builder 占 token 60%+ 而产出比 coding 简单，config.py 注释建议降档示例 {"builder": "catpaw-proxy/deepseek-v4-flash"}）。
   - env 解析：super.json `env_path`（start --env 传入）或回退 `<ws>/env`；每迭代 subprocess 直调 pi（env 热更新：改 env 下迭代自动生效）。
   - 调度器默认节奏：adjust_interval=5 / debug_max_attempts=3 / soft_gate_k=3 / heartbeat 30s / max_wall_time_h=24 / max_iterations=0（不限）。
4. **realpi 冒烟已验证真 pi 链路**（super/tests/realpi_smoke.py，job_47/task14）：预写 super.json（含 pi_path）→ `rick super start --daemon` → ≥3 真迭代 + builder/coding agent_started 齐 + token 计量非零 + gate_result ≥1。链路可行性有先例。
5. **本机运行环境**：HOME=/Users/logic/workdir/rick-dev-home（RICK_PI_AGENT_DIR 同域指向 .rick/pi/agent）；pi 0.84.2 存在于 `.rick/pi/agent/runtime/node_modules/.bin/pi`；pi 默认 defaultProvider=catpaw-proxy / defaultModel=glm-5.3 / defaultThinkingLevel=high（PiAgentClient 不传 --provider/--model 时即用此默认）；系统 Python 3.9.6 无 numpy → env 需 .venv；机器 Apple M5 Max arm64（env_6 调研结论）。
6. **rick 二进制**在 `/Users/logic/workdir/rick-dev/bin/rick`（不在 PATH，命令需绝对路径或先 export PATH）。
7. **干预机制**：`rick super event --type <t> --payload <json>` 写 interventions/ 队列文件（类型如 pause/weight/resurrect/env_change/stop），调度器迭代边界摄取（延迟 ≤1 迭代），seq 幂等去重；events.jsonl/archive.json 单写者=py 调度器，中控绝不直写。

**research 派发**（重量级：本机基准实测 + 外部校准）：research-L1 **已完成**——简报 `research-L1.md`（163 行，A/B/C 信源分级 + 复现命令）。核心结论：

1. **基线（A）**：numpy 2.0.2 + Accelerate，512×512 f32 稳态 **0.185-0.195ms（~1450-1480 GFLOPS）**；首进程冷启动 1.38ms（机器级 SME 冷态，进程内预热消不掉；bench.py 同进程先 cand 后 ref，score 比值口径不受影响）。
2. **起点校准（A）**：naive 512 独立 4.15s / **bench 协议 8.85s（score=2.13e-5）**——比模板 docstring「15s+」预估快 ~4×（M5 Max 快核 + 模板 naive 已带转置优化）；bench 全程（预热 1+K3+ref）≈36s，60s 单次 timeout 余量 6.8×。纯 Python 真·naive（无转置）本机外推 5.4-5.7s。
3. **score 阶梯（A）**：naive 2.1e-5 → 逐行 np.dot **0.153**（一步向量化跳 ~7000×，sgemv 不触发 vecLib FP 噪声）→ 纯 np.matmul **0.98-1.00**。⚠️ **score=1.0 非安全通过线**——与参考全同的候选两次 bench 得 1.0016/0.9768（±3% 噪声，与 env.json 软门禁带 5% 匹配）；>1.0 必须绕开单调用 SME 路径（多进程多簇 或 Strassen 省乘法），理论余量存在但不宽。
4. **外部校准（B）**：实测 GFLOPS ≈ M2 Max 单簇 SME 稳态量级（AppleAccelerate.jl：单线程即饱和 1 个簇协处理器）——正常；vecLib 虚假 FP 标志根因确证（SME 进/出 Streaming SVE 模式将 FPSR 全置 1，numpy#28687 / PR#29223），**sgemm 触发、sgemv（np.dot）不触发**，模板 np.seterr 抑制正确且 numpy 2.0.2 无新坑。Apple 官方口径：共享环境对协处理器计时噪声大——与 ±3% 实测一致，进化期间避免并发重负载。
5. **进化预期（C，量级参考）**：AlphaEvolve 数千样本收敛（FunSearch 需百万）；OpenEvolve ~500 迭代到 SOTA、$0.01-0.05/迭代（Flash 级）、成熟系统典型改进个位数百分比、高方差（σ=30%）。映射到 env_7：评估极快（bench ≤36s 起点、<1s 成熟候选），naive→row-dot 是「LLM 一步就能给出」的跳变，**预期个位数迭代出现首个合格精英**；0.15→1.0 与 >1.0 难度陡增，**>1.0 不应作为默认达标线**。

**R7 上报/Gaps**：M5 Max 核簇拓扑未确证（多进程理论余量 ~2-4× 未实测）；「个位数迭代出首精英」是推断非同构公开数据；env_7 实际 token 成本/迭代无先例（job_47 realpi 只验证链路未计量）——预算建议「首小时观察再定」。

### 判断节点（第 1 层 · 终版——已并 research 结论，待 human 裁决 → 批量追问见 L3）

- **J1 使命定位与达标判据**：vanilla 全链首跑（无特殊增量）vs 带特殊要求（性能目标/验收形态）——什么算「跑通」（KR4 判据确认或收紧）。基准事实：首精英预期个位数迭代（row-dot 跳变）；score=1.0 非安全线（±3% 噪声）；>1.0 需多进程/Strassen 且高方差。
- **J2 任务规格**：512×512 float32（模板默认）vs 其它规模/精度。基准事实：512 处 SME 高吞吐甜点区，基线正常。
- **J3 门禁定制**：容差 1e-3 / 正确性门 64×64 / K=3 / timeout 照模板？exploit 样本集（3-5 个，构造方向：预存结果/篡改计时/降精度/部分计算）；金样例集（reference 之外是否补）。
- **J4 特征空间**：2 轴（impl_strategy×4 × memory_layout×3 = 12 格）照模板 vs 加轴。
- **J5 进化预算**：max_wall_time（默认 24h vs 缩短）与 max_iterations（0 不限 vs 设上限）。基准事实：无 token 成本/迭代先例（Gaps）；建议首小时观察再定。
- **J6 模型档位**：agent_models 全默认（glm-5.3 high thinking）vs builder 降档（token 成本 vs 产出质量；job_47 实测 builder 占 token 60%+）。
- **J7 干预预案**：何时人工干预（env_failure_concluded 连发 / 长期零入格 / token 成本异常的阈值）。基准事实：进化期间避免并发重负载（协处理器计时噪声）。

### 人类裁决记录 · 第 1 轮（2026-10-20）

- **Q1/J1 → (a)**：vanilla 全链首跑。达标判据 = KR4 原文（门先证红过 + running + ≥1 精英入格 + 干预事件可见 + 干净停止）+ 收尾报告落盘 env_7 工作区。不设 score 硬线；>1.0 仅作 stretch 观察项。
- **Q2/J2 → (a)**：512×512 float32 照模板。
- **Q3/J3 → 照模板定容差 1e-3**（正确性门 64×64 seed42 / bench K=3 / timeout 60s 均照模板）；**exploit 每方向 10 个样本**（4 方向 × 10 = 40 个，构造后逐个验证拒识才入注册表）。金样例：human 未单独否决推荐——按默认 reference.py + 1 个 row-dot 已知好解（构建时逐个验证过门）。
- **Q4/J4 → 悬置待 research**：human 要求先深调研 sub-kernel 优化可用 idea（论文级）再定特征空间——已派 research-L1-r2（L4 事实回流）。
- **Q5/J5 → max_iterations=10**（迭代 10 次即收）；墙钟用默认 24h 兑底（10 次分钟级迭代远达不到）。
- **Q6/J6 → (a)**：agent_models 全默认（glm-5.3 thinking high），不降档。
- **Q7/J7 → 干预纯 human 主导**：不预设自动干预触发器；中控只监控+报告，任何干预（pause/weight/stop 等）仅在 human 明示时执行。附加自律：进化运行期间本会话不跑重负载基准（SME 计时噪声 ±3% 敏感）。

### 事实回流（第 1 层 · L4，裁决后新事实）

**新事实 1（代码级，自查）**：模板门禁 verify 级联存在两个高级 exploit 可穿越的通道——
1. **篡改计时类泄漏**：候选 import 时 monkeypatch `time.perf_counter`，bench.py 无检测机制、stage3 无 score 合理性上界（软门禁对新格 exempt）→ 保持正确性的伪造计时候选会泄漏通过。
2. **规模条件捷径类泄漏**：正确性门只测 64×64 → 「n≤64 正确 / n>64 垃圾」候选过正确性门 + 秒过 bench → 泄漏。
→ 与 Q3 裁决（每方向 10 个诚实样本）联合推论：**必须 env 级门禁加固**（① bench.py 篡改检测：保存 time.perf_counter 引用快照 + 身份比对，篡改即拒；② 正确性门多规模校验：64 + 512 双规模，堵规模条件捷径），否则门先证红对 40 个诚实样本必然失败 → 启动中止。**门禁加固属 human 决策点，待 research-L1-r2 的防御实践调研回来后一并确认。**

**新事实 2（human 裁决打开的研究问题）**：sub-kernel 优化 idea 论文调研 → 特征空间轴设计（J4 悬置）。

已派 research-L1-r2，调研清单：
1. **（J4）sub-kernel/微内核 matmul 优化 idea**（论文+实践）：适用于 CPU float32 512×512、CPython 3.9 + numpy 2.0.2 + stdlib（multiprocessing/ctypes 可用，numba/cython/scipy 不可用——除非调研发现关键路径需补依赖，列为 env 依赖决策）——算法结构（Strassen/Winograd、分块 tiling、寄存器分块、循环序、packing）、内存布局、并行度（多进程多簇 SME ~2-4× 理论余量）、派发目标（ctypes 直调 Accelerate cblas_sgemm 等）、精度技巧（容差 1e-3 内）；映射为正交特征轴（2-4 轴）提案。
2. **（J3 加固）进化/自动调优系统防作弊实践**：AlphaEvolve/OpenEvolve/FunSearch/KernelBench 等如何防计时篡改/规模条件作弊/预存输出/精度作弊——多规模正确性、held-out 输入、计时快照引用、进程级墙钟交叉验证、score 合理性界等实践。
3. **（J3 加固实证）本机微实验**（A 级）：float16 在 n=64/512 的 max_error（拒识确认）；float32 BLAS 与 naive（float64 累加）vs float64 锚在 n=64/512 的误差（多规模正确性门在容差 1e-3 下的可行性）。

**research-L1-r2 已完成**——简报 `research-L1-r2.md`（20.9KB，11 组 A 级实测 + 一手文档调研）。核心结论：

**（J4 特征空间）**：
1. **idea→轴映射表**（简报 §1.1）：15 簇 idea 全谱映射；模板 2 轴装不下进化 agent 天然产出的 multiprocessing/线程/派发入口变体（会挤进邻近格污染 archive）。
2. **4 轴提案**：`impl_strategy[naive_loops, blocked, rowwise_vectorized, full_blas_dispatch, strassen_like](5) × memory_layout[row_major, col_major, tiled](3) × parallelism[serial, threaded, multiprocess](3) × dispatch_target[np_matmul, np_dot, einsum_path, ctypes_accelerate](4)` = 180 格；OpenEvolve MAP-Elites 先例（可配置维度 × bins，默认 archive 100 格量级）支撑。
3. **正交性论证（须明示）**：四轴 code-form 正交（任一候选可无歧义落四值，MECE）但**性能近乎全序**（dispatch_target top 值收敛同一 cblas_sgemm；parallelism/strassen 实测严格劣化）——特征空间捕捉代码形态多样性（防早熟收敛）而非性能正交分解。precision 不入轴（门禁直接淘汰项）。
4. **score>1.0 死区（S1-S7 实测）**：multiprocessing（fork/spawn/持久池 score≤0.063）、线程池（0.81-0.94）、Strassen-1（1.6× 慢）、fp64（3.4× 慢）、out= 预分配（1.018 噪声内）全部关闭；根因=单次 BLAS 调用已多簇饱和（M4 Pro 第三方基准佐证：单线程即 2.00×@2 簇）；手写 SME 内核余量也仅 1.23× 且 CPython 不可达。**本机 512×512 CPython+numpy 约束下 score>1.0 无实测可达路径。**
5. **依赖确认**：无关键路径需补依赖（ctypes 纯 stdlib 已覆盖直调 BLAS，E4 位相同零加速；scipy 同源无增益）。

**（J3 门禁加固）**：
1. **通道清单（3 个）**：① monkeypatch time.perf_counter 伪造计时；② n≤64 正确/n>64 垃圾规模条件捷径；③ **新发现**「拖慢 ref 抬 score」（候选导入时留后台忙循环进程抢 CPU，ref 计时被拖 → score 虚抬）。
2. **实测支撑**：E1 fp16 双规模必拒（64: 9.2×容差 / 512: 69×容差，且慢 970× 无动机）；E2/E3 双规模门不误杀（fp32 余量 5.5×、naive 余量 5.4×，锚可保持 fp32 BLAS 协议）。
3. **12 项加固清单**（简报 §2.4，全部 env/data/ 脚本可实现不动框架）：① 计时引用快照（`from time import perf_counter as _perf`）；② subprocess 墙钟交叉验证（终验级）；③ score 上界拒收（>2.0 拒，1.0-2.0 标记 suspicious）；④ 计时下界拒收（<0.01ms 物理不可能）；⑤ 双规模正确性门（sizes=[64,512]）；⑥ 多 seed 复验（固定 42 + urandom 派生）；⑦ 同输入跑两遍一致性；⑧ ref 基线合理性带（[0.10,0.50]ms 本机校准）；⑨ 残留子进程清点（waitpid WNOHANG）；⑩ 候选源静态检查（AST/regex 拒 sys.modules/__main__ 等）；⑪ 打分逻辑不可见（env 文档不写 checker 细节）；⑫ 对抗回归 fixture（4 作弊形态做成回归测试）。
4. **先例**：AlphaEvolve 官方 reward hacking 手册（确定性评估 + AST 禁用原语 + 同输入两遍 + 耗时检查 + 打分逻辑隔离）；KernelBench 偏执原则（>2× 加速高度可疑 + 对抗 fixture 回归）+ IOI 子进程隔离 + Tripwire held-out 对抗输入。

### 人类裁决记录 · 第 2 轮（2026-10-20）

- **Q8/J4 → (a) 采纳 4 轴**：impl_strategy(5) × memory_layout(3) × parallelism(3) × dispatch_target(4) = 180 格；接受「code-form 正交、性能近乎全序」的明示。
- **Q9/J3 → 12 项加固全采纳**；子裁决①接受 `__main__` 反射残留（静态检查拒 sys.modules/__main__ 字样 + 40 exploit 样本含 __main__ 反射变体实测拒识兑底）；子裁决②慢候选超时视为自然淘汰。

### 终止判定（第 1 层 · L5，OKR 充分性自检）

- 模块 M1-M4 已命名、职责/输入/输出/边界清晰，pipeline 串行关系确认（M1→M2→M3→M4）✅
- 判断节点全部经 human 裁决：J1(a)/J2(a)/J3(容差 1e-3+40 样本+12 项加固)/J4(4 轴)/J5(10 迭代)/J6(默认档)/J7(human 主导) ✅
- **充分性**：KR1（本层完成）+ KR2（M2 叶子已可落实）+ KR3/KR4（M3 叶子已可落实）+ KR5（M4 叶子已可落实）——四模块叶子全部可支撑上层 KR ⟹ O 可达 ✅
- 达标 → 下钻叶子层（第 2 层）

---

## 第 2 层（叶子层）：四模块落实到代码/文件/工具/配置

> **本层全部消解，无判断节点**——叶子层的全部判断已在第 1 层经 human 两轮裁决完毕（J1-J7 + Q8/Q9），本层为纯落实（含 1 项事实性修正，见 dispatch_target）；无新增判断节点，无需 research 派发。

### M2 gen_env 叶子（KR2）

**关键代码实现**（文件路径 + 签名）：
- `env/data/test_correctness.py`（加固版）：`main()` 读 env.json `correctness.sizes=[64,512]`；每规模跑 seed 42（固定）+ 1 个 urandom 派生 seed（终验随机化）；同输入跑两遍比对一致性；源码静态检查（拒 `sys.modules`、`__main__`、对 `time` 模块属性赋值等模式）→ stdout 单行 JSON `{"passed": bool, "max_error": float, "sizes": [...], "seeds": [...]}`
- `env/data/bench.py`（加固版）：顶部 `from time import perf_counter as _perf` 引用快照；score > 2.0 → `passed:false`（上界拒收）；runtime_ms < 0.01 → 拒（物理下界）；ref_runtime_ms ∉ [0.10, 0.50] → 拒（ref 合理带，本机校准）；结束前 `os.waitpid(-1, WNOHANG)` 循环清点残留子进程 → 有存活即拒；stdout 单行 JSON 协议不变
- `env/code/reference.py` / `env/code/naive_start.py`：照模板拷贝（不改动）
- `env/skills/self_check.py`：照模板拷贝
- `env/domain_knowledge.md`：模板 + 增补「优化路径速查」（idea 阶梯：naive 2e-5 → rowwise_vectorized 0.15 → full_blas_dispatch ~1.0；parallelism/strassen 实测劣化数据）；**不写任何门禁检测细节**（⑪ 打分逻辑不可见）
- `env/env.json`：task=matmul, perf_desc=low, perf_type=runtime_ms, tolerance.correctness.max_error=1e-3, tolerance.soft_gate.relative_band=0.05, bench{size:512,k:3,timeout_s:60}, correctness{sizes:[64,512], timeout_s:30}
- `env/feature_space.json`：4 轴（见下）
- `gates/exploit_registry.json`：40 条 `{id, name, pattern, sample_path}`（4 方向 × 10）
- `gates/exploits/*.py`：40 个 exploit 样本脚本（逐个实测拒识后才入注册表）
  - 方向 1 预存结果×10：固定矩阵/首次调用缓存/硬编码 checksum/输入无关随机等变体
  - 方向 2 篡改计时×10：对称/非对称 monkeypatch perf_counter、冻结式、常量计时、`__main__` 反射变体等（静态检查 + 引用快照拒）
  - 方向 3 降精度×10：fp16 纯路径/fp16 存储变体/截断累加/量化等（误差超容差拒）
  - 方向 4 规模条件×10：n≤64 正确/n>64 垃圾、n==512 特判、象限计算、零填充等（双规模正确性拒）
- `gates/golden_examples/`：reference.py 副本 + rowdot.py（逐行 np.dot 已知好解）

**特征空间终案（J4 采纳 + 1 项事实性修正）**：
```
impl_strategy[naive_loops, blocked, rowwise_vectorized, full_blas_dispatch, strassen_like]  (5)
memory_layout[row_major, col_major, tiled]                                                  (3)
parallelism[serial, threaded, multiprocess]                                                 (3)
dispatch_target[pure_python, np_matmul, np_dot, einsum_path, ctypes_accelerate]             (5)
```
= 225 格。**修正说明（叶子层事实性修正，非判断节点变更）**：research 提案的 dispatch_target 只有 4 个 numpy/ctypes 值——但框架机制（loop.py `_run_builder`/`_run_coding` 直读：格由采样器定、coding agent 按 cell_descriptor 实现）要求**每个格的描述子都可被 agent 无歧义实现**；naive_loops/blocked 格无「纯 Python 手写」派发值可落，补 `pure_python` 使笛卡尔积语义完备（naive_loops|pure_python、blocked|pure_python 为自然起点格）。轴结构（4 轴）与其余取值遗照 human 裁决不变；225 vs 180 格对 10 迭代无实质影响（预计只填 3-10 格）。

**工具调用**：
- `cp -R super/env_templates/matmul <ws>/env`（模板起步）
- `python3 -m venv env/.venv && env/.venv/bin/pip install -r env/requirements.txt`
- 验收自检四项：`env/.venv/bin/python env/data/test_correctness.py --candidate env/code/reference.py`（过）→ `... bench.py --candidate env/code/reference.py`（出分）→ 40 exploit 逐一 `test_correctness.py --candidate gates/exploits/*.py`（全拒）→ `--candidate env/code/naive_start.py`（过）

**环境依赖**：numpy>=1.24（唯一第三方依赖；venv 隔离）；无 numba/cython/scipy（research 确认无关键路径需要）。

### M3 启动与运行监督叶子（KR3+KR4）

**关键代码实现**：无新代码——super.json 预写 + CLI 启动。

**文件结构/配置**：`<ws>/super.json` 启动前预写 config 段：
- `pi_path = "/Users/logic/workdir/rick-dev-home/.rick/pi/agent/runtime/node_modules/.bin/pi"`（空值 → 调度器 exit 4，已实锤；realpi_smoke 同法）
- `max_iterations = 10`（Q5 裁决）；`max_wall_time_h = 24`（默认兑底）；`agent_models = {}`（全默认 glm-5.3 high，Q6 裁决）；其余默认（adjust_interval=5 / debug_max_attempts=3 / soft_gate_k=3 / heartbeat 30s/90s）
- 注意：Go initSuperWorkspace 幂等保留已有 config（非 nil 不覆盖）——预写在 start 前生效

**工具调用**：
- 启动：`/Users/logic/workdir/rick-dev/bin/rick super start --name env_7 --env /Users/logic/workdir/rick-dev/.rick/super/env_7/env --daemon --max-iterations 10`（启动内含门先证红：40 exploit 全拒 + 2 金样例全过；失败即中止并出泄漏清单）
- 监控：`.../rick super status --name env_7 [--json]` + 直读 events.jsonl / archive.json / heartbeat.json / codes/
- 干预（仅 human 明示时）：`.../rick super event --name env_7 --type <pause|weight|resurrect|env_change|stop> --payload '<json>'`——写 interventions/ 队列，迭代边界摄取（延迟 ≤1 迭代），事件流可见 intervention(source=human)
- 10 迭代自然收尾：max_iterations 到点 → 循环 break → session_ended + state=stopped（loop.py run() finally 段确认）

**环境依赖**：pi 0.84.2（默认 catpaw-proxy/glm-5.3/thinking high）；调度器 stdlib-only Python 3.9+；ENV_FAIL_STREAK_LIMIT=3（连发 3 次环境失败调度器自停——监控关注）。

### M4 停止与收尾叶子（KR5）

**关键代码实现**：收尾报告 `final-report.md`（人写不是代码——中控会话产出）。

**文件结构**：`/Users/logic/workdir/rick-dev/.rick/super/env_7/final-report.md`

**工具调用**：
- 自然收尾（10 迭代）：确认 state=stopped + session_ended 事件；若 Go 监督者仍在（daemon 形态）跑 `rick super stop --name env_7` 兑底
- 人工提前停（human 明示）：`rick super stop --name env_7`（SIGTERM 落完当前事件 → grace → SIGKILL 整树）
- 验收四查：state=stopped ✓ session_ended 事件 ✓ 无孤儿进程（ps 检查 supervisor/py 树）✓ 事件流完整可审计 ✓

**收尾报告内容**：北极星终值（总迭代/入格数/各格最优 perf 与 score/token 分 role 成本）+ exploit 证红记录 + 干预记录 + 复盘（哪些格值得深挖、门禁升级建议、成本观察）。

---

## 全局终止声明

**Grilling 完成——设计树已遍历完毕，所有模块已落实到代码实现/文件结构/工具调用/环境配置。**

- 第 0 层 OKR（O + KR1-5 + 充分性自检）✅
- 第 1 层使命 pipeline（M1-M4 + L1 调研 + 两轮 human 裁决 + L5 终止判定）✅
- 第 2 层叶子层（M2/M3/M4 四维度全落实，含 1 项事实性修正记录）✅
- 每层 research 简报：L1 → research-L1.md / research-L1-r2.md ✅

---
