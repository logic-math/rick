# task30：gen_env skill 三要素追问 + gate_check 契约 + 轨迹读取（super-loop）

## 任务目标
把「目标/约束/现状」三要素变成 **gen_env 的强制追问环节**（human 亲自描述，判据=可转成脚本），并把 builder 决策落盘与 profile 历史轨迹纳入中控会话的轨迹读取。

## 关键结果
- `.rick/skills/gen_env_skill/skill.md`：
  - 新增**第 0 节「三要素追问（强制前置）」**：
    - **目标**：要优化什么指标、到什么程度算赢、怎么测量（判据：**能写成 `skills/benchmark.py`**——输出 `{perf_type, score, timings, k}`；含 K 次计时与 score 定义）
    - **约束**：什么绝对不能破坏（正确性容差/接口/资源/时间），失败怎么判（判据：**能写成 `skills/verify.py`**——输出 `{passed, max_error, stage}`）
    - **现状**：当前基线是多少、热点在哪、环境事实（BLAS 后端/内存带宽/numpy 版本/父本结构）、**现状如何变化**（判据：**能写成 `skills/profile.py`**——输出事实 JSON）
    - **纪律**：必须由 **human 亲自描述**三者（AI 只做追问与复述确认，不代填）；三者都清晰、可转脚本后才进入 env 生成
    - 每要素附 2-3 个「追问样例」（如「你说的‘快’是哪个指标？怎么测？测几次取什么统计量？」）
    - 附**非 matmul 泛化示例**（如 prompt 优化：目标=评测集分数、约束=格式合法性、现状=当前 prompt 与失败样本分析）
  - 生成清单改为：`skills/{profile,benchmark,verify,gate_check}.py` 四件（**不再生成 self_check.py**）；gate_check 是三者封装（框架与 agent 共用唯一入口）
  - 验收自检清单增：三要素各自独立可跑（跑 reference 出合法 JSON）+ gate_check 融合 pass + reference 过 verify
  - 第 7 节（轨迹分析）增证据源：`<ws>/designs/`（builder 决策落盘：论文/灵感/推理/计划）+ `<ws>/profile_history/`（每格 profile 时间序列）——进化轨迹分析优先读它们
- `.rick/loops/super-loop.md`：
  - Step 2（env 建设）增：三要素追问环节（先于生成）+ gen_env 完成即推进 start（不得停在 env 生成后——env_9 实测：生成完未启动无轨迹）
  - Step 3（监控）/Step 7（轨迹复盘）增：读 `designs/` 与 `profile_history/` 了解进化轨迹与现状变化；三要素被 agent 实际调用的证据（events/prompt 快照）
  
# 写域
- .rick/skills/gen_env_skill/skill.md
- .rick/loops/super-loop.md

# 依赖关系
- 无（L1，与 task27 并行；契约由本 task 定义、task28/31 消费）

# 测试方法
关键词断言：gen_env 含「三要素」「benchmark.py」「verify.py」「profile.py」「gate_check」「human 亲自」「profile_history」「designs」；super-loop 含「三要素追问」「不得停在」「designs」

# 禁止事项
- 不改框架代码；不改 super/prompts（task29）
