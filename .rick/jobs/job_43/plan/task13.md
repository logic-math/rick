# task13：文件读取去白名单（直接读任意路径）

## 任务目标

按 human 裁决（「不要搞特殊逻辑」）：`GET /api/workspaces/{ws}/file` 去掉 allowedFileRoots 根目录白名单判定——输入什么路径就读什么路径。

## 关键结果

- KR14a：`internal/web/routes.go` 的 `workspaceRelativeFile`/`allowedFileRoots` 根目录判定**移除**（或简化为直接解析）：file:// URL → 绝对路径；相对路径 → 按当前 workspace 根解析；不再检查「是否落在允许根内」
- KR14b：保留基本防御（非「白名单特殊逻辑」）：仅文件非目录、大小上限、NUL 拒绝——**symlink 跟随放行**（不再做 symlink 组件拒绝——那也是根目录防御的一部分；读 symlink 目标即可）
- KR14c：组合根 `ExtraFileRoots` 注入与 `resolveDevTreeRoots` 随之删除（无消费者）；Deps.ExtraFileRoots 字段删除
- KR14d：测试简化：任意绝对路径 200（如 /etc/hostname）、相对路径按 workspace 解析 200、目录 400、超大 400；删除跨 workspace/extra root 相关测试（语义已无）
- KR14e：display 路径回显简化：当前 workspace 内 → 相对路径；否则原样（绝对/~/.rick 前缀逻辑可简化为统一原样回显）

# 写域

- internal/web/routes.go
- internal/web/workspacefile_test.go
- internal/web/routes_test.go
- internal/cmd/web.go

# 依赖关系

- task12（pipeline 顺序）

## 实现要点

- 现状（迭代 5 task10 刚加的）：allowedFileRoots = 全注册 workspaces + ~/.rick + ExtraFileRoots；workspaceRelativeFile(ws.Path, roots, raw) 把路径 pin 到根。human 验收后裁决整个白名单思路不要
- 简化后 handleWorkspaceFile：raw → 解析（file:// 剥壳 / 相对按 ws.Path 解析为绝对）→ readWhitelistedFile 的「文件类型/大小」部分保留（改名为 readFileBasic 或内联），根检查删除
- 注释更新：安全说明改为「token 认证 + 单用户局域网部署模型；路径不做根限制（human 裁决 2026-10-01）」

## 测试方法

- `go build ./...` + `go test ./internal/web/ ./internal/cmd/`（gate 用净化 env 跑——剥离 RICK_* 变量，见 gate7 教训）
- gate9 断言：routes.go 无 allowedFileRoots/workspaceRelativeFile 根判定；/etc/hostname 类绝对路径测试存在；ExtraFileRoots 已删

## 禁止事项

- 只改写域 4 文件；不做 git 操作；只在 dev 树
