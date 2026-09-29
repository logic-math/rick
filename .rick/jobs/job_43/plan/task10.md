# task10：后端文件读取白名单扩展（跨 workspace 可读）

## 任务目标

`GET /api/workspaces/{ws}/file` 的可读根从「当前 workspace + ~/.rick」扩展为「**所有已注册 workspaces + ~/.rick + rick-dev worktree**」——消息/工具箱里引用的其他工作区文件（如 MOGIC_HOME）、RSI 会话里的 dev 树路径全部可读。

## 关键结果

- KR11a：`internal/web/routes.go` 的 `allowedFileRoots(workspacePath)` 扩展：所有已注册 workspace 路径 + `~/.rick` + dev worktree 根（默认 `<prodRepo 祖父>/rick-dev` 解析，`RICK_DEV_TREE` 环境变量可覆盖；找不到该目录时静默跳过）
- KR11b：workspace registry 注入 `handleWorkspaceFile`（Deps 或参数传递——现有 workspaceFromPath 已有 registry 访问模式，沿用它列出全部 ws.Path）
- KR11c：白名单仍为**显式根列表**语义（symlink 逃逸/目录/超大文件拒绝逻辑不动）；测试更新：跨 workspace 路径 200、未注册路径仍 invalid_path、dev 树路径 200
- KR11d：display 路径回显：其他 workspace 下的文件显示相对该 workspace 的路径（或保持绝对路径——取实现简洁者，注释说明）

# 写域

- internal/web/routes.go
- internal/web/routes_test.go

# 依赖关系

- task9（pipeline 顺序）

## 实现要点

- 现状：`allowedFileRoots(workspacePath)` 返回 `[workspacePath, ~/.rick]`；`workspaceRelativeFile` 把路径 pin 到其中一个根
- registry 访问：看 `workspaceFromPath` 怎么拿 registry（Deps 字段或闭包），同样方式取全部 `ws.Path` 去重后并入 roots
- dev worktree 根解析：模仿 internal/env/devweb 的默认逻辑（prodRepo 祖父目录下 rick-dev），但 routes.go 拿不到 prodRepo——用 `RICK_DEV_TREE` env 优先，无 env 时从**当前二进制位置或配置**推导均不可靠——**最简：仅 `RICK_DEV_TREE` env + 默认 `/workdir/sunquan20/rick-dev`？不行，硬编码**。改为：白名单根列表由调用侧注入（Deps 增加可选 ExtraFileRoots []string，web.go 组合根把 dev 树路径传进来——组合根能拿到 prodRepo 路径）
- 测试用 t.Setenv/显式注入覆盖三态

## 测试方法

- `go build ./...` + `go test ./internal/web/` 绿（含新测试：注册的另一 workspace 路径 200 / 未注册路径 invalid_path / extra root 200）
- gate7 静态断言 + 生产实例手动 curl 验证（MOGIC_HOME 路径 200）

## 禁止事项

- 只改写域 2 文件（组合根注入如需改 internal/cmd/web.go 或 internal/web/server.go 的 Deps 构造，允许追加：internal/cmd/web.go）——**写域修正：internal/web/routes.go、internal/web/routes_test.go、internal/cmd/web.go**；不做任何 git 操作；只在 dev 树
