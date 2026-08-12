# Rick 命令规范

## rick doing（DIP 全链路）

- `doing.go` 是唯一 import `internal/agent/claudecode` 的地方（组合根）
- `runner.go` 和 `executor.go` 只依赖 `internal/agent` 接口，不 import claudecode
- `actpath.Generate(session, outputFile)` 在每个 task 的 Execute 完成后调用
- session 为 nil 时跳过 act-path 生成（nil guard），不 panic

## rick doing --dry-run

- 打印完整 doing prompt 内容到 stdout
- 不调用 Claude，不执行任何任务
- 展示**第一个非 success 状态的任务**（从 tasks.json 读取，不硬编码 task1）

## rick plan --job

- `--job <job_id>` 为**全局 flag**（定义在 root.go），plan.go 通过 `GetJobID()` 读取
- 不在 plan.go 中重复定义此 flag
- 指定 `--job` 时跳过 `NextJobID()`，直接复用已有 job 的 plan 目录
- plan 目录不存在时返回明确错误，不自动创建

## rick plan --dry-run

- 生成完整 plan prompt 并打印到 stdout（通过 `runPlanDryRun()` 函数）
- 不调用 Claude，不创建任何文件
- 输出包含所有注入内容：job_plan_dir、loops_context 等

## rick learning --dry-run

- 生成完整 learning prompt 并打印到 stdout（通过 `runLearningDryRun()` 函数）
- 不调用 Claude，不创建任何文件
- 输出包含：okr_content、task_md_content、debug 记录、act_path_content 等

## rick dream

- 自动扫描 `.rick/jobs/*/doing/tasks.json` 发现所有 tasks 均 "success" 的 jobs
- 对比 `.rick/dream/dream_run_*_log.md` 排除已处理 jobs，取最多 5 个待处理
- `--job_num <n>`：调整每次处理的 job 数量（默认 5）
- `--background`/`-p`：背景模式，使用 `--dangerously-skip-permissions` 非交互执行
- `--dry-run`：输出完整提示词，不调用 Claude

## rick ctrl

- `--job <job_id>` 为**必传参数**，无默认值
- 调用 `GenerateCtrlPromptFile(jobID, rickDir)` 生成 prompt，写入 `doing/prompts/ctrl_prompt.md`
- `callClaudeCodeCLI(cfg, promptFile)` 启动交互式 Claude 会话（与 plan/human-loop 共用同一函数）
- ctrl 与 doing 之间**仅通过文件通信**：reading tasks.json + raw_session_coding.log，writing tasks.json + plan/task\<N\>.md
- **变更约束**：只能修改 `doing/` 和 `plan/` 下的文件
- dry-run 输出完整 prompt（通过 `runCtrlDryRun()`），需指定 `--job` 否则报错退出

### ctrl 四种干预场景

| 场景 | 操作 |
|------|------|
| A：追加指令 | 在 `plan/task<N>.md` 末尾追加 `## 干预指令 (Intervention)` 章节 |
| B：重置 task | 将 status 改为 `"pending"`，清空 error 字段，更新 updated_at |
| C：查看轨迹 | 读取 act-path.md |
| D：查看原始日志 | 读取 raw_session_coding.log |

**注意**：若目标 task 正在运行（`running`），重置无效，需先 Ctrl+C 停止 doing。

## rick human-loop

- 命令：`rick human-loop <topic>`
- 通过 SENSE 方法论引导 5 阶段深度思考（v3.1，四文件架构），产出存入 `.rick/draft/rfc/` 目录（`GetRFCDir()` 返回 `.rick/draft/rfc`，不是旧 `.rick/RFC/`）
- **四文件 sub agent 模板**（sense_loop/think/research/exporter）通过 Go embed 编译进二进制，运行时写出到 `{{loop_dir}}/prompts/`（loop_dir = `draft/loops/loop_N`），并写 4 个对应 skill 文件到 `templates/skills/`
- 自动创建目录（MkdirAll，幂等）：`draft/rfc/`、`draft/concepts/`、`draft/human-learning/`、`draft/loops/loop_N/`（loop id 由 `NextLoopID()` 分配）
- dry-run 输出中 sub agent 路径为占位符格式（`<draft>/loops/loop_N/prompts/*.md`），不含真实路径
- 配置项：`.rick/config.json` 的 `human_loop` 嵌套字段（`max_retries` 默认 5、`sense_max_backflows` 默认 3、`think_top_n`、`think_min_assumptions` 默认 5、`research_source_weights`）
- 复用 `callClaudeCodeCLI`（plan.go 中定义，同包内共享，不重复声明）

### human-loop 验证

```bash
# 验证 dry-run 输出含关键词（不依赖特定 --phase/--keywords 参数）
./bin/rick human-loop --dry-run '测试主题' | grep "sense_loop"
```

## NDJSON 解析规范（internal/callcli 或 actpath）

Claude Code `--output-format stream-json` 输出的 NDJSON 格式：

- **必须加 `--verbose`**，否则报错退出
- `tool_use`/`tool_result` 嵌套在 `message.content[]` 内，**不在顶层**
- 非 JSON 行处理：`log.Printf("warn: skip non-json line %d: %s")` 后继续，不 panic
- 截断规范：Input/Output 截断 **300 字符**，FinalMessage 截断 **200 字符**，用 `[]rune` 处理 Unicode

```go
// 典型解析结构
type StreamMessage struct {
    Type    string  `json:"type"`
    Message *Msg    `json:"message,omitempty"`
}
type Msg struct {
    Content []ContentBlock `json:"content"`
}
type ContentBlock struct {
    Type  string `json:"type"`  // "tool_use" / "tool_result" / "text"
    Input json.RawMessage `json:"input,omitempty"`
}
```

## Dry-run 通用规范

`--dry-run` 标志必须输出**完整的 prompt 内容**（而非占位消息），便于调试和验证上下文注入效果。

**验证模板变量已替换**：
```bash
./bin/rick doing job_N --dry-run | grep -c '{{'  # 应为 0（无未替换变量）
```

## rick tools init-pi（配置目录隔离，job_33 起）

- 在 `~/.rick/pi/agent` 下引导 rick 托管的 pi 配置（settings.json），与用户 `~/.pi` 完全隔离
- 所有 pi 子进程（交互 + `--mode json` + install/list/version）注入 `PI_CODING_AGENT_DIR=~/.rick/pi/agent`（`piagent.AgentEnv()`）
- 托管 settings.json 固化 `hideThinkingBlock: true`（隐藏 thinking 块，思考仍生成不展示）
- 首次运行从旧 `~/.pi/agent/settings.json` 一次性迁移 theme + rick 托管的扩展包（用户 ad-hoc 包不迁移，避免引用未安装包）
- `RICK_PI_AGENT_DIR` 环境变量可覆盖 agent 目录（测试用；不是 PI_CODING_AGENT_DIR）

## rick tools theme [name]

- 无参数：列出可选主题 + 当前主题
- **默认主题 `rick`**（embedded themes/rick.json）：工具标题/命令绿（toolTitle=greenBright）、md 标题金（mdHeading=gold）、链接/路径蓝（mdLink/syntaxString=linkBlue）、bashMode 金；bootstrap 时无 theme 则自动 seed（ensureRickTheme）；用户已有主题则不覆盖
- **tokyo-night 已剔除**：@wishx127/pi-tokyo-night 捆绑硬编码配色的 Powerline 状态栏扩展，不随主题变化且污染上下文；init-pi 每次运行自动从 packages/theme 清除它
- 带 name：激活主题；自动 `pi install npm:<pkg>` 安装提供方包（写入隔离目录注册），再写 managed settings.json 的 theme 字段（保留 hideThinkingBlock 等字段）
- 已知主题映射：`internal/cmd/tools_theme.go` 的 knownThemes（name → npm pkg）
- 验证：`rick tools theme` 列表 → 切换 → `rick tools theme` 确认
- ⚠️ go:embed 陷阱：改 embedded themes/*.json 后必须先 `go build -o bin/rick ./cmd/rick` 再激活，否则旧 embed 覆盖磁盘主题文件

## rick 自闭环 pi 运行时（job_34 起）

- rick 不用全局 pi：`init-pi` 用 `npm install --prefix ~/.rick/pi/agent/runtime @earendil-works/pi-coding-agent@<全局版本>` 装独立副本（全局有则匹配版本，pinned 失败降级 latest）；`FindBinary`/`piPathOrDefault`/`piCommand`/Executor 默认全部优先托管运行时（`piagent.RuntimeBin()`，cfg.PiPath 仍最高优先）
- **~/.rick/pi 隔离逻辑必须保留**：rick 的 pi（配置 + 运行时）与用户的全局/独立 pi 完全隔离，互不污染，可独立升级
- 运行时副本保持**原样（stock）**，不做任何代码级修改；UI 定制只走主题配置（`rick tools theme`）
- 设计权衡（job_34 结论）：pi 的 diff/命令渲染行为（反显高亮、语法高亮）只能通过改 pi 代码实现，主题 token 覆盖不到；该做法需修改运行时副本，当前**不引入**（后续如需再做），仅保留隔离基础设施
