# skill:pi-orchestration-syntax（pi 子代理编排显式触发语法）

## 触发场景

在 rick 的 prompt 模板（`internal/prompt/templates/*.md`）或 `.rick/skills`、`.rick/loops` 中描述「派发子代理 / 启动 subagent / 多代理协作」时使用，特别是：

- 把旧的自然语言触发词（「派发 subagent」「SPAWN Sub Agent」「Main Agent」「Sub Agent」「父 Agent」「子 Agent」）迁移为 pi 显式触发语法
- 编写 `workflowScript` 编排（`runs.run` / `runs.all`）
- 需要明确某个 agent 是否「不持 subagent 工具、不递归派发」

信号词：模板中出现「派发 / SPAWN / Main Agent / 子 Agent」等描述，但运行时 pi 不真正派发子代理。

## 预期效果

- 模板中的编排描述被 pi 运行时**真正执行**，而非仅作为自然语言说明被忽略
- 自然语言触发词 grep 计数归零，显式语法（`workflowScript|runs.run|runs.all|agent:'`）计数 > 0
- dry-run 输出不含未替换的 `{{xxx_agent_path}}` 占位符

## 核心内容

### 1. 自然语言 → 显式 pi 语法映射

| 旧自然语言 | 显式 pi 语法 |
|-----------|-------------|
| 派发 subagent / SPAWN Sub Agent | `subagent({ workflowScript: "..." })` |
| 每个 Step 启动一个独立子 Agent | `runs.run('key', {agent:'worker', task:'...'})` |
| 并行派发 N 个子 Agent | `runs.all([{key:'...', agent:'...', task:'...'}, ...])` |
| think 子代理 | `agent:'think'` |
| research 子代理 | `agent:'research'` |
| exporter 子代理 | `agent:'exporter'` |
| Main Agent / 父 Agent | 你自己（parent 编排者） |
| 普通实现子代理 | `agent:'worker'` |
| 只读评审子代理 | `agent:'reviewer'` |

### 2. 硬规则

- 编排描述必须写成 pi 能解析的显式语法；只写「派发一个 subagent」这种自然语言，pi 不会真正派发
- 普通 child（think/research/exporter/worker）**不持 subagent 工具**、不递归派发；只有 parent 持编排权
- 单写者原则：同一 cwd/worktree 只保留一个写者；只读评审用 `agent:'reviewer'` + `context:"fresh"`

### 3. 验证

```bash
# 旧自然语言触发词应为 0
grep -roiE '派发\s*subagent|SPAWN\s+Sub\s+Agent|Main\s+Agent|Sub\s+Agent|父\s*Agent|子\s*Agent' internal/prompt/templates/ | wc -l
# 显式语法应存在
grep -rlE 'workflowScript|runs\.run|runs\.all' internal/prompt/templates/ | wc -l
# dry-run 无残留占位符
./bin/rick <cmd> --dry-run 2>&1 | grep -c '{{'   # 应为 0
```

pi 编排语法权威来源：`/home/hadoop-recsys/.rick/pi/agent/npm/node_modules/pi-subagents/skills/pi-subagents/SKILL.md`
