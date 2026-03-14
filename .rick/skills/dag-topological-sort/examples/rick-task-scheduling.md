# 案例: Rick CLI 任务调度系统

## 背景

Rick CLI 使用 DAG 拓扑排序来调度任务执行顺序。任务之间可能存在依赖关系，例如：
- task3 依赖 task1（必须先完成 task1）
- task4 依赖 task2
- task5 依赖 task3 和 task4

## 任务依赖图

```
task1 ──→ task3 ──→ task5
              ↗
task2 ──→ task4
```

## 输入数据

### tasks.json
```json
{
  "tasks": [
    {
      "task_id": "task1",
      "task_name": "分析项目架构",
      "dependencies": []
    },
    {
      "task_id": "task2",
      "task_name": "分析代码规范",
      "dependencies": []
    },
    {
      "task_id": "task3",
      "task_name": "创建 Wiki 索引",
      "dependencies": ["task1"]
    },
    {
      "task_id": "task4",
      "task_name": "完善模块文档",
      "dependencies": ["task2"]
    },
    {
      "task_id": "task5",
      "task_name": "提取技能库",
      "dependencies": ["task3", "task4"]
    }
  ]
}
```

## 执行流程

### 1. 构建 DAG
```go
// 从 tasks.json 加载任务
tasks, err := LoadTasksFromJSON("tasks.json")
if err != nil {
    return err
}

// 构建 DAG
dag := executor.NewDAG()
for _, task := range tasks {
    dag.AddTask(task)
}
```

### 2. 拓扑排序
```go
// 执行拓扑排序
order, err := executor.TopologicalSort(dag)
if err != nil {
    return fmt.Errorf("failed to sort tasks: %w", err)
}

// 输出: ["task1", "task2", "task3", "task4", "task5"]
fmt.Printf("Execution order: %v\n", order)
```

### 3. 串行执行
```go
// 按拓扑排序顺序串行执行任务
for _, taskID := range order {
    task := dag.GetTask(taskID)

    fmt.Printf("Executing %s: %s\n", task.ID, task.Name)

    // 执行任务
    result, err := executeTask(task)
    if err != nil {
        return fmt.Errorf("task %s failed: %w", taskID, err)
    }

    // Git 提交
    if err := gitCommit(task); err != nil {
        return fmt.Errorf("git commit failed: %w", err)
    }

    fmt.Printf("✓ %s completed\n", taskID)
}
```

## 执行输出

```
Execution order: [task1 task2 task3 task4 task5]

Executing task1: 分析项目架构
✓ task1 completed
[git commit] feat(okr): 分析项目架构并生成 OKR.md

Executing task2: 分析代码规范
✓ task2 completed
[git commit] feat(spec): 分析代码规范并生成 SPEC.md

Executing task3: 创建 Wiki 索引
✓ task3 completed
[git commit] feat(wiki): 创建 Wiki 知识库索引和架构文档

Executing task4: 完善模块文档
✓ task4 completed
[git commit] docs(wiki): 完善 Wiki 模块文档

Executing task5: 提取技能库
✓ task5 completed
[git commit] feat(skills): 提取可复用技能并创建 Skills 库
```

## 入度计算过程

### 初始状态
```
task1: 入度 = 0 (无依赖)
task2: 入度 = 0 (无依赖)
task3: 入度 = 1 (依赖 task1)
task4: 入度 = 1 (依赖 task2)
task5: 入度 = 2 (依赖 task3, task4)
```

### 执行步骤

**Step 1**: 队列 = [task1, task2]（入度为 0）
- 取出 task1，输出到结果
- task3 的入度减 1，变为 0，加入队列
- 队列 = [task2, task3]

**Step 2**: 队列 = [task2, task3]
- 取出 task2，输出到结果
- task4 的入度减 1，变为 0，加入队列
- 队列 = [task3, task4]

**Step 3**: 队列 = [task3, task4]
- 取出 task3，输出到结果
- task5 的入度减 1，变为 1
- 队列 = [task4]

**Step 4**: 队列 = [task4]
- 取出 task4，输出到结果
- task5 的入度减 1，变为 0，加入队列
- 队列 = [task5]

**Step 5**: 队列 = [task5]
- 取出 task5，输出到结果
- 队列 = []

**结果**: [task1, task2, task3, task4, task5]

## 循环依赖检测

### 错误案例
如果不小心添加了循环依赖：
```json
{
  "tasks": [
    {
      "task_id": "task1",
      "dependencies": ["task3"]
    },
    {
      "task_id": "task3",
      "dependencies": ["task1"]
    }
  ]
}
```

### 检测结果
```
Error: cycle detected in DAG: tasks [task1, task3] form a cycle
```

### 检测原理
1. 拓扑排序完成后，检查输出的任务数
2. 如果输出任务数 < 总任务数，说明有任务未被处理
3. 未被处理的任务形成循环依赖

## 性能分析

### 任务规模
- 任务数: 5
- 依赖关系数: 4
- 时间复杂度: O(5 + 4) = O(9)

### 实际耗时
```
拓扑排序: < 1ms
任务执行: 约 20 分钟（包括 Claude Code 调用）
总计: 约 20 分钟
```

## 优化建议

### 1. 并行执行
当前实现是串行执行，可以优化为层级并行：
```
Layer 0: [task1, task2]  # 并行执行
Layer 1: [task3, task4]  # 并行执行
Layer 2: [task5]         # 单独执行
```

### 2. 增量执行
如果某个任务失败，可以从失败的任务开始重新执行：
```go
// 跳过已完成的任务
for _, taskID := range order {
    if isTaskCompleted(taskID) {
        continue
    }
    executeTask(taskID)
}
```

### 3. 动态依赖
支持在运行时动态添加依赖关系：
```go
// 根据执行结果动态调整依赖
if result.NeedExtraValidation {
    dag.AddTask(validationTask)
    dag.AddDependency(validationTask.ID, task.ID)
}
```

## 经验总结

### 成功要素
1. ✅ 清晰的依赖关系定义（tasks.json）
2. ✅ 自动循环检测（避免死锁）
3. ✅ 每个任务完成后自动提交（防止丢失）
4. ✅ 失败重试机制（提高成功率）

### 改进空间
1. 🔄 支持并行执行（提高效率）
2. 🔄 支持任务优先级（灵活调度）
3. 🔄 支持条件依赖（动态调整）
4. 🔄 可视化依赖图（便于理解）

---

*案例来源: Rick CLI job_0*
*最后更新: 2026-03-14*
