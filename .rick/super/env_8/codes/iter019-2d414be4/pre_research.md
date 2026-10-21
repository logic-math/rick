# iter019 预研笔记：einsum_path|m_halves 格全设计空间分解（builder）

日期：2026-10-21（iter019 构建期）· 父本：codes/iter013-0a966897（门中位 7.83875ms，score 0.857）

## 一、父本成本分解（@2048 f32，进程内稳态中位，多轮）

| 分量 | 耗时 | 测法 |
|---|---|---|
| 2× 半阵 gemm（swapped-NN einsum） | 7.46（快负载日）/ 7.57-7.64（今日） | V1：只算不拼 |
| 装配（预分配 + 2×8MB 趁热 memcpy） | 0.24 | V2：预生成块单独计时 |
| 交互项（L2 b 逐出等） | 0.19（快日）/ 0.34（今日） | V0 − V1 − V2 |
| **合计** | **≈ V0 实测 7.89-8.22** | 负载态漂移 ±0.2 |

交互项归因（D1/D2 诊断）：
- gemm2 前有 copy1（fresh c）：gemm2 = 3.951 vs 无拷贝间隔 3.839（+0.112）
- gemm2 前有 copy1（persistent c）：3.912（+0.073）→ **b 逐出为主，缺页为辅**
- copy→fresh c 0.121 vs copy→persistent c 0.118 → **稳态下缺页近零（malloc 复用已映射大块）**

## 二、设计空间逐族裁决（全部进程内实测，≥3 轮配对）

| 族 | 变体 | 结果（vs 父本） |
|---|---|---|
| 拼写 | canonical 'ij,jk->ik' (a_blk,b)：TT gemm，输出 **F 连续视图** | **gemm-only 7.35（−0.21，今日 T/C 漂移态）**；但端到端 +0.14~+0.22（装配税） |
| 拼写装配 | canonical + F 序预分配交错拷贝（W2） | +0.25~+0.47 ✗ |
| 拼写装配 | canonical + concat((r1.T,r2.T),axis=1).T（W8，最优形式） | **+0.14~+0.22 ✗（今日 4 轮新鲜配对一致）** |
| 拼写装配 | canonical + C 序散布拷贝（W5）/ einsum order='C'（W6） | +4.7~+5.1 ✗✗ 死 |
| 装配 | concatenate（iter009 形态）/ concatenate(out=c) / 冷拷贝序 / zeros | +0.05~+0.15 ✗ |
| 次序 | X0（gemm 邻接+热拷2+冷拷1）/ X50（四分交错）/ 反转 / 晚分配 c | +0.05~+0.13 ✗（X0 四轮一致 +0.11） |
| 包装 | list-form 操作数 / 显式 path | ±0.02 持平（源码核验：2 操作数无搜索；parse 仍执行） |
| 死路 | einsum out=（c_einsum 恒等拷贝 0.41ms/8MB，numpy 2.0.2 源码 1460-1496 行确认）/ 模块级缓冲（返回复用缓冲=正确性地雷+作弊邻域）/ F 序 4KB 段拷贝地板 0.71（nditer 每段开销，无 C 扩展不可破） | ✗ |

**结论：父本 = 格内全构造空间的最优点。格地板 ≈ 7.75-7.95（负载态依赖），
对 np_dot|m_halves（7.42）的结构性差距 = einsum 包装税（无可用 out=）+ 装配 memcpy + L2 交互。**

## 三、唯一有实质未决信息量的方向：拼写轴的 T/C 漂移

- 今日负载态：canonical-TT 半阵 gemm-only **快 0.21ms**（3 轮×11 次中位稳定）
- 历史：iter013 预研日反向（swapped 快 0.14）；iter012（full 格）swapped 快 0.18（门）
  → T/C 差随负载态漂移，幅度历史范围 ±0.25ms
- canonical 端到端翻盘条件：T/C 差 > 0.47ms（装配税）——**历史幅度的 2.2 倍，未见先例**
- 拼写轴（swapped vs canonical）从未在本格门禁裁决过；iter012 的 ab_notes 明示此风险开放

## 四、本迭代实验（iter019）：canonical-TT + concat-T 装配的门禁裁决

- 变体（W8，已验证）：
  ```python
  r1 = np.einsum('ij,jk->ik', a32[:h], b32, optimize=True)   # TT gemm，(h,N) F 连续视图
  r2 = np.einsum('ij,jk->ik', a32[h:], b32, optimize=True)
  return np.concatenate((r1.T, r2.T), axis=1).T               # (N,M) C 连续装配 + .T 视图
  ```
- 正确性（门协议模拟全过）：64/2048 × 双 seed max_err=0.0；双跑逐位一致；嵌套 list / 奇数形状 ✓；
  返回 F 连续数组（值语义完全合法）
- 预期：端到端 +0.14~+0.22 劣化 → 大概率证伪 → **拼写轴闭合 + 格地板归档**（父本已证
  swapped 的 C 连续输出端到端占优，T/C 漂移收益上限不敌 F 序装配税）
- 尾部机会（~3%）：门日负载态 T/C 差突破 0.47 → 新精英 + 装配范式改写

## 附：本迭代预研脚本（codes/iter019-2d414be4/）

pre_bench.py（V 族分解）/ pre_bench2.py（机制核验+拼写 AB）/ pre_bench3.py（W 族全管线）/
pre_bench4.py（X 族次序）/ pre_bench5.py（D 族交互归因）/ pre_bench6.py（终验：正确性+新鲜配对）
