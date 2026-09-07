# 依赖关系
无

# 写域
internal/web/routes.go
internal/web/routes_test.go
web/src/routes/Sessions.tsx
web/src/api/client.ts
web/src/components/common/Dialog.tsx（如需扩展）

# 任务目标
多级路径选择器：让用户像文件对话框一样逐级浏览目录，将任意（无 .rick 的）目录创建为工作区。

# 关键结果

## 后端（internal/web/routes.go）
1. `GET /api/fs/list?path=<dir>` → 200 `{"path":"/a/b","parent":"/a","entries":[{"path":"/a/b/c","name":"c"},...]}`
   - 列出 path 下的**子目录**（非文件；排除以 . 开头的隐藏目录；上限 100；按名字排序）
   - path 不存在/非目录 → 400 invalid_params；权限错误 → 500
   - parent = filepath.Dir(path)（根目录 parent 为空串）
2. `GET /api/fs/status?path=<dir>` → 200 `{"path":"/a/b","exists":true,"is_dir":true,"has_rick":true}`
   - has_rick = path/.rick 存在且为目录；exists/is_dir 按 os.Stat
3. routes_test.go 补两接口测试（tmpdir fixture：子目录列举/排除隐藏/上限、status 三态）

## 前端（web/src/routes/Sessions.tsx + web/src/api/client.ts）
4. client.ts 加 `fsList(path)` / `fsStatus(path)` 方法（类型：FsListResult/FsStatus）
5. AddWorkspaceCard 增加「🖥 多级选择」按钮（与 注册/浏览/创建 并列，primary 强调——用户反馈现在创建入口不醒目）：
   - 点击打开 Dialog 内嵌**目录浏览器**：顶部当前路径（面包屑式可点击上级）+ 子目录列表（每项一行，点击进入该目录）+「⬆ 上级」按钮 + 「新建子目录」行（输入名字 + 创建按钮，调 fsList 刷新）
   - 初始路径 = 输入框当前值（空则 / 或 ~）；每级进入都调 fsList
   - 底部「使用此目录」按钮 → 回填主输入框 path + 关闭浏览器；根据 fsStatus 自动提示：has_rick=「将注册为工作区」/ 无 .rick=「将创建新工作区（初始化 .rick 结构）」
6. 「创建新工作区」按钮保持（直接输入路径创建）；弹窗内同时清晰展示三种操作（注册=已有 .rick / 创建=无 .rick 初始化 / 浏览=搜索含 .rick 目录 / 多级选择=文件对话框式导航）——按钮组布局加强，创建入口醒目（非 ghost 小按钮）
7. 多级浏览器在移动端可用（Dialog 内纵向滚动）

# 测试方法
- go build ./... + go test ./internal/web/...（含新测试）
- npx tsc --noEmit + npm run build
- playwright 冒烟（6173 验证实例）：点「+ 添加工作区」→ 点「多级选择」→ 目录浏览器出现 → 从 / 或输入路径进入 → 逐级点进子目录 → 「使用此目录」回填 → 创建（无 .rick 目录）成功加入工作区树

# 上下文提示
- 现有：GET /api/workspaces/browse（含 .rick 过滤的深度1搜索）、POST /api/workspaces/create（createWorkspaceDirs 初始化 .rick 六目录 + 注册）
- fs/list 与 browse 的区别：browse 只列含 .rick 的（供「选择已有工作区」）；fs/list 列全部子目录（供逐级导航）
- Dialog 已修复为 Portal 渲染（全屏）——多级浏览器直接复用 Dialog
- 后端写域 routes.go（两接口 + 测试）与前端写域互斥（Sessions.tsx/client.ts）
