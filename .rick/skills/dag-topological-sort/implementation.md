# DAG 拓扑排序 - 实现细节

## 核心数据结构

### DAG 结构
```go
type DAG struct {
    Tasks map[string]*Task  // 所有任务节点
}

type Task struct {
    ID           string
    Name         string
    Dependencies []string  // 依赖的任务 ID 列表
}
```

### 入度表
```go
// 计算每个任务的入度（依赖数量）
inDegrees := make(map[string]int)
for taskID := range dag.Tasks {
    inDegrees[taskID] = 0
}
for taskID, task := range dag.Tasks {
    for range task.Dependencies {
        inDegrees[taskID]++
    }
}
```

## Kahn 算法实现

### 完整代码
```go
package executor

import "fmt"

// TopologicalSort performs a topological sort on the DAG using Kahn's algorithm
// Returns a slice of task IDs in topological order
// Returns an error if a cycle is detected in the DAG
func TopologicalSort(dag *DAG) ([]string, error) {
    if dag == nil {
        return nil, fmt.Errorf("DAG cannot be nil")
    }

    // 步骤1: 计算所有任务的入度
    inDegrees := calculateInDegrees(dag)

    // 步骤2: 将入度为 0 的任务加入队列
    queue := make([]string, 0)
    for taskID, degree := range inDegrees {
        if degree == 0 {
            queue = append(queue, taskID)
        }
    }

    // 步骤3: 处理队列中的任务
    result := make([]string, 0, len(dag.Tasks))
    for len(queue) > 0 {
        // 从队列中取出一个入度为 0 的任务
        current := queue[0]
        queue = queue[1:]
        result = append(result, current)

        // 步骤4: 获取当前任务的所有依赖者（反向依赖）
        dependents, err := dag.GetTaskDependents(current)
        if err != nil {
            return nil, err
        }

        // 步骤5: 将依赖者的入度减 1
        for _, dependent := range dependents {
            inDegrees[dependent]--

            // 如果入度变为 0，加入队列
            if inDegrees[dependent] == 0 {
                queue = append(queue, dependent)
            }
        }
    }

    // 步骤6: 检查是否所有任务都被处理（循环检测）
    if len(result) != len(dag.Tasks) {
        // 找出未处理的任务（循环依赖的任务）
        processedSet := make(map[string]bool)
        for _, taskID := range result {
            processedSet[taskID] = true
        }

        var cycledTasks []string
        for taskID := range dag.Tasks {
            if !processedSet[taskID] {
                cycledTasks = append(cycledTasks, taskID)
            }
        }

        return nil, fmt.Errorf("cycle detected in DAG: tasks %v form a cycle", cycledTasks)
    }

    return result, nil
}

// calculateInDegrees calculates the in-degree for each task in the DAG
// In-degree is the number of tasks that a task depends on
func calculateInDegrees(dag *DAG) map[string]int {
    inDegrees := make(map[string]int)

    // 初始化所有任务的入度为 0
    for taskID := range dag.Tasks {
        inDegrees[taskID] = 0
    }

    // 遍历所有任务，统计每个任务的依赖数量
    for taskID, task := range dag.Tasks {
        for range task.Dependencies {
            inDegrees[taskID]++
        }
    }

    return inDegrees
}
```

## 关键实现细节

### 1. 入度计算
```go
// 入度 = 任务的依赖数量
// 例如: task3 依赖 task1 和 task2，则 task3 的入度为 2
func calculateInDegrees(dag *DAG) map[string]int {
    inDegrees := make(map[string]int)
    for taskID := range dag.Tasks {
        inDegrees[taskID] = 0  // 初始化为 0
    }
    for taskID, task := range dag.Tasks {
        inDegrees[taskID] = len(task.Dependencies)  // 依赖数量
    }
    return inDegrees
}
```

### 2. 队列管理
```go
// 使用切片作为队列（FIFO）
queue := make([]string, 0)

// 入队
queue = append(queue, taskID)

// 出队
current := queue[0]
queue = queue[1:]
```

### 3. 循环检测
```go
// 如果输出的任务数 < 总任务数，说明存在循环
if len(result) != len(dag.Tasks) {
    // 找出未处理的任务
    processedSet := make(map[string]bool)
    for _, taskID := range result {
        processedSet[taskID] = true
    }

    var cycledTasks []string
    for taskID := range dag.Tasks {
        if !processedSet[taskID] {
            cycledTasks = append(cycledTasks, taskID)
        }
    }

    return nil, fmt.Errorf("cycle detected: %v", cycledTasks)
}
```

## 时间和空间复杂度

### 时间复杂度: O(V + E)
- **V**: 顶点数（任务数）
- **E**: 边数（依赖关系数）
- **分析**:
  - 计算入度: O(V + E)
  - 初始化队列: O(V)
  - 处理所有任务: O(V)
  - 处理所有依赖关系: O(E)
  - 总计: O(V + E)

### 空间复杂度: O(V)
- 入度表: O(V)
- 队列: O(V)
- 结果数组: O(V)
- 总计: O(V)

## 最佳实践

### 1. 错误处理
```go
// 检查 DAG 是否为 nil
if dag == nil {
    return nil, fmt.Errorf("DAG cannot be nil")
}

// 检查循环依赖
if len(result) != len(dag.Tasks) {
    return nil, fmt.Errorf("cycle detected")
}
```

### 2. 性能优化
```go
// 预分配结果数组容量
result := make([]string, 0, len(dag.Tasks))

// 使用 map 快速查找
processedSet := make(map[string]bool)
```

### 3. 可读性优化
```go
// 使用有意义的变量名
current := queue[0]  // 当前处理的任务
dependents := dag.GetTaskDependents(current)  // 依赖当前任务的任务

// 添加注释说明关键步骤
// 步骤1: 计算入度
// 步骤2: 初始化队列
// 步骤3: 处理队列
```

## 测试用例

### 测试1: 简单依赖
```go
func TestTopologicalSort_Simple(t *testing.T) {
    dag := &DAG{
        Tasks: map[string]*Task{
            "task1": {ID: "task1", Dependencies: []string{}},
            "task2": {ID: "task2", Dependencies: []string{"task1"}},
            "task3": {ID: "task3", Dependencies: []string{"task2"}},
        },
    }

    result, err := TopologicalSort(dag)
    assert.NoError(t, err)
    assert.Equal(t, []string{"task1", "task2", "task3"}, result)
}
```

### 测试2: 并行依赖
```go
func TestTopologicalSort_Parallel(t *testing.T) {
    dag := &DAG{
        Tasks: map[string]*Task{
            "task1": {ID: "task1", Dependencies: []string{}},
            "task2": {ID: "task2", Dependencies: []string{}},
            "task3": {ID: "task3", Dependencies: []string{"task1", "task2"}},
        },
    }

    result, err := TopologicalSort(dag)
    assert.NoError(t, err)
    // task1 和 task2 可以任意顺序，但都必须在 task3 之前
    assert.Contains(t, result[:2], "task1")
    assert.Contains(t, result[:2], "task2")
    assert.Equal(t, "task3", result[2])
}
```

### 测试3: 循环依赖
```go
func TestTopologicalSort_Cycle(t *testing.T) {
    dag := &DAG{
        Tasks: map[string]*Task{
            "task1": {ID: "task1", Dependencies: []string{"task2"}},
            "task2": {ID: "task2", Dependencies: []string{"task1"}},
        },
    }

    result, err := TopologicalSort(dag)
    assert.Error(t, err)
    assert.Nil(t, result)
    assert.Contains(t, err.Error(), "cycle detected")
}
```

## 扩展: 并行执行调度

### 层级拓扑排序
```go
// GetExecutionLayers 返回可以并行执行的任务层级
func GetExecutionLayers(dag *DAG) ([][]string, error) {
    inDegrees := calculateInDegrees(dag)
    layers := [][]string{}

    for len(inDegrees) > 0 {
        // 找出当前入度为 0 的所有任务（可以并行执行）
        layer := []string{}
        for taskID, degree := range inDegrees {
            if degree == 0 {
                layer = append(layer, taskID)
            }
        }

        if len(layer) == 0 {
            return nil, fmt.Errorf("cycle detected")
        }

        layers = append(layers, layer)

        // 更新入度
        for _, taskID := range layer {
            delete(inDegrees, taskID)
            dependents, _ := dag.GetTaskDependents(taskID)
            for _, dep := range dependents {
                inDegrees[dep]--
            }
        }
    }

    return layers, nil
}
```

### 使用示例
```go
layers, err := GetExecutionLayers(dag)
if err != nil {
    return err
}

// 并行执行每一层的任务
for i, layer := range layers {
    fmt.Printf("Layer %d: %v\n", i, layer)
    // 使用 goroutine 并行执行 layer 中的所有任务
    var wg sync.WaitGroup
    for _, taskID := range layer {
        wg.Add(1)
        go func(id string) {
            defer wg.Done()
            executeTask(id)
        }(taskID)
    }
    wg.Wait()
}
```

## 常见问题

### Q1: 如何处理多个合法的拓扑排序结果？
**A**: Kahn 算法不保证唯一性。如果需要特定顺序，可以在入度为 0 的任务中按优先级排序。

### Q2: 如何优化大规模 DAG 的性能？
**A**:
1. 使用并发计算入度
2. 使用优先级队列替代普通队列
3. 缓存依赖关系查询结果

### Q3: 如何可视化 DAG？
**A**: 可以使用 Graphviz 生成 DOT 格式的图：
```go
func ExportToDot(dag *DAG) string {
    var buf bytes.Buffer
    buf.WriteString("digraph G {\n")
    for _, task := range dag.Tasks {
        for _, dep := range task.Dependencies {
            buf.WriteString(fmt.Sprintf("  %s -> %s;\n", dep, task.ID))
        }
    }
    buf.WriteString("}\n")
    return buf.String()
}
```

---

*参考: Rick CLI `internal/executor/topological.go`*
*最后更新: 2026-03-14*
