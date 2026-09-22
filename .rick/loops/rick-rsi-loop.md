---
name: rick-rsi-loop
trigger: "当需要修改 rick 自身（源码 cmd/ internal/ web/、prompt 模板、或 .rick 知识库/loop）并让它生效到生产时触发——「让 rick 自己改进自己」的任务加载本 loop"
scope: "全局（任意承载会话：plan / doing / easy 均可加载）"
---

# Loop: rick 自进化（RSI，Recursive Self-Improvement）

把「改进 rick 自身」的一次迭代**安全**送达生产：改动在 **dev 工作区**完成、通过层门禁、经**人类确认**后由 `rick tools release` 原子提升并重启，生产健康且 `build_id` 匹配本次构建。

- 本 loop 是一个**普通的 rick loop**（与 loops/ 目录其它成员同构，无专属会话类型）：承载会话的提示词里有「可用的项目 Loops」目录（name+trigger），当任务属于「修改 rick 自身」时，**先完整读取本文件**再动手。
- 本 loop 的兑现方式：产出评估表由 `rick tools rsi_check` **机器校验**——不写证据 = 本次迭代未完成。
- **本 loop 也是自己的改进对象**：修改 `rick-rsi-loop` 自身（或任何 loops/skills/domain 知识）同样触发并走完本 loop 的 S0→S7——递归自指，改制度也按制度来。

## 依赖准备（硬约束，缺失则报错停止）

1. **必须运行在 dev 工作区**（`rick tools dev-web init` 产出的 worktree，路径形如 `<生产仓库祖父目录>/rick-dev`）：
   - `git -C <ws> rev-parse --abbrev-ref HEAD` 以 `dev/` 开头（**不得是 `main`**）
   - `<ws>` 含 `cmd/rick/main.go` 与 `internal/web/`（是 rick 源码树）
   - `<ws>/.rick/loops/rick-rsi-loop.md` 存在（本文件）
2. dev 实例就绪：
   - `rick tools dev-web init`（幂等）→ `rick tools dev-web status` 输出 `DEV_STATUS state=up ... fingerprint=ok`
   - 若未起：`rick tools dev-web up`
3. **禁止事项（硬约束）**：不得直接编辑**生产仓库工作树**；不得在生产树里 `go build -o bin/rick`；不得 `kill`/重启生产实例（8413）。生产侧唯一写入口是**人类确认后**的 `rick tools release`。

不满足任一条件 → **停止并报告**，不要在生产树里继续。
（若因「当前工作区不是 dev 工作区」：提示人类改在 **rick-dev 工作区**开会话承载本任务，
 或先 `rick tools dev-web init` 建好环境——不要试图就地绕过。）

## 目标（Goal）

一次自进化迭代的四件事全部达成（客观判据）：

| # | 达标条件 | 判据（可复制执行） |
|---|---|---|
| G1 | dev 侧改动完成且层门禁全绿 | `python3 .rick/jobs/<job>/plan/gates/gate{N}.py`（**当前迭代所属 job**）→ `"pass": true` |
| G2 | 演练可见且生产零写入 | `rick tools release --dry-run` → 含 `RELEASE_DRYRUN prod_untouched=true` |
| G3 | 人类确认留痕 | `doing/rsi/approval.md` 含 `APPROVED by=human at=<时间>` |
| G4 | 提升成功且生产为新构建 | `rick tools release --merge-source` → `RELEASE_MERGE merged=true` + `/api/health` 的 `build_id` == 本次 version |

- 自评命令：`rick tools rsi_check --job <job> --json` → `"pass": true`（缺证据即未完成）

## 上下文管理（Context Management）

- **保留**：本 job 的设计树（OKR/KR/层/判断节点与裁决）、改动文件清单、每层 gate 的 JSON 结论、release 版本号 + `.last` 回滚点、挂起-恢复记录
- **压缩**：每轮迭代只留一行「改了什么 → gate 结果」；构建/测试输出只留尾部（≤30 行）
- **遗忘**：dev 实例的中间构建产物细节、被回滚的试验性改动、与本迭代无关的历史 job 细节
- **落盘位置**（`rsi_check` 逐个校验，可用 `rick tools rsi_check --job <job> --init` 生成骨架）：
  `<ws>/.rick/jobs/<job>/doing/rsi/{dev-iterations.md,gates.md,approval.md,release.md,resume.md}`

## 可调用工具（Tool Access）

| 工具 / 命令 | 用途 | 约束 |
|---|---|---|
| `rick tools dev-web init` | 幂等准备隔离 dev 环境（worktree + dev HOME + pi 沙盒种子 + overlay 软链） | 只碰 dev 树与 dev HOME |
| `rick tools dev-web status` | 看 dev 实例状态 / 期望 vs 运行指纹 | 只读 |
| `rick tools dev-web restart` | 改**后端**后：重建 + 重启 dev 实例（构建指纹自动复核） | 只重启 dev（8414） |
| `npm run build`（web/，dev 树） | 改**前端**：产物落 `<dev>/web/dist`；dev 实例 overlay 是软链 → 刷新即生效，**无需重启** | 不得动 `~/.rick/web/dist`（那是生产覆盖层） |
| `python3 .rick/jobs/<job>/plan/gates/gate{N}.py` | 层门禁（内含「生产零触碰」断言；`<job>`=当前迭代所属 job，**勿写死历史 job**） | 必须全绿才下钻下一层 |
| `rick tools release --dry-run` | 演练：门禁 + 构建 + 打印计划；**不碰生产、不重启** | 产物落临时暂存，`prod_untouched=true` |
| **人类确认**（S4） | release 前的唯一放行条件 | 无确认**不得**进入 S5 |
| `rick tools release --merge-source` | 源码合并（dev 分支 → 生产 main）+ 二进制/dist 原子换链 + 重启 + `build_id` 校验 | 合并**冲突即中止**（工作树自动恢复干净），交 AI 修复后重跑 |
| `rick tools release --rollback` | 回滚**产物**到上一版（切链 + 重启） | 不含源码回滚（源码用 git 回滚）；版本链默认保留最近 3 版（`--keep N` 可调，current/.last 永不 GC） |
| `rick tools rsi_check --job <job>` | 校验本 loop 的产出评估表 | 失败时输出中文「下一步该做什么」 |
| `rick tools dev-web down` | 停 dev 实例（只杀确认属于该 dev HOME 的进程） | 不影响生产 |

- **权限边界**：所有编辑只发生在 dev 工作区；生产只经 `rick tools release` 被改动一次。

## 子 Agent 工作流（状态机）

每轮一个子 Agent；状态互斥推进，任一状态失败 → 修好后**重进该状态**（不跳步）：

```
S0 设计
   动作：grilling 出设计树（OKR → KR → 层 → pipeline → 判断节点），判断节点交人类裁决
   产出：<ws>/.rick/jobs/<job>/doing/grilling/design-tree.md（+ research 简报）
   出口：设计树每层达标（含 OKR 充分性自检），grilling_gate 通过

S1 隔离开发（在 dev 工作区）
   动作：rick tools rsi_check --job <job> --init（先建证据骨架，随手落盘）
         rick tools dev-web init → status（未起则 up）
         前端改动：npm run build（dev 树 web/）→ 刷新 dev UI（8414）验证
         后端改动：rick tools dev-web restart → 复核 build_id 变化（新构建真的在跑）
   产出：doing/rsi/dev-iterations.md（每轮记一行：build_id + 改了什么 + 验证结论）
   出口：改动可见/可验证，dev 侧不自欺（build_id 已换）

S2 层门禁（逐层）
   动作：python3 .rick/jobs/<job>/plan/gates/gate{N}.py（当前迭代所属 job，每层一个）
   产出：doing/rsi/gates.md（每行 `GATE gateN pass=true`）
   出口：全部门禁 pass=true；失败 → 回 S1 修（禁止带病下钻）
         ※ 每轮重试在 dev-iterations.md 记一行（含失败摘要与轮次）；
           **同一 gate 连续 2 轮失败 → 触发停止标准**（进入失败退出，勿无限重试）

S3 演练（给人类看计划）
   动作：rick tools release --dry-run
   产出：doing/rsi/release.md（贴 target_bin / target_dist / version / rollback_point / prod_untouched）
   出口：prod_untouched=true

S4 人类确认（唯一放行点）
   动作：把 S3 的计划 + 影响面（生产将短暂重启：优雅退出预算 ≤12s、实测健康就绪 ~0.2s；
         所有在跑会话变「挂起」，需人工一键恢复；不做自动续跑）呈给人类，取得**明确批准**
   产出：doing/rsi/approval.md → `APPROVED by=human at=<RFC3339 时间>`
   出口：有批准。未批准 → 停止（dev 产物保留，生产不变）

S5 提升
   动作：rick tools release --merge-source --detach
         ⚠️ **--detach 必须带**（setsid 脱离调用方进程树）：执行 shell 本身可能被中断
         （实测事故：release 被连带杀死，生产停在「已停未起」约 10 分钟）
         · 合并冲突 → 命令**中止报错**并把生产工作树恢复到干净状态 → AI 修复：
           在**生产仓库**执行 `git merge --no-ff --no-commit <dev分支>` →
           冲突文件按语义取舍（若 dev 侧是超集：`git checkout --theirs -- <file>`）→
           `git commit` 完成合并 → **重跑 S5**
           ⚠️ 这是生产树上**唯一允许**的手动 git 操作：只做「完成 release 发起的合并 +
           版本取舍」，**不得借机修改代码内容**；修复后 `git status --porcelain` 必须为空
         · 无冲突 → 换链 + 前端投放 + 重启 + build_id 校验（校验失败自动回滚）
   产出：doing/rsi/release.md 追加 `RELEASE_MERGE merged=true version=<sha7>-<ts> rollback_point=<path>`
   出口：生产 `/api/health` → `status=ok` 且 `build_id` == version

S6 恢复（重启后）
   动作：浏览器自动重连（SSE 游标自愈）；所有重启前在跑的会话/job 变为 **suspended（挂起）**
         人类在 UI 上逐条点「恢复继续」；doing/dream 由人类点「继续执行」后才归一化
         running→pending 并续跑剩余 task
   产出：doing/rsi/resume.md（挂起清单 + 人工恢复结果）
   出口：需要继续的会话都已恢复（或人类明确选择不恢复）

S7 留痕校验
   动作：在 **dev 工作区**执行 `rick tools rsi_check --job <job> --json`（按 cwd 解析 job 与证据目录）
   出口：pass=true → 迭代成功退出
```

**关键纪律（写进流程、不可协商）**

1. **不自动续跑**被打断的会话：实测 pi 不修复悬挂 toolCall（末条 assistant `stopReason=toolUse` 且无对应 toolResult），自动重放有**重复副作用**风险；且配额耗尽**不报错**（返回普通文本 + usage 全 0），自动续跑会静默空转。→ 一律挂起 + 人工一键恢复。
2. **只在 dev 树试错**；生产只在 S5 被改一次，且改动前必有 S4 的人类确认与 S3 的演练记录。
3. **失败即回到上一个可用版本**：任何一步失败都不得留下「换了二进制但没重启」「生产树里有未提交/冲突中的合并」这类中间态。
4. **门禁是唯一进展判据**：不以「代码看起来对」为进展，只以 gate 的 `pass=true` 为进展。

## 产出评估（Output Evaluation）

- 评估类型：**客观（机器校验）**，由 `rick tools rsi_check --job <job> --json` 执行
- 评估项与证据（缺任一 → 本次迭代未完成）：

| 检查项 | 证据文件 | 通过标准 |
|---|---|---|
| dev 迭代记录 | `doing/rsi/dev-iterations.md` | 至少一条构建指纹（`rick.dev.<sha7>-<ts>` 或 `build_id=<sha7>-<ts>`） |
| 门禁全绿 | `doing/rsi/gates.md` | 每行 `GATE ... pass=true`，**不得**出现 `pass=false` |
| **人类确认** | `doing/rsi/approval.md` | 含 `APPROVED by=human` 与时间戳（本 loop 最关键的一项） |
| 提升记录 | `doing/rsi/release.md` | 含 `version=<sha7>-<ts>` 与 `rollback_point=` |
| 挂起-恢复 | `doing/rsi/resume.md` | 记录挂起清单与人工恢复结果 |
| 生产健康 | 实时探测 `GET <prod>/api/health`（默认 `127.0.0.1:8413`，端口变化用 `--prod-url` / `RICK_RSI_PROD_URL` 覆盖） | `status=ok` 且 `build_id` == 本次 version |

- 无进展判断：连续 2 轮门禁无变化，或同一 gate 反复失败 → 停止（见下）

## 停止标准（Termination Condition）

- **成功退出**：`rsi_check --job <job>` pass=true，且生产 `/api/health` 的 `build_id` == 本次 release 版本
- **失败退出**：任一情形即停止并写 `<ws>/.rick/jobs/<job>/doing/debug/rsi-stuck-<n>.md`（含现场：失败的 gate 输出、冲突文件清单、健康探测结果）：
  · 同一层门禁连续 2 轮无进展
  · `--merge-source` 冲突经 AI 修复后仍冲突
  · release 健康/`build_id` 校验失败且自动回滚后生产仍异常
- **人类否决**：S4 未获批准 → 停止（dev 侧产物保留，生产**不变**）
- **优雅退出**：保留 dev 工作区与 `bin/releases/.last` 回滚点；生产保持上一个可用版本；不得留中间态
