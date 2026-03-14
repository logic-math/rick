# Tutorial 1: 使用 Rick 管理简单项目

> 通过完整示例学习如何使用 Rick CLI 管理一个简单的 Go 项目

## 📋 目标

在本教程中，你将学习：
- 如何规划和分解任务
- 如何执行任务并处理失败
- 如何积累和复用知识
- 如何使用 Git 管理版本

## 🎯 项目描述

我们将创建一个简单的 **Todo List API**，包含以下功能：
- RESTful API（GET, POST, PUT, DELETE）
- 内存存储（map 数据结构）
- JSON 序列化
- 单元测试

## 📝 前置要求

- 已安装 Rick CLI（参考 [快速入门](../getting-started.md)）
- 已安装 Go 1.21+
- 已安装 Claude Code CLI

---

## Step 1: 创建项目目录

```bash
# 创建项目目录
mkdir todo-api
cd todo-api

# 初始化 Go 模块（可选，Rick 会自动处理）
# go mod init github.com/yourusername/todo-api
```

---

## Step 2: 规划任务

```bash
rick plan "创建一个 Todo List RESTful API，使用 Go 语言实现，包含 GET/POST/PUT/DELETE 接口，使用内存存储，添加单元测试"
```

### 查看生成的任务

```bash
# 查看任务列表
ls .rick/jobs/job_0/plan/tasks/

# 输出示例：
# task1.md  task2.md  task3.md  task4.md  task5.md
```

### 查看任务依赖关系

```bash
cat .rick/jobs/job_0/plan/tasks.json
```

**示例输出**:
```json
[
  {
    "task_id": "task1",
    "task_name": "初始化 Go 项目结构",
    "dep": [],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task2",
    "task_name": "定义 Todo 数据模型",
    "dep": ["task1"],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task3",
    "task_name": "实现内存存储层",
    "dep": ["task2"],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task4",
    "task_name": "实现 HTTP 处理器",
    "dep": ["task3"],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task5",
    "task_name": "编写单元测试",
    "dep": ["task4"],
    "state_info": {"status": "pending"}
  }
]
```

### 查看具体任务内容

```bash
cat .rick/jobs/job_0/plan/tasks/task1.md
```

**示例输出**:
```markdown
# 依赖关系
(无依赖)

# 任务名称
初始化 Go 项目结构

# 任务目标
创建项目基本结构，包括 go.mod、main.go 和必要的目录

# 关键结果
1. 创建 go.mod 文件，模块名为 github.com/yourusername/todo-api
2. 创建 main.go 入口文件
3. 创建以下目录结构：
   - internal/models/ - 数据模型
   - internal/storage/ - 存储层
   - internal/handlers/ - HTTP 处理器
   - internal/server/ - HTTP 服务器
4. 创建 README.md 说明项目

# 测试方法
1. 检查 go.mod 文件存在且内容正确
2. 检查所有目录已创建
3. 运行 `go mod tidy` 确保模块初始化成功
```

### 修改任务（可选）

如果你对任务分解不满意，可以手动编辑：

```bash
# 编辑任务描述
vim .rick/jobs/job_0/plan/tasks/task1.md

# 编辑依赖关系
vim .rick/jobs/job_0/plan/tasks.json
```

**修改示例**：
假设你想将 task4 拆分为两个任务：

```json
[
  // ... 前面的任务 ...
  {
    "task_id": "task4",
    "task_name": "实现 GET 和 POST 接口",
    "dep": ["task3"],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task5",
    "task_name": "实现 PUT 和 DELETE 接口",
    "dep": ["task4"],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task6",
    "task_name": "编写单元测试",
    "dep": ["task5"],
    "state_info": {"status": "pending"}
  }
]
```

---

## Step 3: 执行任务

```bash
rick doing job_0
```

### 执行过程

Rick 会按照拓扑排序顺序执行任务：

1. **task1: 初始化 Go 项目结构**
   - Claude Code 创建目录和文件
   - 运行测试脚本验证
   - Git commit: "feat: 初始化 Go 项目结构"

2. **task2: 定义 Todo 数据模型**
   - Claude Code 创建 `internal/models/todo.go`
   - 运行测试脚本验证
   - Git commit: "feat: 定义 Todo 数据模型"

3. **task3: 实现内存存储层**
   - Claude Code 创建 `internal/storage/memory.go`
   - 运行测试脚本验证
   - Git commit: "feat: 实现内存存储层"

4. **task4: 实现 HTTP 处理器**
   - Claude Code 创建 `internal/handlers/*.go`
   - 运行测试脚本验证
   - Git commit: "feat: 实现 HTTP 处理器"

5. **task5: 编写单元测试**
   - Claude Code 创建测试文件
   - 运行测试脚本验证
   - Git commit: "test: 添加单元测试"

### 监控执行进度

在另一个终端窗口中：

```bash
# 查看执行日志
tail -f .rick/jobs/job_0/doing/logs/executor.log

# 查看任务状态
cat .rick/jobs/job_0/doing/tasks.json | jq '.[] | {task_id, status: .state_info.status}'
```

### 处理执行失败

如果某个任务失败，Rick 会：
1. 记录错误到 `debug.md`
2. 自动重试（默认最多 5 次）
3. 如果超过重试限制，退出并提示人工干预

**查看失败记录**:
```bash
cat .rick/jobs/job_0/doing/debug.md
```

**示例输出**:
```markdown
# debug1: task3 执行失败

**问题描述**
内存存储层实现时，忘记处理并发访问，导致 race condition

**解决状态**
未解决

**解决方法**
需要添加 sync.RWMutex 保护并发访问
```

**修复并重试**:
```bash
# 1. 根据 debug.md 修改任务描述
vim .rick/jobs/job_0/plan/tasks/task3.md

# 2. 添加并发安全要求
# 在"关键结果"中添加：
# 5. 使用 sync.RWMutex 保护并发访问

# 3. 重新执行
rick doing job_0
```

---

## Step 4: 验证结果

### 查看 Git 提交历史

```bash
git log --oneline
```

**示例输出**:
```
a1b2c3d test: 添加单元测试
e4f5g6h feat: 实现 HTTP 处理器
i7j8k9l feat: 实现内存存储层
m0n1o2p feat: 定义 Todo 数据模型
q3r4s5t feat: 初始化 Go 项目结构
u6v7w8x Initial commit
```

### 查看项目结构

```bash
tree -L 3 -I '.git|.rick'
```

**示例输出**:
```
.
├── go.mod
├── go.sum
├── main.go
├── internal
│   ├── models
│   │   └── todo.go
│   ├── storage
│   │   ├── memory.go
│   │   └── memory_test.go
│   ├── handlers
│   │   ├── todo.go
│   │   └── todo_test.go
│   └── server
│       └── server.go
└── README.md
```

### 运行测试

```bash
go test ./...
```

**示例输出**:
```
ok      github.com/yourusername/todo-api/internal/storage    0.002s
ok      github.com/yourusername/todo-api/internal/handlers   0.003s
```

### 运行服务

```bash
go run main.go
```

**测试 API**:
```bash
# 创建 Todo
curl -X POST http://localhost:8080/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Learn Rick CLI","completed":false}'

# 获取所有 Todos
curl http://localhost:8080/todos

# 更新 Todo
curl -X PUT http://localhost:8080/todos/1 \
  -H "Content-Type: application/json" \
  -d '{"title":"Learn Rick CLI","completed":true}'

# 删除 Todo
curl -X DELETE http://localhost:8080/todos/1
```

---

## Step 5: 知识积累

```bash
rick learning job_0
```

### 查看学习成果

```bash
# 查看任务总结
cat .rick/jobs/job_0/learning/summary.md

# 查看提取的技能
cat .rick/jobs/job_0/learning/skills.md

# 查看识别的模式
cat .rick/jobs/job_0/learning/patterns.md
```

**示例 summary.md**:
```markdown
# Job 0 任务总结

## 任务概述
创建了一个完整的 Todo List RESTful API，使用 Go 语言实现。

## 完成的任务
1. ✅ 初始化 Go 项目结构
2. ✅ 定义 Todo 数据模型
3. ✅ 实现内存存储层（支持并发）
4. ✅ 实现 HTTP 处理器（GET/POST/PUT/DELETE）
5. ✅ 编写单元测试

## 关键成果
- 完整的 RESTful API 实现
- 线程安全的内存存储
- 100% 测试覆盖率
- 清晰的项目结构

## 遇到的问题
- 问题 1: 内存存储层初始忘记处理并发
  - 解决: 添加 sync.RWMutex
- 问题 2: HTTP 处理器错误处理不完善
  - 解决: 添加统一错误处理中间件

## 学到的经验
1. Go 项目结构最佳实践（internal/ 目录）
2. 并发安全的数据结构设计
3. RESTful API 设计原则
4. 单元测试编写方法
```

**示例 skills.md**:
```markdown
# 提取的技能

## Skill 1: Go 项目结构设计
- 使用 internal/ 目录保护内部包
- 按功能分层：models, storage, handlers, server
- 清晰的依赖关系

## Skill 2: 并发安全的内存存储
- 使用 sync.RWMutex 保护共享数据
- 读写锁优化性能
- 原子操作保证数据一致性

## Skill 3: RESTful API 设计
- HTTP 方法语义（GET/POST/PUT/DELETE）
- 资源路由设计
- JSON 序列化/反序列化
- 错误处理和状态码

## Skill 4: Go 单元测试
- 表驱动测试（table-driven tests）
- Mock 和依赖注入
- 测试覆盖率工具
```

**示例 patterns.md**:
```markdown
# 识别的模式

## Pattern 1: 分层架构
- 模型层（models）定义数据结构
- 存储层（storage）处理数据持久化
- 处理器层（handlers）处理业务逻辑
- 服务器层（server）处理 HTTP 路由

## Pattern 2: 依赖注入
- 通过构造函数注入依赖
- 接口定义契约，便于测试
- 避免全局变量

## Pattern 3: 错误处理
- 统一错误类型定义
- 错误传播和包装
- HTTP 错误响应标准化
```

### 查看全局知识库

```bash
# 查看全局技能库
ls .rick/skills/

# 查看全局模式库
ls .rick/patterns/
```

---

## 🎓 学习要点

### 1. 任务分解原则

- **单一职责**: 每个任务只做一件事
- **明确依赖**: 清晰定义任务之间的依赖关系
- **可测试性**: 每个任务都应该有明确的测试方法
- **粒度适中**: 任务不应太大（> 1 小时）或太小（< 5 分钟）

### 2. 依赖关系设计

- **串行依赖**: task2 依赖 task1 完成
- **并行执行**: task2 和 task3 可以并行（无依赖）
- **多重依赖**: task4 依赖 task2 和 task3 都完成

### 3. 测试方法编写

- **明确性**: 测试步骤应该清晰明确
- **可自动化**: 尽量使用脚本自动化测试
- **覆盖性**: 测试应该覆盖所有关键结果

### 4. 失败处理策略

- **记录详细**: 在 debug.md 中记录详细的错误信息
- **修改任务**: 根据错误修改任务描述，而不是直接修改代码
- **重新执行**: 让 Rick 重新执行修改后的任务

---

## 💡 常见问题

### Q1: 任务粒度如何把握？

**A**: 遵循以下原则：
- 单个任务应该在 15-30 分钟内完成
- 任务应该有明确的输入和输出
- 任务应该可以独立测试

### Q2: 如何处理复杂依赖？

**A**: 使用 DAG（有向无环图）：
```json
{
  "task_id": "task4",
  "dep": ["task2", "task3"]  // task4 依赖 task2 和 task3
}
```

### Q3: 测试失败如何处理？

**A**:
1. 查看 `debug.md` 了解失败原因
2. 修改 `task.md` 添加更详细的要求
3. 重新执行 `rick doing job_0`

### Q4: 如何跳过某个任务？

**A**:
1. 手动完成该任务
2. 在 `tasks.json` 中将状态改为 "done"
3. Git commit 提交更改
4. 继续执行 `rick doing job_0`

---

## 🚀 下一步

恭喜完成第一个教程！接下来你可以：

1. **[Tutorial 2: 自我重构](./tutorial-2-self-refactor.md)** - 学习使用 Rick 重构 Rick
2. **[Tutorial 3: 并行版本管理](./tutorial-3-parallel-versions.md)** - 掌握 rick + rick_dev 工作流
3. **[最佳实践](../best-practices.md)** - 学习任务分解和依赖设计的最佳实践

---

*最后更新: 2026-03-14*
