# Plan: dag_executor

## 模块概述

**模块职责**: 实现 DAG 构建、拓扑排序和任务执行引擎，支持串行执行、失败重试、问题记录

**对应 Research**:
- `.morty/research/使用_golang_开发_rick_命令行程序.md` - DAG 算法、执行流程、重试机制
- `.morty/research/RESEARCH_SUMMARY.md` - 执行流程、失败重试机制

**现有实现参考**: 无

**依赖模块**: infrastructure, parser

**被依赖模块**: prompt_manager, git_integration, cli_commands

## 接口定义

### 输入接口
- Task 列表（来自 parser 模块）
- 执行配置（MaxRetries, 超时时间等）
- Claude Code CLI 调用接口

### 输出接口
- 拓扑排序后的 tasks.json
- 执行日志和结果
- debug.md 更新记录

## 数据模型

### DAG 结构体
```go
type DAG struct {
    Tasks map[string]*Task
    Graph map[string][]string // task_id -> [dependent_ids]
}
```

### ExecutionConfig 结构体
```go
type ExecutionConfig struct {
    MaxRetries      int
    TimeoutSeconds  int
    LogFile         string
}
```

### ExecutionResult 结构体
```go
type ExecutionResult struct {
    TaskID    string
    Status    string // pending, running, success, failed, retrying
    Attempts  int
    Error     string
    Output    string
}
```

## Jobs

---

### Job 1: DAG 构建器

#### 目标

实现 DAG 构建功能，从 Task 列表构建有向无环图

#### 前置条件

- infrastructure:job_6 - 错误定义系统完成

#### Tasks

- [x] Task 1: 创建 internal/executor/dag.go，实现 DAG 结构体
- [x] Task 2: 实现 NewDAG(tasks) 函数，创建 DAG 实例
- [x] Task 3: 实现 AddTask(task) 方法，向 DAG 添加任务
- [x] Task 4: 实现 AddDependency(from, to) 方法，添加依赖关系
- [x] Task 5: 实现 ValidateDAG() 方法，验证 DAG 有效性
- [x] Task 6: 实现循环依赖检测逻辑
- [x] Task 7: 编写单元测试，覆盖各种 DAG 配置

#### 验证器

- NewDAG() 能正确创建 DAG 实例
- AddTask() 和 AddDependency() 能正确添加节点和边
- ValidateDAG() 能检测到循环依赖
- 循环依赖检测给出清晰的错误提示
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

✅ 完成 - 2026-03-14

**实现摘要**:
- 创建 internal/executor/dag.go，实现完整的 DAG 结构体
- 实现 NewDAG()、AddTask()、AddDependency() 等核心方法
- 实现循环依赖检测，使用 DFS 算法
- 编写 21 个单元测试，覆盖率 91.2%
- 所有验收标准满足

---

### Job 2: 拓扑排序实现

#### 目标

实现 Kahn 算法的拓扑排序，生成任务执行序列

#### 前置条件

- job_1 - DAG 构建器完成

#### Tasks

- [x] Task 1: 创建 internal/executor/topological.go，实现拓扑排序
- [x] Task 2: 实现 TopologicalSort(dag) 函数，使用 Kahn 算法
- [x] Task 3: 实现入度计算逻辑
- [x] Task 4: 实现队列处理逻辑
- [x] Task 5: 实现循环依赖检测（排序结果与任务数不符）
- [x] Task 6: 编写单元测试，覆盖各种 DAG 配置

#### 验证器

- TopologicalSort() 返回正确的排序序列
- 排序结果满足依赖关系
- 能检测到循环依赖并返回错误
- 排序结果数量等于任务总数
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

✅ 完成 - 2026-03-14

**实现摘要**:
- 创建 internal/executor/topological.go，实现 Kahn 算法拓扑排序
- 实现 TopologicalSort(dag) 函数，返回正确的排序序列
- 实现 calculateInDegrees(dag) 函数，计算任务入度
- 实现队列处理逻辑，按入度排序任务
- 实现循环依赖检测，当排序结果数量不等于任务数时返回错误
- 编写 10 个单元测试，覆盖各种 DAG 配置（线性、多重、独立、复杂、空、单一、钻石形等）
- 测试覆盖率 85.6%，超过 80% 要求
- 所有验收标准满足

---

### Job 3: Tasks.json 生成器

#### 目标

实现 tasks.json 的生成和加载，支持任务状态管理

#### 前置条件

- job_2 - 拓扑排序实现完成

#### Tasks

- [x] Task 1: 创建 internal/executor/tasks_json.go，实现 tasks.json 操作
- [x] Task 2: 实现 GenerateTasksJSON(dag, sortedTasks) 函数
- [x] Task 3: 实现 LoadTasksJSON(filePath) 函数
- [x] Task 4: 实现 SaveTasksJSON(filePath, tasks) 函数
- [x] Task 5: 实现 UpdateTaskStatus(taskID, status) 方法
- [x] Task 6: 实现 GetTaskStatus(taskID) 方法
- [x] Task 7: 编写单元测试，覆盖 JSON 生成、加载、更新

#### 验证器

- ✅ GenerateTasksJSON() 生成格式正确的 JSON
- ✅ LoadTasksJSON() 能正确加载 JSON 文件
- ✅ SaveTasksJSON() 能正确保存文件
- ✅ 状态更新正确反映在 JSON 中
- ✅ JSON 格式符合规范
- ✅ 单元测试覆盖率 >= 80%（实际 86.4%）

#### 调试日志

无

#### 完成状态

✅ 完成 - 2026-03-14

**实现摘要**:
- 创建 internal/executor/tasks_json.go，实现完整的 TaskState 和 TasksJSON 结构体
- 实现 GenerateTasksJSON(dag, sortedTasks) 函数，从 DAG 生成 JSON 结构
- 实现 LoadTasksJSON(filePath) 函数，从文件加载 JSON 数据
- 实现 SaveTasksJSON(filePath, tasks) 函数，将数据持久化到文件
- 实现 UpdateTaskStatus(taskID, status) 方法，支持状态验证
- 实现 GetTaskStatus(taskID) 方法，获取任务状态
- 添加辅助方法：UpdateTaskStatusWithError、UpdateTaskStatusWithOutput、IncrementAttempts
- 添加查询方法：GetTask、GetAllTasks、GetTasksByStatus 等 10+ 个辅助查询方法
- 编写 30 个单元测试，涵盖：
  - JSON 生成和验证
  - 文件 I/O 操作
  - 状态更新和验证
  - 错误处理
  - 数据持久化验证
  - 时间戳更新
  - 任务过滤和计数
- 测试覆盖率 86.4%，超过 80% 要求
- 所有验收标准满足

---

### Job 4: 任务执行器（核心）

#### 目标

实现单个任务的执行逻辑，支持 Claude Code CLI 调用和测试脚本执行

#### 前置条件

- job_3 - Tasks.json 生成器完成

#### Tasks

- [x] Task 1: 创建 internal/executor/runner.go，实现任务执行器
- [x] Task 2: 实现 RunTask(task, config) 函数
- [x] Task 3: 实现 GenerateTestScript(task) 函数，生成测试脚本
- [x] Task 4: 实现 ExecuteTestScript(scriptPath) 函数，执行测试脚本
- [x] Task 5: 实现 ParseTestResult(output) 函数，解析测试结果
- [x] Task 6: 实现超时控制逻辑
- [x] Task 7: 编写单元测试，覆盖任务执行流程

#### 验证器

- ✅ RunTask() 能正确执行任务
- ✅ GenerateTestScript() 生成有效的脚本
- ✅ ExecuteTestScript() 能正确执行脚本
- ✅ ParseTestResult() 能正确解析测试输出
- ✅ 超时控制正常工作
- ✅ 单元测试覆盖率 >= 80%（实际 85%）

#### 调试日志

无

#### 完成状态

✅ 完成 - 2026-03-14

**实现摘要**:
- 创建 internal/executor/runner.go，实现完整的任务执行器
- 实现 ExecutionConfig 结构体，支持 MaxRetries、TimeoutSeconds、LogFile 等配置
- 实现 TaskRunner 结构体和 NewTaskRunner() 工厂函数
- 实现 RunTask(task) 函数，完整的任务执行流程（生成脚本 -> 执行 -> 解析结果）
- 实现 GenerateTestScript(task) 函数，生成 shell 脚本，支持自动化测试
- 实现 ExecuteTestScript(scriptPath) 函数，执行脚本并支持超时控制（默认 30 秒）
- 实现 ParseTestResult(output) 函数，智能解析测试输出（查找 PASS/FAIL/ERROR 标记）
- 实现 TaskExecutionResult 结构体，记录任务执行结果和耗时
- 编写 28 个单元测试，覆盖：
  - TaskRunner 创建
  - 脚本生成（包括特殊字符、复杂测试方法、无测试方法等场景）
  - 脚本执行（包括超时、错误、stderr 等场景）
  - 结果解析（包括 PASS/FAIL/ERROR 标记、多行输出等）
  - 完整任务执行流程（包括依赖、关键结果等）
  - 执行结果字段验证
- 测试覆盖率 85%，超过 80% 要求
- 代码编译通过，所有验收标准满足

---

### Job 5: 重试机制实现

#### 目标

实现失败重试机制，支持可配置的重试次数和退避策略

#### 前置条件

- job_4 - 任务执行器完成

#### Tasks

- [x] Task 1: 创建 internal/executor/retry.go，实现重试逻辑
- [x] Task 2: 实现 RetryTask(task, config) 函数
- [x] Task 3: 实现重试循环逻辑（最多 MaxRetries 次）
- [x] Task 4: 实现每次重试时加载 debug.md 作为上下文
- [x] Task 5: 实现失败记录到 debug.md 的逻辑
- [x] Task 6: 实现超过重试限制时的退出逻辑
- [x] Task 7: 编写单元测试，覆盖重试流程

#### 验证器

- ✅ RetryTask() 能正确重试失败的任务
- ✅ 重试次数不超过 MaxRetries
- ✅ 每次重试都加载最新的 debug.md
- ✅ 失败信息正确追加到 debug.md
- ✅ 超过重试限制时返回错误
- ✅ 单元测试覆盖率 >= 80%（实际 83%）

#### 调试日志

无

#### 完成状态

✅ 完成 - 2026-03-14

**实现摘要**:
- 创建 internal/executor/retry.go，实现完整的重试机制
- 实现 RetryResult 结构体，记录重试执行结果和调试日志列表
- 实现 TaskRetryManager 结构体和 NewTaskRetryManager() 工厂函数
- 实现 RetryTask(task) 函数，核心重试逻辑：
  - 支持可配置的重试次数（默认5次）
  - 每次重试前加载 debug.md 作为上下文
  - 失败时自动生成调试日志条目
  - 超过重试限制时返回 max_retries_exceeded 状态
- 实现 loadDebugContext(debugFile) 函数，加载现有调试信息
- 实现 buildDebugEntry(task, attempt, maxRetries, result, context) 函数，生成调试日志条目：
  - 格式：`debug_N: [现象], [复现], [猜想], [验证], [修复], [进展]`
  - 自动提取错误信息作为现象
  - 自动生成猜想（基于错误类型分析）
  - 支持输出内容分析（FAIL/ERROR 标记）
- 实现 analyzeError(errMsg, output) 函数，智能错误分析：
  - 识别超时、文件不存在、权限、连接、脚本执行等常见错误
  - 基于输出内容进一步细化分析
- 实现 getNextDebugNumber(context) 函数，自动计算下一个调试编号
- 实现 appendToDebugFile(entry) 函数，追加调试条目到文件：
  - 自动创建嵌套目录结构
  - 支持文件不存在时创建
  - 正确处理换行符
- 实现 RetryTaskSimple(task, runner, config, debugFile) 便捷函数
- 编写 24 个单元测试，覆盖：
  - 成功执行（无重试）
  - 重试次数限制
  - 默认重试次数（5次）
  - 调试日志生成
  - 调试目录创建
  - 错误分析（超时、文件、权限、连接等）
  - 调试编号计算
  - 调试文件追加
  - 结果时间计算
  - nil 输入处理
  - 复杂错误消息分析
- 测试覆盖率 83%，超过 80% 要求
- 代码编译通过，所有验收标准满足

---

### Job 6: 执行协调器

#### 目标

实现执行协调器，支持串行执行所有任务，管理整个执行流程

#### 前置条件

- job_5 - 重试机制实现完成

#### Tasks

- [x] Task 1: 创建 internal/executor/executor.go，实现执行协调器
- [x] Task 2: 实现 ExecuteJob(jobID, config) 函数
- [x] Task 3: 实现任务串行执行逻辑（按拓扑排序顺序）
- [x] Task 4: 实现任务状态更新和持久化
- [x] Task 5: 实现执行日志记录
- [x] Task 6: 实现错误处理和恢复逻辑
- [x] Task 7: 编写单元测试，覆盖完整执行流程

#### 验证器

- ✅ ExecuteJob() 能正确执行所有任务
- ✅ 任务按拓扑排序顺序执行
- ✅ 任务状态正确更新
- ✅ 执行日志正确记录
- ✅ 错误处理机制正常工作
- ✅ 单元测试覆盖率 >= 80%（实际 80.5%）

#### 调试日志

无

#### 完成状态

✅ 完成 - 2026-03-14

**实现摘要**:
- 创建 internal/executor/executor.go，实现完整的执行协调器
- 实现 ExecutionJobResult 结构体，记录执行结果和统计信息
- 实现 Executor 结构体和 NewExecutor() 工厂函数
- 实现 ExecuteJob() 函数，核心功能：
  - 初始化 DAG、拓扑排序、tasks.json
  - 按拓扑排序顺序串行执行所有任务
  - 支持任务状态管理（pending -> running -> success/failed）
  - 支持 tasks.json 持久化
  - 实现完整的执行日志记录
  - 生成执行结果和错误摘要
- 实现任务状态管理：
  - 执行前更新为 "running"
  - 执行成功更新为 "success"
  - 执行失败更新为 "failed"
  - 每次状态变化后持久化到 tasks.json
- 实现执行日志系统：
  - logf() 方法记录带时间戳的日志
  - getExecutionLog() 返回完整日志
  - SaveExecutionLog() 保存日志到文件
- 实现错误处理：
  - 处理任务不存在错误
  - 处理 tasks.json 保存失败
  - 处理重试管理器执行失败
  - 生成错误摘要 (ErrorSummary)
  - 正确传播错误信息
- 实现辅助方法：
  - GetTasksJSON()、GetDAG()、GetSortedTaskIDs()
  - generateErrorSummary()、SaveExecutionLog()
- 编写 15 个新的单元测试，覆盖：
  - 单个任务执行
  - 多个任务执行
  - 任务依赖关系
  - tasks.json 持久化
  - 执行日志记录
  - 时间戳管理
  - 错误处理
  - 状态管理
  - 计数器管理
  - 错误摘要生成
- 测试覆盖率 80.5%，超过 80% 要求
- 所有 141 个单元测试通过
- 代码编译成功，无错误

---

### Job 7: 集成测试

#### 目标

验证 dag_executor 模块所有组件协同工作正确，能正确执行完整的任务流程

#### 前置条件

- job_1 - DAG 构建器完成
- job_2 - 拓扑排序实现完成
- job_3 - Tasks.json 生成器完成
- job_4 - 任务执行器完成
- job_5 - 重试机制实现完成
- job_6 - 执行协调器完成

#### Tasks

- [ ] Task 1: 创建测试 job 目录和测试 task 文件
- [ ] Task 2: 验证 DAG 构建正确
- [ ] Task 3: 验证拓扑排序正确
- [ ] Task 4: 验证 tasks.json 生成正确
- [ ] Task 5: 验证任务执行流程正确
- [ ] Task 6: 验证重试机制正常工作
- [ ] Task 7: 编写集成测试脚本，覆盖完整执行流程

#### 验证器

- DAG 构建和拓扑排序正确
- tasks.json 生成格式正确
- 所有任务按顺序执行
- 任务状态正确更新
- 重试机制正常工作
- 集成测试脚本通过

#### 调试日志

无

#### 完成状态

⏳ 待开始

