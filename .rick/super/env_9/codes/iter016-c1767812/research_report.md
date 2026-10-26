# 调研报告 · 迭代 16 · dispatch_target=np_matmul|impl_strategy=m_halves

> 调研时间：2026-10-26 21:00-21:15（负载窗：load ≈ 6.9/18，CrossOver/Wine 后台 ~470%）。
> 方法：research_brief 指定论文检索（arXiv）+ 本 builder 单次探针实验（7 臂交错 5 轮 @8192，全部 bitwise == 整阵 np.matmul 锚）。
> 预算背景：iterations 17/17 已尽——本报告压缩深度，只留能落成动作的结论。

## 1. 现有代码调研（父本与相邻格精英的结构、瓶颈、可复用点）

**父本（本格精英 iter004，484.43ms，2026-10-24 安静窗）**：
- 结构：`asarray f32 → mid = m//2 → c = np.empty((m,n)) → np.matmul(a32[:mid], b32, out=c[:mid]) → np.matmul(a32[mid:], b32, out=c[mid:]) → return c`。契约齐备（matmul(a,b) 签名 / seterr 三项抑制 / _self_check 用例表 / 奇数行鲁棒）。
- iter16 profile：`matmul` self 100%（无任何 Python 层热点）——**父本已无赘肉，改进只能来自执行结构，不是代码微优化**。
- iter004 的 experiment_plan.json 是 turn-budget-exceeded 残片（plan_steps 空）——本格从未有过成型设计记录；iter004 的 484.43 是「首访单点」读数。

**同族相邻格证据**：
- np_dot|m_halves（iter010，462.82ms，10-26 窗）：同结构换 np.dot 入口。其记录给出同窗对照：np.dot 两半 449.6/466.5/459.8 vs iter004 复测 480.5ms（~5% 入口差，归因 numpy#18669 matmul out= 路径）。
- full_blas_dispatch（iter003，430.83ms = 全局纪录）：整阵单次派发。同批（10-24 同分钟 admit）整阵 430.8 vs 两半 484.4 → **m_halves 结构罚项 ≈ +54ms / +12.4%（同窗实测，非噪声）**。
- iter13/14（full 格）证伪清单：缓冲/预 fault/out= 变体全部与 plain 同 floor（本机 macOS minflt/call=0，numpy#18669 预 fault 收益不适用）；线程数收紧恒劣；V2/P1/P6 全拒。

**本 builder 探针（2026-10-26 21:10，n=8192，seed=42，7 臂交错 round-robin × 5 轮 + 逐臂 bitwise 校验，脚本 `_probe_variants.py`）**：

| 臂 | 形态 | med (ms) | 结论 |
|---|---|---|---|
| A | 整阵 np.matmul（ref 形态） | 589.2 | 负载窗地板参照 |
| B | 父本形态：顺序两半 out= | 602.4 | 官方 benchmark 同窗复测父本 607.9（ref 593.3）——与探针一致 |
| C | np.dot 两半 out=（iter010 形态） | 586.0 | 入口差 ~16ms 复现（诊断用，本格不可用 dot） |
| D | 两半无 out= + concatenate | 606.7 | 256MB 拷贝无收益，弃 |
| E | 持久缓冲 out=（iter14 P1 同型） | 607.0 | 与 B 平价——iter14 结论在 halves 形态同样成立，弃 |
| **F** | **两 Python 线程并发两半 out=** | **498.1** | **-104ms / -17% vs B；同窗还低于整阵 A 91ms** |
| G | 堆叠 gufunc 单调用（reshape(2,mid,n)） | 625.8 | 劣化，弃 |

- 全部 7 臂 `np.array_equal(result, np.matmul(a,b))` = True（bitwise，含 F）。
- F 的 worker 线程内出现 vecLib 虚假 FP 警告（divide/overflow/invalid）——**np.seterr 是线程局部状态，worker 内必须先 seterr**（数值本身 bitwise 正确，警告只是噪声 + 警告机制短暂持 GIL）。
- F 重叠成立本身证明 np.matmul 在 BLAS 段释放 GIL，且 **Accelerate（vecLib）允许两个并发 sgemm 重叠执行**（无 OpenBLAS 式全局串行锁）。

## 2. 业界落地方法（同类思路实际怎么做：机制 / 参数 / 工程细节）

- **OpenMathLib/OpenBLAS#4741「Enhancing Core Utilization in BLAS Calls: A Scalable Architecture」**：明确指出「同一时刻只允许执行一个 BLAS 调用，即使该调用只需 64 核中的 8 核」导致核利用率差；解法方向 = 解除单调用独占、允许多个 BLAS 调用并发共享核（需管理共享变量竞态）。——m_halves 的两次半阵 gemm 顺序执行正是该罚项的形态；本机 Accelerate 无此锁（探针 F 实测重叠 -104ms），并发两半 = 该业界方向在 np.matmul halves 上的落地。
- **Julia ThreadPinning.jl 文档（BLAS×语言线程交互警示）**：语言线程与 BLAS 线程并发易过订阅（oversubscription）反噬性能。——这是本实验的主要证伪面：安静窗（18 核全空）下 2×并发 gemm 若各自拉满线程队 → 36 线程过订阅，可能抹平甚至反超顺序形态。负载窗收益已实证，安静窗是待验假设。
- 工程细节（探针实证）：worker 线程需各自 `np.seterr(divide/over/invalid='ignore')`；两线程写同一 c 的不相交行切片（无别名，numpy#8440 不触发）；线程创建/join 开销 ~0.1ms 级，对 500ms 量级无感。

## 3. 论文的实验落地细节（实验怎么配、规模与硬件差异、可复现要点）

- **GEMMbench（arXiv:1511.03742，Lokhmotov 2015）**：GEMM 评测几十年无公认方法论；框架主张 = 固定随机 seed、分离 warmup 与计时循环、聚合中位数、实验数据可回放（CK repo）。与 env benchmark 协议（warmup 1 + k=3 取中位、seed 42、ref 同进程同协议）同构——本实验遵循其可复现纪律：**每个门禁工具至多跑一次，读数即记录，不复跑不追噪声**（亦是 iter12-15 三轮 900s 进程死亡的教训）。
- **Euro-Par'07（Marker et al.，Toward Scalable Matrix Multiply）+ arXiv:1905.03136（tall&skinny GEMM）**：vendor BLAS 对块高减半/非方阵形状吞吐下降（1-D M 划分的并行效率损失）——归因 m_halves vs 整阵 ~12% 罚项的机制来源；并发重叠两个半阵正是回收该罚项的手段（一个 gemm 流内存时另一个算）。
- 硬件差异提示：以上文献多为 OpenBLAS/MKL/GPU 语境；本机是 macOS Accelerate + 18 核，行为以探针实测为准（探针即本报告 §1 的落地数据）。

## 4. 迁移结论（本实验具体要借哪一条，落到 plan_steps 的哪一步）

1. **并发两半（探针 F）→ plan_steps 第 2 步**：父本 matmul() 内核由「顺序两次 np.matmul out=」改为「两个 threading.Thread 并发，worker 先 seterr 再各自半阵 np.matmul out=，try/finally join」（依据 §2 OpenBLAS#4741 并发解除方向 + §1 探针 F -104ms 实测）。
2. **per-thread seterr → plan_steps 第 2 步实现要点**（§1/§2 探针警告实证；契约与父本其余部分逐字保留，单变量：顺序→并发）。
3. **GEMMbench 可复现纪律 → plan_steps 第 4 步**：verify/benchmark/gate_check 各至多一次，读数即记录（含 ref 读数——负载窗 reject 时用 ref 判窗，见实验记录 §3）。
4. **证伪面如实保留（§2 ThreadPinning 过订阅警示）→ 实验记录 §2/§3**：安静窗若 F ≥ 484.43 且 ref 正常带内 → 拒绝并记录「并发不迁移到安静窗」。
