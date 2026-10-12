# 依赖关系
无

# 写域
internal/web/archived.go
internal/web/jobs.go
internal/web/routes.go
internal/web/archived_test.go
web/src/components/jobs/JobsList.tsx
web/src/routes/Jobs.tsx

# 任务目标
归档语义升级（用户定义）：**归档 = 被 dream 学习过的 job**——`<ws>/.rick/dream/dream_run_{job_id}_log.md` 存在即视为已归档（自动）；手动归档保留（补充手段）。归档 job 不在工作区下方（Jobs 主列表）显示，在 Jobs 页「已归档」区找到。

# 关键结果

## 后端
1. `internal/web/archived.go`：
   - 现有 `ArchivedStore`（手动归档 ~/.rick/web/archived.json）保留
   - 新增**自动归档判定**：`DreamArchivedJobs(rickDir string) map[string]bool`——扫 `<rickDir>/dream/` 下 `dream_run_{job_id}_log.md`（复用 internal/workspace 的 GetDreamProcessedJobs——查其导出性，可用则 import；不可用则自实现同款扫描：prefix dream_run_ + suffix _log.md，取 job_id）
2. `internal/web/jobs.go`：`ListJobsFiltered(rickDir, archived []string)` 改为接受**手动+自动**：调用方（routes）先算 `manual := ArchivedStore.List(wsID)` + `auto := DreamArchivedJobs(rickDir)`，合并去重后过滤；JobSummary 加 `Archived bool`（含手动/自动任一）与 `ArchivedBy string`（"manual"/"dream"——前端标注来源）
3. `internal/web/routes.go`：GET jobs 默认过滤（手动∪自动）；`include_archived=true` 返回全部 + Archived + ArchivedBy；archive/unarchive API 保留（手动）；自动归档的 job 调 unarchive 无效（仍被 dream 判定归档——返回说明或允许 unarchive 但列表仍隐藏？——设计：unarchive 只清手动标记，dream 判定仍归档；响应 200 但 ArchivedBy=dream 说明）
4. 测试：dream 日志存在 → 自动归档过滤；手动+自动合并；include_archived 带 ArchivedBy

## 前端
5. `web/src/types.ts`：JobSummary 加 `archived_by?: "manual" | "dream"`
6. `Jobs.tsx`/`JobsList.tsx`：
   - 主列表默认不含已归档（现状 ✓——后端过滤）
   - 「已归档 (N)」折叠区：显示归档 job，**来源标注**（🏭 dream 已学习 / 📦 手动归档）；dream 归档的 job 显示「已由 dream 学习」标签（不显示恢复按钮或恢复后仍隐藏的提示——设计：dream 归档的显示「dream 已处理」标签 + 无恢复按钮；手动归档的保留恢复按钮）
   - 归档区刷新：打开折叠区时拉 include_archived=true
7. 无页面错误

# 测试方法
- go build ./... + go test ./internal/web/...（含 dream 归档测试）
- npx tsc --noEmit + npm run build
- playwright（6173 验证实例，HOME=/tmp/rick-e2e-home——test 工作区 .rick/dream/ 下造 dream_run_job_X_log.md fixture）：该 job 从主列表消失、归档区出现且标注「dream 已学习」

# 上下文提示
- dream 日志格式：`<rickDir>/dream/dream_run_{job_id}_log.md`（internal/workspace/dream.go:40 getDreamProcessedJobs 的解析逻辑——前缀 dream_run_、后缀 _log.md、job_id 形如 job_5）
- 现有归档 UI（手动按钮 + 已归档折叠区 + 恢复）已实现——本次升级语义（dream 自动归档 + 来源标注）
- 手动归档按钮保留（用户说「应该支持归档操作」——手动是操作，dream 是自动）
