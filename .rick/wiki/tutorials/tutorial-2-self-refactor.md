# Tutorial 2: 使用 Rick 重构 Rick（自我重构）

> 学习如何使用 Rick CLI 重构 Rick CLI 自身 —— 这是 Rick 最强大的特性之一

## 📋 目标

在本教程中，你将学习：
- 如何使用生产版本规划重构任务
- 如何安装和使用开发版本
- 如何在两个版本之间协作
- 如何验证重构结果并更新生产版本

## 🎯 场景描述

假设我们要重构 Rick CLI 的日志系统，将简单的 `log` 包升级为结构化日志（使用 `slog`）。

## ⚠️ 重要概念

### 版本管理机制

Rick 支持两个并行版本：

| 版本 | 安装路径 | 命令名 | 用途 |
|------|---------|--------|------|
| **生产版本** | `~/.rick/` | `rick` | 稳定版本，用于日常工作 |
| **开发版本** | `~/.rick_dev/` | `rick_dev` | 开发版本，用于实验和重构 |

### 自我重构流程

```
┌─────────────────────────────────────────────────────────────┐
│                    自我重构工作流                             │
└─────────────────────────────────────────────────────────────┘

1. 使用 rick (生产版本) 规划重构任务
   └─> rick plan "重构日志系统"

2. 安装 rick_dev (开发版本)
   └─> ./scripts/install.sh --dev

3. 使用 rick (生产版本) 执行重构
   └─> rick doing job_0
   └─> Claude Code 修改源代码
   └─> 构建新的 rick_dev 二进制

4. 使用 rick_dev (开发版本) 验证
   └─> rick_dev plan "验证重构"
   └─> rick_dev doing job_1

5. 更新生产版本
   └─> ./scripts/update.sh
   └─> rick 现在包含重构后的代码

6. 卸载开发版本
   └─> ./scripts/uninstall.sh --dev
```

---

## Step 1: 规划重构任务（使用生产版本）

```bash
cd ~/ai_coding/CODING/rick

# 使用生产版本规划重构
rick plan "重构日志系统：将 log 包升级为 slog，支持结构化日志，包含日志级别、时间戳、上下文信息"
```

### 查看生成的任务

```bash
cat .rick/jobs/job_0/plan/tasks.json
```

**示例输出**:
```json
[
  {
    "task_id": "task1",
    "task_name": "创建新的日志模块",
    "dep": [],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task2",
    "task_name": "实现结构化日志接口",
    "dep": ["task1"],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task3",
    "task_name": "迁移现有日志调用",
    "dep": ["task2"],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task4",
    "task_name": "添加日志配置",
    "dep": ["task2"],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task5",
    "task_name": "更新测试",
    "dep": ["task3", "task4"],
    "state_info": {"status": "pending"}
  }
]
```

---

## Step 2: 安装开发版本

```bash
# 安装开发版本到 ~/.rick_dev
./scripts/install.sh --dev

# 验证安装
rick_dev --version
```

**输出**:
```
rick_dev version v0.1.0 (dev)
```

### 验证两个版本并存

```bash
# 生产版本
rick --version

# 开发版本
rick_dev --version

# 查看安装路径
which rick       # ~/.local/bin/rick -> ~/.rick/bin/rick
which rick_dev   # ~/.local/bin/rick_dev -> ~/.rick_dev/bin/rick
```

---

## Step 3: 执行重构（使用生产版本）

```bash
# 使用生产版本执行重构
rick doing job_0
```

### 执行过程详解

Rick（生产版本）会：

1. **task1: 创建新的日志模块**
   ```bash
   # Claude Code 创建 internal/logging/slog.go
   # 定义结构化日志接口
   ```

2. **task2: 实现结构化日志接口**
   ```bash
   # Claude Code 实现 Logger 接口
   # 支持 Debug, Info, Warn, Error 级别
   # 支持上下文字段（key-value pairs）
   ```

3. **task3: 迁移现有日志调用**
   ```bash
   # Claude Code 替换所有 log.Printf 调用
   # 使用新的结构化日志接口
   ```

4. **task4: 添加日志配置**
   ```bash
   # Claude Code 在 config.json 添加日志配置
   # 支持日志级别、输出格式配置
   ```

5. **task5: 更新测试**
   ```bash
   # Claude Code 更新所有测试
   # 运行 go test ./...
   ```

### 监控执行

在另一个终端：

```bash
# 查看执行日志
tail -f .rick/jobs/job_0/doing/logs/executor.log

# 查看 Git 提交
watch -n 2 "git log --oneline -5"
```

### 构建新版本

重构完成后，构建新的开发版本：

```bash
# 构建并安装到开发版本
./scripts/build.sh
./scripts/install.sh --dev
```

---

## Step 4: 验证重构（使用开发版本）

现在使用开发版本验证重构结果：

```bash
# 使用开发版本规划验证任务
rick_dev plan "验证日志系统重构：测试所有日志级别、结构化字段、配置加载、性能测试"
```

### 查看验证任务

```bash
cat .rick/jobs/job_1/plan/tasks.json
```

**示例输出**:
```json
[
  {
    "task_id": "task1",
    "task_name": "测试日志级别过滤",
    "dep": [],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task2",
    "task_name": "测试结构化字段",
    "dep": [],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task3",
    "task_name": "测试配置加载",
    "dep": [],
    "state_info": {"status": "pending"}
  },
  {
    "task_id": "task4",
    "task_name": "性能基准测试",
    "dep": ["task1", "task2", "task3"],
    "state_info": {"status": "pending"}
  }
]
```

### 执行验证

```bash
# 使用开发版本执行验证
rick_dev doing job_1
```

### 验证测试

```bash
# 运行所有测试
go test ./...

# 运行基准测试
go test -bench=. ./internal/logging/

# 检查测试覆盖率
go test -cover ./...
```

**示例输出**:
```
ok      github.com/anthropics/rick/internal/logging    0.015s  coverage: 95.2% of statements
ok      github.com/anthropics/rick/internal/cmd        0.023s  coverage: 87.4% of statements
ok      github.com/anthropics/rick/internal/executor   0.019s  coverage: 91.3% of statements
```

---

## Step 5: 对比两个版本

### 功能对比

```bash
# 使用生产版本（旧日志系统）
rick plan "测试任务"

# 查看日志输出（简单文本）
# 2026/03/14 10:30:45 Starting plan command
# 2026/03/14 10:30:46 Creating workspace
# 2026/03/14 10:30:47 Calling Claude Code

# 使用开发版本（新日志系统）
rick_dev plan "测试任务"

# 查看日志输出（结构化）
# time=2026-03-14T10:30:45+08:00 level=INFO msg="Starting plan command" command=plan
# time=2026-03-14T10:30:46+08:00 level=INFO msg="Creating workspace" job_id=job_2
# time=2026-03-14T10:30:47+08:00 level=INFO msg="Calling Claude Code" provider=anthropic
```

### 性能对比

```bash
# 基准测试
go test -bench=BenchmarkLogger -benchmem ./internal/logging/
```

**示例输出**:
```
BenchmarkLogger/Old-8         1000000    1234 ns/op    256 B/op    4 allocs/op
BenchmarkLogger/New-8         2000000     678 ns/op    128 B/op    2 allocs/op
```

---

## Step 6: 知识积累

### 使用生产版本积累知识

```bash
rick learning job_0
```

### 使用开发版本积累知识

```bash
rick_dev learning job_1
```

### 对比学习成果

```bash
# 生产版本的学习成果
cat .rick/jobs/job_0/learning/summary.md

# 开发版本的学习成果
cat .rick/jobs/job_1/learning/summary.md
```

---

## Step 7: 更新生产版本

验证通过后，更新生产版本：

```bash
# 方法 1: 使用 update.sh（推荐）
./scripts/update.sh

# 方法 2: 手动更新
./scripts/uninstall.sh
./scripts/install.sh

# 验证更新
rick --version
```

---

## Step 8: 清理开发版本

```bash
# 卸载开发版本
./scripts/uninstall.sh --dev

# 验证卸载
which rick_dev  # 应该返回空
```

---

## 🎓 学习要点

### 1. 为什么需要两个版本？

- **安全性**: 生产版本保持稳定，开发版本可以大胆实验
- **验证性**: 使用开发版本验证重构结果
- **并行性**: 两个版本可以同时运行，互不干扰

### 2. 自我重构的优势

- **吃自己的狗粮**: Rick 使用自己来重构自己
- **快速迭代**: 无需手动编写代码
- **质量保证**: 通过测试和验证确保质量

### 3. 版本切换时机

| 场景 | 使用版本 |
|------|---------|
| 日常开发 | 生产版本 (rick) |
| 实验新功能 | 开发版本 (rick_dev) |
| 重构核心模块 | 生产版本规划 + 开发版本验证 |
| 验证稳定性 | 开发版本 |

---

## 💡 高级技巧

### 技巧 1: 使用 Git 分支管理

```bash
# 在开发版本中创建特性分支
git checkout -b feature/structured-logging

# 使用 rick_dev 开发
rick_dev plan "实现特性"
rick_dev doing job_0

# 合并到主分支
git checkout main
git merge feature/structured-logging

# 更新生产版本
./scripts/update.sh
```

### 技巧 2: 增量验证

```bash
# 不要一次性重构所有模块
# 而是逐步验证每个模块

# 步骤 1: 重构日志模块
rick plan "重构日志模块"
rick doing job_0
./scripts/install.sh --dev
rick_dev plan "验证日志模块"
rick_dev doing job_1

# 步骤 2: 重构配置模块
rick plan "重构配置模块"
rick doing job_2
./scripts/install.sh --dev
rick_dev plan "验证配置模块"
rick_dev doing job_3

# ... 依次类推
```

### 技巧 3: 保留开发版本日志

```bash
# 在卸载前备份开发版本的工作空间
cp -r ~/.rick_dev/jobs ~/.rick/jobs_dev_backup

# 卸载开发版本
./scripts/uninstall.sh --dev

# 需要时查看开发版本的历史
ls ~/.rick/jobs_dev_backup/
```

---

## 🚧 常见问题

### Q1: 两个版本的工作空间会冲突吗？

**A**: 不会。两个版本使用相同的 `.rick/` 工作空间，但：
- Job ID 是递增的，不会冲突
- Git 仓库是同一个，提交历史共享
- 配置文件共享（`~/.rick/config.json`）

### Q2: 如何在开发版本中测试新功能？

**A**:
```bash
# 1. 安装开发版本
./scripts/install.sh --dev

# 2. 修改源代码
vim internal/cmd/new_feature.go

# 3. 重新构建并安装
./scripts/build.sh
./scripts/install.sh --dev

# 4. 测试新功能
rick_dev new-feature
```

### Q3: 开发版本验证失败怎么办？

**A**:
```bash
# 1. 查看失败原因
cat .rick/jobs/job_1/doing/debug.md

# 2. 回滚到上一个稳定版本
git log --oneline
git revert <commit-hash>

# 3. 重新构建开发版本
./scripts/build.sh
./scripts/install.sh --dev

# 4. 重新验证
rick_dev doing job_1
```

### Q4: 如何比较两个版本的性能？

**A**:
```bash
# 创建性能测试脚本
cat > benchmark.sh << 'EOF'
#!/bin/bash
echo "Testing rick (production)..."
time rick plan "test" > /dev/null

echo "Testing rick_dev (development)..."
time rick_dev plan "test" > /dev/null
EOF

chmod +x benchmark.sh
./benchmark.sh
```

---

## 🚀 下一步

恭喜完成自我重构教程！接下来你可以：

1. **[Tutorial 3: 并行版本管理](./tutorial-3-parallel-versions.md)** - 深入学习版本管理技巧
2. **[Tutorial 4: 自定义提示词](./tutorial-4-custom-prompts.md)** - 定制提示词模板
3. **[最佳实践](../best-practices.md)** - 学习重构的最佳实践

---

## 📚 相关资源

- [版本管理机制](../../DEVELOPMENT_GUIDE.md#版本管理机制)
- [安装脚本详解](../../DEVELOPMENT_GUIDE.md#安装脚本详解)
- [自我重构工作流](../../DEVELOPMENT_GUIDE.md#场景2使用-rick-重构-rick自我重构)

---

*最后更新: 2026-03-14*
