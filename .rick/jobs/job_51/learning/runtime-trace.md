# Runtime Trace（job_51）

## 轨迹 1：super session 报错 → 两层修复
1. web UI 报「缺失 super-loop.md」→ 读 sessions.go 发现资产检查逐 workspace（rickDir=<ws>/.rick）
2. 部署 9 资产到 MT_INFER → 深挖第二层：superopt 调度器 `rick super start` 实测「No module named superopt」（cwd=MT_INFER 向上无 super/）
3. resolveRepoRoot 加 exe fallback（os.Executable+EvalSymlinks 穿 symlink 链）+ 单测（假 repo+symlink 链+外部 cwd 三要素，macOS /var↔/private/var 归一断言）
4. 顺手修 TestDefaultPlanReadsPort 既有失败（job_49 P1-1 加固 start-web.sh 用 PORT=6137+$PORT 写法，parseScriptFlag 命中 $PORT 提前返回）——字面量 $ 前缀跳过 + 变量赋值回退
5. S3 演练绿 → S5 release → 生产 65c2cd1

## 轨迹 2：proxy 杀不死之谜（5 次重启才定位）
1. 用户要求清 .rick 里 proxy → start-web.sh 已干净（11-01 裁决过）
2. 重启实例仍带 proxy → 怀疑 prodEnv 透传（release 从带 proxy shell 跑）→ 修 prodEnv（08b5782）
3. 手动干净重启（env -i）→ 会话死在 kill 后（自己杀自己宿主）→ 实例回来还是带 proxy
4. setsid 重启 → macOS 无 setsid 命令！失败
5. nohup+disown 重启 → 又被抢锁（84721 先到）→ 发现每次「恢复」都从 launchd zsh env 拉起
6. **破案**：launchctl list 发现 com.rick.web.plist（KeepAlive=true + EnvironmentVariables 硬编码 http_proxy/https_proxy）——每次 kill，launchd 5 秒拉回带 proxy 实例（ps eww 对 pi 进程有截断假象曾误导排查方向）
7. 修 plist（删 2 变量留 no_proxy）+ bootout/bootstrap → 实例/会话/直连全干净
8. GitHub push：104+105 条积压 → 403 根因也是 proxy（plist 灌的）→ env -u 绕过 + GH_TOKEN（zshrc）→ 成功

## 关键工具性教训
- `ps eww <pid>` 在 macOS 有输出截断假象：pi 进程显示无 proxy 但 bash 有——排查继承链时不可全信，用 `env`（进程内自证）交叉验证
- 排查「杀不死的进程」第一步就查 launchd（launchctl list | grep xxx）——KeepAlive=true 会自动复活
- macOS 无 setsid 命令：脱离进程组用 rick 自身 --daemon（Go syscall setsid）或 nohup+disown
