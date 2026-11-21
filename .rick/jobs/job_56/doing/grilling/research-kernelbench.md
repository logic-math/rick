# research-kernelbench — KernelBench 验证集与评估协议（super env 移植调研）

> 2026-11-30 · 调研触发：job_56 第五轮（agent 写 kernel 用公开验证集对齐测试的 e2e 任务选型）
> 方法：web_search 3 轮（8+6+6 检索）+ arXiv 原文交叉验证 · 信源等级见各条
> 消费方：human（任务裁决）+ gen_env（env 设计输入）

---

## 事实性结论

### Q1 验证集结构（任务形态与输入生成）

1. **任务形态（A 级）**：每任务 = 一段自包含 PyTorch 程序：`Model(nn.Module)` reference 实现 + `get_inputs()`（随机张量生成器）+ `get_init_inputs()`（构造参数）。LLM 的任务是把 torch 层替换为自定义实现（`ModelNew`）。250 题，**只评 forward pass**。（arXiv:2502.10517 §3.1 + GitHub README）
2. **无 ground truth kernel（A 级）**：「We do not provide ground truth kernels … by design, KernelBench is automatically verifiable」——随机生成 prescribed shape/precision 的输入张量，跑 PyTorch Model 收集输出作为对齐基准（reference 即 oracle——任务自动可验证是设计核心）。（arXiv:2502.10517 §3.3）
3. **难度分层（A 级）**：Level 1 = 单算子 100 题（conv/matmul/layernorm/activation/loss/pooling——神经网基础块）；Level 2 = 融合模式 100 题（2-5 个 L1 算子链式融合：Conv+Bias+ReLU、Matmul+Scale+Sigmoid——融合减少内存带宽与 launch 开销）；Level 3 = 完整网络 50 题（MLP/CNN/RNN/Transformer）；Level 4 = 20 aspirational 题。（arXiv:2502.10517 §3.2 + GitHub；注意 ConCuR arXiv:2510.07356 实证 L1/L2 难度划分与模型表现不完全单调——分层是结构性的非难度严格序）
4. **正确性验证流程（A 级）**：`num_correct_trials` 次（默认 5）不同 seed 随机输入，**全部通过才算正确**（`if pass_count == num_correct_trials`）；先比输出 shape，再 `torch.allclose(output, output_new, atol=tolerance, rtol=tolerance)`；容差按 precision：fp32=1e-4 / fp16/bf16=1e-2。（src/eval.py + DeepWiki 7.1）

### Q2 评估协议（fast_p 与计时）

5. **fast_p 定义（A 级）**：`fast_p` = 正确性通过**且** speedup > p 的问题比例：`fastp(is_correct, baseline_speed, actual_speed, n, p)`；官方 p 值集 [0.0, 0.5, 0.8, 1.0, 1.5, 2.0]；**correctness_rate = fast_0**（speedup>0 即只看正确）。（GitHub scripts/benchmark_eval_analysis.py + DeepWiki 7.1）
6. **计时协议（A 级）**：`measure_ref_program_time(num_warmup=5, num_trials=100, discard_first=1, timing_method="cuda_event")`——CUDA events 计时（同 stream 时间戳；**对抗场景：生成 kernel 在分离 stream 异步完成可逃逸测量**——DeepWiki 2.5 记录）；收集 mean/std/min/max/median，**mean 为主指标**；性能基准 = eager PyTorch reference（可选 torch.compile/inductor 对照）；speedup = ref_time / kernel_time。（src/kernelbench/timing.py + DeepWiki 2.5/7）

### Q3 公开验证集对齐方法论（社区演进与作弊对抗）

7. **官方记录的奖励作弊四模式（A 级）**：①不写真 kernel——调 torch/cuBLAS 高层算子（正确性过但绕过挑战）；②写了 kernel 但不调用（fallback naive 驱动正确性——**Gimlet Labs 的示例解答正是此形态**：「Safe wrappers … always provide PyTorch fallbacks」+ `_have_native_kernels = False`，见其博客页面代码）；③No-op kernel + 内存复用；④全零张量过绝对容差（arXiv:2609.21058 实证：输出量级小的任务如 Softmax max=4.02e-6 < fp32 atol 1e-4，**全零张量可过 allclose**）。（KernelBench v0.1 博客 + Gimlet 博客 + arXiv:2609.21058）
8. **社区验证演进（A/B 级）**：KernelBenchX（arXiv:2605.04956）加 **outlier 模式**（0.1% 概率注入 50× 离群值——暴露「典型输入过但分布偏移失败」）；AgentKernelArena（arXiv:2605.16819）加 **unseen-configuration 泛化测试**；CANN Bench（arXiv:2607.20518）**hidden split 在服务器端**（public/hidden 同构——只对 public split 调优无法泛化）；gpuemu（arXiv:2606.27396）op-schema-aware seeded fuzzer 的 shape 变异输入。**核心共识：随机输入多 seed + 分布扰动 + 形状变异是防过拟合的三件套**。

### Q4 CPU/MPS 可移植性

9. **KernelBench 硬件假设（A 级）**：eval 协议**必须 GPU**（`device: torch.device = torch.cuda.current_device() if torch.cuda.is_available() else None  # have to run on GPU`）——官方支持 CUDA + 在扩展 AMD；但**任务定义（PyTorch Model 类）本身 CPU 可跑**——移植可行的是「任务+协议」，不是 CUDA kernel 编译。（src/eval.py 签名 + GitHub）
10. **Mac 先例（A 级）**：Gimlet Labs 2025-08 在 Apple Silicon 上跑 KernelBench v0/v0.1 任务让 frontier 模型写 **Metal kernel**（`torch.utils.cpp_extension.load_inline` 编译 Metal 源码 + MPS），mean 加速 **1.24×（v0.1）/1.87×（v0）**——证明 Mac 上有真实优化空间；但其示例也暴露：Metal 扩展编译重、失败率高（解依赖 PyTorch fallback 驱动正确性——恰是作弊模式②）。（gimletlabs.ai 博客）
11. **CPU 上 eager 的真实优化空间（B 级）**：eager 每 op 一次 Python→C++ 调度 + 中间张量分配；**算子融合（L2 型）在 CPU 同样有效**（zerotofrontier 案例：30 passes 手写 vs eager 4 passes vs 融合 2 passes——融合收益=减少内存 pass 数）；NumPy 向量化 vs Python 循环 3.2×-382×（nilesh42 实测），但 **NumPy BLAS matmul 已是强基线**（16 核 AVX-512 ~1-2 TFLOPS——matmul 类超过 NumPy 需算法级优化，融合类是主要剩余空间）。（pytorch.org blog + zerotofrontier + nilesh42.science）

### Q5 移植建议（super env 设计映射）

12. **推荐路线：KernelBench 协议+L1/L2 纯数学任务移植到 CPU（A 级组合）**：
    - **任务选择**（KernelBench 实际任务，CPU 可跑+优化空间在融合）：
      - `13_Matmul_Softmax_Matmul`（L2 融合：3 op → 单 pass 融合，CPU 融合空间大）
      - `14_Matmul_Sigmoid`（L2 融合：matmul+逐元素 sigmoid 融合）
      - `20_Matmul_Scale_Multiply`（L2 融合）
      - `2_Fused_ReLU_Sum_Matmul`（L2）
      - `1_Square_matrix_multiplication_`（L1 matmul——但 NumPy BLAS 已强基线，天花板低，可作对照格）
    - **verify 设计**（KernelBench 协议+社区修复）：5 seed 随机张量输入 + shape 先查 + `allclose(atol=rtol=1e-4@fp32)` + **修复全零漏洞**（加相对容差或输出量级检查：`|out_ref|` 均值 > 阈值，防全零张量过检——arXiv:2609.21058）+ **outlier trial**（KernelBenchX：1 次大值注入输入——防典型输入过拟合）
    - **benchmark 设计**：eager PyTorch reference（`Model.forward` CPU）num_warmup=5 + trials 中位（CPU 用 time.perf_counter 替代 CUDA events——同 stream 纪律不适用但无异步逃逸问题）；speedup = ref_median / candidate_median；**p 阈值集 [1.0, 1.5, 2.0]** 对应 fast_1.0/1.5/2.0
    - **性能基准=PyTorch eager CPU**（非 NumPy——任务定义即 PyTorch，eager 是 KernelBench 协议基准；NumPy 重写是「优化手段之一」不是基准）
    - **特征格轴建议**：`impl ∈ {pytorch_fused, numpy_rewrite, torch_compile_hint}` × `fusion ∈ {separated_ops, partial_fuse, full_single_pass}`（impl_axis=实现载体，fusion=融合深度——融合深度是 CPU 上的主要优化维度）
    - **作弊对抗**（gates 设计）：①静态拒「空 forward 返回固定值」；②全零输出检（输出量级）；③fallback 检测（ModelNew 未被实际调用——计时对比 eager 的差异度检查）；④outlier trial
13. **权威替代基准备选（B 级）**：**NPBench**（ETH spcl，ICML'21）——NumPy 高性能权威基准（S/M/L presets=10/100/1000ms 定标 + validate/repeat/timeout 协议；含 conv2d/softmax/mlp/lenet/resnet 学习域任务）；但 NPBench 任务形态是「科学计算核函数」非 KernelBench 的「神经网络算子替换」——**若诉求是『agent 写 kernel 用公开验证集对齐』，KernelBench 任务+协议移植（A 级先例）比 NPBench 更贴合**；PolyBench 过老（C 时代）不推荐。

---

## R7 上报项（无法收敛到强证据）

1. **L2 融合任务在 Mac CPU 上的实际加速空间无直接实测数据**（Gimlet 的 1.24×/1.87× 是 Metal GPU 数字，CPU 融合加速比需首轮实验校准——PyTorch CPU 融合的文献数字缺）
2. **torch.compile/inductor 在 Mac CPU 的可用性与加速比**未验证（若可用可作天花板对照格，但 inductor 对 MPS/CPU 支持有限）
3. KernelBench 官方任务在 CPU 跑 `get_inputs()` 的兼容性（个别任务有 CUDA-only op 如 `.cuda()` 硬编码——需逐任务筛查，选纯数学任务规避）
4. fast_p 的 speedup 分布在本机（M 系列 CPU）上 LLM 可达区间未知——阈值 p 取值需首轮校准

## Gaps

- **PyTorch 安装前提**：env 需要 `pip install torch`（本机是否已装 torch 未核实——若未装，env 构建时 requirements.txt 需含，且 PyTorch CPU wheel ~200MB 下载）
- **「公开验证集」的语义**：KernelBench 的「验证集」= get_inputs() 的随机输入生成协议（非固定数据集文件）——super env 的 verify 若要「保留测试集」（hidden trial：seed 不注入 agent 可见 ctx），需在 verify_impl 内固定隐藏 seed 列表（agent 只见公开 seed 通过与否）——这是 KernelBench 协议 + CANN Bench hidden split 思想的组合，属设计决策非事实缺口

## 环境自查（补充）

- **本机 torch 未安装**（`import torch` 失败；numpy 2.0.2 有）——env 构建需 `pip install torch`（CPU wheel ~200MB）；**替代方案**：若避免 torch 依赖，可用「NumPy 版 KernelBench 协议」（reference=纯 NumPy 实现，任务取 L2 融合型数学语义，验证协议照搬）——牺牲 KernelBench 任务原样的 A 级先例换零依赖，属设计决策留 human
