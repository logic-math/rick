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
