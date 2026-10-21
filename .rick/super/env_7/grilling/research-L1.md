# research-L1：env_7 matmul（512×512 float32）基准校准与外部知识（grilling L1 简报）

- 派发：env_7 grilling L1（设计树 §L1「research 派发」，服务 J1 预算/达标判据、J3 门禁定制裁决）
- 日期：2026-10-20 | 机器：Mac Studio（Apple M5 Max，18 核 = 6 Super + 12 Performance，64GB，macOS Darwin 27.0.0 arm64）
- 信源等级（本简报约定）：**A** = 本机实测（附命令+原始输出）；**B** = 官方文档/权威社区（附链接）；**C** = 二手/推测
- 方法：一次性 venv（`python3 -m venv /tmp/matmul_calib_venv && pip install -q "numpy>=1.24"`）→ numpy 2.0.2（Python 3.9 下最新），BLAS = **Accelerate/vecLib**（`np.__config__`：blas/lapack name=accelerate, found=true）

---

## 一、本机基准实测（A 级）

### 1.1 reference.py（numpy 基线 = score 分母）—— 冷启动效应 + 稳态

复现：`cd super/env_templates/matmul && /tmp/matmul_calib_venv/bin/python code/reference.py`（连跑多遍）

```
第1遍(冷):  {"ref": true, "n": 512, "dtype": "float32", "runtime_ms": 1.383, "timings": [1.354, 1.383, 1.392]}
第2遍:      {"runtime_ms": 0.1873, "timings": [0.1855, 0.1873, 0.1883]}
第3-8遍:    0.1859 / 0.1929 / 0.1937 / 0.1951 / 0.1851 / 0.1939 （全部 0.185~0.195ms）
```

- **稳态基线 = 0.185~0.195ms（中位 ~0.19ms）≈ 1450~1480 GFLOPS**（2·512³/0.19ms）。
- **冷启动效应（机器级，非进程级）**：本次会话第 1 个计时进程 1.38ms（慢 ~7×），此后所有进程都 0.19ms。进程内预热 1 次消不掉——是机器/SME 协处理器冷态。**对 score 无影响**（bench.py 同进程内先计 cand 后计 ref，二者同冷同热），但绝对值首跑偏慢可预期。
- `VECLIB_MAXIMUM_THREADS=1` 下同速（0.1878ms）——现代 Accelerate 不走该变量（§2.1 机制解释）。
- 1024×1024 补测 = 1.532ms（1401 GFLOPS）；256×256 = 0.459ms（仅 73 GFLOPS，小矩阵不吃 SME 高吞吐）。

### 1.2 naive_start.py（进化起点）

复现：`/tmp/matmul_calib_venv/bin/python code/naive_start.py --n 64 | --n 512`

```
--n 64  ×2:  {"n": 64, "impl": "naive_loops", "runtime_ms": 7.886 / 7.960, "checksum": 960.351...}
--n 512 ×3:  {"n": 512, "runtime_ms": 4153.714 / 4185.139 / 4116.813, "checksum": 65094.917...}
```

- **naive 512 独立运行 = 4.12~4.19s（稳）**——比任务/模板 docstring 预估「15s+」快 ~4×。原因：①M5 Max P-core 快（纯 Python mul-add ~42ns/op，见 §1.5）；②模板 naive 已带转置视图优化（`zip(*b)` + 局部变量）。
- **bench 协议下 = 8.85s（关键差异）**：bench.py 传 numpy 数组而非嵌套 list，逐元素索引 numpy 标量装箱开销 ~2×：

```
$ /tmp/matmul_calib_venv/bin/python data/bench.py --candidate code/naive_start.py
{"runtime_ms": 8853.31, "score": 2.132e-05, "timings": [8928.8, 8853.3, 8842.1], "ref_runtime_ms": 0.1888, "n": 512, "k": 3}
```

### 1.3 正确性门（64×64）实测

```
$ .../python data/test_correctness.py --candidate code/naive_start.py
{"passed": true, "max_error": 3.8147e-06, "n": 64, "tolerance": 0.001}
rowdot 候选: {"passed": true, "max_error": 7.6294e-06}   blas 候选: {"passed": true, "max_error": 0.0}
```

naive 过门（误差 3.8e-6 ≪ 1e-3）；门耗时 ~8ms，30s timeout 余量 >3000×。

### 1.4 np.matmul vs 逐行 np.dot（agent 进化典型中间形态）

复现：`/tmp/matmul_calib_venv/bin/python /tmp/matmul_calib_rowdot.py`（c[i]=np.dot(a[i], b)，512×512 f32；两次独立运行）

```
run1: matmul_ms 0.1817 (1477 GFLOPS) | rowdot_ms 1.1902 (225 GFLOPS) | 比值 6.55× | max_err=0.0
run2: matmul_ms 0.1850 (1451 GFLOPS) | rowdot_ms 1.1821 (227 GFLOPS) | 比值 6.39×
```

用真实 bench 协议测逐行 np.dot 候选与纯 BLAS 候选（`data/bench.py --candidate <文件>`）：

```
rowdot: {"runtime_ms": 1.190, "score": 0.1534,  "timings": [1.194, 1.180, 1.190], "ref_runtime_ms": 0.1826}
blas:   {"runtime_ms": 0.1876, "score": 1.0016, "timings": [0.185, 0.188, 0.188], "ref_runtime_ms": 0.1879}
blas重跑: {"runtime_ms": 0.1904, "score": 0.9768, "timings": [0.183, 0.191, 0.190], "ref_runtime_ms": 0.1860}
```

### 1.5 score 预期汇总（供 J1 达标判据 / J5 预算裁决）

| 形态 | runtime_ms | score（ref/cand） | 说明 |
|---|---|---|---|
| naive 起点（bench 协议） | 8853 | **2.13e-05** | 起点，比 BLAS 慢 ~47000× |
| naive 起点（list 输入） | 4154 | ~4.6e-05 | 独立运行口径 |
| 逐行 np.dot | 1.19 | **0.153** | 一步向量化即可跳 ~7000× |
| 纯 np.matmul（= 参考实现） | 0.188 | **0.98~1.00** | 「天花板」= score 1.0 |

- **⚠️ 噪声带（J1/J3 关键事实）**：与参考实现完全相同的候选，两次 bench 得 **1.0016 与 0.9768**（±3% 波动）。**score=1.0 不是安全通过线**——判据若定 1.0，完美实现也会随机挂；env.json 软门禁带 5%（relative_band 0.05）与本实测噪声匹配。
- **>1.0 的理论余量存在但不宽**：单次 np.matmul 走单 SME 协处理器路径 ~1.45-1.5 TFLOPS（§2.1）；row-dot 单核 226 GFLOPS × 18 核理论上限 ~4 TFLOPS，且 Strassen 一层递归省 ~12.5% 乘法。但 Python 进程/线程开销大，实际能到 >1.0 需要相当巧的候选（多进程分块或 Strassen-on-BLAS）。这是事实校准，不预判判据。
- 纯 Python 真·naive（无转置优化，list 输入）本机外推 512 = **5.4~5.7s**（n=64: 11ms, n=128: 84ms，立方外推）——即模板 naive 的转置优化在本机只值 ~1.3×（4.15 vs 5.5s），并非文档说的 3-5×（M5 Max 上 cache 效应被快核掩盖）。
- **时长预算校准（J5）**：bench 协议下 naive 起点 4 次运行（预热1+K3）≈ 35s + ref ~1s ≈ **36s/次 bench**；正确性门 ~1s。迭代改进后 bench 时长骤降（row-dot 候选 <1s）。60s 单次 timeout 对起点余量 ~6.8×。

## 二、外部校准（B 级）

### 2.1 Apple Silicon Accelerate/vecLib GEMM 性能——本机实测在正常范围

来源：[AppleAccelerate.jl 官方 benchmarks](https://julialinearalgebra.github.io/AppleAccelerate.jl/dev/benchmarks/)（M2 Max, macOS 26, 2026-07 复测，等级 B）：

- Float32 GEMM 单线程（OpenBLAS vs Accelerate）：N=256: 103 vs **1213 GFLOPS**（11.8×）；N=1024: 105 vs **1381 GFLOPS**（13.2×）；N=4096: 104 vs 1150。
- **机制**：Accelerate 把大 GEMM 派发给 **SME/AMX 矩阵协处理器**（每 CPU 核簇共享 1 个，非 GPU）；单线程即可饱和 1 个簇的协处理器 → 吞吐只随「簇数」扩展，不随线程数。M2 Max（2 个 P 簇）实测 1 线程 1158 GFLOPS → 2 线程 2252 GFLOPS → 8 线程 2240（平坦）。
- 独立佐证：arXiv:2409.18779（Apple Silicon 矩阵协处理器吞吐测量）。
- **校准结论**：本机 M5 Max 512×512 实测 1450-1480 GFLOPS ≈ M2 Max 单簇稳态值（1158-1381）量级、略高（代际进步），**正常**。numpy 单次 matmul 调用只吃到 ~1 个簇的 SME，与 VECLIB_MAXIMUM_THREADS=1 无差别的实测一致。
- **对判据的推论**：score=1.0 线 = 单簇 SME 路径；>1.0 必须绕开单调用路径（多进程多簇 或 算法性省 FLOPs）。

### 2.2 macOS vecLib 虚假 FP 标志——问题确认、根因已知、模板抑制正确且仍必要

来源（均为 B 级一手 issue/PR/release notes）：
- [numpy#28687](https://github.com/numpy/numpy/issues/28687)「wrong errors e.g. "divide by zero encountered in matmul" on MacOS M4」——Apple 已知悉，M4 用户大量复现。
- [numpy PR#29223](https://github.com/numpy/numpy/pull/29223)：**根因 = SME 与 FPSR 交互**。Arm 官方规范：进入/退出 Streaming SVE 模式时 FPSR.{IOC,DZC,OFC,UFC,IXC,IDC,QC} **全部置 1**——即只要用了 SME，FP 标志就全亮，与数值无关。
- [numpy#29820](https://github.com/numpy/numpy/issues/29820)：`identity(15)@identity(15)` 报警告而 `identity(14)` 不报——SME 内核有规模阈值（小矩阵走普通 NEON 路径）。
- [numpy 2.3.0 release notes](https://numpy.org/doc/2.4/release/2.3.0-notes.html)：官方确认 M4 Mac 会看到此类警告。

**本机 A 级复现**（不设 seterr 抑制）：

```
$ .../python -W always -c "... np.matmul(a,b) ..."  # a,b = 512×512 rand float32
RuntimeWarning: divide by zero encountered in matmul
RuntimeWarning: overflow encountered in matmul
RuntimeWarning: invalid value encountered in matmul
结果完全正常: finite=True, max=149.67
```

- **新发现的边界**（本机实测，A）：**sgemm（np.matmul）触发，sgemv（np.dot 行点积）不触发**——逐行 np.dot 中间形态天然免疫；上界为真。
- **模板处置确认**：reference.py / bench.py / test_correctness.py 均已在 import 时 `np.seterr(divide/over/invalid="ignore")`，进程级生效（候选同进程被覆盖），numpy 2.0.2 下无新坑。唯一残留：候选若自设 `np.seterr` 回 warn 或包 `np.errstate` 上下文，警告只会污染 stderr（stdout JSON 协议不受影响），不影响功能。
- **附带的测量学警示**（AppleAccelerate.jl 原话）：共享/嘈杂环境下「对协处理器计时噪声太大」——bench 的 ±3% 噪声带（§1.5）与此一致，进化期间避免并发重负载。

### 2.3 纯 Python 512×512 三重循环公认耗时——本机偏快但正常

外部锚点（B/C 级）：
- [PyPy 官方博客 2018 基准](https://morepypy.blogspot.com/2018/06/repeating-matrix-multiplication.html)（cfbolz）：naive CPython 三重循环 500×500 在 2018 桌面级 CPU 为 **数十秒量级**（PyPy 快 ~50-100×）。
- [Martin Thoma「Matrix Multiplication 2020 Update」](https://martin-thoma.com/matrix-multiplication-2020/)：CPython 3.8 naive 在 2000×2000 为小时级 → 折算 500-512 在旧机器 ~10-60s。
- 教学/社区基准普遍口径：**naive 纯 Python 512×512 ≈ 15~60s（旧 CPU）/ 数秒（新快核）**。

**校准结论**：本机实测（无转置 5.4-5.7s；模板带转置 4.15s；bench 协议 numpy 输入 8.85s）处于公认量级的**快端但完全正常**——M5 Max P-core 纯 Python mul-add ~42ns/op（旧机器典型 100-300ns/op）。模板 docstring「512 单次 15s+」按旧基准写，**本机实际 ~4-9s，bench 60s timeout 余量充足**；「首迭代 15-30s」的预估应修正为「bench 全程（预热+3 计时+ref）约 36s」。

## 三、进化预期（C 级——公开数据均为二手/异构任务，仅作量级参考）

### 3.1 已知公开数据点

- **AlphaEvolve**（[DeepMind 论文 arXiv:2506.13131](https://arxiv.org/html/2506.13131v1) + [官方博客](https://deepmind.google/blog/alphaevolve-a-gemini-powered-coding-agent-for-designing-advanced-algorithms/)）：
  - FunSearch 需 **数百万 LLM 样本**，AlphaEvolve **数千样本即足**（论文原话对比）——进化收敛所需评估次数从 10^6 量级降到 10^3~10^4 量级。
  - matmul 相关战绩：4×4 复数矩阵乘 49（Strassen 递归一层）→ **48 次乘法**（~2% 改进，56 年来首次）；通用 GEMM 内核（Gemini Kernel）对最佳人工内核 **最高 +23% 实测加速**（大矩阵、GB200/H100）。
  - Borg 调度启发式：+0.7% 全球算力回收（生产运行 1 年+）——**成熟系统上典型改进是 个位数百分比**。
  - 约束（论文）：FunSearch 只进化单函数（10-20 行）且需快评估（≤20min/1CPU）；AlphaEvolve 可进化整文件、可并行加速器评估数小时。
- **OpenEvolve**（开源复现，[GitHub](https://github.com/algorithmicsuperintelligence/openevolve) + [HF 博客](https://huggingface.co/blog/codelion/openevolve)）：
  - circle packing n=26：Gen 0 ≈0.959 → **Gen 460/500 迭代** 匹配 AlphaEvolve 的 2.635（随机起点到 SOTA 需 ~500 迭代）。
  - 官方成本估算：**每迭代 $0.01-0.05**（Gemini-2.5-Flash）/ $0.08-0.30（Pro）/ $0.15-0.60（o3）；建议起步 **100-200 迭代**。
  - [GPU 内核进化实测](https://huggingface.co/blog/codelion/openevolve-gpu-kernel-discovery)：平均吞吐 +10.4% 但 **σ=30.7%**（部分候选大幅回退）——进化产出高方差。
  - 每迭代墙上时间含评估与模型调用，典型**分钟级**（mlx_metal_kernel_opt 分析：~10min/迭代含加载+基准）。
- **反例警示**（OpenEvolve [issue #100](https://github.com/algorithmicsuperintelligence/openevolve/issues/100)）：平行 best-of-N（无进化、只反复采初始程序）在 circle packing 上也有竞争力——「进化优于纯采样」并非无条件成立。

### 3.2 映射到 env_7（量级换算，C 级推断）

- 本任务 **评估极快**（bench ≤36s 起点、<1s 成熟候选）——比 AlphaEvolve 场景快 1-2 个数量级，同样 token 预算可支撑更多迭代。
- 从 naive（score 2e-5）到 row-dot（0.15）只需**一次向量化结构跳变**——属于「LLM 一步就能给出」的形态（domain_knowledge SOP 即写了该路径），预期 **个位数迭代内**出现首个合格精英（合格=过正确性门 + bench 出分）。
- 从 0.15 → ~1.0（BLAS 平价）与 **>1.0**（多进程/Strassen 绕 SME）难度陡增；参照公开数据（成熟系统个位数百分比、高方差），**>1.0 不应作为默认达标线**，是否设、设多少属 J1 裁决。
- token 成本换算：OpenEvolve 口径 $0.01-0.05/迭代（Flash 级）×数百迭代 ≈ **$10-50/百迭代** 量级；本机 glm-5.3（catpaw-proxy）成本结构不同，但「每迭代 1 次 LLM 调用 × 10²-10³ 迭代」的样本数量级可参考。

## R7 上报项

1. **M5 Max 核簇拓扑未确证**（6S+12P 的簇划分/SME 单元数无公开资料）：>1.0 的多进程理论上限（~4 TFLOPS）基于 18 核全 row-dot 假设，未实测多进程候选——影响 J1「是否存在 >1.0 空间」的精确判断（不影响 1.0 线本身）。
2. **进化 agent 首精英迭代数无同构公开数据**：公开系统（AlphaEvolve/OpenEvolve）任务与 env_7 差异大，§3.2 的「个位数迭代出首精英」是推断非数据——需 J5 预算裁决时按区间对待。

## Gaps

- 未实测「多进程分块候选」在本机的真实 score 上限（理论余量 ~2-4×，实际可能被 IPC 吃光）——若 J1 想设 >1.0 判据，建议先做一次 15 分钟的 spike 实测再定。
- 冷启动效应的根因（SME 协处理器冷态 vs 调度 QoS）未溯源到一手文档——不影响 score（比值口径），只影响绝对值解读。
- env_7 进化迭代实际 token 成本（glm-5.3 + 本 super-loop 的 prompt 结构）无先例数据——job_47 realpi 冒烟只验证链路未计量 token/迭代，J5 预算可先按「首小时观察再定」处理。

---
*复现总命令：`python3 -m venv /tmp/matmul_calib_venv && /tmp/matmul_calib_venv/bin/pip install -q "numpy>=1.24"`；后续命令见各节。原始 JSON 均为原文粘贴。*
