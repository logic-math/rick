# skill:rick-release-ops（rick release 共享机实操）

## 触发场景

在本共享开发机（生产仓库属主非当前用户、bin/releases 不可写、机器会休眠）执行 `rick tools release` 提升时使用。信号词：
- release 报 `mkdir <prodRepo>/bin/releases/<ver>: permission denied`
- release --detach 后跨了机器休眠期才完成
- dry-run 版本号与实际提升版本号不一致，rsi_check 报 build_id 不匹配

## 预期效果

- release 一次成功（不再踩 bin/releases 权限坑）
- 版本号/证据记录正确（rsi_check prod-health 一次过）
- detach 进程跨休眠期完成时不误判失败

## 核心内容

1. **发布目录重定向**（bin/releases 属主 sankuai 不可写——release.go ReleasesRoot 注释记载的已知问题）：
   ```bash
   RICK_RELEASES_DIR=/home/hadoop-recsys/.rick/releases rick tools release --merge-source --detach --prod-repo <prodRepo>
   ```
   bin/rick 软链 → releases/current/rick 不变，start-web.sh 不用改
2. **版本号语义**：`version=<sha7>-<构建时刻>`，非 dry-run 时刻——批准留痕提交会推进 dev HEAD，实际构建 sha7/时间戳与 dry-run 不同属正常；rsi_check 的 release.md 头部 `version=` 必须写**实际生效版本**（`cat ~/.rick/release-state.json` 取）
3. **detach 跨休眠**：setsid 后台进程随机器休眠暂停、恢复后继续完成（实测跨 4 天完成 merge+restart）——launch 后看 `~/.rick/release-state.json` 的 `stage/ok` 判定，勿因「没立刻完成」误判失败
4. **生产接管**：若休眠期有外部进程占了 8413（如 ~/.rick/bin/rick），release 重启会 `stopped=[...]` 接管——健康校验 build_id 匹配即成功
5. **会话挂起预期**：S5 必挂起 web 承载的 RSI 会话——恢复流程见 rick-rsi-loop S6（human 点「恢复继续」）
6. **S3 演练阶段 proactive 触发**：本共享机做 release 前必读本 skill——勿等失败后才在争夺窗口里被动选方案。dry-run 全绿 ≠ 落点可写（Plan.DryRun 落 /tmp 暂存，结构性测不到真实提升时的 permission denied）。dry-run 后立即做 30 秒「落点可写性体检」，逐项探测并输出 `RELEASE_PREFLIGHT writable=<path>=true/false`：
   - ReleasesRoot：试建+删临时子目录
   - bin/rick 软链
   - overlay dist 目录
   - state 文件（~/.rick/release-state.json）
   - start-script（start-web.sh）

   出现 false 时勿自行拍板，按两条路线向 human 报价（附权衡）：
   - 解药1 `RICK_RELEASES_DIR` 重定向（internal/env/release/defaults.go:51-52 实证生效）：自助立即生效，但非 canonical 位置
   - 申请属主授权 `setfacl + chmod`：一次性授权，default ACL 对新建子目录持久生效，canonical 位置不变
7. **载荷定界索引手术三协议**（做 `git rm --cached` 等索引手术时逐条执行）：
   - ① 显式路径枚举：逐个列出完整路径，禁止目录前缀通配
   - ② commit 前 delta 核对：`git diff --cached --stat` 逐文件核对，预期清单之外多一条即停
   - ③ base 视角复核：`git diff --stat <merge-base>...HEAD` 确认只有 base 中不存在的新路径退出跟踪；base 内路径退出跟踪 = merge 时删除生产历史，须 human 显式确认

   反例（job_44）：`git rm --cached .rick/jobs` 目录前缀 → 921 条删除 delta（merge 会删生产 job 历史），靠 43 秒人工警觉救回
8. **release 前预检：dev web 在跑则先 down**（job_50 实测门禁连挂 ×4，~30 分钟诊断+重试成本）：release 门禁 buildEnv 用 dev HOME（`Plan.DevHomeOr` → rick-dev-home），internal/web 测试 spawn 的 worker 与在跑的 dev web 实例（活动会话）共享该 HOME 的 pi sessions → worker 异步终止滞后于 `t.TempDir` 清理 → TempDir RemoveAll cleanup 竞态失败（单测/低载通过、门禁全量必挂，症状：不同测试轮换挂 + `directory not empty`）。**release 前 `rick tools dev-web down`，release 后 `rick tools dev-web up --no-build` 恢复**（迁移中的教学会话产物已落盘 loop 目录，无损失）。代码级根治（门禁自动 down/恢复 或 测试 cleanup 前 wait worker 退出）记 domain/bugs.md backlog。
