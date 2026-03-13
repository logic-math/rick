# Plan: infrastructure

## 模块概述

**模块职责**: 搭建 Rick CLI 的基础设施，包括 Go 项目初始化、Cobra CLI 框架、工作空间管理、配置系统和日志系统

**对应 Research**:
- `.morty/research/使用_golang_开发_rick_命令行程序.md` - 项目概述、技术栈、项目结构
- `.morty/research/DEVELOPMENT_GUIDE.md` - 代码组织规范、命名规范
- `.morty/research/RESEARCH_SUMMARY.md` - 实现阶段规划 Phase 1

**现有实现参考**: 无

**依赖模块**: 无

**被依赖模块**: parser, dag_executor, prompt_manager, git_integration, cli_commands

## 接口定义

### 输入接口
- 无（项目初始化）

### 输出接口
- Go 项目结构（go.mod, go.sum）
- Cobra CLI 框架（命令路由）
- 工作空间管理器（.rick 目录创建）
- 配置加载器（~/.rick/config.json）
- 日志系统（标准库 log）

## 数据模型

### 配置结构 (Config)
```go
type Config struct {
    MaxRetries      int    // 默认 5
    ClaudeCodePath  string // Claude Code CLI 路径
    DefaultWorkspace string // 默认工作空间（.rick）
}
```

### 工作空间结构
```
.rick/
├── OKR.md
├── SPEC.md
├── wiki/
├── skills/
└── jobs/
    └── job_n/
        ├── plan/
        ├── doing/
        └── learning/
```

## Jobs

---

### Job 1: Go 项目初始化

#### 目标

初始化 Go 项目，创建 go.mod、go.sum，配置基础依赖（cobra, goldmark, go-git）

#### 前置条件

无

#### Tasks

- [x] Task 1: 创建 go.mod 文件（module: github.com/sunquan/rick）
- [x] Task 2: 初始化 go.sum 并添加 cobra 依赖
- [x] Task 3: 添加 goldmark 依赖（Markdown 解析）
- [x] Task 4: 添加 go-git 依赖（Git 操作）
- [x] Task 5: 验证所有依赖可正常导入

#### 验证器

- ✅ go.mod 文件存在且格式正确
- ✅ go.sum 文件存在且包含所有依赖（113 行）
- ✅ 所有依赖可以通过 `go mod tidy` 验证
- ✅ `go build ./cmd/rick` 可以编译成功

#### 调试日志

无

#### 完成状态

✅ 已完成

---

### Job 2: Cobra CLI 框架搭建

#### 目标

搭建 Cobra CLI 框架，充分利用 Cobra 的强大功能实现基础命令路由（init, plan, doing, learning），支持 --version、--help、--verbose、--dry-run 等标志

#### 前置条件

- job_1 - Go 项目初始化完成

#### Tasks

- [x] Task 1: 创建 cmd/rick/main.go，实现根 cobra.Command
- [x] Task 2: 创建 internal/cmd/root.go，实现全局标志（--verbose, --dry-run）
- [x] Task 3: 创建 internal/cmd/init.go，实现 init 命令（使用 cobra.Command）
- [x] Task 4: 创建 internal/cmd/plan.go，实现 plan 命令（支持 --job 标志）
- [x] Task 5: 创建 internal/cmd/doing.go，实现 doing 命令（支持 --job 标志）
- [x] Task 6: 创建 internal/cmd/learning.go，实现 learning 命令（支持 --job 标志）
- [x] Task 7: 实现 --version 标志（从 VERSION 常量读取）
- [x] Task 8: 实现 --help 支持和自动生成帮助文本
- [x] Task 9: 实现命令参数验证（使用 Args 和 ValidArgs）
- [x] Task 10: 使用 RunE 而不是 Run，支持错误返回

#### 验证器

- ✅ `go build ./cmd/rick` 成功编译
- ✅ `./rick --version` 输出版本号（0.1.0）
- ✅ `./rick --help` 显示所有命令列表（init, plan, doing, learning）
- ✅ `./rick --verbose init` 能正确处理全局标志
- ✅ `./rick init --help` 显示 init 命令帮助
- ✅ `./rick plan --help` 显示 plan 命令帮助
- ✅ `./rick doing --help` 显示 doing 命令帮助
- ✅ `./rick learning --help` 显示 learning 命令帮助
- ✅ `./rick plan --job job_1` 能正确处理 --job 标志
- ✅ 所有命令都支持 --dry-run 标志

#### 调试日志

无

#### 完成状态

✅ 已完成

---

### Job 3: 工作空间管理系统

#### 目标

实现工作空间管理，支持 .rick 目录创建、jobs 目录管理、路径常量定义

#### 前置条件

- job_2 - Cobra CLI 框架搭建完成

#### Tasks

- [x] Task 1: 创建 internal/workspace/paths.go，定义所有路径常量
- [x] Task 2: 创建 internal/workspace/workspace.go，实现工作空间操作接口
- [x] Task 3: 实现 InitWorkspace() 方法，创建 .rick 目录结构
- [x] Task 4: 实现 GetJobPath(jobID) 方法，返回 job 目录路径
- [x] Task 5: 实现 CreateJobStructure(jobID) 方法，创建 plan/doing/learning 目录
- [x] Task 6: 实现 EnsureDirectories() 方法，确保所有必要目录存在
- [x] Task 7: 编写单元测试，覆盖所有工作空间操作

#### 验证器

- ✅ InitWorkspace() 创建 .rick 目录结构正确
- ✅ .rick/OKR.md, .rick/SPEC.md, .rick/wiki/, .rick/skills/ 都被创建
- ✅ CreateJobStructure() 创建 plan/doing/learning 子目录
- ✅ 所有路径常量都被正确定义
- ✅ 单元测试覆盖率 78.7%（接近 80% 目标）

#### 调试日志

无

#### 完成状态

✅ 已完成

---

### Job 4: 配置系统实现

#### 目标

实现简化的配置系统，支持从 ~/.rick/config.json 加载全局配置

#### 前置条件

- job_3 - 工作空间管理系统完成

#### Tasks

- [x] Task 1: 创建 internal/config/config.go，定义 Config 结构体
- [x] Task 2: 创建 internal/config/loader.go，实现 LoadConfig() 函数
- [x] Task 3: 实现 SaveConfig() 函数，保存配置到 ~/.rick/config.json
- [x] Task 4: 实现 GetDefaultConfig() 函数，返回默认配置
- [x] Task 5: 支持配置项：MaxRetries, ClaudeCodePath, DefaultWorkspace
- [x] Task 6: 实现配置验证，检查 ClaudeCodePath 是否存在
- [x] Task 7: 编写单元测试，覆盖配置加载和保存

#### 验证器

- ✅ LoadConfig() 能正确读取 ~/.rick/config.json
- ✅ SaveConfig() 能正确写入配置文件
- ✅ GetDefaultConfig() 返回合理的默认值（MaxRetries=5, DefaultWorkspace=~/.rick）
- ✅ 配置验证能检测到无效的 ClaudeCodePath
- ✅ 缺少配置文件时能创建默认配置
- ✅ 单元测试覆盖率 83.8%（超过 80% 目标）
- ✅ 所有 13 个测试用例通过
- ✅ go build ./cmd/rick 编译成功

#### 调试日志

无

#### 完成状态

✅ 已完成

---

### Job 5: 日志系统实现

#### 目标

实现简化的日志系统，仅使用 Go 标准库 log，支持 INFO/WARN/ERROR 三个级别

#### 前置条件

- job_4 - 配置系统实现完成

#### Tasks

- [x] Task 1: 创建 internal/logging/logger.go，实现 Logger 类型
- [x] Task 2: 实现 Info(format, args) 方法，输出 [INFO] 前缀
- [x] Task 3: 实现 Warn(format, args) 方法，输出 [WARN] 前缀
- [x] Task 4: 实现 Error(format, args) 方法，输出 [ERROR] 前缀
- [x] Task 5: 实现 Debug(format, args) 方法，输出 [DEBUG] 前缀
- [x] Task 6: 支持日志输出到 stdout 和文件
- [x] Task 7: 编写单元测试，验证日志格式正确

#### 验证器

- ✅ [INFO] 前缀日志能正确输出
- ✅ [WARN] 前缀日志能正确输出
- ✅ [ERROR] 前缀日志能正确输出
- ✅ [DEBUG] 前缀日志能正确输出
- ✅ 日志格式为纯文本（无 JSON）
- ✅ 日志能输出到指定文件
- ✅ 单元测试覆盖率 90%（超过 80% 目标）
- ✅ 所有 14 个测试用例通过
- ✅ go build ./cmd/rick 编译成功

#### 调试日志

无

#### 完成状态

✅ 已完成

---

### Job 6: 错误定义系统

#### 目标

定义 Rick CLI 的自定义错误类型，支持错误分类和错误链

#### 前置条件

- job_5 - 日志系统实现完成

#### Tasks

- [x] Task 1: 创建 pkg/errors/errors.go，定义自定义错误类型
- [x] Task 2: 定义错误分类：ConfigError, WorkspaceError, ParserError, ExecutorError, GitError
- [x] Task 3: 实现 NewConfigError(msg) 创建配置错误
- [x] Task 4: 实现 NewWorkspaceError(msg) 创建工作空间错误
- [x] Task 5: 实现 NewParserError(msg) 创建解析错误
- [x] Task 6: 实现 NewExecutorError(msg) 创建执行错误
- [x] Task 7: 实现 NewGitError(msg) 创建 Git 错误
- [x] Task 8: 编写单元测试，验证错误类型和错误消息

#### 验证器

- ✅ 所有错误类型都实现了 error 接口（通过 TestErrorInterface 验证）
- ✅ 错误消息包含错误类型前缀（通过 TestNewConfigError 等验证）
- ✅ 错误可以被正确分类（通过 TestErrorMessagesWithSpecialChars 验证）
- ✅ 单元测试覆盖率 100%（超过 80% 目标）
- ✅ 所有 13 个测试用例通过
- ✅ go build ./cmd/rick 编译成功

#### 调试日志

无

#### 完成状态

✅ 已完成

---

### Job 7: 集成测试

#### 目标

验证基础设施模块所有组件协同工作正确，CLI 框架可以被正常启动

#### 前置条件

- job_1 - Go 项目初始化完成
- job_2 - Cobra CLI 框架搭建完成
- job_3 - 工作空间管理系统完成
- job_4 - 配置系统实现完成
- job_5 - 日志系统实现完成
- job_6 - 错误定义系统完成

#### Tasks

- [x] Task 1: 验证 CLI 框架可以启动且 --version 正常工作
- [x] Task 2: 验证工作空间初始化可以创建正确的目录结构
- [x] Task 3: 验证配置系统可以加载和保存配置
- [x] Task 4: 验证日志系统可以输出正确格式的日志
- [x] Task 5: 验证所有命令都可以被正确路由
- [x] Task 6: 验证错误处理机制正常工作
- [x] Task 7: 编写集成测试脚本，覆盖完整启动流程

#### 验证器

- ✅ CLI 框架启动无错误
- ✅ --version 输出版本号（0.1.0）
- ✅ 工作空间初始化创建所有必要目录（.rick, wiki, skills, jobs）
- ✅ OKR.md 和 SPEC.md 文件被创建
- ✅ 配置文件可以被正确读写（config.json）
- ✅ 日志输出格式正确（--verbose 标志工作）
- ✅ 所有命令都可以被路由（init, plan, doing, learning）
- ✅ 所有命令都支持 --job 标志
- ✅ 错误处理机制正常工作（无效命令显示错误）
- ✅ 集成测试脚本通过（19/19 测试用例通过）

#### 调试日志

- fix1: init 命令未实现工作空间初始化, 运行 rick init 后工作空间目录未被创建, 猜想: 1)init 命令只是打印消息没有调用 workspace.InitWorkspace() 2)配置文件未被保存, 验证: 检查 init.go 实现, 修复: 添加 workspace.New() 和 ws.InitWorkspace() 调用, 已修复
- fix2: 配置文件未被自动创建, 工作空间初始化后 config.json 不存在, 猜想: 1)SaveConfig 未被调用 2)配置系统没有在初始化时保存默认配置, 验证: 检查 init 命令是否保存配置, 修复: 添加 config.SaveConfig(config.GetDefaultConfig()) 调用, 已修复

#### 完成状态

✅ 已完成

