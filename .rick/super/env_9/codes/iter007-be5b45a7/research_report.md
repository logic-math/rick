# 调研报告 · 迭代 7 · dispatch_target=np_dot|impl_strategy=full_blas_dispatch

## 1. 现有代码调研（父本与相邻格精英的结构、瓶颈、可复用点）

- **父本（本格无精英，首访 visit=1，从起点出发）**：`env/code/naive_start.py`（62 LOC，rowwise np.dot 无转置）。profile @8192（ctx 注入）：runtime 11688ms，热点 `<listcomp>` self 11506ms / cum 11507ms（**99.8%**），`stack` 12.9ms，`dot` 1.0ms。瓶颈归因明确：**Python 层 8192 次逐行循环 + 每行 gemv 全量重读 B（内存墙）+ np.stack 装配**——BLAS 内核本身只值 ~440ms。
- **相邻格精英（跨格重组父本）**：`codes/iter003-0de54e75/solution.py`（cell `np_matmul|full_blas_dispatch`，perf 430.83ms，**当前全场纪录**）。结构：契约照抄父本（`matmul(a,b)` 接口、asarray→f32、`np.seterr` 抑制、main 自检 JSON 协议），唯一改动是 kernel 换成单次 `np.matmul(a32, b32)` 整阵派发。**可复用点：整阵派发的代码模板与契约保持方式——本实验唯一差异是把入口换成 `np.dot`**。
- **本 session 预检（同进程交错 A/B，RandomState(42)，@8192，warmup1+3 计时取中位）**：
  - 正确性：`np.dot(a,b)`（2-D）与 `np.matmul(a,b)` 在 1024 与 8192 均 `array_equal=True`、maxdiff **0.0**（逐位一致——同一 cblas_sgemm 调用路径，K 求和序不变）。
  - 性能：np.dot 中位 437.4ms（timings 435.2/437.4/500.7），np.matmul 中位 433.1ms（432.7/433.1/507.9）——同带 ±1%，属噪声量级。
- **相邻格另一格** `einsum_path|full_blas_dispatch` 无精英（iter6 no_candidate，einsum optimize 路径未产出候选）；本实验与其方向不同（np.dot 直派，无 optimize 路径搜索环节）。

## 2. 业界落地方法（同类思路实际怎么做：机制 / 参数 / 工程细节）

- **NumPy 前端→BLAS 派发**：对 2-D float32 连续阵，`np.dot(a, b)` 与 `np.matmul(a, b)` 都直接落到同一条 cblas_sgemm 调用（NumPy 的 `cblas_matrixproduct` / matmul 路径最终同一内核；ashvardanian《NumPy vs BLAS》实测 np.dot 一维直调 cblas_sdot，二维同理走 sgemm）。单次整阵派发 = **一次 BLAS 调用**（Python 层开销 µs 级）；逐行派发 = 8192 次 gemv + listcomp/stack 装配（秒级开销）。
- **测量协议**（GEMMbench 框架）：固定 seed 输入、warmup 后 K 次计时取中位、同输入跨实现对比、显式给出噪声带——与本 env `bench.py`（k=3 中位 + ref 合理带 [380,700]ms + score 上界拒收）完全同口径。

## 3. 论文的实验落地细节（实验怎么配、规模与硬件差异、可复现要点）

- **arXiv 1511.03742（GEMMbench: reproducible benchmarking of matrix multiplication）**：提出可复现 GEMM 基准方法论——预热消除冷启动、重复计时取中位抗离群、固定输入复用、跨实现同输入对比。可复现要点：**同进程交错 A/B** 是区分 ±1% 级差异的最低成本手段（本 session 预检即此法）。
- **arXiv 1911.09421（The Linear Algebra Mapping Problem）**：系统研究「线性代数表达式→高性能库调用」的映射：不同前端入口（dot/matmul/@）在 2-D float32 下映射到同一 BLAS gemm 内核。可复现要点：**换入口不换内核 ⇒ 数值逐位一致**（预检 array_equal=True 证实），性能差异只在入口开销。
- **arXiv 1609.00076（BLISlab, FLAME WN#80）**：GEMM 优化教学沙盒——调优 BLAS/BLIS 内核是性能上限，上层（Python/装配层）只贡献开销。规模差异：@8192（工作集 ~768MB，内存墙区间）大阵行为由内核与缓存分块决定，Python 层循环是纯额外开销（本格父本 99.8% 热点即此）。

## 4. 迁移结论（本实验具体要借哪一条，落到 plan_steps 的哪一步）

1. **单次整阵 `np.dot(a32, b32)` 直派 cblas_sgemm**（1911.09421 的映射结论 + 邻格 iter003 整阵派发模板 + 本 session 逐位预检）→ **plan_step 2 的唯一 kernel 改动**；删除逐行 listcomp + np.stack（父本 99.8% 热点）。
2. **契约与自检协议照抄**（邻格精英 iter003 的契约保持方式：`matmul(a,b)` 接口、asarray→f32、`np.seterr`、main 的 warmup/计时/JSON 输出）→ **plan_step 1**；不改 env 定义、不加 out=、不预转置 B。
3. **GEMMbench 式读数口径**（k=3 中位 + 同进程 A/B + 噪声带意识）→ **plan_step 3-5 的验证命令与 expected_behavior 量级**：@8192 中位落在 434-488ms 平价带（预检 437.4ms），score ~0.93-1.05，max_error 0.0。
