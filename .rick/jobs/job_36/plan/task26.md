# 依赖关系
task24

# 写域
internal/cmd/tools_rsi_check.go
internal/cmd/tools.go
internal/cmd/tools_rsi_check_test.go

# 任务目标
把 `rick-rsi-loop` 的「产出评估」从文档变成**机器契约**：`rick tools rsi_check` 校验一次自进化迭代是否留下了 loop 要求的全部证据（否则 loop 只是愿望）。

# 关键结果
1. `internal/cmd/tools_rsi_check.go`：`rick tools rsi_check [--job job_N] [--json]`，默认取当前 job（`--job` 或 `<cwd>/.rick/jobs/` 下最近修改的 job）。校验项（每项给「通过/失败 + 证据路径」）：
   - **dev 实例记录**：存在 `doing/rsi/dev-iterations.md`（或 `.json`），且其中至少记录一次 `build_id`（形如 `rick.dev.<sha7>-<ts>` 或 `build_id=<sha7>-<ts>`）；
   - **门禁全绿**：存在 `doing/rsi/gates.md`，且每行门禁记录都是 pass（形如 `GATE gate7 pass=true`；有 `pass=false` 即失败）；
   - **人类确认痕迹**：`doing/rsi/approval.md` 存在且含确认标记（例如 `APPROVED by=human at=<时间>`）——**这是最关键的校验**（loop 要求人类确认才 release）；
   - **release 记录**：`doing/rsi/release.md` 含版本号与回滚点（`version=<sha7>-<ts>`、`rollback_point=` 或 `released=yes`）；
   - **挂起-恢复记录**：`doing/rsi/resume.md`（重启后挂起清单与人工恢复结果）。
   - 输出：`{"pass":bool,"job":"job_N","checks":[{"name","pass","evidence","detail"}],"errors":[]}`，退出码 0/1；**缺失项要给出「下一步该做什么」的中文指引**（例如「先跑 `rick tools release --dry-run` 并把计划落盘到 doing/rsi/release.md」）。
2. `internal/cmd/tools.go`：注册子命令 + help 文本。
3. 提供模板生成：`--init` 时按 loop 产出评估表创建 `doing/rsi/{dev-iterations.md,gates.md,approval.md,release.md,resume.md}` 骨架（让 AI 有明确落盘位置；模板头部写明每项的通过标准）。
4. 测试 `tools_rsi_check_test.go`（`t.TempDir()`）：全合规 → pass；逐项缺失/含 `pass=false`/approval 无标记 → 各自失败且 detail 指向正确文件；`--init` 幂等（已存在不覆盖）。

# 测试方法
go test ./internal/cmd/ -timeout 600s
go build ./... && go vet ./internal/cmd/
go run ./cmd/rick tools rsi_check --job job_36 --json   # 当前应 fail（尚无 doing/rsi/*），并给出中文指引
python3 .rick/jobs/job_36/plan/gates/gate13.py

# 上下文提示
- 参照 `tools_learning_check.go` / `tools_dream_check.go`（既有 checker 风格：返回 errors 列表 + JSON 输出）。
- loop 侧对应段落：`.rick/loops/rick-rsi-loop.md` 的「产出评估」表（task23 写），两边必须逐项一致 —— 若发现不一致，**改 loop 而不是放松 checker**。
