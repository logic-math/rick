# task2 — M2 env 体系：多维融合分协议 + gates/ 隔离 + gen_env 四步构建 + 验证金字塔

> job_56 · DAG L1（与 task1 并行，写域互斥）· 无依赖
> 源设计：grilling/design-tree.md KR5+KR6（含 Q5/Q6/Q8/Q9 裁决）
> 依赖协调：gates.py 对外接口（GateResult/verify 签名）保持兼容——task1 的 loop.py 调用零改动

# 任务目标

落地开放性 env 构建体系的**规则与模板层**：gates.py 多维优化目标融合分协议、env 四部分结构定义（含 gates/ 目录隔离规则与 subagent 超参数配置位）、gen_env skill 重写为四步构建流程（设计树追问→四 skill 构建方法→演练→验证金字塔分层验证）。

# 关键结果

1. **KR-A【gates.py 多维融合分协议】**：`super/superopt/gates.py` 扩展 gate_check 产物解析：
   - gate_check.py（env 提供）stdout JSON 协议升级：`{"passed": bool, "score": <融合分>, "metrics": {"<指标名>": <值>, ...}, "constraints": {"<约束名>": {"value": <值>, "target": <目标值>, "satisfied": bool}, ...}, "diagnostics": "..."}`——**score<0=未通过（负值大小可携带失败类别）**；score≥0=通过且为进化分数
   - **多维优化目标**：env 可声明多约束+多追求指标（如「A>1 前提下 B/C 越大越好」）——constraints 承载约束具体值（tp99/qps/ttft/tpot 等），metrics 承载全部实测指标；**融合分=多目标加权平均/融合公式**（env.json 定义 `objective`：约束列表+指标权重+融合函数选择）
   - `GateResult.metrics` 扩展承载 constraints/metrics 字典（向后兼容：单指标路径照旧）；级联门禁 stage1/2/3 与中位数逻辑保持——stage3 的 score 语义切换为融合分
   - env 侧 main 函数实现协议写入 gates.py docstring（gen_env 生成的 gate_check.py 遵循）
2. **KR-B【env 四部分结构定义】**：env 目录结构规则定稿（写入 gen_env skill 与 env_templates）：
   - `env/domain_knowledge.md`（领域专家知识——所有实验 agent 的基本准则与文档入口：任务背景/优化目标多维定义/验证金字塔矩阵/资源分时复用规则/skill 务必调用清单）
   - `env/skills/`（profile/benchmark/verify/research 四基本 skill + 可扩展；skill 支持分层验证参数——见 KR-D）
   - `env/gates/`（**验证器目录**：gate_check.py 入口+验证实现 py 文件；builder/coding/debug **禁读**（superopt-json-guard hook 拦截，task3 落 hook；本 task 落 prompt 纪律+目录结构）；adjust 可读写；verify/benchmark skill **直接调用 gates/ 内 py 引用实现**——执行引用而非内容复制）
   - `env/code/`（基线代码目录=进化 target）
   - `env/env.json`（新增字段：`objective` 多维目标定义/`agent_hyperparams`（每 subagent 的 {model, thinking} 配置位——Q3 裁决：超参数在 env 中配置）/`env_slots`（同环境并发实验数声明——同环境分时复用规则由 env 设计者在 domain_knowledge.md 描述）/`verify_levels`（验证金字塔定义））
3. **KR-C【gen_env skill 四步构建流程】**：`.rick/skills/gen_env_skill/skill.md` 重写：
   - 步骤①设计树追问：下钻追问得到完整 env 设计方案后生成（遵循 env 基本结构定义=KR-B）
   - 步骤②四 skill 构建方法：**追问中必须从 human 主动获得 profile/benchmark/verify/research 四个基本 skill 的构建方法**；四 skill 组合出 gate_check.py（**代码复用**：gate_check.py 调用 verify/benchmark skill 的实现，不重复实现）
   - 步骤③演练：逐 skill 真实执行+gate_check.py 真实运行——确保基础设施无问题、实验可稳定运行
   - 步骤④分层验证机制（验证金字塔）：引导 human 对验证分级——先分钟级快速验证论证改动有效性，再端到端验证
4. **KR-D【验证金字塔分层参数】**：verify/benchmark skill 约定**层次验证参数**：`L1-M0`（系统 L1 层 M0 模块单独验证）/`L{n}`（第 n 层整层验证）/`E2E`（端到端）；参数协议写入 domain_knowledge.md 模板（引导实验 agent 自行小范围验证、无效解以最低成本排除）；调度器 gate_check.py（E2E）与 agent 侧 L1 级预检（builder 可选调用）共享同一实现入口
5. **KR-E【env_templates 更新】**：`super/env_templates/` 全部模板按新结构重写（matmul 系列改造或新增一个示范模板 `env_v2/`——以最小可用示范为准）；模板含 domain_knowledge.md 骨架（多维目标/验证矩阵/资源分时复用规则节）+ gates/gate_check.py 骨架（多维融合分协议实现示例）+ env.json 示例（objective/agent_hyperparams/env_slots/verify_levels）
6. **KR-F【super-loop.md 更新】**：`.rick/loops/super-loop.md` 增补：env 四部分结构说明/gates/ 隔离规则（builder/coding/debug 禁读、adjust 可读写）/gate_check.py 负分协议/humans 两类队列（干预+提案）/验证金字塔——与 task1/3/4 的实现呼应（文档先行，实现引用）

# 写域

（文件级清单，与 task1 互斥）

- super/superopt/gates.py
- super/tests/test_gates.py
- super/env_templates（全部模板目录——可新增 env_v2 示范）
- .rick/skills/gen_env_skill/skill.md
- .rick/loops/super-loop.md

**禁改**：super/superopt/ 其余文件（loop.py/events.py 等归 task1/task3/task4）；super/prompts/（归 task3/task4）。

# 实现要点

1. **GateResult 兼容**：`GateResult(passed, stage, metrics, ...)` 字段与 `verify()` 签名不变；metrics dict 内新增 `constraints`/`objective`/`score_detail` 键——调用方（loop.py）对 metrics 的既有读取键（perf.value/perf.score/soft.tolerance_band）保持存在（单指标场景从多维中投影填充）。
2. **融合公式**：env.json `objective` 形如 `{"constraints": [{"name": "tp99", "op": "<=", "value": 100}], "metrics": [{"name": "qps", "weight": 0.7}, {"name": "ttft", "weight": 0.3}], "fusion": "weighted_avg"}`；gate_check.py（env 侧）计算融合分；gates.py（框架侧）解析+校验协议（score 数值性/passed 一致性：score≥0 ⟺ passed=true）。
3. **负分语义**：stage1/2 失败 → GateResult.passed=False + score=-1（环境错误）/-2（正确性失败）；stage3 融合分 <0 → 拒绝。进化分数=score（入格比较用，同 perf_desc 方向感知：env.json 声明 score 方向 `high`）。
4. **gates/ 隔离的模板侧落地**：env_templates 的 gate_check.py 骨架放在 `env/gates/`；domain_knowledge.md 模板含「验证器保密纪律」节（builder/coding/debug 禁读 gates/——hook 拦截由 task3 落地，模板先写纪律）。
5. **verify_levels 矩阵**：`{"E2E": {"cmd": "gate_check.py", "budget": "minutes.."}, "L1": {"cmd": "skills/verify.py --level L1", ...}}`——domain_knowledge.md 渲染成人类/agent 可读的验证矩阵表。
6. **演练步骤可执行**：gen_env 步骤③写成 checklist（每 skill 一次真实运行+一次 gate_check 运行+产物断言），供 super loop 中控执行。

# 测试方法（自测，过程性）

- RED：test_gates.py 先写新断言（多维 JSON 解析/约束违反→passed=False+score<0/融合分计算/兼容单指标路径）先跑红
- GREEN：gates.py 实现至全绿；现有 test_gates 断言全绿（级联/中位数/soft gate 回归）
- 模板演练：用 env_v2 示范模板跑一次 gate_check.py 骨架（手动 python3 执行）确认协议闭环
- 结构断言：gen_env skill 含四步/四 skill 追问/验证金字塔关键词；super-loop.md 含新增节

# 验收（gate1.py 承担）

`python3 .rick/jobs/job_56/plan/gates/gate1.py`（与 task1 同门禁）：test_gates 全绿 + gates.py 多维协议结构断言 + gen_env skill 四步结构断言 + env_v2 模板结构断言。
