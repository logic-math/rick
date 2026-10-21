# env_8 设计树（grilling 活文档）

> 使命锚点：`requirement.md` = 「实现一个 2048 * 2048 的大矩阵乘法性能优化」
> 协议：逐层追加；每层 = 模块 + pipeline + 判断节点；L1 调研记录随层落盘。
> 前作：env_7（512×512，2026-10-20）——全链首跑成功（4.5 迭代、3 入格、门先证红 40/40、干净度尚可），但**性能无收益**（最高 score 0.0072，距 BLAS 平价差 140×）。本次为同族使命第 8 次尝试：规模升至 2048，且核心增量从「验证链路」转为「真出性能」。

---

## 第 0 层（根）：OKR

**O（Objective）**：交付 env_8「2048×2048 大矩阵乘法性能优化」super 进化使命的完整闭环，并在 2048 规模上取得**可验证的真实性能产出**——链路正确性已由 env_7 验证（不再重复验证 vanilla 链路），本次的使命主体是让进化搜索在合理预算内**收敛到高性能解**（性能判据待 J1 裁决）。

**KR 集（演绎：KR1 ∧ … ∧ KR5 ⟹ O）**：
- **KR1 需求理清**：design-tree.md 落盘且过 grilling_gate——2048 特有判断节点（naive 起点可行性 / 门禁重校准 / 特征空间 vs 预算错配 / token 记账在同事账号）全部经 human 裁决。
- **KR2 env 合格**：以 env_7 加固版 env 为基底做 2048 适配（bench size / correctness sizes / ref_runtime_band 重校准 + exploit 注册表 2048 回归），验收自检四项全过。
- **KR3 启动成功**：super.json 预写（pi_path 显式）→ `rick super start` 门先证红 100% → state=running + 迭代递增。
- **KR4 进化产出（本次主体）**：性能判据达成（J1 裁决，如 best score ≥ 阈值）+ 北极星可读 + 事件流完整；若判据未达成，须有可归因的失败分析（哪个环节吃掉了预算）。
- **KR5 干净收尾**：state=stopped + 无孤儿进程（**含 env_7 发现的 stop 不收监督者树缺陷的手动补刀**）+ 收尾报告落盘 env_8 工作区。

**OKR 充分性自检**：KR1 是前提；KR2 是硬前置（2048 门禁校准错误 = 进化空转或放水）；KR3 打通启动；KR4 是本次使命主体（区别于 env_7 的定位）；KR5 含 env_7 遗留缺陷的已知规避。五者联合 ⟹ O。✅（待 J1 裁决后复核 KR4 判据的充分性）

---

## 第 1 层：使命 pipeline（四大模块）

```
[M1 grilling 需求追问] → [M2 gen_env 环境建设（env_7 基底 2048 适配）] → [M3 启动与运行监督] → [M4 停止与收尾报告]
     （现在）                （KR2）                                  （KR3+KR4）            （KR5）
```

| 模块 | 职责 | 输入 | 输出 | 边界 |
|---|---|---|---|---|
| M1 grilling | 设计树追问循环，2048 特有判断节点全部经 human 裁决 | requirement + env_7 遗产（收尾报告/加固 env/实测数据）+ 本机 2048 校准 | design-tree.md（过 grilling_gate） | 不建设 env、不启动进化 |
| M2 gen_env | env_7 加固 env 基底 → 2048 适配（规模/超时/带/注册表回归）+ venv | KR1 裁决 + env_7/env | `env_8/env/` + `env_8/gates/`，验收自检四项全过 | 门禁阈值属 human 决策 |
| M3 启动与运行监督 | super.json 预写 → start（门先证红）→ status 监控 → event 干预（仅 human 明示） | KR2 的 env + KR1 的预算/特征空间裁决 | events.jsonl + archive 精英 + 北极星 | 中控不执行进化逻辑 |
| M4 停止与收尾 | stop + 监督者树补刀 + 收尾报告 | M3 运行痕迹 | state=stopped + 报告落盘 | 报告含 KR4 判据对照与失败归因 |

**层内 pipeline 依赖**：M1 → M2 → M3 → M4 严格串行；M3 的预算/特征空间/模型档位在 M1 裁决（直接影响 start 参数）。

### 调研记录（第 1 层 · L1 自查消解，信源等级 A：仓库/运行时直读）

1. **env_7 遗产与教训**（收尾报告直读）：
   - 全链真实跑通（builder/coding/debug/adjust 四角色、门先证红 40/40、双规模不误杀、两遍一致性抓缓存型 exploit）。
   - **性能无收益根因 = 特征空间 × 预算错配**：225 格分殖优先，10 迭代内每格只首访（gen 全 =1），UCB 利用/同格改进不发生；采到的格多数结构性慢（einsum/multiprocess/pure_python 系）；**BLAS 族（唯一能出 score~1.0 的 20+ 格）零采样**。
   - 后续跑法建议（报告原文）：max_iterations ≥ 30 或先 weight 收缩采样域。
   - token 实测：~9M/4.5 迭代（builder 4.4M / debug 2.75M / coding 2.2M，含缓存）；debug 占 31% 的教训（结构性不可达修复烧满 attempts）。
   - 框架缺陷三条：①`rick super stop` 不收监督者树（stop 后 OnDead 重启新调度器）→ 收尾需手动 kill 监督者进程组；②SIGKILL 路径无 session_ended；③North Star total_tokens 恒 0（真值从 agent_ended.usage 聚合）。
2. **框架语义**（super/superopt 直读）：
   - 无父本时 `_seed_parent_code` 拷 **env/code/ 全部文件**（reference.py + naive_start.py 都进迭代工作区）——coding agent 能看到参考实现。
   - 超时双层：env.json 内部 alarm（bench timeout_s 60 / correctness timeout_s 30）+ 框架子进程超时（stage2 120s / stage3 600s 缺省）。
   - agent_models 可 per-role 降档（config.py，默认 {}）。
3. **2048 规模核心风险（env_8 特有）**：纯 Python naive 512 实测 4.15s（独立）/ 正确性门 512 规模 ~8.4s → O(n³) 外推 2048 ≈ **265~537s/次** → 正确性门（30s alarm + 120s stage2）与 bench（60s alarm + 600s stage3）**双必超时**。⇒ 纯 Python naive 起点在 2048 不可行，「慢是特性」的设计前提在 2048 破产——起点方案是 J-节点（rowwise np.dot 起点实测 512 时 0.153 score，2048 待测）。
4. **机器/环境**：Apple M5 Max / 64GB / 18 核（6P+12E）；numpy 2.0.2 venv 可直接复用 `env_7/env/.venv`（research 与 M2 均用）；vecLib FP 噪声（sgemm 触发）已知且脚本已 seterr 抑制。
5. **token 记账现状**：dev HOME 已切 **zwm**（同事）token（config.yaml + auth.json 双写，探针 ok）——本次进化全部消耗记在 zwm 日额度；额度按天重置；量级未知。**预算裁决（J6）须把这个记账号事实纳入**。
6. **可执行**：`/Users/logic/workdir/rick-dev/bin/rick`（arm64，10-21 15:22 构建）；pi_path = `/Users/logic/workdir/rick-dev-home/.rick/pi/agent/runtime/node_modules/.bin/pi`（env_7 同款）。
7. **exploit 注册表迁移性**：env_7 的 40 样本模式大多规模无关（D4 规模条件型用 n==64/n≤64 条件——在 sizes=[64,2048] 双规模门下仍被 2048 侧拒识）；需重跑 verify_exploits 回归确认 2048 下 40/40 拒识（D3 量化型误差随规模变化是回归重点）。

### research-L1 派发（重量级，本机 2048 校准 + 外部校准）——**进行中**

调研清单：
1. **本机 2048 基线族**（A 级实测，env_7 venv）：np.matmul 中位数+稳定性（ref band 校准）/ rowwise np.dot / blocked-tiled numpy（块 128/256/512）/ 纯 Python 512+1024 实测外推 2048 / Strassen-1（叶 1024）/ multiprocessing 2/4 进程 / threaded / einsum / ctypes cblas_sgemm 直调。
2. **容差余量**：诚实 fp32 实现在 2048 的 max_error（vs np.matmul fp32 锚）——1e-3 容差在 2048 的安全边际。
3. **门禁成本**：correctness sizes=[64,2048]（2 seeds + double-run）对 rowwise/blocked 候选的耗时；bench 全协议（warmup+K3+ref）时长。
4. **外部校准**（B 级）：Apple Silicon 大矩阵 sgemm 单调用多簇饱和行为；Strassen 交叉点（BLAS 叶节点）在 2048 附近的经验；手工 cache-blocking vs BLAS 内建 tiling 在 16MB 操作数规模的真实余量。

### research-L1 核心结论（简报已落盘 `research-L1.md`，17/17 项完成，13 个复现脚本留存 /tmp/mm8/）

1. **基线**：ref=np.matmul@2048 中位 **6.88ms**（进程内 IQR 仅 0.06；跨进程 6.7-8.0，±10% 散布含背景负载）；协议内 BLAS 平价 score 实测 0.98。**[A]**
2. **天花板确证（全族扫描无 >1.0 路径）**：blocked-K 10.6-24.7ms（score 0.28-0.65）/ Strassen-1 11.3ms（0.61）/ mp 多进程 ≤0.65（**fork 纯开销 6.8ms ≈ 整个 matmul**）/ threaded ≤0.92 / ctypes 10.0ms（0.69）/ einsum_opt 7.9ms（0.87）/ row_halves 7.6ms（0.90）。外部佐证：**Accelerate 大 GEMM 由共享 SME 协处理器单线程饱和**（AppleAccelerate.jl M2-M4 文档 + 本机 VECLIB_MAXIMUM_THREADS/cpu_ratio 2.7 实证）——多线程/多簇均无增益。**[A+B]**
3. **容差紧约束（2048 特有，env_7 未遇）**：uniform 输入下 C 元素量级 ~561，f32 锚自身离 fp64 真值 **1.73e-3 已超 1e-3**（「绝对正确」不可判定，正确性=「与 f32 BLAS 锚一致」）；**任何改变 K 求和序的实现（K 分块/象限和/Strassen/转置 gemv）误差 1.6-2.9e-3 全被 1e-3 门拒**；只有保持 BLAS 求和序的 **M 维分解系**（rowwise 无转置 / row_halves / einsum_opt）逐位一致 0.0 可过。放宽到 5e-3 则 blocked/Strassen 数值复活（但性能仍全劣于 BLAS）。**[A]**
4. **naive 起点死刑确证**：纯 Python @2048 外推 274-550s → correctness 30s alarm 必死。**可行起点 = rowwise 无转置 np.dot：143ms（score 0.048）、逐位一致必过门、门成本 0.79s**——进化留有 ~20× 攀升空间。**[A]**
5. **env_7 门参数 @2048 全失配（KR2 硬证据）**：ref_runtime_band [0.1,0.5]ms 拒一切候选（含 BLAS 平价）→ 建议 **[5.0,10.0]ms**；bench/correctness 成本对诚实候选 ≤0.8s，30s/60s alarm 充裕；K=3+warmup1 足够（进程内极稳）。**[A]**
6. **exploit 注册表迁移性**：40 样本模式多数规模无关；D4 规模条件型需按 sizes=[64,2048] 适配；D3 量化型误差随规模变化，须重跑 verify 回归确认 40/40 拒识。**[A]**

### 判断节点（第 1 层 · L2 终版——并 research 结论，待 human 裁决 → L3 批量追问）

- **J1 达标判据（KR4）**：全族无 >1.0 路径（A 级确证）⟹ score≥1.0 判据必然失败。候选：(a) best score ≥0.9（BLAS 平价 10% 内）为成功、≥0.5 为部分成功、<0.5 须失败归因；(b) ≥0.5 即成功；(c) 不设阈值纯 best-effort。row_halves/einsum_opt（0.87-0.90）距 0.9 一步之遥——达成与否主要取决于采样能否到达 BLAS 族（与 J5/J6 联动）。
- **J2 任务规格**：2048×2048 **float32**（requirement 未指明 dtype；env_7 先例 f32）、uniform 协议、score=ref/cand vs np.matmul 锚——确认或改 f64。
- **J3 容差**：(a) 保持 1e-3——正确性=「BLAS 求和序一致」，判定干净零假阳性，合法解空间收窄为 M 分解系+直调系（阶梯 rowwise 0.048 → m_split ~0.90 → 直调 ~0.98）；(b) 放宽 5e-3——blocked/Strassen 数值复活成为「合法但慢」格（0.57-0.65），搜索面更宽但 A 级已知它们不通向更好性能。两案均无作弊通道（fp16/int16 量化误差 @2048 远超 5e-3，需回归重验）。
- **J4 naive 起点**：(a) rowwise 无转置 np.dot（143ms，慢但诚实，过门，20× 攀升空间）——推荐；(b) 纯 Python（首迭代必死于 30s alarm，debug 烧 3 attempts——env_7 教训 debug 占 31% token）；(c) row_halves（0.90，距顶太近无进化空间）。
- **J5 特征空间**：env_7 教训（225 格 × 10 迭代 = 只铺开不爬山，BLAS 族零采样）。候选：(a) 2-3 轴 ~9-16 格收缩域（M 分解粒度 × 派发目标 [× 线程]），10-15 迭代内可发生同格改进（gen>1）；(b) 照搬 env_7 4 轴 225 格 + ≥30 迭代（token 成本 60M+，记 zwm 账号）；(c) 4 轴 + 启动即 weight 干预收缩采样域。
- **J6 预算**：env_7 实测 ~2M token/迭代（含缓存）、4.5 迭代/1.9h。候选：max_iterations 10-15 + 墙钟 6-8h 兑底 + 首 2-3 迭代后 token 观测点；**记账在 zwm（同事）日额度**。
- **J7 模型档位**：(a) 全默认 glm-5.3 high（env_7 先例、链路已验证）；(b) builder 降档（config.agent_models 示例 catpaw-proxy/deepseek-v4-flash，token 减半但计划质量风险；需先探可用性）。
- **J8 干预与收尾预案**：承 env_7「纯 human 主导」+ 已知缺陷规避（stop 后手动收监督者树；North Star token 从 agent_ended.usage 聚合；运行期不跑重负载基准防 ±10% 计时噪声）。

### 人类裁决记录 · 第 1 轮（2026-10-21 19:5x，「全部按推荐」）

- **Q1/J1 → (a)**：成功 = best score ≥0.9；部分成功 = ≥0.5；<0.5 须失败归因。不设 >1.0 目标（A 级已证不可达，仅作观察项）。
- **Q2/J2 → (a)**：2048×2048 **float32**、uniform 协议、score=ref/cand vs np.matmul f32 锚。
- **Q3/J3 → (a)**：容差保持 **1e-3**（正确性 ≡ BLAS 求和序一致）；domain_knowledge 写明任务级容差事实（K 求和序改变实现 1.6-2.9e-3 会被拒）防 debug 空烧；特征空间不含已知死族格。
- **Q4/J4 → (a)**：naive 起点 = **rowwise 无转置 np.dot**（143ms / score 0.048 / 逐位一致）。
- **Q5/J5 → (a)**：特征空间收缩为 **2 轴 12 格**：`impl_strategy[rowwise_vectorized / m_halves / m_quarters / full_blas_dispatch] × dispatch_target[np_dot / np_matmul / einsum_path]`（ctypes 0.69 剔除——已知第三梯队；threaded 不入轴——同格可自然尝试）。
- **Q6/J6 → (a)**：**max_iterations=12**，墙钟 **8h** 兑底，迭代 3 后 token 观测点；记账 zwm（同事）日额度，异常即停。
- **Q7/J7 → (a)**：agent_models 全默认（glm-5.3 thinking high）；若 token 烧太快可停机改 agent_models 重启（档案/游标保留）。
- **Q8/J8 → (a)**：纯 human 主导干预承袭；运行期不跑重负载基准；收尾手动收监督者树 + token 从 agent_ended.usage 聚合 + 报告落盘。

### 终止判定（第 1 层 · L5，OKR 充分性自检）

- 模块 M1-M4 已命名、职责/输入/输出/边界清晰，pipeline M1→M2→M3→M4 串行确认 ✅
- 判断节点 J1-J8 全部经 human 裁决（全推荐案），无吞节点 ✅
- **充分性**：KR1（本层完成）+ KR2（M2 叶子可落实：env_7 基底 + 实测校准值全部就绪）+ KR3（M3 叶子可落实：pi_path/max_iterations=12/8h 已定）+ KR4（判据 0.9/0.5/归因已定，特征空间 12 格 + 12 迭代支撑可达性：m_halves/full 系格 0.87-1.0，采样必达概率高）+ KR5（M4 叶子可落实：监督者树补刀 + usage 聚合已知）⟹ O 可达 ✅
- 达标 → 下钻第 2 层叶子

---

## 第 2 层（叶子层）：四模块落实到代码/文件/工具/配置

> **本层全部消解，无判断节点**——叶子层的全部判断已在第 1 层经 human 一轮裁决完毕（J1-J8 全推荐案），本层为纯落实（含零项事实性修正）；无新增判断节点，无需 research 派发。

### M2 gen_env 叶子（KR2）——env_7 加固基底 × 2048 适配

**关键代码实现**（文件 + 要点）：
- `env/data/test_correctness.py` / `env/data/bench.py`：**照抄 env_7 加固版不改**（12 项加固全保留；规模/带/容差全部从 env.json 读）
- `env/env.json`：bench{size:2048, dtype:float32, warmup:1, k:3, timeout_s:60}；correctness{sizes:[64,2048], timeout_s:30, extra_seeds:1, double_run_consistency:true}；tolerance{correctness.max_error:1e-3, soft_gate.relative_band:0.05}；gates{score_max:2.0, runtime_ms_min:0.01, ref_runtime_band_ms:[5.0,10.0], static_check:true}
- `env/feature_space.json`：2 轴 12 格（Q5 裁决）；orthogonality 注明 code-form 正交 + 性能全序性质与死族剔除依据
- `env/code/reference.py`：np.matmul（锚，不变）
- `env/code/naive_start.py`：**rowwise 无转置 np.dot**（Q4 裁决；逐位一致必过门；不带 B 转置——预转置版本 1.71e-3 会被容差拒）
- `env/domain_knowledge.md`：2048 重写——score 阶梯（rowwise 0.048 → m_split ~0.90 → 直调 0.98）+ 容差紧约束事实（K 求和序改变 1.6-2.9e-3 被拒，只允许 M 维分解/BLAS 求和序）+ 死族清单（mp/blocked-K/Strassen/ctypes/threaded 实测劣化数据）+ 分析/调优 SOP；**不写门禁检测机制细节**（⑪）
- `env/skills/self_check.py` / `env/requirements.txt`：照抄
- `gates/exploit_registry.json` + `gates/exploits/*.py`：40 样本从 env_7 生成器再生成，D4 规模条件型按 [64,2048] 适配（n≤64/n==64/n==2048 变体）；**verify_exploits.py 逐一实测 40/40 拒识后才入注册表**（D3 量化型 @2048 误差随规模上升，预期更易拒；若个别泄漏→修样本）
- `gates/golden_examples/`：reference.py + rowwise.py（=naive_start）+ m_halves.py（0.90 已知好解，风格多样）
- `env/.venv/`：python3 -m venv + pip install -r requirements.txt（numpy 2.0.x）

**工具调用**：cp -R env_7/env → env_8/env（再改）· venv 安装 · 验收自检四项（reference 过正确性门 / bench 出分 / 40 exploit 全拒 / naive 过门）· 金样例 3/3 过门

**环境依赖**：numpy>=1.24（唯一第三方）；无 numba/cython/scipy

### M3 启动与运行监督叶子（KR3+KR4）

**配置**：super.json 预写 config 段——pi_path=/Users/logic/workdir/rick-dev-home/.rick/pi/agent/runtime/node_modules/.bin/pi；max_iterations=12；max_wall_time_h=8；agent_models={}；其余默认（adjust_interval=5 / debug_max_attempts=3 / soft_gate_k=3 / heartbeat 30s/90s）。预写须在 start 前（Go initSuperWorkspace 幂等保留已有 config）。

**工具调用**：`/Users/logic/workdir/rick-dev/bin/rick super start --name env_8 --env <abs>/env_8/env --daemon --max-iterations 12`（含门先证红：40/40 拒 + 金样例零误杀，失败即中止出清单）→ status --json 监控 → 迭代 3 后 token 观测点 → 干预仅 human 明示。

### M4 停止与收尾叶子（KR5）

`rick super stop --name env_8` → **手动收监督者树**（ps 找 PySupervisor/调度器进程组 kill，env_7 已知缺陷）→ 确认 state=stopped + 无孤儿 → 收尾报告 `env_8/final-report.md`（北极星终值从 agent_ended.usage 聚合 / KR4 判据对照 / 门先证红记录 / 复盘建议）。

---

## 全局终止声明

**Grilling 完成——设计树已遍历完毕，所有模块已落实到代码实现/文件结构/工具调用/环境配置。**

- 第 0 层 OKR（O + KR1-5 + 充分性）✅
- 第 1 层（M1-M4 + research-L1 简报 + 8 判断节点一轮裁决 + L5）✅
- 第 2 层叶子（M2/M3/M4 四维度全落实）✅
