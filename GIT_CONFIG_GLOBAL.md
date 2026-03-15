# Git 配置全局化

## 修改日期
2026-03-14

## 改动说明

将 Git 用户信息（`user.name` 和 `user.email`）从硬编码改为从全局配置文件读取。

## 配置文件位置

- **生产版本**: `~/.rick/config.json`
- **开发版本**: `~/.rick_dev/config.json`

## 配置文件格式

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

## 配置项说明

### `git.user_name`
- **类型**: String
- **默认值**: `"Rick CLI"`
- **说明**: Git commit 时使用的用户名
- **示例**: `"Zhang San"`, `"Rick Bot"`

### `git.user_email`
- **类型**: String
- **默认值**: `"rick@localhost"`
- **说明**: Git commit 时使用的邮箱
- **示例**: `"zhangsan@example.com"`, `"bot@company.com"`

## 使用方法

### 方法 1：手动创建配置文件

```bash
# 创建配置目录
mkdir -p ~/.rick

# 创建配置文件
cat > ~/.rick/config.json << 'EOF'
{
  "max_retries": 5,
  "claude_code_path": "",
  "default_workspace": "",
  "git": {
    "user_name": "Zhang San",
    "user_email": "zhangsan@example.com"
  }
}
EOF
```

### 方法 2：使用示例配置

```bash
# 复制示例配置
cp config.example.json ~/.rick/config.json

# 编辑配置
vim ~/.rick/config.json
```

### 方法 3：使用默认配置

如果不创建配置文件，Rick 会使用默认值：
- `user_name`: `"Rick CLI"`
- `user_email`: `"rick@localhost"`

## 工作原理

### 配置优先级

```
1. 仓库本地配置 (git config user.name)
   ↓ 如果未配置
2. Rick 全局配置 (~/.rick/config.json)
   ↓ 如果未配置或为空
3. 硬编码默认值 ("Rick CLI" / "rick@localhost")
```

### 配置流程

```go
// 1. 加载全局配置
cfg, _ := config.LoadConfig()

// 2. 检查仓库是否已配置
cmd := exec.Command("git", "config", "user.name")
if output, err := cmd.Output(); err != nil || output == "" {
    // 3. 从全局配置读取
    userName := cfg.Git.UserName
    if userName == "" {
        userName = "Rick CLI" // 4. 使用默认值
    }

    // 5. 设置到仓库
    exec.Command("git", "config", "user.name", userName).Run()
}
```

## 代码改动

### 1. 修改 `internal/config/config.go`

**新增 `GitConfig` 结构体**：
```go
// GitConfig represents Git-related configuration
type GitConfig struct {
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
}
```

**修改 `Config` 结构体**：
```go
type Config struct {
	MaxRetries       int        `json:"max_retries"`
	ClaudeCodePath   string     `json:"claude_code_path"`
	DefaultWorkspace string     `json:"default_workspace"`
	Git              GitConfig  `json:"git"`  // 新增
}
```

### 2. 修改 `internal/config/loader.go`

**更新默认配置**：
```go
return &Config{
	MaxRetries:       5,
	ClaudeCodePath:   "",
	DefaultWorkspace: filepath.Join(home, rickDirName),
	Git: GitConfig{
		UserName:  "Rick CLI",
		UserEmail: "rick@localhost",
	},
}
```

### 3. 修改 `internal/cmd/doing.go`

**更新 `ensureGitUserConfigured()` 函数**：
```go
func ensureGitUserConfigured(projectRoot string) error {
	// Load global config
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Use config values with fallback
	userName := cfg.Git.UserName
	if userName == "" {
		userName = "Rick CLI"
	}

	userEmail := cfg.Git.UserEmail
	if userEmail == "" {
		userEmail = "rick@localhost"
	}

	// Set to repository if not configured
	// ...
}
```

## 使用场景

### 场景 1：个人开发者

**配置**：
```json
{
  "git": {
    "user_name": "Zhang San",
    "user_email": "zhangsan@company.com"
  }
}
```

**效果**：所有通过 Rick 创建的 commit 都会使用这个身份。

### 场景 2：团队机器人

**配置**：
```json
{
  "git": {
    "user_name": "Rick Bot",
    "user_email": "rick-bot@company.com"
  }
}
```

**效果**：便于识别哪些 commit 是 Rick 自动生成的。

### 场景 3：CI/CD 环境

**配置**：
```json
{
  "git": {
    "user_name": "CI Bot",
    "user_email": "ci@company.com"
  }
}
```

**效果**：CI 环境中的自动化任务使用统一的 Git 身份。

### 场景 4：使用默认值

**不创建配置文件**：Rick 会使用默认值 `Rick CLI <rick@localhost>`

## 测试验证

### 测试 1：使用自定义配置

```bash
# 创建配置
cat > ~/.rick/config.json << 'EOF'
{
  "max_retries": 5,
  "git": {
    "user_name": "Test User",
    "user_email": "test@example.com"
  }
}
EOF

# 创建新项目
cd /tmp/test_custom_config
rick plan "测试任务"
rick doing job_0

# 验证 Git 配置
git config user.name    # 应该输出: Test User
git config user.email   # 应该输出: test@example.com

# 验证 commit
git log -1 --format="%an <%ae>"
# 应该输出: Test User <test@example.com>
```

### 测试 2：使用默认配置

```bash
# 删除配置文件
rm ~/.rick/config.json

# 创建新项目
cd /tmp/test_default_config
rick plan "测试任务"
rick doing job_0

# 验证 Git 配置
git config user.name    # 应该输出: Rick CLI
git config user.email   # 应该输出: rick@localhost
```

### 测试 3：尊重现有配置

```bash
# 创建项目并手动配置 Git
cd /tmp/test_existing_config
git init
git config user.name "Existing User"
git config user.email "existing@example.com"

# 运行 Rick
rick plan "测试任务"
rick doing job_0

# 验证 Git 配置（应该保持不变）
git config user.name    # 应该输出: Existing User
git config user.email   # 应该输出: existing@example.com
```

## 配置文件示例

项目根目录提供了示例配置文件：`config.example.json`

```bash
# 复制并编辑
cp config.example.json ~/.rick/config.json
vim ~/.rick/config.json
```

## 向后兼容性

- ✅ 如果没有配置文件，使用默认值
- ✅ 如果配置文件中 `git` 字段为空，使用默认值
- ✅ 如果仓库已有 Git 配置，不会覆盖
- ✅ 现有项目不受影响

## 与全局 Git 配置的关系

Rick 的配置与全局 Git 配置（`~/.gitconfig`）**独立**：

| 配置级别 | 位置 | 优先级 | 说明 |
|---------|------|--------|------|
| 仓库配置 | `.git/config` | 最高 | Rick 检查到已配置则不修改 |
| Rick 配置 | `~/.rick/config.json` | 中等 | Rick 从这里读取默认值 |
| 全局 Git 配置 | `~/.gitconfig` | 低 | Rick 不读取此配置 |

**注意**：Rick 只在仓库本地设置 Git 配置，不会修改全局 Git 配置。

## 最佳实践

### 1. 团队协作

在团队文档中说明 Rick 的 Git 配置：

```markdown
## Rick CLI 配置

团队成员使用 Rick 时，请配置 Git 用户信息：

\`\`\`bash
cat > ~/.rick/config.json << 'EOF'
{
  "git": {
    "user_name": "你的名字",
    "user_email": "你的邮箱@company.com"
  }
}
EOF
\`\`\`
```

### 2. 机器人标识

使用明确的机器人标识：

```json
{
  "git": {
    "user_name": "Rick Bot [Auto]",
    "user_email": "rick-bot+auto@company.com"
  }
}
```

### 3. 环境区分

不同环境使用不同的配置：

```bash
# 开发环境
~/.rick/config.json

# CI 环境
/etc/rick/config.json  # 通过环境变量指定
```

## 常见问题

### Q1: 如何查看当前配置？

```bash
# 方法 1：查看配置文件
cat ~/.rick/config.json

# 方法 2：在项目中查看 Git 配置
cd /path/to/project
git config user.name
git config user.email
```

### Q2: 配置不生效怎么办？

1. 检查配置文件格式是否正确（JSON 格式）
2. 检查仓库是否已有配置（Rick 不会覆盖）
3. 使用 `rick doing --verbose` 查看详细日志

### Q3: 可以为不同项目使用不同配置吗？

可以，在项目中手动设置 Git 配置：

```bash
cd /path/to/project
git config user.name "Project Specific Name"
git config user.email "project@example.com"
```

Rick 会尊重项目的本地配置，不会覆盖。

## 总结

✅ **全局配置化**：Git 用户信息从 `~/.rick/config.json` 读取

✅ **灵活配置**：支持自定义用户名和邮箱

✅ **智能默认**：未配置时使用合理的默认值

✅ **向后兼容**：不影响现有项目和配置

✅ **尊重现有配置**：不覆盖仓库已有的 Git 配置

---

**修改完成日期**: 2026-03-14
**修改人**: Claude Opus 4.6
