# matmul 领域知识（v2：多维融合分 + 验证金字塔 + gates/ 隔离）

> 面向进化 agent 的领域速查与**基本准则**（本文件是所有实验 agent 的文档入口）。
> 矩阵乘（CPU、float32、512×512）在 CPython/NumPy 语境下的性能结构与调优路径。
> 正确性锚点 = `code/reference.py`（`np.matmul`，底层 BLAS）。

## 任务定义与多维优化目标

- 输入：固定 seed 生成的两个 `n×n` `float32` 矩阵（n=512 性能门 / n=64 正确性门）
- 输出：矩阵积 `c = a @ b`，形态 `(n, n)`，`max_error ≤ 1e-3`（约束，见下）
- **多维优化目标（env.json `objective`，v2 融合分协议）**：

```json
"objective": {
  "constraints": [{"name": "max_error", "op": "<=", "value": 0.001}],
  "metrics": [{"name": "speedup", "weight": 1.0}],
  "fusion": "weighted_avg"
}
```

- **约束**：`max_error ≤ 1e-3`——违反 → `gates/gate_check.py` 返回 score=-3（stage3_perf 拒绝）。优化目标可以是多维的：**约束（如 A>1、max_error≤1e-3、tp99≤100ms）之下，追求多指标（qps/ttft/speedup…）的加权和最大**——本 env 示范单指标 speedup；benchmark 输出多指标时在 `objective.metrics` 加权即可（如 `speedup 0.7 + memory_efficiency 0.3`）。
- **融合分（进化分数）**：score = Σ(weight×指标) / Σ(weight)，**越高越好**（低优方向指标由 gate_check 派生归一——如 speedup = ref_runtime/candidate_runtime）。score ≥ 0 = 通过；score < 0 = 拒绝（-1 环境错误 / -2 正确性失败 / -3 约束违反 / -4 软门禁劣化）。

## 环境结构（v2 四部分，所有 agent 必须知晓）

| 部分 | 路径 | 用途 | 访问边界 |
|---|---|---|---|
| **领域专家知识** | `domain_knowledge.md`（本文件） | 所有实验 agent 的基本准则与文档入口 | 全员必读 |
| **skills** | `skills/` | agent 入口 skill（profile/benchmark/verify/research；verify/benchmark 是**薄入口**→执行引用 `gates/` 实现） | 全员可调用 |
| **gates 验证器** | `gates/` | 验证器入口与实现（gate_check.py + verify_impl/benchmark_impl） | **builder/coding/debug 禁读内容**（可执行）；adjust 可读写维护 |
| **code 基线** | `code/` | 基线代码 = 进化 target | 全员可读写（实验候选） |

**验证器保密纪律（硬约束）**：builder/coding/debug **不得读取 `gates/` 目录任何文件内容**（read/cat/grep 等——superopt-json-guard hook 拦截）；只允许经 `skills/verify.py`、`skills/benchmark.py` 薄入口**执行**拿结论。理由：防止针对验证器做定向优化/作弊（涨幅异常会触发 debug 作弊检查）。adjust agent 例外——step7 实验自改进需要读写验证器（识别机制性错误/噪声/过拟合，经提案队列走 human 裁决）。

## skill 清单（务必按场景调用）

| skill | 命令 | 何时调用 |
|---|---|---|
| `profile` | `python3 skills/profile.py --candidate <path>` | 设计实验前剖析父本（热点/环境事实/代码结构） |
| `verify` | `python3 skills/verify.py --candidate <path> [--level L1]` | 判断候选是否合法（正确性容差）——秒级排除无效解 |
| `benchmark` | `python3 skills/benchmark.py --candidate <path> [--level L3]` | 拿 K 次中位耗时与 score——分钟级自证 |
| `research` | `python3 skills/research.py --role <role>` | 检索外部方法（论文/文档），design 落盘前 |
| `gate_check` | `python3 gates/gate_check.py --candidate <path> [--baseline <score>]` | **框架终验专用**（E2E）；agent 一般用 verify/benchmark 分层预检即可 |

## 验证金字塔（分层验证矩阵——以最低成本排除无效解）

| 层级 | 参数 | 命令 | 耗时 | 用途 |
|---|---|---|---|---|
| L1 | `--level L1` | `skills/verify.py --candidate <p> --level L1` | 秒级 | 单规模正确性快检（无效解秒级排除） |
| L3 | `--level L3` | `skills/benchmark.py --candidate <p> --level L3` | 分钟级 | 全量基准 K 次中位（自证改进） |
| E2E | （默认） | `gates/gate_check.py --candidate <p> [--baseline <s>]` | 分钟级 | 框架终验：verify+benchmark+约束校验+融合分+软门禁 |

**纪律**：实验计划先推演风险（builder 仿真预演段），可先用 L1 预检关键假设（分钟级以内），再交框架 E2E 终验——**无效解以最低成本排除**，不烧端到端预算。

## 资源与分时复用规则（调度器仅管资源分配）

- 同一 env 允许 `env_slots` 个实验并发（本 env = 1：matmul benchmark 计时对独占 CPU 敏感——并发会互相污染测量）。
- **同环境分时复用是 ENV 的设计责任**：本 env 的 benchmark 为独占测量设计（env_slots=1）；若 env 声明 env_slots>1，env 设计者须在本节写清并发实验如何交错使用独占资源（分时复用/资源池/独立数据副本）。
- 调度器只管资源位分配（实验槽/环境槽），不介入实验内部的资源交错。

## 特征格实验指引（6 格精选空间，3 策略 × 2 布局）

> 格手术依据（env_7 实测 score 阶梯）：naive 三重循环 2.1e-5 → 逐行 np.dot **0.153**
> （LLM 一步可达的跳变，~7000×）→ 整阵 BLAS **0.98-1.00**（天花板族）。
> strassen / 多进程 / 线程池 / einsum 系在本机 512×512 实测**劣化**（死格族已裁除）；
> >1.0 需绕开单调用 SME 路径，本机无实测可达方案，不作默认目标。

### impl_strategy 轴（算法结构）

- **blocked_loops（分块循环）**：把 n×n 切成块（如 64×64），块内乘积装进 L1/L2。
  纯 Python 收益有限，但它是向量化/BLAS 路径的**前置结构**——分块边界预计算、
  行引用提升（`ai = a[i]`）、循环序调优在这里做。适合作为「结构性改进」的实验格。
- **rowwise_vectorized（逐行向量化）**：`c[i] = np.dot(ai, b)` 行向量×矩阵——
  speedup 0.153 的已验证跳板。深挖方向：批量行处理（`np.dot(a[i:i+8], b)`）、
  避免逐行调度开销。
- **full_blas_dispatch（整阵 BLAS 派发）**：`np.matmul(a, b)` / `a @ b` / ctypes 直调
  cblas_sgemm——**speedup ~1.0 的天花板族**。深挖方向：out= 预分配、dtype 全程
  float32、ctypes 走 Accelerate 的调用开销消减。⚠️ 与参考全同的候选 speedup 在
  0.977-1.002 波动（±3% 噪声）——软门禁带 5% 匹配。

### memory_layout 轴（数据排布）

- **contiguous_rows（C 连续）**：默认行主序——BLAS sgemm 零拷贝路径。
- **preblocked（预分块排布）**：候选自管分块副本（`as_strided`/reshape），构造开销
  与主循环摊销的权衡实验。

## 常用命令（venv 解释器）

```bash
PY=.venv/bin/python
$PY skills/profile.py    --candidate <path>              # 现状：hotspots/env_facts
$PY skills/verify.py     --candidate <path> --level L1   # 约束：秒级 {passed, max_error}
$PY skills/benchmark.py  --candidate <path> --level L3   # 目标：{runtime_ms, speedup, score}
$PY gates/gate_check.py  --candidate <path> [--baseline <score>]   # E2E 终验（框架同源）
```

（`data/bench.py`、`data/test_correctness.py` 保留为薄转发，仅兼容旧调用方；
verify/benchmark 唯一实现位于 `gates/`，agent 经 `skills/` 薄入口执行引用。）
