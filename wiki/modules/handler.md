# 调度聚合模块（internal/handler）

## 职责

`internal/handler` 是三层金字塔第二层「调度聚合」。它接受入口路由与解析后的参数，编排 env → builder → runtime 三个模块完成 rick 命令的功能实现，并把 runtime 返回的 sessionID 持久化到 job 目录。

- **做**：编排顺序 env（保证 pi 就绪）→ builder（拼提示词）→ runtime（拉 pi）；持久化 sessionID 到 job 目录。
- **不做**：不 import `internal/cmd`（避免跨包循环依赖，cli 在调用点把 flag 值经 `Options` 透传）；不做具体安装、不做具体 pi 调用、不做具体提示词拼接。

## 编排契约

```
handler 编排顺序 = env（保证 pi 就绪）→ builder（拼提示词）→ runtime（拉 pi）
handler 持久化 = sessionID 写入 job 目录
```

## 关键设计

### DIP 组合根（Runtime 接口注入）

`handler.Doing` 依赖 `runtime.Runtime` **接口**，不依赖具体实现。具体 `piRuntime` 由组合根 `internal/cmd` 构造后注入：

```go
// internal/cmd/doing.go（组合根）
rt := runtime.NewPiRuntime(cfg.PiPath, cfg.PiExtraArgs...)
handler.Doing(jobID, opts, rt)

// internal/handler/doing.go
func Doing(jobID string, opts Options, rt runtime.Runtime) error {
    // ...
    _, _, err = rt.Run(method, promptFile, cfg)
}
```

这样将来新增 dsh runtime 只实现并注册 `dshRuntime`，handler 不改。

### doing 门禁下沉

`handler.Doing` 在 pi 会话 `agent_settled` 后运行确定性门禁脚本：

```bash
python3 .rick/skills/rick-gates/helper.py <doing_dir>
```

校验三项：tasks.json 可解析、无遗留 running 状态任务、success 任务有非空 commit_hash。

## 文件清单

| 文件 | 职责 |
|------|------|
| `doing.go` | doing 工作流（重试 + 门禁） |
| `plan.go` | plan 工作流 |
| `easy.go` | easy 会话（新建/恢复） |
| `human_loop.go` | SENSE 深度思考 |
| `ctrl.go` | 黑箱干预 |
| `dream.go` | 跨 job 反思 |
| `learning.go` | 知识提取 |
| `options.go` | `Options` 结构（flag 值透传） |
