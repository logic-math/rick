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
