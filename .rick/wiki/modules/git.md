# Git Module（Git 操作模块）

## 概述
Git Module 负责 Git 相关操作，包括自动初始化、提交、分支管理和状态检查。

## 模块位置
`internal/git/`

## 核心功能

### 1. 自动初始化 Git
**职责**: 在项目根目录自动初始化 Git 仓库

**触发时机**: 首次执行 `rick doing` 命令时

**核心函数**:
```go
// EnsureGitInitialized 确保 Git 仓库已初始化
func EnsureGitInitialized(projectRoot string) error {
    // 1. 检查 .git/ 是否存在
    gitDir := filepath.Join(projectRoot, ".git")
    if _, err := os.Stat(gitDir); err == nil {
        // 已初始化，直接返回
        return nil
    }

    // 2. 初始化 Git 仓库
    cmd := exec.Command("git", "init")
    cmd.Dir = projectRoot
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("git init failed: %s", output)
    }

    log.Println("[INFO] Git 仓库已初始化")
    return nil
}
```

### 2. Git 提交
**职责**: 提交代码变更

**触发时机**: 每个 task 执行成功后

**核心函数**:
```go
// Commit 提交代码
func Commit(message string) error {
    // 1. git add .
    addCmd := exec.Command("git", "add", ".")
    if err := addCmd.Run(); err != nil {
        return fmt.Errorf("git add failed: %w", err)
    }

    // 2. git commit
    commitCmd := exec.Command("git", "commit", "-m", message)
    output, err := commitCmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("git commit failed: %s", output)
    }

    log.Printf("[INFO] Git 提交成功: %s", message)
    return nil
}
```

**提交信息格式**:
```
feat: 完成 task1 - 创建基础设施模块
fix: 修复 task2 - 解析器 bug
test: 添加 task3 - 单元测试
```

### 3. 分支管理
**职责**: 创建、切换、删除分支

**核心函数**:
```go
// CreateBranch 创建分支
func CreateBranch(branchName string) error {
    cmd := exec.Command("git", "branch", branchName)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("git branch failed: %s", output)
    }
    return nil
}

// CheckoutBranch 切换分支
func CheckoutBranch(branchName string) error {
    cmd := exec.Command("git", "checkout", branchName)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("git checkout failed: %s", output)
    }
    return nil
}

// DeleteBranch 删除分支
func DeleteBranch(branchName string) error {
    cmd := exec.Command("git", "branch", "-d", branchName)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("git branch -d failed: %s", output)
    }
    return nil
}
```

### 4. Git 状态检查
**职责**: 检查 Git 状态

**核心函数**:
```go
// GetStatus 获取 Git 状态
func GetStatus() (string, error) {
    cmd := exec.Command("git", "status")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return "", fmt.Errorf("git status failed: %w", err)
    }
    return string(output), nil
}

// HasUncommittedChanges 检查是否有未提交的变更
func HasUncommittedChanges() (bool, error) {
    status, err := GetStatus()
    if err != nil {
        return false, err
    }

    // 检查是否包含 "nothing to commit"
    return !strings.Contains(status, "nothing to commit"), nil
}
```

### 5. Git 日志
**职责**: 获取 Git 提交日志

**核心函数**:
```go
// GetLog 获取 Git 日志
func GetLog(limit int) ([]string, error) {
    cmd := exec.Command("git", "log", fmt.Sprintf("-%d", limit), "--oneline")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("git log failed: %w", err)
    }

    lines := strings.Split(string(output), "\n")
    return lines, nil
}

// GetLogForJob 获取特定 job 的 Git 日志
func GetLogForJob(jobID string) ([]string, error) {
    // 搜索包含 job_id 的提交
    cmd := exec.Command("git", "log", "--grep", jobID, "--oneline")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("git log failed: %w", err)
    }

    lines := strings.Split(string(output), "\n")
    return lines, nil
}
```

## 使用示例

### 示例1: 自动初始化 Git
```go
func main() {
    // 首次 doing 时自动初始化
    err := git.EnsureGitInitialized(".")
    if err != nil {
        log.Fatal(err)
    }
}
```

### 示例2: 提交代码
```go
func executeTask(task *Task) error {
    // 执行任务...

    // 任务成功，提交代码
    commitMsg := fmt.Sprintf("feat: 完成 %s", task.TaskName)
    err := git.Commit(commitMsg)
    if err != nil {
        return err
    }

    return nil
}
```

### 示例3: 检查状态
```go
func beforeExecute() error {
    // 检查是否有未提交的变更
    hasChanges, err := git.HasUncommittedChanges()
    if err != nil {
        return err
    }

    if hasChanges {
        log.Println("[WARN] 存在未提交的变更，请先提交")
        return errors.New("uncommitted changes")
    }

    return nil
}
```

## Git 工作流

### Rick 的 Git 工作流
```
1. 首次 doing → 自动 git init

2. 执行 task1
   ├─ 编写代码
   ├─ 运行测试
   ├─ 测试通过 → git commit "feat: 完成 task1"
   └─ 测试失败 → 重试（不提交）

3. 执行 task2
   ├─ 编写代码
   ├─ 运行测试
   ├─ 测试通过 → git commit "feat: 完成 task2"
   └─ 测试失败 → 重试（不提交）

4. 所有任务完成
   └─ git log 查看提交历史
```

### 提交信息规范
遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
feat: 新功能
fix: 修复 bug
test: 添加测试
docs: 文档更新
refactor: 重构
style: 代码格式
chore: 构建/工具变更
```

**示例**:
```
feat: 完成 task1 - 创建基础设施模块
fix: 修复 task2 - 解析器依赖解析 bug
test: 添加 task3 - DAG 执行器单元测试
```

## 错误处理

### 常见错误
1. **Git 未安装**: 检查 `git` 命令是否可用
2. **权限问题**: 检查目录权限
3. **合并冲突**: 提示用户手动解决
4. **网络问题**: push/pull 失败时提示

### 错误处理示例
```go
func Commit(message string) error {
    err := exec.Command("git", "add", ".").Run()
    if err != nil {
        return fmt.Errorf("git add failed: %w", err)
    }

    err = exec.Command("git", "commit", "-m", message).Run()
    if err != nil {
        // 检查是否因为没有变更而失败
        if strings.Contains(err.Error(), "nothing to commit") {
            log.Println("[INFO] 没有需要提交的变更")
            return nil
        }
        return fmt.Errorf("git commit failed: %w", err)
    }

    return nil
}
```

## 测试

### 单元测试
```bash
go test ./internal/git/
```

### 测试用例
```go
func TestEnsureGitInitialized(t *testing.T) {
    tmpDir, _ := os.MkdirTemp("", "test")
    defer os.RemoveAll(tmpDir)

    err := EnsureGitInitialized(tmpDir)
    if err != nil {
        t.Fatal(err)
    }

    // 验证 .git/ 目录存在
    gitDir := filepath.Join(tmpDir, ".git")
    if _, err := os.Stat(gitDir); os.IsNotExist(err) {
        t.Error(".git directory should exist")
    }
}

func TestCommit(t *testing.T) {
    // 创建测试仓库
    tmpDir, _ := os.MkdirTemp("", "test")
    defer os.RemoveAll(tmpDir)

    EnsureGitInitialized(tmpDir)

    // 创建测试文件
    os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)

    // 提交
    err := Commit("test commit")
    if err != nil {
        t.Fatal(err)
    }

    // 验证提交
    log, _ := GetLog(1)
    if !strings.Contains(log[0], "test commit") {
        t.Error("commit message should contain 'test commit'")
    }
}
```

## 最佳实践

1. **提交粒度**: 每个 task 完成后立即提交，避免大量变更
2. **提交信息**: 使用清晰的提交信息，包含 task 名称
3. **错误处理**: 提交失败时记录详细错误信息
4. **状态检查**: 执行前检查 Git 状态，避免冲突

## 常见问题

### Q1: 如何回滚到上一个提交？
**A**: `git reset --hard HEAD~1`

### Q2: 如何查看特定 task 的提交？
**A**: 使用 `GetLogForJob(jobID)` 函数。

### Q3: 如何处理合并冲突？
**A**: Rick 不自动处理合并冲突，需要人工解决。

## 未来优化

1. **Git Hooks**: 支持 pre-commit、post-commit hooks
2. **分支策略**: 支持为每个 job 创建独立分支
3. **远程仓库**: 支持自动 push 到远程仓库
4. **冲突检测**: 自动检测合并冲突并提示

---

*最后更新: 2026-03-14*
