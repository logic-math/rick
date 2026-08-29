# rick web 用户指南

> rick web 是 rick 的浏览器入口（第一层 WEB-UI）：在多端浏览器中完整操作 rick 的核心功能——多工作区管理、7 类 cmd 会话、doing 监控看板、知识库浏览与前端自迭代。风格致敬 Rick and Morty（深色星空 + 传送门 + 飞碟）。

## 功能总览

### 多工作区（全机唯一服务）

- `rick web` 是**机器级单例**：任意目录启动，全机只跑一个实例（pid 判活 + 端口检测，重复启动会提示已在运行）。
- **工作区 = 含 `.rick/` 目录的项目根**。在 web UI 左侧栏「添加工作区」输入路径注册（服务端校验 `.rick` 存在），机器级持久化于 `~/.rick/web.json`。
- 一个会话锚定一个工作区（pi 子进程 `cmd.Dir` = 工作区根）；一个工作区可开多个会话；**多工作区同屏、多 job 跨工作区并行**。
- ⚠️ 已知限制：同一工作区并发跑两个 doing 不做互锁（会竞争 git commit，CLI 亦然）——由用户自行保证同一工作区同一时间只有一个 doing。

### session 中心模型（7 类会话）

基本组件是 **session = 一个聊天窗口事件流**；每个 session 有类型（对应 rick cmd）：

| 类型 | 形态 | 启动参数 | 说明 |
|---|---|---|---|
| plan | 交互聊天 | requirement（可选复用已有 job） | 规划会话，产物落 `jobs/{id}/plan/` |
| easy | 交互聊天 | requirement（可选 ctx-path） | 轻量执行 |
| doing | 监控看板 | job | task 状态看板 + 编排事件流 + gate 结果；可中止（与 CLI Ctrl+C 同语义，attempt+1 续跑） |
| ctrl | 交互聊天 | job | 干预会话 |
| human-loop | 交互聊天 | topic | SENSE 深度思考，产物落 `draft/rfc/` |
| learning | 交互聊天 | job | 知识沉淀 |
| dream | 双模 | job-num + mode（交互/后台） | 交互=聊天；后台=监控看板 |

- **交互聊天型**：pi `--mode rpc` 常驻子进程；事件流实时渲染（assistant 文本流式、工具调用折叠卡、thinking 折叠）；输入框可发送 prompt，agent 运行中自动转为 steer（中途纠偏）；abort 按钮中止。
- **监控型**：goroutine + pi `--mode json` 一次性执行；看板实时刷新（tasks.json watcher → SSE）。
- **closed 会话**：离线浏览（读 pi session JSONL，不起进程）；**Resume** 按钮重启（`--session` 恢复原 pi 会话 + 增量补流）。
- agent 需要人确认时（confirm/select/input/editor）会弹出 **web 表单对话框**（extension_ui 双向桥）。

### Jobs 看板（P0 只读）

工作区维度查看所有 jobs：任务状态点阵、task 详情、plan/doing 文件浏览（task1.md、act-path、debug 日志等）。

### Knowledge 浏览（P0 只读）

浏览 `.rick/domain|loops|skills` 知识库：文件树 + Markdown 渲染。

### 前端自迭代（对话改 UI）

```
embed baseline（随 rick 二进制）
   │ rick web customize（或 Settings → Customize UI）
   ▼
~/.rick/web/src/（可写源码副本 + 根构建配置 + public/）
   │ 任意 agent 会话中：「把会话列表改成双栏」——agent bash 编辑 ~/.rick/web/src/** 后 npm ci && npm run build
   ▼
~/.rick/web/dist/（覆盖层，静态服务优先于 embed baseline）
   │ watcher（2s 轮询）→ SSE frontend_reload → 浏览器自动刷新
   ▼
改动自动生效
   │ 改坏了？
   ▼
rick web reset（或 Settings → Reset to baseline）→ 删覆盖层回 baseline
```

## 启动

```bash
rick web                    # 默认 127.0.0.1:6137（C-137 彩蛋），token 自动生成（打印一次并写 ~/.rick/config.json 的 web_token）
rick web --port 8080        # 换端口
rick web --listen 0.0.0.0   # 对局域网开放（默认仅本机）
rick web --token <token>    # 显式指定 token（优先级高于 config）
```

首次启动后浏览器打开 `http://127.0.0.1:6137`，输入 token 解锁（存在浏览器 localStorage）。

子命令：

| 命令 | 作用 |
|---|---|
| `rick web customize` | 抽取内嵌前端源码到 `~/.rick/web/src/`（自迭代入 口；幂等——已有自定义层则跳过） |
| `rick web reset` | 删除自定义层回内嵌 baseline（注册表 `web.json`/`sessions.json` 保留不受影响） |

## PWA 安装说明

rick web 是 PWA（可安装 Web 应用）：

- **localhost 或 HTTPS** 下：浏览器地址栏「安装」图标（或菜单「添加到主屏幕/安装应用」）→ 全屏运行、独立窗口、离线壳缓存。
- **局域网 IP 直连 http** 时：service worker 不可用（浏览器安全上下文要求），降级为普通响应式网页——功能完整，仅无「安装到桌面」能力。
- 移动端推荐经反向代理（HTTPS）访问以获得完整 PWA 体验。

## 安全模型

- **单用户 token 认证**：REST 走 `Authorization: Bearer <token>`；SSE 走 `?token=`（EventSource 无法带 header）。静态资源不设防（SPA 壳）。
- 默认只监听 `127.0.0.1`；`--listen 0.0.0.0` 显式开放局域网（自担风险，token 是唯一防线）。
- **公网暴露**：务必走反向代理 + HTTPS（如 caddy/nginx/cf tunnel），token 经 header 传递；参考响应头 `X-Accel-Buffering: no`（SSE 代理透传，nginx 需 `proxy_buffering off`）。
- 云端同步 = agent 自主 git 行为（.rick 仓库 commit/push 由 rick 协议与 agent 自主完成），web UI 无同步控制面。

## 已知限制

1. 同工作区并发 doing 无互锁（用户自行保证；跨工作区并行无限制）。
2. LAN http 下 PWA 降级（见上）。
3. 大文件（>2MB）的 job 文件/knowledge 文件会拒绝读取（防内存炸）——在仓库里直接看。
4. web UI 不管理 `tools init-pi/theme`（P3 明确排除，CLI 已够用）。
5. 真实 pi 会话交互需本机 pi 就绪（`rick tools init-pi` 已配置的托管运行时）。
