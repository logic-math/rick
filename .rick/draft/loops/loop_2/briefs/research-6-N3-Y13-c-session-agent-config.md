# research-6 N3-Y13-c：pi session/agent/config 存储扩展性

节点路径：[根 > N3-Y13-c：pi session/agent/config 存储扩展性]
事实陈述：pi session 存储路径是否可配置、agent 定义是否可重定向到 .rick/、config 是否支持环境变量覆盖、并存方案下 pi 创建哪些 .pi 子目录。

## 执行动作

1. Read `/tmp/pi_repo/packages/coding-agent/src/config.ts`（agent dir / session dir 路径 + 环境变量）
2. Read `/tmp/pi_repo/packages/coding-agent/src/core/session-manager.ts`（session 存储路径计算 + getDefaultSessionDir）
3. Read `/tmp/pi_repo/packages/coding-agent/examples/extensions/subagent/README.md`（agent 定义路径）
4. Read `/tmp/pi_repo/packages/coding-agent/src/core/settings-manager.ts`（settings.json 配置项）

## 信源验证结果

### 代码原文（权重 0.4）✅

**session 存储路径**（session-manager.ts line 476-489）：
```ts
function getDefaultSessionDirPath(cwd: string, agentDir: string = getDefaultAgentDir()): string {
    const resolvedCwd = resolvePath(cwd);
    const resolvedAgentDir = resolvePath(agentDir);
    const safePath = `--${resolvedCwd.replace(/^[/\\]/, "").replace(/[/\\:]/g, "-")}--`;
    return join(resolvedAgentDir, "sessions", safePath);
}

export function getDefaultSessionDir(cwd: string, agentDir: string = getDefaultAgentDir()): string {
    const sessionDir = getDefaultSessionDirPath(cwd, agentDir);
    if (!existsSync(sessionDir)) {
        mkdirSync(sessionDir, { recursive: true });
    }
    return sessionDir;
}
```
- session 默认存储在 `{agentDir}/sessions/{cwd-encoded}/` 下
- cwd 编码为 `--{path-with-dashes}--` 格式
- `agentDir` 默认 `~/.pi/agent/`，可通过 `PI_CODING_AGENT_DIR` 重定向

**session 路径 flag 覆盖**（args.ts line 114-115, 267）：
- `--session-dir <dir>` — Directory for session storage and lookup
- `--session <path|id>` — Use specific session file or partial UUID
- `--session-id <id>` — Use exact project session ID
- `--fork <path|id>` — Fork specific session file
- `--no-session` — Don't save session (ephemeral)
- 环境变量 `PI_CODING_AGENT_SESSION_DIR` 被 `--session-dir` 覆盖

**agent 定义路径**（subagent README line 140-144）：
```
- `~/.pi/agent/agents/*.md` - User-level (always loaded)
- `.pi/agents/*.md` - Project-level (only with `agentScope: "project"` or `"both"`)
```
- agent 定义是 markdown + YAML frontmatter（name/description/tools/model）
- user scope：`{agentDir}/agents/`（可通过 `PI_CODING_AGENT_DIR` 重定向）
- project scope：`{cwd}/.pi/agents/`（`CONFIG_DIR_NAME` 编译期常量，不可运行时改）

**config 环境变量覆盖**（config.ts + args.ts Environment Variables 段）：
- `PI_CODING_AGENT_DIR` — 重定向 agent dir（含 skills/prompts/themes/extensions/agents/sessions/tools/bin）
- `PI_CODING_AGENT_SESSION_DIR` — 重定向 session dir（被 `--session-dir` 覆盖）
- `PI_PACKAGE_DIR` — 重定向 package dir（themes/docs/assets 解析根）
- `PI_OFFLINE` — 禁用启动时网络操作
- `PI_TELEMETRY` — 覆盖安装遥测
- `PI_SHARE_VIEWER_URL` — /share 命令的 base URL
- 各 LLM provider API key 环境变量（ANTHROPIC_API_KEY / OPENAI_API_KEY / GEMINI_API_KEY 等共 30+）
- `piConfig.configDir`（package.json，编译期）：.pi 目录名

### 运行时行为（权重 0.3）✅

**settings.json 配置项**（settings-manager.ts 验证）：
- settings.json 在 `{agentDir}/settings.json`（可通过 `PI_CODING_AGENT_DIR` 重定向）
- 包含：enabledExtensions / enabledTools / enabledSkills / enabledPromptTemplates / enabledThemes / quietStartup / imageAutoResize / uiMode 等
- settings-manager 支持项目级覆盖（`{cwd}/.pi/settings.json`）

**并存方案下 pi 创建的 .pi 子目录**（基于 N1 调研 + 本节点验证）：
- user scope（`~/.pi/agent/`）：pi 首次运行会创建 `sessions/`（getDefaultSessionDir 强制 mkdir）、`auth.json`（首次认证）、`settings.json`（首次写设置）、`models.json`（首次写模型）、`tools/`（首次装 tool）、`bin/`（首次装 fd/rg）、`-debug.log`（调试日志）
- project scope（`{cwd}/.pi/`）：pi **不会主动创建** project scope 目录，只在资源发现时扫描（若不存在则跳过）
- 但若 rick fork pi 改 configDir 为 `.rick`，则 project scope 变为 `{cwd}/.rick/`，与 rick 现有 `.rick/` **同名冲突**

### 文档（权重 0.2）✅

- README "Customization" 章节：所有资源路径均可通过 flag 或环境变量重定向
- extensions.md "Locations" 段：`~/.pi/agent/extensions/` 和 `.pi/extensions/` 是默认发现路径
- subagent README：agent 定义路径明确
- args.ts help 文本：`--session-dir` / `--session` / `--session-id` / `--fork` / `--no-session` 完整文档

### 反事实（权重 0.1）N/A

本节点为外部文档调研，无代码修改。

## 还原确认

无 rick 代码修改，无需还原。

## 关键事实

1. **session 存储路径可配置**：
   - 默认：`{agentDir}/sessions/{cwd-encoded}/`
   - `PI_CODING_AGENT_SESSION_DIR` 环境变量 → 重定向 session 根
   - `--session-dir <dir>` flag → 重定向 session 根（覆盖环境变量）
   - `--session <path|id>` / `--session-id <id>` → 指定具体 session 文件
   - `--no-session` → 不持久化（ephemeral）

2. **agent 定义可重定向到 .rick/**：
   - user scope agent 定义在 `{agentDir}/agents/`，通过 `PI_CODING_AGENT_DIR=.rick` 可重定向
   - project scope agent 定义在 `{cwd}/.pi/agents/`，**不可运行时重定向**（CONFIG_DIR_NAME 编译期常量）
   - 但 subagent extension 的 `agentScope` 参数控制加载范围（`"user"` / `"project"` / `"both"`），可设为 `"user"` 只读 user scope
   - 适配方案：rick extension 重写 subagent 的 agent 加载逻辑，从 `.rick/agents/` 加载（需自定义 extension）

3. **config 支持环境变量覆盖**：
   - 6 个 PI_ 前缀环境变量 + 30+ provider API key 环境变量
   - settings.json 配置项不支持环境变量覆盖（仅 flag / settings.json / 默认值三层）
   - `piConfig.configDir` 不支持环境变量覆盖（编译期常量）

4. **并存方案下 pi 创建的 .pi 子目录**：
   - user scope（`~/.pi/agent/`）：pi 首次运行**强制创建** `sessions/`（mkdirSync recursive）+ 首次认证创建 `auth.json` + 首次设置创建 `settings.json` + 调试日志 `{APP_NAME}-debug.log`
   - project scope（`{cwd}/.pi/`）：pi **不主动创建**，仅扫描发现
   - 若 rick 用 `PI_CODING_AGENT_DIR=/tmp/rick-agent` 重定向 user scope，则 `~/.pi/` 不被创建（但 project scope `.pi/` 在资源发现时若存在则扫描）

5. **Y13 human 期望"所有上下文从 .rick 读取"的实现路径**：
   - 方案 A（环境变量重定向 user scope）：
     - `PI_CODING_AGENT_DIR=.rick/agent` → user scope 变为 `.rick/agent/skills/` `.rick/agent/agents/` 等
     - 但 project scope 仍为 `.pi/skills/`（CONFIG_DIR_NAME 不可改）
     - 用 `--no-skills --no-extensions --no-prompt-templates --no-themes --no-context-files` 禁用 project scope
     - session 用 `--session-dir .rick/sessions` 或 `PI_CODING_AGENT_SESSION_DIR=.rick/sessions`
   - 方案 B（fork pi 改 configDir）：
     - fork pi，package.json `piConfig.configDir = ".rick"`
     - project scope 变为 `.rick/skills/` 等，与 rick 现有 `.rick/` 合并
     - 但 user scope 仍为 `~/.rick/agent/`（与 rick 项目级 `.rick/` 不冲突，因为是 home 目录）
   - 方案 C（并存，human 已确认"首次合并可以允许并存"）：
     - 允许 `~/.pi/agent/` 存在（pi 用户级配置）
     - 允许 `{cwd}/.pi/` 存在但为空（pi project scope，不使用）
     - rick 通过 flag 显式加载 `.rick/skills/` `.rick/agents/` 等
     - session 用 `--session-dir .rick/sessions` 重定向到 .rick

## 疑问点

无。本节点事实清晰，源码三重交叉验证（config.ts + session-manager.ts + subagent README）一致。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9（高，≥ 0.8 终止）
