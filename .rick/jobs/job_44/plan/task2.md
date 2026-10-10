# task2：humans 页面后端 API（draft 浏览）

## 任务目标

为 web 新增工作区级 humans 页面提供后端接口：浏览 human-loop 学习材料与进度（`.rick/draft/` 下的 loop 目录、rfc 历史文件），镜像既有 knowledge 接口模式（路径校验 + 仅文件 + 大小上限）。

## 关键结果

- **KR2a 新 handler** `internal/web/humans.go`：
  - `handleHumansTree() http.HandlerFunc` — 返回 `<ws>/.rick/draft/` 的文件树（JSON，与 knowledge tree 同构：`name/path/type/children`）
  - `handleHumansFile() http.HandlerFunc` — 返回单文件内容（`{"path","content"}`，复用 `fileResponse` 契约）
  - draft 根不存在时返回空树（不报错——全新工作区无 human-loop）
- **KR2b 路由注册** `internal/web/routes.go`：`GET /api/workspaces/{ws}/humans/tree` + `GET /api/workspaces/{ws}/humans/file`（`authWrap` 与 knowledge 一致）
- **KR2c 安全语义**（沿用 job_43 裁决：跨工作区文件读取放开 + 保留基本防御）：路径不做白名单根校验，但保留：仅文件（禁目录读取）、大小上限、路径穿越基本防御（`..` 归一化后仍在允许读取的语义内——与 knowledge/jobs 文件接口一致，不引入更严限制）
- **KR2d 测试** `internal/web/humans_test.go`：tree 结构（loops/loop_N/{briefs,lessons,research,mission.md,judgment.md} + rfc/*.md）、file 读取、目录路径拒绝、超限拒绝、draft 缺失返回空树

# 写域

- internal/web/humans.go（新建）
- internal/web/humans_test.go（新建）
- internal/web/routes.go（仅新增两条路由）

# 依赖关系

- 与 task1/task3 无依赖（同层并行）；接口契约已在 plan 冻结（tree/file 与 knowledge 同构）

## 实现要点

- `handleKnowledgeTree`/`handleKnowledgeFile`（`internal/web/routes.go:707/725`）为镜像模板；`jobs.go` 的 `KnowledgeTree`/`ReadKnowledgeFile` 为可复用实现参考
- draft 目录解析：`<ws>/.rick/draft`（`workspace.DraftDirName` 或字面量，与 `handler.PrepareHumanLoopDirsIn` 保持一致）
- 树节点排序稳定（目录优先 + 字典序），与 knowledge 树一致以免测试抖动
- 不修改 draft 内容（只读）

## 测试方法

- `go test ./internal/web/...` 绿（含新增 humans 测试与既有回归）
- 手工：`go run ./cmd/rick web`（或 dev 实例）→ `curl -H "Authorization: Bearer <token>" 'http://127.0.0.1:8414/api/workspaces/<ws>/humans/tree'` 返回树 JSON；`.../humans/file?path=<loop_N/judgment.md>` 返回内容
- 断言 draft 不存在的工作区返回空树而非 500

## 禁止事项

- 不得改动 knowledge/jobs 接口行为
- 不得在 `internal/web/routes.go` 之外注册路由；不得触碰 `internal/web/sessions.go`（task3 依赖其前端契约但不改后端会话逻辑）
- 不得引入写接口（humans 页面只读）
