# Plan: git_integration

## 模块概述

**模块职责**: 实现 Git 集成，支持项目初始化、自动提交、版本管理、回滚操作

**对应 Research**:
- `.morty/research/使用_golang_开发_rick_命令行程序.md` - Git 集成设计
- `.morty/research/DEVELOPMENT_GUIDE.md` - 提交规范

**现有实现参考**: 无

**依赖模块**: infrastructure

**被依赖模块**: dag_executor, cli_commands

## 接口定义

### 输入接口
- 项目路径
- 提交消息
- 文件列表

### 输出接口
- Git 操作结果（初始化、提交、回滚）
- 提交历史

## 数据模型

### GitConfig 结构体
```go
type GitConfig struct {
    RepoPath string
    Author   string
    Email    string
}
```

### CommitInfo 结构体
```go
type CommitInfo struct {
    Hash    string
    Message string
    Author  string
    Date    time.Time
}
```

## Jobs

---

### Job 1: Git 基础操作

#### 目标

实现 Git 基础操作，支持初始化、添加文件、提交、查看日志

#### 前置条件

- infrastructure:job_1 - Go 项目初始化完成

#### Tasks

- [x] Task 1: 创建 internal/git/git.go，实现 Git 操作接口
- [x] Task 2: 实现 InitRepo(path) 函数，初始化 Git 仓库
- [x] Task 3: 实现 AddFiles(paths) 方法，添加文件到暂存区
- [x] Task 4: 实现 Commit(message) 方法，提交变更
- [x] Task 5: 实现 GetLog(limit) 方法，获取提交历史
- [x] Task 6: 实现 GetCurrentBranch() 方法，获取当前分支
- [x] Task 7: 编写单元测试，覆盖 Git 基础操作

#### 验证器

- ✅ InitRepo() 能正确初始化 Git 仓库 - PASS
- ✅ AddFiles() 能正确添加文件 - PASS
- ✅ Commit() 能正确提交变更 - PASS
- ✅ GetLog() 能正确获取提交历史 - PASS
- ✅ GetCurrentBranch() 返回正确的分支名 - PASS
- ✅ 单元测试覆盖率 >= 80% - PASS (86.8% coverage)

#### 调试日志

无

#### 完成状态

✅ 已完成 (2026-03-14 00:50)

---

### Job 2: 自动提交系统

#### 目标

实现自动提交系统，支持每个任务完成后的自动提交

#### 前置条件

- job_1 - Git 基础操作完成

#### Tasks

- [x] Task 1: 创建 internal/git/commit.go，实现自动提交逻辑
- [x] Task 2: 实现 CommitTask(taskID, taskName) 函数
- [x] Task 3: 实现提交消息格式：feat(task_id): 任务名称
- [x] Task 4: 实现 CommitJob(jobID) 函数，提交整个 job
- [x] Task 5: 实现 CommitDebug(jobID, debugInfo) 函数，提交问题记录
- [x] Task 6: 实现自动检测修改的文件
- [x] Task 7: 编写单元测试，覆盖自动提交流程

#### 验证器

- ✅ CommitTask() 能正确提交任务 - PASS
- ✅ 提交消息格式正确 - PASS (feat(task_id): task_name)
- ✅ CommitJob() 能正确提交 job - PASS
- ✅ CommitDebug() 能正确提交问题记录 - PASS
- ✅ 自动检测修改文件正确 - PASS (GetModifiedFiles, GetStagedFiles, GetUntrackedFiles)
- ✅ 单元测试覆盖率 >= 80% - PASS (83.3% coverage)

#### 调试日志

无

#### 完成状态

✅ 已完成 (2026-03-14 01:05)

---

### Job 3: 版本管理

#### 目标

实现版本管理功能，支持版本标签、版本查询、版本回滚

#### 前置条件

- job_2 - 自动提交系统完成

#### Tasks

- [x] Task 1: 创建 internal/git/version.go，实现版本管理
- [x] Task 2: 实现 CreateTag(version, message) 函数，创建版本标签
- [x] Task 3: 实现 GetCurrentVersion() 函数，获取当前版本
- [x] Task 4: 实现 ListVersions() 函数，列出所有版本
- [x] Task 5: 实现 Checkout(version) 函数，切换到指定版本
- [x] Task 6: 实现版本号格式验证（vMAJOR.MINOR.PATCH）
- [x] Task 7: 编写单元测试，覆盖版本管理

#### 验证器

- ✅ CreateTag() 能正确创建版本标签 - PASS
- ✅ GetCurrentVersion() 返回正确的版本号 - PASS
- ✅ ListVersions() 返回所有版本列表 - PASS
- ✅ Checkout() 能正确切换版本 - PASS
- ✅ 版本号格式验证正确 - PASS (vMAJOR.MINOR.PATCH)
- ✅ 单元测试覆盖率 >= 80% - PASS (82.5% coverage)

#### 调试日志

无

#### 完成状态

✅ 已完成 (2026-03-14 01:15)

---

### Job 4: 回滚和恢复

#### 目标

实现回滚和恢复功能，支持回滚到指定版本或提交

#### 前置条件

- job_3 - 版本管理完成

#### Tasks

- [x] Task 1: 创建 internal/git/rollback.go，实现回滚逻辑
- [x] Task 2: 实现 ResetToCommit(hash) 函数，回滚到指定提交
- [x] Task 3: 实现 ResetToVersion(version) 函数，回滚到指定版本
- [x] Task 4: 实现 GetDiff(fromCommit, toCommit) 函数，查看差异
- [x] Task 5: 实现 GetFileHistory(filePath) 函数，获取文件历史
- [x] Task 6: 实现安全检查，防止误操作
- [x] Task 7: 编写单元测试，覆盖回滚操作

#### 验证器

- ✅ ResetToCommit() 能正确回滚 - PASS
- ✅ ResetToVersion() 能正确回滚 - PASS
- ✅ GetDiff() 能正确显示差异 - PASS
- ✅ GetFileHistory() 能正确获取文件历史 - PASS
- ✅ 安全检查正常工作 - PASS (checkUncommittedChanges, commitExists)
- ✅ 单元测试覆盖率 >= 80% - PASS (83.2% coverage)

#### 调试日志

无

#### 完成状态

✅ 已完成 (2026-03-14 02:15)

---

### Job 5: 集成测试

#### 目标

验证 git_integration 模块所有组件协同工作正确，能正确管理 Git 仓库

#### 前置条件

- job_1 - Git 基础操作完成
- job_2 - 自动提交系统完成
- job_3 - 版本管理完成
- job_4 - 回滚和恢复完成

#### Tasks

- [ ] Task 1: 验证 Git 仓库初始化正确
- [ ] Task 2: 验证文件添加和提交正确
- [ ] Task 3: 验证自动提交系统正常工作
- [ ] Task 4: 验证版本管理正常工作
- [ ] Task 5: 验证回滚和恢复正常工作
- [ ] Task 6: 验证提交历史记录正确
- [ ] Task 7: 编写集成测试脚本，覆盖完整 Git 工作流

#### 验证器

- Git 仓库初始化成功
- 文件添加和提交成功
- 自动提交系统正常工作
- 版本管理正常工作
- 回滚和恢复正常工作
- 提交历史记录完整
- 集成测试脚本通过

#### 调试日志

无

#### 完成状态

⏳ 待开始

