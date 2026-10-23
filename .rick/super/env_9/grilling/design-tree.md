# env_9 设计树（grilling 活文档）

> 使命锚点：`requirement.md` = 「8192 * 8192 矩阵乘法优化」
> 协议：逐层追加；每层 = 模块 + pipeline + 判断节点；L1 调研记录随层落盘。
> 前作：env_7（512×512，链路验证，性能无收益）/ env_8（2048×2048，**成功**：26 迭代收敛 best score 0.9917 @ 6.86ms，BLAS 平价达成，12 格特征空间 + 加固门禁 40 exploit 全拒 + rowwise 起点 + M 分解系容差事实全部验证有效）。
> 本次为同族使命第 9 次尝试：规模升至 8192（FLOPs = env_8 的 64×，单矩阵 256MB，工作集 ~768MB——**内存墙时代**，新物理区间）。

---

## 第 0 层（根）：OKR

O（Objective）：交付 env_9「8192×8192 矩阵乘法优化」super 进化使命的完整闭环，并在 8192 规模上取得**可验证的真实性能产出**——env_8 已证明链路在 2048 可收敛到 BLAS 平价（0.99），本次主体是验证/复现该结论在**内存受限大矩阵**区间是否成立，并探索 8192 特有的合法性能杠杆（输出缓冲复用等 allocation-avoidance 族，初步实测 score ~1.04 可达——待 research-L1 A 级确认）。

KR 集（演绎：KR1 ∧ … ∧ KR5 ⟹ O）：
- KR1 需求理清：design-tree.md 落盘且过 grilling_gate——8192 特有判断节点（容差边际 @8192 / 门禁成本与超时重校准 / 特征空间是否引入 output-buffer 轴 / ref_runtime_band 重校准 / 预算与账号）全部经 human 裁决。
- KR2 env 合格：以 env_8 加固版 env 为基底做 8192 适配（bench size / correctness sizes / ref_runtime_band / D4 exploit 规模适配重验 40/40），验收自检四项全过。
- KR3 启动成功：super.json 预写（pi_path 显式）→ `rick super start` 门先证红 100% → state=running + 迭代递增。
- KR4 进化产出（本次主体）：性能判据达成（J1 裁决，初拟 best score ≥0.9 为成功——8192 是否把判据上调到 >1.0 待 research 后定）+ 北极星可读 + 事件流完整；若判据未达成，须有可归因的失败分析。
- KR5 干净收尾：state=stopped + 无孤儿进程（含 env_7 已知 stop 不收监督者树缺陷的手动补刀）+ 收尾报告落盘 env_9 工作区（env_8 欠了收尾报告——本次必须补上）。

**OKR 充分性自检**：KR1 前提；KR2 硬前置（8192 门禁校准错误 = 进化空转或放水）；KR3 打通启动；KR4 使命主体；KR5 含 env_8 的遗留教训（报告补齐）。五者联合 ⟹ O。✅（待 J1 裁决后复核 KR4 判据充分性）

---

## 第 1 层：使命 pipeline（四大模块）

```
[M1 grilling 需求追问] → [M2 gen_env 环境建设（env_8 基底 8192 适配）] → [M3 启动与运行监督] → [M4 停止与收尾报告]
     （现在）                （KR2）                                  （KR3+KR4）            （KR5）
```

| 模块 | 职责 | 输入 | 输出 | 边界 |
|---|---|---|---|---|
| M1 grilling | 设计树追问循环，8192 特有判断节点全部经 human 裁决 | requirement + env_8 遗产（加固 env / 12 格特征空间 / 0.99 收敛证据 / builder prompt 进化版）+ 本机 8192 校准 | design-tree.md（过 grilling_gate） | 不建设 env、不启动进化 |
| M2 gen_env | env_8 加固 env 基底 → 8192 适配（规模/超时/ref 带/注册表回归）+ venv | KR1 裁决 + env_8 env | `env_9/env/` + `env_9/gates/`，验收自检四项全过 | 门禁阈值属 human 决策 |
| M3 启动与运行监督 | super.json 预写 → start（门先证红）→ status 监控 → event 干预（仅 human 明示） | KR2 的 env + KR1 的预算/特征空间裁决 | events.jsonl + archive 精英 + 北极星 | 中控不执行进化逻辑 |
| M4 停止与收尾 | stop + 监督者树补刀 + 收尾报告 | M3 运行痕迹 | state=stopped + 报告落盘 | 报告含 KR4 判据对照与失败归因 |

**层内 pipeline 依赖**：M1 → M2 → M3 → M4 严格串行；M3 的预算/特征空间/模型档位在 M1 裁决（直接影响 start 参数）。

### 调研记录（第 1 层 · L1 轻量自查消解，信源等级 A：直读/单点实测）

1. **本机**：Apple M5 Max / 64GB RAM / 18 核（6P+12E）；当前 load ~4.5（中等背景负载——校准测量须多进程多次取中位，运行期纪律「不跑重负载基准」在 8192 更关键）。**[A]**
2. **pi runtime**：pi_path 符号链接在位（node_modules/.bin/pi → cli.js）；env_8 venv numpy 2.0.2（python3.9，Accelerate/vecLib 后端——env_8 已实证 SME 单线程饱和物理）。**[A]**
3. **单点初测 @8192**（grep 排噪，单次非协议测量）：np.matmul 首调 605ms（含 256MB 分配+缺页）/ 稳态 471ms / **out= 预分配复用 452ms（bitwise 一致 ✓）**——「分配规避族」合法 score ~1.04 的初步信号（vs env_8@2048 分配开销可忽略、全族 ≤1.0）。**[A-]**
4. **env_8 遗产直读**：26 迭代 / best 0.9917（np_matmul|full_blas_dispatch 6.86ms）/ 12 格 2 轴特征空间收敛有效 / 40 exploit 全拒 / 门禁脚本 12 项加固全保留 / env_8 侧欠收尾报告（KR5 补课）+ auto-adjudicate.sh 守护脚本可复用（expand_cell 免审批裁决，human 裁决下放内置保守规则）。**[A]**
5. **框架语义**（承 env_8 调研，不变项）：无父本时拷 env/code/ 全部进迭代工作区；超时双层（env.json 内部 alarm + 框架子进程 stage2 120s / stage3 600s 缺省）。**[A]**

### research-L1 核心结论（简报已落盘 `research-L1.md`，116 行；脚本与原始数据留存 /tmp/mm9/：10 脚本 + 7 结果文件）

1. **基线族 @8192**：BLAS 族坍缩到 434–488ms 窄带，np.matmul fresh 440ms = 天花板（score 1.00）；rowwise 9.77s（0.045，IQR 极稳的内存带宽型）；m_quarters 0.93 / m_halves 0.96–0.97 / einsum_opt·dot2d 0.99–1.00；ctypes 657ms（0.67）。**[A]**
2. **容差 @8192 更绝对**：锚自身 vs fp64 = 1.42e-2（14× 容差）——「正确」只能=与 f32 BLAS 锚逐位一致；合法族仅 M 分解系 + 直调系（双 seed 双进程全逐位 0.0）；K 序改变系超容差 14–29×（Strassen 2.5–2.9e-2 性能已平但数值单杀）；fp16 双杀（49× 容差 + 无 BLAS 路径慢 833–1234×，@8192 外推 ≥366s 必超时）；int8 930× / bf16 11,353×。**[A]**
3. **分配规避假设证伪**：256MB 输出首触+零填实测仅 2.05ms（0.47% 理论天花板）+ malloc 跨调用回收（地址复用实测 true）→ bench 稳态不重复付缺页成本；交错配对（12 对×2 进程）增益 0–2% 在噪声内、符号不稳定——**不建议 output_buffer 轴**；中控预探针 1.04 为顺序效应不可复现。**[A]**
4. **门禁成本**：correctness [64,8192]×2seed×双跑：rowwise 级 ≈41.5s / BLAS 级 ≈4s；bench 全协议：rowwise 级 ≈41.6s / BLAS 级 ≈4.7s。单调用 alarm 30s/60s 余量 3.1×/6.1× 维持；**stage2 框架超时 120s 余量仅 2.9×（合法慢中间态可到 103–115s）→ 建议 180s**（env.json `gates.correctness_timeout_s` 配置，无需改框架）；stage3 600s 维持。**[A]**
5. **ref_runtime_band 建议 [380,700]ms**（跨进程 fresh 中位 434–484，单轮 427.5–600.5 含负载尖峰；下界 −11% 防背景负载消退误杀）。**[A]**
6. **新发现：fork-after-BLAS 死锁**：父进程用 BLAS 后 fork 常驻池 → 子进程挂死（>7min 实测）——门 harness 先算 expected 再调候选，任何 fork 池候选必挂死至 alarm 死 + 门⑨残留子进程拒收 → mp 族三重死（性能 0.84 + 挂死 + 门⑨），写进 domain_knowledge。**[A 观察]**
7. **确定性**：全族（fresh×7 / out= / 页对齐变体 / rowwise 头尾 / ctypes / mp 半块）逐位一致——加固⑦双跑门对合法候选安全。**[A]**
8. **无 >1.0 合法路径**（全族 A 级扫描 + 分配规避证伪）→ KR4 判据按 ≥0.9 设定与 env_8 同构；协议内预期 best 0.96–1.00。**[A]**

### 判断节点（第 1 层 · L2 终版——并 research 结论，待 human 裁决 → L3 批量追问）

- **J1 达标判据（KR4）**：A 级已证无可复现 >1.0 合法路径（分配规避证伪、全族坍缩窄带）⟹ >1.0 目标必败。候选：(a) best score ≥0.9 为成功、≥0.5 为部分成功、<0.5 须失败归因（与 env_8 同构；协议内预期 0.96–1.00）；(b) ≥0.5 即成功；(c) 不设阈值纯 best-effort。(a) 的达标需采样到达 BLAS 族——12 格空间 10-15 迭代内必达（env_8 实证 26 迭代 best 0.9917）。
- **J2 任务规格**：8192×8192 **float32**、uniform 协议、score=ref/cand vs np.matmul f32 锚——确认或改规格（requirement 未指明 dtype；env_7/8 先例 f32）。
- **J3 容差**：(a) 保持 1e-3——正确性 ≡ BLAS K 求和序一致（@8192 更绝对：合法族仅 M 分解+直调系，双 seed 全逐位；判定干净零假阳性）；(b) 放宽到 5e-2——blocked-K/Strassen 数值复活（性能 0.90–0.98 已平但无更好性能，只是「合法但不再更优」格，且 fp16 4.9e-2 逼近边界需重验 exploit）。
- **J4 naive 起点**：(a) rowwise 无转置 np.dot（9.77s/调，score 0.045，逐位过门，门成本 41.5s 可客，22× 攀升空间）——推荐；(b) 纯 Python（@8192 完全天文数字，首迭代必死）；(c) m_quarters（0.93，距顶太近无进化空间）。
- **J5 特征空间**：research 证伪 output_buffer 轴（增益 0–2% 噪声内）。候选：(a) 照搬 env_8 2 轴 12 格（impl_strategy[rowwise/m_halves/m_quarters/full] × dispatch_target[np_dot/np_matmul/einsum_path]，全部 @8192 实测合法且 score 阶梯 0.045–1.00）——推荐；(b) 加 output_buffer 轴（已证伪）；(c) 收缩到 1 轴（丢失 code-form 多样性防早熟能力）。
- **J6 门禁参数**：correctness alarm 30s / bench alarm 60s 维持（余量 3.1×/6.1×）；**stage2 框架超时 120→180s**（env.json gates.correctness_timeout_s，防「合法慢中间态」被框架杀而非门拒）；stage3 600s 维持；**ref_runtime_band [380,700]ms**（research 建议，下界防背景负载消退误杀）；score_max 2.0 / runtime_ms_min 0.01 / K=3+warmup1 / 双 seed+双跑+静态检查 全维持。
- **J7 预算**：env_8 实测 ~0.9M token/迭代、~15 分钟/迭代、~$0.52/迭代；env_9 门禁更重（rowwise 级 41.5s）但 agent 回合主导，预计持平或略升。候选：(a) max_iterations=15 + 墙钟 8h 兑底 + 迭代 3 后 token 观测点——推荐；(b) max_iterations=12（env_8 原设，后中途升到 36）；(c) ≥30 长跑（token 成本 30M+）。
- **J8 记账账号**：env_8 在同事（zwm）日额度上跑。候选：(a) 继续当前账号（需确认额度充足）；(b) 切换账号（mcopilot-account-switch skill）；(c) human 指定。
- **J9 模型档位**：(a) 全默认 glm-5.3 thinking high（env_8 先例、链路已验证）——推荐；(b) builder 降档轻模型（token 减半但计划质量风险）。
- **J10 干预与收尾预案**：承 env_7/8 已知缺陷规避（stop 后手动收监督者树；North Star token 从 agent_ended.usage 聚合；运行期不跑重负载基准——@8192 score 噪声 ±2–5% 对背景负载更敏感）；env_8 遗产继承：auto-adjudicate.sh 守护（expand_cell 免审批）+ builder.md env 级 prompt（8192 资产路径适配后继承）；M2 验收自检补端到端真门直测 1 次 rowwise 级（关闭 §3 外推 gap）。

### 人类裁决记录 · 第 1 轮（2026-10-22 23:3x，「全部按推荐」）

- **Q1/J1 → (a)**：成功 = best score ≥0.9（协议内预期 0.96–1.00）；部分成功 = ≥0.5；<0.5 须失败归因。不设 >1.0 目标（A 级已证不可达，仅作观察项）。
- **Q2/J2 → 确认**：8192×8192 **float32**、uniform 协议、score=ref/cand vs np.matmul f32 锚。
- **Q3/J3 → (a)**：容差保持 **1e-3**（正确性 ≡ BLAS K 求和序一致）；domain_knowledge 写明 @8192 容差事实（锚噪声 1.42e-2、K 序系 14–29×、合法族=M 分解+直调）；特征空间不含已知死族。
- **Q4/J4 → (a)**：naive 起点 = **rowwise 无转置 np.dot**（9.77s/调 / score 0.045 / 逐位一致 / 门成本 41.5s）。
- **Q5/J5 → (a)**：特征空间照搬 env_8 **2 轴 12 格**（impl_strategy[rowwise_vectorized/m_halves/m_quarters/full_blas_dispatch] × dispatch_target[np_dot/np_matmul/einsum_path]）；**不加 output_buffer 轴**（已证伪）。
- **Q6/J6 → 按推荐**：alarm 30s/60s 维持；**stage2 框架超时 120→180s**（env.json gates.correctness_timeout_s=180）；**ref_runtime_band [380,700]ms**；score_max 2.0 / runtime_ms_min 0.01 / K=3+warmup1 / 双 seed+双跑+静态检查全维持。
- **Q7/J7 → (a)**：**max_iterations=15**，墙钟 **8h** 兑底，迭代 3 后 token 观测点；异常即停机请示。
- **Q8/J8 → 默认现状**：继续当前 catpaw-proxy 账号（human 未另指定——额度风险由迭代 3 观测点监控，异常即停机请示；需切换随时走 mcopilot-account-switch）。
- **Q9/J9 → (a)**：agent_models 全默认 glm-5.3 thinking high。
- **Q10/J10 → 按推荐**：承袭全部已知缺陷规避 + env_8 遗产继承（auto-adjudicate.sh / builder.md 8192 适配）+ M2 验收自检补端到端真门直测 1 次。

### 终止判定（第 1 层 · L5，OKR 充分性自检）

- 模块 M1-M4 已命名、职责/输入/输出/边界清晰，pipeline M1→M2→M3→M4 串行确认 ✅
- 判断节点 J1-J10 全部经 human 裁决（全推荐案），无吞节点 ✅
- **充分性**：KR1（本层完成）+ KR2（M2 叶子可落实：env_8 基底全部实测校准值就绪：ref band [380,700] / stage2 180s / sizes [64,8192] / 死族清单）+ KR3（M3 叶子可落实：pi_path/max_iterations=15/8h 已定）+ KR4（判据 0.9/0.5/归因已定；12 格空间 score 阶梯 0.045–1.00 全合法，采样必达 BLAS 族——env_8 同构实证）+ KR5（M4 叶子可落实：监督者树补刀 + usage 聚合 + 报告落盘）⟹ O 可达 ✅
- 达标 → 下钻第 2 层叶子

---

## 第 2 层（叶子层）：四模块落实到代码/文件/工具/配置

> **本层全部消解，无判断节点**——叶子层的全部判断已在第 1 层经 human 一轮裁决完毕（J1-J10 全推荐案），本层为纯落实；无新增判断节点，无需 research 派发。

### M2 gen_env 叶子（KR2）——env_8 加固基底 × 8192 适配

**关键代码实现**（文件 + 要点）：
- `env/data/test_correctness.py` / `env/data/bench.py`：**照抄 env_8 加固版不改**（12 项加固全保留；规模/带/容差/超时全部从 env.json 读——含新键 gates.correctness_timeout_s）
- `env/env.json`：bench{size:8192, dtype:float32, warmup:1, k:3, timeout_s:60}；correctness{sizes:[64,8192], timeout_s:30, extra_seeds:1, double_run_consistency:true}；tolerance{correctness.max_error:1e-3, soft_gate.relative_band:0.05}；gates{score_max:2.0, runtime_ms_min:0.01, **ref_runtime_band_ms:[380,700]**, static_check:true, **correctness_timeout_s:180**, bench_timeout_s:600}；description 同步 8192
- `env/feature_space.json`：照搬 env_8 2 轴 12 格；orthogonality/desc 更新 @8192 实测阶梯（rowwise 0.045 → m_quarters 0.93 → m_halves 0.96 → 直调 0.99–1.00）与死族剔除依据（mp 三重死+fork 死锁 / blocked-K·Strassen 数值死 14–29× / ctypes 0.67 / 低精度双杀 / 分配规避已证伪不入轴）
- `env/code/reference.py`：np.matmul（锚，不变；main 自检改 n=8192）
- `env/code/naive_start.py`：**rowwise 无转置 np.dot**（Q4 裁决；逐位一致必过门；不带 B 转置——预转置 gemv 1.39e-2 被容差拒）
- `env/domain_knowledge.md`：8192 重写——score 阶梯 + 容差紧约束事实（锚噪声 1.42e-2、K 序改变系 14–29×、合法族=M 分解+直调）+ 死族清单（mp 三重死含 **fork-after-BLAS 死锁警告**、Strassen/blocked-K 数值单杀、ctypes 0.67、fp16/int8/bf16、f64 对称死）+ 分配规避证伪结论（out= 合法形态但无增益）+ 分析/调优 SOP；**不写门禁检测机制细节**
- `env/skills/self_check.py` / `env/skills/perf_profile.py`：照抄（默认规模参数检查适配 8192）
- `env/prompts/builder.md`：**继承 env_8 env 级进化版**，「可用工具」节的 env 资产绝对路径全部改为 env_9 路径
- `env/requirements.txt`：照抄（numpy>=1.24）
- `gates/exploit_registry.json` + `gates/exploits/*.py`：40 样本从 env_8 生成器再生成（gen_exploits.py 适配 sizes=[64,8192]，D4 规模条件型 n==8192 变体）；**verify_exploits.py 逐一实测 40/40 拒识后才入注册表**（D3 量化型 @8192 误差更大更易拒；若个别泄漏→修样本）
- `gates/golden_examples/`：reference.py + rowwise.py（=naive_start）+ m_halves.py（0.96 已知好解，风格多样）
- `env/.venv/`：python3 -m venv + pip install -r requirements.txt（numpy 2.0.x）

**工具调用**：cp -R env_8/env → env_9/env（改 env.json/feature_space/domain_knowledge/builder.md/reference.py）· venv 安装 · 验收自检（见下）· verify_exploits 40/40

**验收自检清单（含 J10 补充项）**：
1. reference 过正确性门（双 seed 双跑）
2. bench 出分（ref 中位落 [380,700] 带）
3. 40 exploit 全拒（拒识 100%）
4. naive（rowwise）过正确性门
5. 金样例 3/3 过门（reference / rowwise / m_halves）
6. **端到端真门直测 1 次 rowwise 级**（关闭 research §3 外推 gap：test_correctness.py 全协议实测总时长 ≈41.5s < 180s stage2 预算）

### M3 启动与运行监督叶子（KR3+KR4）

**配置**：super.json 预写 config 段——pi_path=/Users/logic/workdir/rick-dev-home/.rick/pi/agent/runtime/node_modules/.bin/pi；max_iterations=15；max_wall_time_h=8；agent_models={}；其余默认（adjust_interval=5 / debug_max_attempts=3 / soft_gate_k=3 / heartbeat 30s/90s）。预写须在 start 前（Go initSuperWorkspace 幂等保留已有 config）。

**工具调用**：`/Users/logic/workdir/rick-dev/bin/rick super start --name env_9 --env /Users/logic/workdir/rick-dev/.rick/super/env_9/env --daemon --max-iterations 15 --max-wall-time 8`（含门先证红：40/40 拒 + 金样例零误杀，失败即中止出清单）→ status --json 监控 → **迭代 3 后 token 观测点**（J7/J8 裁决：烧太快即停机请示）→ 干预仅 human 明示。

**auto-adjudicate 守护**（J10 继承）：cp env_8/auto-adjudicate.sh → env_9/ → start 后 `nohup bash auto-adjudicate.sh <ws> > auto-adjudicate.log 2>&1 &`（expand_cell 提案免审批，内置保守规则裁决）。

### M4 停止与收尾叶子（KR5）

`rick super stop --name env_9` → **手动收监督者树**（ps 找 PySupervisor/调度器进程组 kill，env_7 已知缺陷）→ 确认 state=stopped + 无孤儿 → 收尾报告 `env_9/final-report.md`（**env_8 欠的收尾报告本次必须补**：北极星终值从 agent_ended.usage 聚合 / KR4 判据对照（best score vs 0.9/0.5）/ 门先证红记录 / token·成本总账 / 复盘建议）。

---

## 全局终止声明

**Grilling 完成——设计树已遍历完毕，所有模块已落实到代码实现/文件结构/工具调用/环境配置。**

- 第 0 层 OKR（O + KR1-5 + 充分性）✅
- 第 1 层（M1-M4 + research-L1 简报 116 行 + 10 判断节点一轮裁决 + L5）✅
- 第 2 层叶子（M2/M3/M4 四维度全落实）✅
