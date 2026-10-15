# skill:baseline-verify-with-worktree（git worktree 隔离做 baseline 对比）

## 触发场景

当需要验证「某个失败是否由我的改动引入」（检出改动前的 commit 跑同样命令做对照）时使用。信号词：

- 疑似 pre-existing flaky / 环境性失败，需要 baseline 对照
- 想用 `git stash` 暂存当前改动去跑旧代码——**停，改用本技能**

## 预期效果

- 零风险对比：主工作树自始至终未动，未提交改动不受影响，无 stash 残留
- 一次到位：不需要事后验证「工作树是否被 stash 事故污染」（job_46 中 stash 方案每次都需要 `git diff stash@{0}` + build 复验来确认完整性，worktree 方案零此类成本）

## 核心内容

```bash
git worktree add /tmp/rick-baseline <commit>      # 检出任意历史 commit 到隔离目录
cd /tmp/rick-baseline && <同样的验证命令>          # 例：go test ./internal/web/ -count=1
git worktree remove --force /tmp/rick-baseline    # 用完即删（--force 处理含构建产物的树）
```

**为什么禁 stash**（job_46 两次事故实录）：`git stash -u` + 多命令链 + `git stash pop` 在 agent 工具环境下不可靠——链中输出被吞、pop 与 untracked 交互产生「stash 残留 + 工作树疑似污染」的混乱中间态（表现为：stash list 非空但工作树又有全部改动，无法直接判断 pop 是否成功）。worktree 从根上消除这类状态。

**对照结论的读法**：baseline 同样失败 → pre-existing（你的改动无因果，可安全继续，但要在产出记录里注明证据）；baseline 通过 → 你的改动引入，回 worker 修。

**顺带的进程管理纪律**（同一类「编排 shell 自伤」事故，job_46 C2）：远程/脚本里杀进程先 `pgrep -f <pattern>` 拿 pid 再 `kill $PID`，禁 `pkill -f`——后者会匹配承载命令自身的 bash -c 字符串把自己杀掉。
