# Rick CLI 安装与部署指南

本文档全面介绍 Rick CLI 的安装、配置和部署方法，帮助用户快速上手使用 Rick CLI。

## 目录

- [系统要求](#系统要求)
- [安装方法](#安装方法)
  - [源码安装](#源码安装)
  - [二进制安装](#二进制安装)
- [脚本详解](#脚本详解)
  - [build.sh](#buildsh)
  - [install.sh](#installsh)
  - [uninstall.sh](#uninstallsh)
  - [update.sh](#updatesh)
- [配置文件](#配置文件)
- [版本管理](#版本管理)
  - [生产版本](#生产版本)
  - [开发版本](#开发版本)
- [自我重构工作流](#自我重构工作流)
- [常见问题](#常见问题)

---

## 系统要求

在安装 Rick CLI 之前，请确保您的系统满足以下要求：

### 必需依赖

1. **Go 语言环境**
   - 版本要求：Go 1.21 或更高版本
   - 检查方法：`go version`
   - 安装指南：https://golang.org/doc/install

2. **Claude Code CLI**
   - Rick CLI 依赖 Claude Code CLI 来执行 AI 辅助编程任务
   - 确保 `claude` 命令在 PATH 中可用
   - 检查方法：`which claude` 或 `claude --version`
   - 安装指南：参考 Claude Code 官方文档

3. **Python 3**
   - 版本要求：Python 3.6 或更高版本
   - 用途：生成和执行任务测试脚本
   - 检查方法：`python3 --version`

4. **Git**
   - 版本要求：Git 2.0 或更高版本
   - 用途：版本控制和任务状态管理
   - 检查方法：`git --version`

### 可选依赖

1. **curl**
   - 用途：二进制安装时下载预编译二进制文件
   - 检查方法：`curl --version`

2. **jq**
   - 用途：解析 JSON 配置文件（可选）
   - 检查方法：`jq --version`

### 系统兼容性

- **操作系统**：Linux, macOS, Windows (WSL)
- **架构**：amd64 (x86_64), arm64 (aarch64)
- **Shell**：bash, zsh

---

## 安装方法

Rick CLI 提供两种安装方式：源码安装和二进制安装。

### 源码安装

源码安装适合开发者和需要自定义构建的用户。

#### 步骤 1：克隆仓库

```bash
git clone https://github.com/anthropics/rick.git
cd rick
```

#### 步骤 2：运行安装脚本

**安装生产版本**（推荐）：

```bash
./scripts/install.sh
```

安装完成后，Rick CLI 将被安装到 `~/.rick` 目录，命令名为 `rick`。

**安装开发版本**：

```bash
./scripts/install.sh --dev
```

安装完成后，Rick CLI 将被安装到 `~/.rick_dev` 目录，命令名为 `rick_dev`。

#### 步骤 3：配置 PATH

安装脚本会自动创建符号链接到 `~/.local/bin/rick`。确保该目录在您的 PATH 中：

```bash
# 添加到 ~/.bashrc 或 ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"

# 重新加载配置
source ~/.bashrc  # 或 source ~/.zshrc
```

#### 步骤 4：验证安装

```bash
rick --version
rick --help
```

### 二进制安装

二进制安装适合不需要 Go 环境的普通用户（仅 Linux 系统）。

#### 步骤 1：下载并运行安装脚本

```bash
curl -fsSL https://raw.githubusercontent.com/anthropics/rick/main/scripts/install.sh | bash -s -- --binary
```

或者手动下载安装脚本：

```bash
wget https://raw.githubusercontent.com/anthropics/rick/main/scripts/install.sh
chmod +x install.sh
./install.sh --binary
```

#### 步骤 2：配置 PATH

与源码安装相同，确保 `~/.local/bin` 在您的 PATH 中。

#### 步骤 3：验证安装

```bash
rick --version
rick --help
```

### 自定义安装路径

如果您希望将 Rick CLI 安装到自定义目录：

```bash
./scripts/install.sh --prefix /path/to/custom/directory
```

然后手动将 `/path/to/custom/directory/bin` 添加到 PATH：

```bash
export PATH="/path/to/custom/directory/bin:$PATH"
```

---

## 脚本详解

Rick CLI 提供了一套完整的脚本来管理构建、安装、卸载和更新。

### build.sh

**用途**：从源码构建 Rick CLI 二进制文件。

**位置**：`scripts/build.sh`

**用法**：

```bash
./scripts/build.sh [OPTIONS]
```

**选项**：

- `--output OUTPUT_PATH`：指定输出二进制文件的路径（默认：`./bin/rick`）
- `-h, --help`：显示帮助信息

**示例**：

```bash
# 构建到默认位置（./bin/rick）
./scripts/build.sh

# 构建到自定义位置
./scripts/build.sh --output /tmp/rick
```

**工作流程**：

1. **检查 Go 版本**：确保 Go 版本 >= 1.21
2. **创建输出目录**：如果不存在，创建输出目录
3. **编译二进制文件**：使用 `go build` 编译 `cmd/rick` 包
4. **验证二进制文件**：检查文件存在性、可执行性和基本功能

**输出**：

- 成功：在指定路径生成 `rick` 可执行文件
- 失败：输出错误信息并返回非零退出码

**常见错误**：

- `Go is not installed`：未安装 Go 或 Go 不在 PATH 中
- `Go version X.X is not supported`：Go 版本过低，需要升级到 1.21+
- `Build failed`：编译错误，检查源码是否完整

### install.sh

**用途**：安装 Rick CLI 到系统目录。

**位置**：`scripts/install.sh`

**用法**：

```bash
./scripts/install.sh [OPTIONS]
```

**选项**：

- `--source`：从源码安装（默认）
- `--binary`：从 GitHub Releases 下载预编译二进制文件安装
- `--dev`：安装到开发目录（`~/.rick_dev`），命令名为 `rick_dev`
- `--prefix PREFIX`：自定义安装路径
- `--version VERSION`：指定安装版本（默认：latest）
- `-h, --help`：显示帮助信息

**示例**：

```bash
# 源码安装生产版本
./scripts/install.sh

# 源码安装开发版本
./scripts/install.sh --dev

# 二进制安装生产版本
./scripts/install.sh --binary

# 二进制安装开发版本
./scripts/install.sh --binary --dev

# 安装到自定义路径
./scripts/install.sh --prefix /opt/rick

# 安装特定版本
./scripts/install.sh --binary --version 1.0.0
```

**工作流程**：

1. **解析参数**：确定安装模式、安装目录和命令名
2. **检查依赖**：
   - 源码安装：检查 Go 版本
   - 二进制安装：检查 curl 是否可用
3. **构建/下载二进制文件**：
   - 源码安装：调用 `build.sh` 构建二进制文件
   - 二进制安装：从 GitHub Releases 下载对应平台的二进制文件
4. **创建安装目录**：创建 `~/.rick/bin` 或 `~/.rick_dev/bin`
5. **复制二进制文件**：将二进制文件复制到安装目录
6. **创建符号链接**：在 `~/.local/bin` 创建符号链接
7. **验证安装**：运行 `rick --version` 验证安装成功

**安装目录结构**：

```
~/.rick/                    # 生产版本安装目录
├── bin/
│   └── rick                # 二进制文件

~/.rick_dev/                # 开发版本安装目录
├── bin/
│   └── rick                # 二进制文件（命名相同）

~/.local/bin/               # 符号链接目录
├── rick -> ~/.rick/bin/rick
└── rick_dev -> ~/.rick_dev/bin/rick
```

**常见错误**：

- `Go is not installed`：源码安装需要 Go 环境
- `Failed to download binary`：网络问题或版本不存在
- `Command 'rick' not found in PATH`：`~/.local/bin` 不在 PATH 中

### uninstall.sh

**用途**：卸载 Rick CLI。

**位置**：`scripts/uninstall.sh`

**用法**：

```bash
./scripts/uninstall.sh [OPTIONS]
```

**选项**：

- `--dev`：卸载开发版本（`~/.rick_dev`）
- `--all`：卸载生产版本和开发版本
- `--prefix PREFIX`：卸载自定义路径的安装
- `-h, --help`：显示帮助信息

**示例**：

```bash
# 卸载生产版本
./scripts/uninstall.sh

# 卸载开发版本
./scripts/uninstall.sh --dev

# 卸载所有版本
./scripts/uninstall.sh --all

# 卸载自定义路径的安装
./scripts/uninstall.sh --prefix /opt/rick
```

**工作流程**：

1. **确认卸载**：提示用户确认卸载操作（输入 `y` 继续）
2. **删除符号链接**：删除 `~/.local/bin/rick` 或 `~/.local/bin/rick_dev`
3. **删除安装目录**：删除 `~/.rick` 或 `~/.rick_dev` 目录
4. **输出摘要**：显示卸载完成信息

**注意事项**：

- 卸载操作不可逆，请确保已备份重要数据
- 卸载不会删除用户的工作空间（`.rick` 目录）和配置文件（`~/.rick/config.json`）

### update.sh

**用途**：更新 Rick CLI 到最新版本或指定版本。

**位置**：`scripts/update.sh`

**用法**：

```bash
./scripts/update.sh [OPTIONS]
```

**选项**：

- `--dev`：更新开发版本
- `--version VERSION`：更新到指定版本（默认：latest）
- `--prefix PREFIX`：更新自定义路径的安装
- `-h, --help`：显示帮助信息

**示例**：

```bash
# 更新生产版本到最新
./scripts/update.sh

# 更新开发版本到最新
./scripts/update.sh --dev

# 更新到特定版本
./scripts/update.sh --version 1.0.0
```

**工作流程**：

1. **获取当前版本**：运行 `rick --version` 获取当前版本
2. **获取目标版本**：
   - 如果指定 `--version`，使用指定版本
   - 否则，从 GitHub API 获取最新版本
3. **比较版本**：如果当前版本与目标版本相同，提示无需更新
4. **确认更新**：提示用户确认更新操作
5. **备份当前安装**：创建当前安装的临时备份
6. **卸载当前版本**：删除当前安装（不删除配置）
7. **安装新版本**：调用 `install.sh` 安装新版本
8. **验证更新**：运行 `rick --version` 验证更新成功
9. **清理备份**：删除临时备份

**回滚机制**：

如果更新失败，脚本会自动从备份恢复到原来的版本：

```
[ERROR] Installation of new version failed
[INFO] Rolling back to previous version...
[SUCCESS] Rollback completed successfully
```

**常见错误**：

- `Failed to fetch latest version from GitHub`：网络问题或 API 限流
- `Already on version X.X.X. No update needed.`：当前版本已是最新
- `Installation of new version failed`：新版本安装失败，已自动回滚

---

## 配置文件

Rick CLI 使用 JSON 格式的配置文件来管理全局设置。

### 配置文件位置

- **全局配置**：`~/.rick/config.json`
- **示例配置**：项目根目录下的 `config.example.json`

### 配置文件格式

```json
{
  "max_retries": 5,
  "claude_code_path": "",
  "default_workspace": "",
  "git": {
    "user_name": "Your Name",
    "user_email": "your.email@example.com"
  }
}
```

### 配置项说明

#### max_retries

- **类型**：整数
- **默认值**：5
- **说明**：任务执行失败时的最大重试次数
- **用途**：控制任务重试机制，超过此次数后任务将失败，需要人工干预
- **示例**：

```json
{
  "max_retries": 3
}
```

#### claude_code_path

- **类型**：字符串
- **默认值**：空字符串（自动检测）
- **说明**：Claude Code CLI 可执行文件的路径
- **用途**：如果 `claude` 命令不在 PATH 中，可以手动指定完整路径
- **示例**：

```json
{
  "claude_code_path": "/usr/local/bin/claude"
}
```

#### default_workspace

- **类型**：字符串
- **默认值**：空字符串（使用当前目录）
- **说明**：默认工作空间路径
- **用途**：指定 Rick CLI 的默认工作目录
- **示例**：

```json
{
  "default_workspace": "/home/user/projects"
}
```

#### git.user_name

- **类型**：字符串
- **默认值**：空字符串（使用全局 Git 配置）
- **说明**：Git 提交时使用的用户名
- **用途**：覆盖全局 Git 配置中的用户名
- **示例**：

```json
{
  "git": {
    "user_name": "Rick Bot"
  }
}
```

#### git.user_email

- **类型**：字符串
- **默认值**：空字符串（使用全局 Git 配置）
- **说明**：Git 提交时使用的邮箱地址
- **用途**：覆盖全局 Git 配置中的邮箱地址
- **示例**：

```json
{
  "git": {
    "user_email": "rick@example.com"
  }
}
```

### 创建配置文件

首次使用 Rick CLI 时，可以从示例配置创建配置文件：

```bash
# 复制示例配置
cp config.example.json ~/.rick/config.json

# 编辑配置文件
vim ~/.rick/config.json
```

或者手动创建：

```bash
mkdir -p ~/.rick
cat > ~/.rick/config.json <<EOF
{
  "max_retries": 5,
  "claude_code_path": "",
  "default_workspace": "",
  "git": {
    "user_name": "Your Name",
    "user_email": "your.email@example.com"
  }
}
EOF
```

### 配置优先级

Rick CLI 按以下优先级读取配置：

1. **命令行参数**（最高优先级）
2. **环境变量**
3. **配置文件** (`~/.rick/config.json`)
4. **默认值**（最低优先级）

示例：

```bash
# 使用命令行参数覆盖配置文件
rick doing job_1 --max-retries 3

# 使用环境变量
export RICK_MAX_RETRIES=3
rick doing job_1
```

---

## 版本管理

Rick CLI 支持同时安装和使用生产版本和开发版本，这对于自我重构和测试非常有用。

### 生产版本

- **安装目录**：`~/.rick`
- **命令名**：`rick`
- **符号链接**：`~/.local/bin/rick -> ~/.rick/bin/rick`
- **配置文件**：`~/.rick/config.json`
- **用途**：日常使用、生产环境

**安装生产版本**：

```bash
./scripts/install.sh
```

**使用生产版本**：

```bash
rick plan "重构 Rick 架构"
rick doing job_1
rick learning job_1
```

### 开发版本

- **安装目录**：`~/.rick_dev`
- **命令名**：`rick_dev`
- **符号链接**：`~/.local/bin/rick_dev -> ~/.rick_dev/bin/rick`
- **配置文件**：`~/.rick_dev/config.json`（可选，否则使用 `~/.rick/config.json`）
- **用途**：开发、测试、实验性功能

**安装开发版本**：

```bash
./scripts/install.sh --dev
```

**使用开发版本**：

```bash
rick_dev plan "新功能开发"
rick_dev doing job_1
rick_dev learning job_1
```

### 版本切换

由于生产版本和开发版本使用不同的命令名，可以在同一系统上并行使用：

```bash
# 使用生产版本处理日常任务
rick plan "修复 Bug"
rick doing job_1

# 使用开发版本测试新功能
rick_dev plan "实验性功能"
rick_dev doing job_2

# 两个版本互不干扰
rick --version
rick_dev --version
```

### 版本隔离

生产版本和开发版本的工作空间完全隔离：

```
项目目录/
├── .rick/                  # Rick 工作空间（由生产版本管理）
│   ├── jobs/
│   │   ├── job_1/
│   │   └── job_2/
│   ├── skills/
│   └── wiki/
└── .rick_dev/              # Rick Dev 工作空间（由开发版本管理）
    ├── jobs/
    │   ├── job_1/
    │   └── job_2/
    ├── skills/
    └── wiki/
```

**注意事项**：

- 生产版本和开发版本的工作空间互不影响
- 每个版本都有独立的配置文件
- 可以使用开发版本测试新功能，而不影响生产版本的稳定性

---

## 自我重构工作流

Rick CLI 的一个独特特性是能够使用自身来重构自身，这得益于版本管理机制。

### 场景 1：使用 rick 重构 rick（标准流程）

这是最常见的自我重构场景，使用生产版本重构自身。

**步骤**：

```bash
# 1. 规划重构任务
rick plan "重构 Rick CLI 架构"

# Rick 会创建 .rick/jobs/job_1/plan/ 目录，生成任务列表

# 2. 执行重构任务
rick doing job_1

# Rick 会按任务列表执行重构，修改源码，运行测试，自动提交

# 3. 学习和总结
rick learning job_1

# Rick 会分析执行过程，提取可复用技能，更新知识库

# 4. 重新安装更新后的 Rick
./scripts/update.sh

# 5. 验证更新
rick --version
```

**工作流程图**：

```
规划阶段 (rick plan)
    ↓
执行阶段 (rick doing)
    ↓
学习阶段 (rick learning)
    ↓
重新安装 (./scripts/update.sh)
    ↓
验证更新 (rick --version)
```

### 场景 2：使用 rick_dev 开发新功能

这个场景适合开发实验性功能或大型重构，使用开发版本避免影响生产版本。

**步骤**：

```bash
# 1. 安装开发版本
./scripts/install.sh --dev

# 2. 使用开发版本规划新功能
rick_dev plan "实现并行任务执行"

# 3. 使用开发版本执行任务
rick_dev doing job_1

# 4. 测试新功能
rick_dev --help
rick_dev doing --help

# 5. 如果功能稳定，使用生产版本集成
rick plan "集成并行执行功能"
rick doing job_2

# 6. 集成完成后，卸载开发版本
./scripts/uninstall.sh --dev
```

**工作流程图**：

```
安装 rick_dev
    ↓
使用 rick_dev 开发新功能
    ↓
测试新功能
    ↓
使用 rick 集成新功能
    ↓
卸载 rick_dev
```

### 场景 3：并行开发多个版本

这个场景适合需要同时开发多个独立功能或进行 A/B 测试。

**步骤**：

```bash
# 1. 安装开发版本
./scripts/install.sh --dev

# 2. 使用生产版本开发功能 A
rick plan "功能 A：优化任务调度"
rick doing job_1

# 3. 同时使用开发版本开发功能 B
rick_dev plan "功能 B：增强日志系统"
rick_dev doing job_1

# 4. 两个版本独立工作，互不干扰
rick learning job_1      # 总结功能 A
rick_dev learning job_1  # 总结功能 B

# 5. 功能 A 和 B 都完成后，合并到主分支
git merge feature-a
git merge feature-b

# 6. 重新安装生产版本
./scripts/update.sh

# 7. 卸载开发版本
./scripts/uninstall.sh --dev
```

### 最佳实践

1. **使用 Git 分支管理**：为每个重构任务创建独立的 Git 分支
   ```bash
   git checkout -b refactor-architecture
   rick plan "重构架构"
   rick doing job_1
   ```

2. **频繁提交**：Rick 会自动提交每个任务的完成状态，保持提交历史清晰
   ```bash
   # Rick 自动执行
   git commit -m "feat(executor): implement parallel task execution"
   ```

3. **测试驱动**：在重构前编写测试，确保重构不破坏现有功能
   ```bash
   # 在 task.md 中定义测试方法
   ### 测试方法
   1. 运行单元测试：`go test ./internal/executor/...`
   2. 运行集成测试：`./scripts/test_executor.sh`
   ```

4. **渐进式重构**：将大型重构分解为小任务，逐步完成
   ```bash
   rick plan "重构架构（第 1 阶段）"
   rick doing job_1
   rick plan "重构架构（第 2 阶段）"
   rick doing job_2
   ```

5. **保持开发版本同步**：定期将生产版本的更新合并到开发版本
   ```bash
   # 在开发版本的仓库中
   git fetch origin main
   git merge origin/main
   ./scripts/update.sh --dev
   ```

---

## 常见问题

### 安装问题

#### Q1: 提示 "Go is not installed"

**问题**：运行安装脚本时提示 Go 未安装。

**解决方法**：

```bash
# 检查 Go 是否安装
which go

# 如果未安装，前往 https://golang.org/doc/install 下载安装

# 安装后，验证版本
go version
```

#### Q2: 提示 "Go version X.X is not supported"

**问题**：Go 版本过低。

**解决方法**：

```bash
# 升级 Go 到 1.21 或更高版本
# macOS (使用 Homebrew)
brew upgrade go

# Linux (下载最新版本)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# 验证版本
go version
```

#### Q3: 提示 "Command 'rick' not found in PATH"

**问题**：安装完成后无法运行 `rick` 命令。

**解决方法**：

```bash
# 检查符号链接是否存在
ls -l ~/.local/bin/rick

# 检查 ~/.local/bin 是否在 PATH 中
echo $PATH | grep ".local/bin"

# 如果不在 PATH 中，添加到 ~/.bashrc 或 ~/.zshrc
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

# 或者手动添加到 PATH（临时）
export PATH="$HOME/.local/bin:$PATH"
```

#### Q4: 二进制安装失败，提示 "Failed to download binary"

**问题**：从 GitHub Releases 下载二进制文件失败。

**解决方法**：

```bash
# 检查网络连接
curl -I https://github.com

# 检查是否需要代理
export https_proxy=http://your-proxy:port

# 或者使用源码安装
./scripts/install.sh --source
```

### 配置问题

#### Q5: 如何修改最大重试次数？

**解决方法**：

编辑配置文件 `~/.rick/config.json`：

```json
{
  "max_retries": 3
}
```

或者使用命令行参数：

```bash
rick doing job_1 --max-retries 3
```

#### Q6: Claude Code CLI 路径不正确

**问题**：Rick 无法找到 Claude Code CLI。

**解决方法**：

1. **检查 Claude Code CLI 是否安装**：
   ```bash
   which claude
   ```

2. **如果未安装，参考 Claude Code 官方文档安装**

3. **如果安装在非标准路径，在配置文件中指定**：
   ```json
   {
     "claude_code_path": "/usr/local/bin/claude"
   }
   ```

#### Q7: Git 提交时提示 "user.name" 或 "user.email" 未设置

**问题**：Git 配置不完整。

**解决方法**：

1. **设置全局 Git 配置**（推荐）：
   ```bash
   git config --global user.name "Your Name"
   git config --global user.email "your.email@example.com"
   ```

2. **或者在 Rick 配置文件中设置**：
   ```json
   {
     "git": {
       "user_name": "Your Name",
       "user_email": "your.email@example.com"
     }
   }
   ```

### 使用问题

#### Q8: 任务执行失败，如何重试？

**解决方法**：

Rick 会自动重试失败的任务（默认最多 5 次）。如果超过重试限制：

1. **查看调试日志**：
   ```bash
   cat .rick/jobs/job_1/doing/debug.md
   ```

2. **根据错误信息修改任务定义**：
   ```bash
   vim .rick/jobs/job_1/plan/task1.md
   ```

3. **重新执行任务**：
   ```bash
   rick doing job_1
   ```

#### Q9: 如何查看任务执行日志？

**解决方法**：

```bash
# 查看任务执行日志
cat .rick/jobs/job_1/doing/execution.log

# 查看调试日志
cat .rick/jobs/job_1/doing/debug.md

# 查看 Git 提交历史
git log --oneline
```

#### Q10: 如何停止正在执行的任务？

**解决方法**：

```bash
# 使用 Ctrl+C 停止任务执行
# Rick 会保存当前进度，下次执行时从中断点继续

# 或者使用 kill 命令
ps aux | grep rick
kill <pid>
```

### 版本管理问题

#### Q11: 如何同时使用生产版本和开发版本？

**解决方法**：

```bash
# 安装生产版本
./scripts/install.sh

# 安装开发版本
./scripts/install.sh --dev

# 使用生产版本
rick plan "任务 A"

# 使用开发版本
rick_dev plan "任务 B"

# 两个版本互不干扰
```

#### Q12: 如何更新到最新版本？

**解决方法**：

```bash
# 更新生产版本
./scripts/update.sh

# 更新开发版本
./scripts/update.sh --dev

# 更新到特定版本
./scripts/update.sh --version 1.0.0
```

#### Q13: 更新失败，如何回滚？

**解决方法**：

更新脚本会自动备份当前版本，如果更新失败会自动回滚。如果需要手动回滚：

```bash
# 卸载失败的版本
./scripts/uninstall.sh

# 重新安装之前的版本
./scripts/install.sh --version <previous-version>
```

### 自我重构问题

#### Q14: 使用 rick 重构 rick 时，如何避免循环依赖？

**解决方法**：

Rick 使用版本隔离机制避免循环依赖：

1. **使用生产版本规划和执行重构**
2. **重构完成后，重新构建并安装**
3. **新版本不会影响正在运行的 Rick 实例**

```bash
rick plan "重构 Rick"
rick doing job_1      # 使用旧版本 Rick 执行重构
./scripts/update.sh   # 安装新版本
rick --version        # 验证新版本
```

#### Q15: 如何测试重构后的 Rick 而不影响生产版本？

**解决方法**：

使用开发版本进行测试：

```bash
# 安装开发版本
./scripts/install.sh --dev

# 使用开发版本测试
rick_dev plan "测试任务"
rick_dev doing job_1

# 测试通过后，使用生产版本集成
rick plan "集成新功能"
rick doing job_2

# 卸载开发版本
./scripts/uninstall.sh --dev
```

---

## 总结

本文档详细介绍了 Rick CLI 的安装、配置和部署方法，包括：

1. **系统要求**：Go 1.21+, Claude Code CLI, Python 3, Git
2. **安装方法**：源码安装和二进制安装
3. **脚本详解**：build.sh, install.sh, uninstall.sh, update.sh
4. **配置文件**：max_retries, claude_code_path, git 配置
5. **版本管理**：生产版本（rick）和开发版本（rick_dev）
6. **自我重构工作流**：使用 Rick 重构 Rick 的最佳实践
7. **常见问题**：安装、配置、使用和版本管理问题的解决方案

通过本文档，您应该能够顺利安装和使用 Rick CLI，并掌握自我重构的高级用法。如果遇到其他问题，请参考项目文档或提交 Issue。
