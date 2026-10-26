# 调研报告 · 迭代 15 · dispatch_target=np_matmul|impl_strategy=full_blas_dispatch

## 0. 本报告的事实来源（先声明口径）

- 本机实测探针（builder 本迭代为选题跑的，全部落盘 `probes/`）：
  `thread_probe.py`（扫描 11 档）、`thread_ab.py`（3 轮交错 A/B）、`final_confirm.py`
  （跨线程数 SHA 逐位比对 + benchmark 协议全真模拟）。读数见 §1。
- 父本/邻格代码与台账：iter003 精英（本格）、iter013/014 实验记录（本格前两次重访）、
  domain_brief（天花板与死族清单）。
- 外部检索：arXiv 3 篇 + Apple 文档/threadpoolctl/anracon 工程佐证（§2/§3）。

## 1. 现有代码调研（父本与相邻格精英的结构、瓶颈、可复用点）

**父本（本格精英 iter003-0de54e75，430.83ms）**：`matmul(a,b)` = asarray f32 ×2 +
单次 `np.matmul(a32,b32)`，61 行，无 out=、无预分配、无转置。profile（iter13-15 三次）：
`matmul self 100.0%`、`asarray 0.002ms`——**包装层零开销，时间 100% 在 vecLib sgemm 本体**。

**本格前两次重访（iter13/14）的收口结论（必读，避免重复）**：
- iter13 V2（fresh+memset 预 fault+out=）：447.25ms 平价，reject worse_than_elite。
- iter14 P1（per-shape 持久缓冲+out=）/P5/P6（spacer 布局）/P7（dot+out）：探针
  6 形态**同 floor ≈433ms = sgemm 本体**，全臂 minflt/call=0（macOS 分配器回收已 fault
  页，numpy#18669/OpenBLAS#3197 的预 fault 收益在本机无成本可省）。gate 432.81ms
  ratio 1.0046 band，仍被框架按 worse_than_elite 拒。
- **缓冲管理维度已证伪殆尽**：分配规避/out=/预 fault/持久复用/布局规避全部平价。
  iter14 debug 复核结论：「plain np.matmul warm 后 minflt=0，等效已预 fault」。

**相邻格（m_halves 484.4 / m_quarters 481.8 / np_dot m_halves 462.8 / einsum 482-493）**：
全族坍缩 434-488ms 窄带（±6%）。结构维度（M 分解 / 换入口）无增益；einsum 入口每调用
多 ~40ms 包装税（iter12）。**跨格结论：结构维度全部收口，剩余自由度不在「怎么组织 FLOPs」。**

**本机硬件（本轮 sysctl 实测，关键新事实）**：Apple M5 Max，`hw.ncpu=18`，
`hw.perflevel0.physicalcpu=6`（P 核），`hw.perflevel1.physicalcpu=12`（E 核），64GB。
**18 线程 = 6P+12E 异构池**——vecLib 默认把 BLAS 工作线程撒满全部核。这是历次 profile
从未刻画过的一层：sgemm 本体的 floor 由「P 核干完活等 E 核过栅栏」的 straggler 决定。

## 2. 业界落地方法（同类思路实际怎么做：机制 / 参数 / 工程细节）

- **BLAS 线程池线程数调优（非全核）**：vecLib/Accelerate 官方旋钮 =
  `VECLIB_MAXIMUM_THREADS` 环境变量（Apple BLAS_THREADING 文档；threadpoolctl#136
  确认 vecLib GEMM 线程池受该 env 控制）。桌面实测经验（anracon/Linux-Blog，
  OpenBLAS on 8 核）："It is a bad idea to provide all CPU cores to OpenBlas – which
  unfortunately is the default… using only 4 out of 8 core threads is reasonable to get
  an optimum performance"——**最优线程数 < 物理核数是普遍现象**（超线程争用/频率/缓存）。
  异构场景更极端（§3 论文）。
- **本机实测（builder 探针，决定性读数）**：
  - **旋钮有效性**：`VECLIB_MAXIMUM_THREADS=1` → 8192 sgemm 650ms（vs 多线程 ~450ms）
    ——旋钮确实约束 vecLib 池。
  - **2-18 线程中位无增益（plateau），但 floor 有**：3 轮交错 A/B（3 臂换序、每臂
    fresh 进程 1 预热 + 7 计时）——`default` floor [453.2, 441.6, 449.3]（pooled 441.6，
    median 455.5）；`v6` floor [429.5, 452.9, 429.5]（pooled **429.5**，median **433.2**）；
    `v12` floor pooled 434.8；`late6`（numpy import 后才 setenv）floor pooled **425.3**。
    **6 线程（= P 核数）floor 比 default 深 12-16ms（~3%）**；v6 两轮 floor 低破 430。
  - **在位可实现性（Q2，关键）**：`late6`（numpy 已 import、首次 BLAS 调用前 setenv）
    同样生效 → **可在 solution.py 模块 import 期设置**（benchmark.py 先 import numpy、
    后 exec 候选模块，但 BLAS 池在首次 gemm 调用才创建）。
  - **跨线程数逐位一致（Q3，关键）**：default 与 v6 的输出在 n=64 与 n=8192 的
    sha256(uint32 view) **完全相同** → 线程数不改变任何 C 元素的 K 累加序
    （BLAS 线程切 M×N 输出域，单元素 K 点积始终单线程固定序）。
  - **当日机器噪声**：协议模拟臂撞上外部争用窗（cand [588,574,439] / [455,453,460]）
    ——与 iter13/14 观察的 580-590ms 尖峰窗一致；floor 读数不受其影响（A/B r1 全臂
    同抬，窗口是全局的不是臂特有的）。
- **数值正确性口径**：正确性 ≡ 与 f32 BLAS 锚逐位一致（domain_brief）。线程旋钮
  不动 K 求和序（SHA 证明）+ verify 的锚与候选同进程同池 → max_error 预期 0.0。
  **不属于任何死族**（死族全是改 K 求和序的算法变体；线程旋钮是进程级执行配置）。

## 3. 论文的实验落地细节（实验怎么配、规模与硬件差异、可复现要点）

（检索按 brief：GEMM benchmark 方法论 / 线程规模影响 / 硬件差异；≤3 篇，只取能落动作的）

1. **arXiv:1506.08988** — *Architecture-Aware Configuration and Scheduling of Matrix
   Multiplication on Asymmetric Multicore Processors*（BLIS gemm on ARM big.LITTLE AMP）。
   - 机制：多线程 gemm 在非对称多核上的性能取决于**微内核在集群间的分布**；
     对 big.LITTLE 做 asymmetric-static/dynamic 调度、按集群配置 cache-aware 分块，
     而不是把线程均匀撒满全核。
   - 落地：本机即 AMP（6P+12E）。把 vecLib 池限到 6（= P 集群规模）等价于「不要让
     E 核线程进栅栏同步」的最简实现——不动代码结构，只动池配置。
2. **arXiv:1511.02171** — *Multi-Threaded Dense Linear Algebra Libraries for Low-Power
   Asymmetric Multicore Processors*。
   - 机制：多线程 BLAS「线程撒满所有核」在 AMP 上让快核在栅栏处等慢核（straggling），
     大矩阵（m=n=k≥2000 段）受限线程数/非对称调度才是高效形态。
   - 落地：本实验规模 8192 正在其「大矩阵」段；限 6 线程的 floor 收益方向与论文一致。
3. **arXiv:1511.03742** — *GEMMbench: a framework for reproducible and collaborative
   benchmarking of GEMM*。
   - 机制：可复现 GEMM 评测 = 固定 seed 输入、warmup 丢弃、重复计时、防漂移的交错
     轮次与统计口径（中位/分布）。
   - 落地：本实验探针方法论照此执行（3 轮换序交错 A/B、每臂 fresh 进程、warmup 丢弃、
     floor=min 作为无争用信号）；门禁口径（warmup1+K3 中位、ref 同进程）天然对齐。

## 4. 迁移结论（本实验具体要借哪一条，落到 plan_steps 的哪一步）

1. **借 1506.08988/1511.02171 的「限线程到快集群」机制** → plan_steps 第 1 步：
   在 `solution.py` 模块顶部（`import numpy` 之前）加
   `os.environ.setdefault("VECLIB_MAXIMUM_THREADS", "6")`（6 = `hw.perflevel0`），
   其余与 iter003 精英逐行相同。这是一行改动、单变量、可归因。
2. **借本机 A/B 数据（§2）** → 选 **6** 而非 12/18：v6/late6 是全部 11 档里唯二
   floor 深破 430 的臂；plan_steps 第 2 步的自检命令预期 max_error=0.0（SHA 证明）。
3. **借 GEMMbench/iter14 过程纪律** → plan_steps 第 4 步：门禁三件每件至多跑一次、
   读数即记录、不追噪声（当日存在 580-590ms 全局争用窗，读数撞窗要如实记录而非复跑）。
4. **合法性边界（给 coding/debug 的红线，来自 benchmark.py 源码审计）**：
   - 旋钮必须**进程级、先于首次 BLAS 调用、对称**设置（candidate 加载先于任何 BLAS 调用
     → ref 同池受益，ref≈429-435 落在 [380,700] 带内，score≈1.0 平价）。
     **禁止**任何「只让 ref 慢」的非对称用法（= 加固⑧要堵的作弊通道）。
   - 静态检查黑名单（sys.modules / `__main__.` / time 属性赋值 / signal.alarm= /
     `__builtins__`）均不涉及；`os.environ.setdefault` 干净。
