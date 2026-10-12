# 写域
web/src/**（components/sessions/、components/layout/SessionBadge.tsx、routes/Sessions.tsx、types.ts、api/client.ts、stores/sessions.ts）

# 任务目标
会话「人工归档」前端：每个会话 ⋮ → 会话设置（信息 + 归档/恢复）；Sessions 页归档折叠区（分页查询 + 恢复）；侧栏/列表行入口。

# 后端契约（backend worker 同步实现中——按此对接，勿改后端）
- GET /api/sessions?workspace=X（默认）→ 裸数组（**不含已归档**）
- GET /api/sessions?workspace=X&archived=true&limit=20&offset=0 → {items: SessionInfo[], total, limit, offset}（created_at desc）
- POST /api/sessions/{id}/archive → 204（幂等；active 会话会先终止变 closed 再归档）
- POST /api/sessions/{id}/unarchive → 204（幂等；恢复出列表）
- SessionInfo 新增字段：archived?: boolean；archived_at?: string
- resume 已归档会话自动清归档

# 改动清单
1. types.ts：SessionInfo + archived?/archived_at?
2. api/client.ts：
   - archiveSession(id): POST /api/sessions/{id}/archive
   - unarchiveSession(id)
   - listArchivedSessions(workspaceId, {limit, offset}): Promise<{items: SessionInfo[]; total: number}>（GET archived=true）
3. 新组件 components/sessions/SessionSettingsDialog.tsx：
   - props {session, open, onClose, onChanged}
   - 内容：dl 信息（类型徽标/状态徽标/标题/创建时间/pi_session_id 摘要/params 摘要）+ 动作：
     - 未归档 → 「归档」按钮（danger 样式）：文案按状态：active/running「结束任务并归档（worker 将终止）」/closed|error「归档（标记完成，从列表隐藏）」；确认后调 api.archiveSession → onChanged
     - 已归档 → 「恢复」按钮 → api.unarchiveSession → onChanged
   - busy 态 + 错误显示
4. 侧栏入口：components/layout/SessionBadge.tsx——行结构从 NavLink 改为相对容器（div 包 NavLink + ⋮ 按钮 stopPropagation hover 显示，仿工作区 ⋮ 样式）→ 打开 SessionSettingsDialog；归档成功后 store load(workspaceId) 刷新（见 6）
5. Sessions.tsx（/ws/:id/sessions）：
   - 会话卡片行尾加 ⋮ 入口（复用 SessionSettingsDialog）
   - 「已归档会话」折叠区（默认折叠，展开后拉 listArchivedSessions 分页：每页 20；总数 total；「‹ 上一页 / 下一页 ›」；每条：类型徽标+标题+状态(closed)+相对时间+「恢复」按钮+点击进详情）；恢复成功刷新本区 + 通知主列表刷新
6. stores/sessions.ts：archive/unarchive 成功后调用 load(workspaceId) 重拉（byWorkspace 默认不含归档；active 会话归档后从列表消失）；或加 applyArchived(id) 本地剔除（若 load 幂等简单则用 load）
7. 归档会话若在侧栏 WorkspaceListNode 会话列表里（旧数据未刷新）——依赖 load 刷新即可

# UI 细节
- 样式对齐 R&M 主题（Dialog/危险按钮参考 components/common/Dialog + 工作区设置弹层 WorkspaceListNode 内嵌样式）
- 会话状态徽标：active=绿活跃 / running=绿 / error=黄已中断 / closed=灰已完成（复用 SessionBadge.StatusBadge 语义——它未导出则本文件内写）
- 空归档区文案「暂无归档会话」
- 勿破坏：Resume 流程、侧栏展开计数（incomplete = active/running/error 不变——归档会话不计）

# 测试方法
- npx tsc --noEmit + npm run build
- playwright（6173/8412 verify）：真实会话行 ⋮ 打开设置 → 归档（确认）→ 从侧栏/列表消失 + 归档区出现（total 增加）；恢复 → 回列表；分页翻页正常；无 JS 错误

# 纪律
不碰 git；回执：改动清单 + 验证结果
