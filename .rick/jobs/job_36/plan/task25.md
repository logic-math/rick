# 依赖关系
task23

# 写域
web/src/types.ts
web/src/components/sessions/NewSessionModal.tsx
web/src/components/layout/SessionBadge.tsx
web/src/components/sessions/SessionSettingsDialog.tsx
web/src/lib/rsi.ts

# 任务目标
在 Web UI 提供「RSI 自进化」入口：新建会话时可选 `rsi` 类型，界面明确说明「加载 rick-rsi-loop 驱动自进化」，并对 workspace 做**前置提示**（引导选择 dev 工作区），避免用户在生产仓库根上启动而被后端拒绝。

# 关键结果
1. `web/src/types.ts`：`SessionType` 加 `"rsi"`。
2. `web/src/lib/rsi.ts`：`isRickSourceTreeWorkspace(ws)`（用工作区路径启发式判断，例如路径以 `-dev` 结尾或包含 `rick`；仅用于 UI 提示，**权威判定在后端**）+ `rsiWorkspaceHint(ws)` 返回中文提示文案。
3. `NewSessionModal.tsx`：
   - 类型列表加入 `rsi`（label `RSI 自进化`，描述「加载 rick-rsi-loop：隔离 dev 开发 → 门禁 → 人类确认 → release」）；
   - 选中 `rsi` 时：只显示 workspace 选择（隐藏 requirement/job 字段）；对非 dev 工作区显示**醒目提示**「RSI 会话必须指向 dev 工作区（后端会拒绝生产仓库根）」；
   - 提交后的 400 错误直接展示后端中文 message（不吞）。
4. `SessionBadge.tsx` / `SessionSettingsDialog.tsx`：`rsi` 的类型徽标（label `RSI`，tone 用 `nebula` 紫，与 dream 同色系但文案区分）+ 会话设置里显示「本会话由 rick-rsi-loop 驱动」。
5. 必须通过 `npx --prefix web tsc --noEmit -p web` 与 `npm --prefix web run build`。

# 测试方法
npx --prefix web tsc --noEmit -p web && npm --prefix web run build
go build ./...
python3 .rick/jobs/job_36/plan/gates/gate12.py

# 上下文提示
- 后端契约（task24 已定）：`POST /api/sessions {"workspace_id":"...","type":"rsi"}`；错误 400 + `{error:{code:"invalid_workspace"|"invalid_params",message:"中文原因"}}`。
- 现有类型 UI 参照 `human-loop`（`NewSessionModal.tsx` 的 `cmdType` 分支与 `missing` 校验列表）。
