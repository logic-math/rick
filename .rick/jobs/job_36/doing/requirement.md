
基于 PI 的扩展，开发一个专用的 rick  web ui 支持云端同步，多端适配，完备的 rick 功能 
---

## Grilling 澄清结论（2026-08-29，设计树与调研简报见 doing/grilling/）

**原始需求**：基于 PI 的扩展，开发一个专用的 rick web ui 支持云端同步，多端适配，完备的 rick 功能

### 架构终判

1. **架构形态 = C**：Go 服务（`rick web` 子命令）+ pi rpc/json 子进程。「基于 PI 的扩展」落地为基于 pi 生态能力（rpc 协议/SDK 能力面），非字面 pi extension（extension 单 session 生命周期无法承载多 job，调研已证）。复用社区：ygncode/pi-web（Go 侧摘代码移植：rpc client/workers manager/SSE registry/embed 方案）、pi-web-ui（React 组件族）、dashboard（事件序号断线续传协议+渲染栈）
2. **部署 = 中心实例**：单份状态在服务端，多端浏览器访问同一实例；无文件级双向同步（opencode 双实例状态分裂为反面教材）
3. **session 中心模型**：基本组件=session（聊天窗口事件流），类型=rick cmd；交互聊天型（plan/easy/ctrl/human-loop/learning）走 pi --mode rpc 常驻子进程；监控型（doing；dream 后台模式）= goroutine + --mode json + task 看板/事件流；dream 支持双模（交互/后台，启动参数选）；closed 会话离线读 session JSONL 浏览，resume = spawn + --session
4. **多工作区**：`rick web` 全机唯一（任意目录启动，pid+端口双重检测）；工作区=含 .rick 的目录，web UI 手动注册（校验含 .rick，机器级持久化 ~/.rick/web.json）+ 最近会话目录建议；一会话锚定一工作区（pi 子进程 cmd.Dir=工作区），一工作区多会话，多工作区同屏、多 job 跨工作区并行；同工作区并发 doing 不做互锁（用户自行保证）
5. **传输 = 单流多路复用 SSE**（GET /api/events + Last-Event-ID 重放缓冲 + 心跳）；交互命令走 POST；extension_ui 双向桥（select/confirm/input/editor → web 弹窗表单）
6. **云端同步 = agent 自主 git 行为**：web UI 无同步控制面；协议不改动（level_complete 已有 commit，push 由 agent 自主判断）
7. **认证 = 单用户 token**：~/.rick/config.json 新增 web_token（首次启动自动生成）；API Bearer + SSE ?token=；默认监听 127.0.0.1:6137（C-137 彩蛋），--listen/--port 显式开放；HTTPS 交反向代理（文档化）
8. **前端 = React SPA**（Zustand+react-router+Vite+Tailwind 4；渲染栈 react-markdown+remark-gfm+rehype-highlight+ansi-to-react+@git-diff-view+dompurify）；**Rick and Morty 动漫深色主题**（飞碟/动态星空+流星/绿色传送门；prefers-reduced-motion 降级）；PWA（vite-plugin-pwa）；布局=左侧栏（多工作区+会话列表）+顶部功能区（Sessions/Jobs/Knowledge）；响应式 768/1024 断点
9. **分发 = embed 混合 + 自迭代**：go:embed 同时内嵌 web/src+web/dist 为 baseline；~/.rick/web/ 可写覆盖层（优先服务）；**用户可通过对话让 agent 修改前端源码→npm 构建→fs watcher→SSE reload→自动生效**；`rick web customize`（env 幂等抽取 baseline 源码）/`rick web reset`（复位）；dist 提交仓库（make web-dist），node 不进 rick 构建链
10. **功能范围 = P0+P1+P2**：P0 jobs/tasks 看板+doing 监控+debug/act-path 查看+knowledge（domain/loops/skills）浏览；P1 全部交互会话（plan/easy/ctrl/human-loop/learning）+dream 双模；P2 触发与结果查看。P3（tools init-pi/theme web 化）明确排除
11. **rick 四层架构遵从**：cmd 层 rick web 命令 → handler 层 web server 编排+会话管理（handler 抽取路径参数化 core，CLI 零行为变化）→ runtime 层 pi rpc supervisor（per-session worker：Setpgid/SIGTERM→5s→SIGKILL/get_state 心跳/respawn 补流/上限8）→ env 层前端基座部署（deployRickAgents 同款幂等模式）
