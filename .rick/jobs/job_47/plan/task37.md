# task37：gen_env skill 流程修复 + 优化幅度阈值追问

# 任务目标
修 human 认同的 3 偏差 + 2 缺口，并新增「优化幅度阈值」追问（human 确认；用于 debug 触发）。

# 关键结果
`.rick/skills/gen_env_skill/skill.md` 重构：
1. **流程重排**（偏差②）：`0. 任务定义（参考实现+naive）→ 0.5 领域研究/实测（跑 reference 拿基线/热点事实）→ 1. 三要素追问（此时有实测数据可依）→ 1.5 检索三问 → 2. 门禁参数（定量：容差/规模/K/score 语义）→ 2.5 优化幅度阈值（debug_policy）→ 3. 特征空间 → 4. 初始解 → 5. 验收自检`
2. **职责划分**（偏差①）：三要素节只做**定性收敛**（能转脚本），门禁参数节做**定量决策**（具体数值），开头写明「本节承接三要素，只定参数，不重复追问目标/约束/现状」
3. **前置输入**（偏差③）：开头增「先读 `<ws>/grilling/design-tree.md` 与 `research-*.md`（若存在）——三要素/检索三问的素材多半在那里，避免重复访谈」
4. **键一致性检查**（缺口①）：验收清单增「env.json 键 vs 脚本读取键一致性」——列出各脚本读取的 env.json 键（benchmark: bench.size/k/timeout_s；verify: correctness.sizes/tolerance；gate_check: tolerance.soft_gate.relative_band/gates.*），逐项确认配置存在（env_9 实测：脚本读 sizes 而配置只写 size → 自检失真）
5. **preverify 自动化**（缺口②）：验收清单的 exploit 拒识改为「`gate_check.py --preverify` 或框架 `CricketGate.preverify_red` 一次跑完」并注明输出判据（exploits 全拒 + goldens 全过）
6. **优化幅度阈值追问**（新）：问 human「多大比例的收益才算可疑（需 debug 复验防作弊/误测）？」「软门禁容差带内的解是否也要复验？」→ 落 `env.json debug_policy: {gain_threshold_pct: <human 定，默认 20>, recheck_tolerance_band: true|false}`；写进 env.json 契约说明
7. 五件 skill 清单更新（profile/benchmark/verify/gate_check/research）+ 与 task36 的 super_* skill 的关系说明（env 级三要素 vs 仓级方法 skill）

# 写域
- .rick/skills/gen_env_skill/skill.md

# 依赖关系
- 无（L1，与 task36 并行）

# 测试方法
`grep` 含「优化幅度」「debug_policy」「键一致性」「前置输入」「0.5 领域研究」；三要素节与门禁参数节职责描述不重叠

# 禁止事项
- 不改 gen_env 的既有术语（三要素/检索三问/五件）；不改代码
