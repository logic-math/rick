# 编码执行计划 · 迭代 15

格：`dispatch_target=np_matmul|impl_strategy=full_blas_dispatch`（父本 = iter003 精英 430.83ms）
单变量改动：**BLAS 线程池配置维度**——模块加载期 `VECLIB_MAXIMUM_THREADS=6`。

## 1. 改动点（父本结构 → 目标结构，逐处说明）

父本（61 行，已 diff 确认与 iter003 逐行一致；profile 复核 matmul self 100.0%、
asarray 0.0%、blas=accelerate、cpu_count=18）只有**一处**改动，其余逐行不动：

| # | 位置 | 父本 | 目标 |
|---|---|---|---|
| 1 | 模块 import 区 | `import json` / `import time` / `import numpy as np` | 增加 `import os`；在 `import numpy as np` **之前**插入 `os.environ.setdefault("VECLIB_MAXIMUM_THREADS", "6")`（附依据注释）；numpy/求和序/main 协议零改动 |

- **为什么是 6**：本机 Apple M5 Max = 6 P 核（`hw.perflevel0.physicalcpu=6`）+ 12 E 核。
  vecLib 默认撒满 18 核，sgemm floor 含 E 核 straggler 栅栏成本。builder 3 轮换序
  交错 A/B（probes/thread_ab.json）：v6 pooled floor **429.48** / median 433.16 vs
  default pooled floor 441.58 / median 455.49（深 ~12ms，~3%）；late6（numpy import
  后设置）floor 425.25——import 期设置同样生效（BLAS 池在首次 gemm 才惰性创建）。
- **为什么 setdefault**：不覆盖外部显式配置（benchmark/verify/框架均未预设该变量，
  探针已确认）；进程级、先于首次 BLAS 调用、对 candidate 与 ref **对称**
  （benchmark.py 先 import numpy 后 exec 候选模块、任何 gemm 都在候选加载之后——
  加固⑧的「只让 ref 慢」非对称通道不涉及，ref 同池受益落 [380,700] 带内）。
- **数值不受影响**：probes/final_confirm.json——default 与 v6 输出在 n=64 与 n=8192
  的 sha256(uint32 view) 完全相同（BLAS 线程切 M×N 输出域，单元素 K 点积单线程
  固定序）→ verify max_error 预期 0.0。

## 2. 风险卡点（已知的 API 限制 / 边界条件 / 性能陷阱，附来源）

1. **静态检查黑名单**（verify.py `_STATIC_PATTERNS` 源码审计）：`sys.modules` /
   `__main__.` / `time.<attr>=` / `signal.alarm=` / `__builtins__`——
   `os.environ.setdefault` 不命中任何模式（research_report §4 第 4 条同结论）。
   注意注释文本里不要出现这些字面模式。
2. **设置时机**：必须在首次 BLAS gemm 调用前。benchmark.py 的加载序 =
   import numpy → 生成 a/b（rand 非 BLAS）→ exec 候选（此处 setdefault）→ warmup
   （首次 gemm，池此时才创建并读 env）。late6 探针证明该时序生效；为最大稳妥仍按
   plan 放在 `import numpy as np` 之前（两位置在本协议下等价）。
3. **加固通道复核**（benchmark.py 源码）：score≤2.0（v6 下 ref/cand 同池，
   score≈1.0）；runtime≥0.01ms（~430ms 远离）；ref∈[380,700]（ref 同池 ≈429-460
   带内，争用窗 <700）；无子进程残留（不 fork）。全部不涉及。
4. **机器争用窗**（当日实测存在 580-590ms 全局窗，final_confirm.json proto 两臂
   均撞上）：门禁三件每件**至多跑一次、读数即记录**，撞窗如实标注、不复跑不追噪声
   （iter14 过程纪律）。「机制成立（floor 对照 + max_error）」与「录取与否（绝对中位
   vs 430.83）」分开记录——band 内读数框架按 worse_than_elite 拒是已知规则。
5. **测试进程环境污染**：`import solution` 的 setdefault 副作用是进程级的——测试
   里 env 断言须先 `os.environ.pop("VECLIB_MAXIMUM_THREADS", None)` 建立干净基线，
   setdefault 语义测试用独立子进程（预设 12 验证不被覆盖）。

## 3. 测试清单（每个测试断言什么；对应实验假设的哪一部分）

**区分性测试（父本必红 → 证明测试真的在测本次改动）**：

| 测试 | 断言 | 对应假设 |
|---|---|---|
| `test_module_sets_veclib_threads_to_6` | 干净基线下 `import solution` 后 `os.environ["VECLIB_MAXIMUM_THREADS"]=="6"` | 假设主句「模块加载期设置 6」 |
| `test_env_set_before_numpy_import` | 源码中 setdefault 行先于 `import numpy` | 假设「先于首次 BLAS 调用」的时序纪律 |
| `test_fresh_process_import_sets_env` | fresh 子进程（env 无该变量）import solution 后读到 "6" | benchmark 加载路径同款（fresh 进程生效） |

**语义/不变量测试（父本即绿，防走偏）**：

| 测试 | 断言 | 对应假设 |
|---|---|---|
| `test_external_config_not_overridden` | 子进程预设 "12" 时 import 后仍 "12" | setdefault 不覆盖外部显式配置 |
| `test_contract_list_input` | list 输入 → f32 ndarray (n,m) 逐位=锚 | 契约保持 |
| `test_bitwise_vs_blas_anchor` | n∈{64,300,512}×seed{42,123} 逐位=锚 | max_error=0.0（跨线程 SHA 证据） |
| `test_double_run_bitwise` | 同输入两遍逐位一致 | verify 加固⑦镜像 |
| `test_recompute_not_cached` | 换输入=新结果（非缓存） | 红线：预存结果 |
| `test_rectangular_shapes` | (37,53)×(53,29)→(37,29) 逐位=锚 | 泛化签名 |
| `test_non_2d_fallback` | 1-D×2-D 与 np.matmul 一致 | 泛化兜底 |
| `test_shapes_independent` | 不同形状输出互不别名、各自正确 | fresh 分配语义 |
| `test_single_matmul_dispatch` | 单次调用恰一次 np.matmul 派发、f32 入参 | 「唯一改动」纪律（kernel 结构零改动） |

## 4. 验证顺序（verify → benchmark → gate_check；每件至多一次，读数即记录）

1. TDD 红：`env/.venv/bin/python test_solution.py`（父本上 3 个区分性测试红）
2. 最小实现（唯一改动）→ TDD 绿：全部测试通过
3. 自检：`env/.venv/bin/python solution.py --n 8192` → 预期 `max_error_vs_matmul=0.0`
4. `env/.venv/bin/python env/skills/verify.py --candidate solution.py` → 预期
   passed=true, max_error=0.0（sizes [64,8192] 双 seed 两遍）
5. `env/.venv/bin/python env/skills/benchmark.py --candidate solution.py` → 预期
   timings floor≤436、score≈1.0（撞争用窗如实记录）
6. `env/.venv/bin/python env/skills/gate_check.py --candidate solution.py
   --baseline 430.8255829999998` → 软门禁 verdict（better/band/worse）如实记录
7. 第 6 段回写 experiment_record.md（候选 timings/min + default 同窗口对照
   [thread_ab.json: floor 441.58/median 455.49] + 框架判定分开记录）
