# 真实 release 首次执行复盘（2026-09-22，含一次生产中断事故）

## 时间线
1. ⓪ 提交 prod 树遗留 job 文件（17 个）+ 清掉游离的 research.md（与 grilling/L5-leaf-4 md5 相同的重复产物）→ 工作树干净，满足 --merge-source 前置
2. ② 注册 rick-dev 工作区到生产（additive，201）
3. ③ dry-run：门禁 pass、版本 41726a8-260922133031、prod_untouched=true
4. ④ **第一次真实 release → 冲突中止（安全网按设计生效）**：design-tree.md 冲突（prod main 5ab529e 与 dev 分支都改了它）→ release 自动 `git merge --abort`，工作树零残留
5. AI 修复合并：分析确认 dev 版本是 prod 版本的**严格超集**（643→694 行，132 段落 0 缺失）→ `git checkout --theirs` → 提交 merge commit `7f6665e` → main 自检（go build + 3 包测试全绿）
6. ⑤ **第二次 release → 提升 + 换链成功，但发生生产中断事故**（见下）
7. ⑥ 手动拉起生产 → health 200，build_id == 版本 ✓；RSI 闭环在生产实测通过（loop 注入 6/6、守卫 400 中文原因、恢复报告结构正确）

## ⚠️ 事故：生产中断 ~10 分钟（根因：release 未用 --detach）

- 现象：release 停掉旧生产、完成换链后，**新进程没有起来**（8413 无监听）。
- 根因：执行 release 的 shell（AI 会话的 bash 工具调用）**被中断**，release 作为其子进程被连带杀死 —— 恰好停在 stopProd 之后、startProd 之前/之中。判断依据：工具调用「No result provided」+ 换链已完成但无新进程。
- 恢复：`setsid nohup ~/.rick/start-web.sh`（标准启动路径，走 releases/current 软链）→ 0.5s 内 health 200。
- **教训（写入 loop 与 wiki 的改进项）**：
  1. release **永远**用 `--detach`（setsid 脱离调用方进程树）—— 不论执行者是否被生产托管。本次的「我不在 prod 托管下所以不需要」的判断是错的：调用方本身可能被中断（工具超时/会话切换）。
  2. `rick-rsi-loop` 的 S5 步骤应把 `--detach` 从「建议」升级为**必须**（下一次 loop 修订时改）。
  3. release 中断后的恢复路径已验证：直接跑 start-web.sh 即可（换链已原子完成，current 指向健康构建）。

## 其他事实记录
- `.last` 回滚点为空 = **首次建链的正常语义**（之前没有版本链，无上一版可记）；下一次 release 起会正常记录。
- 旧二进制（4.4.15, md5 643b54a3）已随 bin/rick 被替换；应急回退路径 = `git worktree` 到 a60035a 重建，或切 current 到任一已有 release 目录。
- 生产重启后无任何会话被挂起：重启前 6 条会话全是 error/closed（无 active）→ recovery-report `suspended:[]` ✓ 符合「只挂起在跑的」语义。
- 用户当天自己在 MLOPS 建的 easy job_15 会话不受影响。

## 生产终态（验证输出）
- health: {"status":"ok","build_id":"41726a8-260922133419","started_at":"2026-09-22T13:43:41+08:00"}
- bin/rick → releases/current/rick（版本链建立，current → 41726a8-260922133419）
- 8 个工作区（含 rick-dev）
- RSI 会话实测：type=rsi、title="RSI rsi_1"、_method_file=/workdir/sunquan20/rick-dev/.rick/loops/rick-rsi-loop.md、status=active
- 守卫实测：workspace=生产仓库根 → 400「请改用 dev 工作区…」

---

## 附：F6 —— level_complete 的 tasks.json 写回晚于提交（发现于 learning 收尾）

**现象**：`level_complete(task29)` 打印了 commit `59ac918f`，但该提交里 tasks.json 的 task29 仍是 `pending`；success 状态只存在于 dev 树的**工作树**（被我随后的 `git add -A` 同步提交进 70a259e）。结果 main（经 release 的 --merge-source 合并 59ac918f）上 task29=pending。
**影响**：仅 job 记账陈旧（29/29 实际全 success）；不影响代码与生产。
**修复**：手工把 prod 树 tasks.json 的 task29 标 success（commit_hash=59ac918f）并提交。
**待办（下次 RSI 迭代的候选修复项）**：检查 level_complete 钩子的实现顺序——状态写回应在 `git add -A` **之前**落盘（或写回后再次提交），否则每次层提交都可能把 pending 状态固化进 git。
