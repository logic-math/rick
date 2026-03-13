# 使用 Golang 开发 Rick 命令行程序 - 调研报告

**调查主题**: 使用 Golang 开发 Rick 命令行程序
**调研日期**: 2026-03-13

---

## 1. 项目概述

### 1.1 项目类型
- **主项目**: Rick（理论框架与设计规范）
- **参考实现**: Morty（Go语言实现）
- **本次研究**: Rick CLI 的 Go 实现

### 1.2 Rick 项目核心定义

Rick 是一个**Context-First AI Coding Framework**，核心理论：
```
AICoding = Humans + Agents
其中 Agents = Models + Harness
Harness 包含两个反馈循环：
1. Agent Loop: UI → Model → Tools（Claude Code已实现）
2. Context Loop: Plan → Doing → Learning（Rick需要完成）
```

### 1.3 Rick 的三个核心阶段

| 阶段 | 命令 | 输入 | 输出 | 目的 |
|------|------|------|------|------|
| **Plan** | `rick plan "需求"` | 用户需求 | task*.md | 规划任务与依赖关系 |
| **Doing** | `rick doing job_n` | task*.md | 代码+debug.md | 执行任务并记录问题 |
| **Learning** | `rick learning job_n` | debug.md+git历史 | OKR/SPEC/Wiki更新 | 知识积累与优化 |

### 1.4 关键文件格式规范

#### task.md（任务定义）
```markdown
# 依赖关系
task1, task2

# 任务名称
创建 server.go 源文件

# 任务目标
基于gRPC框架，完成gRPC Server的搭建工作

# 关键结果
1. 检查是否已安装gRPC最新版本
2. 检查Go版本是否满足最新gRPC要求
3. 查阅gRPC最新版本Server构建文档
4. 编写Golang代码：实现一个gRPC Hello World处理函数
5. 确保代码通过lint检查

# 测试方法
1. 学习gRPC服务端单元测试规范
2. 编写单元测试：启动Server，发起RPC请求
3. 运行单元测试：`go test -v ./...`
```

#### tasks.json（任务DAG）
```json
[
    {
        "task_id": "task1",
        "task_name": "环境检查与依赖安装",
        "dep": [],
        "state_info": {"status": "pending"}
    },
    {
        "task_id": "task2",
        "task_name": "学习gRPC最佳实践",
        "dep": ["task1"],
        "state_info": {"status": "pending"}
    }
]
```

#### debug.md（问题记录）
```markdown
# debug1: 域名解析失败

**问题描述**
无法科学上网，无法解析github.com域名

**解决状态**
已解决

**解决方法**
- step1: 启动proxy.sh配置代理
- step2: ping github.com验证连通性
```

### 1.5 工作空间结构

```
.rick/                          # 全局上下文（人类把控）
├── OKR.md                      # 项目全局目标
├── SPEC.md                     # 项目开发规范
├── wiki/                       # 项目知识库
└── skills/                     # 可复用技能库

jobs/                           # 任务执行集合
├── job_1/
│   ├── plan/
│   │   ├── draft/             # 调研草稿
│   │   └── tasks/             # 标准化任务定义
│   ├── doing/
│   │   ├── doing.log          # 执行日志
│   │   ├── debug.md           # 问题记录
│   │   ├── tasks.json         # 任务DAG
│   │   └── tests/             # 测试脚本
│   └── learning/              # 知识沉淀
```

---

## 2. 参考实现分析（Morty）与简化设计

### 2.1 Morty 项目架构（参考）

**项目位置**: `/Users/sunquan/ai_coding/CODING/morty/`

Morty 是一个完整的参考实现，具有完善的日志系统、配置管理、状态追踪等功能。Rick 的设计将**有选择性地复用**其中的核心部分，同时删除不必要的复杂性。

### 2.2 Rick 的简化设计原则

#### 原则1：最小化日志系统
- ✅ 使用 Go 标准库 `log` 包
- ✅ 仅输出文本格式（无JSON）
- ❌ 不需要日志轮转、多格式支持
- **输出**: 简单的 stdout/stderr 即可

#### 原则2：移除状态追踪命令
- ❌ 不需要 `rick status` / `rick stat` 命令
- ❌ 不需要 `rick reset` 命令
- ✅ 通过 Git 本身进行版本管理
- ✅ 使用 `git log` 查看执行历史
- ✅ 使用 `git checkout` 进行回滚

#### 原则3：简化配置系统
- ✅ 单一全局配置文件：`~/.rick/config.json`
- ❌ 不需要5层级配置
- ❌ 不需要环境变量覆盖
- ❌ 不需要项目级配置
- **配置内容**: 仅包含必要的全局设置（如 Claude Code CLI 路径等）

### 2.3 Rick 的核心工作流

```
┌─────────────────────────────────────────────────────┐
│ rick plan "需求描述"                                │
│ - 启动交互式Claude Code会话                         │
│ - 生成task*.md文件                                 │
│ - 保存到 .rick/jobs/job_n/plan/tasks/             │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│ rick doing job_n                                    │
│ - 加载 task*.md 构建 DAG                           │
│ - 生成 tasks.json（拓扑排序）                       │
│ - 串行执行任务（按依赖关系）                        │
│ - 自动提交到 git                                   │
│ - 每个task完成后 commit 一次                       │
└─────────────────────────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────┐
│ rick learning job_n                                 │
│ - 读取 debug.md 和 git 历史                        │
│ - 人类审核后更新 .rick/OKR.md 等                  │
│ - 或在 init 模式下自动化处理                       │
└─────────────────────────────────────────────────────┘
```

---

## 3. Rick CLI 的 Go 实现需求

### 3.1 核心命令集

| 命令 | 功能 | 输入 | 输出 |
|------|------|------|------|
| `rick init` | 初始化项目 | 项目路径 | .rick/ 目录结构 |
| `rick plan` | 规划任务 | 需求描述 | job_n/plan/ |
| `rick doing` | 执行任务 | job_n | job_n/doing/ |
| `rick learning` | 知识积累 | job_n | job_n/learning/ + .rick/更新 |
| `rick status` | 显示状态 | job_n | 当前执行状态 |
| `rick reset` | 回滚版本 | job_n/版本号 | 回滚到指定版本 |

### 3.2 需要实现的核心功能

#### 3.2.1 文件系统操作
- [ ] 创建/管理 .rick 目录结构
- [ ] 创建/管理 jobs/job_n 目录
- [ ] 创建 plan/draft、plan/tasks、doing、learning 子目录
- [ ] 读写 task.md、tasks.json、debug.md 文件

#### 3.2.2 Markdown 解析
- [ ] 解析 task.md 中的依赖关系（# 依赖关系 字段）
- [ ] 提取任务名称、目标、关键结果、测试方法
- [ ] 支持多个 task*.md 文件的批量解析

#### 3.2.3 DAG 构建与拓扑排序
- [ ] 基于 task.md 中的依赖关系构建有向无环图（DAG）
- [ ] 执行拓扑排序生成任务执行序列
- [ ] 生成 tasks.json 文件
- [ ] 检测循环依赖并报错

#### 3.2.4 任务执行引擎（核心）
- [ ] **串行执行**：按拓扑排序顺序执行任务
- [ ] 加载 task.md、debug.md、OKR.md、SPEC.md 构建提示词
- [ ] 调用 Claude Code CLI 生成测试脚本
- [ ] 执行测试脚本并收集结果
- [ ] **失败重试循环**（可配置，默认5次）：
  - 根据测试失败信息调用 Claude Code 进行修复
  - 失败时自动添加错误信息到下一轮提示词
  - 超过重试次数后退出 doing 进程，由人工干预
- [ ] 成功后更新 tasks.json 状态为 done

#### 3.2.5 问题记录系统
- [ ] 解析测试失败信息
- [ ] 追加到 debug.md（按 debug1、debug2... 编号）
- [ ] 在下一轮执行时加载 debug.md 作为上下文

#### 3.2.6 Git 集成
- [ ] 初始化 .git（如不存在）
- [ ] 每个 task 完成后自动 commit
- [ ] 通过 `git log` 查看执行历史
- [ ] 通过 `git checkout` 进行版本回滚

#### 3.2.7 配置管理（简化）
- [ ] 支持 ~/.rick/config.json（全局配置）
- [ ] 简单 JSON 格式，无复杂层级
- [ ] 仅包含必要配置项

#### 3.2.8 提示词管理模块（新增）⭐
- [ ] 提示词模板目录：`internal/prompt/templates/`
- [ ] 提示词构建器：动态组装提示词
- [ ] 上下文管理：加载 task.md、debug.md、OKR.md、SPEC.md
- [ ] 模板变量替换：支持 {{variable}} 形式的变量替换

---

## 4. 技术栈选择建议

### 4.1 核心依赖（最小化）

| 功能 | 推荐库 | 原因 |
|------|--------|------|
| CLI框架 | `cobra` | 业界标准，功能完整 |
| JSON处理 | `encoding/json` | 标准库，无外部依赖 |
| Markdown解析 | `goldmark` | 功能完整，活跃维护 |
| DAG/拓扑排序 | 自实现 | 简单场景自实现，无需外部依赖 |
| 日志 | Go 标准库 `log` | 最小化，仅文本输出 |
| 配置管理 | `encoding/json` | 直接使用标准库，无需 viper |
| 文件操作 | `os`/`io` | 标准库足够 |
| Git操作 | `go-git` | 纯Go实现，无需外部依赖 |

**依赖原则**：最小化外部依赖，优先使用 Go 标准库。

### 4.2 Rick 项目结构（简化设计）

```
rick/
├── cmd/
│   └── rick/
│       └── main.go              # 入口点，命令路由
├── internal/
│   ├── cmd/                     # 命令处理器（仅保留核心）
│   │   ├── init.go              # 初始化项目
│   │   ├── plan.go              # 规划阶段
│   │   ├── doing.go             # 执行阶段
│   │   └── learning.go          # 学习阶段
│   ├── config/                  # 配置管理（简化）
│   │   ├── config.go            # 单一配置结构
│   │   └── loader.go            # 从 ~/.rick/config.json 加载
│   ├── workspace/               # 工作空间管理
│   │   ├── workspace.go         # 工作空间操作
│   │   └── paths.go             # 路径常量
│   ├── parser/                  # 内容解析
│   │   ├── task.go              # task.md 解析
│   │   ├── debug.go             # debug.md 解析与追加
│   │   └── markdown.go          # 基础 Markdown 解析
│   ├── executor/                # 任务执行引擎
│   │   ├── executor.go          # 执行协调器
│   │   ├── dag.go               # DAG 构建与拓扑排序
│   │   └── runner.go            # 单个任务执行
│   ├── prompt/                  # ⭐ 提示词管理模块（新增）
│   │   ├── manager.go           # 提示词管理器
│   │   ├── builder.go           # 提示词构建器
│   │   └── templates/           # 提示词模板目录
│   │       ├── plan.md
│   │       ├── doing.md
│   │       ├── test.md
│   │       └── learning.md
│   ├── git/                     # Git 集成
│   │   ├── git.go               # Git 操作
│   │   └── commit.go            # 自动提交
│   └── callcli/                 # Claude Code CLI 交互
│       └── caller.go            # 直接参考 Morty 实现
├── scripts/                     # ⭐ 安装脚本（新增）
│   ├── build.sh                 # 构建脚本
│   ├── install.sh               # 安装脚本（支持 --dev 模式）
│   ├── uninstall.sh             # 卸载脚本
│   └── update.sh                # 更新脚本
├── pkg/
│   └── errors/                  # 错误定义
│       └── errors.go
├── go.mod
├── go.sum
├── README.md
└── Makefile
```

---

## 5. 版本管理与安装机制 ⭐

### 5.1 设计目标
- **灵活安装**：支持源码安装和二进制安装
- **开发友好**：支持 dev 版本与生产版本并行运行
- **易于更新**：统一的更新机制
- **自我重构**：使用旧版本 rick 重构新版本 rick

### 5.2 安装目录结构

```
~/.rick/                        # 生产版本
├── bin/
│   └── rick                    # 生产命令
├── config.json                 # 全局配置
└── ...

~/.rick_dev/                    # 开发版本
├── bin/
│   └── rick_dev                # 开发命令
├── config.json                 # 开发配置
└── ...
```

### 5.3 脚本说明

#### build.sh（构建脚本）
```bash
./build.sh [--output <path>]
# 构建二进制文件
# --output: 指定输出路径（默认 ./bin/rick）
```

#### install.sh（安装脚本）
```bash
# 源码安装（生产版）
./install.sh --source [--prefix ~/.rick]

# 源码安装（开发版）
./install.sh --source --dev [--prefix ~/.rick_dev]

# 二进制安装（生产版）
./install.sh --binary [--version v1.0.0]

# 二进制安装（开发版）
./install.sh --binary --dev [--version v1.0.0-dev]

# 默认行为：源码安装到 ~/.rick
./install.sh
```

**安装逻辑**：
1. 源码安装：调用 build.sh 构建，然后复制到目标路径
2. 二进制安装：从 GitHub releases 下载二进制包
3. 创建命令符号链接（rick 或 rick_dev）
4. 更新 PATH 环境变量

#### uninstall.sh（卸载脚本）
```bash
# 卸载生产版本
./uninstall.sh

# 卸载开发版本
./uninstall.sh --dev

# 卸载所有版本
./uninstall.sh --all
```

#### update.sh（更新脚本）
```bash
# 更新生产版本
./update.sh [--version v1.0.0]

# 更新开发版本
./update.sh --dev [--version v1.0.0-dev]

# 默认：更新到最新版本
./update.sh
```

**更新逻辑**：
1. 调用 uninstall.sh 卸载旧版本
2. 调用 install.sh 安装新版本

### 5.4 开发工作流示例

```bash
# 初始状态：只有生产版本 rick
rick --version  # v1.0.0

# 开发人员准备开发环境
./install.sh --source --dev
rick_dev --version  # v1.0.0

# 使用 rick（生产版）来重构 rick
rick plan "重构Rick CLI架构"
rick doing job_1
rick learning job_1

# 使用 rick_dev（开发版）来测试新功能
rick_dev plan "测试新功能"
rick_dev doing job_2

# 测试完成后，更新生产版本
./update.sh

# 清理开发版本
./uninstall.sh --dev
```

### 5.5 版本管理最佳实践

**开发规范**：
1. **版本号格式**：`vMAJOR.MINOR.PATCH[-dev]`
   - 例如：`v1.0.0`、`v1.1.0-dev`

2. **生产版本**：
   - 安装到 `~/.rick/`
   - 命令名：`rick`
   - 用于生产环境和自我重构

3. **开发版本**：
   - 安装到 `~/.rick_dev/`
   - 命令名：`rick_dev`
   - 用于新功能开发和测试

4. **并行运行**：
   - 两个版本可同时运行
   - 使用不同的配置文件
   - 使用不同的工作空间（可选）

5. **更新流程**：
   ```
   开发 → 测试（rick_dev）→ 集成测试 → 更新生产版（rick）
   ```

---

## 6. 核心算法与设计

### 6.1 DAG 构建算法

```go
type Task struct {
    ID           string
    Name         string
    Dependencies []string
    Status       string
}

type DAG struct {
    tasks map[string]*Task
    graph map[string][]string  // task_id -> [dependent_ids]
}

// 拓扑排序（Kahn 算法）
func (d *DAG) TopologicalSort() ([]string, error) {
    // 1. 计算每个节点的入度
    inDegree := make(map[string]int)
    for id := range d.tasks {
        inDegree[id] = 0
    }
    for id, deps := range d.graph {
        for _, dep := range deps {
            inDegree[id]++
        }
    }

    // 2. 找出所有入度为0的节点
    queue := []string{}
    for id, degree := range inDegree {
        if degree == 0 {
            queue = append(queue, id)
        }
    }

    // 3. 执行拓扑排序
    result := []string{}
    for len(queue) > 0 {
        node := queue[0]
        queue = queue[1:]
        result = append(result, node)

        for _, dependent := range d.graph[node] {
            inDegree[dependent]--
            if inDegree[dependent] == 0 {
                queue = append(queue, dependent)
            }
        }
    }

    // 4. 检测循环依赖
    if len(result) != len(d.tasks) {
        return nil, errors.New("循环依赖检测失败")
    }

    return result, nil
}
```

### 6.2 任务执行循环（核心）

```go
type ExecutionConfig struct {
    MaxRetries int  // 默认5次
}

func (e *Executor) ExecuteJob(jobID string, cfg ExecutionConfig) error {
    // 1. 加载 tasks.json
    tasks := e.LoadTasks(jobID)

    // 2. 串行执行（按拓扑排序顺序）
    for _, taskID := range tasks {
        task := e.GetTask(taskID)
        if task.Status != "pending" {
            continue
        }

        // 3. 执行单个任务的重试循环
        retryCount := 0
        for retryCount < cfg.MaxRetries {
            // 4. 生成测试脚本
            testScript := e.GenerateTestScript(task)

            // 5. 构建提示词
            prompt := e.BuildPrompt(task, retryCount)

            // 6. 调用 Claude Code 执行任务
            result := e.CallClaudeCode(prompt)

            // 7. 运行测试脚本
            testResult := e.RunTest(testScript)

            if testResult.Pass {
                // 8. 提交变更
                e.CommitTask(jobID, taskID)
                e.UpdateTaskStatus(jobID, taskID, "done")
                break
            }

            // 9. 失败处理
            retryCount++
            if retryCount < cfg.MaxRetries {
                // 记录问题到 debug.md
                e.AppendDebug(jobID, testResult.Error)
                // 继续重试
            } else {
                // 超过重试次数，退出 doing 进程
                return fmt.Errorf("任务 %s 失败次数超过限制(%d)，请人工干预",
                    taskID, cfg.MaxRetries)
            }
        }
    }

    return nil
}

// 构建提示词（包含历史失败信息）
func (e *Executor) BuildPrompt(task *Task, retryCount int) string {
    // 1. 加载基础提示词模板
    // 2. 加载 task.md
    // 3. 加载 debug.md（如果存在）
    // 4. 加载 OKR.md、SPEC.md（如果存在）
    // 5. 如果是重试，添加上一次的失败信息
    // 6. 组装最终提示词
}
```

### 6.3 Markdown 解析规则

```go
type TaskMarkdown struct {
    Dependencies []string
    Name         string
    Goal         string
    KeyResults   []string
    TestMethod   string
}

// 解析规则（从 task.md）
// # 依赖关系 -> Dependencies (逗号分隔)
// # 任务名称 -> Name
// # 任务目标 -> Goal
// # 关键结果 -> KeyResults (Markdown 列表项)
// # 测试方法 -> TestMethod (Markdown 列表项)

// 示例 task.md：
/*
# 依赖关系
task1, task2

# 任务名称
创建 server.go 源文件

# 任务目标
基于gRPC框架，完成gRPC Server的搭建工作

# 关键结果
1. 检查是否已安装gRPC最新版本
2. 检查Go版本是否满足最新gRPC要求
3. 查阅gRPC最新版本Server构建文档

# 测试方法
1. 学习gRPC服务端单元测试规范
2. 编写单元测试：启动Server，发起RPC请求
3. 运行单元测试：`go test -v ./...`
*/
```

---

## 7. 实现阶段规划

### Phase 1: 基础设施（Week 1）
- [ ] 项目初始化（go mod, cobra CLI框架）
- [ ] 基础命令路由（init, plan, doing, learning）
- [ ] 工作空间管理（.rick 目录创建与维护）
- [ ] 配置系统（简化版，仅 ~/.rick/config.json）
- [ ] 日志系统（标准库 log，文本格式）

### Phase 2: 核心解析（Week 1-2）
- [ ] Markdown 解析（task.md 格式解析）
- [ ] DAG 构建与拓扑排序
- [ ] tasks.json 生成与加载
- [ ] debug.md 解析与追加
- [ ] 提示词管理模块（builder + templates）

### Phase 3: 执行引擎（Week 2-3）
- [ ] Claude Code CLI 集成（参考 Morty）
- [ ] 测试脚本生成与执行
- [ ] 任务执行循环（串行 + 重试）
- [ ] 失败重试机制（可配置次数）

### Phase 4: Git 与提交（Week 3）
- [ ] Git 初始化与操作
- [ ] 自动提交每个 task
- [ ] 使用 git log 查看历史
- [ ] 使用 git checkout 回滚

### Phase 5: 安装机制（Week 3-4）
- [ ] build.sh 脚本
- [ ] install.sh 脚本（源码 + 二进制，支持 --dev）
- [ ] uninstall.sh 脚本
- [ ] update.sh 脚本
- [ ] 环境变量设置

### Phase 6: Learning 阶段（Week 4）
- [ ] 自动化 Learning（init 模式）
- [ ] 人工审核 Learning（learning 模式）
- [ ] OKR/SPEC/Wiki 更新

### Phase 7: 测试与文档（Week 4-5）
- [ ] 单元测试编写
- [ ] 集成测试编写
- [ ] 文档完善
- [ ] 开发规范文档

---

## 8. 潜在挑战与解决方案

### 8.1 上下文窗口限制
**问题**: task.md + debug.md + OKR.md + SPEC.md 可能超过 Claude 的上下文限制

**解决方案**:
- 分块加载（只加载相关部分）
- 使用摘要而非全文
- 实现上下文管理系统（在提示词 builder 中）

### 8.2 Claude Code CLI 集成
**问题**: 需要可靠地调用 Claude Code 并解析输出

**解决方案**:
- 直接参考 Morty 的 callcli 实现
- 使用标准化的提示词模板
- 添加重试机制

### 8.3 DAG 循环依赖检测
**问题**: task.md 中的依赖关系可能形成循环

**解决方案**:
- 在生成 tasks.json 时检测循环（拓扑排序中检测）
- 提供清晰的错误提示
- 要求用户修正 task.md

### 8.4 失败重试与人工干预
**问题**: 任务失败后如何有效地重试和人工干预

**解决方案**:
- 配置失败重试次数（默认5次）
- 超过限制后退出 doing 进程
- 由人工修改 task.md 后重新运行 `rick doing job_n`
- debug.md 记录所有失败信息供参考

### 8.5 Dev 版本与生产版本管理
**问题**: 开发过程中需要两个版本并行运行

**解决方案**:
- 使用不同的安装路径（~/.rick vs ~/.rick_dev）
- 使用不同的命令名（rick vs rick_dev）
- 支持 --dev 标志来区分版本
- 脚本自动处理路径和命令名映射

---

## 9. 相关资源

### 9.1 官方文档
- [Go 官方文档](https://golang.org/doc/)
- [Cobra CLI 框架](https://github.com/spf13/cobra)
- [Go-Git 库](https://github.com/go-git/go-git)
- [Goldmark Markdown 解析](https://github.com/yuin/goldmark)

### 9.2 参考实现
- [Morty 项目](file:///Users/sunquan/ai_coding/CODING/morty/)
  - 特别参考：callcli 包的 Claude Code 集成
  - 参考：executor 包的任务执行逻辑
- [Rick 规范](file:///Users/sunquan/ai_coding/CODING/rick/Rick_Project_Complete_Description.md)

### 9.3 相关技术
- DAG 和拓扑排序：[Kahn 算法](https://en.wikipedia.org/wiki/Topological_sorting#Kahn%27s_algorithm)
- Markdown 解析：[Goldmark 文档](https://pkg.go.dev/github.com/yuin/goldmark)
- Shell 脚本最佳实践：[Google Shell 风格指南](https://google.github.io/styleguide/shellguide.html)

---

## 10. 开发规范文档 ⭐

### 10.1 版本管理规范

**版本号格式**
```
vMAJOR.MINOR.PATCH[-dev]
例如：v1.0.0, v1.1.0-dev, v2.0.0-beta
```

**版本类型**
| 版本 | 安装路径 | 命令名 | 用途 | 配置路径 |
|------|---------|--------|------|---------|
| 生产版 | ~/.rick | rick | 生产环境、自我重构 | ~/.rick/config.json |
| 开发版 | ~/.rick_dev | rick_dev | 新功能开发、测试 | ~/.rick_dev/config.json |

### 10.2 安装脚本规范

**build.sh**
```bash
#!/bin/bash
# 用途：构建二进制文件
# 用法：./build.sh [--output <path>]
# 默认输出：./bin/rick
# 环境变量：GO_VERSION（指定Go版本）

# 步骤：
# 1. 检查 Go 环境
# 2. 执行 go build
# 3. 输出二进制文件
```

**install.sh**
```bash
#!/bin/bash
# 用途：安装 Rick CLI
# 用法：
#   ./install.sh                          # 源码安装到 ~/.rick
#   ./install.sh --source                 # 源码安装到 ~/.rick
#   ./install.sh --source --dev           # 源码安装到 ~/.rick_dev
#   ./install.sh --binary                 # 二进制安装（Linux only）
#   ./install.sh --binary --dev           # 二进制安装 dev 版

# 步骤：
# 1. 解析参数（--source, --binary, --dev）
# 2. 根据参数调用 build.sh 或下载二进制
# 3. 复制文件到目标路径
# 4. 创建符号链接
# 5. 更新 PATH（如需要）
# 6. 验证安装
```

**uninstall.sh**
```bash
#!/bin/bash
# 用途：卸载 Rick CLI
# 用法：
#   ./uninstall.sh                        # 卸载生产版
#   ./uninstall.sh --dev                  # 卸载开发版
#   ./uninstall.sh --all                  # 卸载所有版本

# 步骤：
# 1. 解析参数
# 2. 删除安装目录
# 3. 删除符号链接
# 4. 清理 PATH（如需要）
```

**update.sh**
```bash
#!/bin/bash
# 用途：更新 Rick CLI
# 用法：
#   ./update.sh                           # 更新到最新版本
#   ./update.sh --dev                     # 更新 dev 版本
#   ./update.sh --version v1.0.0          # 更新到指定版本

# 步骤：
# 1. 调用 uninstall.sh
# 2. 调用 install.sh
```

### 10.3 开发工作流规范

**场景1：新功能开发**
```bash
# 1. 安装 dev 版本
./install.sh --source --dev

# 2. 使用 dev 版本开发和测试
rick_dev plan "新功能描述"
rick_dev doing job_1

# 3. 测试完成后，使用生产版本重构
rick plan "集成新功能"
rick doing job_2

# 4. 卸载 dev 版本
./uninstall.sh --dev
```

**场景2：Bug 修复**
```bash
# 1. 安装 dev 版本
./install.sh --source --dev

# 2. 在 dev 版本中修复 bug
rick_dev plan "修复 Bug"
rick_dev doing job_1

# 3. 测试完成后，更新生产版本
./update.sh

# 4. 验证修复
rick plan "验证修复"
rick doing job_2
```

**场景3：自我重构**
```bash
# 1. 使用生产版本规划重构
rick plan "重构 Rick 架构"

# 2. 安装 dev 版本作为新实现
./install.sh --source --dev

# 3. 使用生产版本执行重构任务
rick doing job_1

# 4. 验证新实现
rick_dev plan "验证新实现"
rick_dev doing job_2

# 5. 更新生产版本
./update.sh

# 6. 清理 dev 版本
./uninstall.sh --dev
```

### 10.4 代码组织规范

**包结构**
```
cmd/rick/main.go                # 入口点，仅包含 main() 函数
internal/
  cmd/                          # 命令处理器
    init.go                     # init 命令实现
    plan.go                     # plan 命令实现
    doing.go                    # doing 命令实现
    learning.go                 # learning 命令实现
  config/                       # 配置管理
    config.go                   # 配置结构体和加载逻辑
    loader.go                   # 从 JSON 文件加载
  workspace/                    # 工作空间管理
    workspace.go                # 工作空间操作
    paths.go                    # 路径常量
  parser/                       # 内容解析
    task.go                     # task.md 解析
    debug.go                    # debug.md 处理
    markdown.go                 # 基础 Markdown 解析
  executor/                     # 任务执行
    executor.go                 # 执行协调器
    dag.go                      # DAG 构建和拓扑排序
    runner.go                   # 单个任务执行
  prompt/                       # 提示词管理
    manager.go                  # 提示词管理器
    builder.go                  # 提示词构建器
  git/                          # Git 操作
    git.go                      # Git 命令封装
    commit.go                   # 提交逻辑
  callcli/                      # Claude Code CLI 交互
    caller.go                   # 调用 Claude Code CLI
pkg/
  errors/                       # 错误定义
    errors.go                   # 自定义错误类型
```

**命名规范**
- 包名：小写，单词，不使用下划线
- 函数名：大写开头（导出），驼峰式
- 变量名：驼峰式
- 常量名：大写，单词间用下划线分隔

---

## 11. 下一步行动

### 当前进度
- [x] 项目概述与核心理论理解
- [x] Morty 参考实现分析
- [x] 技术栈选择（最小化）
- [x] 项目结构设计（简化版）
- [x] 版本管理机制设计
- [x] 开发规范文档
- [ ] 开始实现 Phase 1

### 建议的下一步
1. **创建项目初始化**：`go mod init github.com/your/rick`
2. **搭建基础框架**：Cobra CLI 框架
3. **实现命令路由**：init, plan, doing, learning
4. **开始 Phase 1 实现**

---

**文档版本**: 2.0
**研究完成时间**: 2026-03-13
**状态**: 完成
**探索子代理使用**: 是
**关键更新**:
- 简化了日志系统、配置管理
- 移除了 status/reset 命令
- 新增了独立的提示词管理模块
- 详细设计了版本管理和安装机制
- 添加了完整的开发规范文档
