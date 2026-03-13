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

- [ ] Task 1: 创建 internal/executor/dag.go，实现 DAG 结构体
- [ ] Task 2: 实现 NewDAG(tasks) 函数，创建 DAG 实例
- [ ] Task 3: 实现 AddTask(task) 方法，向 DAG 添加任务
- [ ] Task 4: 实现 AddDependency(from, to) 方法，添加依赖关系
- [ ] Task 5: 实现 ValidateDAG() 方法，验证 DAG 有效性
- [ ] Task 6: 实现循环依赖检测逻辑
- [ ] Task 7: 编写单元测试，覆盖各种 DAG 配置

#### 验证器

- NewDAG() 能正确创建 DAG 实例
- AddTask() 和 AddDependency() 能正确添加节点和边
- ValidateDAG() 能检测到循环依赖
- 循环依赖检测给出清晰的错误提示
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 2: 拓扑排序实现

#### 目标

实现 Kahn 算法的拓扑排序，生成任务执行序列

#### 前置条件

- job_1 - DAG 构建器完成

#### Tasks

- [ ] Task 1: 创建 internal/executor/topological.go，实现拓扑排序
- [ ] Task 2: 实现 TopologicalSort(dag) 函数，使用 Kahn 算法
- [ ] Task 3: 实现入度计算逻辑
- [ ] Task 4: 实现队列处理逻辑
- [ ] Task 5: 实现循环依赖检测（排序结果与任务数不符）
- [ ] Task 6: 编写单元测试，覆盖各种 DAG 配置

#### 验证器

- TopologicalSort() 返回正确的排序序列
- 排序结果满足依赖关系
- 能检测到循环依赖并返回错误
- 排序结果数量等于任务总数
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 3: Tasks.json 生成器

#### 目标

实现 tasks.json 的生成和加载，支持任务状态管理

#### 前置条件

- job_2 - 拓扑排序实现完成

#### Tasks

- [ ] Task 1: 创建 internal/executor/tasks_json.go，实现 tasks.json 操作
- [ ] Task 2: 实现 GenerateTasksJSON(dag, sortedTasks) 函数
- [ ] Task 3: 实现 LoadTasksJSON(filePath) 函数
- [ ] Task 4: 实现 SaveTasksJSON(filePath, tasks) 函数
- [ ] Task 5: 实现 UpdateTaskStatus(taskID, status) 方法
- [ ] Task 6: 实现 GetTaskStatus(taskID) 方法
- [ ] Task 7: 编写单元测试，覆盖 JSON 生成、加载、更新

#### 验证器

- GenerateTasksJSON() 生成格式正确的 JSON
- LoadTasksJSON() 能正确加载 JSON 文件
- SaveTasksJSON() 能正确保存文件
- 状态更新正确反映在 JSON 中
- JSON 格式符合规范
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 4: 任务执行器（核心）

#### 目标

实现单个任务的执行逻辑，支持 Claude Code CLI 调用和测试脚本执行

#### 前置条件

- job_3 - Tasks.json 生成器完成

#### Tasks

- [ ] Task 1: 创建 internal/executor/runner.go，实现任务执行器
- [ ] Task 2: 实现 RunTask(task, config) 函数
- [ ] Task 3: 实现 GenerateTestScript(task) 函数，生成测试脚本
- [ ] Task 4: 实现 ExecuteTestScript(scriptPath) 函数，执行测试脚本
- [ ] Task 5: 实现 ParseTestResult(output) 函数，解析测试结果
- [ ] Task 6: 实现超时控制逻辑
- [ ] Task 7: 编写单元测试，覆盖任务执行流程

#### 验证器

- RunTask() 能正确执行任务
- GenerateTestScript() 生成有效的脚本
- ExecuteTestScript() 能正确执行脚本
- ParseTestResult() 能正确解析测试输出
- 超时控制正常工作
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 5: 重试机制实现

#### 目标

实现失败重试机制，支持可配置的重试次数和退避策略

#### 前置条件

- job_4 - 任务执行器完成

#### Tasks

- [ ] Task 1: 创建 internal/executor/retry.go，实现重试逻辑
- [ ] Task 2: 实现 RetryTask(task, config) 函数
- [ ] Task 3: 实现重试循环逻辑（最多 MaxRetries 次）
- [ ] Task 4: 实现每次重试时加载 debug.md 作为上下文
- [ ] Task 5: 实现失败记录到 debug.md 的逻辑
- [ ] Task 6: 实现超过重试限制时的退出逻辑
- [ ] Task 7: 编写单元测试，覆盖重试流程

#### 验证器

- RetryTask() 能正确重试失败的任务
- 重试次数不超过 MaxRetries
- 每次重试都加载最新的 debug.md
- 失败信息正确追加到 debug.md
- 超过重试限制时返回错误
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 6: 执行协调器

#### 目标

实现执行协调器，支持串行执行所有任务，管理整个执行流程

#### 前置条件

- job_5 - 重试机制实现完成

#### Tasks

- [ ] Task 1: 创建 internal/executor/executor.go，实现执行协调器
- [ ] Task 2: 实现 ExecuteJob(jobID, config) 函数
- [ ] Task 3: 实现任务串行执行逻辑（按拓扑排序顺序）
- [ ] Task 4: 实现任务状态更新和持久化
- [ ] Task 5: 实现执行日志记录
- [ ] Task 6: 实现错误处理和恢复逻辑
- [ ] Task 7: 编写单元测试，覆盖完整执行流程

#### 验证器

- ExecuteJob() 能正确执行所有任务
- 任务按拓扑排序顺序执行
- 任务状态正确更新
- 执行日志正确记录
- 错误处理机制正常工作
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

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

