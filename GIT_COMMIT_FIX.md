# Git Commit 失败问题修复

## 修复日期
2026-03-14

## 问题描述

执行 `rick doing` 命令后，任务执行成功，但在 commit 阶段失败：

```
[WARN] Failed to commit results: failed to commit changes: failed to commit: exit status 1
Job job_0 execution completed!
```

## 根本原因

1. **Git 用户未配置**：新初始化的 Git 仓库没有配置 `user.name` 和 `user.email`，导致 `git commit` 失败
2. **文件未 staged**：`CommitJob()` 直接调用 `git commit`，但没有先 `git add` 文件

## 修复方案

### 1. 添加 Git 用户配置函数

在 `internal/cmd/doing.go` 中新增 `ensureGitUserConfigured()` 函数：

```go
// ensureGitUserConfigured ensures Git user is configured for the repository
func ensureGitUserConfigured(projectRoot string) error {
	// Check if user.name is configured
	cmd := exec.Command("git", "config", "user.name")
	cmd.Dir = projectRoot
	if output, err := cmd.Output(); err != nil || strings.TrimSpace(string(output)) == "" {
		// Set default user.name
		cmd = exec.Command("git", "config", "user.name", "Rick CLI")
		cmd.Dir = projectRoot
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set git user.name: %w", err)
		}
		if GetVerbose() {
			fmt.Println("[INFO] Set git user.name to 'Rick CLI'")
		}
	}

	// Check if user.email is configured
	cmd = exec.Command("git", "config", "user.email")
	cmd.Dir = projectRoot
	if output, err := cmd.Output(); err != nil || strings.TrimSpace(string(output)) == "" {
		// Set default user.email
		cmd = exec.Command("git", "config", "user.email", "rick@localhost")
		cmd.Dir = projectRoot
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set git user.email: %w", err)
		}
		if GetVerbose() {
			fmt.Println("[INFO] Set git user.email to 'rick@localhost'")
		}
	}

	return nil
}
```

**功能**：
- 检查 `user.name` 是否配置，未配置则设置为 "Rick CLI"
- 检查 `user.email` 是否配置，未配置则设置为 "rick@localhost"
- 只在本地仓库配置，不影响全局配置

### 2. 在 Git 初始化时调用配置函数

修改 `ensureGitInitialized()` 函数：

```go
// Initialize Git repository in project root
if GetVerbose() {
	fmt.Printf("[INFO] Initializing Git repository in project root: %s\n", projectRoot)
}

cmd := exec.Command("git", "init")
cmd.Dir = projectRoot
if output, err := cmd.CombinedOutput(); err != nil {
	return fmt.Errorf("failed to run git init: %w\nOutput: %s", err, string(output))
}

// Configure Git user if not already configured
if err := ensureGitUserConfigured(projectRoot); err != nil {
	return fmt.Errorf("failed to configure git user: %w", err)
}
```

### 3. 修改 commit 逻辑使用 AutoAddAndCommit

修改 `commitDoingResults()` 函数：

**之前（错误）**：
```go
// Commit using auto committer
if err := ac.CommitJob(jobID, commitMsg); err != nil {
	return fmt.Errorf("failed to commit changes: %w", err)
}
```

**现在（正确）**：
```go
// Check if there are any changes to commit
hasChanges, err := ac.HasChanges()
if err != nil {
	return fmt.Errorf("failed to check for changes: %w", err)
}

if !hasChanges {
	if GetVerbose() {
		fmt.Println("[INFO] No changes to commit")
	}
	return nil
}

// Add all files before committing
if err := ac.AutoAddAndCommitJob(jobID, commitMsg); err != nil {
	return fmt.Errorf("failed to commit changes: %w", err)
}
```

**改进**：
1. 先检查是否有变更（避免空 commit）
2. 使用 `AutoAddAndCommitJob()` 自动 add 所有文件
3. 包含 modified、untracked 文件

## 完整流程

### Git 初始化流程
```
1. 检查 .git 目录是否存在
2. 如果不存在：
   a. 运行 git init
   b. 配置 user.name = "Rick CLI"
   c. 配置 user.email = "rick@localhost"
   d. 创建 .gitignore
3. 如果存在：跳过初始化
```

### Commit 流程
```
1. 检查是否有变更 (git status --porcelain)
2. 如果没有变更：跳过 commit
3. 如果有变更：
   a. 获取所有 modified 文件
   b. 获取所有 untracked 文件
   c. git add <所有文件>
   d. git commit -m "<commit message>"
```

## 测试验证

### 测试场景 1：新项目
```bash
cd /tmp/test_rick_new
rick plan "创建测试文件"
rick doing job_0
```

**预期结果**：
- ✅ Git 仓库自动初始化
- ✅ Git 用户自动配置
- ✅ 任务执行成功
- ✅ 变更自动提交

### 测试场景 2：已有 Git 仓库
```bash
cd /tmp/test_rick_existing
git init
git config user.name "Test User"
git config user.email "test@example.com"
rick plan "创建测试文件"
rick doing job_0
```

**预期结果**：
- ✅ 使用现有 Git 配置
- ✅ 不覆盖已有配置
- ✅ 任务执行成功
- ✅ 变更自动提交

### 测试场景 3：无变更
```bash
cd /tmp/test_rick_no_changes
rick plan "不产生任何文件的任务"
rick doing job_0
```

**预期结果**：
- ✅ 检测到无变更
- ✅ 跳过 commit
- ✅ 不报错

## 代码改动总结

| 文件 | 改动 | 行数 |
|------|------|------|
| `internal/cmd/doing.go` | 新增 `ensureGitUserConfigured()` | +34 |
| `internal/cmd/doing.go` | 修改 `ensureGitInitialized()` | +3 |
| `internal/cmd/doing.go` | 修改 `commitDoingResults()` | +12 |

## 相关文件

- `internal/git/git.go` - Git 基础操作
- `internal/git/commit.go` - Commit 相关操作
- `internal/cmd/doing.go` - Doing 命令实现

## 关键技术点

### 1. Git 配置检查
```bash
# 检查配置是否存在
git config user.name
git config user.email

# 如果为空或失败，则设置默认值
git config user.name "Rick CLI"
git config user.email "rick@localhost"
```

### 2. 自动 Add 和 Commit
```go
// 获取所有需要提交的文件
modified, _ := ac.GetModifiedFiles()
untracked, _ := ac.GetUntrackedFiles()
allFiles := append(modified, untracked...)

// 添加到 staging area
ac.gm.AddFiles(allFiles)

// 提交
ac.CommitJob(jobID, commitMsg)
```

### 3. 变更检测
```go
// 使用 git status --porcelain 检测变更
cmd := exec.Command("git", "status", "--porcelain")
output, _ := cmd.Output()
hasChanges := len(strings.TrimSpace(string(output))) > 0
```

## 与 Morty 的对比

| 特性 | Morty | Rick (修复后) |
|------|-------|---------------|
| Git 初始化 | ✅ 自动初始化 | ✅ 自动初始化 |
| Git 用户配置 | ❌ 需要手动配置 | ✅ 自动配置默认值 |
| 文件 Add | ✅ 自动 add | ✅ 自动 add |
| 空 Commit 检测 | ❌ 无检测 | ✅ 自动跳过 |
| Commit 消息 | ✅ 标准格式 | ✅ 标准格式 |

## 下一步优化

1. **支持全局配置**：允许用户在 `~/.rick/config.json` 中配置默认的 Git 用户信息
2. **Commit 消息模板**：支持自定义 commit 消息模板
3. **Pre-commit Hook**：支持运行 pre-commit 检查
4. **选择性 Add**：支持指定哪些文件需要 commit

## 总结

✅ **问题已解决**：Git commit 失败的问题已完全修复

✅ **自动化配置**：新项目自动配置 Git 用户信息

✅ **智能 Commit**：自动检测变更，跳过空 commit

✅ **向后兼容**：不影响已有 Git 配置

---

**修复完成日期**: 2026-03-14
**修复人**: Claude Opus 4.6
