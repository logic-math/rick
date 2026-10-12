# 依赖关系
无

# 写域
internal/web/registry.go
internal/web/registry_test.go
internal/web/web.go

# 任务目标
创建 internal/web 包基础：workspace 注册表（~/.rick/web.json）+ 会话注册表（~/.rick/web/sessions.json）+ 包 doc。

# 关键结果
1. `internal/web/web.go`：包 doc（WEB-UI 入口层实现——rick-spec 第一层 WEB-UI 的 Go 落地；路径约定常量：`WebStateDir()=~/.rick/web`、`WebConfigPath()=~/.rick/web.json`、`SessionsPath()=~/.rick/web/sessions.json`、`PidPath()=~/.rick/web.pid`；常量尊重 `RICK_PI_AGENT_DIR` 隔离惯例？——否：web 状态跟随 `HOME`（`os.UserHomeDir()`），测试用 t.Setenv("HOME", tmp) 隔离）
2. `internal/web/registry.go`：
   - `type WorkspaceEntry struct{ID, Path, Name string; AddedAt time.Time}`（ID=sha1(path) 前 8 位）
   - `type WorkspaceRegistry struct` + `LoadWorkspaceRegistry(path) (*WorkspaceRegistry, error)`（文件不存在→空表）+ `Add(path, name) (WorkspaceEntry, bool, error)`（bool=是否新建——task13 据此区分 201/200；校验 `filepath.Join(path, ".rick")` 是目录——`invalid_workspace` 错误；幂等：同 path 返回既有+false）+ `Remove(id) error` + `List() []WorkspaceEntry` + `Get(id)`；写回原子（tmp+rename）
   - `type SessionEntry struct{ID, WorkspaceID, Type, Title string; Params map[string]any; Status string; PISessionID string; CreatedAt, ClosedAt time.Time}` + `type SessionRegistry`（同款 Load/Add/Update/List/GetByWorkspace；status 枚举 active/running/closed/error）
   - SessionType 常量与参数校验：`ValidateSessionRequest(type, params) error`（按 api-contract.md 的 params 表：plan.requirement 必填字符串、ctrl.job 必填、dream.job_num 默认 5/mode 默认 background 等）
3. `internal/web/registry_test.go`：HOME 隔离的表驱动测试——workspace 增删查/幂等（返回 created=false 断言）/非法路径（无 .rick → 错误码）/json 落盘往返；session 参数校验矩阵（每 type 合法+非法各 1 例）；原子写（写失败不留半文件——模拟目录只读可跳过，注释说明）
4. `go build ./...` + `go test ./internal/web/ -run TestRegistry -timeout 60s` 通过

# 测试方法
go test ./internal/web/ -run TestRegistry -v 全绿；手工：临时 HOME 下跑一轮 Add/List/落盘检查 JSON 内容

# 上下文提示
- 包名 web（import path github.com/sunquan/rick/internal/web）；与仓库根 web/（前端 embed 包）同名不同包——import 时后端内部互引无碰撞，cmd 层引前端包用别名 `webassets "github.com/sunquan/rick/web"`
- 时间戳一律 RFC3339 带时区（bugs.md：tasks.json 无时区踩过坑）
- api-contract.md（plan/api-contract.md）是字段名单源，先读
