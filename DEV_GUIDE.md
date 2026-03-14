# Rick CLI Dev 版本开发指南

## 当前状态（2026-03-14）

### Dev 版本已创建
```bash
命令名：rick_dev
安装位置：~/.rick_dev/
二进制路径：~/.rick_dev/bin/rick
符号链接：~/.local/bin/rick_dev
```

### 验证安装
```bash
rick_dev --version
# 输出：Rick CLI version 0.1.0

rick_dev --help
# 显示所有可用命令
```

## 项目结构理解

### 核心代码位置
```
/Users/sunquan/ai_coding/CODING/rick/
├── cmd/rick/main.go              # 入口点
├── internal/
│   ├── cmd/
│   │   ├── init.go               # Init 命令（需优化）
│   │   ├── plan.go               # Plan 命令
│   │   ├── doing.go              # Doing 命令
│   │   └── learning.go           # Learning 命令
│   ├── executor/                 # 任务执行引擎
│   ├── parser/                   # 内容解析
│   ├── prompt/                   # 提示词管理
│   ├── config/                   # 配置管理
│   ├── workspace/                # 工作空间管理
│   └── git/                      # Git 操作
└── scripts/
    ├── build.sh                  # 构建脚本
    ├── install.sh                # 安装脚本
    └── uninstall.sh              # 卸载脚本
```

## Init 命令优化任务

### 当前实现（问题分析）

**文件**: `internal/cmd/init.go`

当前 init 仅执行：
```go
1. ws.InitWorkspace()   // 创建 .rick 目录结构
2. config.SaveConfig()  // 保存默认配置
3. initGitRepo()        // 初始化 Git 仓库
```

**问题**：
- ❌ 没有自动执行 plan → doing → learning 流程
- ❌ 没有进行深度源码探索
- ❌ 没有生成 OKR.md、SPEC.md、Wiki、Skills 等全局上下文
- ❌ 与规范（Rick_Project_Complete_Description.md 第 354-370 行）不符

### 规范要求

```
rick init
  ├─ 创建.rick目录结构
  ├─ 自动生成源码探索task.md（跨越上下文窗口深度探索）
  ├─ 执行 plan 生成 job_0
  ├─ 执行 doing job_0（自动执行源码探索）
  ├─ 生成.rick/wiki/（深度调研报告，含index.md索引）
  ├─ 执行 learning job_0（自动化）
  │  ├─ 读取debug.md + wiki
  │  └─ 生成.rick/OKR.md、SPEC.md、skills/
  └─ 人类审核并调整OKR.md和SPEC.md
```

### 优化方案

#### 方案 A：组合流程（推荐）

在 `init.go` 中修改 `NewInitCmd()` 函数：

```go
func NewInitCmd() *cobra.Command {
    initCmd := &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            // 1. 基础初始化（保持现有逻辑）
            ws, err := workspace.New()
            ws.InitWorkspace()
            config.SaveConfig(defaultConfig)
            initGitRepo(rickDir)

            // 2. 自动执行 plan → doing → learning 流程
            // 2.1 生成特殊的源码探索 task.md
            requirement := "深度探索项目源码结构和架构设计"
            if err := executePlanWorkflow(requirement); err != nil {
                return err
            }

            // 2.2 自动执行 doing job_0
            if err := executeDoingWorkflow("job_0"); err != nil {
                return err
            }

            // 2.3 自动执行 learning job_0
            if err := executeLearningWorkflow("job_0"); err != nil {
                return err
            }

            fmt.Println("Project initialized successfully with global context!")
            return nil
        },
    }
    return initCmd
}
```

#### 方案 B：独立命令（备选）

```bash
rick init                    # 仅创建目录、配置、Git
rick init --explore        # 进行深度探索并生成全局上下文
```

### 实现步骤

1. **理解现有流程**（已完成）
   - ✅ 理解 plan 命令如何生成 task.md
   - ✅ 理解 doing 命令如何执行任务
   - ✅ 理解 learning 命令如何生成全局上下文

2. **设计源码探索 task.md**
   - 创建特殊的 task.md 模板，用于深度源码探索
   - 支持跨越上下文窗口的多轮探索
   - 自动积累到 wiki/

3. **修改 init.go**
   - 添加 plan/doing/learning 的自动调用
   - 处理错误和边界情况
   - 添加进度提示

4. **测试与验证**
   - 使用 `rick_dev init` 测试
   - 验证生成的全局上下文质量
   - 对比规范要求

## 开发工作流

### 快速开始

```bash
# 1. 修改源码
vim /Users/sunquan/ai_coding/CODING/rick/internal/cmd/init.go

# 2. 重新构建 dev 版本
cd /Users/sunquan/ai_coding/CODING/rick
./scripts/install.sh --source --dev

# 3. 测试新功能
mkdir -p /tmp/rick_test
cd /tmp/rick_test
git init
rick_dev init

# 4. 验证结果
ls -la .rick/
cat .rick/OKR.md
cat .rick/SPEC.md
```

### 对比测试

```bash
# 使用生产版本 rick（旧实现）
rick init

# 使用开发版本 rick_dev（新实现）
rick_dev init

# 对比差异
diff -r .rick_prod/.rick .rick_dev/.rick
```

## 关键文件说明

### internal/cmd/init.go
- **当前功能**: 创建目录、配置、Git
- **需要修改**: 添加 plan/doing/learning 的自动调用
- **关键函数**:
  - `NewInitCmd()`: 命令定义
  - `executeDoingWorkflow()`: 在 doing.go 中定义，可复用
  - `executeLearningWorkflow()`: 在 learning.go 中定义，可复用

### internal/cmd/plan.go
- **关键函数**: `executePlanWorkflow(requirement string)`
- **用途**: 根据需求生成任务计划

### internal/cmd/doing.go
- **关键函数**: `executeDoingWorkflow(jobID string)`
- **用途**: 执行任务

### internal/cmd/learning.go
- **关键函数**: `executeLearningWorkflow(jobID string)`
- **用途**: 生成学习文档和全局上下文

## 下一步行动

1. ✅ 已创建 dev 版本
2. ✅ 已理解现有实现
3. 📋 设计优化方案（本文档）
4. 📋 实现 init 优化
5. 📋 测试与验证
6. 📋 合并到生产版本

## 相关文档

- [Rick 完整规范](file:///Users/sunquan/ai_coding/CODING/rick/Rick_Project_Complete_Description.md)
- [Init 优化详情](file:///Users/sunquan/.claude/projects/-Users-sunquan-ai-coding-CODING-rick/memory/init_optimization.md)
- [开发指南](file:///Users/sunquan/ai_coding/CODING/rick/DEVELOPMENT_GUIDE.md)
