# ab_notes.md — 迭代 9 A/B 归因（swapped einsum m_halves vs canonical halves / full）

- 候选：`solution.py` = M 维二分 swapped 拼写 einsum 派发 + concatenate
  （`r = np.einsum('jk,ij->ik', b32, a32[:h], optimize=True)` 上下半两块 + concat）
- 父本：iter008（m_halves 骨架）+ iter007（einsum M 分裂装配纪律，预研 canonical
  halves 基线 8.23ms）+ iter002（einsum 派发解剖，canonical/swapped 拼写差异）
- A/B 脚本：`ab_test.py`（进程内配对、每轮随机次序 10 轮、每变体 K=3 取中位；
  另含 concat 微基准与正确性 sanity；不改变提交实现）
- 环境：env_8 venv numpy 2.0.2，macOS Accelerate/vecLib BLAS，@2048×2048 f32。
  本次测量横跨两个负载态：轻载 load avg ~4-5 与重载 ~5.0-5.3（共享机，
  框架并行迭代导致负载漂移——归因结论按负载态分层报告）。

## 正确性 sanity（ab_test.py 内联，3 次运行全过）

- swapped halves == np.matmul **逐位一致 @2048**（max_error=0.0），双跑逐位一致
- 布局断言实测：swapped 块 **C 连续**（C-标志 gemm）、canonical 块 **F 布局转置
  视图**（T-标志 gemm）——本实验机制主张成立
- 嵌套 list 输入正常（float32 输出）；奇数 M（65 方阵 + 101×37 非方阵）
  max_error=0.0；m=1/2/3 退化块 max_error ≤1.2e-7；m=0 不崩溃
- 官方自检门（env skills/self_check.py）：passed，max_error=0.0，
  sizes [64,2048] × 双 seed，双跑逐位一致

## 配对归因矩阵（@2048 中位之中位，10 轮 × K=3，随机次序）

预研（plan 注入，轻载门态）vs 本次三跑（load avg 4-5 / 5.3 / 5.2）：

| 变体 | 预研 | 跑1（轻载） | 跑2（重载） | 跑3（重载） |
|---|---|---|---|---|
| full canonical einsum（iter002 路径） | 7.44 | 7.150 | 7.164 | 7.079 |
| full swapped einsum | 7.59 | 6.818 | 6.980 | 6.928 |
| canonical 2×半块 gemm（T-标志，无包装无组装） | 7.58 | 7.326 | 7.117 | 7.019 |
| swapped 2×半块 gemm（C-标志，无包装无组装） | 7.44 | 7.450 | 7.847 | 7.364 |
| einsum2 swapped（两块 einsum 无 concat） | — | 7.588 | 7.942 | 7.474 |
| canonical halves + concat（iter007 基线形态） | 8.23 | 8.168 | 8.139 | 8.094 |
| **swapped halves + concat（候选）** | ~8.0 | **8.058** | **8.404** | **8.016** |
| concat 微基准：C+C 块 | 0.234 | 0.247 | 0.239 | 0.240 |
| concat 微基准：F+F 转置视图 | 0.648 | 0.750 | 0.736 | 0.751 |

配对逐轮差（canonical halves − 候选，正值=候选快）：

| 跑 | 中位差 | 候选更快轮数 |
|---|---|---|
| 跑1（轻载） | +0.11 ms | **10/10** |
| 跑2（重载） | −0.27 ms | 1/10 |
| 跑3（重载） | +0.08 ms | 9/10 |

## 归因分解（vs canonical halves 基线，两因子）

1. **装配因子（鲁棒，3/3 跑 + 预研全部复现）**：swapped 块天然 C 连续 →
   concatenate 退化为纯 2×8MB 块 memcpy（0.24ms），canonical 的 F 转置视图
   拼接要 0.74-0.75ms——**结构性节省 ~0.50ms**，跨负载态稳定。
   三段式验证（跑1/跑3）：gemm2_swapped 7.45/7.36 + einsum 包装 ~0.14 +
   concat 0.24 ≈ 7.8-7.9，与候选实测 8.02-8.06 吻合（余项为块刚写完即拼的
   cache 态差）；canonical 侧 7.02-7.33 + 包装 + concat 0.75 ≈ 8.1 ✓。
2. **gemm 标志因子（负载态脆弱，预研未复现）**：预研主张 M=1024 半块规模
   C-标志 gemm 反超 +0.14ms；本次三跑 canonical T-标志 gemm 反而全部持平或
   更快（−0.12/−0.73/−0.35ms），重载态 C-标志路径劣化明显（跑2 gemm2_swapped
   7.85ms 离群）。**「拼写选择随 M 规模翻转」的 gemm 项在本机不鲁棒**——
   只在装配项不被 gemm 劣化吃掉时净赢（跑1/跑3 净 +0.08~+0.11ms，
   跑2 净 −0.27ms）。

## 官方口径复跑（env/data/bench.py，K=3，warmup1）

轻载段 4 跑 + 重载段 5 跑（同一候选、同一会话，负载态漂移）：

| 轮 | runtime_ms | score | ref_runtime_ms | ref ∈ [5,10] |
|---|---|---|---|---|
| 轻1 | 8.303 | 0.884 | 7.336 | ✓ |
| 轻2 | 8.101 | 0.864 | 6.996 | ✓ |
| 轻3 | 8.045 | 0.964 | 7.754 | ✓ |
| 轻4 | 8.143 | 0.882 | 7.185 | ✓ |
| 重1 | 8.475 | 0.868 | 7.353 | ✓ |
| 重2 | 8.473 | 0.849 | 7.192 | ✓ |
| 重3 | 8.742 | 0.773 | 6.757 | ✓ |
| 重4 | 8.656 | 0.812 | 7.033 | ✓ |
| 重5 | 8.742 | 0.783 | 6.846 | ✓ |

- 轻载段中位 **8.12 ms**（区间 8.05-8.30），重载段中位 **8.74 ms**
  （区间 8.47-8.74）；self_check（k=1）7.976ms。9 轮 ref 全落 [5,10] 合理带，
  无 plausibility 拒收。
- 预期带 7.9-8.1ms 在轻载态命中（self_check 7.976 / 轻3 8.045）；
  重载态整体抬升 ~0.5-0.7ms（ref 同步漂移，score 稳定 0.77-0.96 在
  预期 0.84-1.07 邻域）。
- 正确性门官方预验：passed=true，max_error=0.0，sizes [64,2048]，
  seed 42+urandom，双跑一致。

## 结论

1. **谱系数据点落位**：einsum M 粒度阶梯（@2048 f32，本机）确认为
   full ~7.1-7.3 → **halves ~8.0-8.4（本格）** → quarters 9.3-9.4——
   单调劣化，M 方向继续细分无望；本格补全 halves 数据点。
2. **可复用结论（鲁棒部分）**：swapped 拼写（'jk,ij->ik', (b, a_blk)）的真正
   价值在**装配**——C-标志 gemm 产出 C 连续块，concatenate 退化为纯
   2×8MB memcpy，比 canonical 的 F 转置视图拼接**结构性省 ~0.5ms**
   （4/4 测量一致，含预研）。任何「块结果要拼接」的 einsum 路线都应
   优先 swapped 拼写。
3. **可复用结论（脆弱部分，修正预研）**：gemm 标志项（C vs T）在 M=1024
   半块规模的优劣是**负载态敏感**的——预研 swapped +0.14ms 优势本次
   3/3 跑未复现（持平或反向最多 −0.73ms）。「2048 行 canonical 快、
   1024 行 swapped 反超」的规模翻转主张应降级为「装配项确定省、
   gemm 项负载态漂移 ±0.2-0.7ms」。
4. **死族远离声明**：未用 einsum out= / C 缓冲逐块直写 / K 分块 / Strassen /
   转置后 gemv / 降精度 / 多进程 / 多线程；无模块级缓存或状态；
   契约（matmul(a,b)、asarray f32、seterr、嵌套 list、--n 自检 main）
   与父本一致。
