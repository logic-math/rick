# 依赖关系
task20
task21

# 写域
scripts/self-evolve-e2e.sh
wiki/self-evolve.md
.rick/jobs/job_36/plan/sim-report.md

# 任务目标
端到端验收「在 rick web 里改进 rick web」这条链路，并把运维手册与已知边界固化下来（含本轮调研顺带发现的两个缺陷的记录）。

# 关键结果
1. `scripts/self-evolve-e2e.sh`（可重放、幂等、**默认不碰生产**）：
   - ① `rick tools dev-web init`（worktree + dev HOME + pi 沙盒 + overlay 软链）
   - ② `rick tools dev-web up` → 断言 dev 健康 + `build_id` 存在 + 指纹一致
   - ③ **隔离断言**：dev 与 prod 的 `sessions.json`/`web.json` 互不可见；dev 注册生产已注册的 workspace → 被拒；同 state dir 起第二实例 → 被 flock 拒
   - ④ 前端热更断言：改 `web/dist`（touch/替换 index.html）→ 覆盖层生效（`GET /` 返回新内容），**dev 进程未重启**
   - ⑤ 后端闭环断言：改一处 Go 源码（如常量）→ `dev-web restart` → 新 `build_id` 变化且健康
   - ⑥ 挂起/恢复断言（对**模拟生产**：临时 home + 端口 18413）：重启模拟生产 → 在跑会话变 `suspended` 且**未自动恢复**（worker 数 0）→ 调 `/continue` → 状态回 active；doing 归一化：构造遗留 `running` task → `/continue` 后变 `pending` 并被续跑
   - ⑦ 提升/回滚断言（模拟生产）：`tools release` → `build_id` 更新 + 健康 → `--rollback` → 回到上一版且健康
   - ⑧ 收尾：打印 `{"pass":bool,"steps":[...],"prod_touched":false}`；**任何一步触碰真实 8413/`~/.rick` 视为失败**
2. `wiki/self-evolve.md`（运维手册）：
   - 心智模型：prod（8413 / `~/.rick` / 生产仓库工作树 + `bin/rick`）vs dev（8414 / dev HOME / dev worktree / `releases`）谁持有什么
   - 日常命令：`dev-web init|build|up|restart|status|down`、`tools release [--yes|--rollback|--dry-run]`
   - 恢复语义：平台自动回来、**会话/任务挂起待人工一键恢复**（为什么不做自动恢复：实测 pi 不修复悬挂 toolCall → 重复副作用；配额耗尽不报错 → 静默空转）
   - 故障排查：dev 起不来 / 端口占用 / 指纹不一致 / 提升后回滚 / 残留 worker 收尸
   - 已知边界（诚实记录）：`internal/handler/doing.go` 的 gates `helper.py` 路径仍硬编码 `UserHomeDir()`（不尊重 `RICK_PI_AGENT_DIR`）；PWA service worker 因 `precache` 缺 `index.html` 而注册失败（`non-precached-url`，独立议题）；`web/dist` 已入库，dev 树自带 `.rick` 快照可能陈旧；无跨进程锁（本次用 flock 覆盖状态目录，workspace 级写冲突仍靠纪律）
3. `.rick/jobs/job_36/plan/sim-report.md`：本增量的仿真/验收报告（复现命令、实测输出、与设计树 KR 的逐条对照表）。

# 测试方法
bash scripts/self-evolve-e2e.sh（应输出 pass=true 且 prod_touched=false）
python3 .rick/jobs/job_36/plan/gates/gate11.py
curl -s --noproxy '*' http://127.0.0.1:8413/api/health（**生产仍健康**，回归断言）

# 上下文提示
- 复用 gate7-gate10 的断言思路，避免重复实现；本 task 的价值在**串成一条可重放链路 + 文档**。
- 文档写在 dev 树 `wiki/`（若仓库无 `wiki/` 目录则新建；既有 gate6 引用过 `wiki/web-ui.md`，风格对齐）。
- 报告需逐条对照设计树 KR1-KR4（隔离/闭环/提升/挂起恢复）给出「证据 → 结论」。
