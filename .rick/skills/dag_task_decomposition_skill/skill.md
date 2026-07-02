# skill:dag-task-decomposition（DAG 任务分解）

## 触发场景

plan 阶段将复杂需求分解为多个相互依赖的 task 时使用，特别是：
- 需要分析哪些 task 可以并行
- 需要设计汇聚点确保阶段性成果
- 需要避免循环依赖

## 预期效果

- 任务 DAG 无循环依赖，拓扑排序有效
- 并行 task 真正独立，无隐式冲突
- 每个 task 都能利用前序 task 的输出作为上下文

## 核心内容

### 三层 DAG 结构（推荐）

```
         基础层 (task1: 目录/索引/接口定义)
         /    \
    生产层A  生产层B  (并行，无依赖)
        \    /
       汇聚点 (确认基础完成后批量生产)
          |
     专题层 (串行，每层利用前层输出)
          |
       验证层 (最终 task，依赖所有前序)
```

### 扇出-扇入（Fan-out/Fan-in）

适合大量独立同类 task：
```
task1 (基础)
  ↓ 扇出
task2, task3, task4, task5 (并行，各写不同文件)
  ↓ 扇入
task6 (汇聚验证)
```

### 循环依赖检查

```python
# 简单检查：每个 task 的依赖不能包含依赖自己的 task
for task in tasks:
    for dep in task.dependencies:
        assert task.id not in tasks[dep].dependencies
```

### 并行化判断标准

✅ 可并行：
- 写不同文件
- 读相同文件但不写
- 独立的 API 调用

❌ 不能并行：
- 写同一文件
- B 依赖 A 的输出
- 有共享状态（如 tasks.json）

### 汇聚点设计原则

在阶段性成果完成后设汇聚 task：
- 汇聚 task 的 `dependencies` 包含所有前序 task
- 汇聚 task 负责验证前序产出的完整性
- 发现问题在此处暴露，不要让错误传播到下一阶段

### tasks.json DAG 示例

```json
{
  "tasks": [
    {"task_id": "task1", "dependencies": []},
    {"task_id": "task2", "dependencies": ["task1"]},
    {"task_id": "task3", "dependencies": ["task1"]},
    {"task_id": "task4", "dependencies": ["task2", "task3"]},
    {"task_id": "task5", "dependencies": ["task4"]}
  ]
}
```
