# matmul env 模板（三要素契约）

矩阵乘优化任务的 env 模板。**结构遵循三要素契约**——每个 env 必备三个基石脚本
（人类亲自描述 → 由 gen_env 转成脚本），外加一个融合门禁封装：

| 要素 | 泛化语义 | 脚本 | 输出协议（stdout 最后一行 JSON） |
|---|---|---|---|
| **现状** | 事实信息获取（现在是什么样） | `skills/profile.py` | `{ok, n, hotspots[{function,self_ms,cum_ms,pct}], env_facts{python,numpy,blas,cpu_count,memcpy_gbps,dtype}, parent_facts{candidate,loc,imports,functions,runtime_ms}}` |
| **目标** | 目标优化的评估方法（怎么算变好） | `skills/benchmark.py` | `{runtime_ms, score, timings[K], ref_runtime_ms, n, k}`；失败 `{passed:false, error, stage}` |
| **约束** | 优化过程中的约束（什么不能破） | `skills/verify.py` | `{passed, max_error, n, tolerance}`；失败 `{passed:false, error, stage}` |
| **融合门禁** | 三者级联 + 软门禁 | `skills/gate_check.py` | `{passed, failure_stage, metrics{correctness,perf,soft}, details}` |

## 双侧共用（消灭语义漂移）

- **框架侧**（`superopt/gates.py` 新布局）：stage1 检查四件齐备 + reference 过 verify；
  stage2+stage3 合并为调 `gate_check.py` 一次（框架只解析结论 + 传 `--baseline`）。
- **agent 侧**（builder / coding / debug）：调度器把四个命令注入 prompt，agent 随时调用
  ——**与框架拿到的结论同源**（不再有「自检过、框架拒」的漂移，env_8 it19 教训）。

## 目录

```
matmul/
├── env.json              # 任务定义：perf_type/perf_desc/tolerance/bench/correctness
├── feature_space.json    # 特征轴（6 格精选：3 策略 × 2 布局）
├── code/
│   ├── reference.py      # 参考实现（np.matmul；正确性锚点 + 金样例）
│   └── naive_start.py    # 简单起点（纯 Python 三重循环）
├── skills/
│   ├── profile.py        # 三要素①现状（热点/环境事实/父本结构）
│   ├── benchmark.py      # 三要素②目标（K 次计时中位 + score）
│   ├── verify.py         # 三要素③约束（正确性容差）
│   └── gate_check.py     # 融合门禁（verify→benchmark→软门禁；框架与 agent 共用）
├── data/                 # legacy 兼容薄转发（→ skills/ 对应脚本，不含逻辑）
├── domain_knowledge.md   # 领域知识 + 三要素用法 + 分析/调优 SOP
└── requirements.txt      # numpy（gen_env 引导装 env/.venv）
```

## 快速自验（venv 内）

```bash
python3 -m venv .venv && .venv/bin/pip install -r requirements.txt
PY=.venv/bin/python
$PY skills/verify.py     --candidate code/reference.py     # passed=true, max_error≈0
$PY skills/benchmark.py  --candidate code/reference.py     # runtime_ms/score/timings
$PY skills/profile.py    --candidate code/naive_start.py --size 128   # hotspots/env_facts
$PY skills/gate_check.py --candidate code/reference.py     # 首解豁免 passed=true
$PY skills/gate_check.py --candidate code/reference.py --baseline 0.001  # 软门禁拒（soft_gate）
```

## 迁移说明（legacy env）

旧 env（env_7/8/9）保持 `data/test_correctness.py` + `data/bench.py` + `skills/self_check.py`
布局，框架走 legacy 路由，不受影响。三要素化只对**新生成 env** 生效；如需升级旧 env：
按上表把两件 data 脚本迁移为 `skills/verify.py` / `skills/benchmark.py`、新增
`skills/profile.py` 与 `skills/gate_check.py`、删除 `self_check.py`，框架自动切新布局。
