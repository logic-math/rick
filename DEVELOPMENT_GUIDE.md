# Rick CLI 开发指南

## 版本管理与安装规范

### 版本定义

Rick 支持两个并行版本，便于自我重构：

| 版本 | 安装路径 | 命令名 | 配置文件 | 用途 |
|------|---------|--------|---------|------|
| **生产版** | `~/.rick/` | `rick` | `~/.rick/config.json` | 生产环境、自我重构 |
| **开发版** | `~/.rick_dev/` | `rick_dev` | `~/.rick_dev/config.json` | 新功能开发、测试 |

### 版本号格式

```
vMAJOR.MINOR.PATCH[-dev|-beta|-rc]
示例：v1.0.0, v1.1.0-dev, v2.0.0-beta
```

---

## 安装脚本使用规范

### build.sh - 构建二进制

**用途**: 编译 Rick CLI

**用法**:
```bash
./build.sh [--output <path>]
```

**参数**:
- `--output <path>`: 指定输出路径（默认 `./bin/rick`）

**示例**:
```bash
./build.sh                          # 输出到 ./bin/rick
./build.sh --output /tmp/rick       # 输出到 /tmp/rick
```

**实现步骤**:
1. 检查 Go 环境（版本 >= 1.21）
2. 执行 `go build` 命令
3. 验证二进制文件
4. 输出成功消息

---

### install.sh - 安装 Rick

**用途**: 安装 Rick CLI 到系统

**用法**:
```bash
./install.sh [OPTIONS]
```

**选项**:
- `--source`: 源码安装（默认）
- `--binary`: 二进制安装（Linux only）
- `--dev`: 安装开发版本（默认安装生产版本）
- `--prefix <path>`: 自定义安装路径（源码安装时）
- `--version <version>`: 指定版本号（二进制安装时）

**示例**:
```bash
# 源码安装生产版本到 ~/.rick
./install.sh

# 源码安装生产版本到自定义路径
./install.sh --source --prefix /opt/rick

# 源码安装开发版本到 ~/.rick_dev
./install.sh --source --dev

# 二进制安装生产版本
./install.sh --binary

# 二进制安装开发版本（从 GitHub releases 下载）
./install.sh --binary --dev --version v1.0.0
```

**实现步骤**:
1. 解析参数
2. 根据模式调用 build.sh 或下载二进制
3. 创建目标目录
4. 复制二进制文件
5. 创建符号链接到 `/usr/local/bin` 或 `~/.local/bin`
6. 验证安装（运行 `rick --version` 或 `rick_dev --version`）
7. 提示用户更新 PATH（如需要）

---

### uninstall.sh - 卸载 Rick

**用途**: 卸载已安装的 Rick CLI

**用法**:
```bash
./uninstall.sh [OPTIONS]
```

**选项**:
- `--dev`: 卸载开发版本（默认卸载生产版本）
- `--all`: 卸载所有版本

**示例**:
```bash
# 卸载生产版本
./uninstall.sh

# 卸载开发版本
./uninstall.sh --dev

# 卸载所有版本
./uninstall.sh --all
```

**实现步骤**:
1. 解析参数
2. 确认卸载操作
3. 删除安装目录
4. 删除符号链接
5. 清理 PATH（如需要）
6. 验证卸载成功

---

### update.sh - 更新 Rick

**用途**: 更新已安装的 Rick CLI 到新版本

**用法**:
```bash
./update.sh [OPTIONS]
```

**选项**:
- `--dev`: 更新开发版本（默认更新生产版本）
- `--version <version>`: 指定更新到的版本（默认最新版本）
- `--source`: 从源码更新（默认）
- `--binary`: 从二进制更新

**示例**:
```bash
# 更新生产版本到最新版本
./update.sh

# 更新生产版本到指定版本
./update.sh --version v1.1.0

# 更新开发版本
./update.sh --dev

# 从二进制更新生产版本
./update.sh --binary
```

**实现步骤**:
1. 解析参数
2. 调用 `uninstall.sh` 卸载旧版本
3. 调用 `install.sh` 安装新版本
4. 验证更新成功

---

## 开发工作流

### 场景1：开发新功能

```bash
# 1. 安装开发版本
./install.sh --source --dev

# 2. 使用开发版本规划新功能
rick_dev plan "实现新功能：支持并行执行任务"

# 3. 使用开发版本执行任务
rick_dev doing job_1

# 4. 测试完成后，使用生产版本进行集成测试
rick plan "集成新功能到主分支"
rick doing job_2

# 5. 验证集成成功后，卸载开发版本
./uninstall.sh --dev
```

### 场景2：修复 Bug

```bash
# 1. 安装开发版本
./install.sh --source --dev

# 2. 使用开发版本规划 Bug 修复
rick_dev plan "修复 Bug：DAG 拓扑排序中的循环检测"

# 3. 执行修复任务
rick_dev doing job_1

# 4. 验证修复后，更新生产版本
./update.sh

# 5. 使用生产版本验证修复
rick plan "验证 Bug 修复"
rick doing job_2

# 6. 卸载开发版本
./uninstall.sh --dev
```

### 场景3：自我重构

```bash
# 1. 使用生产版本规划重构
rick plan "重构 Rick CLI 架构：简化日志系统"

# 2. 安装开发版本作为新实现
./install.sh --source --dev

# 3. 使用生产版本执行重构任务
rick doing job_1

# 4. 验证新实现
rick_dev plan "验证重构后的架构"
rick_dev doing job_2

# 5. 重构完成后，更新生产版本
./update.sh

# 6. 清理开发版本
./uninstall.sh --dev
```

### 场景4：版本回滚

```bash
# 查看 git 历史
git log --oneline

# 回滚到指定版本
git checkout <commit-hash>

# 重新构建和安装
./build.sh
./install.sh --source
```

---

## 代码组织规范

### 包结构

```
rick/
├── cmd/
│   └── rick/
│       └── main.go              # 入口点，仅包含 main() 函数和基础命令路由
├── internal/
│   ├── cmd/                     # 命令处理器
│   │   ├── init.go              # init 命令实现
│   │   ├── plan.go              # plan 命令实现
│   │   ├── doing.go             # doing 命令实现
│   │   └── learning.go          # learning 命令实现
│   ├── config/                  # 配置管理
│   │   ├── config.go            # 配置结构体和加载逻辑
│   │   └── loader.go            # 从 JSON 文件加载
│   ├── workspace/               # 工作空间管理
│   │   ├── workspace.go         # 工作空间操作
│   │   └── paths.go             # 路径常量
│   ├── parser/                  # 内容解析
│   │   ├── task.go              # task.md 解析
│   │   ├── debug.go             # debug.md 处理
│   │   └── markdown.go          # 基础 Markdown 解析
│   ├── executor/                # 任务执行引擎
│   │   ├── executor.go          # 执行协调器
│   │   ├── dag.go               # DAG 构建和拓扑排序
│   │   └── runner.go            # 单个任务执行
│   ├── prompt/                  # ⭐ 提示词管理模块
│   │   ├── manager.go           # 提示词管理器
│   │   ├── builder.go           # 提示词构建器
│   │   └── templates/           # 提示词模板目录
│   │       ├── plan.md
│   │       ├── doing.md
│   │       ├── test.md
│   │       └── learning.md
│   ├── git/                     # Git 操作
│   │   ├── git.go               # Git 命令封装
│   │   └── commit.go            # 提交逻辑
│   ├── callcli/                 # Claude Code CLI 交互
│   │   └── caller.go            # 调用 Claude Code CLI
│   └── logging/                 # 日志系统
│       └── logger.go            # 简单日志封装
├── pkg/
│   └── errors/                  # 错误定义
│       └── errors.go            # 自定义错误类型
├── scripts/
│   ├── build.sh
│   ├── install.sh
│   ├── uninstall.sh
│   └── update.sh
├── go.mod
├── go.sum
└── README.md
```

### 命名规范

**包名**:
- 小写，单词，不使用下划线
- 例如：`config`, `parser`, `executor`, `callcli`

**函数名**:
- 大写开头（导出），驼峰式
- 例如：`LoadConfig`, `ParseTask`, `ExecuteJob`

**变量名**:
- 驼峰式
- 短变量名用于局部变量（如 `err`, `cfg`）
- 长变量名用于全局变量和结构体字段

**常量名**:
- 大写，单词间用下划线分隔
- 例如：`DEFAULT_RETRY_COUNT`, `CONFIG_FILE_NAME`

### 错误处理

- 所有错误使用 `pkg/errors` 中定义的自定义错误类型
- 函数返回 `error` 接口，调用者负责处理
- 不要忽略错误，总是检查并适当处理

### 日志记录

- 使用 Go 标准库 `log` 包
- 仅输出文本格式，无 JSON
- 日志级别：INFO, WARN, ERROR（通过前缀区分）
- 示例：
  ```go
  log.Printf("[INFO] 任务 %s 启动", taskID)
  log.Printf("[WARN] 任务 %s 失败，重试 %d/%d", taskID, retry, maxRetries)
  log.Printf("[ERROR] 任务 %s 失败: %v", taskID, err)
  ```

---

## 测试规范

### 单元测试

- 所有包都应有对应的 `*_test.go` 文件
- 测试函数命名：`Test<FunctionName>`
- 使用 Go 标准库 `testing` 包
- 示例：
  ```go
  func TestParseTask(t *testing.T) {
      // 测试代码
  }
  ```

### 集成测试

- 在 `tests/` 目录下编写集成测试脚本
- 使用 shell 脚本测试 CLI 命令
- 示例：`tests/test_plan_command.sh`

### 运行测试

```bash
# 运行所有单元测试
go test ./...

# 运行特定包的测试
go test ./internal/parser/...

# 运行测试并显示覆盖率
go test -cover ./...
```

---

## 提交规范

### 提交消息格式

```
<type>(<scope>): <subject>

<body>

<footer>
```

**type**:
- `feat`: 新功能
- `fix`: 修复 Bug
- `refactor`: 代码重构
- `docs`: 文档更新
- `test`: 测试相关
- `chore`: 构建相关

**scope**: 影响范围（可选），如 `parser`, `executor`, `config`

**subject**: 简短描述（不超过50字符）

**body**: 详细描述（可选）

**footer**: 关联的 issue（可选），如 `Closes #123`

### 示例

```
feat(executor): 实现任务执行循环和重试机制

- 添加 ExecutionConfig 结构体，支持可配置的重试次数
- 实现 ExecuteJob 方法，支持串行执行和重试
- 失败超过限制后退出进程，由人工干预

Closes #1
```

---

## 常见问题

### Q1: 如何在生产版和开发版之间切换？

```bash
# 使用生产版本
rick plan "..."

# 使用开发版本
rick_dev plan "..."

# 两个版本可以同时运行
```

### Q2: 如何更新到新版本？

```bash
# 从源码更新
./update.sh

# 从二进制更新
./update.sh --binary

# 更新到指定版本
./update.sh --version v1.1.0
```

### Q3: 如何卸载 Rick？

```bash
# 卸载生产版本
./uninstall.sh

# 卸载开发版本
./uninstall.sh --dev

# 卸载所有版本
./uninstall.sh --all
```

### Q4: 如何处理任务执行失败？

1. 检查 `debug.md` 中的错误信息
2. 修改 `plan/tasks/task*.md` 中的任务定义
3. 重新运行 `rick doing job_n`
4. 如果仍然失败，查看日志并联系开发者

### Q5: 如何使用 rick 重构 rick？

```bash
# 1. 使用生产版本规划重构
rick plan "重构 Rick CLI 架构"

# 2. 安装开发版本
./install.sh --source --dev

# 3. 执行重构任务
rick doing job_1

# 4. 验证新实现
rick_dev plan "验证重构"
rick_dev doing job_2

# 5. 更新生产版本
./update.sh

# 6. 清理开发版本
./uninstall.sh --dev
```

---

## 参考资源

- [Rick 项目规范](./Rick_Project_Complete_Description.md)
- [研究报告](./research/使用_golang_开发_rick_命令行程序.md)
- [Morty 参考实现](../morty/)
- [Go 官方文档](https://golang.org/doc/)
- [Cobra CLI 框架](https://github.com/spf13/cobra)

---

**最后更新**: 2026-03-13
**维护者**: Rick 开发团队
