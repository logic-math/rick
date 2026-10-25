# 实验记录 · 迭代 7 · dispatch_target=np_dot|impl_strategy=full_blas_dispatch

## 1. 背景与价值（为什么现在做这个实验）

- **现状读数**：本格首访（visit=1，无精英）。目标格 profile（ctx 注入，@8192）显示父本 `env/code/naive_start.py`（rowwise np.dot）runtime **11688ms**，热点 `<listcomp>` self 11506ms 占 **99.8%**，`stack` 12.9ms，`dot` 仅 1.0ms——瓶颈不在 BLAS，而在 Python 层 8192 次逐行循环 + 每行 gemv 全量重读 B + np.stack 装配。
- **缺口（主要矛盾）**：BLAS 内核本身只值 ~440ms（ref np.matmul 稳态带 434-484ms），父本却花 26×——本格定义形态「np_dot × full_blas_dispatch」（单次整阵 2-D np.dot 派发）正是消除这 26× 缺口的最短路径，且属于容差约束下的合法族（「np.dot 2-D」直调系）。
- **调研依据**（research_report §2-4 / §3）：1911.09421——2-D float32 下 np.dot 与 np.matmul 映射到同一 cblas_sgemm，换入口不换内核；本 session 预检（@1024/@8192）：`array_equal=True`、maxdiff **0.0**（逐位一致）；同进程交错 A/B 计时：np.dot 中位 **437.4ms** vs np.matmul **433.1ms**（±1%，噪声带内）。邻格精英 iter003（np_matmul|full_blas_dispatch，**430.83ms 全场纪录**）提供整阵派发的代码模板与契约保持方式。
- **为什么值得占一轮预算**：本格是 dispatch_target 轴上仅剩的未访入口（np_dot 直派形态），预检已证数值零风险、性能落点明确；产出预期与全场纪录同带（~434-488ms），可把该格写入精英并补全轴证据。

## 2. 实验假设（一句话，可证伪）

把 kernel 从「逐行 np.dot + np.stack」（11688ms）替换为**单次整阵 `np.dot(a32, b32)`**（2-D BLAS gemm 直派，K 求和序不变）后，@8192 runtime 塌缩到 BLAS 平价带——**中位 434-488ms（预检 437ms），score ~0.93-1.05**，且 max_error = 0.0（与 np.matmul 锚逐位一致）。

## 3. 验证逻辑（什么结果支持假设、什么结果否定它）

- **支持假设**：`verify.py` passed=true 且 max_error=0.0（[64,8192] 双 seed 双跑逐位一致）；`benchmark.py` @8192 中位 runtime ∈ [434, 488]ms（ref ∈ [380,700] 合理带），score ∈ [0.90, 1.05]（软门禁相对带 ±5% 内或紧邻）。
- **否定假设**：中位 runtime 仍 >1s（说明 2-D np.dot 未走 gemm 派发或存在意外装配开销）；或 max_error > 1e-3（说明 K 求和序被改变——按死族清单不该发生，发生即假设被否定）；或 score 显著 <0.9（说明 np.dot 入口有未预期开销，与预检矛盾）。
- **对照**：邻格精英 np_matmul|full_blas_dispatch（430.83ms，全场纪录）为同带对照；本 session 预检同进程交错 A/B（np.dot 437.4 vs np.matmul 433.1ms 中位）为入口开销对照（±1% 内视为等价）。

## 4. 具体实验步骤（可复现：改哪、怎么改、跑什么命令）

1. **落候选**：从 `env/code/naive_start.py` 出发，在 `codes/iter007-be5b45a7/solution.py` 产出候选；契约整体保持不动（`matmul(a,b)` 接口 / 接受嵌套 list 或 array / asarray→f32 / `np.seterr` 三项抑制 / main 的 warmup+计时+自检 JSON 协议）——照抄邻格精英 iter003 的契约保持方式（research_report §4 第 2 条）。
2. **唯一 kernel 改动**：删除逐行 listcomp + np.stack（父本 99.8% 热点），改为单次整阵派发并直接返回：
   ```python
   def matmul(a, b):
       """单次整阵 np.dot（2-D BLAS gemm）派发。逐位一致于 np.matmul 锚（同一 cblas 调用）。"""
       a32 = np.asarray(a, dtype=np.float32)
       b32 = np.asarray(b, dtype=np.float32)
       return np.dot(a32, b32)
   ```
   不加 out=、不预转置 B、不切 K/N、不引入中间拷贝（research_report §4 第 1 条；1911.09421 的映射结论保证 K 求和序不变）。main 中 impl 标签改为 `full_blas_dispatch_np_dot`。
3. **自检**：`env/.venv/bin/python codes/iter007-be5b45a7/solution.py --n 64`（快速绿）与 `--n 8192`（预检口径：warmup1 + 1 次计时；预期 runtime ~434-488ms、max_error_vs_matmul 0.0）。
4. **正确性门**：`env/.venv/bin/python env/skills/verify.py --candidate codes/iter007-be5b45a7/solution.py`——预期 passed=true，max_error=0.0（[64,8192] × 双 seed × 同输入双跑一致性全过）。
5. **性能门**：`env/.venv/bin/python env/skills/benchmark.py --candidate codes/iter007-be5b45a7/solution.py`——预期 stdout 单行 JSON：@8192、k=3 中位 ∈ [434,488]ms、ref ∈ [380,700]ms、score ~0.93-1.05、无拒收（score ≤ 2.0、runtime ≥ 0.01ms、无残留子进程）。
6. **（可选）融合判定**：`env/.venv/bin/python env/skills/gate_check.py --candidate codes/iter007-be5b45a7/solution.py --baseline 430.8255829999998`——与邻格精英对照，落在 ±5% 软带内即视为「np_dot 入口与 np_matmul 入口等价」的轴级证据。

## 5. 观测现象（预期指标与量级，与 gate_defs 同尺度）

- `--n 8192` 自检：runtime_ms ~**434-488**（预检 437.4），max_error_vs_matmul **0.0**；对照父本 11688ms（**~26× 塌缩**）。
- verify：passed=true，max_error **0.0**（逐位），双 seed 双跑一致。
- benchmark：runtime_ms 中位 ∈ [434,488]，score ∈ **[0.90, 1.05]**（中心 ~1.0，预检 np.dot 比 np.matmul 慢 ~1% 属噪声）；ref_runtime_ms ∈ [380,700]。
- 机制生效信号：`<listcomp>`/`stack` 热点完全消失（kernel 只剩单次 BLAS 调用）；机制失效信号：任何 >1s 中位或非零 max_error（见第 3 段证伪线）。

## 6. 执行过程与结论

<!-- 占位：coding 追加「实现与测试」；debug 追加「卡点/根因/解决」；门禁结果由框架回填 -->
