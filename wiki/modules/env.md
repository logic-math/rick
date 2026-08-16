# 环境模块（internal/env）

## 职责

`internal/env` 是三层金字塔第三层「执行」的一员，保证 rick 在当前机器的运行环境（pi 及扩展、rick 定制）就绪。它不管理 session（session 归 runtime）、不做 dag 调度与门禁。

## env 四职责

1. **安装/更新 pi agent**（`pi.go`）：rick 自闭环 pi 运行时（`~/.rick/pi/agent/runtime`）就绪，缺失则 `npm install --prefix` 安装独立副本。
2. **安装/更新 pi 生态扩展/插件/skill**（`extensions.go`）：`pi-subagents`、`pi-web-access` 等 npm 扩展注册。
3. **安装/更新 rick 自有 hook/skill/agent 定制**（`customizations.go`）：rick-gates hook 扩展、rick skills、think/research/exporter 自定义 agent 落盘。
4. **提供 pi 功能点就绪 check 函数，不含 session**（`check.go`）：`CheckReady()` 返回未就绪的功能点（空切片 = 就绪）。

## 扩展 seam

```go
type RuntimeEnv interface {
    Ensure() error                     // 保证 runtime 就绪
    DeployCustomizations() error       // 落盘 rick 自有定制
    CheckReady() []string              // 返回未就绪功能点
}
```

`piEnv` 是当前唯一实现；将来 dsh 只新增 `dshEnv` 并注册，cli/handler 不改。

## 定制落盘

`DeployRickCustomizations()` 把 rick 定制幂等写入 `AgentDir()`（默认 `~/.rick/pi/agent`）：

| 源 | 目标 |
|----|------|
| `<repo>/.rick/skills/rick-gates/` | `extensions/rick-gates/` |
| `<repo>/.rick/skills/*_skill/` | `skills/<name>/` |
| embedded templates/skills/{think,research,exporter}.md | `agents/{name}.md` |

agent 文件带 `rick-managed: true` 标记，覆盖语义：无标记的同名用户文件跳过，避免覆盖用户内容。

## 相关命令

```bash
rick tools init-pi     # 完整「保证 pi 就绪」流程（含职责 3 定制落盘）
```
