# skill:avatar-upgrade-debug（分身升级与调试——升级分身二进制/远程调试/修改分身源码）

## 触发场景

当需要**升级分身的 rick 二进制**、**远程调试分身的工作**、或**修改分身的源码**时使用。信号词：

- 「给分身升级 rick」「分身版本落后了」「远程更新那台机器的 rick」
- 「分身上跑的不对，帮我调试」「拉分身的日志看看」
- 「改分身的源码」（内网 git 仓库分身，如 infer-rsi super 版）

**本质定性（human 2026-11-29）**：dev rick 就是 rick 的一个**特殊分身**——dev-web 模式（独立 worktree + 独立 HOME + 独立 pi 沙盒 + 8414 实例）是分身在本机的完整代码级先例；远程分身是同一套四要素（隔离树/隔离状态/指纹校验/单向提升）的跨机器泛化。操作 dev rick 时按本 skill 的本机分支走。

## 预期效果

- 分身升级后 build_id 指纹一致（**测的就是升的那个**——dev-web-fingerprint 教训：旧二进制启动 health 照样 ok，30-60 分钟整轮浪费）
- 远程调试拿到分身真实状态（日志/会话/健康），不靠猜
- 源码修改经 git 分支同步（可审计、可回滚），不 scp 覆盖
- 分身会话跨升级零中断（daemon 架构 8.1.12 起——升级重启不掉任何进行中的 job）

## 核心内容

### ① 升级分身二进制（远程版 release 流程）

```bash
# 1) 构建（主身，按分身平台）：rick tools dump --goos linux/amd64（含 bin/linux-amd64/rick）或 CI 产物
# 2) 传输 + 校验：
scp rick rick.sha256 <user>@<avatar>:/tmp/ && ssh <user>@<avatar> 'shasum -a 256 -c rick.sha256'
# 3) 原子替换（分身机）：先备份再换——
ssh <user>@<avatar> 'cp ~/rick/bin/rick ~/rick/bin/rick.bak.$(date +%s) && install -m 0755 /tmp/rick ~/rick/bin/rick'
# 4) 重启（分身机，按服务管理方式）：
#    macOS launchd：launchctl kickstart -k gui/$(id -u)/<label>（或 kill 让 KeepAlive 拉起）
#    systemd：systemctl --user restart rick-web / nohup 后台：kill <pid> 后同命令重启
# 5) 指纹复核（关键——不可省）：
ssh <user>@<avatar> 'curl -s --noproxy "*" http://127.0.0.1:<port>/api/health'
#    build_id 必须 == 本次升级版本；不符 = 旧二进制还在跑（软链没换/服务没重启/缓存进程）
```

### ② 远程调试三件套

```bash
# 日志拉取：
ssh <user>@<avatar> 'tail -100 ~/.rick/web.log'        # web 主日志（[rick-web] 前缀）
ssh <user>@<avatar> 'tail -100 <agent-dir>/sockets/*.sock.log'   # pi worker stdio（daemon 架构落文件）
# 会话状态观察（分身 web API——见 avatar-communication）：
ssh <user>@<avatar> 'curl -s --noproxy "*" -H "Authorization: Bearer <token>" http://127.0.0.1:<port>/api/sessions'
# 远程复现：ssh 执行 + 输出回传
ssh <user>@<avatar> 'cd <ws> && python3 .rick/jobs/<job>/plan/gates/gate1.py'
```

**环境差异前置检查（job_55 门禁实战教训）**：分身/门禁环境 ≠ 本地环境——`resolveRepoRoot` 等按 HOME/RICK_PI_AGENT_DIR 解析的逻辑在 `HOME=dev-home` 下走不同分支（① `~/.rick` 安装快照优先于 ② cwd 上溯）。在分身上跑测试/复现前，先 `ssh … 'env | grep -E "HOME|RICK"'` 看环境变量，必要时按分身 HOME 复跑目标测试。

### ③ 修改分身源码（git 分支同步模式，job_55 实战：infer-rsi super 版）

```bash
# 1) 主身改动 → 推到分身可拉取的分支（内网 git）：
git checkout -b dev_rick && git commit -m "fix: …" && git push origin dev_rick
# 2) 分身机拉取 + 重建：
ssh <user>@<avatar> 'cd /workdir/hpc/build_rick && git pull origin dev_rick && go build -o bin/rick ./cmd/rick/'
# 3) ③升级流程替换二进制 + 指纹复核
# 4) 回流：分身上验证过的修复合并回主身分支（PR/merge），保持单一事实源
```

**纪律**：不 scp 散文件覆盖分身源码（不可审计）；checked-in 二进制（如 Linux 产物）在构建期更新（`chore: linux/amd64 二进制更新（main@<sha> 构建）`），不留过期二进制。

### ④ dev rick（本机特殊分身）操作分支

```bash
# dev = 本机分身：独立 worktree + 独立 HOME（RICK_DEV_HOME）+ 8414 实例
rick tools dev-web up        # 改完代码标准动作（build + up + 指纹）
rick tools dev-web status    # DEV_STATUS … fingerprint=ok/MISMATCH
# 指纹前置检查 + 部署收敛 + 升级回主身（release --merge-source）：见 skill:dev-web-fingerprint 与 skill:rick-release-ops
```

## 踩坑预案

- **旧二进制陷阱**：health `status:ok` ≠ 新版本在跑——必须比 build_id（dev-web-fingerprint 实测 30-60 分钟浪费案例）
- **`up --no-build` 挑错产物**：dev-home/bin 里陈旧二进制名字典序可能排在最新之后（如 f/e 前缀 > 7/8 前缀）——清理陈旧二进制或用全量 up
- **生产/分身树脏数据阻塞 release 门禁**：并发会话写 .rick/（jobs/learning/drafts）→ merge 门禁「工作树不干净」中止——快照提交解锁：`git add .rick/ && git commit -m "chore: 快照（解锁 release merge 门禁）"`（job_55 实战 3 次）
- **并发 release 抢跑**（job_55 实战）：别人同时发版会用旧源构建——发版前对齐「这次谁发、什么版本」，冲突后核对 CHANGES 的 dev HEAD
- **分身机服务管理差异**：macOS launchd KeepAlive（kill 自动拉起）vs Linux systemd vs nohup——升级重启前先 `ps` 确认管理方式，kill 后验证拉起

## 待实战补全（如实标注）

- **Linux systemd 服务单元**：分身 Linux 化的正式 unit 文件/开机自启（当前实战仅 macOS launchd + nohup）
- **升级的灰度/回滚**：分身升级失败的自动回滚（当前=手工 rick.bak.<ts> 恢复）
- **分身源码的持续同步**：主身→分身的自动 CI 同步（当前=手工 git push/pull）
