# 依赖关系
无

# 写域
web/src/components/layout/WorkspaceTree.tsx
web/src/components/layout/WorkspaceListNode.tsx（新建，如需拆）
web/src/components/layout/SessionBadge.tsx（新建）

# 任务目标
侧边栏重构（用户反馈：不要选择器式切换，要把所有工作区完整展示——最上一级 = 全部工作区列表，点击展开看到 sessions/jobs/dreams 与未完成会话）。

# 关键结果

1. **移除「当前工作区选择器」模式**（WorkspaceTree v3 的顶部下拉 + localStorage 记忆）→ 改为「全部工作区平铺列表」：
   - 最上一级：每个已注册工作区一行（名称 + 状态点：该工作区有 active/running 会话时绿点提示 + 未完成会话数徽标）
   - 点击工作区行 → 展开/折叠其子项（默认折叠；折叠状态本地 useState 记录）
   - 展开后子项：Sessions / Jobs / Dreams 三导航（保留 /ws/:id/* 路由）+ 该工作区会话列表
2. **会话列表三态展示**（SessionBadge 组件）：每个会话一条：类型徽标（PLAN/EASY/CTRL/...）+ 标题 + **状态徽标三态**：活跃（active/running=传送门绿·呼吸点）/ 终止（error=Morty 黄·「已中断」）/ 完成（closed=灰·「已完成」）；点击进入 /session/:id
3. 会话状态实时同步：SSE session_state 事件驱动徽标更新（现有 sessions store 已处理——确认列表渲染消费 store 状态）
4. 「+ 添加工作区」按钮保留（底部常驻）；「+ 新建会话」按钮保留（当前展开的工作区新建）
5. 移动端：drawer 内同布局；工作区行触屏友好（整行可点）
6. 空态（无工作区）不变

# 测试方法
npx tsc --noEmit + npm run build；playwright 冒烟（6173 验证实例）：侧边栏显示全部工作区（test/rick/new-ws-demo 等每行一个）→ 点击某工作区展开 → 看到 Sessions/Jobs/Dreams + 会话列表（含 error 会话显示「已中断」徽标）→ 再点收起

# 上下文提示
- 现有 WorkspaceTree.tsx 是「当前工作区选择器」模式（v3）——重写为平铺列表
- 三态语义（与 task-session-state 对齐）：active/running=活跃、error=终止（已中断）、closed=完成
- React #185 教训：selector ?? [] 用模块级常量
- 路由保持 /ws/:id/* 与 /session/:id（不破坏现有导航）
