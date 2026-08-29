# 依赖关系
task7

# 写域
web/src/components/monitor/
web/src/components/jobs/
web/src/components/knowledge/
web/src/components/common/

# 任务目标
监控视图（doing/dream 后台型）+ jobs 看板 + knowledge 浏览 + 公共组件。

# 关键结果
1. `web/src/components/monitor/MonitorView.tsx`：后台型会话主视图 props{sessionId}——头部（类型徽标+Saucer 飞碟运行指示动画：running 时飞碟悬停+光束、closed 停止；Abort 按钮）；主体双栏（移动端上下堆叠）：左 TaskBoard 右 EventStream
2. `monitor/TaskBoard.tsx`：task 状态看板——任务卡（task_id/name/status 徽标：pending 灰/running 传送门绿脉动/success Rick 蓝灰/error Morty 黄+重试角标；commit_hash 短哈希展示）；jobs_update diff 事件驱动高亮变更（闪一下）；GateResult.tsx：gate 通过/失败横幅（事件流里 gate 输出解析：含「✅/⛔/gate」关键词的事件渲染为横幅卡）
3. `monitor/EventStream.tsx`：编排事件流（session_event 中非 chat 类：工具调用行/派发行/门禁行——精简单行卡+时间戳，最多保留 500 行滚动）；区别于 ChatView 的富渲染——监控视图密度优先
4. `web/src/components/jobs/JobsList.tsx`+`JobDetail.tsx`：工作区 jobs 列表（卡片：job_id/更新时间/任务进度点阵——每 task 一个色点）+ 详情（TaskStatus 表：复用 TaskBoard 任务卡横排简化版 + job 文件浏览器：plan//doing/ 文件树（JobFiles.tsx）点开看 markdown 源码/日志（含 debug/*.md、act-path.md、raw_session_coding.log——复用 knowledge 的 MarkdownView/FileTree 组件）
5. `web/src/components/knowledge/KnowledgeBrowser.tsx`+`FileTree.tsx`+`MarkdownView.tsx`：三根（domain/loops/skills）文件树（缩进列表+文件图标）+ 右侧 markdown 渲染（react-markdown 同 chat 栈；代码高亮）；移动端树折叠为下拉
6. `web/src/components/common/`：Dialog（模态基类——extui/NewSessionModal 复用）/Button/Spinner（传送门旋涡 loader——Portal 组件复用）/ErrorBanner/EmptyState（空态插画：飞碟+文字）/StatusDot
7. 视觉同 R&M 主题；`npx tsc --noEmit` 通过（npm run build 留给 gate3 层完成后统一执行——同层 task9 并行 build 会踩 dist 竞争）

# 测试方法
tsc+build 全绿；手工 smoke：占位数据目检 TaskBoard 状态流转/文件树/monospace 日志渲染

# 上下文提示
- doing 事件流数据源=doingCore progress 回调（task11 后端产出 jobs_update + session_event），组件只管渲染
- JobFiles/KnowledgeFile 走同一 file API（api-contract Jobs.file 与 Knowledge.file）——FileTree 做成通用组件两处复用
- raw_session_coding.log 可能大（>2MB 截断 400）——前端对超限显示「文件过大，请在仓库查看」
