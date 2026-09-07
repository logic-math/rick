# 依赖关系
无

# 写域
web/src/api/client.ts
web/src/types.ts
web/src/components/jobs/JobsList.tsx
web/src/routes/Jobs.tsx

# 任务目标
job 归档前端：已完成 job 卡片加「归档」按钮；已归档列表可查看/恢复；避免 job 堆积。

# 关键结果

1. `web/src/api/client.ts`：`archiveJob(wsId, jobId)` / `unarchiveJob(wsId, jobId)` / `listJobs(wsId, includeArchived?)`（加 includeArchived query 参数）；`web/src/types.ts` JobSummary 加 `archived?: boolean`
2. `web/src/components/jobs/JobsList.tsx`：
   - 每个 job 卡片加「归档」按钮（仅 status=success 的 job 显示——已完成才可归档；按钮小号 ghost：📦 归档；点击 → archiveJob → 列表移除该卡 + 轻提示「已归档」；防连击 busy）
   - 非 success 的 job 不显示归档按钮（tooltip 说明「完成后可归档」）
3. `web/src/routes/Jobs.tsx`：
   - 顶部加「已归档 (N)」折叠区（默认折叠）：显示已归档 job 列表（include_archived=true 拉取），每项带「恢复」按钮（unarchiveJob → 从归档区移除 + 回主列表）；空时折叠区不显示
   - 主列表用 listJobs(wsId)（默认过滤归档）
4. 归档/恢复后列表刷新（store 或本地 state 更新）；移动端按钮可用
5. 无页面错误；后端接口 409（非 success 归档）时显示错误提示

# 测试方法
npx tsc --noEmit + npm run build；playwright 冒烟（6173 验证实例）：Jobs 页成功 job 卡片有「归档」按钮 → 点击 → 卡片消失 + 已归档区出现 → 恢复 → 回主列表

# 上下文提示
- 后端并行 task 实现 archive/unarchive/include_archived 接口（契约如上）
- 现有 JobsList：卡片（job_id/进度点阵/完成度）+ onOpen 跳详情；Jobs.tsx 有工作区上下文（路由 /ws/:id/jobs）
