# DAG Executor Module（DAG 执行模块）

## 概述
DAG Executor Module 负责构建任务依赖图（DAG）、拓扑排序、任务执行和失败重试机制。

## 模块位置
`internal/executor/`

## 核心功能

### 1. DAG 构建
**职责**: 根据 tasks.json 构建有向无环图（DAG）

**数据结构**:
```go
type DAG struct {
    Nodes map[string]*Node
    Edges map[string][]string
}

type Node struct {
    TaskID   string
    Task     *Task
    InDegree int
}
```

**核心函数**:
```go
// BuildDAG 构建 DAG
func BuildDAG(tasks []Task) (*DAG, error)
```

### 2. 拓扑排序
**算法**: Kahn 算法

**步骤**:
1. 计算每个节点的入度
2. 将入度为 0 的节点加入队列
3. 从队列中取出节点，将其邻居的入度 -1
4. 重复步骤 3，直到队列为空

**核心函数**:
```go
// TopologicalSort 拓扑排序
func TopologicalSort(dag *DAG) ([]string, error)
```

**实现**:
```go
func TopologicalSort(dag *DAG) ([]string, error) {
    var result []string
    queue := make([]string, 0)

    // 1. 找到所有入度为 0 的节点
    for taskID, node := range dag.Nodes {
        if node.InDegree == 0 {
            queue = append(queue, taskID)
        }
    }

    // 2. 拓扑排序
    for len(queue) > 0 {
        // 取出队首节点
        taskID := queue[0]
        queue = queue[1:]
        result = append(result, taskID)

        // 将邻居的入度 -1
        for _, neighbor := range dag.Edges[taskID] {
            dag.Nodes[neighbor].InDegree--
            if dag.Nodes[neighbor].InDegree == 0 {
                queue = append(queue, neighbor)
            }
        }
    }

    // 3. 检查是否有环
    if len(result) != len(dag.Nodes) {
        return nil, errors.New("DAG has cycle")
    }

    return result, nil
}
```

### 3. 任务执行
**执行模式**: 串行执行（按拓扑排序顺序）

**执行流程**:
```
对每个 task:
  1. 生成测试脚本
  2. 调用 Claude Code CLI 执行
  3. 运行测试脚本
  4. 通过 → git commit + 标记 done
  5. 失败 → 记录 debug.md + 重试
```

**核心函数**:
```go
// ExecuteTask 执行单个任务
func ExecuteTask(task *Task, retryCount int) error

// ExecuteDAG 执行整个 DAG
func ExecuteDAG(tasks []Task) error
```

**实现**:
```go
func ExecuteDAG(tasks []Task) error {
    // 1. 构建 DAG
    dag, err := BuildDAG(tasks)
    if err != nil {
        return err
    }

    // 2. 拓扑排序
    sortedTasks, err := TopologicalSort(dag)
    if err != nil {
        return err
    }

    // 3. 串行执行
    for _, taskID := range sortedTasks {
        task := dag.Nodes[taskID].Task
        err := ExecuteTaskWithRetry(task)
        if err != nil {
            return err
        }
    }

    return nil
}
```

### 4. 失败重试机制
**配置**: `MaxRetries`（默认 5 次）

**重试策略**:
- 每次失败记录到 debug.md
- 下一轮执行时加载 debug.md 作为上下文
- 超过限制后退出进程，需人工干预

**核心函数**:
```go
// ExecuteTaskWithRetry 执行任务（带重试）
func ExecuteTaskWithRetry(task *Task) error
```

**实现**:
```go
func ExecuteTaskWithRetry(task *Task) error {
    maxRetries := config.Get().MaxRetries

    for retryCount := 0; retryCount <= maxRetries; retryCount++ {
        // 执行任务
        err := ExecuteTask(task, retryCount)
        if err == nil {
            // 成功：git commit + 标记 done
            git.Commit(fmt.Sprintf("feat: 完成 %s", task.TaskName))
            task.StateInfo.Status = "done"
            return nil
        }

        // 失败：记录 debug.md
        RecordDebug(task, err, retryCount)

        if retryCount >= maxRetries {
            // 超过重试限制，退出
            log.Printf("[ERROR] 任务 %s 超过重试限制，需人工干预", task.TaskID)
            return fmt.Errorf("超过重试限制")
        }

        log.Printf("[WARN] 任务 %s 失败，重试 %d/%d", task.TaskID, retryCount+1, maxRetries)
    }

    return nil
}
```

### 5. 测试脚本生成
**职责**: 根据 task.md 的测试方法生成测试脚本

**核心函数**:
```go
// GenerateTestScript 生成测试脚本
func GenerateTestScript(task *Task) (string, error)
```

**示例**:
```bash
#!/bin/bash

# 测试步骤1：编译项目
go build -o bin/rick cmd/rick/main.go
if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi

# 测试步骤2：运行单元测试
go test ./internal/parser/
if [ $? -ne 0 ]; then
    echo "❌ 单元测试失败"
    exit 1
fi

echo "✅ 所有测试通过"
exit 0
```

## 状态管理

### 任务状态
- **pending**: 等待执行
- **doing**: 正在执行
- **done**: 执行完成
- **failed**: 执行失败

### 状态转换
```
pending → doing → done
              ↓
           failed (retry) → doing → done
                                 ↓
                              failed (超过限制)
```

## 测试

### 单元测试
```bash
go test ./internal/executor/
```

### 测试用例
```go
func TestBuildDAG(t *testing.T) {
    tasks := []Task{
        {TaskID: "task1", Dependencies: []string{}},
        {TaskID: "task2", Dependencies: []string{"task1"}},
    }

    dag, err := BuildDAG(tasks)
    if err != nil {
        t.Fatal(err)
    }

    if len(dag.Nodes) != 2 {
        t.Errorf("expected 2 nodes, got %d", len(dag.Nodes))
    }
}

func TestTopologicalSort(t *testing.T) {
    tasks := []Task{
        {TaskID: "task1", Dependencies: []string{}},
        {TaskID: "task2", Dependencies: []string{"task1"}},
    }

    dag, _ := BuildDAG(tasks)
    sorted, err := TopologicalSort(dag)
    if err != nil {
        t.Fatal(err)
    }

    if sorted[0] != "task1" || sorted[1] != "task2" {
        t.Errorf("unexpected order: %v", sorted)
    }
}
```

## 最佳实践

1. **DAG 验证**: 构建 DAG 后检查是否有环
2. **错误处理**: 详细记录每次执行的错误信息
3. **状态持久化**: 及时更新 tasks.json 的状态
4. **测试覆盖**: 确保测试脚本覆盖所有关键功能

## 常见问题

### Q1: 如何检测循环依赖？
**A**: 拓扑排序后，检查 `len(result) != len(dag.Nodes)`。

### Q2: 如何支持并行执行？
**A**: 按 DAG 的层级分组，同一层级的任务可并行执行。

### Q3: 如何处理长时间运行的任务？
**A**: 添加超时机制，超时后自动重试。

## 未来优化

1. **并行执行**: 支持同一层级任务并行执行
2. **超时机制**: 添加任务执行超时控制
3. **进度跟踪**: 实时显示执行进度
4. **断点续传**: 支持从上次失败的地方继续执行

---

*最后更新: 2026-03-14*
