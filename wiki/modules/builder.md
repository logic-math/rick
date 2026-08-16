# 提示词拼接模块（internal/builder）

## 职责

`internal/builder` 是三层金字塔第三层「执行」的一员，按不同入口功能拼接提示词，在 cmd 触发时创建一组符合 runtime 要求的产物。它不直接调用 pi、不管理环境依赖。

## builder 三件

- **templates**（`internal/prompt` 承载）：go embed 内嵌提示词（方法层源码 md）。
- **pibuilder**（`pibuilder.go`）：pi 统一入口组合子 builder，内部组合 plan/doing/easy/human-loop/ctrl/dream/learning 等子 builder。
- **xxxxbuilder**（`xxxxbuilder.go`）：扩展位（未来新增 runtime 只扩展这一 builder，其他组件不改动）。

## 扩展 seam

```go
type RuntimeBuilder interface {
    Name() string
    BuildAgents(method []Method) ([]AgentDef, error)
    BuildPrompt(cmd string, params map[string]string) (string, error)
}
```

`PIBuilder` 是当前唯一实现。将来 dsh 只新增 `dshBuilder` 并注册。

## 单文件内聚

每个命令产物包含两份：

- **method**：命令特定方法层（rick 全局方法前缀 + 命令 SOP），经 `--append-system-prompt` 注入 pi 系统提示词。
- **instance**：job 上下文/路径，作为 user prompt 文件。

关键 skill/loop 内容（doing_loop、grilling 等）在构建时**内联**进主产物，单文件内聚，pi 无需读散落文件。

## doing 编排

`SaveDoingPrompt` 生成含 `workflowScript` 编排的 doing prompt：builder 在 prompt 生成时（rick 侧）把 pending task 按依赖拓扑排序，渲染成 `runs.run` + `await` 的 workflowScript（pi workflowScript 沙箱无文件系统访问，需提前解析依赖）。

## 相关文件

| 文件 | 职责 |
|------|------|
| `pibuilder.go` | PIBuilder（组合子 + Build*/Save*） |
| `xxxxbuilder.go` | RuntimeBuilder 接口 + Method/AgentDef 类型 |
| `orchestration.go` | workflowScript 编排渲染 + EnsureTasksJSON |
