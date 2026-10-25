# 设计树：super 调度升级为「实验驱动的四 Agent Loop」（迭代 8）

> 源需求（human 设计原则，2026-10-23）：adjust/builder/coding/debug 四角色按各自的 loop + skill 运行；实验记录 .md 作为跨 agent 恒常上下文；特征格四要素；adjust 四步（全局实验分析→SENSE→research→格管理）+ 决策记录；builder 调研报告+实验设计；coding 严格 TDD；debug 门禁不符/异常幅度复验。

## 第 1 层：OKR

**O（目标）**：
super 的调度从「四状态顺序调用」升级为「**实验驱动的四 Agent Loop**」——每个 agent 有自己的 loop 与 skill 组合，共享「**实验记录 .md**」这一恒常上下文；adjust 具备「全局实验分析 → SENSE 定主要矛盾 → research 找突破 → 特征格管理（新建/改进/调权）」的完整决策回路并留下可解释的决策记录；builder 产出调研报告+可复现实验设计；coding 以 TDD 实现；debug 在门禁不符/异常幅度/容差带内时复验并回写实验记录。全部系统状态可从事件流+实验记录恢复与解释。

**KR 集**：
| # | KR | 判据 |
|---|---|---|
| KR1 | **实验记录契约**：`codes/iter{N}-{uuid}/experiment_record.md`——builder 建（背景/价值/假设/验证逻辑/步骤/观测），coding 补（执行过程/测试/计划），debug 补（卡点/根因/解决） | 文件由三 agent 依次追加，事件可追溯 |
| KR2 | **特征格四要素**：`{name, description(方法描述), weight, score}`；description 为自由文本可引用论文/调研文档地址；新建格需正交性校验 | archive 落盘含 description；正交校验拒绝重复维度 |
| KR3 | **adjust 四步 loop + 决策记录**：全局实验分析（区间内全部实验记录 + 按需 session）→ SENSE（主要矛盾）→ research（按 research_brief 选关注点）→ 格管理（new_cell/modify_cell/weight）→ 决策记录 `<ws>/decisions/iter{N}.md` | 决策记录含四步痕迹；四类动作可达 |
| KR4 | **builder loop**：读选中格描述+精英代码+精英实验记录+**两个随机相邻格**（描述/精英代码/实验记录）→ research（builder 关注点）→ 调研报告 → 实验设计 skill → 实验记录 | 调研报告落盘；实验记录六段齐 |
| KR5 | **coding loop**：细化 research（落地细节/风险卡点）→ 编码执行计划 → **TDD**（测试红→绿→重构→保绿）→ 精英解；卡点→debug skill | 实验记录含 TDD 痕迹（测试文件/红绿记录） |
| KR6 | **debug loop 扩展**：触发条件=门禁失败 ∪ 性能提升超阈值 ∪ 软门禁容差带内；流程=复现→research 用户手册→三假设验证→修复/维持结论；回写实验记录 | 三类触发均可达；三假设记录在案 |
| KR7 | **skill 资产与加载规范**：仓级 `super_*` skills（实验分析/SENSE/格管理/实验设计/debug 协议）三段式；agent prompt 声明「skill 路径 + 触发条件」，按需 read | 五件 skill 存在；四模板均声明 |
| KR8 | **gen_env 修复 + 优化幅度追问**：流程重排（任务定义→研究实测→三要素→检索三问→门禁参数→特征空间→初始解→验收）、职责划分、前置输入（读 grilling/research 产物）、键一致性检查、preverify 自动化、`debug_policy` 追问 | gen_env 结构达标；env.json 含 debug_policy |

**OKR 充分性**：KR1 上下文载体 ✓ KR2 数据结构 ✓ KR3 决策中枢 ✓ KR4 设计 ✓ KR5 实现 ✓ KR6 质量兜底 ✓ KR7 方法资产 ✓ KR8 env 侧契约 ✓ —— 覆盖 O 全部主张。

## 第 2 层：四 Agent Loop 与触发/IO/skill 映射

### 2.0 触发时机与串行关系

```
py 调度器（每迭代）：
  ① [每 N 迭代] adjust —— 后台线程并行（不阻塞迭代）
       loop: 全局实验分析 → SENSE → research → 格管理 → 决策记录
  ② 加权随机采样格子（权重由 adjust 维护）
  ③ builder —— 前台串行
       loop: 读格/相邻格/精英 → research → 调研报告 → 实验设计 → 实验记录(建)
  ④ coding —— 前台串行
       loop: 细化 research → 编码计划 → TDD（红→绿→重构→保绿）
       内部卡点 → debug skill（同 agent 内 1..M 次）→ 实验记录(补)
  ⑤ 框架门禁 gate_check（正确性→性能→软门禁）—— 外部判定
       失败/异常幅度/容差带内 → debug agent（独立派发，attempt≥100，每候选 1 次）
       → 复现 → research 用户手册 → 三假设逐一验证 → 修复或维持 → 实验记录(补)
  ⑥ 入格 / 拒绝 → iteration_ended
```

### 2.1 各 agent 的输入 / skill / 产物

| Agent | 触发 | 读取（ctx 注入） | 使用 skill（按序） | 产出文件 |
|---|---|---|---|---|
| **adjust** | 每 `adjust_interval` 迭代边界（后台线程） | 区间内**全部实验记录**（上次 adjust→本次）、决策历史、cells_status（描述/权重/得分/visit）、perf 时间线、research_brief(adjust)、domain_brief、台账 | `super_experiment_analysis` → `super_sense` → research（brief）→ `super_cell_management` | `decisions/iter{N}.md`（思考+动作）、proposals/、weight_changed 事件 |
| **builder** | 采样后（前台） | 选中格 description/elite/visit/weight + **精英实验记录** + **两个随机相邻格**（description/elite code/实验记录）+ history_failures + axis_perf + research_brief(builder) + profile_history + designs_dir | research（brief，builder 关注点=实验细节/数据/方法）→ `super_experiment_design` | 调研报告 `research_report.md` + `experiment_record.md`(建) + `experiment_plan.json` + `designs/iter{N}_{cell}.json` |
| **coding** | builder 完成（前台） | experiment_plan + 实验记录 + builder 调研报告 + 三要素命令 + research_brief(coding) + gate_defs + perf_baseline | research（coding 关注点=API/落地细节）→ 编码计划 → TDD（写测试→红→绿→重构）→ 卡点则 debug skill | `coding_plan.md` + 测试文件 + solution.py + `experiment_record.md`(补) |
| **debug** | ① coding 自检失败（同 agent 内循环 1..M）② **框架门禁失败**（attempt=100）③ **性能提升 > gain_threshold_pct** ④ **软门禁容差带内** | failure_stage/metrics/stderr + 实验记录 + research_brief(debug=用户手册/正确用法) + gate_defs | 复现 → research（用户手册）→ **三假设逐一验证** | `fix_report` + `experiment_record.md`(补：卡点/根因/解决) |

### 2.2 skill 加载规范（J-i8-1 裁决）

**加载方式**：agent 的系统提示词（`super/prompts/<role>.md`）**声明 skill 清单（名称 + 路径 + 触发条件）**，agent 在执行到对应步骤时用 `read` 读取——**不内联全文、不自动全量加载**。
理由：①省 token（skill 全文 100-300 行，四角色全内联会撑爆 prompt）②skill 可独立进化（env 级/仓级覆盖机制现成）③符合 rick skill 规范（三段式 + 触发场景 → 由触发条件决定何时读）。

**skill 文件**（仓级，`.rick/skills/<name>/skill.md`，三段式）：
| skill | 服务对象 | 触发条件 |
|---|---|---|
| `super_experiment_analysis` | adjust | 每轮 adjust 开始（读区间实验记录与 session） |
| `super_sense` | adjust | 实验分析完成后（定主要矛盾 + 辩证逆转；方法论主体引用 rick `sense` canon） |
| `super_cell_management` | adjust | SENSE+research 完成后（新建/改进/调权的手续与正交性判据） |
| `super_experiment_design` | builder | 调研报告完成后（设计可复现实验 + 实验记录六段） |
| `super_debug_protocol` | debug | 进入 debug 时（复现→用户手册调研→三假设验证；方法论引用 rick `debug-skill` canon） |
| （复用）`test-driven-development` | coding | TDD 实现时（引用 rick tdd-zh canon） |
| （复用）`sense` / `debug-skill` | adjust / debug | 方法论 canon，super_* 中引用其路径 |

### 2.3 文件与数据结构

```
<ws>/
├── decisions/iter{N}.md                      # adjust 决策记录（思考全过程 + 动作 + 依据）
├── designs/iter{N}_{cell}.json               # builder 决策落盘（已有，增 research_report 引用）
├── profile_history/<cell>/iter{N}.json       # 现状时间序列（已有）
├── ledger.jsonl                              # 实验台账（已有）
└── codes/iter{N}-{uuid}/
    ├── experiment_record.md                  # ★实验记录（builder 建 / coding 补 / debug 补）
    ├── research_report.md                    # builder 调研报告
    ├── coding_plan.md                        # coding 编码执行计划
    ├── test_solution.py                      # TDD 测试（coding 产）
    ├── solution.py / experiment_plan.json / prompt_*.md
```

**特征格 schema（KR2）**：
```json
{"cell_id": "axis1=v1|axis2=v2",
 "description": {"name": "短名", "method": "方法描述（可引用论文/调研地址）", "source_refs": ["arxiv:2108.02025", "..."]},
 "weight": 1.0, "score": 0.93, "visit": 3, "elite": {...}, "history": []}
```
- `description.method` 由 adjust 的 `modify_cell` 动作改进；`new_cell` 新建时必填 + 正交性论证
- 正交性：新格必须引入**未被现有格覆盖的优化维度**（新轴值或新描述维度），校验记录在决策记录里

**实验记录 .md 六段（builder 建）**：
```markdown
# 实验记录 iter{N} · {cell_id}
## 1. 背景与价值（为什么做这个实验）
## 2. 实验假设（一句话可证伪）
## 3. 验证逻辑（什么结果支持/否定假设）
## 4. 具体实验步骤（可复现）
## 5. 观测现象（预期指标变化）
## 6. 执行过程与结论（coding/debug 追加：实现要点/测试红绿/卡点根因/门禁结果）
```

## 第 3 层：每个模块的内部实现（loop.py 改动点）

### 3.1 adjust 四步（`_run_adjust` 重写）
- ctx 增：`experiment_records`（区间内全部 `experiment_record.md` 内容摘要，超长截断）、`experiment_sessions`（区间内 agent session 路径清单，供按需 read）、`decision_history`（近 3 份 decisions 摘要）、`cell_descriptions`（各格 description）、`research_brief`、真实 `recent_events`
- 输出契约扩展（在 soft_weight/expand_cell 基础上）：
  - `new_cell`: `{type, action:"create", descriptor:{...}, name, method, source_refs, orthogonality_rationale}`
  - `modify_cell`: `{type, action:"improve_description", target_cell, method, source_refs, rationale}`
  - `terminate_recommend`（已有）
- 应用：`new_cell` → feature_space 热重载加值（已有 expand_cell 链复用）+ archive 建格（带 description）；`modify_cell` → archive 更新 description（事件 `cell_modified`）
- 决策记录：`decisions/iter{N}.md`（iteration、区间实验摘要、主要矛盾判定、research 引用、动作清单与依据、正交性论证）

### 3.2 builder（`_run_builder` 扩展）
- ctx 增：`target_record`（目标格精英实验记录）、`neighbor_samples`（**随机 2 个相邻格**：共享≥1 轴值的池中随机；不足则全库随机）含 description/elite code path/实验记录、`research_report_path`、`experiment_record_path`
- 产物：`research_report.md`（agent 写）、`experiment_record.md`（agent 写六段）、`experiment_plan.json`（扩展增 `hypothesis`/`validation_logic` 与实验记录对齐）

### 3.3 coding（`_run_coding` 扩展）
- ctx 增：`research_report`（内容）、`experiment_record`（内容）、`coding_plan_path`、`test_path`
- 纪律（模板）：细化 research → `coding_plan.md` → TDD（先写 `test_solution.py` 并看到红 → 实现 → 绿 → 重构）
- 产物：`coding_plan.md`、`test_solution.py`、solution.py、实验记录追加段

### 3.4 debug 触发扩展（`_verify_and_admit` 后）
- 现有：门禁失败 → debug（attempt=100，1 次）
- 新增触发：
  - `perf_gain_pct >= debug_policy.gain_threshold_pct`（默认 20）→ debug 复验（防作弊/误测的异常收益进地图）
  - `gr.tolerance_band`（软门禁容差带内）且 `debug_policy.recheck_tolerance_band` → debug 复验（无明显收益的复验）
- debug 输出：`three_hypotheses`（模板要求）+ fix_report；维持原解时记 `verdict: keep_original`

### 3.5 实验记录注入（跨 agent 恒常上下文）
- 每个 agent 派发前：若 `experiment_record.md` 存在 → 内容注入 ctx（coding/debug 必读；adjust 读区间全部）
- agent 完成后：loop 检查实验记录是否被更新（mtime）→ 未更新则在事件里标注 `record_updated: false`（提示质量缺口）

## 判断节点（默认裁决，供 human 复核）

| # | 节点 | 裁决 |
|---|---|---|
| J-i8-1 | skill 加载方式 | prompt 声明路径+触发条件，agent 按需 read（不内联/不自动） |
| J-i8-2 | 实验记录位置 | `codes/iter{N}-{uuid}/experiment_record.md` |
| J-i8-3 | 相邻格抽样 | 「共享 ≥1 轴值」邻格池随机抽 2；不足则全库随机补 |
| J-i8-4 | 决策记录位置 | `<ws>/decisions/iter{N}.md` |
| J-i8-5 | debug 阈值默认 | `env.json debug_policy: {gain_threshold_pct: 20, recheck_tolerance_band: true}`（gen_env 追问 human 确认） |
| J-i8-6 | 动作兼容 | 保留 soft_weight/expand_cell/hard_*；新增 new_cell/modify_cell |
| J-i8-7 | skill 主体复用 | SENSE/TDD/debug 方法论引用 rick canon 路径，super_* skill 只写场景适配（避免重复维护） |
