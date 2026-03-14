# Config 模块详解

> 配置管理模块 - 全局配置加载、验证和持久化

## 📋 模块概述

Config 模块负责管理 Rick CLI 的全局配置，包括配置文件的加载、验证、保存和默认值管理。该模块遵循"简化设计"原则，仅使用一个全局配置文件 `~/.rick/config.json`。

### 功能职责
- 加载和解析配置文件
- 提供默认配置
- 配置验证
- 配置持久化
- 支持生产版和开发版隔离

### 模块位置
```
internal/config/
├── config.go       # 配置结构定义
└── loader.go       # 配置加载和保存
```

---

## 🏗️ 核心类型和接口

### Config 结构

```go
type Config struct {
    MaxRetries       int    `json:"max_retries"`       // 任务最大重试次数
    ClaudeCodePath   string `json:"claude_code_path"`  // Claude Code CLI 路径
    DefaultWorkspace string `json:"default_workspace"` // 默认工作空间路径
}
```

**字段说明**：

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `MaxRetries` | `int` | `5` | 任务执行失败时的最大重试次数 |
| `ClaudeCodePath` | `string` | `""` (空) | Claude Code CLI 的完整路径，空则使用 `claude` |
| `DefaultWorkspace` | `string` | `~/.rick` | 默认工作空间目录 |

---

## 🔧 主要函数说明

### 1. 配置路径管理

#### `GetConfigPath() (string, error)`

获取配置文件路径。

**行为**：
- 生产版：`~/.rick/config.json`
- 开发版：`~/.rick_dev/config.json`

**自动检测**：
根据二进制文件名自动选择配置目录：
- `rick` → `~/.rick/`
- `rick_dev` → `~/.rick_dev/`

**示例**：
```go
configPath, err := config.GetConfigPath()
if err != nil {
    log.Fatal(err)
}
fmt.Println("Config path:", configPath)
// 输出: Config path: /Users/username/.rick/config.json
```

**实现细节**：
```go
func getConfigPath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("failed to get home directory: %w", err)
    }

    // 根据二进制名称确定目录
    binaryPath := os.Args[0]
    binaryName := filepath.Base(binaryPath)
    rickDirName := ".rick"
    if strings.HasSuffix(binaryName, "_dev") {
        rickDirName = ".rick_dev"
    }

    return filepath.Join(home, rickDirName, "config.json"), nil
}
```

### 2. 默认配置

#### `GetDefaultConfig() *Config`

返回默认配置。

**默认值**：
```go
{
    MaxRetries:       5,
    ClaudeCodePath:   "",
    DefaultWorkspace: "~/.rick" // 或 ~/.rick_dev
}
```

**使用场景**：
- 配置文件不存在时
- 初始化新安装时
- 测试时提供基准配置

**示例**：
```go
defaultCfg := config.GetDefaultConfig()
fmt.Printf("Max retries: %d\n", defaultCfg.MaxRetries)
// 输出: Max retries: 5
```

### 3. 配置加载

#### `LoadConfig() (*Config, error)`

加载配置文件。

**加载逻辑**：
```
1. 获取配置文件路径
   └─> GetConfigPath()

2. 检查文件是否存在
   ├─> 存在: 读取并解析 JSON
   └─> 不存在: 返回默认配置

3. 解析 JSON 到 Config 结构
   └─> json.Unmarshal()

4. 返回配置对象
```

**错误处理**：
- 无法获取 Home 目录
- 文件读取失败
- JSON 解析失败

**示例**：
```go
cfg, err := config.LoadConfig()
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

fmt.Printf("Loaded config: MaxRetries=%d\n", cfg.MaxRetries)
```

**完整示例（带错误处理）**：
```go
func loadAndValidateConfig() (*config.Config, error) {
    // 加载配置
    cfg, err := config.LoadConfig()
    if err != nil {
        return nil, fmt.Errorf("failed to load config: %w", err)
    }

    // 验证配置
    if err := config.ValidateConfig(cfg); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    return cfg, nil
}
```

### 4. 配置保存

#### `SaveConfig(cfg *Config) error`

保存配置到文件。

**保存流程**：
```
1. 获取配置文件路径
   └─> GetConfigPath()

2. 确保目录存在
   └─> os.MkdirAll()

3. 序列化配置为 JSON
   └─> json.MarshalIndent() (格式化输出)

4. 写入文件
   └─> os.WriteFile()
```

**JSON 格式**：
```json
{
  "max_retries": 5,
  "claude_code_path": "/usr/local/bin/claude",
  "default_workspace": "/Users/username/.rick"
}
```

**示例**：
```go
cfg := &config.Config{
    MaxRetries:       10,
    ClaudeCodePath:   "/usr/local/bin/claude",
    DefaultWorkspace: "/Users/username/.rick",
}

if err := config.SaveConfig(cfg); err != nil {
    log.Fatalf("Failed to save config: %v", err)
}
```

**错误处理**：
- 目录创建失败
- JSON 序列化失败
- 文件写入失败

### 5. 配置验证

#### `ValidateConfig(cfg *Config) error`

验证配置的有效性。

**验证规则**：

1. **MaxRetries 验证**：
   - 必须 >= 0
   - 推荐范围：1-10

2. **ClaudeCodePath 验证**：
   - 如果不为空，必须是有效的文件路径
   - 文件必须存在

3. **DefaultWorkspace 验证**：
   - （当前未验证，未来可添加）

**示例**：
```go
cfg := &config.Config{
    MaxRetries:     -1, // 无效
    ClaudeCodePath: "/nonexistent/path",
}

if err := config.ValidateConfig(cfg); err != nil {
    fmt.Println("Validation error:", err)
    // 输出: Validation error: MaxRetries must be non-negative, got -1
}
```

**实现细节**：
```go
func ValidateConfig(cfg *Config) error {
    // 验证 MaxRetries
    if cfg.MaxRetries < 0 {
        return fmt.Errorf("MaxRetries must be non-negative, got %d", cfg.MaxRetries)
    }

    // 验证 ClaudeCodePath（如果不为空）
    if cfg.ClaudeCodePath != "" {
        if _, err := os.Stat(cfg.ClaudeCodePath); os.IsNotExist(err) {
            return fmt.Errorf("ClaudeCodePath does not exist: %s", cfg.ClaudeCodePath)
        }
    }

    return nil
}
```

---

## 💡 使用示例

### 示例 1: 基本使用

```go
package main

import (
    "fmt"
    "log"
    "github.com/sunquan/rick/internal/config"
)

func main() {
    // 加载配置
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatal(err)
    }

    // 使用配置
    fmt.Printf("Max Retries: %d\n", cfg.MaxRetries)
    fmt.Printf("Claude Path: %s\n", cfg.ClaudeCodePath)
    fmt.Printf("Workspace: %s\n", cfg.DefaultWorkspace)
}
```

### 示例 2: 修改配置

```go
package main

import (
    "log"
    "github.com/sunquan/rick/internal/config"
)

func main() {
    // 加载现有配置
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatal(err)
    }

    // 修改配置
    cfg.MaxRetries = 10
    cfg.ClaudeCodePath = "/usr/local/bin/claude"

    // 验证配置
    if err := config.ValidateConfig(cfg); err != nil {
        log.Fatalf("Invalid config: %v", err)
    }

    // 保存配置
    if err := config.SaveConfig(cfg); err != nil {
        log.Fatalf("Failed to save config: %v", err)
    }

    log.Println("Config updated successfully")
}
```

### 示例 3: 配置初始化工具

```go
package main

import (
    "bufio"
    "fmt"
    "os"
    "strconv"
    "strings"
    "github.com/sunquan/rick/internal/config"
)

func main() {
    reader := bufio.NewReader(os.Stdin)

    // 获取 MaxRetries
    fmt.Print("Enter max retries (default 5): ")
    retriesStr, _ := reader.ReadString('\n')
    retriesStr = strings.TrimSpace(retriesStr)
    maxRetries := 5
    if retriesStr != "" {
        if n, err := strconv.Atoi(retriesStr); err == nil {
            maxRetries = n
        }
    }

    // 获取 ClaudeCodePath
    fmt.Print("Enter Claude Code path (default: claude): ")
    claudePath, _ := reader.ReadString('\n')
    claudePath = strings.TrimSpace(claudePath)

    // 创建配置
    cfg := &config.Config{
        MaxRetries:       maxRetries,
        ClaudeCodePath:   claudePath,
        DefaultWorkspace: config.GetDefaultConfig().DefaultWorkspace,
    }

    // 验证配置
    if err := config.ValidateConfig(cfg); err != nil {
        fmt.Printf("Invalid config: %v\n", err)
        os.Exit(1)
    }

    // 保存配置
    if err := config.SaveConfig(cfg); err != nil {
        fmt.Printf("Failed to save config: %v\n", err)
        os.Exit(1)
    }

    fmt.Println("Configuration saved successfully!")
}
```

### 示例 4: 测试中使用配置

```go
package mypackage_test

import (
    "testing"
    "github.com/sunquan/rick/internal/config"
)

func TestWithCustomConfig(t *testing.T) {
    // 使用默认配置进行测试
    cfg := config.GetDefaultConfig()
    cfg.MaxRetries = 3 // 测试时减少重试次数

    // 验证配置
    if err := config.ValidateConfig(cfg); err != nil {
        t.Fatalf("Invalid test config: %v", err)
    }

    // 使用配置进行测试
    // ...
}
```

---

## ❓ 常见问题

### Q1: 配置文件在哪里？

**A**: 配置文件路径取决于使用的版本：
- **生产版** (`rick`): `~/.rick/config.json`
- **开发版** (`rick_dev`): `~/.rick_dev/config.json`

查看配置路径：
```bash
# 方法 1: 直接查看
cat ~/.rick/config.json

# 方法 2: 使用 Go 代码
go run -ldflags "-X main.version=dev" cmd/rick/main.go
```

### Q2: 如何重置配置？

**A**: 删除配置文件，下次运行时会自动使用默认配置：
```bash
rm ~/.rick/config.json
rick plan "test"  # 会使用默认配置
```

或者在代码中：
```go
cfg := config.GetDefaultConfig()
config.SaveConfig(cfg)
```

### Q3: 配置文件格式错误怎么办？

**A**: Rick 会报错并拒绝加载。修复方法：
1. 手动编辑 `~/.rick/config.json`
2. 或删除配置文件，使用默认配置

验证 JSON 格式：
```bash
cat ~/.rick/config.json | jq .
```

### Q4: 如何在不同环境使用不同配置？

**A**: 使用环境变量或配置文件切换：

**方法 1**: 使用开发版
```bash
./install.sh --source --dev  # 安装 rick_dev
rick_dev plan "test"          # 使用 ~/.rick_dev/config.json
```

**方法 2**: 符号链接
```bash
ln -s ~/.rick/config.prod.json ~/.rick/config.json
# 或
ln -s ~/.rick/config.dev.json ~/.rick/config.json
```

### Q5: ClaudeCodePath 为空会怎样？

**A**: Rick 会使用系统 PATH 中的 `claude` 命令。确保 Claude Code CLI 已安装并在 PATH 中：
```bash
which claude
# 输出: /usr/local/bin/claude
```

如果未安装，设置完整路径：
```json
{
  "claude_code_path": "/usr/local/bin/claude"
}
```

---

## 🔗 相关模块

- [Workspace 模块](./workspace.md) - 使用 `DefaultWorkspace` 配置
- [Executor 模块](./dag_executor.md) - 使用 `MaxRetries` 配置
- [CMD 模块](./cli_commands.md) - 加载配置并传递给其他模块

---

## 📚 设计原则

### 1. 单一配置文件
遵循"简化设计"原则，仅使用一个全局配置文件，避免多层级配置的复杂性。

### 2. 默认优先
提供合理的默认值，用户无需配置即可使用。

### 3. 版本隔离
生产版和开发版使用独立的配置目录，避免相互干扰：
- `rick` → `~/.rick/`
- `rick_dev` → `~/.rick_dev/`

### 4. 验证优先
在使用配置前进行验证，避免运行时错误。

### 5. JSON 格式
使用 JSON 格式，易于编辑和程序解析。

---

## 🎯 最佳实践

### 1. 配置加载
始终在程序入口处加载配置：
```go
func main() {
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatal(err)
    }
    // 传递配置到其他模块
}
```

### 2. 配置验证
修改配置后立即验证：
```go
cfg.MaxRetries = 10
if err := config.ValidateConfig(cfg); err != nil {
    // 处理错误
}
```

### 3. 错误处理
使用错误包装，保留错误链：
```go
if err := config.SaveConfig(cfg); err != nil {
    return fmt.Errorf("failed to save config: %w", err)
}
```

### 4. 配置持久化
仅在必要时保存配置，避免频繁 I/O：
```go
// 不推荐：每次修改都保存
cfg.MaxRetries = 10
config.SaveConfig(cfg)

// 推荐：批量修改后保存
cfg.MaxRetries = 10
cfg.ClaudeCodePath = "/usr/local/bin/claude"
config.SaveConfig(cfg)
```

### 5. 测试配置
测试时使用默认配置或临时配置：
```go
func TestSomething(t *testing.T) {
    cfg := config.GetDefaultConfig()
    cfg.MaxRetries = 1 // 测试时减少重试
    // 使用 cfg 进行测试
}
```

---

## 🔮 未来扩展

### 可能的新配置项

```go
type Config struct {
    MaxRetries       int    `json:"max_retries"`
    ClaudeCodePath   string `json:"claude_code_path"`
    DefaultWorkspace string `json:"default_workspace"`

    // 未来扩展
    LogLevel         string `json:"log_level"`          // 日志级别
    ParallelTasks    int    `json:"parallel_tasks"`     // 并行任务数
    TimeoutSeconds   int    `json:"timeout_seconds"`    // 超时时间
    AutoCommit       bool   `json:"auto_commit"`        // 自动提交
    RemoteMode       bool   `json:"remote_mode"`        // 远程模式
}
```

### 配置迁移

未来版本可能需要配置迁移机制：
```go
func MigrateConfig(oldVersion string) error {
    // 从旧版本配置迁移到新版本
    // 例如：v1.0 -> v2.0
}
```

---

*最后更新: 2026-03-14*
