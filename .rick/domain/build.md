# Build Domain

**最后更新**: 2026-08-17  **来源 Job**: job_35

项目构建/测试/门禁/提交的事实性命令。

## 事实列表

### 构建

- **构建仓库二进制**: `go build -o bin/rick ./cmd/rick`
  - 验证命令: `ls -la bin/rick`
  - 来源: job_35
  - 状态: ✅ 已确认

- **全量编译检查**: `go build ./...`
  - 验证命令: `go build ./... && echo BUILD_OK`
  - 来源: job_35
  - 状态: ✅ 已确认

- **⚠️ bin/rick 强制暂存（bin/ 在 .gitignore）**: `git add -f bin/rick`（`git add bin/rick` 静默 no-op）
  - 验证命令: `git status --short bin/`（应显示 M）
  - 来源: job_35 / task7
  - 状态: ✅ 已确认

### 测试

- **精确范围单元测试**: `go test ./internal/<pkg>/... -timeout 60s`（禁止无脑 `go test ./...` 全量，会混入依赖真实环境的测试）
  - 来源: job_35 / verify_go_changes_skill
  - 状态: ✅ 已确认

- **全量收敛测试**: `go test ./... -timeout 120s`
  - 来源: job_35
  - 状态: ✅ 已确认

- **集成测试脚本**: `bash tests/tools_integration_test.sh`
  - 来源: job_35 / task12
  - 状态: ✅ 已确认

### 门禁与验收

- **doing 门禁**: `python3 .rick/skills/rick-gates/helper.py .rick/jobs/job_N/doing`（tasks.json 可解析 / 无 zombie / success 有 commit_hash）
  - 来源: job_35 / task8
  - 状态: ✅ 已确认

- **task 验收脚本**: `python3 .rick/jobs/job_N/doing/tests/taskN.py`（输出 `{"pass": true, ...}`）
  - 来源: job_35
  - 状态: ✅ 已确认

## 已知问题与解决方案

### `git add bin/rick` 静默失败（未加 -f 漏提交）

**根因**: `bin/` 在 .gitignore，`git add bin/rick` 无报错但文件未暂存。

**精确解决步骤**:
```bash
go build -o bin/rick ./cmd/rick
git add -f bin/rick
git commit -m "chore(taskX): rebuild bin/rick"
```

**首次发现**: job_35  **验证状态**: ✅ 已修复

## update-pi（v4.1.0 新增）

- **更新 pi runtime**：`rick tools update-pi pi` —— 托管 runtime（`~/.rick/pi/agent/runtime`）走 rick 自己的 `npm install --prefix`（与 init-pi 同语义，绕开 `pi update --self` 对非全局安装的 guard）；无托管 runtime 时委托 `pi update --self`
- **更新全部扩展**：`rick tools update-pi extensions`（`pi update --extensions`，作用于 rick 托管 agent 目录，不碰用户 ~/.pi）
- **更新单个扩展**：`rick tools update-pi pi-subagents`（源名自动解析为注册形态 `npm:pi-subagents`；未注册报错并列出已注册项）
- **刷新模型目录**：`rick tools update-pi models`
- **全量更新**：`rick tools update-pi`（= all：pi → extensions → models，顺序固定）
- **更新后快速自检**（自动执行）：pi 版本 / 必需扩展注册 / rick agents+hooks 落盘 / rick-gates helper.py 语法 / human-loop 提示词渲染冒烟（非空且无 `{{` 残留）
- 注意：裸扩展名是注册行（`npm:pi-subagents`）的子串，解析必须 npm: 前缀形态优先，否则 `pi update pi-subagents` 可能子串误命中传裸名（env.resolveExtensionSource）
