## [job_35] Domain 事实同步 - 2026-08-17

### 新增已知问题与解法
- **`git add bin/rick` 静默失败（bin/ 在 .gitignore）**：必须用 `git add -f bin/rick` 强制暂存，否则 feat commit 漏二进制（来源：domain/bugs.md）
- **tasks.json `updated_at` 缺时区导致 Go time.Parse 失败**：每条 `updated_at` 必须带 RFC3339 时区（`+08:00`），mark_task_success.py 已用 `timezone(timedelta(hours=8))` 自动修复（来源：domain/bugs.md）

### 新增架构/构建事实
- **四层架构 + 5 模块 + env 四职责契约**：cli → handler → builder → runtime/env/workspace/prompt 四层，pi 调用逻辑收口到 runtime 层（来源：domain/rick-spec.md）
- **构建/门禁命令链**：`go build -o bin/rick ./cmd/rick` + `git add -f bin/rick` + `go test ./... -timeout 120s` + `python3 .rick/skills/rick-gates/helper.py .rick/jobs/job_N/doing`（来源：domain/build.md）

### 其他新增事实
- rick 自定义 agent（think/research/exporter）经 env 职责 3 `deployRickAgents` 幂等落盘到 `~/.rick/pi/agent/agents/`（来源：domain/env.md）
