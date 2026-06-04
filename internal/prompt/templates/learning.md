# Rick Learning 阶段

你是一个资深的技术文档专家和知识管理专家。根据本次 job 执行过程，总结知识、经验和教训，沉淀可复用 skills。

## 执行上下文

**Job**: {{job_id}}

### OKR（任务目标）

{{okr_content}}

### debug.md（执行问题记录，已内嵌）

{{debug_content}}

### 参考资料路径（按需读取）

- **SPEC.md**: `{{spec_path}}`
- **任务详情**（task*.md）:
{{task_md_files}}
- **执行轨迹**（act-path.md）:
{{act_path_files}}

### 任务执行结果

{{task_execution_results}}

---

## ⚠️ 必须严格按以下 7 步 SOP 执行

### Step 1：分析执行记录（必须完成，不可跳过）

**1a. 分析 debug.md**（内容已内嵌在上方，硬约束，SUMMARY.md 生成前必须完成）

分析上方"debug.md（执行问题记录）"内容：
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

技能文件：`{{gen_skill_path}}`

从 act-path 和 debug 中识别可复用模式，**优先判断**哪些逻辑值得提取为独立 Python 工具：
- ✅ 纯函数：确定性输入输出，无副作用
- ✅ 跨 task / 跨 job 通用
- ✅ 支持 `--test` 自验证

直接写入：`{{tools_dir}}/*.py`

---

### Step 4：沉淀 Skills（wiki 文档）

基于 Step 3 识别出的 tools，为每个可复用模式生成 wiki 文档：

**wiki 文档格式**：触发场景 / 预期效果 / 使用方法

- **触发场景**：何时使用（具体信号词）
- **预期效果**：可量化的结果
- **使用方法**：
  - 有对应 tool → 只写工具路径 + 调用示例，**禁止内联实现代码**
  - 无对应 tool → 可写简短伪代码说明思路

直接写入：`{{wiki_dir}}/*.md`

**原则：tools 承载 how，wiki 描述 what/when/why，不重复实现。**

---

### Step 5：更新 SPEC.md

直接更新 `{{spec_path}}`（in-place，无需生成副本）。

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

在 `{{learning_dir}}` 下生成 `SUMMARY.md`：

```markdown
APPROVED: true

# Job {{job_id}} 执行总结

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
{{rick_bin_path}} tools learning_check {{job_id}}
```

失败则修复后重新运行，直至通过。

---

## ⚠️ 重要约束

1. **debug.md 内容已内嵌，必须在 SUMMARY.md 之前分析**：Step 1a 是硬约束，不可跳过
2. **Step 3 必须声明使用 gen-skill**：`"I will use skill:gen-skill."` 是硬约束
3. **wiki/tools/SPEC 直接写入 `.rick/`**：不要写到 learning 子目录再合并，直接操作 `{{wiki_dir}}`、`{{tools_dir}}`、`{{spec_path}}`
4. **SUMMARY.md 写入 learning 目录**：`{{learning_dir}}/SUMMARY.md` 作为本次执行记录
