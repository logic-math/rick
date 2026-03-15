# Learning 命令超时和中文化修复

## 修复日期
2026-03-14

## 问题描述

用户反馈两个问题:
1. ❌ **超时限制**: Learning 命令 10 分钟后自动超时退出
2. ❌ **英文提示词**: 提示词全是英文,不符合中文使用习惯

```
Error: Claude analysis failed: Claude Code CLI timeout after 10m0s
```

---

## 修复内容

### 1. ✅ 移除超时限制

**修改文件**: `internal/cmd/learning.go`

**原代码** (有超时):
```go
// Set timeout
done := make(chan error, 1)
go func() {
    done <- cmd.Run()
}()

timeout := 600 * time.Second // 10 minutes timeout
select {
case err := <-done:
    if err != nil {
        return fmt.Errorf("Claude Code CLI execution failed: %w", err)
    }
    return nil
case <-time.After(timeout):
    cmd.Process.Kill()
    return fmt.Errorf("Claude Code CLI timeout after %v", timeout)
}
```

**新代码** (无超时):
```go
// Run without timeout
if err := cmd.Run(); err != nil {
    return fmt.Errorf("Claude Code CLI 执行失败: %w", err)
}

return nil
```

**改进**:
- ✅ 移除 10 分钟超时限制
- ✅ 移除 goroutine + channel 的复杂超时逻辑
- ✅ 直接调用 `cmd.Run()`,让 Claude 运行到完成
- ✅ 简化代码,提高可读性

---

### 2. ✅ 提示词中文化

**修改文件**: `internal/cmd/learning.go`

#### 2.1 用户界面提示中文化

**原代码**:
```go
fmt.Printf("\n📝 Prompt saved to: %s\n", tmpFile.Name())
fmt.Println("🤖 Starting Claude Code CLI in interactive mode...")
fmt.Println("📌 Claude will analyze the execution and update documentation automatically.")
```

**新代码**:
```go
fmt.Printf("\n📝 提示词已保存到: %s\n", tmpFile.Name())
fmt.Println("🤖 启动 Claude Code CLI 交互模式...")
fmt.Println("📌 Claude 将自动分析执行结果并更新文档")
```

#### 2.2 Learning Prompt 中文化

**原 Prompt 结构** (英文):
```markdown
# Learning Analysis Task

Analyze the execution results for Job job_0 and provide insights.

## Debug Information
...

## Task Metadata
| Task ID | Task Name | Status | Task File | Commit Hash | Attempts |
...

## Instructions

Based on the debug information and task metadata above, please:

1. **Analyze** what went wrong (if anything) and what was learned
2. **Review** the commit changes by using `git show <commit_hash>` for each task
3. **Identify** key insights, patterns, and improvements
4. **Update** the following files in `.rick/` directory:
   - `OKR.md` - Update project objectives based on learnings
   - `SPEC.md` - Update development specifications if needed
   - `wiki/<topic>.md` - Create or update wiki pages for new concepts
   - `skills/<skill>.md` - Extract reusable skills for future tasks

5. **Commit** your changes with a descriptive message

Please provide a comprehensive analysis and update the documentation automatically.
You have full access to git, file system, and all necessary tools.
```

**新 Prompt 结构** (中文):
```markdown
# Learning 分析任务

分析 Job job_0 的执行结果并提取经验教训。

## 调试信息
...

## 任务元信息
| Task ID | 任务名称 | 状态 | 任务文件 | Commit Hash | 重试次数 |
...

## 执行指令

基于上述调试信息和任务元信息，请执行以下操作：

1. **分析执行过程**
   - 使用 `git show <commit_hash>` 查看每个任务的代码变更
   - 分析遇到的问题和解决方法（如果有）
   - 识别关键洞察、模式和改进点

2. **更新项目文档**（在 `.rick/` 目录下）
   - `OKR.md` - 根据学到的经验更新项目目标
   - `SPEC.md` - 如需要，更新开发规范
   - `wiki/<主题>.md` - 为新概念创建或更新 wiki 页面
   - `skills/<技能>.md` - 提取可复用的技能供未来任务使用

3. **提交变更**
   - 使用清晰的 commit message 提交你的文档更新
   - Commit message 格式: `docs(learning): <简短描述>`

**注意事项**：
- 你拥有完整的 git、文件系统和所有工具的访问权限
- 请提供全面的分析并自动更新文档
- 重点关注可复用的经验和模式
- 确保文档更新后的一致性和完整性
```

**改进点**:
- ✅ 所有标题和说明都改为中文
- ✅ 表格列名中文化 (任务名称、状态、任务文件、重试次数)
- ✅ 指令更详细、更清晰
- ✅ 添加 commit message 格式规范
- ✅ 添加注意事项部分
- ✅ 更符合中文表达习惯

---

## 代码改动汇总

### 修改文件

| 文件 | 改动内容 | 行数变化 |
|------|---------|---------|
| `internal/cmd/learning.go` | 移除超时 + 中文化 | ~30 行 |

### 移除的代码

- ❌ `timeout := 600 * time.Second` - 超时设置
- ❌ `done := make(chan error, 1)` - 超时 channel
- ❌ `select { case <-time.After(timeout): ... }` - 超时逻辑
- ❌ `import "time"` - 未使用的 import

### 新增的改进

- ✅ 直接调用 `cmd.Run()` - 无超时限制
- ✅ 中文用户界面提示
- ✅ 完整的中文 Learning Prompt
- ✅ 更详细的执行指令
- ✅ Commit message 格式规范

---

## 使用效果对比

### 修复前

```bash
rick learning job_0

=== Learning Workflow ===

=== Step 1: Collecting execution data ===
✅ Read debug.md (123 bytes)
✅ Loaded tasks.json (3 tasks)

=== Step 2: Analyzing with Claude ===
Calling Claude Code CLI for analysis...

📝 Prompt saved to: /tmp/rick-learning-abc123.md
🤖 Starting Claude Code CLI in interactive mode...
📌 Claude will analyze the execution and update documentation automatically.

[Claude 运行中...]
[10 分钟后]
Error: Claude analysis failed: Claude Code CLI timeout after 10m0s
```

### 修复后

```bash
rick learning job_0

=== Learning Workflow ===

=== Step 1: Collecting execution data ===
✅ Read debug.md (123 bytes)
✅ Loaded tasks.json (3 tasks)

=== Step 2: Analyzing with Claude ===
Calling Claude Code CLI for analysis...

📝 提示词已保存到: /tmp/rick-learning-abc123.md
🤖 启动 Claude Code CLI 交互模式...
📌 Claude 将自动分析执行结果并更新文档

[Claude 运行中... 无时间限制]
[Claude 自动完成所有操作]
✅ Learning workflow completed!
```

---

## 测试验证

### 测试 1: 无超时限制

```bash
# 启动 learning
rick learning job_0

# 预期: Claude 可以运行任意长时间
# - 不会在 10 分钟后被强制终止
# - 可以完成复杂的分析和文档更新
# - 直到 Claude 自然完成任务
```

### 测试 2: 中文提示

```bash
# 查看生成的 prompt 文件
cat /tmp/rick-learning-*.md

# 预期输出:
# # Learning 分析任务
# 
# 分析 Job job_0 的执行结果并提取经验教训。
# 
# ## 调试信息
# ...
# 
# ## 任务元信息
# | Task ID | 任务名称 | 状态 | 任务文件 | Commit Hash | 重试次数 |
# ...
```

### 测试 3: 编译和安装

```bash
./scripts/build.sh
./scripts/install.sh --source

# 预期:
# ✅ 编译成功
# ✅ 安装成功
# ✅ 版本验证通过
```

---

## 技术细节

### 为什么移除超时?

1. **Learning 是复杂任务**
   - 需要查看多个 git commit
   - 需要分析代码变更
   - 需要更新多个文档文件
   - 需要 commit 变更
   - 10 分钟可能不够

2. **交互式模式的特点**
   - Claude 会自动完成任务
   - 用户可以随时 Ctrl+C 中断
   - 不需要程序强制超时

3. **简化代码**
   - 移除 goroutine + channel
   - 移除 select + timeout 逻辑
   - 代码更简洁、更易维护

### 为什么使用中文提示词?

1. **用户体验**
   - 中文用户更容易理解
   - 减少理解障碍
   - 提高使用效率

2. **Claude 支持**
   - Claude 完全支持中文
   - 中文指令同样清晰准确
   - 不影响执行质量

3. **本地化**
   - Rick 是中文项目
   - 文档是中文
   - 提示词也应该中文

---

## 常见问题

### Q1: 如果 Claude 运行太久怎么办?

**A**: 用户可以随时按 `Ctrl+C` 中断执行:
```bash
rick learning job_0
[Claude 运行中...]
^C  # 按 Ctrl+C 中断
```

### Q2: 会不会出现死循环?

**A**: 不会。Claude Code CLI 会自然完成任务并退出。如果真的遇到问题:
1. 用户可以 Ctrl+C 中断
2. 检查 Claude 的输出日志
3. 手动查看生成的 prompt 文件
4. 必要时手动运行: `claude /tmp/rick-learning-*.md`

### Q3: 中文提示词会影响 Claude 的理解吗?

**A**: 不会。Claude 对中文和英文的理解能力相同:
- ✅ 完全理解中文指令
- ✅ 可以查看英文代码
- ✅ 可以生成中英文混合文档
- ✅ 执行质量不受影响

### Q4: 如何查看生成的提示词?

**A**: 提示词保存在临时文件中:
```bash
# Learning 命令会打印文件路径
📝 提示词已保存到: /tmp/rick-learning-abc123.md

# 查看内容
cat /tmp/rick-learning-abc123.md

# 或者手动运行
claude /tmp/rick-learning-abc123.md
```

---

## 总结

✅ **移除超时限制**: Claude 可以运行任意长时间,直到自然完成

✅ **中文化提示词**: 所有提示都改为中文,更符合使用习惯

✅ **简化代码**: 移除复杂的超时逻辑,代码更简洁

✅ **保持功能**: 所有 Learning 功能完全保留,只是改进用户体验

✅ **向后兼容**: 不影响现有的 doing 和 plan 命令

---

## 对比总结

| 项目 | 修复前 | 修复后 |
|------|--------|--------|
| 超时限制 | 10 分钟强制终止 | 无限制,自然完成 |
| 用户提示 | 英文 | 中文 |
| Prompt 语言 | 英文 | 中文 |
| 代码复杂度 | 高 (goroutine + channel) | 低 (直接调用) |
| 用户体验 | ❌ 容易超时 | ✅ 流畅完成 |
| 本地化 | ❌ 英文为主 | ✅ 完全中文 |

---

**修复完成日期**: 2026-03-14  
**修复人**: Claude Opus 4.6  
**版本**: Rick CLI 0.1.0
