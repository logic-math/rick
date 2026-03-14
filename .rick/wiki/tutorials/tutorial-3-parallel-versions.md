# Tutorial 3: 并行版本管理（rick + rick_dev）

> 掌握 Rick CLI 的双版本并行工作流，提升开发效率

## 📋 目标

在本教程中，你将学习：
- 如何同时维护生产版本和开发版本
- 如何在两个版本之间切换和协作
- 如何使用版本隔离避免冲突
- 如何管理多个开发分支

## 🎯 场景描述

假设你正在使用 Rick CLI 开发一个大型项目，同时需要：
1. 使用生产版本处理日常任务
2. 使用开发版本实验新功能
3. 在两个版本之间无缝切换

---

## 版本管理架构

### 目录结构

```
~/.rick/                    # 生产版本
├── bin/rick                # 生产二进制
├── config.json             # 共享配置
└── jobs/                   # 共享工作空间

~/.rick_dev/                # 开发版本
├── bin/rick                # 开发二进制
└── (共享 ~/.rick/config.json 和 jobs/)

~/.local/bin/
├── rick -> ~/.rick/bin/rick          # 生产命令
└── rick_dev -> ~/.rick_dev/bin/rick  # 开发命令

项目目录/
└── .rick/                  # 共享工作空间
    ├── config.json         # 项目配置
    ├── jobs/               # 所有 Job
    ├── skills/             # 全局技能库
    └── patterns/           # 全局模式库
```

### 共享与隔离

| 资源 | 共享/隔离 | 说明 |
|------|----------|------|
| 二进制文件 | 隔离 | 每个版本有独立的二进制 |
| 配置文件 | 共享 | 使用相同的 config.json |
| 工作空间 | 共享 | 使用相同的 .rick/ 目录 |
| Git 仓库 | 共享 | 使用相同的 Git 仓库 |
| Job ID | 共享 | Job ID 递增，不冲突 |

---

## Step 1: 安装和配置

### 安装两个版本

```bash
# 安装生产版本
cd ~/ai_coding/CODING/rick
./scripts/install.sh

# 安装开发版本
./scripts/install.sh --dev

# 验证安装
rick --version      # 生产版本
rick_dev --version  # 开发版本
```

### 配置环境变量（可选）

```bash
# 添加到 ~/.bashrc 或 ~/.zshrc

# 为两个版本设置别名
alias rp='rick'      # rick production
alias rd='rick_dev'  # rick development

# 设置默认编辑器
export EDITOR=vim

# 设置 Rick 配置
export RICK_LOG_LEVEL=info
```

---

## Step 2: 并行工作流示例

### 场景 1: 日常开发 + 实验新功能

```bash
# 终端 1: 使用生产版本处理日常任务
rick plan "修复用户登录 bug"
rick doing job_0
rick learning job_0

# 终端 2: 使用开发版本实验新功能
rick_dev plan "实验新的缓存策略"
rick_dev doing job_1
rick_dev learning job_1
```

### 场景 2: 生产版本规划 + 开发版本执行

```bash
# 步骤 1: 使用生产版本规划（稳定）
rick plan "重构数据库层"

# 步骤 2: 查看任务分解
cat .rick/jobs/job_2/plan/tasks.json

# 步骤 3: 使用开发版本执行（实验性）
rick_dev doing job_2

# 步骤 4: 验证结果
go test ./...

# 步骤 5: 如果成功，更新生产版本
./scripts/update.sh
```

### 场景 3: 多分支并行开发

```bash
# 分支 1: 使用生产版本开发特性 A
git checkout -b feature/a
rick plan "实现特性 A"
rick doing job_3

# 分支 2: 使用开发版本开发特性 B
git checkout -b feature/b
rick_dev plan "实现特性 B"
rick_dev doing job_4

# 合并
git checkout main
git merge feature/a
git merge feature/b
```

---

## Step 3: 版本切换策略

### 策略 1: 基于任务类型

| 任务类型 | 使用版本 | 原因 |
|---------|---------|------|
| 日常 Bug 修复 | 生产版本 | 稳定可靠 |
| 新功能开发 | 开发版本 | 可以实验 |
| 核心重构 | 生产版本规划 + 开发版本执行 | 安全验证 |
| 紧急修复 | 生产版本 | 快速响应 |

### 策略 2: 基于风险级别

```bash
# 低风险任务（使用生产版本）
rick plan "更新文档"
rick doing job_5

# 中风险任务（使用开发版本）
rick_dev plan "优化性能"
rick_dev doing job_6

# 高风险任务（双版本协作）
rick plan "重构核心模块"           # 规划
rick_dev doing job_7                # 执行
rick_dev plan "验证重构"            # 验证
rick_dev doing job_8
./scripts/update.sh                 # 更新生产版本
```

### 策略 3: 基于开发阶段

```bash
# 探索阶段（开发版本）
rick_dev plan "探索新技术栈"
rick_dev doing job_9

# 实现阶段（生产版本）
rick plan "实现确定的方案"
rick doing job_10

# 测试阶段（开发版本）
rick_dev plan "压力测试"
rick_dev doing job_11

# 部署阶段（生产版本）
rick plan "准备发布"
rick doing job_12
```

---

## Step 4: 工作空间管理

### 查看 Job 历史

```bash
# 查看所有 Job
ls .rick/jobs/

# 输出示例：
# job_0/  job_1/  job_2/  job_3/  job_4/  ...

# 查看每个 Job 的元信息
for job in .rick/jobs/job_*; do
    echo "=== $job ==="
    cat "$job/plan/tasks.json" | jq '.[] | {task_id, task_name}'
done
```

### 区分版本创建的 Job

```bash
# 在 Job 描述中添加版本标记
rick plan "[PROD] 修复登录 bug"
rick_dev plan "[DEV] 实验新缓存"

# 查看时可以快速识别
ls .rick/jobs/ | xargs -I {} sh -c 'echo "=== {} ==="; cat .rick/jobs/{}/plan/tasks.json | jq ".[0].task_name"'
```

### 清理旧 Job

```bash
# 备份旧 Job
mkdir -p ~/.rick/jobs_archive
mv .rick/jobs/job_{0..5} ~/.rick/jobs_archive/

# 或删除旧 Job
rm -rf .rick/jobs/job_{0..5}
```

---

## Step 5: Git 工作流集成

### 分支策略

```bash
# main 分支：使用生产版本
git checkout main
rick plan "稳定特性"
rick doing job_13

# develop 分支：使用开发版本
git checkout -b develop
rick_dev plan "实验特性"
rick_dev doing job_14

# feature 分支：根据需要选择
git checkout -b feature/new-api
rick_dev plan "开发新 API"  # 实验阶段用开发版本
rick_dev doing job_15

# 验证通过后切换到生产版本
git checkout feature/new-api
rick plan "集成新 API"       # 集成阶段用生产版本
rick doing job_16
```

### 提交信息标记

```bash
# 生产版本的提交
rick doing job_17
# Git commit: "[PROD] feat: 添加用户认证"

# 开发版本的提交
rick_dev doing job_18
# Git commit: "[DEV] feat: 实验新缓存策略"
```

### 合并策略

```bash
# 开发版本的更改合并到生产版本
git checkout main
git merge develop

# 更新生产版本二进制
./scripts/update.sh

# 验证合并结果
rick --version
go test ./...
```

---

## Step 6: 配置管理

### 共享配置

```bash
# 编辑共享配置
vim ~/.rick/config.json
```

**示例配置**:
```json
{
  "version": "0.1.0",
  "log_level": "info",
  "max_retries": 5,
  "claude_code_path": "/usr/local/bin/claude",
  "default_branch": "main"
}
```

### 版本特定配置（可选）

如果需要为两个版本设置不同的配置：

```bash
# 生产版本配置
cat > ~/.rick/config.prod.json << EOF
{
  "log_level": "info",
  "max_retries": 3
}
EOF

# 开发版本配置
cat > ~/.rick_dev/config.dev.json << EOF
{
  "log_level": "debug",
  "max_retries": 10
}
EOF

# 修改 Rick 代码以支持环境变量
export RICK_CONFIG=~/.rick/config.prod.json    # 生产版本
export RICK_DEV_CONFIG=~/.rick_dev/config.dev.json  # 开发版本
```

---

## Step 7: 监控和调试

### 并行执行监控

```bash
# 终端 1: 监控生产版本
watch -n 2 "echo '=== Production ==='; rick --version; tail -5 .rick/jobs/job_*/doing/logs/executor.log 2>/dev/null"

# 终端 2: 监控开发版本
watch -n 2 "echo '=== Development ==='; rick_dev --version; tail -5 .rick/jobs/job_*/doing/logs/executor.log 2>/dev/null"
```

### 日志分离

```bash
# 为两个版本创建独立的日志目录
mkdir -p ~/.rick/logs/{prod,dev}

# 生产版本日志
rick doing job_19 2>&1 | tee ~/.rick/logs/prod/job_19.log

# 开发版本日志
rick_dev doing job_20 2>&1 | tee ~/.rick/logs/dev/job_20.log
```

### 性能对比

```bash
# 创建性能对比脚本
cat > compare_versions.sh << 'EOF'
#!/bin/bash

echo "=== Performance Comparison ==="

echo "Testing rick (production)..."
time rick plan "test task" > /dev/null 2>&1

echo "Testing rick_dev (development)..."
time rick_dev plan "test task" > /dev/null 2>&1

echo "Done."
EOF

chmod +x compare_versions.sh
./compare_versions.sh
```

---

## 🎓 学习要点

### 1. 版本隔离原则

- **二进制隔离**: 每个版本有独立的二进制文件
- **配置共享**: 共享配置文件，减少维护成本
- **工作空间共享**: 共享工作空间，方便协作
- **Git 共享**: 共享 Git 仓库，统一版本控制

### 2. 版本选择决策树

```
任务开始
    ├─ 是否为实验性任务？
    │   ├─ 是 → 使用开发版本
    │   └─ 否 → 继续判断
    ├─ 是否为核心模块重构？
    │   ├─ 是 → 生产版本规划 + 开发版本执行
    │   └─ 否 → 继续判断
    ├─ 是否为紧急修复？
    │   ├─ 是 → 使用生产版本
    │   └─ 否 → 继续判断
    └─ 默认 → 使用生产版本
```

### 3. 版本更新时机

| 时机 | 操作 | 原因 |
|------|------|------|
| 开发版本验证通过 | `./scripts/update.sh` | 将新功能合并到生产版本 |
| 发现严重 Bug | 回滚 + 修复 + 更新 | 保证生产版本稳定 |
| 定期更新（每周） | `./scripts/update.sh` | 保持生产版本最新 |
| 重大版本发布 | 完整测试 + 更新 | 确保质量 |

---

## 💡 高级技巧

### 技巧 1: 使用 Shell 函数简化工作流

```bash
# 添加到 ~/.bashrc 或 ~/.zshrc

# 快速切换版本
rick_switch() {
    case $1 in
        prod)
            alias rick='~/.rick/bin/rick'
            echo "Switched to production version"
            ;;
        dev)
            alias rick='~/.rick_dev/bin/rick'
            echo "Switched to development version"
            ;;
        *)
            echo "Usage: rick_switch [prod|dev]"
            ;;
    esac
}

# 对比两个版本
rick_compare() {
    echo "=== Production Version ==="
    ~/.rick/bin/rick --version

    echo "=== Development Version ==="
    ~/.rick_dev/bin/rick --version

    echo "=== Binary Sizes ==="
    ls -lh ~/.rick/bin/rick ~/.rick_dev/bin/rick
}

# 同步开发版本到生产版本
rick_sync() {
    echo "Syncing development to production..."
    ./scripts/update.sh
    echo "Done."
}
```

### 技巧 2: 使用 Git Hooks 自动化

```bash
# 创建 post-commit hook
cat > .git/hooks/post-commit << 'EOF'
#!/bin/bash

# 如果是开发分支的提交，自动重建开发版本
current_branch=$(git branch --show-current)
if [[ "$current_branch" == "develop" ]]; then
    echo "Rebuilding rick_dev..."
    ./scripts/build.sh
    ./scripts/install.sh --dev
    echo "rick_dev updated."
fi
EOF

chmod +x .git/hooks/post-commit
```

### 技巧 3: 使用 Makefile 管理版本

```makefile
# Makefile

.PHONY: install-prod install-dev update-prod update-dev clean

install-prod:
	./scripts/install.sh

install-dev:
	./scripts/install.sh --dev

update-prod:
	./scripts/update.sh

update-dev:
	./scripts/update.sh --dev

clean:
	./scripts/uninstall.sh
	./scripts/uninstall.sh --dev

compare:
	@echo "=== Production ==="
	@rick --version
	@echo "=== Development ==="
	@rick_dev --version

test-both:
	@echo "Testing production version..."
	@rick plan "test" && rick doing job_test
	@echo "Testing development version..."
	@rick_dev plan "test" && rick_dev doing job_test
```

---

## 🚧 常见问题

### Q1: 两个版本会互相干扰吗？

**A**: 不会。两个版本：
- 使用不同的二进制文件
- 共享相同的工作空间（.rick/）
- Job ID 递增，不会冲突
- Git 提交历史共享

### Q2: 如何知道当前使用的是哪个版本？

**A**:
```bash
# 方法 1: 查看版本号
rick --version      # 生产版本
rick_dev --version  # 开发版本

# 方法 2: 查看二进制路径
which rick          # ~/.local/bin/rick -> ~/.rick/bin/rick
which rick_dev      # ~/.local/bin/rick_dev -> ~/.rick_dev/bin/rick

# 方法 3: 在提示符中显示
# 添加到 ~/.bashrc
PS1='[\u@\h \W $(basename $(readlink -f $(which rick)))]\$ '
```

### Q3: 开发版本和生产版本的 Job ID 会冲突吗？

**A**: 不会。Job ID 是全局递增的：
```bash
rick plan "task 1"      # 创建 job_0
rick_dev plan "task 2"  # 创建 job_1
rick plan "task 3"      # 创建 job_2
```

### Q4: 如何回滚到旧版本？

**A**:
```bash
# 方法 1: 从 Git 历史回滚
git log --oneline
git checkout <commit-hash>
./scripts/build.sh
./scripts/install.sh

# 方法 2: 从备份恢复
cp ~/.rick/bin/rick.backup ~/.rick/bin/rick

# 方法 3: 重新安装特定版本
./scripts/install.sh --version v0.1.0
```

---

## 🚀 下一步

恭喜掌握并行版本管理！接下来你可以：

1. **[Tutorial 4: 自定义提示词](./tutorial-4-custom-prompts.md)** - 定制提示词模板
2. **[Tutorial 5: CI/CD 集成](./tutorial-5-cicd-integration.md)** - 集成到 CI/CD 流程
3. **[最佳实践](../best-practices.md)** - 学习版本管理的最佳实践

---

*最后更新: 2026-03-14*
