# 实验记录 · 迭代 10 · dispatch_target=np_dot|impl_strategy=m_halves

## 1. 背景与价值（为什么现在做这个实验）

- **现状读数**：本格 visit=2、无精英（首访实作）。目标格 profile（iter10 注入，@8192）：父本 `env/code/naive_start.py`（rowwise np.dot）runtime **11575ms**，热点 `<listcomp>` self 11510ms 占 **99.9%**、`stack` 9.5ms、`dot` 仅 1.1ms——瓶颈是 Python 逐行装配层（8192 次 gemv + B 全量重读），BLAS 内核只值 ~440ms，缺口 **26×**。
- **缺口（主要矛盾）**：本格定义形态「np_dot × m_halves」（M 维二分，每半单次 2-D np.dot 整阵派发）正是消除这 26× 的最短路径，且属容差约束下的合法族（「np.dot 2-D」直调系 + M 分解系）。
- **本 session 预检（research_report §1，决定性新事实）**：@8192 复刻 benchmark 协议 3 次——m_halves_np_dot 中位 **449.6 / 466.5 / 459.8ms**，ref 434.3 / 446.8 / 475.5ms，score **0.966 / 0.958 / 1.034**；`array_equal=True, maxdiff=0.0`（逐位一致）。同协议下父格精英 iter004（m_halves+matmul+out）同 session 复测 **480.5ms / score 0.9455**——**np.dot 入口比 np.matmul 入口在此形态下低 ~20-30ms**（numpy#18669：matmul 的 out= 慢路径，dot 无此坑）。iter007 遗留的「np.dot 578ms」为负载噪声（交错计时已排除入口开销）。
- **为什么值得占一轮预算**：预检已证数值零风险、落点明确（平价带中心）；本格补上后 np_dot 轴向证据闭环（rowwise 0.045 → m_halves ~0.96），且相对父格精英同 session 有 ~5% 可见优势可写入本格精英。近 4 轮连续 no_candidate 均为过程失败（turn-budget），本实验步骤极简、单变量，是高确定性的回血轮。

## 2. 实验假设（一句话，可证伪）

把 kernel 从「逐行 np.dot + np.stack」（11575ms）替换为 **M 维二分 + 每半单次 2-D `np.dot` 整阵派发（C 预分配 + 行切片 out= 直写）**后，@8192 runtime 塌缩到 BLAS 平价带——benchmark 中位 **430-490ms**（预检 449.6-466.5），score **0.90-1.05**（中心 ~0.96），且 max_error = **0.0**（与 np.matmul 锚逐位一致）；同 session 下 m_halves_np_dot 中位 ≤ m_halves_matmul（iter004 复测 480.5ms）。

## 3. 验证逻辑（什么结果支持假设、什么结果否定它）

- **支持假设**：`verify.py` passed=true 且 max_error=0.0（[64,8192] 双 seed 双跑逐位一致）；`benchmark.py` @8192 k=3 中位 ∈ [430, 490]ms、ref ∈ [380,700]ms 合理带、score ∈ [0.90, 1.05]（软门禁 ±5% 带内或紧邻）；自检 `--n 8192` max_error_vs_matmul = 0.0；profile 热点 `<listcomp>`/`stack` 完全消失。
- **否定假设**：中位 > 550ms（说明 np.dot 入口存在预检未见的开销，或 2-D dot 未走 gemm 派发）；或 max_error > 0（说明 K 求和序被改变——按 2411.00442 与死族清单，M 分解不动 K 序，发生即假设被否定）；或 score < 0.90（m_halves 惩罚超出平价带，与预检矛盾）。
- **对照**：父格精英 iter004（np_matmul|m_halves，档案 484.43ms / 本 session 复测 480.5ms）为同带 A/B 对照——同 session 下 cand 中位低于它即支持「np.dot 入口避开 matmul+out= 慢路径」（numpy#18669）的次级假设。

## 4. 具体实验步骤（可复现：改哪、怎么改、跑什么命令）

1. **落候选**：以 `codes/iter004-e77e66ca/solution.py`（父格精英）为父本复制到 `codes/iter010-be7d11f3/solution.py`，契约整体不动：`matmul(a,b)` 接口 / 嵌套 list→asarray f32 / `np.seterr` 三项抑制 / `main()` 的 `--n` 自检 JSON 协议（预热 1 次 + 计时 1 次 + `max_error_vs_matmul`）/ `_self_check()` 形状鲁棒用例表（research_report §4 第 1 条）。
2. **唯一 kernel 改动（单变量）**：两个半块 `np.matmul(a32[:mid], b32, out=c[:mid])` / `np.matmul(a32[mid:], b32, out=c[mid:])` 改为 `np.dot(a32[:mid], b32, out=c[:mid])` / `np.dot(a32[mid:], b32, out=c[mid:])`——`mid = m//2`、`c = np.empty((m,n), f32)` 骨架不变；docstring 更新为 np_dot 派发口径（引 numpy#18669 与预检数据）；main 中 impl 标签改 `"m_halves_np_dot"`（research_report §4 第 2 条）。不加转置、不切 K/N、无中间拷贝（§4 第 3 条，2411.00442）。
3. **自检（红绿）**：先跑 `_self_check()`（含逐位断言）见绿；再 `env/.venv/bin/python codes/iter010-be7d11f3/solution.py --n 64`（快速绿）与 `--n 8192`（预期 runtime ~450-470ms、max_error_vs_matmul 0.0）。
4. **正确性门（跑一次即记录）**：`env/.venv/bin/python env/skills/verify.py --candidate codes/iter010-be7d11f3/solution.py`——预期 passed=true、max_error=0.0。
5. **性能门（跑一次即记录）**：`env/.venv/bin/python env/skills/benchmark.py --candidate codes/iter010-be7d11f3/solution.py`——预期单行 JSON：n=8192、k=3、runtime_ms ∈ [430,490]、ref ∈ [380,700]、score ∈ [0.90,1.05]、无拒收（score ≤ 2.0、runtime ≥ 0.01ms）。
6. **对照判定（可选，一次）**：`env/.venv/bin/python env/skills/gate_check.py --candidate codes/iter010-be7d11f3/solution.py --baseline 484.42954199999997`——落 ±5% 软带内或优于基线即成立。**过程纪律：每个工具至多跑一次，读数即记录，不复跑、不追噪声**（research_report §4 第 5 条——近 4 轮 no_candidate 全是 turn-budget 耗尽型过程失败）。

## 5. 观测现象（预期指标与量级，与 gate_defs 同尺度）

- `--n 8192` 自检：runtime_ms **430-490**（预检 449.6-466.5），max_error_vs_matmul **0.0**；对照父本 11575ms（**~25× 塌缩**）。
- verify：passed=true，max_error **0.0**，双 seed 双跑逐位一致。
- benchmark：runtime_ms 中位 ∈ [430,490]、score ∈ **[0.90, 1.05]**（中心 ~0.96，预检 3 连 0.958-1.034）、ref_runtime_ms ∈ [380,700]。
- 机制生效信号：`<listcomp>`/`stack` 热点完全消失（kernel 只剩 2 次 BLAS 调用 + 1 次 empty）；同 session 对照下 cand ≤ 480.5（iter004 复测值）。机制失效信号：中位 > 550ms、或任何非零 max_error、或 score < 0.90（见第 3 段证伪线）。

## 6. 执行过程与结论

<!-- 占位：coding 追加「实现与测试」；debug 追加「卡点/根因/解决」；门禁结果由框架回填 -->

### 实现（coding）· 迭代 10 落盘于 2026-10-26

- **改动点（单变量 + 契约移植）**：
  1. 父本落位：`codes/iter004-e77e66ca/solution.py`（np_matmul|m_halves 父格精英）逐字复制为 `solution.py`；按计划补齐契约缺口——从 `env/code/naive_start.py`（iter003/iter007 同款模板）移植 `main()` 的 `--n` 自检 JSON 协议（预热 1 次 + 计时 1 次 + `max_error_vs_matmul`），impl 标签 `"m_halves_np_dot"`；`__main__` 顺序 = `_self_check()` → `main()`。
  2. **唯一 kernel 改动**：两半块 `np.matmul(a32[:mid], b32, out=c[:mid])` / `np.matmul(a32[mid:], b32, out=c[mid:])` → `np.dot(a32[:mid], b32, out=c[:mid])` / `np.dot(a32[mid:], b32, out=c[mid:])`；`mid=m//2`（奇数行安全）/ `c=np.empty((m,n),f32)` 骨架不动；无转置、不切 K/N、无中间拷贝（累加序约束）。
  3. docstring 更新为 np_dot 派发口径；`_self_check` 断言分级（常规块 K>=16 逐位 / 退化小形状 1-ULP 界 2.4e-07，iter005 先例——实际全部逐位，界未触发）。
- **检索记录（role=coding，≤3 来源，写入 solution.py「API 依据」）**：① numpy.dot 官方文档 v2.x：out 必须类型正确、C 连续、dtype 与 dot(a,b) 返回一致，不满足直接 ValueError；② GitHub #7124：dot out= 非 C-array 报错实证（memmap 案例）→ 行切片合法；③ GitHub #18669/#23588：matmul out= 慢路径（输出页未 fault-in）与 noblas 严苛检查 vs dot 入口宽容——本格入口不对称依据。
- **profile 热点确认（动手前，1 次）**：`<listcomp>` self 10918.7ms 占 99.9% / stack 8.5ms / dot 0.29ms，runtime 11017.5ms @8192——与注入记录一致，确认瓶颈在 Python 装配层（本实验改「怎么调 BLAS」正对瓶颈）。
- **TDD 记录（test_solution.py，stdlib unittest——venv 无 pytest；先在父本 iter004 副本上跑测试）**：
  - **红（3 例，失败信息均指向功能缺失）**：① `test_kernel_dispatches_two_np_dot_calls` → `AssertionError: 0 != 2 : expected exactly 2 np.dot dispatch calls (m_halves), got 0 — kernel not using the np.dot entry`；② `test_kernel_independent_of_matmul` → 父本 solution.py L47 触发炸弹 `np.matmul must not be called by the m_halves_np_dot kernel`；③ `test_main_json_protocol` → `no JSON line in output (main --n protocol missing?)`（父本只有 self-check 行）。
  - **回归护栏（3 例）**：bitwise @512 双 seed（42/123）/ 形状鲁棒六例 + 嵌套 list / 契约 dtype——父本上即绿（护栏性质，编码计划 §3 预先声明）。
  - **转绿关键改动**：kernel 两行 `np.matmul`→`np.dot` + `main()` 移植；改后 **6/6 全绿**（`Ran 6 tests ... OK`）；bitwise 断言全线 0.0（1-ULP 分级界未触发——dot-halves vs matmul 锚在 64/512/全部自检形状逐位一致）。重构仅 docstring/注释结构化，测试保持绿。
- **自检读数（每命令一次）**：`solution.py --n 64` → runtime 0.003ms / max_error 0.0；`--n 8192` → runtime **454.988ms** / max_error_vs_matmul **0.0**（对照父本 naive_start 11017ms（本次 profile 实测）/注入记录 11575ms ≈ **25× 塌缩**）。
- **门禁结果（每工具至多一次，读数即记录，未复跑）**：
  - **verify**：`{"passed": true, "max_error": 0.0, "sizes": [64, 8192], "seeds": {"64": [42, 1936951000], "8192": [42, 1963003914]}, "tolerance": 0.001}`——双规模 × 双 seed × 双跑逐位一致。
  - **benchmark**：`runtime_ms 463.4497499999999, score 0.9770405702020559, timings [463.45, 462.90, 464.72], ref_runtime_ms 452.809, n 8192, k 3`——IQR 1.8ms 极稳，无拒收。
  - **gate_check（--baseline 484.42954199999997）**：`passed=true, failure_stage=null`；value **453.444292**（timings 453.44/453.69/449.18）、score **0.973538645845387**、ref_value 441.4455；软门禁 verdict **"better"**（ratio 0.936038，±5% 带外更优；tolerance_band=false, exempt=false）。
- **与假设的对照：支持（全部判据命中，无证伪线触发）**：
  - 主判据：中位 453.44-463.45ms ∈ 预期带 [430,490]（三次独立测量 454.99/463.45/453.44）✓；score 0.9735-0.977 ∈ [0.90,1.05] ✓；verify max_error=0.0 + 自检 @8192 max_error=0.0 ✓；ref 441.45-452.81 ∈ [380,700] ✓；中位 <550 ✓。
  - 次级假设（numpy#18669：np.dot 入口避开 matmul+out= 慢路径）：同 session 同协议下 cand 453.44/463.45 < iter004 复测 **480.5ms**（-3.6%~-5.6%）✓；相对档案基线 484.43 ratio 0.936 带外更优（软门禁 "better"）✓。
  - 机制生效：结构测试钉死 kernel = 恰好 2 次 2-D np.dot + 预分配行切片 out= 直写（`out.base is c`），`<listcomp>`/`stack` 装配层彻底消失（父本 profile 99.9% 热点不复存在）✓。
  - np_dot 轴向闭环：rowwise 0.045（11575ms）→ m_halves ~0.97（453-463ms），与父格 m_halves（matmul 入口）同平价带且入口侧更优。
