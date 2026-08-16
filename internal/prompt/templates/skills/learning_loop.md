---
name: learning-loop
description: Learning 阶段 Loop；每个 Step 由 parent 用 runs.run 派发独立 worker child，Step 3-4 产出需人类反复审核，直至沉淀完成
---

## 全局目标

分析本次 job 的执行记录，沉淀可复用的 skills 和 loops，生成 SUMMARY.md 总结报告。

**成功标准**（全部满足时退出）：
- `{{rick_bin_path}} tools learning_check {{job_id}}` pass
- `{{learning_dir}}/SUMMARY.md` 已写入，首行 `APPROVED: true`
- 人类已审核并确认 skills 和 loops 产出

---

## 上下文管理

**输入**（从 learning prompt 上下文获取）：
- debug/ 记录（已内嵌在 prompt）
- runtime-trace.md 文件列表
- task*.md 文件列表
- 任务执行结果表

**输出**：
- `{{skills_dir}}/{name}_skill/skill.md` — 新 skill 目录
- `{{loops_dir}}/{name}-loop.md` — 新 loop 文件
- `{{learning_dir}}/SUMMARY.md` — 执行总结

---

## 工作流

**每个 Step 由 parent 用 `runs.run` 派发一个独立 worker child（`agent:'worker'`）执行。Step 3-4 完成后需人类审核，审核未通过则重新迭代。单写者：同一时间只有一个 child 写 skills/loops/domain。**

触发语法（串行派发 worker child；默认 `async: true`）：
```text
subagent({ workflowScript: "const s1 = await runs.run('learning-step1', { agent: 'worker', task: '分析执行记录' }); const s2 = await runs.run('learning-step2', { agent: 'worker', task: '评估 runtime-trace' }); return { step1: s1.output, step2: s2.output }" })
```

```
[parent 编排者]
   │
   ├─ runs.run 派发 Step 1 worker child → 分析执行记录 → parent 验收
   │
   ├─ runs.run 派发 Step 2 worker child → 评估 runtime-trace → parent 验收
   │
   ├─ runs.run 派发 Step 3 worker child → 提取 Skill → parent 验收
   │
   ├─ runs.run 派发 Step 4 worker child → 提取 Loop → parent 验收
   │
   ├─ [人类审核 Skills & Loops] ←──────────────────────────┐
   │     │                                                  │
   │     ├─ 审核通过 → 继续 Step 5                          │
   │     │                                                  │
   │     └─ 需修改 → 重启 Step 3/4 ────────────────────────────┘
   │
   ├─ runs.run 派发 Step 5 worker child → 整理 Domain 事实 → parent 验收
   │
   ├─ runs.run 派发 Step 6 worker child → 生成 SUMMARY.md → parent 验收
   │
   └─ runs.run 派发 Step 7 worker child → 运行 learning_check → [DONE] ✅
```

---

### Step 1 worker child：分析执行记录（硬约束，必须完成）

**1a. 分析 debug/**

分析 debug/ 目录内容：
- 每个 debug 条目的根因与解决过程
- 未解决的问题（进展状态为"未解决"的条目）

**1b. 还原完整执行轨迹**

读取所有 runtime-trace.md 文件，逐 task 列出轨迹摘要（1-3 句）：
- 每个 task 的工具调用序列
- 报错次数与修复路径
- 执行耗时与关键决策点

**parent 验收**：child 输出包含每个 task 的轨迹摘要，且 debug 分析结论明确。

---

### Step 2 worker child：评估更合理的 runtime-trace

针对每个 task 评估：
1. 是否存在冗余工具调用（可合并或省略）？
2. 是否存在可预防的错误（通过前置检查或更好的顺序）？
3. 是否有更短的执行路径能达到同样目标？

为路径最长或报错最多的 1-2 个 task 输出改进建议。

**parent 验收**：child 输出包含具体改进建议。

---

### Step 3 worker child：提取可复用 Skill

声明：`"I will use skill:gen-skill."`

读取 `{{gen_skill_path}}`，按其定义的格式（触发场景 / 预期效果 / 核心内容）从 runtime-trace 和 debug 中提取可复用技能：

- 每个 skill 创建目录 `{{skills_dir}}/{name}_skill/`
- 主文件写入 `{{skills_dir}}/{name}_skill/skill.md`
- 如有辅助脚本，写入同目录的 `.py` 文件

**parent 验收**：`ls {{skills_dir}}/` 确认目录已创建（如无可提取的 skill 则跳过）。

---

### Step 4 worker child：提取 Loop 模式

声明：`"I will use skill:gen-loop."`

读取 `{{gen_loop_path}}`，从 runtime-trace 和 debug 中识别 job 内反复出现的循环模式，按其定义的完整格式写入 `{{loops_dir}}/{name}-loop.md`：

- ✅ 有明确触发条件（trigger）
- ✅ 包含依赖准备（软件版本、工具、环境安装）
- ✅ 每个 Step 引用对应的 `.rick/{name}_skill/skill.md`
- ✅ 有具体的产出评估 skill

**parent 验收**：`ls {{loops_dir}}/*.md` 确认文件已写入（如无可识别的 loop 则跳过）。

---

### 人类审核 Skills & Loops

**parent 向人类展示产出摘要后，等待人类反馈：**

```
已生成：
- Skills: ls {{skills_dir}}/
- Loops:  ls {{loops_dir}}/

请审核上述产出，确认后输入 "approved" 继续，
或指出需要修改的内容，parent 将重新启动对应 Step。
```

**迭代规则**：
- 人类要求修改 skill → 重启 Step 3 worker child（携带修改意见）
- 人类要求修改 loop → 重启 Step 4 worker child（携带修改意见）
- 人类输入 "approved" → 进入 Step 5

---

### Step 5 worker child：整理 Domain 事实知识

声明：`"I will use skill:gen-domain."`

读取 `{{gen_domain_path}}`，从本次 job 的执行记录中提取事实性知识，写入 `{{domain_dir}}/`：

- 已知问题与**精确解决命令** → `{{domain_dir}}/bugs.md`（追加，不重复）
- 环境配置、版本事实 → `{{domain_dir}}/env.md`（追加）
- 构建/测试命令事实 → `{{domain_dir}}/build.md`（追加）
- 其他主题事实 → `{{domain_dir}}/{topic}.md`

**parent 验收**：`ls {{domain_dir}}/` 确认有更新（如本次 job 无新事实则跳过，说明原因）。

---

### Step 6 worker child：生成 SUMMARY.md

⚠️ **前置检查**：Step 1a 必须已完成，且人类已审核通过 Skills & Loops。

写入 `{{learning_dir}}/SUMMARY.md`：

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

- [ ] skills/{name}_skill/skill.md - 技能描述
- [ ] loops/{name}-loop.md - loop 描述（如有）
```

**parent 验收**：`{{learning_dir}}/SUMMARY.md` 存在，首行为 `APPROVED: true`。

---

### Step 7 worker child：运行 learning_check

```bash
{{rick_bin_path}} tools learning_check {{job_id}}
```

失败则修复后重新运行，直至通过。

**parent 验收**：命令输出 `✅ learning check passed`。

---

## 产出评估

| 检查项 | 判断方法 |
|--------|----------|
| learning_check pass | 命令输出 ✅ |
| SUMMARY.md 存在 | `ls {{learning_dir}}/SUMMARY.md` |
| APPROVED: true | SUMMARY.md 首行包含 `APPROVED: true` |
| 人类审核通过 | 人类输入 "approved" 确认 |

---

## 停止标准

**完成退出**：Step 7 worker child 的 learning_check pass，且人类已审核确认。

**优雅退出**：人类明确要求停止。

---

## ⚠️ 约束

1. **Step 1a 是硬约束**：未完成 Step 1a 禁止启动 Step 6 worker child（SUMMARY）
2. **人类审核是必经环节**：Step 3-4 产出未经人类确认，不得进入 Step 5（Domain）
3. **skill 目录结构**：`{{skills_dir}}/{name}_skill/skill.md`
4. **loop 文件**：`{{loops_dir}}/{name}-loop.md`
5. **SUMMARY.md 写入 learning 目录**：`{{learning_dir}}/SUMMARY.md`
