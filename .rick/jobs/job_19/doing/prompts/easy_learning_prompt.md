# Rick Learning 阶段

你是一个资深的技术文档专家和知识管理专家。根据本次 job 执行过程，总结知识、经验和教训，沉淀可复用 skills。

## 执行上下文

**Job**: job_19

### OKR（任务目标）

<!-- 变更说明：本次 job_14 执行后更新
- 新增：O4 - 建立 act-path 进化循环（原因：v2.0 核心升级，通过程序性 NDJSON 解析建立负反馈机制）
- 新增：KR4.1 - act-path 自动生成（原因：doing 执行后需产出可机读的行为轨迹）
- 新增：KR4.2 - learning 六步 SOP + act-path 注入（原因：learning 需消费 act-path 提取优化信号）
- 新增：KR4.3 - dream 命令落地（原因：人工触发的进化层，消费 act-path + run_log）
- 新增：KR4.4 - core-skills 精准注入（原因：不同 SOP 阶段需要不同 skill 组合，避免信息污染）
- 修改：KR3.1 - 补充 rick dream（原因：核心命令扩展为四个）
-->
# OKR

**愿景**: 打造以促进人类深度学习、思考、表达为目的的可控人工智能系统。

## O1: 构建上下文优先的可控人工智能系统

Rick 的核心假设是：AI 的输出质量取决于上下文质量。通过结构化的上下文管理（SPEC、OKR、debug、skills、wiki），让 AI agent 在每次任务执行时都能获得完整、准确、可控的上下文，从而产出高质量的结果。

### 关键结果 (Key Results)

- KR1.1: doing 提示词自动注入 SPEC、已完成任务历史、debug 记录、项目 skills、项目 tools、job OKR，覆盖率 100%
- KR1.2: `rick tools plan_check` 能检测 6 类上下文结构错误，确保进入 doing 阶段的任务格式正确
- KR1.3: debug.md 作为强制工作日志，每次任务执行必须记录，确保失败上下文可追溯
- KR1.4: 任务重试时自动加载 debug.md 作为上下文，重试成功率相比无上下文提升可测量
- KR1.5: `projectRoot/tools/*.py` 自动扫描并注入 plan/doing 提示词，项目特定工具对 AI agent 可见

## O2: 构建使人成长、使 AI 进化的双循环学习引擎

每次 job 执行后，人类通过审核 learning 产出获得深度思考和总结的机会；AI 通过 skills/wiki 的积累在下次任务中获得更好的起点。两者形成正向循环，随时间共同进化。

### 关键结果 (Key Results)

- KR2.1: learning 阶段产出四类标准化文档（SUMMARY / skills / OKR / SPEC），每类有明确格式规范
- KR2.2: learning 产出经人工审核后手动合并到 `.rick/`，审核 SUMMARY.md 确认质量后逐文件 `git add` 提交
- KR2.3: `.rick/skills/index.md` 在下次 doing/plan 时自动注入提示词（优先于 .py 扫描），含触发场景描述，形成知识复用闭环
- KR2.4: 每次 job 的 SUMMARY.md 包含可量化的执行指标（完成率、重试次数、问题数量）

## O3: 构建开发者体验优先、生产级可用的 AI Coding 框架

Rick 应该足够简单，让开发者能在 5 分钟内上手；足够健壮，能在真实项目中稳定运行；足够通用，不绑定特定项目或团队。

### 关键结果 (Key Results)

- KR3.1: 核心命令只有四个（`rick plan` / `rick doing` / `rick learning` / `rick dream`），无需 init，自动初始化
- KR3.2: 核心模块（cmd/executor/prompt）单元测试覆盖率 ≥ 70%，集成测试覆盖所有 tools 子命令
- KR3.3: 移除所有硬编码项目名称，Rick 可用于任意 Git 项目，零配置启动
- KR3.4: 支持生产版（`rick`）和开发版（`rick_dev`）并行运行，用于 Rick 自我重构场景
- KR3.5: `--auto-fix` 标志为 opt-in 设计，check 命令默认行为确定性，可在 CI 中稳定使用
- KR3.6: plan/doing/learning/dream 的 `--dry-run` 标志输出完整 prompt 内容，便于调试和验证上下文注入效果

## O4: 建立可靠的 act-path 进化循环

通过程序性 NDJSON 解析建立 act-path 负反馈机制，使 learning/dream 层能够从真实行为轨迹中提取优化信号，形成"执行→观测→进化"的闭环，而非依赖 LLM 自觉记录。

### 关键结果 (Key Results)

- KR4.1: `rick doing` 执行后自动生成 `doing/tasks/{taskID}/act-path.md`，包含工具调用轨迹（含行号链接）、报错次数、执行时长，原始日志双写到 `raw_session.log`
- KR4.2: `rick learning` 升级为七步 RFC SOP，自动收集所有 act-path 内容注入 `{{act_path_content}}`，Step 2 评估更优轨迹，Step 6 写入 `.rick/dream/run_log_{n}.md` 度量文件
- KR4.3: `rick dream` 命令可运行，`--dry-run` 正常输出完整提示词，消费 act-path + run_log，执行 SENSE 反思和 evolve-skills 进化
- KR4.4: 8 个 core-skill 文件通过 `embed.FS` 编译进二进制，按 SOP 阶段精准注入（plan/doing/learning/dream 各不相同），无跨阶段污染


### .rick/jobs/job_19/doing/debug/（执行问题记录，已内嵌）

（本次 job 无 debug 记录）

### 参考资料路径（按需读取）

- **SPEC.md**: `/Users/sunquan/ai_coding/CODING/rick/.rick/SPEC.md`
- **任务详情**（task*.md）:
  （easy 模式无 task*.md 文件）
- **执行轨迹**（act-path.md）:
  （easy 模式无 act-path.md 文件）

### 任务执行结果

| Task ID | 任务名称 | 状态 | Commit Hash | 重试次数 |
|---------|---------|------|-------------|----------|
| easy_session | Easy Mode Session | success | N/A | 1 |


---

## ⚠️ 必须严格按以下 7 步 SOP 执行

### Step 1：分析执行记录（必须完成，不可跳过）

**1a. 分析 debug/**（内容已内嵌在上方，硬约束，SUMMARY.md 生成前必须完成）

分析上方".rick/jobs/job_19/doing/debug/（执行问题记录）"内容：
- 每个 debug 条目的根因与解决过程
- 未解决的问题（进展状态为"未解决"的条目）

**1b. 还原完整执行轨迹**

读取上方列出的所有 act-path.md 文件，按任务顺序还原本次 job 的完整执行轨迹：
- 每个 task 的工具调用序列
- 报错次数与修复路径
- 执行耗时与关键决策点

输出格式：逐 task 列出轨迹摘要（1-3 句）。

---

### Step 2：评估更合理的 act-path

针对每个 task，评估：
1. 是否存在冗余工具调用（可合并或省略）？
2. 是否存在可预防的错误（通过前置检查或更好的顺序）？
3. 是否有更短的执行路径能达到同样目标？

为路径最长或报错最多的 1-2 个 task 输出改进建议。

---

### Step 3：提取 Tools

**YOU MUST declare: "I will use skill:gen-skill." Before writing any skill proposal.**

技能文件：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_19/doing/prompts/skill_gen_skill.md`

从 act-path 和 debug 中识别可复用模式，**优先判断**哪些逻辑值得提取为独立 Python 工具：
- ✅ 纯函数：确定性输入输出，无副作用
- ✅ 跨 task / 跨 job 通用
- ✅ 支持 `--test` 自验证

直接写入：`/Users/sunquan/ai_coding/CODING/rick/.rick/tools/*.py`

---

### Step 4：沉淀 Skills（wiki 文档）

基于 Step 3 识别出的 tools，为每个可复用模式生成 wiki 文档：

**wiki 文档格式**：触发场景 / 预期效果 / 使用方法

- **触发场景**：何时使用（具体信号词）
- **预期效果**：可量化的结果
- **使用方法**：
  - 有对应 tool → 只写工具路径 + 调用示例，**禁止内联实现代码**
  - 无对应 tool → 可写简短伪代码说明思路

直接写入：`/Users/sunquan/ai_coding/CODING/rick/.rick/wiki/*.md`

**原则：tools 承载 how，wiki 描述 what/when/why，不重复实现。**

---

### Step 5：更新 SPEC.md

直接更新 `/Users/sunquan/ai_coding/CODING/rick/.rick/SPEC.md`（in-place，无需生成副本）。

#### 5a. 将 Step 4 所有 wiki 文档注册到技能列表

**每一个 wiki 文档都必须在 `## 技能列表` 中有对应条目**，格式：

```markdown
| 名称 | 触发词 | 路径 |
|------|--------|------|
| rick-test-isolation | plan_check 错误被自动修复 | .rick/wiki/rick_test_isolation.md |
```

#### 5b. SPEC 内容瘦身（渐进式披露）

若 SPEC.md 某节内容过长（详细步骤、示例、背景说明），将其迁移到 wiki，SPEC 只保留一行摘要 + 链接：

```markdown
## 编译与运行方法

详见 → [编译与运行指南](wiki/build_and_run.md)
```

**原则：SPEC ≤ 512 行；超出部分卸载到 wiki，SPEC 保留入口链接。**

---

### Step 6：生成 SUMMARY.md

**⚠️ 前置检查**：确认已完成 Step 1a（分析 debug.md 内容）。未完成 Step 1a 禁止生成 SUMMARY.md。

在 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_19/learning` 下生成 `SUMMARY.md`：

```markdown
APPROVED: true

# Job job_19 执行总结

## 执行概述

**项目目标**: ...
**实际完成**: ...
**整体评价**: ⭐⭐⭐⭐⭐ (1-5 星)

## 关键成就

1. **成就1**: 描述和意义

## 问题与教训

### 问题1: 问题描述

**根本原因**: ...
**解决方案**: ...
**经验教训**: ...

## 知识沉淀清单

- [ ] skills/xxx.md - 技能描述
- [ ] SPEC.md - 变更说明（如有）
```

---

### Step 7：运行 learning_check 验证 SUMMARY.md

```bash
/Users/sunquan/ai_coding/CODING/rick/bin/rick tools learning_check job_19
```

失败则修复后重新运行，直至通过。

---

## ⚠️ 重要约束

1. **debug/ 内容已内嵌，必须在 SUMMARY.md 之前分析**：Step 1a 是硬约束，不可跳过
2. **Step 3 必须声明使用 gen-skill**：`"I will use skill:gen-skill."` 是硬约束
3. **wiki/tools/SPEC 直接写入 `.rick/`**：不要写到 learning 子目录再合并，直接操作 `/Users/sunquan/ai_coding/CODING/rick/.rick/wiki`、`/Users/sunquan/ai_coding/CODING/rick/.rick/tools`、`/Users/sunquan/ai_coding/CODING/rick/.rick/SPEC.md`
4. **SUMMARY.md 写入 learning 目录**：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_19/learning/SUMMARY.md` 作为本次执行记录
