# Rick CLI - 技术规范文档 (SPEC)

> **版本**: v1.0
> **最后更新**: 2026-03-14
> **状态**: ✅ 已审核

---

## 📋 文档概述

本文档定义 Rick CLI 项目的技术规范，包括代码风格、测试规范、Git 工作流、文档规范和发布流程。所有贡献者必须遵循本规范，以确保代码质量和团队协作的一致性。

---

## 1. 代码风格规范

### 1.1 Go 语言规范

#### 基础规范
- **遵循官方规范**: 严格遵循 [Effective Go](https://golang.org/doc/effective_go.html) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- **格式化工具**: 使用 `gofmt` 或 `goimports` 格式化代码
- **Linter**: 使用 `golangci-lint` 进行代码检查

#### 命名规范

**1. 包名（Package）**
```go
// ✅ 推荐：小写，单词，简短
package parser
package executor
package workspace

// ❌ 避免：下划线、驼峰、复数
package task_parser
package TaskParser
package parsers
```

**2. 文件名**
```bash
# ✅ 推荐：小写，下划线分隔
task_parser.go
auto_committer.go
context_manager.go

# ❌ 避免：驼峰、中划线
TaskParser.go
task-parser.go
```

**3. 变量和函数名**
```go
// ✅ 推荐：驼峰命名
var taskID string
var maxRetries int
func ParseTask(content string) (*Task, error)
func NewExecutor(tasks []*Task) *Executor

// ❌ 避免：下划线、全大写（非常量）
var task_id string
var MAX_RETRIES int
func parse_task(content string) (*Task, error)
```

**4. 常量**
```go
// ✅ 推荐：驼峰或全大写（视情况）
const DefaultMaxRetries = 5
const VERSION = "0.1.0"
const (
    StatusPending   = "pending"
    StatusCompleted = "completed"
)

// ❌ 避免：小写或混合风格
const default_max_retries = 5
```

**5. 结构体和接口**
```go
// ✅ 推荐：大写开头（导出），驼峰
type Task struct {
    ID           string
    Name         string
    Dependencies []string
}

type TaskParser interface {
    Parse(content string) (*Task, error)
}

// ❌ 避免：小写开头（除非内部使用）、下划线
type task struct { ... }
type Task_Parser interface { ... }
```

#### 注释规范

**1. 包注释**
```go
// Package parser provides task parsing functionality.
// It supports parsing task.md files into Task structs.
package parser
```

**2. 函数注释**
```go
// ParseTask parses a complete task.md content and returns a Task struct.
// It extracts task name, goal, key results, test method, and dependencies.
//
// Returns an error if:
//   - content is empty
//   - required sections are missing (task name, goal, test method)
//   - parsing fails for any section
func ParseTask(content string) (*Task, error) {
    // Implementation
}
```

**3. 结构体注释**
```go
// Task represents a task extracted from task.md.
// It contains all necessary information for task execution.
type Task struct {
    // ID is the unique identifier for the task (e.g., "task1", "task2")
    ID string

    // Name is the human-readable task name
    Name string

    // Dependencies is a list of task IDs that must be completed before this task
    Dependencies []string
}
```

**4. 注释原则**
- 导出的函数、类型、常量必须有注释
- 注释以类型/函数名开头，使用完整句子
- 复杂逻辑必须添加行内注释
- 避免无意义的注释（如 `// set x to 10`）

#### 错误处理规范

**1. 错误返回**
```go
// ✅ 推荐：返回详细的错误信息
func ParseTask(content string) (*Task, error) {
    if content == "" {
        return nil, fmt.Errorf("content cannot be empty")
    }

    name, err := ParseTaskName(content)
    if err != nil {
        return nil, fmt.Errorf("failed to parse task name: %w", err)
    }

    return task, nil
}

// ❌ 避免：返回 nil 或忽略错误
func ParseTask(content string) *Task {
    name, _ := ParseTaskName(content) // 忽略错误
    return task
}
```

**2. 错误检查**
```go
// ✅ 推荐：立即检查错误
task, err := ParseTask(content)
if err != nil {
    return fmt.Errorf("failed to parse task: %w", err)
}

// ❌ 避免：延迟检查或忽略
task, _ := ParseTask(content)
```

**3. 错误包装**
```go
// ✅ 推荐：使用 %w 包装错误，保留错误链
return fmt.Errorf("failed to load config: %w", err)

// ❌ 避免：使用 %v 丢失错误链
return fmt.Errorf("failed to load config: %v", err)
```

#### 代码组织规范

**1. 导入顺序**
```go
import (
    // 标准库
    "fmt"
    "os"
    "path/filepath"

    // 第三方库
    "github.com/spf13/cobra"
    "github.com/yuin/goldmark"

    // 本项目包
    "github.com/sunquan/rick/internal/config"
    "github.com/sunquan/rick/internal/parser"
)
```

**2. 结构体字段顺序**
```go
type Task struct {
    // 导出字段（按重要性排序）
    ID           string
    Name         string
    Goal         string
    KeyResults   []string
    TestMethod   string
    Dependencies []string

    // 内部字段
    status       string
    retryCount   int
}
```

**3. 函数顺序**
```go
// 1. 导出的构造函数
func NewExecutor(...) *Executor { }

// 2. 导出的公共方法
func (e *Executor) ExecuteJob() error { }

// 3. 导出的辅助方法
func (e *Executor) GetStatus() string { }

// 4. 内部方法（小写开头）
func (e *Executor) executeTask() error { }
```

### 1.2 项目结构规范

#### 目录结构
```
rick/
├── cmd/rick/               # 命令行入口
│   └── main.go             # main 函数，简洁明了
├── internal/               # 内部包（不对外暴露）
│   ├── cmd/                # 命令处理器
│   ├── config/             # 配置管理
│   ├── workspace/          # 工作空间管理
│   ├── parser/             # 内容解析
│   ├── executor/           # 任务执行
│   ├── prompt/             # 提示词管理
│   ├── git/                # Git 操作
│   └── callcli/            # Claude Code CLI 交互
├── pkg/                    # 公共包（可对外暴露）
├── scripts/                # 构建和安装脚本
├── tests/                  # 集成测试
└── .rick/                  # Rick 自身的工作空间
```

#### 包设计原则
1. **单一职责**: 每个包只负责一个明确的功能领域
2. **最小依赖**: 优先使用 Go 标准库，谨慎引入第三方库
3. **循环依赖**: 严禁包之间的循环依赖
4. **internal 优先**: 除非明确需要对外暴露，否则放在 internal/

### 1.3 依赖管理规范

#### 依赖原则
1. **最小化原则**: 能用标准库就不引入第三方库
2. **稳定性优先**: 优先选择成熟、活跃维护的库
3. **许可证兼容**: 确保依赖的许可证与项目兼容

#### 当前依赖
```go
// 核心依赖（必需）
require (
    github.com/spf13/cobra v1.10.2      // CLI 框架
    github.com/yuin/goldmark v1.7.16    // Markdown 解析
    github.com/go-git/go-git/v5 v5.17.0 // Git 操作
)
```

#### 依赖更新规范
- 定期检查依赖更新（每月）
- 重大版本更新需要充分测试
- 记录依赖更新原因和影响

---

## 2. 测试规范

### 2.1 测试策略

#### 测试金字塔
```
       /\
      /E2E\        10%  - 端到端测试
     /------\
    /  集成  \      20%  - 集成测试
   /----------\
  /   单元测试  \    70%  - 单元测试
 /--------------\
```

### 2.2 单元测试规范

#### 文件组织
```go
// 源文件: internal/parser/task.go
// 测试文件: internal/parser/task_test.go
package parser

import "testing"

// 测试函数命名: Test<FunctionName>
func TestParseTask(t *testing.T) { }
func TestParseTaskName(t *testing.T) { }
func TestValidateTask(t *testing.T) { }
```

#### 测试结构（AAA 模式）
```go
func TestParseTask(t *testing.T) {
    // Arrange - 准备测试数据
    content := `
# 依赖关系
task1, task2

# 任务名称
示例任务

# 任务目标
完成示例功能
...
`

    // Act - 执行被测试的函数
    task, err := ParseTask(content)

    // Assert - 验证结果
    if err != nil {
        t.Fatalf("ParseTask failed: %v", err)
    }
    if task.Name != "示例任务" {
        t.Errorf("expected task name '示例任务', got '%s'", task.Name)
    }
}
```

#### 表驱动测试
```go
func TestParseTaskName(t *testing.T) {
    tests := []struct {
        name    string
        content string
        want    string
        wantErr bool
    }{
        {
            name:    "valid task name",
            content: "# 任务名称\n示例任务\n",
            want:    "示例任务",
            wantErr: false,
        },
        {
            name:    "missing task name",
            content: "# 其他标题\n内容\n",
            want:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseTaskName(tt.content)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseTaskName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ParseTaskName() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

#### 测试覆盖率要求
- **目标**: 整体覆盖率 ≥ 80%
- **核心模块**: 覆盖率 ≥ 90%（parser, executor, prompt）
- **命令模块**: 覆盖率 ≥ 70%（cmd）
- **工具函数**: 覆盖率 100%

#### 运行测试
```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/parser

# 运行测试并生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行测试（详细输出）
go test -v ./...
```

### 2.3 集成测试规范

#### 测试文件命名
```bash
# 集成测试文件以 _integration_test.go 结尾
internal/cmd/cli_integration_test.go
tests/workflow_integration_test.go
```

#### 集成测试示例
```go
func TestPlanDoingWorkflow(t *testing.T) {
    // 跳过短测试模式
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    // 创建临时工作目录
    tmpDir := t.TempDir()

    // 测试 plan 阶段
    // ...

    // 测试 doing 阶段
    // ...

    // 验证结果
    // ...
}
```

#### 运行集成测试
```bash
# 运行所有测试（包括集成测试）
go test ./...

# 仅运行单元测试（跳过集成测试）
go test -short ./...

# 仅运行集成测试
go test -run Integration ./...
```

### 2.4 E2E 测试规范

#### E2E 测试脚本
```bash
# tests/e2e_test.sh - 端到端测试脚本
#!/bin/bash

# 1. 安装 Rick
./scripts/install.sh --dev

# 2. 测试 plan 命令
rick_dev plan "示例需求"

# 3. 测试 doing 命令
rick_dev doing job_1

# 4. 测试 learning 命令
rick_dev learning job_1

# 5. 验证结果
# ...
```

#### E2E 测试覆盖场景
1. 完整的 plan → doing → learning 工作流
2. 失败重试和恢复机制
3. 并行版本管理（rick + rick_dev）
4. 自我重构能力验证
5. 性能和稳定性测试

---

## 3. Git 工作流规范

### 3.1 分支管理策略

#### 分支类型
```
main          - 主分支（稳定版本）
develop       - 开发分支（集成分支）
feature/*     - 功能分支
bugfix/*      - 修复分支
hotfix/*      - 紧急修复分支
release/*     - 发布分支
```

#### 分支命名规范
```bash
# 功能分支
feature/prompt-manager
feature/dag-executor

# 修复分支
bugfix/task-parser-error
bugfix/git-commit-failure

# 紧急修复
hotfix/critical-security-issue

# 发布分支
release/v1.0.0
```

### 3.2 提交信息规范

#### 提交格式（Conventional Commits）
```
<type>(<scope>): <subject>

<body>

<footer>
```

#### 类型（Type）
```bash
feat:     新功能
fix:      修复 bug
docs:     文档更新
style:    代码格式调整（不影响功能）
refactor: 重构（不改变功能）
test:     测试相关
chore:    构建过程或辅助工具的变动
perf:     性能优化
```

#### 提交示例
```bash
# 新功能
feat(parser): add support for parsing dependencies

# 修复 bug
fix(executor): fix retry mechanism not working

# 文档更新
docs(readme): update installation instructions

# 重构
refactor(prompt): simplify template loading logic

# 测试
test(parser): add unit tests for ParseTask

# 性能优化
perf(dag): optimize topological sort algorithm
```

#### 提交消息规范
1. **主题行（Subject）**
   - 不超过 50 字符
   - 使用动词开头（add, fix, update, remove）
   - 不以句号结尾
   - 使用英文（中文项目可用中文）

2. **正文（Body）**
   - 与主题行之间空一行
   - 解释"为什么"而不是"是什么"
   - 每行不超过 72 字符

3. **页脚（Footer）**
   - 关闭 Issue: `Closes #123`
   - 破坏性变更: `BREAKING CHANGE: ...`

### 3.3 自动提交规范（Rick 特有）

#### Rick 自动提交格式
```bash
morty: <phase> <job_id> - <status>

<details>

Co-Authored-By: Claude <model> <noreply@anthropic.com>
```

#### 示例
```bash
morty: doing job_1 - COMPLETED

All 5 tasks executed successfully.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
```

### 3.4 PR（Pull Request）规范

#### PR 标题格式
```
<type>: <description>
```

#### PR 描述模板
```markdown
## 变更类型
- [ ] 新功能
- [ ] Bug 修复
- [ ] 文档更新
- [ ] 重构
- [ ] 性能优化
- [ ] 测试

## 变更描述
简要描述本次 PR 的目的和内容。

## 相关 Issue
Closes #123

## 测试情况
- [ ] 单元测试已通过
- [ ] 集成测试已通过
- [ ] 手动测试已完成

## 检查清单
- [ ] 代码遵循项目规范
- [ ] 已添加必要的注释
- [ ] 已更新相关文档
- [ ] 已添加或更新测试
- [ ] 所有测试通过
```

#### PR 审查要求
1. 至少 1 人审查批准
2. 所有 CI/CD 检查通过
3. 代码覆盖率不降低
4. 无冲突

---

## 4. 文档规范

### 4.1 文档类型

#### 必需文档
```
README.md           - 项目概述和快速入门
DEVELOPMENT_GUIDE.md - 开发指南
CHANGELOG.md        - 变更日志
.rick/OKR.md        - 项目目标和关键结果
.rick/SPEC.md       - 技术规范（本文档）
```

#### 可选文档
```
CONTRIBUTING.md     - 贡献指南
CODE_OF_CONDUCT.md  - 行为准则
SECURITY.md         - 安全政策
```

### 4.2 代码注释规范

#### 包级别文档
```go
// Package parser provides task parsing functionality.
//
// This package supports parsing task.md files into Task structs,
// extracting dependencies, goals, key results, and test methods.
//
// Example usage:
//   content, _ := os.ReadFile("task1.md")
//   task, err := parser.ParseTask(string(content))
//   if err != nil {
//       log.Fatal(err)
//   }
//   fmt.Println(task.Name)
package parser
```

#### 函数级别文档
```go
// ParseTask parses a complete task.md content and returns a Task struct.
//
// The task.md file must follow this format:
//   # 依赖关系
//   task1, task2
//
//   # 任务名称
//   Task Name
//
//   # 任务目标
//   Task Goal
//
//   # 关键结果
//   - Result 1
//   - Result 2
//
//   # 测试方法
//   - Test 1
//   - Test 2
//
// Returns an error if:
//   - content is empty
//   - required sections are missing (task name, goal, test method)
//   - parsing fails for any section
func ParseTask(content string) (*Task, error) {
    // Implementation
}
```

### 4.3 README 规范

#### README 结构
```markdown
# 项目名称

简短描述（一句话）

## 特性

- 特性 1
- 特性 2
- 特性 3

## 快速开始

### 安装

### 使用

### 示例

## 文档

## 贡献

## 许可证
```

### 4.4 Changelog 规范

#### Changelog 格式（Keep a Changelog）
```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- New feature X

### Changed
- Improved feature Y

### Fixed
- Bug Z

## [1.0.0] - 2026-03-14

### Added
- Initial release
- Core features: plan, doing, learning
```

---

## 5. 发布流程规范

### 5.1 版本号规范（Semantic Versioning）

#### 版本格式
```
vMAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]

示例:
v1.0.0
v1.1.0-beta
v2.0.0-rc.1
v1.0.1+20260314
```

#### 版本递增规则
- **MAJOR**: 不兼容的 API 变更
- **MINOR**: 向后兼容的新功能
- **PATCH**: 向后兼容的 bug 修复
- **PRERELEASE**: 预发布版本（alpha, beta, rc）
- **BUILD**: 构建元数据

### 5.2 发布检查清单

#### 发布前检查
- [ ] 所有测试通过（单元、集成、E2E）
- [ ] 代码覆盖率 ≥ 80%
- [ ] 文档已更新（README, CHANGELOG）
- [ ] 版本号已更新（cmd/rick/main.go）
- [ ] 已创建 Git tag
- [ ] 已推送到 GitHub

#### 发布步骤
```bash
# 1. 更新版本号
vim cmd/rick/main.go  # 修改 VERSION 常量

# 2. 更新 CHANGELOG
vim CHANGELOG.md

# 3. 提交变更
git add .
git commit -m "chore: bump version to v1.0.0"

# 4. 创建 tag
git tag -a v1.0.0 -m "Release v1.0.0"

# 5. 推送到远程
git push origin main
git push origin v1.0.0

# 6. 创建 GitHub Release
gh release create v1.0.0 --title "v1.0.0" --notes "Release notes"

# 7. 构建和上传二进制文件
./scripts/build.sh
gh release upload v1.0.0 bin/rick
```

### 5.3 发布周期

#### 版本类型
- **Major Release**: 每年 1-2 次（重大功能或架构变更）
- **Minor Release**: 每季度 1-2 次（新功能）
- **Patch Release**: 按需发布（bug 修复）

#### 发布分支策略
```bash
# 创建发布分支
git checkout -b release/v1.0.0 develop

# 在发布分支上进行最后的调整
# ...

# 合并到 main
git checkout main
git merge --no-ff release/v1.0.0

# 合并回 develop
git checkout develop
git merge --no-ff release/v1.0.0

# 删除发布分支
git branch -d release/v1.0.0
```

---

## 6. 性能规范

### 6.1 性能目标

#### 命令响应时间
- `rick plan`: < 5 秒
- `rick doing`: 平均每任务 < 5 分钟
- `rick learning`: < 10 秒

#### 资源占用
- 内存占用: < 500MB（100 个任务）
- CPU 占用: < 50%（单核）
- 磁盘占用: < 100MB（不含日志）

### 6.2 性能测试

#### 性能测试工具
```bash
# 使用 Go 内置的性能测试
go test -bench=. -benchmem ./...

# 使用 pprof 进行性能分析
go test -cpuprofile=cpu.prof -memprofile=mem.prof
go tool pprof cpu.prof
```

#### 性能基准测试
```go
func BenchmarkParseTask(b *testing.B) {
    content := loadTestContent()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        ParseTask(content)
    }
}
```

---

## 7. 安全规范

### 7.1 安全原则

1. **最小权限原则**: 只请求必要的权限
2. **输入验证**: 验证所有外部输入
3. **输出编码**: 防止注入攻击
4. **敏感数据保护**: 不在日志中记录敏感信息

### 7.2 安全检查清单

- [ ] 不在代码中硬编码密钥或密码
- [ ] 不在日志中记录敏感信息
- [ ] 验证所有用户输入
- [ ] 使用参数化查询（如适用）
- [ ] 定期更新依赖以修复安全漏洞
- [ ] 使用 HTTPS 进行网络通信

### 7.3 安全工具

```bash
# 使用 gosec 进行安全扫描
gosec ./...

# 检查依赖的安全漏洞
go list -json -m all | nancy sleuth
```

---

## 8. 持续集成/持续部署（CI/CD）

### 8.1 CI 流程

#### GitHub Actions 工作流
```yaml
name: CI

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test -v -coverprofile=coverage.out ./...
      - run: go tool cover -func=coverage.out

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: golangci/golangci-lint-action@v3
```

### 8.2 CI 检查项

- [ ] 代码编译通过
- [ ] 单元测试通过
- [ ] 集成测试通过
- [ ] 代码覆盖率 ≥ 80%
- [ ] Linter 检查通过
- [ ] 安全扫描通过

---

## 9. 代码审查规范

### 9.1 审查检查清单

#### 功能性
- [ ] 代码实现了需求
- [ ] 代码逻辑正确
- [ ] 边界条件处理正确
- [ ] 错误处理完善

#### 可读性
- [ ] 代码清晰易懂
- [ ] 命名恰当
- [ ] 注释充分
- [ ] 结构合理

#### 可维护性
- [ ] 代码遵循 DRY 原则
- [ ] 函数职责单一
- [ ] 没有过度设计
- [ ] 易于测试

#### 性能
- [ ] 没有明显的性能问题
- [ ] 算法复杂度合理
- [ ] 资源使用合理

#### 测试
- [ ] 测试覆盖充分
- [ ] 测试用例合理
- [ ] 测试通过

### 9.2 审查流程

1. **提交 PR**: 开发者提交 Pull Request
2. **自动检查**: CI/CD 自动运行测试和检查
3. **代码审查**: 至少 1 人审查代码
4. **修改反馈**: 根据审查意见修改代码
5. **批准合并**: 审查通过后合并到主分支

---

## 10. 附录

### 10.1 工具和资源

#### 开发工具
- **IDE**: VSCode, GoLand
- **Linter**: golangci-lint
- **测试**: Go testing, testify
- **性能分析**: pprof
- **安全扫描**: gosec

#### 学习资源
- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- [Semantic Versioning](https://semver.org/)

### 10.2 常见问题

#### Q: 为什么要遵循这些规范？
A: 规范确保代码质量、可读性和可维护性，降低团队协作成本。

#### Q: 规范太严格了，会不会降低开发效率？
A: 短期内可能需要适应，但长期来看会大幅提升效率和质量。

#### Q: 如果规范与实际情况冲突怎么办？
A: 规范不是一成不变的，可以根据实际情况调整，但需要团队讨论和共识。

### 10.3 规范更新

#### 更新流程
1. 提出规范变更建议（Issue 或 PR）
2. 团队讨论和评审
3. 达成共识后更新 SPEC.md
4. 通知所有团队成员

#### 版本历史
- **v1.0** (2026-03-14): 初始版本，定义核心规范

---

**文档版本**: v1.0
**最后更新**: 2026-03-14
**维护者**: Rick Core Team
**审核状态**: ✅ 已完成
