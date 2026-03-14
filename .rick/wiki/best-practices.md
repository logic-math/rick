# Rick CLI 最佳实践

> 基于实践经验总结的 Rick CLI 使用最佳实践

## 📋 目录

- [任务分解原则](#任务分解原则)
- [依赖关系设计](#依赖关系设计)
- [测试方法编写](#测试方法编写)
- [失败重试策略](#失败重试策略)
- [Learning 阶段审核](#learning-阶段审核)
- [版本管理实践](#版本管理实践)
- [提示词优化](#提示词优化)
- [性能优化](#性能优化)

---

## 任务分解原则

### 1. 单一职责原则

**原则**: 每个任务只做一件事

**✅ 好的示例**:
```markdown
# task1.md
## 任务名称
创建用户数据模型

## 任务目标
定义 User 结构体，包含基本字段和 JSON 标签

## 关键结果
1. 创建 internal/models/user.go
2. 定义 User 结构体（ID, Name, Email, CreatedAt）
3. 添加 JSON 标签
```

**❌ 不好的示例**:
```markdown
# task1.md
## 任务名称
创建用户系统

## 任务目标
实现完整的用户管理功能

## 关键结果
1. 创建数据模型
2. 实现 CRUD 接口
3. 添加认证
4. 编写测试
5. 部署到生产环境
```

**问题**: 任务太大，包含多个职责，难以测试和验证。

### 2. 粒度适中原则

**原则**: 任务应该在 15-30 分钟内完成

**粒度判断**:
- **太大** (> 1 小时): 需要进一步分解
- **适中** (15-30 分钟): 理想粒度 ✅
- **太小** (< 5 分钟): 可以合并

**示例**:
```json
{
  "task_id": "task1",
  "task_name": "初始化 Go 模块",
  "estimated_time": "5 分钟"  // ✅ 适中
}

{
  "task_id": "task2",
  "task_name": "实现完整的用户认证系统",
  "estimated_time": "3 小时"  // ❌ 太大，需要分解
}
```

### 3. 明确输入输出原则

**原则**: 任务应该有明确的输入和输出

**✅ 好的示例**:
```markdown
## 任务目标
读取 config.json，解析配置，返回 Config 结构体

## 输入
- config.json 文件路径

## 输出
- Config 结构体实例
- 错误信息（如果解析失败）

## 关键结果
1. 创建 internal/config/loader.go
2. 实现 LoadConfig(path string) (*Config, error)
3. 处理文件不存在、JSON 格式错误等异常
```

### 4. 可测试性原则

**原则**: 每个任务都应该有明确的测试方法

**✅ 好的示例**:
```markdown
## 测试方法
1. 创建测试文件 config_test.go
2. 测试正常情况：加载有效的 config.json
3. 测试异常情况：
   - 文件不存在
   - JSON 格式错误
   - 缺少必需字段
4. 运行 `go test ./internal/config/` 确保所有测试通过
```

### 5. 依赖最小化原则

**原则**: 任务依赖应该尽可能少

**✅ 好的示例**:
```json
[
  {
    "task_id": "task1",
    "task_name": "定义数据模型",
    "dep": []
  },
  {
    "task_id": "task2",
    "task_name": "实现存储层",
    "dep": ["task1"]  // 只依赖 task1
  },
  {
    "task_id": "task3",
    "task_name": "实现 HTTP 处理器",
    "dep": ["task2"]  // 只依赖 task2
  }
]
```

**❌ 不好的示例**:
```json
[
  {
    "task_id": "task4",
    "task_name": "实现业务逻辑",
    "dep": ["task1", "task2", "task3"]  // 依赖过多
  }
]
```

---

## 依赖关系设计

### 1. DAG 原则

**原则**: 任务依赖关系必须是有向无环图（DAG）

**✅ 有效的 DAG**:
```
task1 → task2 → task4
  ↓       ↓
task3 ----→
```

**❌ 无效的循环依赖**:
```
task1 → task2
  ↑       ↓
  ← task3 ←
```

### 2. 串行 vs 并行

**串行依赖**: 任务必须按顺序执行
```json
{
  "task_id": "task2",
  "dep": ["task1"]  // task2 必须在 task1 完成后执行
}
```

**并行执行**: 任务可以同时执行
```json
[
  {
    "task_id": "task2",
    "dep": ["task1"]
  },
  {
    "task_id": "task3",
    "dep": ["task1"]  // task2 和 task3 可以并行执行
  }
]
```

### 3. 多重依赖

**原则**: 任务可以依赖多个前置任务

```json
{
  "task_id": "task4",
  "dep": ["task2", "task3"]  // task4 依赖 task2 和 task3 都完成
}
```

**执行顺序**:
```
1. task1 (无依赖，首先执行)
2. task2 和 task3 (并行执行，都依赖 task1)
3. task4 (等待 task2 和 task3 都完成)
```

### 4. 依赖关系最佳实践

| 场景 | 依赖设计 | 原因 |
|------|---------|------|
| 数据模型 → 存储层 | 串行 | 存储层需要数据模型定义 |
| 前端 ↔ 后端 | 并行 | 可以独立开发 |
| 单元测试 → 集成测试 | 串行 | 集成测试依赖单元测试通过 |
| 多个独立功能 | 并行 | 互不依赖，加速开发 |

---

## 测试方法编写

### 1. 测试方法结构

**完整的测试方法应包含**:
```markdown
## 测试方法

### 1. 单元测试
- 创建 xxx_test.go
- 测试函数 A、B、C
- 覆盖正常情况和异常情况

### 2. 集成测试
- 测试模块间交互
- 验证数据流

### 3. 验证命令
\`\`\`bash
go test ./...
go test -cover ./...
\`\`\`

### 4. 预期结果
- 所有测试通过
- 测试覆盖率 > 80%
```

### 2. 测试驱动开发（TDD）

**原则**: 先写测试，再写实现

**流程**:
```markdown
1. 编写测试用例
2. 运行测试（应该失败）
3. 编写最小实现
4. 运行测试（应该通过）
5. 重构代码
```

### 3. 测试覆盖率目标

| 模块类型 | 覆盖率目标 |
|---------|-----------|
| 核心业务逻辑 | > 90% |
| 工具函数 | > 80% |
| HTTP 处理器 | > 75% |
| 配置加载 | > 70% |

### 4. 测试最佳实践

**✅ 好的测试**:
```go
func TestUserService_CreateUser(t *testing.T) {
    tests := []struct {
        name    string
        input   *User
        wantErr bool
    }{
        {
            name:    "valid user",
            input:   &User{Name: "Alice", Email: "alice@example.com"},
            wantErr: false,
        },
        {
            name:    "empty name",
            input:   &User{Name: "", Email: "alice@example.com"},
            wantErr: true,
        },
        {
            name:    "invalid email",
            input:   &User{Name: "Alice", Email: "invalid"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := service.CreateUser(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

---

## 失败重试策略

### 1. 重试配置

**默认配置**:
```json
{
  "max_retries": 5,
  "retry_delay": "1s",
  "exponential_backoff": true
}
```

### 2. 失败记录

**debug.md 格式**:
```markdown
# debug1: task3 执行失败

**问题描述**
HTTP 处理器未正确处理 JSON 解析错误

**错误信息**
\`\`\`
panic: interface conversion: interface {} is nil, not map[string]interface {}
\`\`\`

**解决状态**
未解决

**解决方法**
添加 JSON 解析错误处理：
\`\`\`go
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
    http.Error(w, "invalid JSON", http.StatusBadRequest)
    return
}
\`\`\`
```

### 3. 重试决策树

```
任务失败
  ├─ 是否为临时错误？
  │   ├─ 是（网络超时、资源不可用）→ 自动重试
  │   └─ 否 → 继续判断
  ├─ 重试次数 < MaxRetries？
  │   ├─ 是 → 记录到 debug.md，自动重试
  │   └─ 否 → 退出，需人工干预
  └─ 是否为代码逻辑错误？
      ├─ 是 → 退出，需人工修改任务描述
      └─ 否 → 自动重试
```

### 4. 人工干预时机

| 场景 | 操作 |
|------|------|
| 超过最大重试次数 | 查看 debug.md，修改 task.md |
| 代码逻辑错误 | 修改任务描述，重新执行 |
| 测试失败 | 调整测试方法，重新执行 |
| 依赖冲突 | 调整依赖关系，重新执行 |

---

## Learning 阶段审核

### 1. Learning 阶段目标

- 提取可复用的知识和经验
- 识别设计模式和最佳实践
- 更新全局知识库
- 为未来任务提供参考

### 2. 审核检查清单

**summary.md 审核**:
- [ ] 任务概述完整
- [ ] 关键成果清晰
- [ ] 遇到的问题有详细记录
- [ ] 解决方案可复用

**skills.md 审核**:
- [ ] 技能描述清晰
- [ ] 包含代码示例
- [ ] 可应用到其他项目
- [ ] 与项目架构对齐

**patterns.md 审核**:
- [ ] 模式定义明确
- [ ] 包含使用场景
- [ ] 有优缺点分析
- [ ] 可推广到其他模块

### 3. 知识积累流程

```
1. 自动生成 learning 文件
   ├─> summary.md
   ├─> skills.md
   └─> patterns.md

2. 人工审核
   ├─> 检查准确性
   ├─> 补充细节
   └─> 修正错误

3. 更新全局知识库
   ├─> .rick/skills/
   └─> .rick/patterns/

4. 应用到未来任务
   └─> 在提示词中引用
```

### 4. 知识库管理

**技能库结构**:
```
.rick/skills/
├── go/
│   ├── error-handling.md
│   ├── concurrency.md
│   └── testing.md
├── web/
│   ├── rest-api.md
│   └── authentication.md
└── database/
    ├── sql-optimization.md
    └── migration.md
```

**模式库结构**:
```
.rick/patterns/
├── architectural/
│   ├── layered-architecture.md
│   └── dependency-injection.md
├── design/
│   ├── factory-pattern.md
│   └── strategy-pattern.md
└── coding/
    ├── error-handling-pattern.md
    └── logging-pattern.md
```

---

## 版本管理实践

### 1. 双版本策略

| 版本 | 用途 | 更新频率 |
|------|------|---------|
| 生产版本 (rick) | 日常开发 | 每周 |
| 开发版本 (rick_dev) | 实验新功能 | 每次修改 |

### 2. 版本切换时机

```markdown
## 使用生产版本
- 日常 Bug 修复
- 稳定功能开发
- 生产环境部署

## 使用开发版本
- 实验新功能
- 核心模块重构
- 性能优化验证

## 双版本协作
- 生产版本规划
- 开发版本执行
- 开发版本验证
- 更新生产版本
```

### 3. Git 分支策略

```
main (生产分支)
  ├─ feature/* (特性分支，使用 rick)
  ├─ hotfix/* (紧急修复，使用 rick)
  └─ develop (开发分支，使用 rick_dev)
      ├─ feature/* (实验特性，使用 rick_dev)
      └─ refactor/* (重构分支，使用 rick_dev)
```

---

## 提示词优化

### 1. 提示词结构

**必需元素**:
- 角色定义（你是一个...工程师）
- 任务信息（任务 ID、名称、目标）
- 项目背景（SPEC、架构）
- 执行上下文（已完成任务、依赖关系）
- 执行要求（步骤、质量标准）

### 2. 提示词优化技巧

**添加具体示例**:
```markdown
## 代码示例

\`\`\`go
// 错误处理示例
if err != nil {
    return fmt.Errorf("failed to process: %w", err)
}
\`\`\`

请遵循类似的错误处理模式。
```

**添加检查清单**:
```markdown
## 提交前检查

- [ ] 代码已格式化
- [ ] 测试已通过
- [ ] 文档已更新
```

**添加约束条件**:
```markdown
## 技术约束

- 不使用第三方依赖
- 性能要求: < 100ms
- 内存限制: < 100MB
```

### 3. 领域特定提示词

为不同领域创建专用模板：
- Web 开发
- 数据科学
- DevOps
- 移动开发
- 嵌入式系统

---

## 性能优化

### 1. 任务执行优化

**并行执行**:
```json
// 识别可并行的任务
[
  {"task_id": "task1", "dep": []},
  {"task_id": "task2", "dep": []},  // 可与 task1 并行
  {"task_id": "task3", "dep": ["task1", "task2"]}
]
```

**缓存优化**:
```bash
# 缓存依赖
export RICK_CACHE_DEPS=true

# 缓存提示词模板
export RICK_CACHE_TEMPLATES=true
```

### 2. 工作空间优化

**定期清理**:
```bash
# 归档旧 Job
rick archive --before 30d

# 清理日志
rick clean-logs --before 7d
```

### 3. 监控和分析

**性能指标**:
- 任务执行时间
- 失败重试次数
- Claude Code 调用次数
- Git 提交频率

**分析工具**:
```bash
# 查看任务执行统计
rick stats --job job_0

# 生成性能报告
rick report --type performance
```

---

## 📚 相关资源

- [快速入门](./getting-started.md)
- [核心概念](./core-concepts.md)
- [架构设计](./architecture.md)
- [教程系列](./tutorials/)

---

*最后更新: 2026-03-14*
