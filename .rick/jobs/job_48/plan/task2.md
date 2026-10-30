# task2：web super 会话类型（registry + prep + spawn）

## 任务目标

web 新增 `super` 会话类型：创建入口校验、prompt 产物生成（严格加载 super-loop + gen_env skill）、交互 spawn 接线。

## 关键结果

- `internal/web/registry.go`：
  - `SessionTypeSuper = "super"` 常量
  - ValidateSessionRequest：`case SessionTypeSuper:` 参数 `name`（可选字符串，缺省/空 → `default`；**不允许非字符串**）
- `internal/web/sessions.go` prepareInteractive：
  - `case SessionTypeSuper:`——name 默认 `default`；解析工作区 rickDir（会话所属 ws 的 `.rick`）→ superEnvRoot=`<ws>/.rick/super_env`（task1 的解析函数复用/对齐）→ ws=`superEnvRoot/<name>`（不存在则 init 骨架：目录 + super.json 骨架，可复用 handler 的 init 或轻量自建）
  - 调 `superpkg.BuildSuperPrompt(rickDir, ws)`（**严格加载**：super-loop.md / gen_env_skill/skill.md 任一缺失 → BuildSuperPrompt 返回的降级语义若为文本内嵌提示则**改为 error**——检查 BuildSuperPrompt 现行为，若它只降级不报错，本 task 在 prep 层预检文件存在性，缺失即 `newWebError(400, "invalid_params", "super session requires ...")` 显式失败）
  - prep：`{promptFile, methodFile, persistDir=<ws>/sessions/<uuid>, title: "super " + name}`
- 会话创建/列表/close 通用链路零改动（类型驱动）；`api-contract.md` 若有 params 表补 super 行
- Go 测试 `internal/web/`（或 sessions_test）：
  - ValidateSessionRequest：super 无参 OK / name 字符串 OK / name 非字符串 invalid_params
  - prepareInteractive（fake rickDir 含 loops/super-loop.md + skills/gen_env_skill）：promptFile 内容含 gen_env skill 全文关键词与 super_env 路径；缺 gen_env skill → 创建失败（400）
  - 会话 spawn 冒烟（照既有 web 会话测试范式，若重则留给 e2e）

## 写域

- internal/web/registry.go
- internal/web/sessions.go
- internal/web/*_test.go（新增/适配）
- internal/api-contract.md（若存在 params 表）

## 依赖关系

- task1（super_env root 解析与 init 语义就位）

## 实现要点

- prepareInteractive 拿到的 rickDir 即 ws 的 `.rick` 目录（确认现有调用如何传 rickDir——从 ws entry 解析）；super_env root = rickDir + "/super_env"（不走 superWsRoot() 的 cwd 逻辑——web 侧是显式 rickDir，语义一致路径同型）
- 严格加载预检放 prep 层（两个文件 stat），错误信息给出缺失文件路径与修复指引
- session spawn 的 cwd=ws 根（沿用既有 spawn 语义）

## 测试方法

`go build ./... && go test ./internal/web/ -run "TestSession|TestValidate|TestSuper" -timeout 600s` 全绿

## 禁止事项

- 不动前端（task3）；不改 BuildSuperPrompt 本体（task4 改）；不实现 SSE/API 新端点（J1=A 裁决）
