# worker 任务简报：job_49 S1 隔离开发（迭代 1）

你是 job_49 的 doing worker（迭代 1）。任务：在 **dev 工作区**实现 release 链路 P0 根因修复（4 项）+ P1-2 源码部分（1 项）+ 单测。

## 上下文（开工前必读，按序）

1. `/Users/logic/workdir/AI_CODING/rick/wiki/handover-2026-10-30-web-start-macos.md` —— 交接单：根因链 + 每项验收标准
2. `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/grilling/research-L1.md` —— 代码事实简报（file:line 锚点已核实到 HEAD）
3. `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/grilling/design-tree.md` —— 设计树（叶子层四维度：每项的落点/验收/环境）
4. `/Users/logic/workdir/AI_CODING/rick/.rick/domain/bugs.md` —— 已知坑（重点：TestWebSingleton 读生产状态假失败、release gate env 污染）

## 工作区与写域（硬约束，越界即失败）

- **代码只写 dev 工作区** `/Users/logic/workdir/rick-dev`（分支 dev/self-evolve，与 main 的 02d144d 差一个 release 合并提交）：
  - `internal/env/release/release.go`（P0-1/2/3/4 全在这）
  - `internal/env/release/release_test.go`（新增用例）
  - `internal/env/relocate/import.go`（P1-2）
- **证据追加写**（main 树，绝对路径）：`/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/rsi/dev-iterations.md`（每轮一行：build_id + 改了什么 + 验证结论）
- **禁止**：碰 main 树（/Users/logic/workdir/AI_CODING/rick）的任何源码；碰 6137（生产，当前已死，别拉起）/8414（dev 实例，重启它用命令而非 kill）；任何 git 操作（add/commit 由 parent 在门禁后统一做）；改 wiki/ 或其他文档

## 实现清单（human 已裁决，按序做，每项先读 research-L1 对应条目）

1. **P0-1 promote 可执行性预校验**：Promote 切链**之前**新增 verifyRunnable（读 magic 判平台——复用/内联 import.go:812-823 的 wrongPlatformBinary 逻辑（它只护 rg，未导出；导出或在 release 包实现均可）+ `<bin> --version` 带超时冒烟（参考 smokeCheck release.go:556-563 的写法））。失败 → 返回 stage 语义错误（build/gate 阶段定性），**current 不变、零进程被杀**。
2. **P0-2 rollback 校验**：Rollback()（release.go:1000-1031）切链前同样校验 .last 目标；不可执行 → 拒绝切链 + 保留现场 + 错误点名「回滚点不可执行」+ 提示可用健康版本。
3. **P0-3 stopProd 认领补强（B 方案，human 裁决）**：stopProd（release.go:795-818）/ownedByProd（release.go:1276-1306）——新增兜底：pid **持有 `<stateDir>/web.lock`** 或**监听生产端口**（lsof）且进程是 rick web（comm/exe 名义）→ 即使三断言不匹配也认领停止（本案 11135 裸构建 build_id=dev 漏杀场景）；**非 rick 进程仍宁可不杀**；Cwd/Exe 取不到时不再静默 false——记录进 skipped 诊断信息（与 P0-4 联动）。
4. **P0-4 health 诊断增强**：waitHealthy（release.go:896-929）+ Restart 错误组装（release.go:985 附近）——失败信息含 old_pid / old_build_id / 端口占用者（lsof -nP -iTCP:<port>）/ 新进程日志尾部 20 行；**区分两种文案**：「旧进程占端口（build_id=X）」vs「新进程没起来」。
5. **P1-2 源码部分**：import.go mergeState（约 L316）对 `<StateDir>/bin/rick` 写入前加 wrongPlatformBinary 校验——错平台不覆盖（现状只暂存 rick-import-<goos>-<goarch>，F6：~/.rick/bin/rick 至今是 ELF）。

## 测试方法（TDD：先红后绿；自测代码随交付，不落盘共享测试目录）

新增单测全部放 `internal/env/release/release_test.go`（对齐既有 fakeBuildBinary/newFakeProd 风格，research-L1 §5 已确认可行）：
- TestPromoteRejectsNonExecutableTarget：构造 ELF fake 版本目录 → Promote 拒绝 + current 不变 + 无进程被杀
- TestRollbackRejectsNonExecutableTarget：.last 指向 ELF 版本 → rollback 拒绝 + current 不变 + 服务仍在跑
- stopProd 三场景：①cwd 不可读时回退（不误判 false，带诊断）②锁持有者认领（裸构建持锁 → 被停止）③非 rick 进程不误杀
- health 诊断：旧进程占端口 → 报错含 pid=X build_id= 字样
- import 拒绝 ELF：dump 树含 ELF rick → mergeState 不覆盖 ~/.rick/bin/rick 本体

**跑测试必须环境隔离**（domain/bugs.md 已知坑：TestWebSingleton 读生产状态假失败）：
```bash
cd /Users/logic/workdir/rick-dev && env -u RICK_STATE_DIR -u RICK_PI_AGENT_DIR -u PI_CODING_AGENT_DIR go test ./internal/env/... ./internal/env/relocate/... -count=1
```
全量回归同上但 `go test ./...`。

## 完成验证（S1 出口）

- `rick tools dev-web restart`（用 /Users/logic/workdir/AI_CODING/rick/bin/rick 这个 CLI 跑，它最新）→ 输出 DEV_UP build_id=新指纹 → curl http://127.0.0.1:8414/api/health 确认 build_id 变化（证明新代码真的在跑）
- ⚠️ 已知缺陷（勿修，记录即可）：`dev-web status` 报 state=down 是 status 的 Running 判定盲区（findDevPIDs 只认 cwd==dev 树的进程，而 dev 实例 cwd 是 rick-dev-home）——以 /api/health 的 build_id 为准
- 在 dev-iterations.md 追加一行（含新 build_id）

## 回执（完成时输出，不执行任何 git）

改动文件清单（限写域内）+ 每项实现要点（1-2 行）+ 自测结果摘要（测试数/通过数）+ 遇到的问题与处置 + 遗留问题。

## 失败处理

- 遇测试红/编译错/行为异常：先搜 `/Users/logic/workdir/AI_CODING/rick/.rick/domain/bugs.md` 有无精确解法；再声明 "I will use skill:debug-skill" 读 `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/prompts/skill_debug_skill.md` 按其流程走（debug 记录写 `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/debug/bug{N}-*.md`）
- 分析开工时声明 "I will use skill:sense."（Symptoms/Evidence/Next 三段分析现状）；实现声明 "I will use skill:tdd for implementation."
