# Learning 阶段工作流

## 概述

Rick 的 learning 阶段是 plan→doing→learning 循环的最后一步，负责从 job 执行过程中提取可复用的知识，并将其沉淀到项目的知识库（`.rick/`）中。

## 背景

每次 job 执行后，AI agent 和人类都积累了新的经验：解决了什么问题、发现了什么模式、哪些代码可以复用。learning 阶段的目标是将这些隐性知识显性化，并通过 `rick tools merge` 合并到项目知识库，让后续 job 能够复用。

## 四类产出

### 1. SUMMARY.md（必需）

执行总结报告，包含：
- 执行概述（目标、完成情况、评分）
- 关键成就列表
- 问题与教训（根本原因 + 解决方案 + 经验教训）
- 技术总结（关键决策、新技术、代码质量）
- 建议和改进
- 知识沉淀清单

**第一行格式**：`<!-- APPROVED: false -->`（人工审核通过后改为 `<!-- APPROVED: true -->`）

### 2. skills/*.py（按需）

可复用的 Python 技能脚本，遵循统一标准：
- 使用 argparse 解析参数
- 输出 JSON 格式结果：`{"pass": bool, "errors": [...]}`
- 文件名使用小写字母和下划线

适合提取为 skill 的场景：
- 可在多个 job 中复用的校验逻辑
- 数据处理和转换工具
- 测试辅助脚本

### 3. OKR.md（按需）

完整的 OKR 文件（不是 diff，而是完整新版本）：
- 顶部注释说明本次更新内容
- 格式：`## O1: 目标` + `### 关键结果 (Key Results)` + `- KR1.1: 可衡量结果`
- 包含所有目标（新增 + 保留 + 修改）

### 4. SPEC.md（按需）

完整的 SPEC 文件（不是 diff，而是完整新版本）：
- 顶部注释说明本次更新内容
- 四个必需章节：技术栈、架构设计、开发规范、工程实践
- 每个类别使用 `- 项目名: 内容` 格式

### 5. wiki/*.md（按需）

新概念、新技术或重要模式的 wiki 文档：
- 面向人类读者，重点讲解"是什么"和"如何控制"
- 包含 Mermaid 图表（系统架构、流程图）
- 章节：概述 / 背景 / 工作原理 / 如何控制 / 注意事项

## 工作原理

### AI agent 执行步骤

1. **分析阶段**：`git show <commit_hash>` 查看每个任务的代码变更，阅读 debug.md
2. **生成阶段**：在 `.rick/jobs/job_N/learning/` 下生成所有文档
3. **展示阶段**：展示生成的文件清单，等待人工审核

### 人工审核步骤

1. 审核 SUMMARY.md，确认执行质量和教训
2. 审核 OKR.md 和 SPEC.md，确认目标和规范更新合理
3. 审核 wiki/ 和 skills/，确认知识提取准确
4. 将 SUMMARY.md 第一行改为 `<!-- APPROVED: true -->`
5. 执行 `rick tools merge <job_id>`
6. 审核 merge 产生的 diff
7. 执行 `git merge --no-ff learning/job_N` 完成合并

## 如何控制

### 验证 learning 产出

```bash
# 检查 learning 目录结构
rick tools learning_check job_1

# 查看生成的文件
ls .rick/jobs/job_1/learning/
ls .rick/jobs/job_1/learning/wiki/
ls .rick/jobs/job_1/learning/skills/
```

### 执行 merge

```bash
# 1. 人工审核并批准（修改 SUMMARY.md 第一行）
# 2. 执行 merge
rick tools merge job_1

# 3. 查看 diff
git diff main learning/job_1

# 4. 合并
git merge --no-ff learning/job_1
git branch -D learning/job_1
```

## 注意事项

- SUMMARY.md 是唯一必需文件，其他按需生成
- OKR.md 和 SPEC.md 必须是完整版本（包含所有内容），不是 patch
- `<!-- APPROVED: false -->` 是安全门，防止未审核的知识被合并
- skills/*.py 必须通过 Python 语法检查（`learning_check` 会验证）
- wiki 文档面向人类，避免过于技术化的细节

## 相关资源

- 相关 Wiki: [rick_tools_commands.md](rick_tools_commands.md)
- 相关 Skill: [mock_agent_testing.py](../skills/mock_agent_testing.py)
- 源码: `internal/cmd/learning.go`, `internal/prompt/templates/learning.md`
