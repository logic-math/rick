重构 rick 的 README.md 产出一份完整的学习文档,包括 human-loop 的设计，plan doing learing dream ctrl 的设计

---

## Grilling 澄清结论（2026-07-11）

### 文档目标
让不了解 rick 的人读完之后:
1. 知道 rick 解决什么问题、提供什么价值
2. 认可 rick 的使用价值
3. 知道怎么使用

组织方式隐式遵循 SENSE 方法论(问题→视角→判断),但不显式分小节,直接一段话行文。

### 顶层结构(pipeline)
1. 标题 + 一句话定位
2. **架构图**(Mermaid flowchart,4 层:人类入口 / 执行 / 知识 / 进化)
3. **设计哲学**(一段散文,隐式 SENSE,覆盖问题→视角→判断)
4. **双维度知识体系**(表格:loops/skills/domain/draft)
5. **快速开始**(两种模式:easy 白箱 / doing 黑箱)
6. **命令体系**(7 节,统一模板)
7. **human-loop 设计**(独立大节)
8. **目录结构**(两层树,标注维度归属)
9. **配置参考**
10. **版本演进**(v2.9.0 → v2.10.9)

### 关键决策
- **架构图**:Mermaid `flowchart TB`,4 层分层
- **设计哲学**:一段话行文,不显式分"问题/视角/判断"小标题
- **双维度知识体系**:
  - 执行维度:loops(可复用工作流,带验收标准的迭代控制流)、skills(原子能力,触发条件→执行步骤)
  - 价值维度:domain(代码事实的客观描述)、draft(个人判断,human-loop 思考记录/RFC)
  - domain/draft 边界:domain 可被代码验证;判断一旦被代码固化可迁移到 domain
- **命令体系**(7 个,不含 tools):
  - plan / doing / easy / learning / dream / ctrl / human-loop
  - **rick easy 是 rick doing --easy 的子模块**:两者共用同一套 easy 函数(runEasyMode/resumeEasyMode),`rick easy` 是 `rick doing --easy` 的等价入口。doing 章节为父级,easy 章节说明从属关系
  - 每命令统一模板:职责 / 用法 / 关键 flags(表格) / 示例 / 产出
- **human-loop 专题**:独立大节,含定位(SENSE 方法论)、调用图(Main Agent ↔ sense_subagent,Mermaid sequenceDiagram)、目录结构(draft/loops/loop_N/{prompts,briefs}/)、示例、与 doing 对比表(白箱/黑箱)
- **ctrl**:独立成节,定位为"黑箱执行的可挂测性设计"(四种干预场景表格)
- **dream**:独立成节,定位为"自我进化"(跨 job 反思)
- **目录结构**:两层树,标注每个目录的维度归属(执行/价值/工作区)
- **配置参考**:基于 config.go 实际字段(max_retries / claude_code_path / default_workspace / git.user_name / git.user_email)
- **版本演进**:只列 v2.9.0 → v2.10.9 大版本,旧架构(v1.x / v2.0-v2.8)不提
- **删除**:旧 README 的"完整示例"小节(冗余)
- **不提及**:旧架构 wiki/tools/SPEC.md/OKR.md 的演进历史

### 命令 flags(代码核实)
- plan: `--job`(全局)、`--dry-run`
- doing: `--job`、`--easy`、`--ctx`、`--dry-run`
- easy: `-r/--requirement`、`--ctx`、`--resume`、`--dry-run`
- learning: `--job`、`--dry-run`
- dream: `--job_num`(默认5)、`-p/--background`、`--dry-run`
- ctrl: `--job`(必传)、`--dry-run`
- human-loop: `<topic>`(必传)、`--dry-run`

### 当前 README 问题
- 停留在 v2.1.1,描述 wiki/tools/SPEC.md/OKR.md 等已删除的旧架构
- 未覆盖 v2.9.0 的三层知识体系(loops/skills/domain)迁移
- 未覆盖 v2.10.x 的 draft 概念、human-loop 设计、ctrl 命令
