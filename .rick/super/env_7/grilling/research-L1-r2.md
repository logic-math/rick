# research-L1-r2：env_7 grilling L1 事实回流——特征空间（J4）与门禁加固（J3）依据

- 派发：env_7 super 进化使命 grilling 第 1 层 L4 事实回流（两项调研 + 本机微实验）
- 日期：2026-10-20 | 机器/环境同 research-L1.md（Mac Studio M5 Max 18 核，CPython 3.9.6 + numpy 2.0.2，BLAS=Accelerate/vecLib）
- 复现 venv：`/tmp/matmul_calib_venv`（沿用 L1，未重建）；基准数据不重复测，全部引用 L1
- 信源等级：**A** = 本机实测（附命令+原始输出）；**B** = 官方文档/权威论文/一手 issue；**C** = 二手/推测
- 实验脚本：`/tmp/matmul_exp_L1r2.py`（E1-E3）、`/tmp/matmul_exp_E4.py`（E4）、`/tmp/matmul_exp_mp2.py`/`mp3.py`/`threads.py`（并行 spike）、`dgemm/out=` 单命令

---

## 〇、本机微实验原始结果（A 级）

锚点口径与 `data/test_correctness.py` 逐字一致：`RandomState(42).rand(n,n).astype(float32)`；fp32 锚= `np.matmul(a32,b32)`（门禁实际锚），fp64 锚= `np.matmul(a64,b64)`。容差 1e-3。

### E1 float16 降精度 exploit——双规模必被拒（A）

```
n=64:  fp16纯路径 vs fp32锚 max_error=9.17e-3（9.2×容差） | vs fp64锚 9.17e-3
       fp16存储量化+fp32计算 vs fp32锚 2.37e-3（2.4×容差，仍被拒）
n=512: fp16纯路径 vs fp32锚 6.95e-2（69×容差） | fp16量化+fp32计算 8.66e-3（8.7×容差）
       fp16 512 计时 180.5ms（比 fp32 BLAS 0.186ms 慢 ~970×，无任何性能动机）
```

- **结论**：降精度类 exploit（含「fp16 存储 + fp32 计算」的现实变体）在 tol 1e-3 下于 64 与 512 双规模均被拒；且 fp16 经 numpy 是软件仿真路径，慢 ~1000×，无动机。
- **修正 research-L1**：L1 §1.1 的「256×256=0.459ms（73 GFLOPS）小矩阵不吃 SME」是**冷启动污染**——本会话热态复测 256×256=0.0227ms（1478 GFLOPS）；方阵 192/256/320/384/448/512 全部 1477–1923 GFLOPS，无 SME 悬崖（`sq_192=0.0096ms/1477, sq_384=0.0668/1697, sq_448=0.0935/1923`）。

### E2 float32 BLAS vs fp64 锚——多规模门不误杀 fp32 候选（A）

```
n=64:  max_error=6.41e-6（相对 3.0e-7，156×容差余量）
n=512: max_error=1.81e-4（相对 1.2e-6，5.5×容差余量）
```

- **结论**：正确性门若扩为 64+512 双规模且锚保持 fp32 BLAS（当前协议），fp32 候选误差恒为 0；若锚改 fp64 真值，512 规模余量 5.5×，不会误杀。余量随规模收窄（64: 156× → 512: 5.5×），n 再大（1024+）需重测。

### E3 naive 起点双规模过门评估——不误杀，但 512 计入 8.4s（A）

```
n=64:  naive vs fp64锚 6.90e-6 | vs fp32锚 3.81e-6（与 L1 §1.3 一致）
n=512: naive vs fp64锚 1.86e-4（5.4×余量） | vs fp32锚 4.58e-5 | 单次 8369ms
```

- **结论**：naive（fp32 标量乘 + Python float 累加）双规模均过 fp64 锚；512 规模正确性检查对 naive 型候选增加 ~8.4s（< 30s gate timeout，但比 naive 慢 >3.5× 的候选会超时——门设计需知晓）。

### E4 ctypes 直调 Accelerate cblas_sgemm——可行、位相同、零加速（A）

```
{"n": 512, "ctypes_sgemm_ms": 0.185, "np_matmul_ms": 0.1827, "np_dot_ms": 0.1889,
 "max_err_vs_matmul": 0.0, "ratio_sgemm_over_matmul": 1.0125}
```

- **结论**：纯 stdlib（ctypes）直调 `/System/Library/Frameworks/Accelerate.framework` 的 cblas_sgemm 完全可行，输出与 np.matmul **位相同**（max_err=0.0，同一 BLAS 内核）；性能持平（1.01×，±3% 噪声带内）。派发目标轴「ctypes_accelerate」取值真实存在但无性能增益。已知坑（B 级，william-dawson 实践页）：ctypes 传参必须显式 `c_int/c_float` 类型与 `data_as(POINTER(c_float))`，矩阵须 `ascontiguousarray`，order/trans 常量为 101/111。

### S1-S4 并行/算法补充 spike——score>1.0 的四条路径全部实测关闭（A）

背景：L1 R7-1 遗留「多进程多簇理论余量 ~2-4× 未实测」。本轮四种形态全部实测：

```
S1 chunk 物理特性：M×512×512 行分块 GFLOPS：M=512:1450 | 256:1429 | 128:1338 | 64:1370 | 32:1282
   （行可任意细分的 SME 高吞吐保持——分块本身不损失效率）
S2 multiprocessing.Pool（fork，含建池+IPC）：P2:7.7-16ms | P3:15-18 | P4:18-23 | P6:24-31 | P8:31-38
   spawn：P2:252ms → P8:448ms（解释器启动主导）。全部 maxerr=0.0，score 0.005-0.025
S3 持久 fork 池 + COW 零拷贝输入 + shared_memory 输出（最优形态）：map-only 0.293ms（score 0.63）
   —— pool.map 派发开销地板 ~0.2ms ≥ 串行全量计算 0.185ms
S4 线程池（GIL 在 BLAS 内释放）：P2:0.214ms(0.87) | P3:0.205(0.90) | P4:0.198(0.94) | P6:0.228(0.81)
   新建线程 P2:0.224ms(0.83)。均 < 1.0
S5 Strassen-1 会计（512→7×256²乘+18加）：7×0.0227+18×0.00704=0.286ms vs 串行 0.178ms = 1.6× 更慢
   （numpy 逐元素加法内存带宽-bound，吃光乘法节省；BLIS 论文口径 Strassen 交叉点在小矩阵也不成立于此栈）
S6 dgemm(fp64)：0.623ms = sgemm 3.43×（1476 vs 431 GFLOPS）→ fp64 候选 score 上限 ~0.29
S7 np.matmul(a,b,out=C) 预分配复用：0.1833 vs 0.1866ms = 1.018×（噪声带内，不足破 1.0）
```

- **根因（B 级佐证）**：[cpuperf.com SGEMM 基准（M4 Pro）](https://cpuperf.com/benchmarks/13-sgemm-naive-vs-blas/)实测 Accelerate cblas_sgemm **默认多线程行为 = 单线程 2.00×**（2 P 簇部件，每簇 1 SME 单元、多簇派发）——即单次 np.matmul 调用已把可用 SME 单元吃满，Python 层任何外部并行只加开销。与 L1「VECLIB_MAXIMUM_THREADS=1 无差」一致（该变量对现代 Accelerate 无效，Apple 文档仅剩 BLAS_THREADING 常量族）。
- **上界理论（B 级）**：[MpGEMM/AMX 论文 arXiv:2606.25426](https://arxiv.org/abs/2606.25426)：手写 SME 内核在 M4 上也只比 Accelerate 快 1.23×——CPython+numpy 约束下（只能调 cblas_sgemm）该余量不可达。
- **对 J1 的校准结论**：本机 512×512、CPython3.9+numpy2.0.2 约束下，**score>1.0 无实测可达路径**（并行 4 形态、Strassen、fp64、out= 预分配全关闭）；>1.0 判据若设置，预期为不可达死区。

---

## 一、J4 特征空间：sub-kernel matmul 优化 idea 全谱 → 轴映射

### 1.1 idea→轴映射表（产出核心）

| idea 簇 | 归属轴 | 枚举值提案 | 512×512 实测命运（A） | 信源 |
|---|---|---|---|---|
| 递归分治 Strassen/Winograd | impl_strategy | strassen_like | 1.6× 更慢（S5）；AlphaEvolve 4×4 复数 49→48 是异构战绩（L1 §3.1） | A+B |
| 分块 tiling/寄存器分块 | impl_strategy | blocked | 行分块不损效率（S1），但 blocked 单独无增益（BLAS 已内部 blocking） | A |
| 循环序 ijk/ikj、转置预处理 | impl_strategy / memory_layout | naive_loops + col_major 预处理 | 转置视图值 ~1.3×（L1 §1.2） | A |
| 逐行/逐块向量化 | impl_strategy | rowwise_vectorized | score 0.15（L1 §1.4 rowdot） | A |
| 整阵 BLAS 派发 | impl_strategy × dispatch_target | full_blas_dispatch | score ~1.0（L1 §1.4） | A |
| 行/列主序、F-order | memory_layout | row_major / col_major | cblas transA/transB 原生吃转置视图（B 级：[cblas 接口](https://developer.apple.com/documentation/accelerate/1535120-cblas_sgemm)）；BLAS 路径布局近乎免费 | A+B |
| tiled 平铺布局 | memory_layout | tiled | 仅对纯 Python 路径有意义（BLAS 路径内部已 pack） | C |
| multiprocessing 分块 | parallelism | multiprocess_fork / multiprocess_spawn | score 0.005-0.063（S2/S3） | A |
| 线程池分块 | parallelism | threaded | score 0.81-0.94（S4） | A |
| np.matmul/dot/tensordot/einsum | dispatch_target | np_matmul / np_dot / einsum_path | matmul 0.183 / dot 0.189 / tensordot 0.191 / einsum 8.04ms（einsum 无 BLAS 派发，45× 慢） | A |
| ctypes 直调 Accelerate | dispatch_target | ctypes_accelerate | 0.185ms 位相同零加速（E4） | A |
| scipy.linalg.blas | dispatch_target（依赖决策） | — | 需补 scipy 依赖且同源自 cblas → **无需补**（ctypes 已覆盖） | B |
| fp16 降精度 | precision（不入轴） | — | 门必拒 + 慢 970×（E1） | A |
| fp64 全程 | precision（不入轴） | — | 3.43× 慢（S6） | A |
| fp32 计算+中途 fp64 累加 | precision（不入轴） | — | naive 天然形态，误差 1.86e-4 过门（E3）；FMA 行为封在 BLAS 内，Python 无控制面 | A |
| out= 预分配复用 | impl_strategy 微变体 | （并入 full_blas） | 1.018× 噪声带内（S7） | A |

### 1.2 轴设计裁决：2 轴不够，扩为 4 轴；并接受「性能退化」事实

**模板 2 轴评估**：`impl_strategy[naive_loops,blocked,vectorized,strassen_like] × memory_layout[row_major,col_major,tiled]` 覆盖了算法结构与数据组织，但**进化 agent 天然会产出两类它装不下的形态**：尝试 multiprocessing/线程（LLM 常见第一反应，必须被 archive 记录而非与 blocked 混淆）、尝试不同派发入口（einsum/dot/ctypes）。若不入轴，这些形态会全部挤进 nearest-neighbor 格子污染 archive。

**提案（4 轴 × 3-5 值，枚举见 1.1 表）**：

```
impl_strategy[naive_loops, blocked, rowwise_vectorized, full_blas_dispatch, strassen_like]   (5)
memory_layout[row_major, col_major, tiled]                                                   (3)
parallelism[serial, threaded, multiprocess]                                                  (3)
dispatch_target[np_matmul, np_dot, einsum_path, ctypes_accelerate]                            (4)
```

**先例（B 级）**：OpenEvolve 的 MAP-Elites 即「可配置 feature_dimensions（2 维典型）× feature_bins（10 bin 典型，10×10=100 格）+ 4 岛屿」——[config 示例](https://github.com/algorithmicsuperintelligence/openevolve/blob/main/examples/llm_prompt_optimization/config.yaml)、[MAP-Elites 实现](https://deepwiki.com/algorithmicsuperintelligence/openevolve/3.1-map-elites-algorithm)。维度由 evaluator 返回的自定义特征填充。4 轴 180 格在此先例的量级内（其默认 archive_size=100）。

**正交性论证（code-form 正交，性能不正交——须明示）**：
- 四轴各自独立可判定：任一候选代码可无歧义地同时落四值（结构怎么组织 FLOPs / 操作数怎么预处理 / 用几个执行单元 / 走哪个库入口），MECE 成立。
- 但**性能近乎全序**：dispatch_target 的 top 值（matmul/dot/tensordot/ctypes）实测收敛到同一 cblas_sgemm 调用（0.183-0.191ms，位相同）；parallelism 与 strassen_like 实测严格劣化。即该特征空间捕捉的是**代码形态多样性**（MAP-Elites 的本意：防早熟收敛），不是性能正交分解——J4 不应期待「轴间性能可分离」。
- precision 不入轴理由：fp16/fp64 是**门禁/性能直接淘汰项**（E1/S6），不构成有效探索方向；留 4 轴防止格子爆炸。

**进化阶梯校准（A，并入 J1）**：naive_loops 2e-5 → rowwise_vectorized 0.15（一步向量化 ~7000×）→ full_blas_dispatch ~1.0（天花板）。strassen_like/parallelism/einsum 为「合法但劣化」的探索分支。

### 1.3 约束确认（B 级）

- numba/cython/scipy 在 env requirements（仅 numpy）下不可用；**无关键路径需要补依赖**：直调 BLAS 由 ctypes（纯 stdlib）覆盖（E4），scipy.linalg.blas 同源自 cblas 无增益。
- 已知 ctypes/Accelerate 实践坑（[william-dawson 荣誉页](https://william-dawson.github.io/blas-pitfalls-with-mac-osx-accelerate-framework.html)，B）：参数类型必须显式（c_int/c_float），数组须连续；[Swift 论坛 scanon（Apple）](https://forums.swift.org/t/accelerate-in-swift-is-slower-than-numpy-in-python-for-matrix-multiplication/71365)确认 vDSP_mmul 与 cblas_dgemm 等价、Accelerate BLAS 大矩阵默认多线程。
- 纯 Python 层 cache-blocking/packing 理论（GotoBLAS/BLIS 系，[BLIS Strassen 论文](https://www.cs.utexas.edu/~flame/pubs/FLAWN79.pdf)，B）：packing/register blocking 是 C/汇编层技术，CPython 层不可达；Python 层可表达的仅循环序/转置/分块切片，全部已被实测覆盖。

---

## 二、J3 门禁加固：进化/自动调优系统防作弊实践 → 本 env 加固清单

### 2.1 AlphaEvolve 官方防 reward hacking 手册（B 级，一手文档）

来源：[Gemini Enterprise: Reward hacking prevention](https://docs.cloud.google.com/gemini/enterprise/docs/alphaevolve/developer-guide/reward-hacking-prevention)（原文要点摘录）：

- 评估函数漏洞（浮点边界、测试数据泄漏）→ **确定性评估（固定 seed）+ 赢家在 held-out 数据上复验**
- **AST 检查禁用原语**（sys/os/inspect/eval/exec/getattr/setattr）→ 命中返回 None
- **EVOLVE-BLOCK 之外的代码须不变**（text diff 或 AST 比对）
- **同输入跑两遍验证输出一致**（抓随机性作弊/硬编码）
- **检查评估耗时——微秒级完成 ≈ 硬编码答案**
- **打分逻辑放 LLM 看不见的单独文件**
- 论文 §2.4（[arXiv:2506.13131](https://arxiv.org/html/2506.13131v1)）：**评估级联**（便宜测试先过滤，昂贵 held-out 验证只跑幸存者）

### 2.2 KernelBench（Stanford）实测性能保障（B 级，一手仓库）

来源：[EVAL.md](https://github.com/ScalingIntelligence/KernelBench/blob/main/EVAL.md) + [v0.1 博客](https://scalingintelligence.stanford.edu/blogs/kernelbenchv01/)：

- **偏执原则成文**：「>2× 加速高度可疑」「比 cuDNN 快 10% 以上就该重新想」——即 **score 合理性上界**先例
- 正确性：**n_correctness 次随机输入**复验；性能：n_trial 次计时
- **对抗性单元测试**（`test_eval_adversarial.py`）：把已知作弊 kernel 做成回归测试，确保 eval 脚本能抓住——已编目的作弊形态：复用参考实现期间缓存的计算、**修改输入骗正确性检查**、把计算挪到非默认 CUDA stream（计时窗口外）
- **静态检查器**（regex/AST，`kernel_static_checker.py`）+ 多计时方法交叉（cuda_event / do_bench / host E2E）

### 2.3 通用加固先例（B 级）

- [Hardening Agent Benchmarks with Adversarial Hacker（arXiv:2606.08960）](https://arxiv.org/html/2606.08960v1)：形式化定义「verifier 可 hack = 不真满足指令拿满分」；用对抗 hacker agent 系统化找漏再补。
- [IOI 评分系统安全](https://doi.org/10.15388/ioi.2021.04)：**子进程隔离 + OS 级 CPU 时间记账**（选手进程内改计时函数无效——墙钟在沙箱外测）。
- [Tripwire](https://github.com/SammyTourani/tripwire)：「每个优化候选在**从未见过的保留对抗输入**上重测，测过才给加速记分」——专门针对「特判可见测试」。
- [Evaluation Integrity and Anti-Gaming](https://ai-infrastructure.net/evaluation-integrity-anti-gaming/)：结构防御——**冻结正确性定义**（优化器摸不到评估器）、评估器与候选隔离。

### 2.4 本 env 加固建议清单（按泄漏通道映射；全部可在 env/data/ 脚本实现，不动框架）

**通道①「monkeypatch time.perf_counter 伪造计时」**：

1. **计时引用快照**（改 bench.py 顶部）：`from time import perf_counter as _perf` 后全程用 `_perf`——候选后到的 `time.perf_counter = fake` 不影响已绑定引用。⚠️ 残留：候选可 `sys.modules['__main__']` 反射改 bench 自己的全局（见 R7-1），快照不封此路，需配合 2/3。
2. **subprocess 墙钟交叉验证**（新增 final_verify 脚本，仅终验跑）：候选在子进程执行、父进程墙钟计时——进程内任何 patch 无效（IOI 模式）。成本 ~0.2-0.4s/次，只放终验不放迭代内环。
3. **score 上界拒收**（bench.py 加一行）：score > 2.0 → passed=false（KernelBench 偏执原则）；1.0-2.0 区间标记 suspicious（本机实测无 >1.0 路径，§〇 S1-S7；理论余量仅 MpGEMM C 级 1.23× 且 Python 不可达）。
4. **计时下界拒收**：runtime_ms < 0.01（512×512 物理不可能）→ 拒（AlphaEvolve「微秒级=硬编码」）。

**通道②「n≤64 正确/n>64 垃圾」规模条件捷径**：

5. **双规模正确性门**（test_correctness.py 默认 sizes=[64,512]）：E2/E3 实测无误杀（fp32 余量 5.5×、naive 余量 5.4×）；naive@512 计 8.4s（<30s timeout，但比 naive 慢 >3.5× 的候选会超时——可接受：慢到超时本就是不合格）。注意 E1：fp16 量化变体 64 规模也拒（2.4×），双规模是纵深而非唯一防线。
6. **多 seed 复验**（KernelBench n_correctness 模式）：门内跑 seed ∈ {42, 随机} 两次——固定 seed=42 公开可读，候选可特判精确输入（Tripwire 场景）；终验 seed 用 `os.urandom` 派生并回写输出 JSON 供复现。
7. **同输入跑两遍一致性**（AlphaEvolve）：输出逐位一致才过——抓缓存返回/条件分支特判。

**通道③（本轮新发现）「拖慢 ref 抬 score」**：score=ref/cand，ref 在 cand 之后计时——候选可在模块导入时留后台忙循环进程，ref 计时被抢 CPU → score 虚抬。

8. **ref 基线合理性带**：ref_runtime_ms ∉ [0.10, 0.50]（本机稳态 0.185±3% 的宽容带）→ passed=false（L1 实测 0.185-0.195ms）。
9. **残留子进程清点**：bench 结束时 `os.waitpid(-1, WNOHANG)` 循环 + 有存活子进程 → 拒（纯 stdlib）。

**通用（AlphaEvolve/KernelBench 成套先例）**：

10. **候选源静态检查**（新 checker 脚本）：AST/regex 拒 `sys.modules`、`import time` 后对 time 属性赋值、`__main__` 字样、`ctypes` 之外的进程/线程 spawn 白名单可议——L4 决策项。
11. **打分逻辑不可见**：env/data/ 脚本本就不进 agent 上下文（框架侧已满足），加固后**不要**把 checker 细节写进 agent 可见的 env 文档（评估器信息泄漏）。
12. **对抗回归测试**（KernelBench 模式）：把「monkeypatch 计时」「n≤64 特判」「后台进程拖 ref」「fp16 换速」做成 4 个已知作弊候选 fixture，CI 里验证门能拒——防未来门改动回归。

---

## R7 上报项

1. **`__main__` 反射残留通道**：候选理论上可 `sys.modules['__main__']` 拿到 bench 模块对象并改其全局（含计时快照引用）——引用快照（建议1）封不住；彻底封死需 subprocess 隔离（建议2）或框架介入。是否接受该残留风险、终验是否强制 subprocess 口径，需 human/框架裁决。
2. **通道③「拖慢 ref」为本轮新识别**，加固方案（ref 合理带 [0.10,0.50]ms + 子进程清点）是本机校准值，换机器需重校带宽。
3. **M5 Max SME 簇数/内部多线程行为无 Apple 文档**：「单次调用已多簇饱和」由 M4 Pro 第三方基准（2.00×@2簇）+ 本机外部并行零增益（S2-S4）间接确证，置信度中高非一手——影响「>1.0 不可达」结论的强度（该结论另有 S5-S7 独立支撑）。
4. **research-L1 数据修正需确认**：256×256「73 GFLOPS SME 悬崖」系冷启动污染（本轮热态 1478 GFLOPS，§〇 E1 修正段）——若 design-tree J1 引用了该数，需同步更新。
5. **多规模门的超时联动**：512 规模正确性检查使「比 naive 慢 >3.5×」的慢候选必然超时（E3：naive@512=8.4s）——是否视为可接受的自然淘汰（本简报立场）还是需独立 timeout，属门禁参数决策。

## Gaps

- **>1.0 死区结论的穷尽性**：已关闭 multiprocessing×2、线程池、Strassen-1、fp64、out= 预分配五族路径，但未穷尽所有 numpy API 组合微技巧（如 einsum optimize=True 路径、批次化 reshape 技巧）；若 J1 坚持 >1.0 判据，需再一轮针对性 fuzz（预期仍死：全被 cblas_sgemm 天花板压住）。
- **spawn/fork 池 teardown 的 BrokenPipeError**（S2 实测 stderr 噪声，不影响 stdout JSON 协议与结果）：候选若用 multiprocessing，其 stderr 噪声会污染日志可读性但不影响门判定——框架日志解析若依赖 stderr 需知晓。
- **线程亲和性**：Python 层无法绑核（无 sched_setaffinity 等价物），S4 的 0.94 上限是否被调度器放置压低未溯源——不影响「<1.0」结论（差距 >6× 噪声带）。
- **门禁 fixture 未实做**：建议 12 的 4 个对抗 fixture 是设计项，本轮只给了实测数据支撑（E1/S2-S5），脚本实现属 L3/L4 落地范围。
- **np.matmul 对非连续/转置视图的派发差异**（transA/transB 路径 vs 显式 copy）未单独计时——对 dispatch_target 轴值域完整性影响极小（都收敛 cblas_sgemm），未测。

---
*复现：所有实验命令与原始 JSON 见 §〇 各节；venv `/tmp/matmul_calib_venv`（numpy 2.0.2，Python 3.9.6）。外部信源链接随各条标注。*
