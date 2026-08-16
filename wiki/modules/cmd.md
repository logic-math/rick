# 命令处理模块（internal/cmd）

## 职责

`internal/cmd` 是三层金字塔第一层「入口」。它用 Cobra 定义各命令，路由命令到 handler、解析参数与 flag（含 `--dry-run`、`--job` 等），把「入口 + 参数」交给 handler。它不直接拼提示词、不直接拉 pi、不维护 dag 调度与门禁。

## 命令清单

| 命令 | 职责 |
|------|------|
| `rick plan` | 将需求分解为 task 列表，生成 plan 目录 |
| `rick doing` | 执行 job 中的 task（dag 调度下沉 pi workflowScript） |
| `rick easy` | 交互式 AI coding 会话（跳过 plan） |
| `rick learning` | 分析执行结果，提取 loops/skills/domain |
| `rick dream` | 跨 job 全局反思，进化知识体系 |
| `rick ctrl` | 黑箱执行的可挂测性设计（人类干预） |
| `rick human-loop` | SENSE 方法论引导深度思考 |
| `rick tools` | 校验/管理工具（init-pi / learning_check / dream_check / theme） |

## 关键设计

### 全局 flag

`--job` / `--dry-run` / `--verbose` 在 `root.go` 用 `PersistentFlags()` 定义，经 `GetJobID()` / `GetDryRun()` / `GetVerbose()` 统一暴露。子命令文件**不重复定义**全局 flag（否则 flag redefined）。

### 组合根（DIP）

`cmd` 是组合根：在 `RunE` 懒加载实例化 `piRuntime`/`piEnv`/`pibuilder`，注入 handler。例如 doing：

```go
cfg, err := config.LoadConfig()
rt := runtime.NewPiRuntime(cfg.PiPath, cfg.PiExtraArgs...)
handler.Doing(jobID, opts, rt)
```

### dry-run

`--dry-run` 输出完整 prompt 到 stdout，不调用 pi、不创建文件。验证未替换模板变量：

```bash
./bin/rick doing job_N --dry-run | grep -c '{{'   # 应为 0
```

## 相关文件

| 文件 | 职责 |
|------|------|
| `root.go` | 根命令 + 全局 flag |
| `plan.go` / `doing.go` / `easy.go` / `learning.go` | 对应命令入口 |
| `dream.go` / `ctrl.go` / `human_loop.go` | 对应命令入口 |
| `tools.go` / `tools_*.go` | tools 子命令（init-pi / learning_check / dream_check / theme） |
