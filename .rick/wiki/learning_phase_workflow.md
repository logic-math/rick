# Learning 阶段工作流

## 概述

Rick 的 learning 阶段是 plan→doing→learning 循环的最后一步，负责从 job 执行过程中提取可复用的知识，沉淀到项目知识库（`.rick/`）中。

## 背景

每次 job 执行后积累了新的经验：解决了什么问题、发现了什么模式、哪些工具可以复用。learning 阶段将这些隐性知识显性化，经人工审核后手动 `git merge` 到主分支。

## 产出文件

### 1. SUMMARY.md（必需，`learning_check` 验证）

执行总结报告，包含：执行概述、关键成就、问题与教训、技术总结、知识沉淀清单。

**第一行格式**：`<!-- APPROVED: false -->`（人工审核通过后改为 `<!-- APPROVED: true -->`）

### 2. wiki/*.md（按需）

系统原理文档或技能说明书。新增的 wiki 文档在审核后手动复制/合并到 `.rick/wiki/`。

### 3. tools/*.py（按需）

新增的 Python 工具脚本。审核后手动复制到 `.rick/tools/`，首行必须有 `# Description:` 注释。

### 4. OKR.md / SPEC.md（按需）

覆盖已有的 `.rick/` 对应文件，必须是完整版本（全量覆盖，非 patch）。

## 工作原理

### AI agent 执行步骤

1. **分析阶段**：读取 job OKR.md、task*.md、debug.md、act-path.md 理解执行过程
2. **生成阶段**：在 `.rick/jobs/job_N/learning/` 下生成所有产出文件
3. **展示阶段**：展示生成文件清单，等待人工审核

### 人工审核步骤

1. 审核 SUMMARY.md（执行质量、教训）
2. 审核 OKR.md 和 SPEC.md（目标和规范更新是否合理）
3. 审核 wiki/ 和 tools/（知识提取是否准确）
4. 将 SUMMARY.md 第一行改为 `<!-- APPROVED: true -->`
5. 手动将各产出文件合并到 `.rick/`（`rick tools merge` 尚未实现，见 RFC-005）
6. `git add .rick/ && git commit -m "learning: merge job_N knowledge"`

## 如何控制

### 触发 learning

```bash
rick learning job_1
```

### 验证 learning 产出

```bash
# 检查 SUMMARY.md 格式
rick tools learning_check job_1

# 查看生成的文件
ls .rick/jobs/job_1/learning/
```

### 手动合并产出

```bash
# 1. 审核并批准（修改 SUMMARY.md 第一行为 <!-- APPROVED: true -->）
# 2. 复制产出到 .rick/
cp -r .rick/jobs/job_1/learning/wiki/* .rick/wiki/
cp -r .rick/jobs/job_1/learning/tools/* .rick/tools/
# 3. 提交
git add .rick/ && git commit -m "learning: merge job_1 knowledge"
```

## 注意事项

- SUMMARY.md 是唯一 `learning_check` 强制校验的文件（含 `# Job` heading）
- `<!-- APPROVED: false -->` 是安全门，防止未审核知识被合并
- OKR.md 和 SPEC.md 必须是完整版本，不是 patch
- wiki 文档面向人类，避免过于技术化的细节

## 相关资源

- 相关 Wiki: [rick_tools_commands.md](rick_tools_commands.md)
- 相关 Tool: [`.rick/tools/mock_agent_testing.py`](.rick/tools/mock_agent_testing.py)
- 源码: `internal/cmd/learning.go`, `internal/prompt/templates/learning.md`
