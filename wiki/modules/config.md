# 配置管理模块（internal/config）

## 职责

`internal/config` 负责加载 `~/.rick/config.json` 并填充默认值。它是第四层「基础设施」，可被任意层直接使用。

## 配置结构

```go
type Config struct {
    MaxRetries       int             `json:"max_retries"`
    Runtime          string          `json:"runtime"`
    PiPath           string          `json:"pi_path"`
    PiExtraArgs      []string        `json:"pi_extra_args,omitempty"`
    DefaultWorkspace string          `json:"default_workspace"`
    Git              GitConfig       `json:"git"`
    HumanLoop        HumanLoopConfig `json:"human_loop"`
}
```

## 配置项

| 字段 | 默认 | 说明 |
|------|------|------|
| `max_retries` | 5 | 标准模式任务失败最大重试次数 |
| `runtime` | `pi` | 当前 agent runtime 标识（为将来 dsh 预留扩展 seam） |
| `pi_path` | 空 | pi CLI 路径（空则按托管运行时 → PATH 解析） |
| `pi_extra_args` | 空 | 透传给 pi 的额外 flags（`--provider`/`--model`/`--api-key`） |
| `default_workspace` | 空 | 默认工作区路径 |
| `git.user_name` / `git.user_email` | 空 | 自动 commit 使用的 Git 身份 |
| `human_loop.*` | 见下 | human-loop 各阈值（max_retries / sense_max_backflows / think_top_n / think_min_assumptions / research_source_weights） |

## 重要约定

- **pi 不读环境变量**：provider/model/api-key 必须通过 `pi_extra_args` 或命令行 flags 配置，不要用 env 注入。
- **配置污染防护**：测试开头注入临时 HOME（`t.Setenv("HOME", dir)`），避免读到真实 `~/.rick/config.json` 触发真实 pi 调用。

## 示例

```json
{
  "max_retries": 5,
  "runtime": "pi",
  "pi_path": "",
  "pi_extra_args": ["--provider", "deepseek", "--model", "deepseek-v4-pro", "--api-key", "sk-..."],
  "default_workspace": "",
  "git": {
    "user_name": "Your Name",
    "user_email": "your.email@example.com"
  }
}
```
