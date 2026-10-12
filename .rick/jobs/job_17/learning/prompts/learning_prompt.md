# Rick Learning 阶段

你是一个资深的技术文档专家和知识管理专家。根据本次 job 执行过程，总结知识、经验和教训，沉淀可复用 skills。

## 执行上下文

**Job**: job_17

### OKR（任务目标）

# Job OKR: 代码库死代码清理与重复逻辑消除

## 目标 (Objective)
清除 RFC-refactor-2 和 RFC-refactor-go-codebase 中记录的代码欠债：删除无用的 skill 文件、消除 frontmatter 解析重复、移除 `--easy` 模式及其所有相关代码、清理 `tools merge` 残留文档引用，使代码库更简洁、维护成本更低。（注：RFC-refactor-go-codebase §1 workspace 死代码已在之前 job 中清理完毕，本 job 不重复处理；§2 tools merge 选择删除文档引用而非实现；§3 RED verification 本 job 跳过；RFC-refactor-2 P2 TODO 2026-08 本 job 跳过，后续单独建 RFC；easy.go 保留文件本身，因已通过 --easy flag 集成，不属于死代码）

## 关键结果 (Key Results)
- KR1: `internal/prompt/templates/skills/` 中的 3 个死代码文件（`tc.md`、`tdd.md`、`tdd/testing-anti-patterns.md`）被删除，`tc.md` 内容无损合并进 `tdd-zh.md`，`go test ./internal/prompt/...` 仍通过
- KR2: `internal/parser/frontmatter.go` 提取公共 frontmatter 解析函数，`debug_dir.go` 和 `easy_prompt.go`（删除前）均改为调用它，消除重复实现
- KR3: `callClaudeCodeCLI` 支持 `extraArgs ...string`，`easy.go` 中的 `callClaudeCodeCLIEasy`/`callClaudeCodeCLIResume` 两个重复函数删除；`rick doing --easy` 功能完整保留；`go build ./...` 通过
- KR4: SPEC.md、wiki/ 中所有 `tools merge`、`easy` 模式引用被更新或删除，文档与实现一致


### .rick/jobs/job_17/doing/debug/（执行问题记录，已内嵌）

（本次 job 无 debug.md 记录）

### 参考资料路径（按需读取）

- **SPEC.md**: `/Users/sunquan/ai_coding/CODING/rick/.rick/SPEC.md`
- **任务详情**（task*.md）:
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/plan/task1.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/plan/task2.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/plan/task3.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/plan/task4.md`
- **执行轨迹**（act-path.md）:
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/doing/tasks/task1/act-path.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/doing/tasks/task2/act-path.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/doing/tasks/task3/act-path.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/doing/tasks/task4/act-path.md`

### 任务执行结果

| Task ID | 任务名称 | 状态 | Commit Hash | 重试次数 |
|---------|---------|------|-------------|----------|
| task1 | 合并 tc.md 内容到 tdd-zh.md 并删除死代码 skill 文件 | success | 31c8eac0 | 0 |
| task2 | 提取公共 frontmatter 解析函数到 internal/parser 包 | success | 540956b0 | 0 |
| task4 | 更新 SPEC 和 wiki 文档，清理 rick easy 独立命令引用和 tools merge 残留引用 | success | 7457b053 | 0 |
| task3 | 重构 easy.go 消除内部重复，复用已有 callClaudeCodeCLI | success | fd4a8649 | 0 |


---

## ⚠️ 必须严格按以下 7 步 SOP 执行

### Step 1：分析执行记录（必须完成，不可跳过）

**1a. 分析 debug/**（内容已内嵌在上方，硬约束，SUMMARY.md 生成前必须完成）

分析上方".rick/jobs/job_17/doing/debug/（执行问题记录）"内容：
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

技能文件：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/learning/prompts/skill_gen_skill.md`

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

在 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/learning` 下生成 `SUMMARY.md`：

```markdown
APPROVED: true

# Job job_17 执行总结

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
/Users/sunquan/ai_coding/CODING/rick/bin/rick tools learning_check job_17
```

失败则修复后重新运行，直至通过。

---

## ⚠️ 重要约束

1. **debug/ 内容已内嵌，必须在 SUMMARY.md 之前分析**：Step 1a 是硬约束，不可跳过
2. **Step 3 必须声明使用 gen-skill**：`"I will use skill:gen-skill."` 是硬约束
3. **wiki/tools/SPEC 直接写入 `.rick/`**：不要写到 learning 子目录再合并，直接操作 `/Users/sunquan/ai_coding/CODING/rick/.rick/wiki`、`/Users/sunquan/ai_coding/CODING/rick/.rick/tools`、`/Users/sunquan/ai_coding/CODING/rick/.rick/SPEC.md`
4. **SUMMARY.md 写入 learning 目录**：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/learning/SUMMARY.md` 作为本次执行记录
