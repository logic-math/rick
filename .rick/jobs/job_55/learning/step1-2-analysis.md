# Job_55 Step 1+2 分析（parent 亲自执行——worker 连续两次 API 瞬时异常后接手）

## Step 1a：执行轨迹的分身体系视角分析

### 轨迹段 1：10.99.19.123 远程部署（分身机制的实战雏形）

**做了什么**：内网服务器（super 版 rick，infer-rsi.git 代码）super 模式报「找不到 super loop 文件」。根因：web `createWorkspaceDirs` 只建空目录零知识复制 → 修复 `internal/super/assets.go` 自动部署 9 件资产（DeploySuperAssets）→ infer-rsi dev_rick 分支推送同步。

**分身启示**：这正是「分身机制」的核心问题——**给一台新机器装 rick 后，知识资产（domain/loops/skills）不会自动到位**。本次修复只覆盖了 super 的 9 件资产；完整的分身机制需要全量知识复制 + 环境适配（IP/密码/路径重写）。

**踩坑**：ssh 22/2222/8080/9090/6137 TCP 全超时浪费多轮探测——实际访问方式是 sshpass 沙箱直连（应该先问 human 或查 domain 记录）。

### 轨迹段 2：daemon 架构（分身通信/控制的地基）

**做了什么**：web 重启杀全部 pi worker（stdin/stdout 管道耦合）→ 新架构：pi 跑在 node shim 后（unix socket），web 重启只 detach，重启后探活 re-attach。四路径：Spawn（幂等拒绝重复）/ 重启（DetachAll 不杀）/ 对账（Attach + get_state 端到端探活 + suspended 探活恢复）/ 显式 close（kill -pgid 整组）。

**分身启示**：
- **分身通信的 API 基础已具备**：`POST /api/sessions/{id}/prompt` = 向分身的会话发消息；`POST /api/sessions/{id}/steer` = 中途纠偏；`GET /api/sessions` = 观察分身状态。daemon 架构让分身会话可长期存活（重启不断）→ 远程控制稳定。
- **EPIPE 教训**（跨进程树 stdio 必须落文件）适用于任何分身进程编排。

**踩坑**：EPIPE 真 bug 只有真 e2e（kill 真进程）才暴露——Go 单测进程活着管道不断。分身 skill 的验证清单必须含真进程死亡测试。

### 轨迹段 3：release 三连 8.1.12/8.2.1/8.2.2（分身升级的实战）

**做了什么**：release --version N --merge-source --detach 从 dev 树发版：门禁（go test ./... + npm build）→ merge dev→main → 构建带指纹 → 原子提升 → 生产重启。门禁两次挂在「测试环境依赖」（HOME=dev-home vs 正常 HOME 的 resolveRepoRoot 优先级链差异）——**本地测试环境 ≠ 门禁环境**。

**分身启示**：**分身升级 = 把这套 release 流程泛化到远程**：向分身机器传新二进制 → 远程健康检查（build_id 指纹比对）→ 远程重启。dev rick 就是本机分身（dev-web init/up + 指纹校验 + release --merge-source 合并回来）——**dev 模式 = 特殊分身的完整代码级先例**。

### 轨迹段 4：dev rick 模式（= 特殊分身实证）

**做了什么**：dev-web init（独立 worktree + 独立 HOME + 独立 pi 沙盒 + 8414 实例 + overlay）→ dev 验证 → release 提升生产。

**分身启示**：dev 模式的四要素（隔离树/隔离状态/指纹校验/单向提升）就是分身机制的骨架。远程分身 = 把 dev 模式从「同一台机器的 worktree」泛化到「跨机器 ssh」。

## Step 1b：素材映射表

| 素材 | → 对应 skill 的核心内容 |
|---|---|
| 10.99.19.123 部署 + infer-rsi 同步 | **avatar-provision**：ssh 直连方式探测（问 human/domain 优先于扫端口）、二进制安装（--goos 交叉编译）、知识资产复制（dump 或 git sync）、IP/密码/路径适配 |
| `rick tools dump/import`（--workspace/--sessions/--goos/--exclude + sha256 边车） | **avatar-provision**（全量记忆迁移）+ **avatar-session-replication**（单 job 定向复制：--workspace 限定 + sessions 选择） |
| pi-session-portability 先验（pi 会话历史跨机可 resume） | **avatar-session-replication**：pi session jsonl + 注册表条目 + workspace 文件三件套一起走 |
| dev-web init/up/status/down + 指纹校验 | **avatar-upgrade-debug**：dev=特殊分身的四要素骨架（隔离树/状态/指纹/单向提升），分身升级=远程泛化 |
| release --version --merge-source --detach + launchd 重启 | **avatar-upgrade-debug**：升级流程（门禁→构建→传输→远程重启→build_id 指纹验证） |
| POST /api/sessions/{id}/prompt / steer / GET sessions | **avatar-communication**：web API 即分身控制协议（发消息/纠偏/观察），daemon 架构保证会话存活 |
| EPIPE 教训 + 真 e2e 验证法 | 全部 skill 的「踩坑预案」+「验证清单」内容源 |
| release 门禁环境差异教训（HOME=dev-home） | **avatar-upgrade-debug**：分身环境与本地环境的差异前置检查 |

## Step 2：路径改进评估（→ skill 的前置检查清单）

### 路径 1：10.99.19.123 排查
- **冗余**：TCP 扫 5 个端口浪费 3 轮 → **前置检查**：先查 domain/bugs.md 和问 human「访问方式」，扫端口是最后手段
- **更短路径**：sshpass 直连方式其实 domain 里可能有记录（或 human 一句话）——分身 skill 里「连接方式发现」要按顺序：问 human → 查 domain → 查 ~/.ssh/config → 扫端口

### 路径 2：daemon 架构交付
- **可预防错误**：EPIPE bug 的前置防线 = 「跨进程死亡的进程树 stdio 一律落文件」写进 domain（已做）；e2e 验证法（kill 真进程 + ps pid 比对）固化为 skill 的验证清单
- **baseline 对照法**（stash 改动跑干净 HEAD 判定既有抖动）值得进 skill——避免「环境抖动误判为自己的回归」

### 路径 3：release 门禁反复
- **可预防错误**：**本地测试环境 ≠ 门禁环境（HOME 差异）**——release 前先在门禁环境（HOME=dev-home）跑一次目标包测试——这条进 avatar-upgrade-debug 的前置检查
- **生产树脏数据**：并发会话（job_57）写 .rick/ 导致 merge 门禁挂 2 次 → 快照提交解锁模式（chore: 快照（解锁 release merge 门禁））已实战 3 次，固化为 skill 步骤

## 未覆盖空白（skill 中标注「待实战补全」）

1. **分身的认证/安全**：sshpass 明文密码管理、token 分发（web_token 给分身生成独立 token）——本 job 未涉及
2. **分身状态上报**：分身主动向主身上报（心跳/工作进度）——本 job 只有主身拉取（GET API），无推模式
3. **多分身编排**：同时控制多个分身的拓扑/路由——本 job 只有单分身（10.99.19.123）
4. **跨平台二进制**：--goos linux/amd64 交叉编译已在 dump 里支持，但远程 Linux 分身的完整安装（launchd vs systemd 差异）未实战
