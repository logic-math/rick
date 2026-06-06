# ctrl 命令工作原理与使用指南

## 概述

`rick ctrl --job <job_id>` 是 Rick 的**监控与干预**命令。当 `rick doing` 在后台运行时，ctrl 启动一个交互式 Claude 会话，让人类实时观测进度并下达干预指令。

## 架构定位

```
rick doing（后台）          rick ctrl（前台交互）
─────────────────          ─────────────────────
执行 tasks.json 任务  ←读取→  jobs/{id}/doing/tasks.json
写入 raw_session_coding.log ←读取→  jobs/{id}/doing/tasks/{task_id}/raw_session_coding.log
                      ↑写入
                      jobs/{id}/plan/task<N>.md（追加干预指令）
                      jobs/{id}/doing/tasks.json（重置 task 状态）
```

ctrl 与 doing 之间**仅通过文件通信**，ctrl 不终止 doing 进程，只改变文件状态影响 doing 未来行为。

## 命令语法

```bash
rick ctrl --job <job_id>          # 启动交互式监控会话
rick ctrl --job <job_id> --dry-run # 预览 prompt（不启动 Claude）
```

`--job` 为必传参数，无默认值。

## 工作流程

### 1. 启动时自动汇报

ctrl 启动后立即：
1. 读取 `tasks.json`，展示任务状态表格（task_id / task_name / status / attempts）
2. 找到 `status = "running"` 的任务，读取其 `raw_session_coding.log` 最后 40 行
3. 读取 `debug.md` 检查是否有未解决的失败记录
4. 输出进度摘要

**汇报格式示例：**
```
📊 进度：2/5 完成，1 运行中，2 待执行

✅ task1 (创建接口定义) — 成功，耗时 3m20s
🔄 task3 (编写单元测试) — 运行中，已重试 0 次
   最近动作：Write → internal/executor/runner_test.go
⏳ task4 — 等待
```

### 2. 定时监控

ctrl agent 会主动提议每 20 分钟自动刷新一次（通过 `/loop 20m`），人类可选择开启。异常判断标准：
- `attempts ≥ 2`：重试多次，建议人类查看
- 连续 `tool_result` is_error=true：连续报错，建议干预
- debug.md 最新条目进展=❌：存在未修复问题

### 3. 干预场景

| 场景 | 触发动作 | ctrl 执行的文件操作 |
|------|---------|-------------------|
| **A** | 追加指令/修改方向 | 在 `plan/task<N>.md` 末尾追加 `## 干预指令` 章节，同时执行场景 B |
| **B** | 重置任务重新执行 | 将 `tasks.json` 中目标 task 的 `status` 改为 `"pending"`，清空 `error` |
| **C** | 查看历史行为 | 读取 `doing/tasks/<task_id>/act-path.md` 展示摘要 |
| **D** | 查看原始日志 | 解析 `raw_session_coding.log` NDJSON，展示工具调用序列 |

> ⚠️ **重置正在运行的 task**：`status = "running"` 时直接重置无效，doing 会覆盖。需先 Ctrl+C 停止 doing，再 `rick doing --job <id>` 重新启动。

## Prompt 生成机制

ctrl 的 prompt 通过 `GenerateCtrlPromptFile(jobID, rickDir string)` 生成：
- 读取 `jobs/{id}/doing/tasks.json` 注入当前任务状态快照
- 将所有路径变量（`{{doing_dir}}`、`{{plan_dir}}`、`{{tasks_json_path}}`）替换为真实绝对路径
- 写入 `doing/prompts/ctrl_prompt.md` 并返回路径
- `callClaudeCodeCLI(cfg, promptFile)` 使用该路径启动交互式会话

## raw_session_coding.log NDJSON 格式

ctrl 读取的日志是标准 NDJSON，每行一个 JSON 对象：

```
type = "system"     → 会话初始化
type = "assistant"  → Claude 行为（tool_use / text）
type = "user"       → 工具执行结果（tool_result，is_error=true 表示失败）
type = "result"     → 会话结束汇总
```

## 如何使用

```bash
# 在一个终端启动 doing（后台运行）
rick doing --job job_5

# 另一个终端启动 ctrl 监控
rick ctrl --job job_5

# 干预示例
# 人类: "task3 好像一直在循环，帮我重置并加一条指令：不要修改 runner_test.go"
# ctrl: 展示 plan/task3.md 当前内容，确认后追加干预指令，重置 status=pending
```

## 相关资源

- 源码：`internal/cmd/ctrl.go`、`internal/prompt/ctrl_prompt.go`、`internal/prompt/templates/ctrl.md`
- 相关 Wiki：[failure_feedback_propagation.md](failure_feedback_propagation.md)、[act_path_mechanism.md](act_path_mechanism.md)
