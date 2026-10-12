
## debug1: ctrl 命令实现 — 零问题首次成功

**现象 (Phenomenon)**: 新功能实现，无 bug

**复现 (Reproduction)**: N/A

**猜想 (Hypothesis)**: N/A

**验证 (Verification)**: `go build ./...` 通过，`go run cmd/rick/main.go --job job_15 --dry-run ctrl` 输出完整 prompt，变量替换正确

**修复 (Fix)**: N/A

**进展 (Progress)**: ✅ 已解决

## debug2: ctrl prompt 缺少日志文件路径和 NDJSON 格式说明

**现象 (Phenomenon)**: 初版 ctrl.md 模板只写了 `raw_session_coding.log` 路径，未说明 NDJSON 格式，未提 `act-path.md`

**复现 (Reproduction)**: 用户指出 doing/tasks/{task_id}/ 目录结构未在 prompt 中说明

**猜想 (Hypothesis)**: 模板内容不足，agent 无法正确解析 NDJSON 日志

**验证 (Verification)**: 阅读 `internal/agent/claudecode/executor.go` 和 `internal/actpath/generator.go` 确认两类文件的完整结构

**修复 (Fix)**: 重写 ctrl.md，补充：目录结构图、NDJSON 四种 type 说明、act-path.md 内容说明、四个干预场景（A/B/C/D）、汇报格式示例

**进展 (Progress)**: ✅ 已解决

## debug3: super-debugging skill 合并实现 — 零问题

**现象 (Phenomenon)**: 新功能实现，无 bug

**复现 (Reproduction)**: N/A

**猜想 (Hypothesis)**: N/A

**验证 (Verification)**:
- `go build -o bin/rick` 通过
- `bin/rick --dry-run plan "测试"` 输出包含 `# 调试方法` 和 doing-prompts 路径
- `bin/rick --dry-run --job job_1 doing` 输出包含 `skill:super-debugging` 三处引用，路径为真实 doing/prompts/ 路径
- `skills/` 目录确认只剩 `super-debugging-zh.md`，旧两文件已删除

**修复 (Fix)**: N/A

**进展 (Progress)**: ✅ 已解决
