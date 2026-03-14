# Git 自动化技能

## 技能概述

Git 自动化技能通过 Go 的 `os/exec` 包调用 Git 命令行工具，实现仓库初始化、文件提交、日志查询等操作。封装常用 Git 操作为 Go 函数，使 CLI 工具能够自动管理代码版本。

核心特点：**无依赖**。直接调用系统 Git 命令，无需第三方 Git 库（如 go-git），保持简单性和兼容性。

## 使用场景

### 1. 自动提交任务结果
- **场景**: 任务执行成功后自动 commit
- **示例**: Rick CLI 每完成一个任务，自动提交代码
- **价值**: 保证每个任务的结果被版本控制

### 2. 版本回退
- **场景**: 任务失败后回退到上一个版本
- **示例**: 测试未通过，回退代码
- **价值**: 快速恢复到稳定状态

### 3. 提交历史查询
- **场景**: 生成 Learning 阶段的任务总结
- **示例**: 查询 job_1 的所有提交记录
- **价值**: 自动生成工作报告

### 4. 自动初始化仓库
- **场景**: 首次执行 `rick doing` 时自动初始化 Git
- **示例**: 在项目根目录创建 .git 仓库
- **价值**: 简化用户操作

### 5. 分支管理
- **场景**: 为不同 job 创建独立分支
- **示例**: job_1 在 feature/job_1 分支上开发
- **价值**: 隔离不同任务的代码

## 核心优势

### ✅ 优点

1. **无外部依赖**: 直接调用系统 Git，无需第三方库
2. **完全兼容**: 支持所有 Git 命令和参数
3. **简单易用**: 封装常用操作为简单函数
4. **错误处理**: 统一的错误处理和日志记录
5. **灵活性**: 可以调用任何 Git 命令

### ⚠️ 注意事项

1. **依赖系统 Git**: 目标机器必须安装 Git
2. **跨平台问题**: 需要处理 Windows 和 Unix 路径差异
3. **权限问题**: 需要文件系统写权限
4. **错误处理**: Git 命令失败需要正确处理
5. **并发安全**: 多个进程同时操作仓库可能冲突

## 适用条件

- ✅ 目标机器已安装 Git
- ✅ 需要版本控制功能
- ✅ 可以接受命令行调用的开销
- ✅ 不需要复杂的 Git 内部操作

## 不适用场景

- ❌ 需要高性能的 Git 操作（考虑 go-git）
- ❌ 需要访问 Git 内部数据结构
- ❌ 无法安装 Git 的受限环境
- ❌ 需要纯 Go 实现（无外部依赖）

## 关键概念

### exec.Command
```go
cmd := exec.Command("git", "init")
cmd.Dir = repoPath
err := cmd.Run()
```

### 工作目录设置
```go
cmd.Dir = "/path/to/repo"  // 设置 Git 命令的工作目录
```

### 输出捕获
```go
output, err := cmd.Output()  // 捕获 stdout
```

## Rick CLI 中的应用

### 自动初始化 Git（项目根目录）
```go
// 在 rick doing 首次运行时自动初始化
func ensureGitInitialized(projectRoot string) error {
    gm := git.New(projectRoot)
    if !gm.IsRepository() {
        return gm.InitRepo()
    }
    return nil
}
```

### 自动提交任务结果
```go
// 任务成功后自动提交
func commitTaskResult(task *Task) error {
    gm := git.New(projectRoot)
    gm.AddFiles([]string{"."})
    message := fmt.Sprintf("feat(%s): %s", task.ID, task.Name)
    return gm.Commit(message)
}
```

## 实际效果

在 Rick CLI 项目中：
- **初始化速度**: < 100ms
- **提交速度**: < 200ms（小规模文件）
- **日志查询**: < 50ms（10 条记录）

## 扩展阅读

- [Go os/exec 包文档](https://pkg.go.dev/os/exec)
- [Git 命令行文档](https://git-scm.com/docs)
- Rick CLI 源码: `internal/git/git.go`

---

*难度: ⭐⭐*
*分类: 版本控制*
*最后更新: 2026-03-14*
