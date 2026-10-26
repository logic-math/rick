# 编码执行计划 · 迭代 13 · dispatch_target=np_matmul|impl_strategy=full_blas_dispatch

## 0. 背景说明（builder 产物状态）

builder（agent_builder.json valid=false，turn-budget 截断）未产出完整 experiment_plan /
research_report / experiment_record；`designs/iter13_*.json` 的 idea 也截断于检索结果摘要。
本计划依据其**落盘的四个探针**（probes/out_ab.py、thread_sweep.py、post_import_probe.py、
v1_correctness.py）重建实验意图：对单元格方法描述的 ①②③ 微形态做 A/B 预验证——
① plain np.matmul（父本/精英形态=对照）、② out= 持久复用（域简报称已证伪，作对照）、
③ 每次新建+预 fault+out=（**该反转侧从未实测**——numpy#18669 / OpenBLAS#3197 的核心机制）。

## 1. 事实（动手前已实测，全部来自 env 三要素/探针）

- profile（solution.py 父本）：matmul self=455.9ms 占 100%，asarray 0.002ms——包装层已为零，收益只能在 BLAS 调用形态里。
- probes/out_ab.py（round-robin×2 轮，K=3 中位）：
  - V0 plain：435.7 / 439.5ms（= 精英形态稳态）
  - V1 out= 持久复用：498.4 / 522.8ms（6/6 计时全部 474-528，**恒慢 ~14%**）
  - V2 新建+预 fault+out=：436.6 / 436.1ms（**与 V0 平价，快于 V1 60-85ms**）
  - V3 np.dot：497.7 / 437.0（双峰噪声）
- benchmark.py 协议复测（每变体独立进程，warmup1+K3 中位）：
  - V1 persist：505.6ms（timings [453, 549, 505]）——A/B 结论在协议态复现（V1 慢）
  - V2 prefault：455.2ms（timings [455, 440, 488]）
  - V0 plain：567.9ms（timings [568, 568, 569]）——当日机器慢模式（同进程 ref=429.7；
    父本精英 430.8 为 10-23 干净态读数；iter13 profile 序列里父本 544/604ms 同现象）
- probes/thread_sweep.py（每配置独立进程）：VECLIB=1→659 / 6→512 / 8→495 / 12→476 / 18→457ms
  ——线程收紧恒劣，默认 18 已最优（与域知识「threaded 无增益」一致）；post-import 控制问题随之失去意义，不再探。
- 机器当日噪声：±2-30% 慢模式插曲（memcpy_gbps 30.8-36.1 波动，疑似共享负载）；K=3 中位只能部分吸收。

## 2. 调研结论（coding 角色，≤3 源，全部落到 API 用法）

- **numpy#18669**（github.com/numpy/numpy/issues/18669）：matmul 带 out= 且输出未 fault 时显著慢；
  预 fault 后反转。→ 本实验形态 ③ 的机制依据。
- **OpenBLAS#3197**：GEMM 输出无物理页时慢，先 memset 预 fault 最快可达 4×（多线程缺页
  争内核锁 / 0 填抖 cache）。→ 预 fault 用 `c[:] = 0`（写入式触页）而非 np.zeros（calloc 惰性页）。
- **numpy 手册 matmul**（numpy.org/doc/stable/reference/generated/numpy.matmul）：out 必须匹配
  (n,k),(k,m)->(n,m) 形状；f32 入 + f32 out 无 cast（casting='same_kind' 放行）；不给 out 则新分配。
  另（numpy#31208 旁证）：matmul 自带 overlap 检查，out 与输入同数组时会临时拷贝——本实现
  out 为独立新建缓冲，无 aliasing。

## 3. 改动点（父本 → 目标，唯一一处，kernel 内部）

父本（iter003 精英形态，plain 整阵派发）：
```python
def matmul(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    return np.matmul(a32, b32)
```
目标（形态 ③：每次新建 + 预 fault + out=，仍为单次整阵 BLAS 派发）：
```python
def matmul(a, b):
    a32 = np.asarray(a, dtype=np.float32)
    b32 = np.asarray(b, dtype=np.float32)
    if a32.ndim == 2 and b32.ndim == 2:          # gemm 形态才走预 fault 路径
        c = np.empty((a32.shape[0], b32.shape[1]), dtype=np.float32)
        c[:] = 0                                  # 预 fault（写入触页，非 calloc 惰性页）
        np.matmul(a32, b32, out=c)
        return c
    return np.matmul(a32, b32)                    # 非 2-D 泛化兜底
```
- K 求和序不动：同一 cblas_sgemm 调用（beta=0 覆写 C 全量），预填 0 不参与结果——逐位一致（v1_correctness.py 已对 out= 路径多规模双 seed 验证 bitwise=0）。
- 每次调用返回**新建**数组（无跨调用共享缓冲——规避 V1 的慢模式与 verify ⑦ 双跑一致性的共享覆写风险）。
- 契约其余不动：matmul(a,b) 签名 / f32 asarray / main 自检协议 / np.seterr。

## 4. 风险卡点

- **V1 慢模式**：out= 持久复用本机恒慢 14%（两协议一致）——已排除，不实现共享缓冲。
- **预 fault 成本**：c[:]=0 写满 256MB ≈ 7-8ms（@34.9GB/s）自付在计时窗内；A/B 实测 V2 与 V0
  平价（Accelerate 缺页开销 ~2ms，域简报口径），预期**平价而非增益**——如实汇报。
- **当日机器噪声 ±30%**：gate_check 单次读数可能落入慢模式（如 V0 的 568ms 插曲）而误判
  worse_than_elite——属环境噪声，记录 timings 供下游归因；不自造判据。
- **out= 形状/dtype 约束**（numpy 手册）：c 形状按 (m,n) 精确构造、dtype=f32，无 cast 分支。
- **非 2-D / 列表输入**：兜底路径保持 plain np.matmul，契约与父本一致。

## 5. 测试清单（test_solution.py，先红后绿）

| 测试 | 断言 | 对应假设 |
|---|---|---|
| test_uses_prefault_out_path | 猴补 np.matmul 记录调用：2-D 输入时 out= 被传入，且派发时刻 out 缓冲为全 0（预 fault 证据） | 形态 ③ 落地（**区分性测试：父本红**） |
| test_bitwise_identity_sq | n∈{64,300,512}×seed{42,123}：输出与 np.matmul(a32,b32) 逐位一致（uint32 视图） | K 求和序不变 |
| test_bitwise_identity_rect | 300×500@500×200 逐位一致 | 非方阵泛化 |
| test_list_input | 嵌套 list 输入与 np.matmul(f32) 逐位一致 | 契约保持 |
| test_fresh_output_each_call | 连续两次调用返回不同对象、值逐位一致 | 规避共享缓冲（V1 风险） |
| test_double_run_consistency | 同输入两遍输出逐位一致 | verify ⑦ |
| test_non_2d_fallback | 1-D×2-D、标量形状异常路径与父本语义一致 | 泛化兜底 |
| test_float64_input_coerced | f64 输入先 asarray f32 后逐位一致 | 契约（dtype 转换） |

## 6. 验证顺序（固定）

1. `verify.py --candidate solution.py`（双规模 64/8192 + 多 seed + 双跑一致 + 静态检查）
2. `benchmark.py --candidate solution.py`（8192，K=3 中位）
3. `gate_check.py --candidate solution.py --baseline 430.8255829999998`（同源融合判定）

预期（对照 gate_defs 与域知识）：正确性逐位一致过；runtime 落在 BLAS 平价带 430-490ms
（当日噪声可能更高）；软门禁大概率 band（±5%）或被噪声推到 worse——两者都如实汇报，
结论写「预 fault 反转侧无净增益（假设否定），V1 持久复用恒慢（新事实）」。
