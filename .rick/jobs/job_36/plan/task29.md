# 依赖关系
task24

# 写域
internal/web/registry.go
internal/web/sessions.go
internal/web/sessions_test.go
internal/prompt/rsi_prompt.go            （删除）
internal/prompt/rsi_prompt_test.go       （删除）
internal/prompt/easy_prompt_test.go
internal/cmd/rsi.go                      （删除）
internal/cmd/root.go
internal/cmd/rsi_test.go                 （若存在，删除）
web/src/types.ts
web/src/lib/rsi.ts                       （删除）
web/src/components/sessions/NewSessionModal.tsx
web/src/components/layout/SessionBadge.tsx
web/src/components/sessions/SessionSettingsDialog.tsx
.rick/loops/rick-rsi-loop.md
wiki/self-evolve.md
.rick/jobs/job_36/plan/gates/gate12.py
.rick/jobs/job_36/plan/gates/gate14.py
scripts/rsi-loop-e2e.sh

# 任务目标
**RSI 去特殊化（human 裁决 2026-09-22）**：RSI 不是特殊会话类型，rick-dev 不是特殊工作区。**RSI 只交付为一个 loop**，走 rick 标准流程（easy/plan 会话的 prompt 注入「可用的项目 Loops」目录 → agent 按触发条件加载 loop → 按 loop 使用 rick 提供的工具自改进）。删除为此发明的所有特殊机制。

# 关键结果
1. **后端删除 rsi 会话类型**：
   - `internal/web/registry.go`：删 `SessionTypeRSI` 常量与校验分支（`type=rsi` 的创建请求应回落到既有的「未知类型」错误路径，返回 400/invalid_params）。
   - `internal/web/sessions.go`：删 `prepareInteractive` 的 rsi 分支及一切 `SessionTypeRSI` 引用（grep 清零）。
   - 删除 `internal/prompt/rsi_prompt.go`（含 `ValidateRSIWorkspace`/`RSIProdRepoRoot`/`EnsureRSIDirs`/`EnsureRSISessionID`/`BuildRSIPrompt`）与 `rsi_prompt_test.go`。
   - 删除 `internal/cmd/rsi.go`（`rick rsi` 子命令）及 `root.go` 里的注册；相关测试删除。
2. **前端删除 RSI 入口**：
   - `web/src/types.ts`：`SessionType` 去掉 `"rsi"`。
   - 删 `web/src/lib/rsi.ts`；`NewSessionModal.tsx` 去掉 rsi 类型与相关提示/校验分支；`SessionBadge.tsx`/`SessionSettingsDialog.tsx` 去掉 RSI 徽标与文案。
3. **loop 正文修订**（`.rick/loops/rick-rsi-loop.md`，保留五要素与 frontmatter）：
   - 「依赖准备」改为标准加载语义：本 loop 由 rick 标准机制发现（easy/plan 会话提示词中的「可用的项目 Loops」目录，按 trigger 匹配「修改 rick 自身」类任务）；**不存在 RSI 专属会话类型**，在任意承载会话（通常是 rick 源码工作区上的 easy 会话）中加载本文件即可。
   - 硬约束保留为**loop 纪律**（非代码强制）：改动应在 dev 工作树进行（`rick tools dev-web init` 产出），禁止直接编辑生产仓库工作树；这些是 loop 对 agent 的要求。
   - 状态机 S0-S7、工具表、产出评估（6 项，对应 `rick tools rsi_check`）、停止标准全部保留（`rsi_check`/`dev-web`/`release --merge-source` 等工具不变）。
4. **标准加载路径验证（新增测试）**：`internal/prompt/easy_prompt_test.go` 增加断言——在含 `.rick/loops/rick-rsi-loop.md` 的临时工作区上构建 easy 提示词，`LoadLoopsContext` 产出的目录里**必须出现 `rick-rsi-loop` 及其 trigger**（证明标准流程能发现该 loop，这是本设计的成立条件）。
5. **门禁重写（gate12.py）**——断言标准机制而非特殊类型：
   - ① `rick tools loops_check --dir .rick` pass（loop 载体合规，含 rick-rsi-loop）；
   - ② **标准发现**：临时 rick 源码树（含 `.rick/loops/rick-rsi-loop.md`）上 `POST /api/sessions {type:"easy", requirement:"改进 rick 自身…"}` 创建会话 → `GET /api/sessions/{id}/prompt` 的 easy 提示词**含 `rick-rsi-loop` 目录条目**（即 `LoadLoopsContext` 注入）；
   - ③ **类型已删**：`POST /api/sessions {type:"rsi"}` → 400（未知类型），且 `rick rsi --help` 失败（子命令不存在）；
   - ④ 前端：`web/src/types.ts` 不含 `"rsi"`；tsc + build 通过；
   - ⑤ 单测 + go build + vet 全绿；
   - ⑥ prod 只读回归（指纹 + 健康），与其它门禁一致。
6. **E2E 修订（scripts/rsi-loop-e2e.sh + gate14.py）**：把「创建 rsi 会话并断言 loop 注入」的段落改为「创建 **easy** 会话并断言 prompt 目录含 rick-rsi-loop」+「type=rsi 返回 400」；`rsi_check` 三段式与 `merge-source` 冲突中止段落**保持不变**；末行仍输出 `{pass,steps,prod_touched:false}`。
7. **wiki 修订（wiki/self-evolve.md）**：RSI 使用方式改为——在任意 rick 源码工作区（如 rick-dev，普通工作区）上起**普通会话**（easy/plan），任务是「改进 rick」时 agent 会按标准 Loops 目录加载 `rick-rsi-loop`；删除「RSI 会话类型/特殊校验」相关描述；保留工具命令、人类确认点、`--detach` 硬规则、故障排查与已知边界。

# 测试方法
go test ./internal/web/ ./internal/prompt/ ./internal/cmd/ -timeout 900s
go build ./... && go vet ./internal/web/ ./internal/prompt/ ./internal/cmd/
npx --prefix web tsc --noEmit -p web && npm --prefix web run build
python3 .rick/jobs/job_36/plan/gates/gate12.py
bash scripts/rsi-loop-e2e.sh && python3 .rick/jobs/job_36/plan/gates/gate14.py

# 上下文提示
- **保留不动的工具**（与本次去特殊化无关）：`rick tools dev-web`、`rick tools release --merge-source`、`rick tools rsi_check`、`rick tools loops_check`、挂起语义与一键恢复。
- 全仓 grep `SessionTypeRSI|"rsi"|rick rsi` 清零（测试与前端类型在内；`rsi_check`/`rsi-loop-e2e`/`rick-rsi-loop` 这些**工具与 loop 名**保留）。
- easy 提示词的 loops 目录注入点在 `internal/prompt/easy_prompt.go`（`LoadLoopsContext`），这是标准机制，不要改它。
