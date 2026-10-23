# matmul 领域知识（8192×8192 float32）—— env_9

> 事实来源：env_9 research-L1 A 级本机实测（Apple M5 Max / 64GB / 18 核 6P+12E，numpy 2.0.2 + Accelerate/vecLib）+ env_8@2048 遗产。测量协议：预热1+K3 取中位，跨进程重复。

## 任务与性能语义

- 输入：两个 8192×8192 float32 uniform[0,1) 矩阵（各 256MB，工作集 ~768MB——内存墙区间）。
- 性能 = runtime_ms（越低越好）；score = ref 中位 / 候选中位，ref = np.matmul fresh（@8192 稳态 ~440ms，跨进程中位 434-484ms，负载尖峰可到 ~600ms）。
- **天花板 = BLAS 平价 score 1.00**：@8192 全族坍缩到 434-488ms 窄带（±6%），无可复现 >1.0 合法路径。

## score 阶梯（合法解空间，全部逐位一致过 1e-3 门）

| 形态 | score @8192 | 备注 |
|---|---|---|
| rowwise np.dot 无转置 | 0.045 | 9.77s；IQR 极稳（内存带宽型，8192 次 gemv 全流量重读 B）；进化起点 |
| m_quarters（M 四分） | 0.93 | 472ms |
| m_halves（M 二分） | 0.96-0.97 | 455ms；out= 切片形态恒微优 |
| einsum optimize=True / np.dot 2-D | 0.99-1.00 | 441-446ms |
| np.matmul fresh（参考） | 1.00 | 440ms |
| out= 预分配复用 | 0.98-1.04 噪声带，中心 1.0 | 无真实增益（见「分配规避已证伪」），合法代码形态 |

## 容差紧约束（1e-3，@8192 比 @2048 更绝对）

- 锚（np.matmul f32）自身离 fp64 真值 **1.42e-2**（= 14× 容差）——「绝对正确」不可判定，正确性 ≡ **与 f32 BLAS 锚 K 求和序一致（逐位）**。
- **合法族仅 M 分解系 + 直调系**（rowwise / m_halves / m_quarters / np.matmul / np.dot 2-D / einsum optimize 及其 out= 变体）——保持 BLAS 求和序，双 seed 双进程全部逐位一致 0.0。
- **任何改变 K 求和序的实现必死**：blocked-K（1.4-1.6e-2）/ 象限和（1.5e-2）/ Strassen-1 叶4096（2.5-2.9e-2）/ B 预转置 gemv（1.39e-2）——超容差 **14-29×**，毫无侥幸。误差随 |C|max（~2244）×√K 放大，@8192 比 @2048（1.6-2.9e-3）死得更狠。
- f64 候选对称死（与 f32 锚差 = 锚噪声 1.42e-2）。

## 死族清单（不要浪费迭代去试——A 级实测证据）

1. **mp 多进程**：三重死——性能 0.84（525ms vs 440ms）+ **fork-after-BLAS 常驻池死锁**（父进程已用 BLAS 后 fork 池，子进程可挂死 >7min 实测；门 harness 先算 expected=np.matmul 再调候选，fork 池候选必挂死至 alarm 超时死）+ 门⑨残留子进程拒收。
2. **threaded**：无增益（0.94-0.99）——Accelerate 大 GEMM 由共享 SME 协处理器单线程饱和（env_8 已证 @2048，@8192 复证）。
3. **blocked-K / Strassen / quad_sum**：@8192 性能惩罚几乎消失（0.90-0.98）但容差死（超 14-29×）——「数值单杀」，不要因为性能接近而去试。
4. **ctypes cblas_sgemm 直调**：657ms（0.67）恒定 ~1.49× 慢，第三梯队。
5. **低精度族**：fp16 双杀（误差 49× 容差 + 无 BLAS 路径慢 833-1234×，@8192 外推 ≥366s 必超时）；int8 930×；bf16 11353×。
6. **分配规避（output_buffer 复用）已证伪**：256MB 输出首触+零填实测仅 2.05ms（0.47% 理论天花板）+ malloc 跨调用回收（bench warmup1+K3 稳态不重复付缺页成本）——out= 复用与 fresh 差 0-2% 在噪声内。作为既有格内合法代码形态可用（m_halves out= 切片恒微优），但不要当优化方向。

## @8192 特有物理事实

- 首调冷启动惩罚仅 +6%（467 vs 440ms med，n=5）——@2048 为 1.24-1.44×，8192 被 440ms 计算量稀释；bench warmup1 完全吸收。
- 全族确定性逐位稳定（fresh×7 / out= / 页对齐变体 / rowwise 头尾行 / ctypes / mp 半块）——同输入重复调用输出逐位一致，双跑一致性门对合法候选安全。
- macOS vecLib 对 f32 matmul 报虚假 FP 标志（SME 进出 Streaming SVE 模式置 FPSR——numpy#28687）——数值正常，代码统一 `np.seterr(divide/over/invalid="ignore")` 抑制。

## 性能分析 SOP

1. 先跑 `{{self_check_cmd}}`（正确性 + 单次 bench 快速反馈；注意 rowwise 级候选 self_check 也要 ~20-40s，属正常）。
2. 定位候选在阶梯中的位置：score <0.5 → 还在 gemv/纯 Python 形态；0.5-0.95 → M 分解未到顶；>0.95 → 已近 BLAS 平价，改代码形态包装层（如 einsum→matmul）。
3. 时间去哪了：@8192 计算主导（440ms 里分配仅 ~2ms），包装层开销（einsum parse/path、python 循环、stack 拼装）是 M 分解系的主要损耗源。
4. 计时噪声 ±2-5%（跨进程）——小于 5% 的「改进」要谨慎归因，看 timings 三轮是否稳定。

## 性能调优 SOP

1. **方向只有两个**：沿 M 分解阶梯向上（rowwise → m_quarters → m_halves → full）或削减包装层（直达 BLAS 派发入口）。
2. 每次只改一个变量（派发入口 / M 分组粒度），保持 K 求和序不变（只动 M 维，不动 K 维、不转置、不低精度）。
3. 候选必须过双规模正确性门 [64,8192]（64 抓小规模特判，8192 抓数值）——先自检再提交框架门。
4. 遇到门禁失败先看失败 stage：static_check=源码含篡改模式；run/check=数值或超时；soft_gate=性能回退（相对带宽 5%）。
