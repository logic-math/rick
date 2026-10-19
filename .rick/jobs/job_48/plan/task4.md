# task4：中控会话提示词增强 + e2e 冒烟

## 任务目标

BuildSuperPrompt 注入**工作区根上下文**（super_env 路径/env 生成约定/观察与干预手段），使中控会话开箱即知全部操作面；e2e 验证 web 创建 → prompt 严格加载 → CLI 互通。

## 关键结果

- `internal/super/session.go` BuildSuperPrompt 增强（prompt 模板段）：
  - 工作区上下文块：当前工作区根（ws 上层的 rick 工作区）、super_env 路径 `<工作区>/.rick/super_env/<name>/`、env 生成约定（gen_env skill 引导，产出落 `super_env/<name>/env/`）
  - **观察手段**（J1=A）：CLI（`rick super status --name X --json` / `event` 投递）+ **直接读文件**：`super_env/<name>/events.jsonl`（事件流真相源）、`archive.json`（精英格档案）、`codes/<iter-*>/`（每代代码快照 + prompt + agent 产物）、`heartbeat.json`（心跳）、以及 pi agent 的执行 session 文件——**任意信息可读**，只要能描绘当前运行状态
  - 干预手段：`rick super event --name X --type <t> --payload '<json>'`（队列投递，单写者保证）
- CLI `rick super --resume` 与 web 会话互通验证（session_id 契约已有）
- `super/tests/super_session_test.py`（或并入 e2e）：e2e 冒烟——
  1. fake rick 工作区（含 .rick/loops/super-loop.md + skills/gen_env_skill/skill.md + super_env/）
  2. 构建临时 rick 二进制 → 模拟 web prep 路径（或直接调 CLI `rick super` 前台 prompt 生成）→ 断言 prompt 文件含 gen_env skill 全文关键词（如「门禁系统设计引导」）+ super_env 路径 + 观察手段清单
  3. CLI 互通：`rick super --resume <session_id>` 能加载（pi 不真跑，只验证 session_id 文件与 --session 参数链）
- 若 task2 的 Go 测试已覆盖 prompt 内容断言，本 task e2e 聚焦 CLI 侧与端到端链

# 写域

- internal/super/session.go
- super/tests/super_session_test.py（新建）或 super/tests/e2e_test.py 追加用例

## 依赖关系

- task2（prep 链就位）

## 实现要点

- 提示词块保持简洁（中控是 agent 的 SOP 不是文档）；「直读文件」列出关键文件与一句话语义即可
- super_env 路径在 prompt 中用绝对路径（会话内 agent 免解析歧义）

## 测试方法

`go build ./... && go test ./internal/super/ -run TestBuildSuperPrompt -timeout 120s` + `python3 super/tests/super_session_test.py`（或 unittest 形态）全绿

## 禁止事项

- 不改 BuildSuperPrompt 的严格加载语义（缺失报错在 task2 层）；不做 SSE；不改 py 调度器
