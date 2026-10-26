# 实验记录 · 迭代 17 · dispatch_target=np_dot|impl_strategy=full_blas_dispatch

## 1. 背景与价值（为什么现在做这个实验）

- **现状读数**（ctx 注入的本迭代 iter17 profile，@8192）：父本 `env/code/naive_start.py` runtime **11192.5ms**，`<listcomp>` self 10400.1ms 占 **99.8%**、`stack` 10.1ms、`dot` 1.23ms——瓶颈不在 BLAS（内核本身值 ~440ms），在 Python 层 8192 次逐行循环 + B 全量重读 + stack 装配。
- **缺口**：本格定义形态「np_dot × full_blas_dispatch」= 单次整阵 2-D `np.dot(a32, b32)` 直派 cblas_sgemm，是消除这 ~25× 缺口的最短合法路径（直调系合法族，保持 BLAS K 求和序）。
- **同格历史（visit=1，iter007）**：同一设计已预检全绿（np.dot 437.4ms vs np.matmul 433.1ms 同进程交错 ±1%；@8192 逐位 maxdiff=0.0），solution.py 已落盘，但三段 agent 全死于 turn-budget → verdict=no_candidate（**过程失败，非性能被否**——archive description 明文）。本轮 visit=2 的核心价值 = **流程修复重跑**：一行内核走完 verify→benchmark→gate 链，产出被 admit 的候选、写入本格精英，补全 dispatch_target=np_dot 轴证据。
- **预算语境**：17/17 末轮——设计从简（调研 3 来源封顶），plan_steps 钉死最小动作链（复制→自检→每工具一次），把预算留给 coding 与门禁。
- **调研依据**：research_report §2（numpy#18669 / SO 交错证据——plain dot/matmul 入口差异在噪声带内）、§3（arXiv:2411.00442——累积序决定逐位结果，同路径即同结果；GEMMbench——warm-up+中位+交错的测量协议）、§4（迁移结论 1-4 条）。

## 2. 实验假设（一句话，可证伪）

把 kernel 从「逐行 listcomp + np.stack」（11192ms）替换为**单次整阵 `np.dot(a32, b32)`**（2-D BLAS gemm 直派，K 求和序不变）后，@8192 中位 runtime 塌缩到 BLAS 平价带 **434-488ms（预检 437.4ms，~25× 塌缩）**、score ~0.93-1.05，且 max_error = 0.0（与 np.matmul 锚逐位一致过 1e-3 门）。

## 3. 验证逻辑（什么结果支持假设、什么结果否定它）

- **支持假设**：`verify.py` passed=true 且 max_error=0.0（[64,8192] × 双 seed × 双跑逐位一致）；`benchmark.py` @8192 k=3 中位 ∈ [434,488]ms（ref ∈ [380,700] 合理带）、score ∈ [0.90,1.05]（±5% 软带内或紧邻）；自检 `--n 8192` runtime ~434-488ms、max_error_vs_matmul 0.0。
- **否定假设**：中位 >1s（2-D dot 未走 gemm 派发或有装配开销残留）；或 max_error > 1e-3（K 求和序被改——死族清单预测不该发生，发生即假设被否定）；或 score < 0.9（np.dot 入口有未预期开销，与 iter007/iter010 预检矛盾）。
- **对照**：邻格精英 np_matmul|full_blas_dispatch **430.83ms（全场纪录）**为同带对照（落在其 ±5% 软带内 ⇒ np_dot 与 np_matmul 入口等价的轴级证据）；iter007 同进程交错预检（dot 437.4 vs matmul 433.1ms）为入口开销对照。
- **过程性失败线（非假设否定）**：若本轮再次 no_candidate（turn-budget/流程断裂），说明流程修复失败，假设保持未检验状态——须在 §6 如实记录。

## 4. 具体实验步骤（可复现：改哪、怎么改、跑什么命令）

1. **落候选**：以 `codes/iter007-be5b45a7/solution.py`（同格上轮已写好的一行内核，未过门）为父本，产出 `codes/iter017-9d0b9598/solution.py`：kernel 保持 `return np.dot(a32, b32)`（无 out=、不预转置、不切 K/N——research_report §4 第 1 条）；契约照 iter003 模板（`matmul(a,b)` 接口 / 嵌套 list 兼容 / asarray→f32 / seterr 三项抑制 / main 的 warmup+计时+JSON 协议，impl 标签 `full_blas_dispatch_np_dot`）；移植 iter010 的 `_self_check` 用例表（方阵/奇数行/非方阵/M=1/M=0/嵌套 list，K≥16 逐位、退化形 ≤1 ULP）。
2. **快速自检**：`env/.venv/bin/python codes/iter017-9d0b9598/solution.py --n 64`（绿）→ `--n 8192`（预期 runtime ~434-488ms、max_error_vs_matmul 0.0）。
3. **正确性门（一次）**：`env/.venv/bin/python env/skills/verify.py --candidate codes/iter017-9d0b9598/solution.py` → 预期 passed=true、max_error=0.0。
4. **性能门（一次）**：`env/.venv/bin/python env/skills/benchmark.py --candidate codes/iter017-9d0b9598/solution.py` → 预期 @8192、k=3 中位 ∈ [434,488]ms、score ~0.93-1.05、无拒收（score ≤ 2.0、runtime ≥ 0.01ms、ref 在带内）。
5. **（可选一次）融合判定**：`env/.venv/bin/python env/skills/gate_check.py --candidate codes/iter017-9d0b9598/solution.py --baseline 430.8255829999998` → 落 ±5% 软带内即「np_dot 与 np_matmul 入口等价」轴级证据。
6. **读数即记录**：上述每个工具至多跑一次，读数立即写入本记录 §6；不复跑门禁（iter007 根因修复，archive description 过程纪律）。

## 5. 观测现象（预期指标与量级，与 gate_defs 同尺度）

- `--n 8192` 自检：runtime_ms ~**434-488**（预检 437.4），max_error_vs_matmul **0.0**；对照父本 11192ms（~25× 塌缩）。
- verify：passed=true、max_error **0.0**（逐位），双 seed 双跑一致。
- benchmark：runtime_ms 中位 ∈ [434,488]，score ∈ **[0.90,1.05]**（中心 ~1.0；预检 np.dot 比 np.matmul 慢 ~1% 属噪声带），ref_runtime_ms ∈ [380,700]。
- 机制生效信号：`<listcomp>`/`stack` 热点完全消失（kernel 只剩单次 BLAS 调用）；失效信号：任何 >1s 中位或非零 max_error（证伪线见第 3 段）。

## 6. 执行过程与结论

<!-- 占位：coding 追加「实现与测试」；debug 追加「卡点/根因/解决」；门禁结果由框架回填 -->

## 6. 执行过程与结论

### 实现（coding）
- **改动点**（coding_plan.md 已落盘）：
  - `solution.py` kernel（唯一实质改动）：`np.stack([np.dot(a32[i], b32) for i in range(a32.shape[0])])` → `return np.dot(a32, b32)`——单次整阵 2-D cblas_sgemm 直派；**无 out=**（numpy#7124：out 须类型/维数/C 连续精确匹配）、**不预转置 B**（转置改求和序，@8192 误差 ~1.4e-2 超容差）、**不切 K/N、不做 1-D 化**（numpy#13732：gemm/gemv 派发由 numpy 决定，父本逐行 1-D gemv 即慢路径根源）。
  - 契约零改动（iter003/iter007 模板）：`matmul(a,b)` 接口 / asarray→f32 / seterr 三项抑制 / main warmup+计时+JSON 协议；impl 标签 `full_blas_dispatch_np_dot`；移植 iter010 `_self_check` 用例表（方阵/奇数行/非方阵/M=1/M=0/嵌套 list；K≥16 逐位、退化形 ≤1-ULP）。
  - `test_solution.py`：12 项单元测试（结构判据 2 + 边界 1 + 数值 6 + 契约 3）。
  - profile 不复跑（ctx 注入 iter17 profile 已锁定 listcomp 99.8% 热点；plan_steps 最小动作链）。
- **TDD 记录**：先 `cp naive_start.py solution.py`（父本基线就位）→ 跑测试 **12 项中 7 红**（均为断言失败、非报错）：
  - `test_single_2d_dot_call`：父本派发 64 次 1-D np.dot（要求恰 1 次双 2-D）；
  - `test_no_stack_assembly`：np.stack 被调用；
  - `test_zero_rows`：np.stack([]) 抛 ValueError（M=0 不合法）；
  - `test_bitwise_square_64/128/odd_rows_nonsquare + test_f64_input_converted`：rowwise gemv 与 gemm 累积差异 7.6e-06/2.3e-05（父本非逐位）。
  - 换 kernel 后 **12/12 绿**；重构仅限 docstring/自检表整理，测试保持绿。
- **门禁结果（每工具恰一次，读数即记录，不复跑）**：
  - 自检：`--n 64` 0.002ms/err 0.0；`--n 8192` 单发 598.044ms / max_error_vs_matmul=**0.0**；`_self_check` 用例表全绿。
  - **verify**：passed=true，**max_error=0.0**（[64,8192] × seed42+urandom seed × 双跑逐位一致；tol 1e-3）。
  - **benchmark**：@8192 k=3 中位 **598.587ms**（timings [609.1, 598.6, 582.0]）、**score=1.0360**、ref=**620.163ms**（带内 [380,700]）——无拒收。
  - **gate_check**（plan step5，--baseline 430.8255829999998 邻格精英轴证据跑）：correctness passed（max_error=0.0）、perf value=589.024 / score=1.0371 / ref=610.892，**soft verdict="worse"（ratio=1.367 > 1.05 带）→ passed=false, failure_stage=soft_gate**。本格无精英（elite=null）→ 框架终验按首解豁免语义裁决，以框架覆盖为准。
- **与假设的对照：部分支持（机制全部兑现；绝对带未落由机器态漂移主导，非设计失效）**
  - ✓ 机制生效：结构测试证 kernel 恰 1 次 np.dot 双 2-D、无 stack 装配；11192.5ms → 598.6ms（**18.7× 塌缩**）；父本热点 `<listcomp>`/`<stack>` 消失。
  - ✓ 逐位一致：max_error=0.0 过 1e-3 门（K 求和序不动）——验证逻辑的「否定线」（err>1e-3）未触发。
  - ✓ 入口等价（同进程交错）：score=1.036/1.037——np.dot 比 np.matmul **快 ~3.6%**，与 iter007 预检（437.4 vs 433.1ms）方向一致；「score<0.9 否定线」未触发。
  - ✗ 绝对带 [434,488] 未落：本轮 session 机器整体慢 ~40%——同进程 ref np.matmul 620.2/610.9ms（对 iter003 精英 430.83 所在 session ~433ms 漂移 **+41%**），候选随机器态同步漂移。非 gemm 派发失败（score>1 已排除入口开销；中位 598.6 < 1s 证伪线未触发）。
  - 软门带外劣化（ratio 1.367）是**跨 session 机器态比较**的读数：ref 自身同幅漂移，同进程对照（score>1）才是入口级证据。
  - 过程性失败线未触发：三工具各恰一次跑完、候选已产出——**iter007 的流程断裂（no_candidate）已修复**，假设不再停留于未检验状态。
- **coding 角色检索落地（≤3 来源，已写进 solution docstring 与 coding_plan §2）**：numpy.dot 官方文档（both 2-D = matrix multiplication，同 BLAS gemm 路径）；numpy#7124（out= 非弹性 C-array 约束 → 选 plain 调用）；numpy#13732（gemm/gemv 派发由 numpy 内部决定 → 禁 1-D 化）。

### 卡点与解决（debug · attempt 1/3 · 触发：自检失败 @ gate_check/soft_gate）

- **触发类型**：自检失败——coding 的 plan step5 可选轴证据跑（`--baseline 430.8255829999998` 邻格精英）soft verdict="worse"（ratio 1.367）。本格 elite=null，待判定：实现的锅，还是测量/环境的锅？
- **复现方式与关键读数**（21:25-21:27，一批串行跑完，≥2 次读数）：
  - **机器态快照**：load **6.69/6.90/6.96**（复验时 7.35）；后台 `winedevice.exe` ×2（169.6%+105.6% CPU）+ `steamwebhelper`（93.1%）+ `wineserver`（70.5%）≈ **439% CPU 抢占**（CrossOver/Wine + Steam）；`pmset -g therm` 无热警告。Apple M5 Max 18 核。
  - **同进程交错微基准 @8192**（独立进程 ×2、各 5 rep、warm-up 后）：run1 dot 中位 **604.3ms** / matmul **611.4ms**（matmul/dot=1.0117）；run2 dot **600.1** / matmul **593.3**（0.9887）——两入口 ±1.2% 噪声带内，两次 `array_equal=True`（逐位）。
  - **gate_check 复现**（--baseline 430.83）：passed=false @ soft_gate，value **566.98ms**、同进程 ref **592.29ms**、**score=1.0446**、ratio **1.316**（与 coding 段 589.02/610.89/1.367 同型；跨时间 gate 值 589.02→566.98，机器态波动中）。
  - **关键事实**：同进程 ref np.matmul 自身 586.9-620.2ms，比基线 session 的 ~433ms（430.83 所在）漂移 **+35~43%**；候选与 ref 同幅漂移（score 摆动 0.987-1.045，中心 ~1.0）。
- **三个假设与验证结论**：
  1. **跨 session 机器态漂移（成立，主因）**：若成立应观察到 ref 自身同幅漂移 + 机器高负载 + 候选随 ref 同步漂移——实测全部命中（ref +37~41% vs 433ms；load ~7 + 439% 后台 CPU；score 摆动中心 ~1.0）。
  2. **候选实现劣化（不成立）**：若成立应观察到同进程交错 dot 比 matmul 慢 >5% / score<0.95 / max_error>1e-3——实测 ±1.2% 噪声带、bitwise equal ×2、verify max_error=0.0；numpy.dot 官方文档证 2-D×2-D 即 BLAS gemm 矩阵乘（"matmul preferred" 为风格推荐非性能差异；#18669 入口差异仅在 out= 未 fault-in 场景，本候选 plain 调用不涉）。
  3. **计时协议伪影（不成立，非主因）**：若成立应观察到首跑离群 >15% / warmup 失效——实测 gate timings 展宽仅 4.5-4.6%（[609.5,594.8,593.6]/[567.0,579.5,553.3]）、微基准 warm-up 后 reps 展宽 ±3.6%、无首跑离群；读数间漂移（589→567ms）与机器态波动（load 6.69→7.35）相关，归因 H1。
- **根因**：**测量环境，非实现**——本 session 后台负载（Wine/Steam ≈439% CPU）令全部 SGEMM 读数系统性慢 ~35-41%（等效 1.86 vs 2.54 TFLOPS）；soft-gate 的 --baseline 430.83 是邻格精英在无负载 session 的跨状态读数，属跨 session 比较伪影。方法论依据：NIST TN 1830（"The ghost in the machine"——共享主机争用破坏 i.i.d. 假设）、tak / zenbench 方法论（跨状态顺序比较失效，同进程交错配对比较才有效——即 benchmark 的 score≈1 才是有效读数）；numpy.dot 官方文档（2-D 用法正确性）。
- **处置**：**keep_original——零代码改动**。候选已处 BLAS 平价（同过程交错 ±1.2%、score 摆动 0.987-1.045 中心 ~1.0）、逐位一致（max_error=0.0）、无 API 误用（plain 2-D np.dot 为官方文档形态）；绝对带 [434,488] 的偏离 100% 由机器态解释——BLAS-bound 内核不存在把 567ms 改到 430ms 的合法代码改动，改门禁/基线/计时设施属作弊，拒绝。
- **复验结果（gate_check 读数）**：
  - **规范终验（无 --baseline，本格 elite=null → 首解豁免）**：**passed=true**（exit 0），soft verdict="exempt"，correctness max_error=**0.0**（[64,8192] 双 seed）、perf value **594.84ms**（timings [609.5, 594.8, 593.6]）、score **0.9867**（同进程 ref 586.90ms）。
  - 复现跑（--baseline 430.83）：passed=false @ soft_gate（ratio 1.316）——与机器态一致（ref 592.29ms 自身 +37% vs 基线 session）。
  - 结论：软门「worse」为跨 session 机器态比较伪影，非实现劣化；候选按本格规范终验**通过**。「np_dot vs np_matmul 入口等价」轴证据由同进程交错 ±1.2% + score 中心 ~1.0 支撑，绝对带 [434,488] 待无负载 session 复测坐实（已记录，不阻塞本格结论）。
