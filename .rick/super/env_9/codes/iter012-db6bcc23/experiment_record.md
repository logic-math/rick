# 实验记录 · 迭代 12 · dispatch_target=einsum_path|impl_strategy=m_quarters

## 6. 执行过程与结论

> coding 段缺席说明：coding agent 于自检前被 900s 进程超时杀死（exit 143，
> agent_coding.json "[timeout after 900s"，wall_ms=900030）——其新探索
> （probe_a-e：einsum 'ki' form_A 转置装配路线）停留在探针阶段，未写入
> solution.py；自检从未运行（诊断包 metrics 为空）。solution.py 为 iter011
> 已准入版本的原样拷贝（与 iter011-607eb59d/solution.py 逐字节一致，含其
> debug attempt 1 的 m<4 退化分支）。以下为 debug 复验补记。

### 卡点与解决（debug · attempt 2，触发：自检失败）
- **触发类型**：自检失败（metrics 为空 = 自检从未跑完——运营性失败，非代码缺陷）。
- **复现方式与关键读数**：
  - solution.py 自身自检：`--n 64` → 0.018ms / err=0.0；`--n 8192` → 480.4ms / err=0.0。
  - verify（三要素）：`passed=true, max_error=0.0`（sizes [64,8192]，seed 42+随机，双跑一致）。
  - benchmark：runtime_ms=480.46（timings 480.5/498.3/478.5），score=0.975，ref=468.4（带内 [380,700]）。
  - gate_check（--baseline 482.359875 = 本格精英 iter011）：`passed=true, failure_stage=null`；
    perf 487.60ms（timings 487.6/472.0/516.4）、score 0.889、ref 433.7；
    soft verdict=band（ratio 1.0109，±5% 带内，tolerance_band=true）。
- **三个假设与验证结论（逐条）**：
  1. 「coding agent 被 900s 进程超时杀死、自检从未运行（空 metrics）」——**成立（根因）**。
     证据：agent_coding.json exit_code=143 / wall_ms=900030 / 摘要截断于 probe_e 探索；
     solution.py mtime(04:08) 早于全部探针(04:15-04:36) 且与 iter011 准入版逐字节一致
     （diff IDENTICAL）；三要素门全绿（若代码真有缺陷则应红）。
  2. 「候选代码存在正确性/性能缺陷（如 m<4 GEMV ULP 复发、perf 超软门带）」——**不成立**。
     verify max_error=0.0（双规模双 seed 双跑）；gate_check perf 487.6ms（ratio 1.0109 带内）、
     score 0.889 < 2.0、ref 433.7 ∈ [380,700]、runtime > 0.01。
  3. 「机器/环境劣化致测量不可信（ref 出带或 perf 出带）」——**不成立**。两次独立 benchmark：
     ref 468.4 / 433.7 均带内；候选中位 480.5 / 487.6 均在基线 +1.1% 内；计时分布紧凑
     （478.5-498.3 / 472.0-516.4）。
- **根因**：运营性超时（coding agent 900s 被杀、自检未执行、空 metrics 触发 debug），
  非候选代码缺陷——现 solution.py 即 iter011 已准入解（含 m<4 整阵单发 tensordot 回退，
  依据 numpy#26791 macOS Accelerate 形状相关内核精度 / SO#76707696 / M=1 GEMV 特殊路径，
  iter011 debug 已溯源并实测；本轮 source_check 复核检索命中同源）。
- **处置**：keep_original（未改动任何代码——「不为产出而硬改」纪律）。复核依据：三要素
  verify/benchmark 与融合 gate_check（带精英基线 482.359875）全绿；attempt 1 已证
  verify max_error=0.0，本次补齐 benchmark + gate_check 闭环。
- **复验结果（gate_check）**：`passed=true, failure_stage=null`；verify max_error=0.0
  （[64,8192] 双 seed）；perf 487.599ms（k=3 中位）；soft band（ratio 1.010861 vs 482.359875）。

### 卡点与解决（debug · attempt 200，触发：容差带内复验）
- **触发类型**：容差带内复验（recheck：tolerance_band_recheck，perf_gain_pct=-2.119，
  soft verdict=band，ratio 1.021185 vs 基线 482.359875）。
- **复现方式与关键读数**：
  - 框架 recheck 原读数：perf 492.579ms（timings 485.56/492.58/498.77）→ -2.119%。
  - 本轮三要素复验：verify `passed=true, max_error=0.0`（[64,8192] 双 seed）；
    benchmark×2：486.777ms（timings 525.65/483.77/486.78，ref 452.8 带内）/
    508.434ms（timings 508.43/520.64/484.76，ref 457.8 带内）；
    gate_check（--baseline 482.359875）：`passed=true, failure_stage=null`，perf 479.914ms
    （timings 480.42/479.91/476.46）、score 0.910、ref 436.7 带内、ratio 0.99493（反侧快 0.5%）。
  - 关键观察：同一份代码的独立 k=3 中位读数族 {479.91, 480.46, 486.78, 487.60, 492.58,
    508.43} 相对基线 482.36 摆动于 +0.5% ~ -5.4%，**符号不定**（有快有慢）。
- **三个假设与验证结论（逐条）**：
  1. 「候选与基线精英为同一份代码（字节相同），期望收益恒为 0，-2.119% 纯属跨次测量
     噪声、亦不存在被掩盖的收益」——**成立（根因）**。证据：`diff iter011-607eb59d/
     solution.py iter012-db6bcc23/solution.py` 逐字节一致；收益读数双侧摆动
     （ratio 0.9949~1.0539），真劣化应表现为单侧稳定超带。
  2. 「机器/环境系统性劣化，-2.119% 为真实劣化」——**不成立**。证据：五次独立 ref 读数
     433.7/436.7/452.8/457.8/468.4 全部落在 [380,700] 带内；最新 gate_check ratio 0.9949
     （快于基线），无单侧偏移。
  3. 「测量作弊/污染假象（预存结果/计时篡改/降精度/拖慢 ref）」——**不成立**。证据：
     verify max_error=0.0；timings 呈自然噪声分布（k=3 内 476-526ms，含 3-8% 波动）；
     代码为逐块实时 tensordot 计算（无预存/缓存返回路径）；ref 未被拖慢（全部带内）。
- **根因**：「容差带内无收益」为**真结论**——solution.py 即 iter011 已准入精英的原样拷贝
  （迭代 12 的 einsum 'ki' form_A 新探索因 coding 900s 超时停留在 probe_e 未落地），
  收益在定义上恒为 0；recheck 的 -2.119% 为本机 BLAS sgemm 跨次运行噪声（±3% 量级。
  旁证：OpenBLAS#1200 报告 cblas_sgemm 平均运行时间跨次大幅波动（big-little core 调度）；
  numpy 官方 benchmark 文档亦以「多次重复测量的时间分布」为基准口径）。
- **处置**：keep_original（未改动任何代码——候选即基线本身，无收益可保留也无劣化可修复；
  「不为产出而硬改」纪律）。probe_e 揭示的 einsum 'ki' 调用 112ms 潜在线索属下轮
  builder/coding 探索范畴，不属 debug 修复范围。
- **复验结果（gate_check）**：`passed=true, failure_stage=null`；verify max_error=0.0
  （[64,8192] 双 seed）；perf 479.914ms（k=3 中位，timings 480.42/479.91/476.46）；
  soft band（ratio 0.99493 vs 基线 482.359875，±5% 带内）；score 0.910 < 2.0；
  ref 436.7 ∈ [380,700]。
