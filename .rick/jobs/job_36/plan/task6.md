# 依赖关系
task2

# 写域
internal/web/jobs.go
internal/web/jobs_test.go

# 任务目标
Jobs/Knowledge 只读 REST 数据层（无 HTTP 挂载——路由在 task13；本 task 提供 handler 函数 + 数据读取）。

# 关键结果
1. `internal/web/jobs.go`：
   - `type JobSummary struct{JobID string; UpdatedAt time.Time; Tasks []TaskBrief}` + `TaskBrief{TaskID, Name, Status, CommitHash string}`——`ListJobs(rickDir string) ([]JobSummary, error)`：扫 `<ws>/.rick/jobs/*/doing/tasks.json`（复用 workspace 包既有 thinTaskState 读取模式或自建轻量解析）；按 updated_at 降序
   - `ReadTasks(rickDir, jobID) (json.RawMessage, error)`（tasks.json 原文）；`ReadJobFile(rickDir, jobID, relPath) (string, error)`——白名单：relPath 必须匹配 `^(plan|doing)/` 前缀且清洗后不含 `..`（filepath.Clean + 前缀复核）；上限 2MB
   - `KnowledgeTree(rickDir) ([]FileNode, error)`：枚举 `.rick/{domain,loops,skills}` 下文件（相对 path+size，跳过二进制——按扩展名 .md/.json/.txt/.yaml/.py/.ts）；`ReadKnowledgeFile(rickDir, relPath) (string, error)`（白名单三前缀+清洗+2MB 上限）
   - 错误类型：`WebError{Code, Message, Status}` 实现 error 接口（invalid_path/not_found/invalid_workspace→400/404/400），task13 直接映射 HTTP
2. `internal/web/jobs_test.go`：tmpdir 造 fixture（假 job 目录树 + tasks.json + knowledge 文件）表驱动——ListJobs 排序与字段；路径逃逸用例（`../config.json`、`/etc/passwd`、`doing/../../x` 全拒）；白名单外前缀拒；大文件上限拒；not_found
3. `go build ./...` + `go test ./internal/web/ -run TestJobs -timeout 60s` 通过

# 测试方法
go test ./internal/web/ -run TestJobs -v 全绿；手工：对本仓库真实 .rick 跑 ListJobs 看输出合理（go test 里加一个 skip 的 smoke 用例或独立 main）

# 上下文提示
- handler core 已路径参数化（task2 产出 PlanIn 等）——本 task 只做「读」，不碰 handler 编排
- 白名单安全：这是对外 HTTP 暴露面，路径清洗必须有显式测试（../、绝对路径、符号链接跳过——用 Lstat 拒 Symlink）
