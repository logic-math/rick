# env_v2 — v2 布局示范模板（多维融合分 + gates/ 验证器隔离 + 验证金字塔）

> job_56/task2 示范模板。基于 matmul 领域，展示 v2 env 四部分结构。
> gen_env skill（四步构建流程）按本结构生成新 env。

## 目录结构（v2 四部分）

```
env_v2/
├── domain_knowledge.md      # ① 领域专家知识（所有实验 agent 的文档入口与基本准则）
├── env.json                 # v2 配置（objective 多维目标/agent_hyperparams/env_slots/verify_levels）
├── feature_space.json       # 特征格轴定义
├── requirements.txt
├── skills/                  # ② agent 入口 skill
│   ├── profile.py           #    现状剖析（agent 直接实现）
│   ├── verify.py            #    约束门薄入口 → 执行引用 gates/verify_impl.py
│   ├── benchmark.py         #    基准门薄入口 → 执行引用 gates/benchmark_impl.py（支持 --level 分层验证）
│   └── research.py          #    检索配置（agent 直接实现）
├── gates/                   # ③ 验证器目录（builder/coding/debug 禁读内容；adjust 可读写）
│   ├── gate_check.py        #    v2 验证器入口（框架必然调用：多维融合分协议）
│   ├── verify_impl.py       #    验证实现（skills/verify.py 执行引用）
│   ├── benchmark_impl.py    #    基准实现（skills/benchmark.py 执行引用）
│   ├── exploit_registry.json  # 门先证红注册表（已知作弊样本）
│   └── golden_examples/     #    金样例（参考实现副本，门先证红零误杀）
├── code/                    # ④ 基线代码（进化 target）
│   ├── reference.py         #    参考实现（正确性锚点 + 首个金样例）
│   └── naive_start.py
└── data/                    # 薄转发（legacy 兼容，唯一实现在 gates/ 与 skills/）
```

## v2 关键协议

1. **多维融合分**（env.json `objective`）：约束列表 + 加权指标列表 →
   `gates/gate_check.py` 校验约束、加权融合出单一 score（越高越好）。
   score ≥ 0 ⟺ passed；score < 0 = 拒绝（-1 环境 / -2 正确性 / -3 约束违反 / -4 软门禁劣化）。
2. **验证器隔离**：`gates/` 内容对 builder/coding/debug 保密（superopt-json-guard hook
   拦截读取）；只允许经 `skills/verify.py`、`skills/benchmark.py` 薄入口**执行引用**；
   adjust 可读写（实验自改进的验证器反思走提案队列，human 裁决后才改）。
3. **验证金字塔**（env.json `verify_levels`）：L1 秒级 → L3 分钟级 → E2E 端到端；
   verify/benchmark 薄入口透传 `--level` 参数（实现侧按级别裁剪规模/次数）。
4. **subagent 超参数**（env.json `agent_hyperparams`）：adjust 流水线每步的
   {model, thinking} 配置位（null = 跟随默认）。
5. **资源槽位**（env.json `env_slots`）：同环境并发实验数声明（本 env = 1：
   benchmark 独占计时）；同环境分时复用规则见 domain_knowledge.md。

## 演练（基础设施验收）

```bash
cd <env_dir>
PY=.venv/bin/python   # 或 python3（无 venv）
$PY skills/verify.py     --candidate code/reference.py            # {passed: true, ...}
$PY skills/benchmark.py  --candidate code/reference.py            # {runtime_ms, speedup, score, ...}
$PY gates/gate_check.py  --candidate code/reference.py            # {passed: true, score >= 0, ...}
```
