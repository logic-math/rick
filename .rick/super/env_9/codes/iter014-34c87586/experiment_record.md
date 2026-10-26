# 实验记录 · 迭代 14 · dispatch_target=np_matmul|impl_strategy=full_blas_dispatch

> 状态说明：builder 产物截断（agent_builder.json valid=false / exit 143 / 900s 超时，idea 止于
> 检索摘要，无 plan_steps，未落 research_report.md 与本记录）。第 1-5 段由 coding 依据 builder
> 落盘的探针 `probes/buffer_ab.py`（臂注释即实验意图）+ iter13 台账 + 本机三要素实测重建，
> 供下游追溯。

## 1. 背景与价值（为什么现在做这个实验）

- 该格精英 = iter003 plain 整阵 np.matmul（430.83ms，score 1.011，profile：matmul self 100%）。
  包装层开销为零，格内剩余自由度只剩「输出缓冲的管理形态」。
- iter13 已收口 V2 形态（每次 fresh + memset 预 fault + out=）：与 plain 平价
  （gate 447.25ms，ratio 1.038 带内，reject worse_than_elite）；同时 builder 探针 out_ab.py
  观察到 V1（持久预 fault 缓冲复用 + out=）「恒慢 ~14%」异常（498-523 vs 436ms）。
- builder（截断前）为该异常写了机制定位探针 `buffer_ab.py`（P0 plain / P1 持久 post 布局 /
  P2 fresh+memset / P5 pre 布局 / P6 spacer 布局=「候选可实现布局规避」/ P7 dot+out 机制判别）。
  **iter14 的实验对象 = V1/P1 持久缓冲形态（含 P6 布局变体）——该家族最后一个未作为候选提交的微形态。**
- 台账 13 迭代：全族坍缩 434-488ms（±6%），天花板 = BLAS 平价。本实验为收口证伪/确认。

## 2. 实验假设（一句话，可证伪）

「per-shape 持久输出缓冲（写入式预 fault 后复用，np.matmul out= 派发；必要时加 import 期
spacer 布局规避）相对父本 plain np.matmul 在 8192 SGEMM 上有 >5%（软门禁带外）的可复现
runtime 收益。」

## 3. 验证逻辑（什么结果支持/否定）

- 支持：benchmark.py 协议（fresh 进程，warmup1 + K3 中位）下候选中位 < 430.83×0.95 ≈ 409ms，
  且 verify 逐位一致。
- 否定：候选落在带内（409-452ms）或更差——预 fault 持久复用无净增益（分配/缺页成本
  在本机不可测，433ms 即 sgemm 本体）。
- 附加可证伪点：V1「恒慢 14%」若不可复现（当日探针 P1 floor=432.7 ≈ P0 floor=433.0）
  → iter13 的慢读数应重分类为机器慢模式而非形态固有；P6 spacer 若更差 → 布局假说无操作价值。

## 4. 具体实验步骤（可复现）

1. profile 父本（三要素 profile.py）确认热点 100% 在 matmul 自身。
2. 跑 builder 的 `probes/buffer_ab.py`（6 臂 × 3 轮交错，每臂 1 预热 + 5 计时中位 +
   ru_minflt/ru_majflt/call + memcpy 漂移标记）。
3. 以 benchmark.py 协议对 3 个候选形态各测 5 次（fresh 进程 ×
   {plain 父本对照, persist P1, spacer P6}，`probes/variant_*.py`）。
4. 按数据选形（persist P1；spacer 实测更劣否决），TDD 写 `test_solution.py`（父本红 → 绿），
   改 `solution.py`。
5. 依次跑 verify → benchmark → gate_check --baseline 430.8255829999998。
6. 结果追加第 6 段。

## 5. 观测现象（预期指标与量级，与 gate_defs 同尺度）

- 正确性：与 f32 BLAS 锚逐位一致（max_error=0.0，双规模 [64,8192] 双 seed 两遍一致）。
- 性能：预期落在 432-460ms 噪声带（当日机器 floor：探针 warm 循环 432-433；fresh 进程
  434-460，偶发 460-590 尖峰）。软门禁大概率 band；不排除噪声性 worse。
- 已知风险：持久形态存在小概率「整机慢模式」窗（当日 5 次 benchmark 中 1 次三计时全
  580-590ms，同窗 ref 亦 484——外部 CPU 争用）。

## 6. 执行过程与结论

### 实现（coding）

- 改动点（父本 iter003 → 本迭代，唯一一处，kernel 内部输出缓冲管理）：
  - `solution.py` 新增模块级 `_BUFS` per-shape 缓冲表 + `_get_buf(n, m)`（首次分配
    `np.empty((n,m), f32)` + `buf[:] = 0` 写入式预 fault）；2-D 路径改为
    `np.matmul(a32, b32, out=c)` 派发后返回持久缓冲；非 2-D 走 plain 兜底（契约不变）。
  - 派发目标（np.matmul）、单次整阵、K 求和序、契约（list/array、asarray f32、返回
    (n,m) ndarray）、main 自检协议、np.seterr 全部不动。
  - **否决的同族变体（数据驱动，未实现）**：P6 spacer（import 期 1GB junk + 持久缓冲）——
    benchmark 5 次中位 439.1/454.2/482.8/468.6/440.6，三形态中最差（1GB 常驻加剧本机
    尖峰）；iter13 V2（fresh+memset+out=）已证伪平价。
- 机制探针读数（`probes/buffer_ab.py`，当日，warm 循环 108 次 sgemm）：
  - P0 plain 432.5/433.1/433.0（floor 稳定）；P1 持久 490.9/463.8/**432.7**（floor 与 P0
    相同——iter13「V1 恒慢 14%」当日不可复现，应重分类为机器慢模式）；P2 fresh 455.9/435.0/458.9；
    P5 pre 452.9/482.4/453.0；P6 spacer 453.8/449.9/481.6；P7 dot+out 454.2/435.2/455.0；
    全臂 minflt/call=0（预 fault 后零缺页）；`bitwise P1_vs_P0=True`（uint32 视图）。
  - 结论：6 形态同 floor（~433ms = sgemm 本体），差异全为机器尖峰噪声；P6 布局假说无操作价值。
- 候选形态 3 臂对比（benchmark.py 协议，各 5 次 fresh 进程）：persist 中位
  589.3⚠/443.6/442.2/432.1/437.0（floor 最佳 432.1；1 次 589 为外部争用窗，同窗 ref 484）；
  plain 459.0/452.8/444.9/435.7/435.9；spacer 468.6/440.6/439.1/454.2/482.8。
- TDD 记录：
  - 测试文件 `test_solution.py`（9 测试）；区分性测试 = `test_persistent_buffer_reuse`
    （同形状两次调用共享底层缓冲）+ `test_reuse_zero_alloc_second_call`（第二次派发 out
    即上一调用返回对象）+ `test_recompute_not_cached`（换输入全量重算 + 旧引用同步覆写）。
  - RED（父本 plain）：上三测试失败（fresh 分配不共享缓冲；out=None），其余 6 契约/不变量绿。
  - GREEN（最小实现后）：9/9 通过；自检 `python solution.py --n 8192` →
    max_error_vs_matmul = 0.0。重构：无（实现即最小形态）。
- 门禁结果（三要素，与框架同源）：
  - verify：`passed=true, max_error=0.0, sizes=[64,8192], seeds={64:[42,474890810],
    8192:[42,1902715872]}`（多 seed + 两遍一致性全过）。
  - benchmark：`runtime_ms=438.416, score=1.034, timings=[438.4,503.3,437.5],
    ref_runtime_ms=453.149, n=8192, k=3`。
  - gate_check --baseline 430.8255829999998：`passed=true, perf=432.808ms
    (timings=[432.8,452.9,431.5]), score=0.999, ref=432.390, soft verdict=band,
    ratio=1.0046, tolerance_band=true`。
- 与假设的对照：**否定（带内平价，机制侧部分成立）**——
  1. 正确性/合法性全成立：out= 持久派发与 plain 逐位一致（max_error 0.0），门禁全过
     （同进程 ref 432.4 vs 候选 432.8，score 0.999 = BLAS 平价）。
  2. 「>5% 带外收益」不成立：候选 432.81 vs 精英 430.83，ratio 1.0046（band 内 0.46%）。
     持久复用省下的分配/缺页成本在本机不可测（探针全臂 minflt/call=0、同 floor 433ms）——
     与 iter13 V2 收口、域简报「分配规避无真实增益」一致。
  3. iter13 的「V1 恒慢 ~14%」当日不可复现（P1 floor 432.7）；但持久形态存在小概率外部
     争用敏感窗（5 次 1 次 589ms）——形态本身不加分也不稳定加分，plain 与 persist 在
     噪声内不可区分。
  - 框架入格判定以框架级联门为准（iter13 先例：band 内 perf 451.14 > 精英 430.83 仍按
    worse_than_elite 拒——本迭代 432.81 同样高于精英 2ms，若按同规则将拒；如实上报）。
- 附加物证：`probes/variant_{plain,persist,spacer}.py`（3 形态对照）、`probes/buffer_ab.py`
  （builder 机制探针，本次运行读数见上）、`coding_plan.md`（含否决变体与 API 约束来源：
  numpy.org matmul 手册 out 形状/签名约束 + github.com/numpy/numpy#18669 预 fault 反转）。

### 卡点与解决（debug，attempt 200，触发：容差带内复验 recheck(tolerance_band_recheck): perf_gain_pct=-2.161）
- 触发类型：容差带内复验（④）——recheck 候选中位 440.14ms vs 精英 430.83ms（ratio 1.0216，band），需判定「真无收益/真劣化」还是「噪声掩盖收益/假劣化」。
- 复现方式与关键读数（当日，gate_check ×3 fresh 进程 + 交错 A/B 探针 `probes/debug_recheck_ab.py` 30 计时）：
  - gate_check ×3（verify 每次随机新 seed，三次全 max_error=0.0）：
    ① cand 453.01（timings 453.0/453.0/452.9 紧致）vs **同进程 plain ref 452.94，score 0.9998（Δ=0.07ms/0.02%）**，ratio 1.0515 worse；② cand 524.92（442.2/578.0/524.9 争用窗）vs ref 440.96，ratio 1.2184 worse；③ cand 439.07 vs ref 449.85，score 1.0245，ratio 1.0191 band passed。
  - 交错 A/B（3 轮正/反/正序，每臂 1 预热 + 5 计时）：pooled 中位 plain 453.0 vs persist 453.8；floor 433.1 vs 434.0（0.2% 噪声级）；逐轮胜负交替（r1 persist 慢、r2/r3 persist 快）；30 计时中低于带外更优线 409.3ms（430.83×0.95）的读数 = **0**；bitwise persist_vs_plain=True。
  - 关键对照：当日同进程 plain ref 中位 440.96/449.85/452.94/476.48——**plain 自身比其精英记录 430.83 慢 2.3%~10.6%**；当日 plain warm-loop floor 432.5-433.1 亦高于 430.83（精英读数当日不可复现）。
- 三个假设与验证结论（逐条）：
  1. 假设 1（**成立**）：-2.161% 为跨会话机器状态漂移 + 外部争用窗噪声，非 persist 形态固有劣化。若成立应观察到：同进程候选 vs plain ref 打平（score≈1）、当日 plain 自身偏离精英记录、重复测量 ratio 两侧波动。实测：run1 score 0.9998（Δ0.02%）；plain ref 当日 440.96-476.48 全部 > 430.83；5 次同进程 score = 1.083/0.999/0.9998/0.840/1.025。✓
  2. 假设 2（不成立）：persist 形态存在真实固有成本（常驻 256MB 缓冲 cache/TLB 压力），被 K=3 中位掩盖。若成立应观察到：交错 A/B 每轮 persist 一致慢、floor 稳定差距。实测：逐轮胜负交替，floor 差 0.9ms（0.2%），buffer_ab P1 floor 432.7 = P0 433.0。✗
  3. 假设 3（不成立）：out= 用法违反 numpy 手册约束（shape/dtype/布局）走 cast/复制慢路径。若成立应观察到：ValueError / 数值不一致 / ≥30ms 级复制开销。对照官方手册（numpy.org matmul：out「must have a shape that matches the signature (n,k),(k,m)->(n,m)」，casting='same_kind'）：实现传精确 (n,m) C-contiguous f32，逐项吻合；实测 bitwise=True、floor 差 0.2%。✗
- 根因：**测量侧**——精英基线 430.83 为跨会话读数，当日机器（vecLib sgemm floor ~433ms + 外部争用窗 580-590ms 尖峰）整体偏慢 2~10%，-2.161% 全部由该漂移贡献；**机制侧**——本机 plain np.matmul warm 后 minflt/call=0（分配器回收已 fault 页，等效「已预 fault」），numpy#18669 的预 fault 收益（Linux/OpenBLAS 36→213.9 GFlops 情境）在本机 macOS 分配行为下无可省成本，persist 与 plain 同 floor = BLAS 平价天花板。来源：numpy.org/doc/stable matmul 手册（out 签名/casting 约束）；github.com/numpy/numpy#18669（未预 fault out= 慢、预 fault 反转）；github.com/xianyi/OpenBLAS#3197（多线程页故障内核锁竞争）。
- 处置：**keep_original**（维持原结论：带内平价、无 >5% 收益）。复核依据：① 无形态固有劣化（同进程 score 0.9998、floor 差 0.2%、逐轮胜负交替）；② 无被噪声掩盖的收益（30 计时零个 <409.3ms，floor 433-434 = sgemm 本体）；③ 实现与手册逐项吻合且逐位一致。候选代码未改动（改回 plain = 精英本身，无意义且违背实验意图）；框架级联判定（band 内高于精英按 worse_than_elite 拒）按其规则如实接受。
- 复验结果：gate_check --baseline 430.8255829999998 当日 3 次 = worse(1.0515) / worse(1.2184，争用窗) / band(1.0191，passed)；最干净同进程对照 run1：cand 453.01 vs plain ref 452.94（score 0.9998）。
