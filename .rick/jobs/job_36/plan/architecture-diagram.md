# rick web 架构图与实施路线（job_36 · grilling 终判 + 流水线设计配套）

> 详图见 doing/grilling/design-tree.md（设计树）与 plan/api-contract.md（API 契约单源）。
> 本文件为 human 确认的架构总览快照（2026-08-29）。

## 一、系统总体架构

- 多端浏览器（React SPA · R&M 深色主题 · PWA）
  - 页面：Sessions / Jobs / Knowledge / Settings；api/client.ts（REST+Bearer）+ api/sse.ts（EventSource）
- rick web 中心服务（Go · 全机唯一 · 127.0.0.1:6137 · 单用户 token）
  - internal/web：auth / routes / sse（Hub 单流多路复用+Last-Event-ID 重放）/ sessions（SessionManager）
    / jobs（只读数据层）/ static（~/.rick/web/dist 覆盖层优先，embed baseline 兜底）/ watcher / registry
  - internal/cmd/web.go + internal/handler/web.go：cobra 命令 + Web 编排（singleton pid+端口）
- 会话执行层（internal/runtime）
  - RpcClient（JSONL 命令/事件）+ Supervisor（每 session 一 worker：Setpgid/SIGTERM→5s→SIGKILL/心跳/respawn）
  - 交互型=pi --mode rpc 常驻；监控型（doing/dream 后台）=pi --mode json 一次性（cmd.Dir=工作区）
- handler core（task2 重构：显式 rickDir 注入，CLI 零行为变化）+ builder（prompt 产出）
- 状态层：.rick/（git 仓库=云端同步载体，agent 自主 push）+ ~/.rick/pi/agent/sessions/（closed 离线浏览源）

## 二、会话生命周期

新建（cmd 按钮+参数弹窗）→ SessionManager 分派：
- 交互型（plan/easy/ctrl/human-loop/learning/dream交互）：Spawn rpc worker + bootstrap prompt
- 监控型（doing/dream后台）：goroutine + core(ctx, progress)
- 事件：pi 事件透传+jobs_update diff+extension_ui → SSE Hub 单流 → 浏览器按 session 分发
- close（优雅终止）/ resume（--session 重启+get_entries 补流）/ closed 离线读 JSONL

## 三、前端自迭代闭环

embed（src+dist baseline）→ rick web customize 抽取到 ~/.rick/web/src → agent 会话对话改造 →
npm build → watcher → SSE frontend_reload → 自动生效 → reset 可回 baseline

## 四、实施路线（15 task / 6 层，gate 驱动）

- L1：task1 rpc / task2 handler core / task3 前端骨架 / task5 registry
- L2：task4 supervisor / task6 jobs 数据层 / task7 前端 api+stores / task8 SSE+auth
- L3：task9 chat 组件 / task10 monitor+jobs+knowledge 组件 / task11 会话 REST（后端心脏★）
- L4：task12 前端组装+PWA / task13 server 组装
- L5：task14 env 基座+cmd/handler 收口
- L6：task15 dist 提交+E2E+文档

前后端两线 L1-L3 并行，L4 合流，L5 收口，L6 终验；同层写域互斥并行，层出口 level_complete（gate→commit）。

