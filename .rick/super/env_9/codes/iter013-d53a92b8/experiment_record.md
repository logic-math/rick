# 实验记录 · 迭代 13 · dispatch_target=np_matmul|impl_strategy=full_blas_dispatch

> 状态说明：builder 产物截断（agent_builder.json valid=false / designs JSON idea 截断于检索摘要），
> 本记录第 1-5 段由 coding 角色依据 builder 落盘的四个探针（probes/）与注入上下文重建，供下游追溯。

## 1. 背景与价值（为什么现在做这个实验）

- 该格精英 = iter003 plain 整阵 np.matmul（430.8ms，score 1.011，profile：matmul self 100%）——
  包装层开销已为零，格内剩余自由度只在「输出缓冲的管理形态」（out= 微形态）。
- 域知识标注「out= 预分配复用 0-2% 噪声带」为已证伪方向，但 numpy#18669 / OpenBLAS#3197 的
  **预 fault 反转侧**（每次新建 + 预 fault + out=）在本机从未实测；builder 为此写了 round-robin
  A/B 探针（out_ab.py）与线程扫描（thread_sweep.py）。
- 台账 12 迭代：全族坍缩 434-488ms（±6%），天花板 = BLAS 平价。本实验是对格内最后一个未测
  微形态的收口证伪/确认。

## 2. 实验假设（一句话，可证伪）

「按 numpy#18669 的反转机制，每次新建输出缓冲并以写入式预 fault 后用 np.matmul(out=) 派发，
在 8192 SGEMM 上相对 plain np.matmul（V0）有 >5%（软门禁带外）的可复现 runtime 收益。」

## 3. 验证逻辑（什么结果支持/否定）

- 支持：out_ab.py 与 benchmark.py 协议下，V2（预 fault+out=）中位稳定低于 V0 ×0.95 以下。
- 否定：V2 与 V0 差异落在 ±5% 带内（噪声级）——预 fault 反转在本机无净增益
  （Accelerate 缺页成本 ~2ms 被 440ms 计算量稀释）。
- 附加可证伪点：V1（out= 持久复用）若恒慢于 V0（两协议一致）→ 域简报「0-2%」需修正为
  「本机当日恒慢 ~14%」；线程收紧若恒劣 → 默认 18 线程已最优。

## 4. 具体实验步骤（可复现）

1. profile 父本（三要素 profile.py）确认热点 100% 在 matmul 自身。
2. 跑 probes/out_ab.py（round-robin×2、K=3 中位）得 V0/V1/V2/V3 读数。
3. 跑 probes/thread_sweep.py @ VECLIB∈{1,6,8,12,18}（每配置独立进程）。
4. 以 benchmark.py 协议（warmup1+K3 中位）独立复测 V1/V2/V0。
5. 实现 V2 形态为候选（TDD：test_solution.py 先红后绿），依次跑 verify → benchmark →
   gate_check --baseline 430.8255829999998。
6. 结果追加第 6 段。

## 5. 观测现象（预期指标与量级，与 gate_defs 同尺度）

- 正确性：与 f32 BLAS 锚逐位一致（max_error=0.0，双规模双 seed）。
- 性能：V2 预期落在 430-490ms BLAS 平价带（预期否定假设：平价）；若机器进入当日慢模式
  （父本亦测得 544-604ms），读数可能整体上浮，timings 记录供归因。
- 软门禁：大概率 band（±5%）；不排除噪声性 worse。

## 6. 执行过程与结论（占位——由 coding 填「实现与测试」，debug 填「卡点/根因/解决」）

### 实现与测试（coding，迭代 13）
- 实现：solution.py 的 matmul 内核改为形态③——每次新建 `c=np.empty((n,m),f32)` + `c[:]=0` 写入式预 fault + `np.matmul(a32,b32,out=c)` 单次整阵 BLAS 派发；非 2-D 输入兜底 plain np.matmul（契约同父本）。TDD：test_solution.py 8/8 全绿（区分性测试 test_uses_prefault_out_path 父本红→目标绿）。
- coding 于 GREEN 落盘后被 turn-budget 中断（exit 143），verify/benchmark/gate_check 均未执行——self_check passed=false 系流程中断，非代码缺陷。

### 卡点与解决（debug，attempt 1，触发：自检失败）
- 触发类型：自检失败（诊断包 metrics 为空——coding turn-budget 截断，验证流水线未跑）
- 复现方式与关键读数：
  - test_solution.py：8/8 PASS（spy 证明 out= 派发且派发时刻缓冲全 0——预 fault 落地）
  - verify.py：passed=true，max_error=0.0（sizes 64/8192，多 seed，与 f32 BLAS 锚逐位一致）
  - benchmark.py：482.93ms（timings [516, 483, 449]），score 1.138，ref 549.48ms（当日机器慢模式，ref 亦上浮）
  - gate_check.py --baseline 430.8255829999998：passed=true，perf 447.25ms（timings [447, 447, 450]），score 0.997，soft verdict=band（ratio 1.038，±5% 带内）
- 三个假设与验证结论：
  1. 「coding 被中断于验证阶段，磁盘实现完整正确」→ 成立：agent_coding.json exit_code=143/truncated=true，summary 止于「现在进行最小实现（GREEN）」；8/8 测试 + 三要素全过。
  2. 「磁盘实现不完整/仍为父本形态」→ 不成立：区分性 spy 测试绿（out= 已派发、缓冲预清零），bitwise 一致全过。
  3. 「候选过不了性能门禁（V2 劣于精英 430.83 出带）」→ 不成立：gate_check passed，band ratio 1.038；当日慢模式下 ref 亦 445-549ms，V2 与 V0 平价。
- 根因：turn-budget 中断使 self_check 流程未执行（流程性失败，非代码缺陷）；实现行为与假设预期一致（预 fault 反转侧与 plain 平价）。
- 处置：修复=完成被中断的验证流水线（test → verify → benchmark → gate_check），候选代码未改动。
- 复验结果：gate_check passed=true，runtime 447.25ms（k=3 中位），soft=band（ratio 1.038），max_error=0.0。
- 对实验假设的结论：**V2 预 fault 反转侧无净增益（假设否定，平价收口）**；V1 out= 持久复用本机恒慢 ~14%（builder 探针 498-523ms vs plain 436ms——域简报「0-2%」应修正为「本机当日恒慢」）；线程收紧恒劣，默认 18 线程已最优。
