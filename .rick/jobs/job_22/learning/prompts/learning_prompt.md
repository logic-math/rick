# Rick Learning 阶段

你是一个资深的技术文档专家和知识管理专家。根据本次 job 执行过程，总结知识、经验和教训，沉淀可复用 loops 和 skills。

## 执行上下文

**Job**: job_22

### OKR（任务目标）

# Job OKR: 实现 RFC-001 上下文架构重设计

## 目标 (Objective)

将 rick 的上下文架构从 `SPEC.md → wiki → tools` 三层迁移到 `loops → skills` 两层，使项目级 loop 和 skill 由 learning 阶段动态产出，agent 通过 loops_context 获取执行时可用的结构化工作流。

## 关键结果 (Key Results)

- KR1: `.rick/loops/` 和 `.rick/skills/` 目录建立，loop.md 三要素格式规范明确（frontmatter: name/trigger/scope）
- KR2: `debug_skill.md` 替换为 diagnosing-bugs Phase 1-6，更精炼的调试抽象落地
- KR3: `LoadLoopsContext()` 函数实现并通过单元测试，遍历 `.rick/loops/*.md` 正确提取 trigger 字段
- KR4: doing/plan/learning/easy/dream 五个 prompt builder 完成迁移：移除 SPEC/OKR/wiki/tools 注入，添加 loops_context 注入
- KR5: 所有模板文件同步更新，`rick tools plan_check job_22` 通过
- KR6: `loop_protocol.md` 通过 embed.FS 内嵌，单一维护；doing/easy 的 dry-run 输出包含真实路径（非字面量 `{{loop_protocol_path}}`），Loop 执行协议正文只存在于 `loop_protocol.md` 一处


### .rick/jobs/job_22/doing/debug/（执行问题记录，已内嵌）

（本次 job 无 debug.md 记录）

### 参考资料路径（按需读取）

- **任务详情**（task*.md）:
  （无 task*.md 文件）
- **执行轨迹**（act-path.md）:
  （无 act-path.md 文件）

### 可用的项目 Loops

## 可用的项目 Loops

- **candidate-loop-1**：when implementing new features
- **go-tdd-loop**："当需要对 Go 代码进行 TDD 迭代直到测试通过时触发"


### 任务执行结果

无任务元信息


---

## ⚠️ 必须严格按以下 6 步 SOP 执行

### Step 1：分析执行记录（必须完成，不可跳过）

**1a. 分析 debug/**（内容已内嵌在上方，硬约束，SUMMARY.md 生成前必须完成）

分析上方".rick/jobs/job_22/doing/debug/（执行问题记录）"内容：
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

### Step 3：提取可复用 Skill（candidate_skill）

从 act-path 和 debug 中识别可复用模式，提取为独立 skill 文件：
- ✅ 跨 task / 跨 job 通用的方法论或流程
- ✅ 有明确触发场景和预期效果
- ✅ 人类可读、agent 可执行的操作手册

直接写入：`/Users/sunquan/ai_coding/CODING/rick/.rick/skills/candidate_skill_{n}.md`

**格式**：
```markdown
---
name: skill 名称
trigger: 触发词或触发场景
scope: 适用范围（如 doing/learning/all）
---

# 使用方法

...
```

---

### Step 4：识别 Loop 模式（candidate_loop）

基于本次 job 识别出的工作流模式，若发现可固化为 loop 的流程，写候选文件：
- ✅ 有明确的触发条件（trigger）
- ✅ 有可重复执行的步骤
- ✅ 解决了特定类型的重复性问题

直接写入：`/Users/sunquan/ai_coding/CODING/rick/.rick/loops/candidate_loop_{n}.md`

**格式**：
```markdown
---
name: loop 名称
trigger: 触发条件
scope: 适用范围
---

# Loop 步骤

...
```

---

### Step 5：生成 SUMMARY.md

**⚠️ 前置检查**：确认已完成 Step 1a（分析 debug.md 内容）。未完成 Step 1a 禁止生成 SUMMARY.md。

在 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/learning` 下生成 `SUMMARY.md`：

```markdown
APPROVED: true

# Job job_22 执行总结

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

- [ ] skills/candidate_skill_1.md - 技能描述
- [ ] loops/candidate_loop_1.md - loop 描述（如有）
```

---

### Step 6：运行 learning_check 验证 SUMMARY.md

```bash
/Users/sunquan/ai_coding/CODING/rick/bin/rick tools learning_check job_22
```

失败则修复后重新运行，直至通过。

---

## ⚠️ 重要约束

1. **debug/ 内容已内嵌，必须在 SUMMARY.md 之前分析**：Step 1a 是硬约束，不可跳过
2. **candidate 文件写入对应目录**：skill 写 `/Users/sunquan/ai_coding/CODING/rick/.rick/skills`，loop 写 `/Users/sunquan/ai_coding/CODING/rick/.rick/loops`，不写到 learning 子目录
3. **SUMMARY.md 写入 learning 目录**：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/learning/SUMMARY.md` 作为本次执行记录
