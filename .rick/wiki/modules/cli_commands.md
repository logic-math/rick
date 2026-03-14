# CLI Commands Module（命令处理模块）

## 概述
CLI Commands Module 负责处理 Rick CLI 的三个核心命令：plan、doing、learning。

## 模块位置
`internal/cmd/`

## 核心命令

### 1. plan 命令
**文件**: `internal/cmd/plan.go`

**功能**: 规划任务，将大目标分解为小任务

**使用方法**:
```bash
rick plan "任务描述"
```

**执行流程**:
```
1. 检查 .rick/ 目录是否存在
   ├─ 不存在 → 创建 .rick/ 目录结构
   └─ 存在 → 继续

2. 生成 job_id（如 job_1）

3. 创建 .rick/jobs/job_n/plan/ 目录

4. 构建 Plan 阶段提示词
   ├─ 加载 plan.md 模板
   ├─ 注入项目背景（OKR.md, SPEC.md）
   └─ 注入任务目标

5. 调用 Claude Code CLI
   └─ 生成 tasks/*.md 文件

6. 解析 tasks/*.md
   └─ 生成 tasks.json

7. 输出结果
   └─ 显示任务列表和依赖关系
```

**输出结构**:
```
.rick/jobs/job_1/plan/
├── tasks/
│   ├── task1.md
│   ├── task2.md
│   └── task3.md
└── tasks.json
```

**核心函数**:
```go
// RunPlan 执行 plan 命令
func RunPlan(objective string) error {
    // 1. 确保工作空间存在
    workspace.EnsureWorkspace()

    // 2. 创建 job 目录
    jobID := generateJobID()
    jobDir, _ := workspace.CreateJobDir(jobID, "plan")

    // 3. 构建提示词
    context := PromptContext{
        ProjectName: readProjectName(),
        ProjectDesc: readProjectDesc(),
        Objectives:  objective,
    }
    prompt, _ := prompt.BuildPlanPrompt(context)

    // 4. 调用 Claude Code CLI
    callcli.CallClaudeCLI(prompt)

    // 5. 解析 tasks/*.md，生成 tasks.json
    tasks := parseTaskFiles(jobDir)
    workspace.SaveTasksJSON(jobDir, tasks)

    return nil
}
```

### 2. doing 命令
**文件**: `internal/cmd/doing.go`

**功能**: 执行任务，完成具体的编码工作

**使用方法**:
```bash
rick doing job_1
```

**执行流程**:
```
1. 检查项目根目录 .git/ 是否存在
   ├─ 不存在 → 自动初始化 Git
   └─ 存在 → 继续

2. 创建 .rick/jobs/job_n/doing/ 目录

3. 加载 tasks.json

4. 构建 DAG（拓扑排序）

5. 对每个 task：
   a. 生成测试脚本
   b. 构建 Doing 阶段提示词
      ├─ 加载 doing.md 模板
      ├─ 注入任务信息
      ├─ 注入项目背景
      ├─ 注入执行上下文（已完成任务、依赖）
      └─ 注入问题记录（如果重试）
   c. 调用 Claude Code CLI 执行
   d. 运行测试脚本
   e. 通过 → git commit + 标记 done
   f. 失败 → 记录 debug.md + 重试

6. 超过重试限制 → 退出，人工干预
```

**输出结构**:
```
.rick/jobs/job_1/doing/
├── tasks.json (updated)
├── debug.md (if failed)
└── test_scripts/
    ├── test_task1.sh
    └── test_task2.sh
```

**核心函数**:
```go
// RunDoing 执行 doing 命令
func RunDoing(jobID string) error {
    // 1. 自动初始化 Git（如果需要）
    git.EnsureGitInitialized(".")

    // 2. 创建 doing 目录
    jobDir, _ := workspace.CreateJobDir(jobID, "doing")

    // 3. 加载 tasks.json
    tasks, _ := workspace.LoadTasksJSON(jobDir)

    // 4. 执行 DAG
    err := executor.ExecuteDAG(tasks)
    if err != nil {
        return err
    }

    return nil
}
```

### 3. learning 命令
**文件**: `internal/cmd/learning.go`

**功能**: 知识沉淀，总结项目经验

**使用方法**:
```bash
rick learning job_1
```

**执行流程**:
```
1. 创建 .rick/jobs/job_n/learning/ 目录

2. 构建 Learning 阶段提示词
   ├─ 加载 learning.md 模板
   ├─ 注入 job 信息（任务数量、完成情况）
   ├─ 注入项目背景
   └─ 注入执行记录（debug.md, git log）

3. 调用 Claude Code CLI
   └─ 生成 summary.md, insights.md

4. 人工审核和编辑

5. 更新 .rick/knowledge/ 知识库
   ├─ 提取设计模式 → patterns/
   ├─ 提取最佳实践 → best_practices/
   └─ 提取经验教训 → lessons_learned/
```

**输出结构**:
```
.rick/jobs/job_1/learning/
├── summary.md       # 项目总结
├── insights.md      # 关键洞察
└── knowledge/       # 知识提取
    ├── pattern1.md
    └── best_practice1.md
```

**核心函数**:
```go
// RunLearning 执行 learning 命令
func RunLearning(jobID string) error {
    // 1. 创建 learning 目录
    jobDir, _ := workspace.CreateJobDir(jobID, "learning")

    // 2. 构建提示词
    context := PromptContext{
        JobID:      jobID,
        TaskCount:  getTaskCount(jobID),
        DebugInfo:  readAllDebugInfo(jobID),
        GitLog:     git.GetLog(jobID),
    }
    prompt, _ := prompt.BuildLearningPrompt(context)

    // 3. 调用 Claude Code CLI
    callcli.CallClaudeCLI(prompt)

    // 4. 提示人工审核
    log.Println("[INFO] 请审核 learning/ 目录的内容")
    log.Println("[INFO] 审核完成后，运行 rick knowledge update 更新知识库")

    return nil
}
```

## 命令行参数

### plan 命令
```bash
rick plan [flags] "任务描述"

Flags:
  -h, --help   帮助信息
```

### doing 命令
```bash
rick doing [flags] <job_id>

Flags:
  -h, --help   帮助信息
```

### learning 命令
```bash
rick learning [flags] <job_id>

Flags:
  -h, --help   帮助信息
```

## 错误处理

### plan 命令错误
- 工作空间创建失败
- 提示词构建失败
- Claude Code CLI 调用失败
- tasks.json 生成失败

### doing 命令错误
- Git 初始化失败
- tasks.json 加载失败
- DAG 构建失败（循环依赖）
- 任务执行失败（超过重试限制）

### learning 命令错误
- learning 目录创建失败
- 提示词构建失败
- Claude Code CLI 调用失败

## 测试

### 单元测试
```bash
go test ./internal/cmd/
```

### 集成测试
```bash
# 测试完整流程
rick plan "测试任务"
rick doing job_1
rick learning job_1
```

## 最佳实践

1. **错误处理**: 详细记录每个步骤的错误信息
2. **用户提示**: 在关键步骤提供清晰的用户提示
3. **状态检查**: 执行前检查前置条件（如 Git 初始化）
4. **日志记录**: 记录每个命令的执行日志

## 常见问题

### Q1: 如何查看 job 列表？
**A**: `ls .rick/jobs/`

### Q2: 如何重新执行失败的任务？
**A**: 修改 task.md 后，重新运行 `rick doing job_n`。

### Q3: 如何跳过某个任务？
**A**: 在 tasks.json 中将任务状态设为 "done"。

## 未来优化

1. **交互式模式**: 支持交互式选择任务
2. **并行执行**: 支持 `rick doing job_1 --parallel`
3. **断点续传**: 支持从上次失败的地方继续
4. **进度显示**: 实时显示执行进度

---

*最后更新: 2026-03-14*
