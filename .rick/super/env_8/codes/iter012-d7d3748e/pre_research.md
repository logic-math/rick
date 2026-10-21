# pre_research.md — 迭代 12 预研（einsum_path|full_blas_dispatch · builder 侧）

## 问题

父本 iter002（canonical 拼写 `einsum('ij,jk->ik', a, b, optimize=True)`，在库精英 7.2757ms）
把「canonical T-标志 gemm 优于 swapped C-标志」写进了实现选择。但其 ab_notes 自认幅度未证实
（仅 +0.04~0.07ms、7/10 对）。T/C 差是负载态相关的——本次预研在当前机器态下**方向翻转**。

## 证据链（2026-10-21 本机，env_8 venv numpy 2.0.2）

### 1. 进程内配对交叉（ab_micro.py，10 轮 × K=5 中位，随机次序）

| 变体 | 中位之中位 (ms) | Δ vs matmul | 解剖 |
|---|---|---|---|
| matmul | 6.662 | — | CC gemm 基线 |
| dot | 6.719 | +0.057 | CC gemm |
| tdot_CC | 6.758 | +0.096 | CC gemm + tensordot 包装 |
| **swapped** | **6.836** | +0.174 | CC gemm + tensordot + einsum 机器（无尾部跳变） |
| tdot_TT | 6.983 | +0.322 | TT gemm + tensordot 包装 |
| canonical（父本） | 7.007 | +0.345 | TT gemm + 尾部 ki->ik 视图 |
| ki_T（尾跳过） | 7.026 | +0.365 | TT gemm + .T 视图 |

- **TT−CC gemm 差 ≈ +0.23ms**（tdot_TT 6.983 vs tdot_CC 6.758，同包装同 C 序输出，
  纯标志差）；einsum Python 机器 ≈ 0.03-0.08ms（perf_profile：einsum_path 0.027ms、
  _parse 0.005ms、99.2% 时间在 tensordot 内）。
- 尾部跳过（ki_T）净值 −0.011ms（10 轮 5 胜 5 负）≈ 噪声级，不作为主变量。
- 全部 einsum 拼写与 np.matmul **逐位一致**（np.array_equal=True；tdot_TT 裸调需补 .T 才可比）。

### 2. 门协议直测（data/bench.py，warmup1 + 候选先计时 + K=3 中位，各 ×3）

| 变体 | R1 | R2 | R3 | 中位 |
|---|---|---|---|---|
| canonical（父本拷贝 pre/canon_cand.py） | 7.191 | 7.211 | 7.120 | **7.191** |
| **swapped（pre/swap_cand.py）** | 6.955 | 7.282 | 7.007 | **7.008** |
| ki_T（pre/ki_cand.py） | 7.093 | 7.130 | 7.395 | 7.130 |

- swapped 门中位比 canonical **快 ~0.18ms**；预研旧主张「CC 先计时在门协议下退化 7.5-8.6ms」
  **未复现**（R2 有 9.58ms 首计离群，K=3 中位稳在 7.28）。
- self_check（[64,2048]×双 seed、双跑一致）：三变体 max_error 均 = 0.0。

### 3. 机制（numpy 2.0.2 _core/einsumfunc.py 源码确认）

- swapped `einsum('jk,ij->ik', b32, a32, optimize=True)`：路径反转后 tmp_operands=[a,b]、
  einsum_str='ij,jk->ik' → `tensordot(a, b, axes=((1,),(0,)))` = **dot(a, b) 纯 C-标志 gemm**
  （与 np.matmul 同内核族）；tensor_result 'ik' == results 'ik' → **跳过尾部 c_einsum**，
  原生 C 序返回（无转置视图链）。
- canonical：`tensordot(b, a, axes=((0,),(1,)))` = dot(b.T 视图, a.T 视图) T-标志 gemm +
  尾部 'ki->ik' 转置视图（F 序输出）。
- K 求和序两者均与 np.matmul 一致 → 逐位一致（容差 1e-3 紧约束下安全）。

## 结论（实验设计输入）

单变量实验：**拼写 canonical(TT) → swapped(CC)**。预期 −0.1~−0.25ms，压过在库精英 7.2757。
风险：T/C 差随负载态漂移（父本 2 天前测得反向 +0.04-0.07ms）——若门协议下 swapped 中位
不优于 canonical，idea 即被证伪（回退父本拼写，如实记录）。

## 文件

- `ab_micro.py` — 进程内配对交叉脚本（可复跑）
- `pre/swap_cand.py` / `pre/ki_cand.py` / `pre/canon_cand.py` — 预研候选（只读参考，
  非交付物；交付物为本迭代目录 solution.py）
