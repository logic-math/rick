# 依赖关系
无

# 写域
.rick/loops/rick-rsi-loop.md
.rick/loops/README.md
internal/cmd/tools_loops_check.go
internal/cmd/tools.go
internal/cmd/tools_loops_check_test.go

# 任务目标
交付自进化的**制度载体**：`rick-rsi-loop`（rick 源码内的 loop），并把仓库已有的 loop 格式校验器挂成 `rick tools loops_check`（此前 `runLoopsAndSkillsCheck` 只存在于代码里、没有命令入口）。

# 关键结果
1. `.rick/loops/rick-rsi-loop.md`：**严格遵循 `.rick/loops/README.md` 的五要素格式** + frontmatter：
   - `name: rick-rsi-loop`；`trigger: "当需要修改 rick 自身（源码 cmd/ internal/ web/、或 .rick 知识库）并让它生效到生产时触发"`；`scope: "全局"`
   - 正文必须把「机制」写成可执行步骤（这是本 loop 的价值所在）：
     · 依赖准备：dev 环境就绪（`rick tools dev-web init`；dev 实例 8414 健康；`build_id` 可读）
     · 全局目标：改动在 **dev 工作区**完成并通过门禁，经**人类确认**后由 `rick tools release` 原子生效到生产
     · 上下文管理：读 `.rick/domain/*`（架构/坑位）+ 本 job 的 grilling 设计树；保留「改动清单 + 门禁结果 + release 版本 + 回滚点」
     · 子 Agent 工作流（状态机）：① 设计（grilling/设计树）→ ② 隔离开发（dev 树 + `dev-web` 迭代：前端 overlay 热更 / 后端 `dev-web restart` 复核指纹）→ ③ 层门禁 → ④ `rick tools release --dry-run` 给人看计划 → ⑤ **人类确认** → ⑥ `rick tools release --merge-source`（源码合并；**冲突即中止**，由 AI 修好重跑）→ ⑦ 重启后会话**挂起**，人工一键恢复 → ⑧ `rsi_check` 留痕
     · 产出评估：给出检查表（dev 实例 build_id 记录 / 门禁全绿 / 人类确认痕迹 / release 版本 + `.last` 回滚点 / 挂起-恢复记录），并声明由 `rick tools rsi_check` 机器校验
     · 停止标准：成功=release 后生产健康且 build_id 匹配；失败/优雅退出=gate 失败或人类否决（保留 dev 树与回滚点，不改生产）
   - **硬约束写进 loop**：RSI 会话必须在 **dev 工作区**运行；禁止直接编辑生产仓库工作树
2. `.rick/loops/README.md`：目录清单与实况对齐（现存条目漏了 `go-refactor-migration-loop`，一并补 + 加入 `rick-rsi-loop`），并加一句「改 rick 自身必须走 rick-rsi-loop」。
3. `internal/cmd/tools_loops_check.go`：把已有 `runLoopsAndSkillsCheck` 暴露为 `rick tools loops_check [--json]` —— 默认校验 `<cwd>/.rick/{loops,skills}`，支持 `--dir`；输出 `{"pass":bool,"errors":[]}`；退出码 0/1。
4. `internal/cmd/tools.go` + `root.go`：注册该子命令。
5. 测试：`tools_loops_check_test.go` —— 用 `t.TempDir()` 造 loops 目录（合规/缺 trigger/缺五要素小节/README 跳过/目录不存在），断言错误信息与退出码；并断言 `rick-rsi-loop.md` 本体合规（对真实文件跑一次校验）。

# 测试方法
go test ./internal/cmd/ -timeout 600s
go build ./... && go vet ./internal/cmd/
go run ./cmd/rick tools loops_check --dir .rick    # 必须 pass=true
python3 .rick/jobs/job_36/plan/gates/gate11.py

# 上下文提示
- loop 五要素小节名（校验器要求）：`## 目标` / `## 上下文管理` / `## 可调用工具` / `## 产出评估` / `## 停止标准`；frontmatter 必须含 `name` 与 `trigger`（`internal/cmd/tools_loops_skills_check.go` 的 `loopSections` 为准）。
- 参照 `go-refactor-migration-loop.md` / `tdd-red-green-refactor-loop.md` 的写法（含「可调用工具（Tool Access）」与带通过标准的检查表）。
