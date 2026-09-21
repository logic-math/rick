# 设计树（Grilling — job_36: rick web ui 云端同步多端适配）

> 活文档：逐层追加（层号 + 模块 + pipeline + 判断节点）。根层 = OKR；下层按 L1-L5 loop 逐层展开。

## 根层（L0）— OKR

O（全局目标）：交付 rick 专用 Web UI（rick-web）：用户在浏览器中（桌面/移动多端均可）完整操作 rick 的核心功能（job 管理、plan/doing/easy/ctrl/human-loop/learning/dream 等），.rick 状态（jobs/知识库）支持云端同步实现多设备一致，最终以可验证状态交付（门禁测试全绿 + E2E 可演示）。

**KR 集（充分性推导：KR1 ∧ KR2 ∧ KR3 ∧ KR4 ∧ KR5 ⟹ O）**：

| KR | 内容 | 验证方式（初步） |
|----|------|------------------|
| KR1 架构与接入 | rick-web 服务架构落地：基于 pi 生态（扩展 / SDK / RPC / json 模式，具体形态待判断节点裁决）驱动 agent 会话，提供 HTTP/WS API + Web 前端，与 rick 现有 handler 层集成（rick-spec 已预留 WEB-UI 为第一层入口） | 架构设计文档 + 代码落地 |
| KR2 功能完备 | Web UI 覆盖 rick 全部核心命令功能面：jobs/tasks 可视化、plan/doing/easy 执行与实时轨迹流、ctrl 干预、learning/dream 触发、domain/loops/skills 知识浏览、tools 管理 | 功能清单逐项 E2E 验证 |
| KR3 云端同步 | .rick 状态的云端同步机制：多设备间 job/知识库状态一致（机制候选：git 远端 / 对象存储 / 同步服务——判断节点），含冲突处理策略 | 双端同步一致性测试 |
| KR4 多端适配 | 响应式 Web UI：桌面/移动浏览器布局自适应 + 触屏交互可用 | 移动端视口（DevTools 模拟）验证 |
| KR5 质量验收 | 测试与门禁：API 集成测试 + 前端构建验证 + 同步正确性测试，gate 全绿 | gate 脚本 exit 0 |

**OKR 充分性自检**：KR1 提供运行底座 → KR2 在其上覆盖功能 → KR3/KR4 满足"云端同步 + 多端适配"两个专项诉求 → KR5 保证可验证交付。五个 KR 联合 ⟹ O 达成。✅（KR 集划分维度：架构 / 功能 / 同步 / 适配 / 质量，MECE）

---

## 第 1 层 — 总体架构层

**调研**：research-L1.md（+详录 A/叶子 B/C/D）。

**模块与 pipeline（候选，主架构=C）**：

```
浏览器多端（桌面/移动响应式 SPA，embed 预构建 dist）
  ↕ HTTP + SSE/WS（LLM 事件流：text/toolcall delta，SSE+POST 为首选传输）
Web 服务层（`rick web`：API 路由 + embed 静态资源 + 会话管理 + 认证）
  ↕
会话执行层（pi 子进程：doing=--mode json 无头（现状复用）；plan/easy/ctrl/human-loop=--mode rpc 常驻会话，prompt/steer/abort/get_entries 游标增量回放）
  ↕
rick 桥接层（复用 internal/handler：jobs/tasks/tasks.json/gates/prompt builder）
  ↕
状态层（.rick/ 目录 = git 仓库，云端同步载体）
```

**关键调研结论（详见 research-L1.md）**：
1. pi extension 生命周期=单 session 存活期、一进程一活动 session ⇒ **extension 无法承载 rick 多 job**（「基于 PI 的扩展」只能取 pi 生态 SDK/RPC 之义，非字面 extension）
2. 竞品（OpenHands/goose/claude-code-webui 等）清一色 headless 事件流+自绘 UI，无一内嵌终端；PTY+xterm.js 移动端硬伤（软键盘无 Esc/Ctrl、IME 乱序），仅作兜底
3. 文件级双向同步遇并发写皆留冲突（git rebase / Syncthing .sync-conflict / rclone 无胜者）；tasks.json 双端并发写无自动解法 ⇒ 单写点或中心实例是正解方向
4. Go embed + 预构建 dist 提交仓库先例充分（AdguardHome/Vaultwarden/Syncthing）——node 不进 rick 构建链可守

**判断节点（待 human 裁决，L3 批量提问）**：
- J1 架构形态：A pi extension（已否决为主架构）/ B Node 常驻服务(SDK) / C Go 服务+pi rpc 子进程（推荐）/ D 混合
- J2 部署与同步语义：中心实例（云端/开发机单份状态，多端浏览器访问）vs 多机各自跑+状态云端同步
- J3 交互呈现：原生 web UI（事件流+chat+工具卡片）vs 终端模拟（xterm.js 兜底通道）
- J4 同步机制（依赖 J2）：git-based / 中心实例天然单写点
- J5 构建链原则：前端预构建 dist 提交仓库，node 不进 rick 构建
- J6 功能范围分期：完备功能（plan/doing/easy/ctrl/human-loop/learning/dream/tools）一期全做 vs 分期
- J7 认证：单用户 token vs 多用户

> 状态：**已终判（human 裁决 2026-08-29 + research-L1.md / research-L1-r2.md 调研支撑）**

**L1 终判（human 裁决固化）**：
- **J1=C**：Go 服务（`rick web` 子命令）+ pi rpc/json 子进程。补充调研确认：pi 官方 examples 零 HTTP/WS server 先例，web 服务层宿主侧自建是生态共识；pi 生态最强第三方 pi-web（⭐620，会话守护进程与 UI 分离）/ pi-agent-dashboard（⭐253）与 rick「Go 服务持子进程+浏览器纯视图」同构——终判可固守。
- **J2=中心实例**：单份状态在服务端，多端浏览器访问；避免文件级双向同步冲突（opencode 双实例状态分裂为反面教材印证）。
- **J3=原生 UI，session 中心模型**（human 定义）：基本组件=session（聊天窗口事件流）；session 类型=rick cmd（plan/easy/ctrl/human-loop/doing/dream 等）；工作区内点 cmd 按钮→参数弹窗→启动会话→聊天窗；支持 close/resume。调研补充：cmd 类型+参数宜作会话属性（窗内可改）而非前置表单门槛；渲染分层共识=文本 delta 直渲+工具折叠卡+thinking 折叠+错误二分+流式与终稿合并去重。
- **J4=git 快照备份 → 修订（human L2 轮裁决）**：**web UI 不控制 git push，同步由 agent 自主完成**（level_complete 已 commit；push 属 agent 侧行为）。KR3 云端同步重定义：agent 自主 git 操作（commit+push）保障 .rick 状态上云，web UI 无同步控制面。
- **J5=遵从 rick 四层架构**：`rick web`（cmd 层）→ handler（web server + pi 驱动编排）→ runtime（pi rpc 子进程 supervisor）→ env（前端产物安装管理）。前端源码同仓库+预构建 dist 提交仓库（AdGuardHome 模式，node 不进 rick 构建链）+ vite dev server 开发态代理。**社区能力优先复用**（human 指示，待 research-L2 深挖）。
- **J6=P0+P1+P2**（tools 管理不 web 化）。
- **J7=单用户 token 认证**，默认 loopback，`--listen` 显式开放，HTTPS 交反代。

**L1 终判后的架构（含 r2 调研固化细节）**：

```
浏览器多端（SPA，768/1024 断点，session=聊天窗事件流）
  ↕ POST 命令 + SSE 事件流（可恢复 SSE：Last-Event-ID + 服务端重放缓冲）
Go web 服务（`rick web`：API 路由 + embed dist + token 认证 + SSE 事件总线）
  ↕
会话执行层（pi rpc 子进程 supervisor：每 active session 一子进程；
           Setpgid+SIGTERM→5s→SIGKILL；get_state 心跳；respawn+补流恢复）
  ├─ 交互型 session（plan/easy/ctrl/human-loop）= pi --mode rpc 常驻，extension_ui 双向桥
  ├─ doing session = 编排监控视图（handler.Doing + --mode json 一次性，tasks.json watcher）
  └─ closed session 浏览 = 离线读 session JSONL（不 spawn 进程）；resume = spawn+--session
  ↕
rick handler 复用桥（cmd 层入口不同，handler+runtime 逻辑保持一套架构）
  ↕
状态层（.rick/ git 仓库 + session JSONL ~/.rick/pi/agent/sessions/）
```

**关键固化决策（来自 r2 调研）**：
1. pi 驱动层做成可替换接口（上游 PiServer/pi-protocol 实验中，留切换退路）
2. subagent/index.ts 子进程管理范本直接移植 Go（分帧/超时/上限/探活）
3. extension_ui_request/response 双向桥必须实现（human-loop web 化必经通道）
4. 传输=可恢复 SSE（POST 发起+GET EventSource，断线退避重连），不用 WS 双连接
5. doing 维持 --mode json 一次性（无需常驻 rpc）

---

## 第 2 层 — 会话执行层与 Web 服务层（后端全貌）

**L1 自查消解的事实**（无需再问）：
- pi 会话恢复：`pi --session <path|id>` 加载已有会话；session JSONL 存 `~/.rick/pi/agent/sessions/<workdir>/<date>_<uuid>.jsonl`；`get_entries(since=<entryId>)` 游标增量回放（含 leafId 判活）
- extension_ui 双向桥：对话框方法 select/confirm/input/editor 需回 extension_ui_response（value/confirmed/cancelled）；单向推送 notify/setStatus/setWidget/setTitle/set_editor_text 免回
- rick handler 现状：plan/easy/ctrl 均是「builder 写 prompt 文件 → runtime.CallCLI(ModeInteractive, --session-id <uuid>)」；easy 有 saveSessionID/loadSessionID 持久化；doing 是「builder 写 prompt → rt.Run(--mode json 一次性) + tasks.json watcher 实时进度」
- rpc 命令面：prompt/steer/follow_up/abort/new_session/get_state/get_messages/get_entries/switch_session/fork/clone/export_html/bash/set_session_name…

**模块与 pipeline（候选）**：
```
HTTP API 面（GET/POST JSON：jobs 看板/sessions 管理/knowledge/git 同步/auth）
  ↓
SSE 事件总线（单流多路复用：pi 子进程事件×session 路由 + jobs 状态变更 + extension_ui 请求）
  ↓
会话执行层：
  ├─ pi rpc supervisor（每 active session 一子进程：spawn/close/resume/respawn/get_state 心跳）
  ├─ doing 执行桥（handler.Doing 后台跑 + 进度流）
  └─ closed session 浏览（离线读 session JSONL）
  ↓
状态层（.rick/ + session JSONL + git 快照）
```

**判断节点（L3 已裁决，human 2026-08-29 第二轮）**：
- J2.1 doing=**监控型 session**（task 看板+编排事件流+gate 结果，无聊天输入，可中止）✅
- J2.2 dream=**双模**（交互聊天型 或 后台监控型，启动时选模式）；learning=交互聊天型 ✅
- J2.3 SSE=**单流多路复用**（GET /api/events，全局事件总线 + Last-Event-ID 重放）✅
- J2.7 **多工作区模型（human 澄清，v1 即支持）**：`rick web` 全机唯一（任意目录启动）；session 启动时锚定工作目录（必有 .rick）；一会话一工作区，一工作区多会话 ⚠️ **架构影响：handler 现从 cwd 解析 rickDir，需重构为显式 workspace/参数注入**
- J2.4 认证 token 认可（config 字段 + Bearer + SSE query param）✅
- J2.5 git push **不属 web UI**（agent 自主）✅
- 新增判断节点（待问）：工作区注册/发现机制；singleton 服务行为（端口/重复启动检测/守护）；agent push 协议是否需扩展；dream 双模的启动参数设计

**实现影响事实（自查）**：`workspace.GetRickDir()` = cwd+".rick"（不向上回溯），handler 内 24 处调用——多工作区需重构为显式 workspace 路径注入（CLI 默认 cwd，web 按 session 的 workspace 传入）；pi 子进程 spawn 需 cmd.Dir=workspace 根。

**L4 事实回流（research-L2，human 追问 pi 生态可复用性）**：
- **新发现 ygncode/pi-web**（MIT，⭐89，活跃 2026-08）：Go1.25+SSE+每 session 一 `pi --mode rpc` worker+go:embed+Vite/Svelte5+PWA+token——与 rick 裁决架构同卵；**Go 侧摘代码移植首选**（internal/rpc/client.go JSONL 命令构建、workers/manager.go worker+Factory+崩溃恢复、SSE registry/sse_format.go、frontend/assets.go Vite manifest+embed）
- pi-agent-dashboard（MIT）：React19 组件+**事件序号+subscribe(lastSeq) 断线续传协议**（映射 rick SSE Last-Event-ID+重放缓冲）+渲染栈可 vendor（chat-embed 子路径，摘取需 git vendor+锁版本 ~24 依赖）
- pi-web-ui（MIT）：React18 组件族（ChatInput/ToolCallBlock/ThinkingBlock/BashBlock/MessageList）+单 CSS 主题热加载——换 SSE 客户端即可移植
- jmfederico/pi-web（MIT）：Node+SDK 进程内+Lit+WS 异构——仅借设计（sessiond 双进程分离、安装器 systemd/launchd 自适配+doctor、REST 域拆分、798 前端测试工程化）；PR#98 拒绝子进程路线与 rick 相反，后端不可复用
- pi-remote-web-ui：**无 LICENSE 勿抄代码**，仅设计借鉴；@firstpick：PIN+QR+LAN 开放模式留未来手机接入；上游 pi-server 系全部 Experimental+breaking 频繁——**不用**，JSONL RPC 是稳定层，驱动层做接口抽象留退路
- 渲染推荐（React 线）：react-markdown+remark-gfm+rehype-highlight+ansi-to-react+@git-diff-view/react+dompurify；Svelte 线：marked+highlight.js+自封装 ANSI+自选 diff 库（ygncode 无 diff 库需补）

> 状态：**已终判（human 两轮裁决 2026-08-29）**

**L2 终判（全部判断节点固化）**：
- **J2.1 doing=监控型 session**：task 看板+编排事件流+gate 结果，无聊天输入，可中止（kill 子进程，与 CLI Ctrl+C 一致，attempt+1 续跑）✅
- **J2.2 dream=双模**（新建时选：交互聊天型 rpc / 后台监控型，参数 job-num+mode）；learning=交互聊天型 ✅
- **J2.3 SSE=单流多路复用**（GET /api/events，Last-Event-ID 服务端重放缓冲——借 dashboard 事件序号+subscribe(lastSeq) 协议设计）✅
- **J2.7 多工作区**：手动注册为主（web UI「添加工作区」输入路径+校验含 .rick，机器级持久化 ~/.rick/web.json）+最近会话目录建议为辅；**UI 同屏显示多工作区，多 job 跨工作区并行**；`rick web` 全机唯一（pid 文件判活+端口冲突检测，前台运行，守护交用户 systemd/nohup 文档化）✅
- **J2.4 认证**：~/.rick/config.json 新增 web_token（首次自动生成打印）+ Authorization Bearer + SSE ?token= query param ✅
- **J2.5/J2.16 git push=完全 agent 自主**（协议不动、hook 不加 push、web UI 无同步控制面；KR3 云端同步=agent 自主 git 行为）✅
- **Q13 前端=React**（社区组件供给面最广：dashboard chat-embed + pi-web-ui 组件族可摘；渲染栈 react-markdown+remark-gfm+rehype-highlight+ansi-to-react+@git-diff-view/react+dompurify）✅

**复用策略（固化）**：
- **摘代码移植**：ygncode/pi-web Go 侧（internal/rpc/client.go JSONL 命令构建、workers/manager.go per-session worker+Factory+崩溃恢复、SSE registry/sse_format.go、frontend/assets.go Vite manifest+embed）
- **摘组件移植**：pi-web-ui React 组件族（ChatInput/ToolCallBlock/ThinkingBlock/BashBlock/MessageList）+ dashboard 渲染栈组合；dashboard chat-embed 需 git vendor+锁版本（~24 依赖）——优先从 pi-web-ui 摘（更轻）
- **借设计**：jmfederico sessiond 双进程/安装器工程化/API 域拆分；firstpick PIN+QR 留未来手机接入
- **不用**：上游 pi-server 系（Experimental）；VVander（无 LICENSE）

**L2 模块清单（终版）**：
```
cmd 层：rick web（cobra 命令：singleton 检测/--port/--listen/--token）
web 服务层（handler 编排 + 实现包）：
  ├─ HTTP API 路由（REST：workspaces/sessions/jobs/knowledge/dream/learning）
  ├─ SSE 事件总线（单流多路复用 + 重放缓冲 + 心跳）
  ├─ token 认证中间件
  └─ 静态资源服务（embed dist）
会话执行层（runtime 层扩展）：
  ├─ pi rpc supervisor（per-session worker：spawn/close/resume/respawn/get_state 心跳；cmd.Dir=workspace）
  ├─ doing/dream 后台执行桥（handler.Doing/Dream 后台跑 + 进度流 + abort）
  └─ closed session 离线浏览（读 session JSONL，不 spawn）
状态层：workspace 注册表（~/.rick/web.json）+ session 注册表 + .rick 状态 + session JSONL
env 层：前端 dist 安装管理（embed + 构建链）
```

> 状态：**已终判**（L4 回流+两轮 L3 裁决已归档在上文）。

---

## 第 3 层 — 前端架构（React SPA）

**调研基础**：research-L1-r2 leaf-4（OpenHands/LibreChat/goose/claude-code-webui 交互模式）+ research-L2（组件复用源/渲染栈）。

**模块（候选）**：
```
React SPA（Vite 构建，dist 提交仓库，go:embed 服务）
  ├─ 导航壳（多工作区同屏 + 会话列表 + 功能区路由）
  ├─ session 聊天窗（事件流渲染分层：delta 直渲/工具折叠卡/thinking 折叠/错误二分）
  ├─ 监控视图（task 看板 + 编排事件流 + gate 结果）——doing/dream 后台模式
  ├─ cmd 参数弹窗（各 session 类型的启动参数表单）
  ├─ jobs 看板（P0：jobs 列表/task 状态/act-path/debug 查看）
  ├─ knowledge 浏览（domain/loops/skills 文件树 + markdown 渲染）
  ├─ SSE 客户端（EventSource + 重连 + 按 session 分发）
  └─ 响应式（768/1024 断点，移动端 drawer+粘性输入框）
```

> 状态：**已终判（human 裁决 2026-08-29）**

**L3 终判**：
- **UI 信息架构=A**：左侧栏（多工作区同屏 + 各工作区会话列表）+ 顶部功能区切换（Sessions/Jobs/Knowledge）+ 主内容区；移动端 <768px 侧栏折叠 drawer；断点 768/1024（Tailwind 与 JS 断点对齐）
- **主题=Rick and Morty 动漫灵感深色默认**（human 设计指令）：飞洒、星空陨石、绿色传送门等代表性物件；**星空背景要有动态性**（动效实现：CSS/canvas 星空 + 流星 + 传送门旋涡动效；遵循 prefers-reduced-motion；动效不得损害可读性/性能）
- **PWA=做**（vite-plugin-pwa；注意：service worker 需 localhost 或 HTTPS——LAN IP http 下降级普通网页，文档说明）
- **技术栈**：Zustand（流式事件批处理）+ react-router + Vite + Tailwind CSS 4
- **并发 doing=不做互锁**（用户自行保证同工作区不并行 doing；跨工作区并行无限制）
- **分发=C 混合 + 自迭代**（human 指令）：embed baseline + 可写覆盖层（~/.rick/web/）+ **用户通过对话让 agent 修改前端、自迭代、自动生效** + baseline 可复位覆盖自改进

**自迭代机制（设计草案，待 human 确认）**：
```
embed（go:embed web/src + web/dist）—— baseline（版本与 rick 二进制一致）
~/.rick/web/——可写自定义层：src/（源码）+ dist/（构建产物）
服务策略：~/.rick/web/dist/index.html 存在 → 服务覆盖层；否则服务 embed baseline
自迭代流：任意 agent 会话 bash 编辑 ~/.rick/web/src/ → npm install && npm run build（node 已是环境依赖）
          → dist 更新 → fs watcher → SSE 推送 reload 事件 → 浏览器自动刷新生效
复位：「Reset to baseline」按钮/`rick web reset` → 删覆盖层 → 回 baseline
入口：UI「Customize UI」按钮（首次抽取 baseline 源码到 ~/.rick/web/src + 开一个预填定制 prompt 的会话）
```

---

## 第 4 层 — 构建链与运维

**自我消解项（惯例/低争议，记录备查）**：
- 构建链：`make web-dist`（cd web && npm ci && npm run build）→ `web/dist/` 提交仓库 → go:embed；开发态 vite dev server 代理 API 到 rick web
- token：`~/.rick/config.json` 新增 `web_token`（首次 `rick web` 启动自动生成并打印）
- 守护：不内置（前台运行；systemd/nohup/tmux 文档化——借 jmfederico 安装器思路，v1 只写文档）
- CI dist 新鲜度校验：v1 不做（可选项记入文档）
- PWA 安全上下文：localhost/HTTPS 可装；LAN http 降级普通网页（文档说明）

**待确认判断节点（已裁决，见下终判）**：
- J4.1 自迭代机制设计（embed src+dist / ~/.rick/web 覆盖层 / SSE reload / reset / customize 入口）
- J4.2 端口默认值（拟 6137——C-137 彩蛋）+ 工作区注册表 ~/.rick/web.json

**终判（human 确认 Q25/Q26）**：
- **J4.1 自迭代机制确认**：embed（src+dist 双内嵌）→ ~/.rick/web/ 覆盖层（优先服务）→ SSE reload 自动生效 → reset 复位回 baseline ✅
- **J4.2 端口默认 6137**（C-137 彩蛋，--port 可改）；工作区注册表 ~/.rick/web.json ✅
- 其余自我消解项（构建链 make web-dist / token web_token / 守护文档化 / CI 校验 v1 不做 / PWA 安全上下文说明）见上文记录

**消解声明：本层全部消解，无剩余不可消解事实**——构建链/分发/嵌入先例已在 research-L1-r2（结论 11/12：AdGuardHome 等先例、SSE 选型）与 research-L2（ygncode Vite manifest+go:embed 方案）覆盖；端口/配置格式为惯例决策且已 human 裁决；无重量级未消解调研项。

---

## 叶子层 — 模块实现落实（四维度）

> 消解声明：叶子层为设计综合层——全部事实问题已在前四层调研与裁决中消解（research-L1/L1-r2/L2/L3 在案），本层无新增调研需求；产出为文件/签名/命令/配置四维度落实。

### M1 cmd 层：rick web 命令

**文件**：`internal/cmd/web.go`（新建）
**函数签名**：
```go
func NewWebCmd() *cobra.Command          // rick web：无参=运行服务；子命令 customize/reset
func runWeb(opts WebOptions) error       // 组合根：singleton 检测 → handler.Web(opts)
type WebOptions struct { Port int; Listen string; Token string; Verbose bool }
```
**命令**：`rick web [--port 6137] [--listen 127.0.0.1] [--token xxx]`、`rick web customize`、`rick web reset`
**配置**：flags + ~/.rick/config.json（web_token）

### M2 web 服务层（WEB-UI 入口实现包）

**文件**（新建 `internal/web/` 包）：
```go
server.go    — func Start(cfg ServerConfig) error          // http.Server 生命周期、优雅关闭
routes.go    — func registerRoutes(mux, deps)               // 路由表挂载
auth.go      — func tokenMiddleware(token) func(next)       // Bearer 校验；SSE 走 ?token=
sse.go       — type Hub struct; func (h *Hub) Subscribe(lastEventID) / Publish(evt) / heartbeat()
               // 单流多路复用 GET /api/events；per-connection 重放环形缓冲（Last-Event-ID）；
               // 事件序号全局单调（借 dashboard 协议设计）
sessions.go  — type SessionRegistry struct                   // 内存 active + ~/.rick/web/sessions.json 持久化
               // REST: POST /api/sessions（创建：类型+工作区+参数 → spawn worker 或后台任务桥）
               //       GET /api/sessions、GET /{id}、POST /{id}/prompt|steer|abort|close|resume|ui_response
               //       GET /{id}/entries?since=（closed 离线回放）
jobs.go      — GET /api/workspaces/{ws}/jobs|/jobs/{job}/tasks、knowledge 文件树+内容接口
               // 复用 workspace/builder 的路径参数化读取；不含写操作（写都经会话/执行桥）
static.go    — 静态资源：~/.rick/web/dist/index.html 存在→服务覆盖层，否则 embed；SPA fallback；
               // index.html no-cache + hashed assets 长缓存（Vite 产物天然 hash）
watcher.go   — fsnotify：~/.rick/web/dist → SSE frontend_reload；各工作区 tasks.json → jobs_update
registry.go  — type WorkspaceRegistry（~/.rick/web.json：workspaces[]）；会话持久化（sessions.json）
```
**事件协议（SSE envelope）**：`{seq, session_id, type, data}`；type ∈ session_event（pi rpc 事件透传：message_update/tool_execution_*/agent_*/extension_ui_request）、session_state、jobs_update、frontend_reload、server_info
**认证**：静态文件不设防；API/SSE 走 token

### M3 会话执行层（runtime 层扩展）

**文件**（`internal/runtime/` 新增）：
```go
rpc.go       — type RpcClient struct{ stdin io.Writer; ... }
               func (c *RpcClient) Prompt(msg string) / Steer(msg) / Abort() / GetState() /
                    GetEntries(since string) / SendUIResponse(id, resp) / SetSessionName(name)
               // JSONL 命令构建（严格 LF 分帧；借 ygncode rpc/client.go 模式）+ 事件行解析
supervisor.go— type Supervisor struct{ workers map[string]*Worker }
               func NewSupervisor(...) / (s) Spawn(spec SpawnSpec) (*Worker, error)
               // SpawnSpec{SessionID, WorkspaceDir, MethodFile, PromptFile, ResumeSessionID}
               // Worker：exec.Cmd + cmd.Dir=WorkspaceDir + AgentEnv() + Setpgid 进程组 +
               //   三管道常驻 reader goroutine + SIGTERM→5s→SIGKILL + get_state 心跳(30s) +
               //   stdout EOF→respawn+get_entries(since) 补流 + idle 超时(30min)回收
               // active 进程上限参照 pi subagent 量级（8）；doing 一次性任务不走 supervisor
runtime.go   — （重构）Runtime.Run 增加 dir 参数变体（doing 后台桥用，cmd.Dir=工作区）
```

### M4 handler 层改造（workspace 显式注入）

**文件**（`internal/handler/` 修改）：
```go
web.go       — func Web(opts WebOptions) error     // 编排：singleton 检测（~/.rick/web.pid 判活+端口冲突）
                                               // → 组装 Supervisor/Hub/Registry → web.Start()
plan.go 等    — 抽取路径参数化核心：planCore(rickDir, ...)、easyCore(...)、ctrlCore(...)、
               humanLoopCore(...)、learningCore(...)、dreamCore(...)、doingCore(...)
               // 现有 Plan()/Easy()... 保持签名（内部 cwd 解析后调 core）——CLI 零行为变化
               // builder 已路径参数化（SavePlanPrompt(requirement, jobPlanDir, rickDir) 等），无改动
doing 也会话化：doingCore 暴露进度回调（tasks.json watcher diff → SSE jobs_update）+ 可取消 context
```

### M5 env 层：前端基座部署

**文件**（`internal/env/` 新增 `web.go`）：
```go
func DeployWebScaffold() error   // 幂等抽取 embed 的 web/src 基线到 ~/.rick/web/src/
                                  // （rick-managed 标记；用户已存在自定义则跳过）——deployRickAgents 同款模式
func CheckWebEmbed() error       // 校验 embed dist/src 存在（tools 检查用）
```

### M6 前端（web/ 目录，React SPA）

**目录结构**（新建 `web/`，仓库根）：
```
web/
  package.json  vite.config.ts  tsconfig.json  index.html  embed.go   // package web：//go:embed all:dist + src
  public/  (manifest.webmanifest、PWA 图标——传送门绿色飞碟)
  src/
    main.tsx  App.tsx
    api/client.ts   — REST 客户端（token 注入、错误二分）
    api/sse.ts      — EventSource 管理（重连指数退避、按 session 分发、frontend_reload 自动刷新）
    stores/*.ts     — Zustand：workspaces/sessions/events（批处理缓冲）/jobs/ui
    components/
      layout/       — Sidebar（多工作区同屏+会话列表+状态点）/ TopNav（Sessions|Jobs|Knowledge）/ MainLayout
      starfield/    — StarfieldBackground（canvas 动态星空：闪烁+流星）/ Portal（绿色传送门旋涡——连接中指示）/ Saucer（飞碟——job 运行指示）
      chat/         — ChatView / MessageList / MessageBubble / ToolCallCard（折叠卡）/ ThinkingBlock /
                      StreamText / ChatInput（斜杠命令/steer）/ SteerBar（中止/纠偏按钮）
      monitor/      — MonitorView（doing/dream 后台监控）/ TaskBoard（task 状态流转看板）/ GateResult / EventStream（编排事件流）
      sessions/     — SessionList / NewSessionModal（cmd 类型选择+参数表单：plan=requirement、
                      easy=requirement+ctx、doing/dream=job+模式、ctrl/learning=job、human-loop=topic）
      jobs/         — JobsList / JobDetail / TaskStatus
      knowledge/    — KnowledgeBrowser / FileTree / MarkdownView
      extui/        — ExtensionUIDialog（select/confirm/input/editor → web 表单回填 ui_response）
      common/       — Dialog / Button / Spinner / ErrorBanner
    routes/         — Sessions.tsx / Jobs.tsx / Knowledge.tsx / Settings.tsx（含 Customize UI/Reset 按钮）
    styles/theme.css — R&M 配色 token：传送门绿 #39ff88、深空底 #0b0e14/#1a1c2e、星云紫 #5d3fd3、
                       Rick 蓝灰 #a6c8dd、Morty 黄 #ffd54a；星场 keyframes；prefers-reduced-motion 降级
```
**渲染栈**：react-markdown + remark-gfm + rehype-highlight + ansi-to-react + @git-diff-view/react + dompurify
**技术栈**：React 18+ / Zustand / react-router / Vite / Tailwind CSS 4 / vite-plugin-pwa
**复用来源**：ygncode Go 侧（M3）；pi-web-ui React 组件族参考（ChatInput/ToolCallBlock/ThinkingBlock 结构）

### M7 会话类型 → 启动映射（终版表）

| 类型 | 形态 | spawn 方式 | 参数 | 产物 |
|---|---|---|---|---|
| plan | 交互 rpc | rpc spawn + plan prompt 系统提示词 + bootstrap | requirement（或 --job 复用） | jobs/{id}/plan/ |
| easy | 交互 rpc | 同上 | requirement、ctx-path? | jobs/{id}/doing/ |
| ctrl | 交互 rpc | 同上 | job | tasks.json/plan 修改 |
| human-loop | 交互 rpc | 同上 | topic | draft/rfc/ |
| learning | 交互 rpc | 同上 | job | learning/ |
| dream | 双模 | 交互=rpc；后台=goroutine+json | job-num、mode | dream 日志 |
| doing | 监控 | goroutine + rt.Run(--mode json, dir=ws) | job | tasks.json+commits |
| closed | 离线 | 不 spawn；读 session JSONL + get_entries | — | 只读浏览；resume=spawn --session |

### M8 工具调用与环境配置

**命令**：
- `make web-dist`（cd web && npm ci && npm run build → dist/）——改前端后执行并随源码提交
- `make build` / `go build`（embed dist；无 dist 时 embed 空目录占位不阻断构建）
- `rick web`（服务）/ `rick web customize`（DeployWebScaffold + 提示定制入口）/ `rick web reset`
- 开发态：`rick web` 起服务 + `cd web && npm run dev`（vite 代理 API→6137）

**环境依赖**：Go 1.21+（现状）；node ≥22.19 + npm（pi 已要求，已有；customize 构建复用）；浏览器（SSE/EventSource）
**配置文件**：
- `~/.rick/config.json` += `"web_token": "<auto-gen>"`
- `~/.rick/web.json`：`{"version":1, "workspaces":[{"path":"/abs","name":"...","added_at":"..."}]}`
- `~/.rick/web/sessions.json`：会话注册表（id/workspace/type/params/pi_session_id/status/created_at）
- `~/.rick/web.pid`：singleton 判活
**环境变量**：PI_CODING_AGENT_DIR（现有，不变）；无新增

### OKR 充分性终检（叶子层）

M1-M8 联合 ⟹ KR1（架构与接入：cmd/web/handler/runtime/env 五层落位）∧ KR2（功能完备：P0 看板/knowledge + P1 全部交互会话 + P2 触发类）∧ KR3（云端同步：中心实例单写点 + agent 自主 git 行为）∧ KR4（多端适配：响应式+PWA+R&M 主题）∧ KR5（质量验收：gates 见流水线设计）⟹ O 达成 ✅

---

# 增量设计树（第 2 棵）：rick 自进化 —— rick web 内改进 rick web

> 背景：本增量是 job_36 的**收尾功能**（用户原话：「这个最后的 rick 自进化改进就是 rick web 升级的最后功能了」）。
> 与第 1 棵树（L0-L3，已终判并交付）的关系：本树是**新增的能力维度**，不是第 1 棵树的子节点——它引入了「运行时/环境/发布」这一全新关注面，
> 因此单独成树（顶层仍是一组 OKR，遵守同一套 MECE + 充分性纪律）。层号从 L4 续编，research 简报编号随之。

## 根层（L0'）— OKR

**O（增量目标）**：在 rick web 里就能完成对 rick 自身的改进闭环——**开发期**在完全隔离的实例中改前端/后端/CLI/pi runtime（生产内核与其上所有运行中 job 不受影响），**交付期**把改动经人类审核后原子替换到生产并可回滚，**切换期**生产重启后所有原本在跑的会话被自动恢复、上下文不丢（含明确的「被中断回合」续跑语义）。

**KR 集（充分性推导：KR1 ∧ KR2 ∧ KR3 ∧ KR4 ⟹ O）**：

| KR | 内容 | 验证方式（初步） |
|----|------|------------------|
| KR1 **隔离实例** | 存在一个与生产完全隔离的开发实例：独立端口 + 独立 web 状态目录 + 独立源码工作树 + 独立二进制 + 独立前端产物；两实例可同时运行且互不干扰（含 workspace/.rick 写冲突面） | 双实例同时跑：生产 8413 的 job/会话不中断；dev 端口可独立启停/重建 |
| KR2 **开发闭环** | 在 dev 实例内改动的生效回路：前端改动即时生效（不重启）；后端/CLI/runtime 改动受控重建并重启 dev 实例后生效；**且开发者（AI 会话）自身不因 dev 重启而失联**（自举问题） | 在一个 rick web 会话里改 rick web：前端改动刷新可见；后端改动重启后会话仍能继续对话 |
| KR3 **受控提升** | 人类审核确认后，把 dev 产物（二进制 + dist + 源码提交）原子提升到生产，保留上一版并可一键回滚 | 提升流程可重放；回滚命令实测把生产恢复到上一版本 |
| KR4 **切换无损恢复** | 生产重启后自动恢复所有「重启前在跑」的会话与 job：worker 重新拉起、前端事件流不断档（cursor 续传）、被中断的回合有可审阅的续跑/待续跑语义、恢复结果有报告 | 重启前后对比：active 会话数不丢；恢复报告逐条列出成功/失败；浏览器不丢事件 |

**OKR 充分性自检**：KR1 给出「改坏也不影响生产」的空间 → KR2 让「在 web 里改 web」成为可重复的日常动作（而不是一次性演示）→ KR3 给出「改动如何变成生产」的受控通道 → KR4 支付 KR3 的价格（重启）并保证用户可感知的连续性。四者联合 ⟹ O 达成 ✅
（划分维度：**空间 / 回路 / 通道 / 代价**，MECE：互不重叠且覆盖「隔离—迭代—交付—连续性」全部关注面。缺 KR4 则「替换必然重启」会让 O 的「任务不停止」落空；缺 KR1 则开发即破坏生产，O 无从谈起。）

## 层映射（待逐层展开，遵守 L1-L5 loop）

| 层 | KR | 该层 pipeline（假设，待 L4/L5/L6 调研校正） |
|----|----|------------------------------------------|
| **L4 隔离层** | KR1 | dev 实例启动配置（HOME/state dir/workspace 注册表/端口） → 源码工作树隔离（git worktree） → 二进制与 dist 隔离 → pi agent dir 共享策略 → 双实例共存约束 |
| **L5 开发闭环层** | KR2 | 改前端 → overlay dist → frontend_reload → 刷新可见 ／ 改后端 → 构建 → dev 进程重启 → **开发会话自举不中断** → AI 侧可观测反馈（健康探测/构建日志） |
| **L6 提升与恢复层** | KR3+KR4 | 人类审核确认 → 提升前门禁（测试/构建/健康） → 原子替换（版本目录 + current 链） → 生产重启 → 自动恢复 active 会话（intent 持久化 + resume） → 中断回合续跑语义 → 恢复报告 → 失败回滚 |

> 状态：**L1 调研进行中**（research-L4/L5/L6 并行派发）。L4 达标后才下钻 L5、L6（不得跳层）。

---

## 第 4 层 — 隔离层（KR1：dev 实例与生产完全隔离）

**调研**：research-L4.md（391 行，§1 路径清单 / §2 pi 沙盒 / §3 工作树 / §4 单例与端口 / §5 写冲突 / §6 前端 overlay / §7 最小隔离集 + 7 条残留泄漏）。
**同一轮并行产出的下游层简报**（已落盘，作为 L4 裁决的代价参考）：research-L5.md（开发闭环+自举）、research-L6.md（提升与恢复）。

### 4.1 关键事实（本轮调研固化，均有代码行号或实测支撑）

| # | 事实 | 证据 | 对设计的影响 |
|---|------|------|--------------|
| F1 | 机器级状态 100% 由 `os.UserHomeDir()` 派生，**唯一隔离开关是 HOME**；`--state-dir` 之类 flag 不存在 | `internal/web/web.go:25-68`、`internal/cmd/web.go:81-124` | 隔离必须走 HOME；想更优雅需改代码（新增开关） |
| F2 | pi 沙盒是**唯一 env 可重定向项**：`RICK_PI_AGENT_DIR` > `$HOME/.rick/pi/agent` | `internal/runtime/agentdir.go:22-30` | pi agent dir 是与 HOME 解耦的独立裁决点 |
| F3 | `rick_dev` 命名法残缺：只切 `config.json` 与 CLI 的 `.rick_dev`，**web 状态/pid 仍落 `.rick`** | 实测 | 不可用「改名法」当隔离；且会与生产抢 `web.pid` |
| F4 | **生产 singleton 当前已失效**：进程在跑但 `~/.rick/web.pid` 不存在；同 HOME 起第二实例被放行，且其 `ReconcileOnStart` 会把生产 active 会话改写成 error 并落盘 | `internal/handler/web.go:112-121`、`internal/web/sessions.go:154-176`；实测 | **现存生产缺陷**，也是 dev 隔离的必须前置（否则误伤生产） |
| F5 | 全仓**无跨进程锁**（grep flock 空）；workspace 级 `tasks.json` 非原子写、job 目录 TOCTOU、`git add -A` 混提交 | research-L4 §5 | 承诺「生产 job 不被打断」必须靠「dev 不共享 workspace」而非靠锁 |
| F6 | **共享源码工作树是最贵泄漏**：生产 `web.json` 已注册本仓库根（`a8ceb938`），worker cwd = `ws.Path`，job 目录硬编码 `<ws.Path>/.rick` | `~/.rick/web.json`、`sessions.go:434,987,1053`、`watcher.go:118` | dev 实例**绝不能注册生产仓库根**，否则 dev 会话直接改生产源码+jobs |
| F7 | 生产二进制 = **仓库工作树里的 `bin/rick`**（不是 `~/.rick/bin/rick`，后者是 8/24 遗留）；`start-web.sh` 用 `./bin/rick` | `/proc/172761/exe`、`md5sum`；research-L6 §1 | 「生产」= 仓库工作树 + bin/rick → 隔离必须换工作树 |
| F8 | 前端 overlay（`$HOME/.rick/web/dist`）随 HOME 自然隔离；且 overlay 可以是**目录软链**（实测可服务软链内文件）；vite dev proxy 硬编码 6137 | `static.go:52-99`、`server.go:66-68`、`web/vite.config.ts:57-60`；实测 | dev 前端零拷贝（软链到 dev 树 dist）；proxy 需参数化 |
| F9 | 端口：默认 6137；生产 8413；**8414 空闲**（8410-8413/8415/8420 被占） | `ss -ltnp` 实测 | dev 端口取 8414 |
| F10 | 覆盖 HOME 会连带换掉 `GOPATH/GOMODCACHE/GOCACHE/GOENV` → `go build` 从 0.98s 退化为冷构建（需联网下载 toolchain） | research-L5 §5 实测 | dev 启动环境必须显式固定 Go 缓存 |
| F11 | `web/dist` **已被提交入库**（`git ls-files` 命中），`.rick/` 也被提交 → worktree 自带一份 `.rick` 快照（与生产分叉） | research-L4 §3 | dev 树的 `.rick/jobs` 是**陈旧副本**，不能当作生产状态用 |

### 4.2 模块与 pipeline（L4 假设，待裁决后固化）

```
M1 启动封装（dev-web.sh：env 白名单 + 硬校验 + 端口/token）
   → M2 状态隔离（HOME=$DEV_HOME：web.json/sessions.json/archived/job-names/web.pid/overlay/config.json）
   → M3 源码工作树隔离（git worktree → $DEV_TREE，独立 ref/分支）
   → M4 产物隔离（$DEV_TREE/bin/rick 唯一命名 + $DEV_HOME/.rick/web/dist 软链 → $DEV_TREE/web/dist）
   → M5 pi 沙盒策略（RICK_PI_AGENT_DIR = 独立 or 共享生产）
   → M6 workspace 注册纪律（dev 注册表只放 $DEV_TREE；生产仓库根绝不由 dev 注册）
   → M7 双实例共存约束（端口 8414、singleton 修复、fail-fast 校验、共享缓存清单）

接口契约（模块间）：
  M1 → {HOME, RICK_PI_AGENT_DIR, GOCACHE, GOMODCACHE, PORT, TOKEN, DEV_TREE}   （纯环境契约）
  M2 ← HOME（无代码改动；只依赖既有 HOME 派生）
  M3 ← git worktree（无代码改动）
  M4 ← M3 的 web/dist 与 bin/（软链 + 唯一文件名）
  M5 ← 显式 env（无代码改动）
  M6 ← 人工/AI 通过现有 Add API 注册（`Add` 要求目录含 `.rick/`，worktree 自带）
  M7 → 需**代码改动**：singleton 加固（flock 或 pid 校验）+ 可选 `--state-dir`
```

**MECE 检查**：M1 环境 → M2 状态 → M3 源码 → M4 产物 → M5 运行时沙盒 → M6 数据面纪律 → M7 共存约束。互斥（进程/文件系统/运行时/数据/共存五个不同资源面），完备（research-L4 的 Q1-Q7 全部落在某个模块内）。

**OKR 充分性自检**：M1-M7 全部达成 ⇒ 两实例可同时运行、dev 的任何写操作都不触达生产状态/源码/jobs ⇒ KR1「隔离实例」成立 ✅（前提：M7 的 singleton 缺陷必须修——否则 M2 的隔离边界可被同 HOME 实例绕过）

### 4.3 判断节点（L3 批量追问 human）

| # | 判断节点 | 选项 | 推荐 |
|---|----------|------|------|
| **J-L4-1** | 隔离主开关 | (a) 仅用独立 HOME（零代码改动）(b) 新增 `--state-dir`/env 开关（改代码，更显式）(c) 两者都做（先 a，后补 b） | **(c)**：先以 HOME 落地（今天就能跑），同时新增 `--state-dir` 使「隔离」成为显式契约而非隐含约定（预防未来有人同 HOME 起 dev） |
| **J-L4-2** | dev 的 pi 沙盒 | (a) 独立 `RICK_PI_AGENT_DIR`（+145MB，需拷 auth/settings 种子）(b) 共享生产 agent dir（省空间，但 dev 跑 `tools init-pi/update-pi` 会原地覆写生产 145MB runtime） | **(a) 独立**：145MB 换「生产 pi runtime 不可被 dev 覆写」 |
| **J-L4-3** | dev 源码树 | (a) `git worktree add` 到 `/workdir/sunquan20/rick-dev`（2.13s / 202MB，同设备）(b) `git clone --local` (c) 目录拷贝 | **(a) worktree**：秒级、共享 object store、天然带 `.rick` 快照；代价是 ref 共享需纪律（dev 分支不要乱推） |
| **J-L4-4** | dev 监听面 | (a) `127.0.0.1:8414`（最安全，但**你的浏览器在局域网另一台机器上时访问不到**）(b) `0.0.0.0:8414` + dev 专用 token（与生产同一暴露模型，你能直接打开 dev UI） | **(b)**：你的实际工作流是从本机浏览器访问 10.128.x.x:8413 —— dev 若只听 loopback 就没法看；用独立 token 控制风险 |
| **J-L4-5** | 是否本次修生产 singleton 缺陷（F4） | (a) 修（flock + 校验 + 拒绝第二个同状态目录实例）(b) 不修（dev 靠纪律绕开） | **(a) 修**：它让「隔离」变成可被绕过，且当前生产确实处于「任何人同 HOME 起实例就会误杀 active 会话」的危险态 |
| **J-L4-6** | dev 数据面纪律 | (a) 硬约束（启动时校验：HOME≠生产 HOME、注册表不含生产仓库根，否则拒绝启动）(b) 仅文档约定 | **(a) 硬约束（fail-fast）**：一个失误就污染生产 jobs/源码，代价不对称 |
| **J-L4-7** | 交付形态 | (a) 只交付脚本 `~/.rick/dev-web.sh`（不进仓库）(b) 进仓库成 `rick dev-web` 子命令 + 脚本包装 | **(b)**：自进化能力应是 rick 的正式能力（可被 AI 会话发现/复用），脚本只做 env 封装 |

> 状态：**L4 待 human 裁决（本轮已提问）**。L5/L6 简报已就绪但**不下钻**（遵守不得跳层）；L4 裁决后按同样 L1→L5 流程展开 L5、L6。

### 4.4 L4 终判（human 裁决 2026-09-20，全部采纳推荐）

| 判断节点 | 裁决 | 落地含义 |
|---|---|---|
| J-L4-1 隔离主开关 | **c（HOME + `--state-dir`）** | 先用独立 HOME 落地（零改码即可跑），同时新增 `--state-dir` 让隔离成为**显式契约**（web 状态目录不再隐含依赖 HOME） |
| J-L4-2 pi 沙盒 | **a（独立 `RICK_PI_AGENT_DIR`）** | `$DEV_HOME/.rick/pi/agent`，首次从生产拷 `auth.json/settings.json/models-store.json` 做种子（chmod 600），再 `tools init-pi`；生产 145MB runtime 不可被 dev 覆写 |
| J-L4-3 dev 源码树 | **a（git worktree）** | `/workdir/sunquan20/rick-dev`（实测 2.13s / 202MB，同设备） |
| J-L4-4 dev 监听面 | **b（`0.0.0.0:8414` + dev 专用 token）** | 保证从局域网浏览器能打开 dev UI（用户的真实工作流） |
| J-L4-5 singleton 缺陷 | **a（修）** | flock + pid 校验 + 拒绝第二个同状态目录实例；这是隔离边界可被绕过的根因，也是现存生产缺陷 |
| J-L4-6 数据面纪律 | **a（启动硬校验 fail-fast）** | 校验 HOME≠生产、注册表不含生产仓库根，否则拒绝启动 |
| J-L4-7 交付形态 | **b（进仓库成 `rick` 子命令）** | 自进化能力是 rick 的正式能力（AI 会话可发现/复用），运维脚本只做 env 封装 |

**L4 终止判定**：M1-M7 pipeline 已澄清（见 4.2），模块职责/接口契约/输入输出明确；OKR 充分性自检通过（含 M7 singleton 修复这一前置）。→ **L4 达标，下钻 L5。**

---

## 第 5 层 — 开发闭环层（KR2：改动如何生效 + 自举不断会）

**调研**：research-L5.md（217 行，§1 前端生效 / §2 五方案对比 / §3 自举四方案 / §4 反馈回路 / §5 构建依赖 / §6 dev 自动恢复）。

### 5.1 关键事实

| # | 事实 | 证据 | 影响 |
|---|------|------|------|
| F1 | **worker 无法被接管**：0/1 是匿名 pipe、`Setpgid` 自成进程组；pi 无 attach/socket 模式；父进程被 KILL 后 worker 下次写 stdout 即 EPIPE 自杀 | `supervisor.go:218-230`；`/proc/<pid>/fd` 实测；pi --help 实测 | 「dev 重启而开发会话不死」**只能靠架构选择**，不能靠重连/接管 |
| F2 | 优雅退出代价：无 SSE 客户端 8.7ms/exit 0；**有 SSE 客户端 9.91s/exit 1**（10s Shutdown 超时）；`kill -9` 后重启仅 42ms 就绪 | 实测 | dev 重启用 kill 快速路径；prod 交付重启要预留 ≤11s |
| F3 | 前端热更现成可用：静态层逐请求 Lstat overlay index.html；watcher 推 `frontend_reload`（**实测延迟 6.8s**，非注释声称的 0.5-2.5s；启动时还有一次伪 reload） | `static.go:71-99`、`watcher.go:78-105`；实测 | dev 走 overlay 软链即可；追求秒级可后续修 watcher debounce（独立小改） |
| F4 | `spec.Dir = ws.Path`，worker cwd = 其工作区路径 | `sessions.go:437-444,984-993`；实测 worker cwd | **方案 (i) 零改码可实现**：prod 托管 + 工作区指向 dev 树 |
| F5 | dev 托管会话受 idle reap（30 分钟无非 response 事件即 Close）且 `IdleTimeout` 未在组合根暴露 | `supervisor.go:808-835,700-708`；`cmd/web.go:105-110` | 长时间离开后开发会话会掉线，需 Resume（或补 env/flag 开关） |
| F6 | `RICK_PI_AGENT_DIR` 不被 `doing.go:235-238` 尊重（gates helper.py 路径硬编码 `UserHomeDir()`） | 代码 | dev 树下 doing 门禁仍读生产 helper（本轮可接受，记录为已知偏差） |
| F7 | 反馈判据缺构建指纹：`/api/health` 仅 `{status:ok}`；`rick_version` 是静态常量 | `routes.go:105-107`、`cmd/rick/main.go:10` | 「新构建真的在跑」需新增 build_id（ldflags + health/config/SSE 携带） |

### 5.2 模块与 pipeline（终判方案 (i)）

```
M1 dev 控制入口（`rick tools dev-web {up|build|restart|status|down}`；~/.rick/dev-web.sh 只做 env 封装）
   → M2 前端生效（$DEV_HOME/.rick/web/dist 软链 → $DEV_TREE/web/dist；可选 vite HMR 需参数化 proxy）
   → M3 后端生效（go build → 唯一文件名（构建指纹）→ 停旧（TERM→12s→KILL）→ 起新 → 健康 + build_id 校验）
   → M4 自举解耦（开发会话托管在 **prod**，其 workspace = $DEV_TREE；dev 实例不托管"必须存活"的会话）
   → M5 可观测反馈（build_id 贯穿 health/config/SSE；构建日志尾部；$DEV_HOME/build.json）

接口契约：
  M1 → env 契约 {DEV_HOME, DEV_TREE, RICK_PI_AGENT_DIR, GOCACHE, GOMODCACHE, PORT=8414, TOKEN}
  M1 → 动作契约 {up, build, restart, status, down}，exit code 语义 {0 ok, 2 build fail, 3 start fail, 4 health fail}
  M3 → 指纹契约：二进制名 `rick.dev.<sha7>-<ts>` + `/api/health` 返回 build_id
  M4 → 注册契约：prod 的 web.json 注册 $DEV_TREE（name=rick-dev），**绝不注册生产仓库根到 dev**
  M5 → 状态契约：$DEV_HOME/build.json {path, sha7, built_at, pid, health_ms}
```

**OKR 充分性自检**：M1-M5 达成 ⇒ 前端改动即时可见、后端改动受控重建重启、开发会话在 dev 任意重启下存活（因为它由 prod 托管）、AI 侧能一条命令判定成败 ⇒ KR2 成立 ✅

### 5.3 判断节点

**已终判（human 2026-09-20）**
- **J-L5-1 自举架构 = (i)**：开发会话托管在 prod，workspace 指向 dev 工作树。补充硬约束（用户原话）：**「交付阶段之前，都不要影响 prod 的任何会话」** → dev 实例不得触发 prod 的 `ReconcileOnStart`/状态改写（由 L4-J5 的 singleton 修复 + HOME 隔离共同保证）。
- **J-L5-2 交付入口 = `rick tools release`**（用户新增要求）：把 dev 切换为 prod 并重启 prod，使刷新 rick web 页面后依旧可连通。即 `release` 是**唯一的提升动作**（人类执行 = 人类确认）。

**待 human 裁决（L5 补充追问）**

| # | 判断节点 | 选项 | 推荐 |
|---|----------|------|------|
| J-L5-3 | dev 实例自身是否允许托管会话 | (a) 允许（复用 L6 auto-resume 恢复）(b) 禁止（纯开发/测试实例） | (a)：反正要建 auto-resume；dev 上跑会话正是验证恢复机制的最好场景 |
| J-L5-4 | dev 工作区在 prod 注册表的形态 | (a) name=`rick-dev` 单独注册 dev 树（prod UI 会同时看到 dev 树的 job/会话）(b) 不注册（开发会话改由 CLI 起，不经 prod UI） | (a)：用户要在 web UI 里跑这个会话，必须注册；用独立 name 便于识别 |
| J-L5-5 | 构建指纹可观测性 | (a) 加 `-ldflags` build_id + `/api/health` 暴露（小改）(b) 不加（靠唯一二进制名 + /proc/exe 比对） | (a)：一条 `curl /api/health` 即可判定"新构建在跑"，对 AI 自举闭环价值最高 |
| J-L5-6 | dev 前端模式 | (a) 只用 `npm run build` + overlay 软链（零额外进程）(b) 同时参数化 vite proxy 支持 HMR | (b)：参数化只 3 行；调 UI 细节时 HMR 明显更快（默认路径仍是 a） |

---

## 第 6 层 — 提升与恢复层（KR3 受控提升 + KR4 切换无损恢复）

**调研**：research-L6.md（432 行，§1 原子替换 / §2 重启与中断边界 / §3 恢复语义 / §4 门禁 / §5 确认留痕 / §6 生态先例 / §7 风险清单）。

### 6.1 关键事实

| # | 事实 | 证据 | 影响 |
|---|------|------|------|
| F1 | 生产二进制 = **仓库工作树 `bin/rick`**（`~/.rick/bin/rick` 是 8/24 遗留未使用） | `/proc/<pid>/exe`、md5 实测 | 提升目标 = `bin/rick`（或改为 `bin/releases/current/rick` 链） |
| F2 | 同 FS `mv` 覆盖运行中二进制**原子且安全**（`cp` 报 ETXTBSY；旧进程继续跑，`/proc/pid/exe` 显示 (deleted)） | 实测 | 提升/回滚可无损切换文件 |
| F3 | `ReconcileOnStart` 只把 active/running → error，**不落盘 reason**；`Busy` 是 `json:"-"` | `sessions.go:154-173`、`registry.go:234-254` | 「重启前是否在跑」必须新增 intent 持久化 |
| F4 | 「回合被中断」**可精确判定**：末条 assistant 且 `stopReason=toolUse`（含 toolCall）无后续 toolResult = 悬挂态；末条 toolResult = 步骤间；`stop/aborted` = 正常收尾 | 137 个真实 jsonl census + 8 文件抽样实测 | 恢复策略可按形状分派 |
| F5 | pi **不修复悬挂 toolCall**（悬挂条目原样保留，不补 toolResult） | leaf-2 实测 | 自动续跑悬挂态有**重复副作用**风险 → 只对安全形状续跑 |
| F6 | 配额耗尽**不报错**（返回普通 assistant 文本 + `stopReason:"stop"` + usage 全 0） | leaf-2 P5 实测 | 续跑后需校验 usage/特征，否则静默空转 |
| F7 | SSE 切换对浏览器自愈：陈旧游标 → `server_info{reason:"replay_overflow"}` → 清游标 + REST 全量 | `sse.go:130-152`、`sse.ts:290-320,375-430` | UI 无感；但 in-flight 回合计算已死，靠恢复补 |
| F8 | doing/dream 后台任务 **不可 resume**（409）；doing 按 `status != success` 续跑剩余 task；但门禁把遗留 `running` 判为 zombie 而失败 | `sessions.go:1018-1020`、`doing.go:144-166`、`helper.py:47-49` | 自动恢复须先归一化 `running → pending` |
| F9 | 现无任何 approve/confirm 语义（最接近的是 `rick web reset` 的 `[y/N]`） | `cmd/web.go:192-210` | `rick tools release` 自带人类确认（交互式确认 + `--yes`） |
| F10 | `MaxActive=8`；重放大会话成本高（12MB/3253 条） | `supervisor.go:43`、leaf-2 | 恢复要限流（队列 + 每会话一次 + 续跑预算 ≤3） |

### 6.2 模块与 pipeline

```
M1 提升入口（`rick tools release`：门禁校验 → 构建版本目录 → 原子切链 → 重启 prod → 健康探测 → 打印报告；`--rollback` 回滚）
   → M2 原子替换（$REPO/bin/releases/<ver>/{rick,dist} + `current` 链 + 保留最近 3 版；bin/rick 为链）
   → M3 关停快照（graceful 路径写 shutdown.json：{session_id, pi_session, last_entry_id, busy, intent}）
   → M4 启动自动恢复（候选集 = 快照 ∪ 注册表 active/running ∪ 近 60s closed 兜底 → 逐条 resume worker
                        → 形状判定 → 安全形状投递一次续跑 → 悬挂态挂起待人工 → 恢复报告）
   → M5 后台 job 恢复（doing/dream：running→pending 归一化 → 重新执行剩余 task）
   → M6 报告与回滚（恢复报告落盘 + 落 SSE/session_state；失败可 `release --rollback`）

接口契约：
  M1 → CLI 契约：`rick tools release [--yes] [--rollback] [--dry-run]`，退出码 {0 ok, 2 gate fail, 3 build fail, 4 restart fail, 5 health fail}
  M3 → 磁盘契约：`<state-dir>/shutdown.json`（version + sessions[]）；**新增 intent 字段走 Entry.Params["_auto_resume"]（零 schema 变更）**
  M4 → 判定契约：`classifyResumeShape(jsonlTail) → {COMPLETE | INTERRUPTED_AFTER_TOOL | DANGLING_TOOLCALL | ABORTED | FAILED | EMPTY}`
  M4 → 预算契约：续跑投递「每会话每次重启最多 1 次」+ 全局 ≤3（防配额风暴 R6）
  M6 → 报告契约：恢复报告 {recovered[], suspended[], failed[]}（落盘 + 前端可读）
```

**OKR 充分性自检**：M1-M6 达成 ⇒ 提升受控（人类确认 + 门禁 + 原子 + 可回滚）∧ 重启后所有「原本在跑」的会话被主动恢复（含后台 job 归一化续跑）∧ UI 无感 ⇒ KR3 ∧ KR4 成立 ✅

### 6.3 判断节点

**已终判（human 2026-09-20：整体确认）**
- J-L6-1 替换方式 = 版本目录 + `current` 链 + 保留前 N 版（回滚 = 切链 + 重启，无需重建）
- J-L6-2 重启方式 = kill 重启（优雅退出 ≤11s 预算）；SSE 自愈保证 UI 无感
- J-L6-3 恢复策略 = **A（全量恢复 worker）+ B（仅安全形状投递一次续跑）+ 悬挂态降级人工确认**
- J-L6-4 doing/dream = `running → pending` 归一化后续跑（幂等敏感点详见 M5）
- J-L6-5 人类确认 = **`rick tools release` 命令本身**（用户新增要求，取代原建议的 `POST /api/promote`；交互式确认 + `--yes`）
- **用户补充语义（重要）**：「交付阶段之前不影响 prod 任何会话」；「交付阶段允许会话中断，但**必须保证都可以恢复重启**」

**待 human 裁决（L6 补充追问）**

| # | 判断节点 | 选项 | 推荐 |
|---|----------|------|------|
| J-L6-6 | 「必须可以恢复」是自动还是人工触发 | (a) 启动**自动**恢复全部（悬挂态除外，挂起待人工）(b) 只标记可恢复，由人逐条点 Resume | (a)：符合用户原话「主动将运行中的 job 都恢复」；失败者降级 error + 进报告 |
| J-L6-7 | 后台 job（doing/dream）要不要自动续跑 | (a) 自动归一化 `running→pending` 并续跑剩余 task (b) 只标 error 等人工 | (a)：doing 的语义本就是「按 task 状态续跑」，且用户要求"job 不停止"；风险是中断的单个 task 会重跑（原子性靠 task 内的 gate/commit 纪律兜底） |

> 状态：**L5/L6 补充追问已发出**；两表裁决后 grilling 完成 → 进入实现流水线设计（tasks.json + 写域 + 分层 + gates）。

### 5.4 L5 终判（human 裁决 2026-09-20）

| 判断节点 | 裁决 | 落地含义 |
|---|---|---|
| J-L5-1 自举架构 | **(i)** | 开发会话托管在 prod，workspace = `$DEV_TREE`；dev 实例可任意重启 |
| J-L5-2 交付入口 | **`rick tools release`** | 唯一提升动作；人类执行即人类确认 |
| J-L5-3 dev 托管会话 | **允许**（用户：「这是我们使用 dev 进行测试关键」） | dev 实例是被测对象本身，必须能起会话；其会话在 dev 重启后走「挂起 + 手动恢复」 |
| J-L5-4 注册形态 | **a** | prod 注册表新增 `rick-dev`（path = `$DEV_TREE`），与生产仓库根并存但互不干扰 |
| J-L5-5 构建指纹 | **加 build_id** | `-ldflags -X` 注入 + `/api/health`（免认证）与 `/api/config`/SSE `server_info` 携带 → 一条 curl 判定「新构建在跑」 |
| J-L5-6 dev 前端模式 | **b** | 参数化 vite proxy（HMR 可用）；默认路径仍是「`npm run build` + overlay 软链」 |

### 6.4 L6 终判（human 裁决 2026-09-20）— **恢复语义被重新定义为「平台自动 + 会话挂起待人工」**

| 判断节点 | 裁决 | 落地含义 |
|---|---|---|
| J-L6-1 替换方式 | 版本目录 + `current` 链 + 保留前 N 版 | 回滚 = 切链 + 重启，无需重建 |
| J-L6-2 重启方式 | kill 重启（≤11s 预算）；SSE 自愈保证 UI 无感 | 浏览器只经历一次重连 |
| J-L6-5 人类确认 | `rick tools release`（交互确认 + `--yes`） | 提升动作本身即确认 |
| **J-L6-6 恢复触发** | **不自动恢复会话**（用户原话：「web ui 和后台 server 恢复即可，具体的每个会话可以是断开的，我们手动点击恢复使其继续。这样避免存在副作用」） | 交付重启后**保证平台可达 + 状态完整**，但**每个会话保持挂起态**，由人类一键恢复 → 彻底规避自动续跑的重复副作用与配额风暴 |
| **J-L6-7 后台 job** | **不自动续跑**（用户原话：「自动挂起，等待人类确认」） | doing/dream 也不自动 `running→pending`；挂起等人工点「继续执行」时才归一化并重跑剩余 task |

**KR4 重新表述（终版）**：
1. **平台级无缝**：`rick tools release` 重启后，web UI + server **自动**恢复可达（健康检查通过、状态文件完整、SSE 自愈、刷新页面依旧连通）——零人工介入；
2. **会话级挂起**：所有「重启前在跑」的会话/后台 job 被标记为**挂起（suspend）**而非含混的 error，UI 明示「因平台升级挂起」+ 一键恢复入口；
3. **人工一键恢复**：会话 → `resume`（复用既有 pi `--session` 语义）；doing/dream job → 恢复时才做 `running → pending` 归一化并续跑剩余 task；
4. **恢复报告**：重启后列出挂起清单与恢复入口（谁被挂起、一键恢复到哪）。

> 副作用边界的收益：不自动恢复 ⇒ 不需要「形状判定 + 安全续跑预算 + 配额风暴限流」（research-L6 §3.5 的 B 策略整体不需要），风险面从「自动重放工具调用」降为「人工确认后重跑」——这是本设计最重要的安全性决策。

**L6 终止判定**：M1-M6 pipeline 已按新语义澄清（M4/M5 由「自动恢复」改为「挂起标记 + 人工恢复入口 + 恢复报告」），接口契约齐备，OKR 充分性自检通过（平台自动 + 人工恢复 ⇒ 「必须保证都可以恢复重启」）。→ **L6 达标。**

---

## Grilling 完成声明

**本增量（第 2 棵设计树）已遍历完毕**：L4 隔离层 / L5 开发闭环层 / L6 提升与恢复层三层全部达标，所有模块已落实到**代码实现（文件 + 函数）/ 文件结构 / 工具调用（命令 + 参数）/ 环境依赖与配置（env + 配置文件）**四个维度；所有判断节点均由 human 裁决（无自行拍板）。

### 结构化决策摘要（按层）

**L4 隔离层**：独立 `HOME=$DEV_HOME`（唯一全局开关）+ 新增 `--state-dir` 显式化；独立 `RICK_PI_AGENT_DIR`（种子拷贝 auth/settings，生产 145MB runtime 不可被覆写）；git worktree `$DEV_TREE=/workdir/sunquan20/rick-dev`；`0.0.0.0:8414` + dev 专用 token；修 singleton（flock + 同状态目录拒绝）；启动 fail-fast 校验（HOME≠生产、不注册生产仓库根）；能力进仓库为 `rick` 正式子命令。
**L5 开发闭环层**：自举 = 开发会话托管在 **prod**、workspace 指向 dev 树（dev 可随意重启）；前端 = overlay 软链（零拷贝）+ 可选 vite HMR（proxy 参数化）；后端 = `build → 唯一命名 → 停旧 → 起新 → 健康 + build_id 校验`；可观测 = build_id 贯穿 health/config/SSE + `build.json` 状态文件；交付前**不影响 prod 任何会话**。
**L6 提升与恢复层**：入口 = `rick tools release`（人类执行即确认，支持 `--yes/--rollback/--dry-run`）；替换 = `bin/releases/<ver>/{rick,dist}` + `current` 链 + 保留前 3 版；重启 = kill（≤11s）；**平台自动恢复可达**；**会话与后台 job 一律挂起**（新增 suspend 语义 + UI「因平台升级挂起」+ 一键恢复）；doing/dream 恢复时才归一化 `running→pending` 续跑；恢复报告落盘 + 可读；不自动续跑 ⇒ 无重复副作用与配额风暴。

---

## 实现流水线映射（落盘后校正）

依赖 DAG 由工具实测的自然分层为 **4 层**（不是设计树里按 KR 画的 5 层），门禁按此 1:1 对应（gate 编号续接已交付旧流水线的 gate1-6，避免覆盖历史门禁）：

| 流水线层 | task（写域互不相交） | 门禁 | 覆盖的 KR |
|---|---|---|---|
| 第 1 层 | task16 隔离底座 · task18 构建指纹/前端参数化 | **gate7** | KR1 + KR3 的前置（指纹） |
| 第 2 层 | task17 dev-web 闭环 · task19 挂起/人工恢复/报告 | **gate8** | KR2 + KR4 |
| 第 3 层 | task20 前端挂起 UI · task21 `rick tools release` | **gate9** | KR2（可观测）+ KR3 |
| 第 4 层 | task22 端到端验收 + 文档 | **gate10** | 全部（验收 + 生产回归） |

**工作树纪律（human 指令 2026-09-21）**：prod 仓库工作树**零写入**——本增量全部产物（流水线规格、gate、代码、文档）落在 dev 工作树 `/workdir/sunquan20/rick-dev`；prod 只在每个 gate 末尾做**只读回归断言**（`sessions.json`/`web.json` 指纹 + 8413 健康）。

---

# 增量设计树（第 3 棵）：rick-rsi-loop —— 把「改进 rick 自身」制度化

> human 指令（2026-09-21）：不注册工作区那种零散做法不够——**每次启动「改进 rick」都必须加载 rick 源码里的 `rick-rsi-loop`**，用它驱动 rick 提供的工具完成自进化；并且**在生产 UI 里启动一个 rick 会话，若任务是改进自己，就自动加载这个 loop 并把流程走完**。
> 事实消解声明：loop 格式（五要素 + frontmatter）、`LoadLoopsContext` 只注入 name+trigger（软触发）、会话类型扩展点（`human-loop` 同构）、`git merge` 冲突/中止语义（**本轮实测**：冲突 exit=1 + `CONFLICT` + `git diff --diff-filter=U` 可列举 + `git merge --abort` 完全恢复；脏工作树被拒）——均已由本轮轻量自查/实测消解，**无遗留判断节点**；两个取舍点由 human 直接裁决（下方 J 表）。

## 根层（L0''）— OKR

**O**：「改进 rick 自身」这件事本身固化为 rick 的一条 loop —— `rick-rsi-loop`，且它是**唯一入口**：在生产/开发的 Web UI 里启动「RSI 自进化」会话即自动加载该 loop，loop 全程驱动 `rick tools dev-web` / 门禁 / `rick tools release`（含源码合并）完成自进化，产出可被机器校验。

**KR 集（充分性：KR1 ∧ KR2 ∧ KR3 ∧ KR4 ⟹ O）**

| KR | 内容 | 验证 |
|---|---|---|
| KR1 制度载体 | `.rick/loops/rick-rsi-loop.md` 存在且符合五要素规范；loops 目录 README 与实况一致；`rick tools loops_check` 可校验（把已有 `runLoopsAndSkillsCheck` 挂成子命令） | gate11 |
| KR2 入口绑定 | 新会话类型 `rsi`（CLI `rick rsi` + Web「RSI 自进化」）把 loop **全文注入系统提示词**；workspace 硬校验（必须是 rick 源码树、必须含该 loop、**不得是生产仓库根**）| gate12 |
| KR3 闭环执行 | `rick tools release --merge-source`：把 dev 分支合并进生产 main，**冲突即中止报错**（保留回滚点与现场供 AI 修复），无冲突才继续 binary/dist 原子提升 + 重启 + build_id 校验 | gate13 |
| KR4 可校验产出 | `rick tools rsi_check` 校验 loop 产出评估表：dev 实例 build_id 记录 / 门禁全绿 / **人类确认痕迹** / release 版本 + `.last` 回滚点 / 挂起-恢复记录 | gate13 |

**充分性自检**：KR1 给出「制度文本」→ KR2 把它绑定到启动动作（**"必须"才成立**）→ KR3 让 loop 的收尾动作可一条命令完成且失败安全 → KR4 让 loop 的承诺可被机器检查（从文档变成契约）。四者联合 ⟹「启动 RSI 会话就能把自进化流程走完」✅（维度：文本 / 入口 / 执行 / 校验，MECE）

## 层映射

| 层 | task | 门禁 |
|---|---|---|
| 第 1 层 | task23 loop 文档 + `loops_check` 挂载 + README 修正 | gate11 |
| 第 2 层 | task24 `rsi` 会话类型后端（校验 + method 注入 + `rick rsi`）· task25 前端 RSI 入口 | gate12 |
| 第 3 层 | task26 `rick tools rsi_check` · task27 `release --merge-source`（冲突即中止）| gate13 |
| 第 4 层 | task28 端到端（模拟生产）+ 文档 + prod 注册 rick-dev 操作手册 | gate14 |

## 判断节点（human 已裁决）
- **J-RSI-1 唯一入口**：loop 必须由**会话类型**绑定（不是靠 agent 自觉读文档）→ 裁决：**是**（否则"必须"无法保证）
- **J-RSI-2 源码合并**：release 合并 dev→main；**冲突即终止报错**，由 AI 修复后重新 release（human 原话）→ 裁决：**是**（不自动解决冲突）
- **J-RSI-3 运行位置**：RSI 会话的 workspace 必须是 **dev 工作区**（生产仓库根被硬拒，否则直接改生产源码）→ 裁决：**是**
