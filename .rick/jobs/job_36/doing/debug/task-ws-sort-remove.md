# 依赖关系
无

# 写域
internal/web/registry.go
internal/web/routes.go
internal/web/registry_test.go
web/src/components/layout/WorkspaceTree.tsx
web/src/api/client.ts

# 任务目标
①工作区拖拽排序（顺序持久化）；②工作区注销（从 rick 移除注册，不删目录）。

# 关键结果

## 后端
1. `WorkspaceRegistry.Reorder(ids []string) error`（registry.go）：按 ids 顺序重排 Items（校验 ids 与现有集合一致——多/少/未知 id → ValidationError 400；空集合不改变）；原子 save
2. `PUT /api/workspaces/order {ids:[...]}` → 204（routes.go 挂载，authWrap）；测试补：Reorder 持久化/校验错误
3. 确认 Remove 只删注册表（现状 ✓——不碰目录），API 注释说明「注销=移除注册，目录保留」

## 前端
4. `web/src/api/client.ts`：`reorderWorkspaces(ids: string[])`（PUT /api/workspaces/order）
5. `WorkspaceTree.tsx`：
   - **拖拽排序**：工作区行 draggable（HTML5 DnD：dragstart/dragover/drop；拖到目标行前后 → 本地重排 + 调 reorderWorkspaces 持久化 + 失败回滚）；拖拽中行高亮（portal 绿边框）；移动端不启用 DnD（touch 手势冲突——用长按菜单替代或直接不加，说明）
   - **注销按钮**：工作区行 hover 显示「注销」按钮（🗑 或「移除」——文案用「注销」），点击 → 确认弹窗（Dialog：「注销后该工作区从 rick 移除，目录与 job 文件保留，可随时重新添加」）→ DELETE /api/workspaces/{id} → 刷新列表；注销后若当前路由在该工作区下 → 重定向到第一个工作区或空态
   - 顺序来源：List 返回顺序即展示顺序（现状）；reorder 后刷新列表
6. 无页面错误；DnD 状态干净（拖拽结束清理）

# 测试方法
- go build ./... + go test ./internal/web/ -run TestWorkspaceRegistry（含 Reorder）
- npx tsc --noEmit + npm run build
- playwright（6173 验证实例）：侧边栏工作区行 hover 出现「注销」→ 点击 → 确认 → 工作区消失（目录仍在——ls 验证）；拖拽排序：用 CDP/鼠标事件模拟 dragstart→dragover→drop（playwright mouse.move/down/up）→ 顺序变化 + 刷新后保持

# 上下文提示
- Registry.Items 数组顺序 = List 顺序 = 展示顺序（现状 List 返回数组序）
- Remove 已存在且只删注册表（registry.go:145）——前端入口是缺的
- 工作区行结构在 WorkspaceTree.tsx（v3 平铺列表——每行一个工作区 + 点击展开子项）
