# 依赖关系
task1, task2

# 任务名称
实现 rick tools 子命令框架和 plan_check

# 任务目标
实现 `rick tools` 父命令及 `plan_check` 子命令。`rick tools` 是 Rick 的元技能系统，**主要使用者是 AI agent**：learning 阶段启动的 AI agent 通过读取 `rick tools --help` 了解所有可用工具，自主决定调用哪些命令完成工作。因此 help 文本必须对 AI 友好——清晰描述每个命令的用途、参数格式和输出含义。

plan_check 对 plan 阶段的输出进行结构性校验，校验失败时自动调用 `claude -p` 非交互模式修复，最多重试 3 次。

# 关键结果
1. 完成 internal/cmd/tools.go：tools 父命令注册，在 root.go 中添加 `rootCmd.AddCommand(NewToolsCmd())`

   **AI 友好的 help 设计要求**：
   - `rick tools --help` 输出必须包含每个子命令的一句话用途描述
   - 每个子命令的 `--help` 必须包含：用途、参数说明、输出格式、退出码含义
   - 示例（plan_check 的 help）：
     ```
     Usage: rick tools plan_check <job_id>

     Check the plan directory structure for a job to ensure it is valid for execution.

     Arguments:
       job_id    Job identifier (e.g. job_1)

     Checks performed:
       - plan/ directory exists
       - at least one task*.md file present
       - each task has required sections: 依赖关系, 任务名称, 任务目标, 关键结果, 测试方法
       - all dependency references exist
       - no circular dependencies

     Output:
       ✅ plan check passed: N tasks, dependencies valid
       ❌ plan check failed: <error description>

     Exit codes:
       0  all checks passed
       1  one or more checks failed
     ```
2. 完成 internal/cmd/tools_plan_check.go：plan_check 子命令，检查以下 6 条规则：
   - .rick/jobs/job_N/plan/ 目录存在
   - 至少有一个 task*.md 文件
   - 每个 task*.md 包含 5 个必要章节：`# 依赖关系`、`# 任务名称`、`# 任务目标`、`# 关键结果`、`# 测试方法`
   - 依赖引用合法（引用的 taskX 对应文件存在）
   - 无循环依赖（复用 executor.NewDAG 的循环检测逻辑）
   - task 文件在 plan/ 目录下（路径约束）
3. 完成自动修复机制：检查失败时将错误描述写入临时 prompt 文件（内容为：错误信息 + 要求修复的指令），调用 `claude --dangerously-skip-permissions <prompt_file>` 非交互执行修复，修复后重新检查，最多 3 次；提取为公共函数 `autoFix(claudePath, promptFile string) error` 供其他 check 命令复用
4. 完成 internal/parser/task.go 修改：将 `# 关键结果` 加入必需字段校验（ValidateTask 函数）
5. 完成 internal/prompt/templates/plan.md 修改：将路径规范从 `plan/tasks/task1.md` 修正为 `plan/task1.md`（与实现一致）

# 测试方法
1. 先运行 `go build -o bin/rick ./cmd/rick/` 构建最新二进制
2. 运行 `./bin/rick tools plan_check job_1`，验证对已有 job_1 输出 `✅ plan check passed: 9 tasks, dependencies valid`
3. 创建测试目录 `/tmp/test_job/plan/`，放入一个缺少 `# 关键结果` 章节的 task.md，运行 `./bin/rick tools plan_check`，验证输出包含具体错误信息
4. 创建一个有循环依赖的 task 集合（task1 依赖 task2，task2 依赖 task1），运行检查，验证检测到循环依赖
5. 创建一个依赖不存在的 task（task1 依赖 task99，但 task99.md 不存在），验证检测到悬空依赖
6. 运行 `./bin/rick tools --help`，验证显示 plan_check 子命令
7. 运行 `go test ./internal/cmd/... ./internal/parser/...`，验证测试通过
