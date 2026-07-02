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
- 通过 SENSE 方法论引导深度分析，产出存入 `.rick/RFC/` 目录
- 三个 sub agent 模板（think/learn/express）通过 Go embed 编译进二进制，运行时写出到系统 tmp
- dry-run 输出中 sub agent 路径为占位符格式（`<tmp>/human_loop_think_*.md`），不含真实 `/tmp/` 路径
- 自动创建 `.rick/RFC/` 目录（MkdirAll，幂等）
- 复用 `callClaudeCodeCLI`（plan.go 中定义，同包内共享，不重复声明）
- 会话结束后 defer 清理所有 tmp 文件

### human-loop 验证

```bash
# 验证 dry-run 输出含关键词（不依赖特定 --phase/--keywords 参数）
./bin/rick human-loop --dry-run '测试主题' | grep "human_loop_think"
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
