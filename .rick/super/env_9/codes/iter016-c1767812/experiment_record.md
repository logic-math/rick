# 实验记录 · 迭代 16 · dispatch_target=np_matmul|impl_strategy=m_halves

## 1. 背景与价值（为什么现在做这个实验）

- **现状**：本格精英 = iter004（484.43ms，2026-10-24 安静窗 admit；同批整阵 full_blas_dispatch 430.83ms）→ m_halves 结构罚项 ≈ +54ms / +12.4%（同窗实测）。iter16 profile：`matmul` self 100%、无 Python 层热点——父本结构干净，剩余改进空间只在**执行结构**（两次半阵 gemm 怎么跑），不在代码微优化。
- **缺口**：iter004 的 experiment_plan.json 是 turn-budget-exceeded 残片（plan_steps 空）——本格首访后从未有过成型实验；预算已尽（17/17），这是本格最后一次机会，必须选**唯一有实测增益的未探变体**。
- **证据（builder 探针，2026-10-26 21:10，n=8192 seed=42，7 臂交错 5 轮，全臂 bitwise == 整阵 np.matmul 锚，脚本 `codes/iter016-c1767812/_probe_variants.py`）**：
  - A 整阵 589.2 / **B 父本（顺序两半 out=）602.4** / C np.dot 两半 586.0 / D concat 606.7 / E 持久缓冲 607.0 / **F 并发两半 498.1** / G 堆叠 gufunc 625.8（ms，负载窗 load≈6.9/18）。
  - 官方 benchmark 同窗复测父本 607.9（ref 593.3）——与探针 B 一致。
  - **F = -104ms / -17% vs B（分布无交叠：F 490-515 vs B 592-621），且同窗低于整阵 A 91ms**；F 重叠成立证明 np.matmul 释放 GIL 且 Accelerate 允许并发 sgemm 重叠（OpenBLAS#4741 所述「单调用独占线程池」罚项在本库不存在）。
  - F 的 worker 线程出现 vecLib 虚假 FP 警告 → np.seterr 线程局部，worker 内需先 seterr（数值 bitwise 正确）。
- **已证伪勿重探**（iter13/14 + 本探针）：持久缓冲/预 fault/out= 变体（E=607.0 与 B 平价）；concat（D 劣）；堆叠 gufunc（G 劣 625.8）；线程数收紧（iter13 恒劣）；换 dtype/切 K/转置（容差墙，死族清单）。

## 2. 实验假设（一句话，可证伪）

把 m_halves 的两个半阵 `np.matmul(..., out=c[...])` 由**顺序执行**改为**两个 Python 线程并发执行**（worker 先 seterr，join 后返回 c），门禁 benchmark 中位 **< 484.43ms**（本格精英；安静窗预期 425-475ms），且同窗读数不高于整阵 ref。

## 3. 验证逻辑（什么结果支持假设、什么结果否定它）

- **支持**：verify 全过（bitwise max_error=0.0、双 seed 双规模、同输入双跑逐位一致）**且** benchmark 中位 < 484.43。**强支持** = 中位 < 同窗 ref（并发重叠使 m_halves 吞吐达到/超过整阵——探针窗 F 498 < A 589 的同型现象）。
- **否定（机制证伪）**：中位 ≥ 484.43 **且** ref ∈ 稳态带（380-460）→ 安静窗 2×并发 gemm 过订阅反噬（ThreadPinning 警示的同型风险），并发收益不迁移，如实 reject。
- **否定（窗口所限，非机制失败）**：中位 ≥ 484.43 **且** ref ≥ 550（负载窗）→ 记录 reject 但在 §6 附 ref 读数判窗，标注「窗口噪声，机制证据见探针」。
- **对照**：探针同窗 B（父本形态 602.4）与 A（整阵 589.2）已录入 §1；门禁窗若与探针窗同型，F 预期落 490-515。

## 4. 具体实验步骤（可复现：改哪、怎么改、跑什么命令）

1. **读父本** `codes/iter004-e77e66ca/solution.py`，候选落 `codes/iter016-c1767812/solution.py`；契约逐字保留：`matmul(a, b)` 签名、`asarray f32`、模块级 `np.seterr` 三项抑制、`_self_check` 用例表（方阵/奇数行/非方阵/嵌套 list/M=0/M=1）、`mid = m // 2` 奇数行语义。
2. **TDD 先红后绿**：先写 `test_solution.py`（bitwise == `np.matmul(a32,b32)` 锚 @ 64/512/8192；奇数行/非方阵/嵌套 list；同输入双调用逐位一致——对齐 verify ⑦），跑红；再实现唯一变更（单变量：顺序→并发）：`import threading`；`matmul()` 内 `c = np.empty((m,n))` 后起 2 个 `threading.Thread`，worker 函数**第一行** `np.seterr(divide="ignore", over="ignore", invalid="ignore")`（线程局部——探针实证），随后各自 `np.matmul(a32[:mid], b32, out=c[:mid])` / `np.matmul(a32[mid:], b32, out=c[mid:])`；`try/finally` join 两线程；返回 `c`。**不做其它任何变更**（不碰缓冲管理/dtype/切分点）。依据：research_report §4 第 1/2 条。
   - 注意：候选源码注释与 docstring **不得出现** `sys.modules`、`__main__.`、`time.xxx =`、`signal.alarm =`、`__builtins__` 字样（verify 静态检查正则全文扫描，含注释）。
3. **自检**：`env/.venv/bin/python codes/iter016-c1767812/solution.py`（跑 `_self_check`，全部 bitwise）。
4. **门禁各至多一次、读数即记录、不复跑不追噪声**（GEMMbench 纪律 + iter12-15 教训）：
   - `env/.venv/bin/python env/skills/verify.py --candidate codes/iter016-c1767812/solution.py`
   - `env/.venv/bin/python env/skills/benchmark.py --candidate codes/iter016-c1767812/solution.py`
   - `env/.venv/bin/python env/skills/gate_check.py --candidate codes/iter016-c1767812/solution.py --baseline 484.4295`
5. **回填本记录 §6**：实现要点、门禁读数（含 ref）、按 §3 判据给 admit/reject 结论；如 reject 附判窗依据。

## 5. 观测现象（预期指标与量级，与 gate_defs 同尺度）

- **正确性**：verify `max_error = 0.0`（bitwise）、sizes [64,512]、双 seed、双跑一致全过（探针已证全臂 bitwise，含并发 F）。
- **性能（runtime_ms，低越好；软门禁相对带 5% ⇒ elite 带约 [460.1, 508.6]）**：
  - 负载窗（ref ~590 型，与探针同窗）：F ≈ **490-515ms**——可能高于 elite 484.43（窗口受限，非机制失败；§3 有判窗条款）。
  - 安静窗（ref 434-460 稳态）：F ≈ **425-475ms**，较 elite 484.43 改善 **10-60ms（2%-12%）**；带外更优（< 460.1）为强信号。
  - **机制生效信号**：F 低于同窗 ref，或低于父本形态同窗复测 ≥ 50ms；**失效信号**：F 反而高于父本形态同窗读数（过订阅劣化 → §3 机制证伪条款）。

## 6. 执行过程与结论（占位——由 coding 填「实现与测试」，debug 填「卡点/根因/解决」）

<!-- coding 追加：实现要点 / 测试红绿 / 门禁各工具单次读数（runtime_ms、ref、score、bitwise）；
     debug 追加：卡点 / 根因 / 解决；
     结论按 §3 判据：admit（< 484.43）或 reject（≥ 484.43，附 ref 判窗）。 -->

## 6. 执行过程与结论

### 实现（coding）
- 改动点（父本 iter004 → 候选 iter016，单变量：顺序→并发，其余逐字保留）：
  1. `import threading`（标准库，无新依赖）。
  2. `matmul()` 内 `c = np.empty((m,n))` 预分配不变；两次半阵 `np.matmul(a32[:mid], b32, out=c[:mid])` / `np.matmul(a32[mid:], b32, out=c[mid:])` 由顺序执行改为两个 `threading.Thread` 并发执行；worker 函数**第一行** `np.seterr(divide="ignore", over="ignore", invalid="ignore")`（np.seterr 线程局部——探针 F 臂实证 vecLib 虚假 FP 警告只在 worker 线程冒出）；`try/finally` 保证两线程 join 后才返回 `c`。
  3. 契约逐字保留：`matmul(a,b)` 签名 / `asarray` f32 / 模块级 seterr 三项抑制 / `_self_check` 用例表 / `mid = m//2` 奇数行语义 / C 连续行切片 out= 直写。源码全文 grep 复核无 `sys.modules` / `__main__.` 属性访问 / time 属性赋值 / `signal.alarm =` / `__builtins__` 字样（verify ⑩ 静态检查规避）。
- TDD 记录（`codes/iter016-c1767812/test_solution.py`，纯 Python runner 无 pytest 依赖）：
  - **先红**：父本原样拷入 `solution.py` 后跑测试 → `test_halves_run_concurrently_off_main_thread` 红（spy 包裹 `np.matmul` 记录每次派发的调用线程 ident；父本两次半阵派发全在主线程，caller_idents=[8472051584] 单 ident → 断言「两次派发须发生在主线程之外的 2 个 worker 线程」失败）；契约三测试（bitwise 锚/形状鲁棒/双跑一致）在父本上绿——由判别测试承担红，确认测试真的在测本次改动。
  - **转绿**：实现上述并发变更（唯一改动）→ 4/4 PASS。
  - **重构**：无需——最小实现即最终形态，`_self_check` 未动。
- 自检：`env/.venv/bin/python codes/iter016-c1767812/solution.py` → `self-check ok (bitwise identical, shapes robust): [(128,128,True,0.0),(129,64,True,0.0),(64,127,True,0.0)] ...`。

### 门禁读数（各至多一次、读数即记录、不复跑）
- **verify**：`{"passed": true, "max_error": 0.0, "sizes": [64, 8192], "seeds": {"64": [42, 346281650], "8192": [42, 1467634526]}}` —— bitwise 0.0，双规模双 seed 全过。
- **benchmark**：`runtime_ms=521.94`（K=3 中位；timings [521.48, 527.64, 521.94]），`ref_runtime_ms=595.60`，`score=1.141` —— ref ≥ 550 ⇒ 负载窗（与探针窗同型）。
- **gate_check**（--baseline 484.4295）：`passed=true`；perf `runtime_ms=497.87`（timings [497.87, 505.90, 497.86]）、`ref_value=564.11`、`score=1.133`；soft `verdict=band`（ratio 1.0278 < 1.05，落在 elite 相对带 [460.1, 508.6] 内，`tolerance_band=true`）。

### 与假设的对照（按 §3 判据）
- 正确性：✅ verify 全过（bitwise 0.0，双 seed 双规模双跑一致）。
- 性能：中位 497.87（gate_check 同源读数）≥ 484.43 ⇒ **不满足 admit 线**；同窗 ref 564.11 ≥ 550 ⇒ **负载窗**（benchmark 独立窗 ref 595.60 同型，与探针窗 load≈6.9/18 一致）→ 按 §3「否定（窗口所限，非机制失败）」条款记录 **reject + 判窗**。
- 机制信号（§5，**生效**）：F 低于同窗 ref **66.2ms**（497.87 vs 564.11；benchmark 窗 521.9 vs 595.6 = -73.7ms）——与探针 F 498.1 < A 589.2 完全同型；低于父本形态同窗官方复测（607.9 @ ref 593.3 窗）约 110ms ≥ 50ms 阈值；无失效信号（未反高于父本形态同窗读数）。gate_check 读数 497.87 落在负载窗预期带 490-515ms 内。
- **结论：reject（窗口所限，非机制失败）**——中位 ≥ elite 484.43 且 ref ∈ 负载窗（564-596 ≥ 550），按 §3 判窗条款如实记录；机制证据见探针（F -104ms vs B）与本次同窗对照（F < ref 66ms / < 父本形态 ~110ms）。安静窗（ref 434-460）预期 425-475ms，但门禁纪律为「各至多一次、不追窗」，预算 17/17 已尽，不再复测。注：框架同源 gate_check 本身为 passed=true / soft=band（带内），最终入格与否由框架级联门裁量，本记录按实验自身判据给 reject（判窗）。

### 卡点与解决（debug，attempt 100，触发：框架门禁失败 soft_gate）
- 触发类型：框架门禁失败（soft_gate：runtime_ms=509.20 vs baseline=484.43，ratio=1.0511，verdict=worse）
- 复现方式与关键读数：gate_check ×2（同命令 `--baseline 484.4295`）：run1 518.03ms / ref 587.58 / ratio 1.0694 worse；run2 512.89ms / ref 597.59 / ratio 1.0588 worse；两次 correctness 均 bitwise max_error=0.0。loadavg 全程 6.49→6.70（负载窗）。连同此前读数（497.87 band / 509.20 / 521.94），候选分布稳定 498-522ms；全部 5 个门禁窗 ref 均在 564-598（≥550 负载窗判据）。
- 同窗 A/B 探针（`_debug_ab_probe.py`，交错 5 轮 @8192 seed=42）：顺序两半中位 598.3 vs 并发两半（候选形态）中位 512.3（**-86.0ms**），整阵单次 574.1；两形态 bitwise 均 == 整阵锚。
- 三个假设与验证结论（逐条）：
  - 假设1（负载窗测量假象：安静窗精英基线 484.43 与当前负载窗读数跨窗不可比）：若成立应观察到所有门禁窗 ref ≥ 550 且候选低于同窗 ref。**成立**——5 窗 ref 全部 564-598，候选每窗低于 ref 66-90ms，且低于同窗整阵与父本顺序形态；loadavg≈6.9 与探针窗一致。
  - 假设2（并发机制失效 / CPU 过订阅反噬，NumPy performant_code 文档警示同型）：若成立应观察到并发 ≥ 顺序或候选 ≥ 同窗 ref。**不成立**——同窗并发低于顺序 86ms，候选低于同窗整阵 ref 62-90ms，无任何失效信号。
  - 假设3（实现缺陷：seterr / out= / 线程开销误用）：若成立应观察到并发慢于顺序父本形态或正确性异常。**不成立**——A/B 并发 -86ms；verify 双跑 bitwise 0.0；用户手册确认 np.seterr 状态存于 contextvars（每线程独立，worker 内 seterr 为正确用法）、matmul 释放 GIL（官方 Thread Safety 文档）、out= 行切片形状精确匹配且 C 连续（is_blasable2d 满足；numpy#18669 预 fault 坑已被探针 E 臂证伪）。
- 根因：非实现缺陷、非机制失效——soft_gate「worse」为**跨窗比较假象**：基线 484.43 是 2026-10-24 安静窗 admit 读数，当前所有门禁窗均为负载窗（ref 564-598ms、loadavg≈6.9、ref 在官方带 [380,700] 内故门禁不标窗）。候选 507-522ms 落在负载窗预期带 490-515 上沿附近，且低于同窗整阵 ref 与父本顺序形态。依据：本记录 §3「否定（窗口所限，非机制失败）」判窗条款 + 上述复现读数 + NumPy 官方文档（seterr contextvars / matmul GIL / out= 约束）。
- 处置：**keep_original**——维持 coding 的「reject（窗口所限，非机制失败）」结论，未改动候选代码：并发机制经同窗 A/B 复证 -86ms，无任何可修缺陷；负载窗内再压 5-10ms 进带属「追窗」行为，违反门禁纪律（各至多一次、不追噪声），且探针已证其余 6 个变体全部更劣（无更优实现可换）。候选最终入格与否由框架级联门裁量（gate_check 曾报 passed=true / soft=band @497.87，本窗复验 false @512.9-518.0，均在带缘 508.65 附近抖动——判窗证据充分）。
- 复验结果：gate_check（--baseline 484.4295）run1 passed=false（soft worse，518.03 / ref 587.58）、run2 passed=false（soft worse，512.89 / ref 597.59）；correctness 全过（bitwise max_error=0.0，双 seed 双规模）。复验确认为**稳定负载窗现象**（非瞬时噪声），根因归窗口不可比而非候选劣化。

### 卡点与解决（debug，attempt 200，触发：recheck tolerance_band_recheck 容差带内复验④）
- 触发类型：容差带内复验（④）——框架 recheck 报 perf_gain_pct=-4.664（中位 507.02 vs elite 484.43，soft verdict=band，tolerance_band=true）：「真没收益」还是「被测噪/热点误判掩盖了收益」？
- 复现方式与关键读数：gate_check ×2（同命令 `--baseline 484.4295`）：run1 520.54ms / ref 589.45 / ratio 1.0745 **soft worse**（timings [495.61, 523.47, 520.54]，correctness bitwise 0.0）；run2 504.57ms / ref 557.89 / ratio 1.0416 **soft band**（timings [508.13, 504.57, 502.73]，correctness bitwise 0.0）；框架 recheck 窗同款 507.02 / ref 598.20 / band。loadavg 7.15→8.92（负载窗，较 attempt100 的 6.5-6.7 更重）。含历史在内全部 8 个候选中位 497.87-521.94，跨带缘 508.65 两侧 → band/worse 判定随窗翻转，但读数分布本身稳定（非瞬时噪声）。
- 三个假设与验证结论（一次跑完三条验证，无中间改动）：
  - 假设1（跨窗比较假象：安静窗基线 484.43 与当前负载窗绝对读数不可比）：若成立应观察到全部测量窗 ref ≥ 550 且候选低于同窗 ref。**成立**——本次 2 窗 ref 557.89 / 589.45（+框架 recheck 窗 598.20）全部 ≥ 550 且在官方带 [380,700] 内（门禁不标窗）；候选每窗低于同窗 ref 53.3 / 68.9 / 91.2ms；loadavg 7.2-8.9。参考实现（整阵）安静窗稳态 434-460（§1），此刻读 558-598——同窗一切绝对读数被系统性抬高约 100ms，-4.66% 为跨窗假象（方法论依据：ASV tuning 文档「run at widely separated times → systematic differences from changes in background system load」；arXiv 2504.11826 基准性能随每日/周期负载系统性漂移）。
  - 假设2（真无收益：并发机制失效/过订阅反噬，收益是探针侥幸）：若成立应观察到同窗 A/B 中 conc ≥ seq。**不成立**——同窗交错 5 轮 A/B（`_debug_ab_probe.py` @8192 seed=42）：seq 中位 606.9（599.9-637.1）vs conc 中位 511.1（501.1-520.4），**conc − seq = −95.8ms**，分布无交叠；两形态 bitwise == 整阵锚（full 单次 599.9）。机制明确有效且与 attempt100（−86.0ms）、builder 探针（−104ms）三窗一致。
  - 假设3（计时噪声/带缘抖动掩盖收益）：若成立应观察到 K=3 内分布宽或中位大幅跳动。**部分成立（次要因素）**——run1 timings 离差 27.9ms（~5.4%）vs run2 5.4ms（~1.1%）；中位跨 8 个历史读数漂移 497.87-521.94（~4.8%）横跨带缘 508.65，soft 判定 band/worse 随窗翻转。但漂移幅度（~24ms）远小于窗口抬升量（~100ms）——主掩盖因素是假设1 的负载窗，而非随机噪声。
- 根因：**容差带「无明显收益」（-4.664%）为跨窗测量假象（主）+ 带缘读数抖动（次），非候选真实劣化**：基线 484.43 是 2026-10-24 安静窗 admit 读数，本次全部测量窗均为负载窗（ref 558-598、loadavg 7-9）；参考实现自身被同窗抬升约 100ms（其安静窗稳态 434-460 vs 此刻 558-598）；同窗 A/B 并发较精英顺序形态快 95.8ms、每窗低于同窗 ref 53-91ms。候选收益被窗口条件掩盖而非不存在。实现无可修缺陷：correctness 全部 bitwise 0.0（双 seed 双规模双跑一致），API 用法经 attempt100 用户手册复核无误（np.seterr 线程局部 / matmul 释放 GIL / out= C 连续行切片）。
- 处置：**keep_original**——维持原结论（coding「reject（窗口所限，非机制失败）」+ 框架 band），未改动候选代码。依据：① 无可修缺陷（正确性 bitwise、同窗机制增益 -95.8ms 三窗复证）；② builder 探针已证其余 6 变体全部更劣，无更优实现可换；③ 负载窗内再压 5-15ms 进带属「追窗」行为，违反门禁纪律（各至多一次、读数即记录、不追噪声）；④ 不改门禁/容差/计时设施（作弊红线，exploit 注册表拒识）。安静窗（ref 434-460）预期 425-475ms 优带，但预算 17/17 已尽，不复测不追窗。
- 复验结果：gate_check（--baseline 484.4295）run1 passed=false（soft worse，520.54 / ref 589.45 / ratio 1.0745）、run2 passed=true（soft band，504.57 / ref 557.89 / ratio 1.0416）；两次 correctness 全过（bitwise max_error=0.0，双 seed 双规模）。复验结论：容差带读数是**稳定负载窗现象 + 带缘抖动**，候选同窗真实优于精英形态 95.8ms——「无收益」判定被窗口噪声掩盖，如实维持原结论交由框架级联门裁量。
