APPROVED: true

# Job job_36 执行总结（rick web ui · 本会话段：验收期修复 + 自进化闭环）

## 执行概述

**项目目标**: 交付 rick 专用 Web UI（多工作区、pi rpc 会话、SSE 事件流、R&M 星空 UI），并在本会话段完成「rick 自进化」闭环：在 web 里安全地改进 rick 自身并原子生效到生产。
**实际完成**: 本会话段 14 个 task（task16-task29）全部通过（含 3 次返工修复），4 道新门禁（gate7-gate14 + gate12 重写）全绿，3 次真实 release（含 1 次冲突中止→AI 修复→重跑成功）。
**整体评价**: ⭐⭐⭐⭐⭐（交付完整；但发生 1 次约 10 分钟的生产中断事故，教训已固化）

## 本会话段的关键交付

1. **验收期大批修复**（35+ 项统一 commit 起）：流式闪烁三连（eid 稳定化/envSeq/openBlockStart）、长会话卡顿根治（无限 SVG 动画 → 主线程样式重算）、SSE 断线无缝续传（cursor 持久化 + 回合边界回退）、服务端权威 busy、resume 409 根因（registry 不更新 + 幂等）、消息乱序（时序锚点 anchorSeq）、easy 生命周期、doing 进度可见化、job 任务名重命名、CLI 会话导入（ImportSession）、本地 md 右侧阅读器 + 链接重定向语义、星空美术（液态传送门/紫星球/HUMAN 徽标）。
2. **自进化增量 1（隔离+提升+挂起，task16-22）**：`--state-dir` + flock singleton（修现存生产缺陷：web.pid 丢失时第二实例会误杀生产会话）+ 隔离守卫；`rick tools dev-web`（worktree/独立 HOME/pi 沙盒/构建指纹 health 校验闭环）；`build_id` 贯穿 health/config/SSE；**suspend 挂起语义**（重启后 active→suspended 而非 error，UI 一键人工恢复，绝不自动续跑）；`rick tools release`（版本目录+current 原子换链+`.last` 回滚点+重启+build_id 校验+`--dry-run/--rollback/--detach`）。
3. **自进化增量 2（rick-rsi-loop，task23-28）**：把「改进 rick」固化为 `.rick/loops/rick-rsi-loop.md`（五要素+状态机 S0-S7+产出评估 6 项）；`rick tools loops_check`（挂载仓库里沉睡的格式校验器）；`rick tools rsi_check`（产出评估机器契约：6 项证据 + 逐项中文指引 + `--init` 骨架带 TODO 防自欺 + prod-health 实时证伪）；`release --merge-source`（源码合并，**冲突即 abort 回滚干净并报错**，交 AI 修复）。
4. **去特殊化修订（task29，human 推翻原设计）**：删除 `rsi` 会话类型/`rick rsi` CLI/前端入口/工作区特殊校验 —— **RSI 只是一个普通 loop**，由标准机制（easy/plan 提示词的「可用的项目 Loops」目录，LoadLoopsContext 注入 name+trigger）发现加载；rick-dev 是普通工作区。
5. **3 次真实 release**：第一次冲突中止（安全网真实生效）→ AI 分析确认 dev 版本为 prod 严格超集 → 取 --theirs 提交 → 重跑成功；第二次提升成功但发生中断事故；第三次（去特殊化）用 `--detach` 全程无事故。

## 问题与教训（本会话段 6 条，全部已固化到 debug/ 与门禁）

### 问题1: 隔离守卫在「换 HOME 形态」下完全没装（F1，高危）
**根本原因**: 守卫启用条件写成 `IsDevStateDir(resolved)`（= 状态目录 ≠ 当前 HOME 的 .rick），而 `dev-web` 的真实启动形态是**换 HOME** → 条件恒 false。gate7 只按「实现者以为的形态」（--state-dir）构造，E2E 用真实形态跑才暴露（实测注册生产工作区 HTTP 201）。
**解决方案**: `DevModeInfo()` 按「状态目录是否等于**真实用户**的生产状态目录」判定（两种形态都算 dev）；`RICK_PI_AGENT_DIR` 强制仅 state-dir-only 形态。gate7 补第 ⑨ 条断言永久钉死。
**经验教训**: **门禁必须按产品真实调用路径构造**（真实命令、真实启动形态、真实宿主形态），不能按实现者以为的分支。

### 问题2: `release --dry-run` 被自保检查误拦（F2）+ 把构建产物写进生产树（F3）
**根本原因**: F2=自保检查跑在 dry-run 分支之前（顺序错）；F3=dry-run 复用真实提升的构建路径，产物落 `<prod>/bin/releases/`，与「不动生产」承诺不符。两者只在「被生产托管的会话里跑 dry-run」这一真实场景出现。
**解决方案**: dry-run 绕过自保（警告保留）+ 构建落 `os.MkdirTemp` 暂存并清理；**纵深防御**：`Promote()` 拒收 Staged 产物。
**经验教训**: 「安全演练命令」必须被门禁**显式断言为无副作用**；拦截逻辑的判定顺序也是契约。

### 问题3: `dev-web` 的 dev 树解析跟随 cwd（F4）
**根本原因**: `DefaultLayout` 用 `dirname(dirname(cwd))/rick-dev` 推导 —— 从 dev 树内执行（最自然的操作）会解析到不存在的 `<祖父>/rick-dev`。
**解决方案**: 解析顺序改为 显式 flag/env → **内容探测 worktree**（`.git` 是文件 + `cmd/rick` + `web/package.json`）→ 回退；`home=<tree>-home` 跟随；树校验前置于 EnsureToken（否则 mkdir 报错掩盖真因）。
**经验教训**: 命令默认参数解析不能依赖 cwd 的偶然形态——要么显式，要么按内容探测。

### 问题4: ⚠️ 生产中断约 10 分钟（release 未用 --detach）
**根本原因**: 执行 release 的 shell 被中断 → release 子进程被连带杀死，恰好停在「已停旧生产、未起新生产」之间。我事先判断「我不在生产托管下所以不需要 --detach」是错的——**调用方本身可能被中断**（工具超时/会话切换）。
**解决方案**: 手动重跑 `~/.rick/start-web.sh` 立即恢复（换链已原子完成）；把「release 永远用 --detach」写进 wiki 硬规则 + debug 复盘。
**经验教训**: 任何「先破坏后重建」的长操作必须脱离调用方进程树（setsid/--detach），不论调用方是谁。

### 问题5: 大会话 resume 被「超大单行 stdout 事件」判死（本会话段早期根因）
**根本原因**: `bufio.Scanner` 遇到超过上限的单行事件**永久停摆**（ErrTooLong 不可恢复）→ 上层把「只是文件大」的会话判成 worker lost；且服务退出不收 worker 留下孤儿 pi 占住会话文件。
**解决方案**: 改用 `bufio.Reader` 韧性读取（64MB 上限、**超限只丢该行继续**、EOF 带数据时交出行）；`Serve` 在 HTTP drain 前调用 `ShutdownWorkers()`（SIGTERM→宽限→SIGKILL）。
**经验教训**: 长流式输入的解析必须「单条损坏不拖垮整条流」；进程退出必须显式收子进程（自成进程组的不会被父进程连带杀）。

### 问题6: RSI 设计推翻 —— 会话类型绑定是过度设计（human 裁决）
**根本原因**: 我把「必须使用 loop」实现为专属会话类型+loop 全文注入+工作区硬校验 —— 发明了 rick 里不存在的新概念，而 rick 已有标准 loop 机制（Loops 目录 name+trigger 注入，agent 按触发条件加载）。
**解决方案**: 全部删除；RSI 只交付 `.rick/loops/rick-rsi-loop.md`；「必须在 dev 工作区开发」保留为 loop 纪律（loop 是制度载体，这正是它的职责）；gate12/E2E 重写为断言标准机制。
**经验教训**: **优先用系统已有的制度载体**，不要为新需求发明平行机制；「必须」如果已有标准达成路径（目录+trigger），就不需要代码强制。

## 战略决策（human 裁决，2026-09-22，已入 domain）

1. **后续迭代全部在 rick web 页面完成**（会话/学习/dream/ctrl 都走 web）；CLI 各命令与 web 保持同能力。
2. **TUI 跟随 pi 社区迭代，不作为 rick 的主要迭代方向**。
3. 自进化的运行形态：普通会话（easy/plan）+ `.rick/loops/rick-rsi-loop.md` + `rick tools {dev-web, release --merge-source, rsi_check, loops_check}`；交付期允许会话中断但必须可恢复（挂起→人工一键恢复，绝不自动续跑）。

## 数据

- task: 29/29 success（本会话段 task16-29 共 14 个，含 3 个返工：task16 F1、task17 F4、task21 F2/F3）
- 门禁: gate7-14 全绿（gate12 按去特殊化重写后仍绿）；两份 E2E（self-evolve 29 断言、rsi-loop 41 断言）可重放且 `prod_touched=false`
- release: 3 次（41726a8 两连 → 59ac918），版本链 `bin/releases/<ver>/{rick,dist}` + current 软链 + `.last` 回滚点
- 生产终态: `build_id=59ac918-260922143616`，8 工作区（含 rick-dev），`type=rsi` → 400，easy 提示词含 rick-rsi-loop 目录条目
