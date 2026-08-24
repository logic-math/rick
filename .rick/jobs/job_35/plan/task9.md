# 依赖关系
task3, task4, task5

# 任务名称
注册 think/research/exporter 为 pi 自定义 agent（经 env 职责 3 落盘）

# 任务目标
按 spec（task2）落地 KR3.2 的 agent 注册部分：将 think/research/exporter 注册为 pi 自定义 agent——frontmatter 落盘到 `~/.rick/pi/agent/agents/`（user 级），system prompt = 对应源码 skill 的 wiki 内容，tools 声明对齐 pi 工具名。此任务产出的是 env 职责 3（`DeployRickCustomizations`）中「agent」定制物的具体内容。

生成 3 个 agent 文件（YAML frontmatter + system prompt）：
- `think.md`：`name: think`、`tools: read, grep, find, ls`、system prompt = `templates/skills/think.md`（推理识别 + 4 维打分 + 3 启发性问题）
- `research.md`：`name: research`、`tools: read, grep, find, ls, bash, web_search, fetch_content`、system prompt = `templates/skills/research.md`（尽调树 + 信源加权）
- `exporter.md`：`name: exporter`、`tools: read, write, bash`、system prompt = `templates/skills/exporter.md`（RFC 输出，大纲 + 内容两阶段）

由 env 职责 3 幂等写入，frontmatter 含 name/description/tools/defaultContext；不再生成普通 markdown 提示词文件（当前 think/research/exporter 是散落 prompt 文件、无 frontmatter、不在 pi 发现目录——见 research-report-S-reasons-agent.md B4）。

依据：research-report-S-reasons-agent.md B1（frontmatter 定义）/B2（user 级 `~/.rick/pi/agent/agents/**/*.md`）/B3（`runs.run(agent:'think')` 触发）；pi docs/agents.md。

参考：loop `agent-runtime-bootstrap-loop`；skill `pi_extension_install_verification_skill`、`pi_runtime_verification_skill`、`subprocess_env_isolation_skill`、`verify_go_changes_skill`。

# 关键结果
1. `internal/env`（职责 3）产出 3 个 agent 的 frontmatter + system prompt 内容，幂等写到 `runtime.AgentDir()/agents/{think,research,exporter}.md`（尊重 RICK_PI_AGENT_DIR/HOME 隔离）
2. 3 文件 frontmatter 含 `name`/`description`/`tools`/`defaultContext`；system prompt 为对应 skill wiki 正文（非空、非仅 frontmatter）
3. `~/.rick/pi/agent/agents/` 下 3 个 agent 可被 pi 发现——**先按 pi docs/agents.md 确认自定义 agent 的真实发现入口（`pi list` 现用于 extensions、未必列 agent），验收断言用真实入口**（think/research/exporter 真实 agent 名可被 `{action:"list"}` 或等价机制列出）
4. 幂等 + 覆盖语义：rick 自有的占位/旧版内容**必须按内容比对覆盖**（task3 未写 agent 文件，若存在旧散落产物则覆盖）；覆盖判定 = 文件 frontmatter 含 rick 固定标记（如 `rick-managed: true`）才覆盖，无标记文件跳过；重复运行不重复插入

# 测试方法
（本 task 测试用例设计遵循 skill:tdd + skill:testing-anti-patterns：先写失败测试；断言真实落盘文件不 mock；测试隔离用 RICK_PI_AGENT_DIR 指向 temp。）

1. 正常路径：前置条件 = task3/5 完成 + `RICK_PI_AGENT_DIR` 指向 temp；输入 = 运行 `DeployRickCustomizations`（或 init-pi）；操作 = 运行 + `test -f "$RICK_PI_AGENT_DIR/agents/think.md"` + 用 pi docs/agents.md 确认的真实 agent 发现入口（`{action:"list"}` 或等价）列出 think/research/exporter；预期 = 3 文件存在，`head -5 think.md` 含 `name: think`，真实发现入口列出 3 个 agent 名。
2. 边界（幂等 + 覆盖语义）：前置条件 = 已注册一次；输入 = 再次运行；操作 = 再次运行 + `sha256sum` 对比前后；预期 = 文件内容不变（幂等，不重复插入 frontmatter）；另预置一个无 `rick-managed: true` 标记的同名文件 → 运行后内容不被覆盖（仅覆盖有 rick 标记的文件）。
3. 异常（system prompt 非空）：前置条件 = 3 文件已写；输入 = 无；操作 = `awk '/^---$/{n++} n>=2 && /[^[:space:]]/{print}' "$RICK_PI_AGENT_DIR/agents/think.md" | grep -c .`；预期 = ≥1（frontmatter 闭合后存在非空正文，wiki 内容注入成功）。
