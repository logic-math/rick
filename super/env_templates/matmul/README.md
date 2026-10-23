# matmul env 模板（三要素契约 + 检索资产）

矩阵乘优化任务的 env 模板。**结构遵循三要素契约**——每个 env 必备三个基石脚本
（人类亲自描述 → 由 gen_env 转成脚本），外加一个融合门禁封装与一类检索指引：

| 要素 | 泛化语义 | 脚本 | 输出协议（stdout 最后一行 JSON） |
|---|---|---|---|
| **现状** | 事实信息获取（现在是什么样） | `skills/profile.py` | `{ok, n, hotspots[{function,self_ms,cum_ms,pct}], env_facts{python,numpy,blas,cpu_count,memcpy_gbps,dtype}, parent_facts{candidate,loc,imports,functions,runtime_ms}}` |
| **目标** | 目标优化的评估方法（怎么算变好） | `skills/benchmark.py` | `{runtime_ms, score, timings[K], ref_runtime_ms, n, k}`；失败 `{passed:false, error, stage}` |
| **约束** | 优化过程中的约束（什么不能破） | `skills/verify.py` | `{passed, max_error, n, tolerance}`；失败 `{passed:false, error, stage}` |
| **融合门禁** | 三者级联 + 软门禁 | `skills/gate_check.py` | `{passed, failure_stage, metrics{correctness,perf,soft}, details}` |
| **检索** | 认知注入（复用前人经验，突破局部最优） | `skills/research.py` | `{role, cell, method, sources[], focus, suggested_queries[], discipline}` |

## 检索 skill（第四类资产：认知注入）

纯组内变异容易困在局部最优（同一格以小角度反复微调出不来）——检索是唯一能从
**外部**注入新原理的通道。`research.py` **不执行检索**（检索由 agent 的 web_search
完成），它把 human 定义的**检索三要素**按角色固化，约束 agent「怎么搜/去哪搜/关注什么」：

| 三元组 | 语义 |
|---|---|
| **搜索方法**（method） | 怎么搜：文档站内 / GitHub 代码搜索 / arXiv 与论文站 |
| **搜索信源**（sources） | 去哪搜：本域最可信来源清单 |
| **关注点**（focus） | 关注什么；**与特征格绑定**（`--cell` 入参时注入） |

**角色分工（human 裁决）**：

| 角色 | 方法 | 信源 | 关注点 |
|---|---|---|---|
| `coding` / `debug` | 产品文档与代码库检索 | numpy 官方文档 / BLAS 文档 / GitHub / Stack Overflow | 该格实现细节：API 用法与限制、已知坑 |
| `builder` | 论文检索（arXiv / Scholar） | arxiv.org / paperswithcode.com | 实验细节与数据：benchmark 配置、规模影响、硬件差异 |
| `adjust` | 论文检索（arXiv / OpenReview） | arxiv.org / openreview.net | 创新点与趋势：尚未覆盖的方法族（供 expand_cell） |

**定向与覆盖**：`--role <role> [--cell <cell_id>]`；默认三元组在本脚本 `ROLE_PROFILES`，
可由 `env.json` 的 `research.<role>` 字段级覆盖（`method`/`sources`/`focus`/`discipline`/
`query_template`/`extra_queries`）——gen_env 引导 human 为本域定制。

**注入方式**：调度器把 `research_cmd`（命令）与预生成的 `research_brief`（三元组 JSON）
注入 builder/coding/debug/adjust 的 prompt——agent 按 brief 检索而非自由发挥，
关注点与当前目标格绑定，检索结论必须落到可实现动作。

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
│   ├── gate_check.py     # 融合门禁（verify→benchmark→软门禁；框架与 agent 共用）
│   └── research.py       # 检索指引（三元组：方法/信源/关注点，按角色分工）
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
$PY skills/research.py   --role builder --cell "impl_strategy=blocked_loops|memory_layout=standard"  # 检索指引
$PY skills/research.py   --role coding  --cell "impl_strategy=full_blas_dispatch"                  # 文档/样例指引
```

## 迁移说明（legacy env）

旧 env（env_7/8/9）保持 `data/test_correctness.py` + `data/bench.py` + `skills/self_check.py`
布局，框架走 legacy 路由，不受影响。三要素化只对**新生成 env** 生效；如需升级旧 env：
按上表把两件 data 脚本迁移为 `skills/verify.py` / `skills/benchmark.py`、新增
`skills/profile.py` 与 `skills/gate_check.py`、删除 `self_check.py`，框架自动切新布局。
检索资产（`skills/research.py`）对旧 env **可选**：缺失时调度器不注入 research_brief，
agent 走「无检索」降级路径。
