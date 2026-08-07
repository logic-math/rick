# research-6 N1-Y13-a：pi .pi 目录结构与默认行为

节点路径：[根 > N1-Y13-a：pi .pi 目录结构与默认行为]
事实陈述：pi 默认创建哪些 .pi 子目录、用途、是否可禁用、是否有全局配置项控制 .pi 目录创建、是否可被环境变量/flag 重定向到其他路径。

## 执行动作

1. Read `/tmp/pi_repo/packages/coding-agent/src/config.ts`（agent dir / session dir / package dir 路径计算）
2. Read `/tmp/pi_repo/packages/coding-agent/src/cli/args.ts`（所有 CLI flag + 环境变量文档）
3. Read `/tmp/pi_repo/packages/coding-agent/src/core/resource-loader.ts`（skills/prompts/themes/extensions 加载路径）
4. Read `/tmp/pi_repo/packages/coding-agent/package.json`（piConfig.configDir 配置点）
5. Grep `.pi` / `CONFIG_DIR_NAME` / `ENV_AGENT_DIR` / `ENV_SESSION_DIR` 验证所有路径分支

## 信源验证结果

### 代码原文（权重 0.4）✅

**config.ts 关键事实**（line 487-566）：

- `pkg.piConfig?.configDir || ".pi"` → `CONFIG_DIR_NAME`（line 491）
- `pkg.piConfig?.name || "pi"` → `APP_NAME`（line 489）
- `ENV_AGENT_DIR = ${APP_NAME.toUpperCase()}_CODING_AGENT_DIR` → 即 `PI_CODING_AGENT_DIR`（line 495）
- `ENV_SESSION_DIR = ${APP_NAME.toUpperCase()}_CODING_AGENT_SESSION_DIR` → 即 `PI_CODING_AGENT_SESSION_DIR`（line 496）

**agent dir 默认路径**（line 515-521）：
```ts
export function getAgentDir(): string {
    const envDir = process.env[ENV_AGENT_DIR];  // 环境变量优先
    if (envDir) return expandTildePath(envDir);
    return join(homedir(), CONFIG_DIR_NAME, "agent");  // 默认 ~/.pi/agent
}
```

**agent dir 下子目录**（line 523-566）：
- `themes/` — 自定义主题（`getCustomThemesDir`）
- `models.json` — 模型配置（`getModelsPath`）
- `auth.json` — 凭证（`getAuthPath`）
- `settings.json` — 设置（`getSettingsPath`）
- `tools/` — 自定义工具（`getToolsDir`）
- `bin/` — 托管二进制（fd/rg）（`getBinDir`）
- `prompts/` — 提示词模板（`getPromptsDir`）
- `sessions/` — 会话存储（`getSessionsDir`）
- `{APP_NAME}-debug.log` — 调试日志（`getDebugLogPath`）
- `skills/` — 技能（resource-loader.ts line 812）
- `extensions/` — 扩展（resource-loader.ts line 815）
- `agents/` — 子代理定义（subagent README：`~/.pi/agent/agents/*.md`）

**package.json 验证**（line 5-7）：
```json
"piConfig": {
    "configDir": ".pi"
}
```
configDir 名是**编译期常量**，写在 package.json 中，运行时不可改。但可通过 fork pi + 修改 package.json 改为 `.rick`。

### 运行时行为（权重 0.3）✅

**args.ts CLI flag 验证**（line 95-200）：

- `--system-prompt <text>` — 替换默认系统提示词
- `--append-system-prompt <text>` — 追加系统提示词（可多次，可传文件路径）
- `--session-dir <dir>` — session 存储目录可重定向（覆盖 ENV_SESSION_DIR）
- `--session <path|id>` / `--session-id <id>` / `--fork <path|id>` / `--continue` / `--resume` / `--no-session`
- `--skill <path>` — 加载自定义 skill 文件/目录（可多次，覆盖默认发现）
- `--no-skills` / `--no-tools` / `--no-builtin-tools` / `--no-extensions` / `--no-prompt-templates` / `--no-themes` / `--no-context-files`
- `--extension, -e <path>` — 加载自定义 extension（可多次）
- `--prompt-template <path>` / `--theme <path>`
- `--models <patterns>` / `--provider <name>` / `--model <pattern>` / `--api-key <key>`

**args.ts 环境变量文档**（line 402-407）：
- `PI_CODING_AGENT_DIR` — Config directory (default: `~/.pi/agent`)
- `PI_CODING_AGENT_SESSION_DIR` — Session storage directory (overridden by `--session-dir`)
- `PI_PACKAGE_DIR` — Override package directory (for Nix/Guix store paths)
- `PI_OFFLINE` / `PI_TELEMETRY` / `PI_SHARE_VIEWER_URL`

### 文档（权重 0.2）✅

- pi 仓库 `.pi/` 目录实际内容（/tmp/pi_repo/.pi/）：`extensions/` `git/` `npm/` `prompts/` `skills/`（5 个子目录，与源码定义一致）
- README "Customization" 章节：Prompt Templates / Skills / Extensions / Themes / Pi Packages 各有专节
- extensions.md：`~/.pi/agent/extensions/` 和 `.pi/extensions/` 是默认发现路径
- subagent README：`~/.pi/agent/agents/` 和 `.pi/agents/` 是 agent 定义路径

### 反事实（权重 0.1）N/A

本节点为外部事实调研，未修改 rick 代码，无反事实验证。

## 还原确认

本轮纯外部调研，未修改 rick 仓库代码，无需 git restore。

## 关键事实

1. **.pi 目录结构**：pi 创建两级目录
   - user scope：`~/.pi/agent/`（`PI_CODING_AGENT_DIR` 可重定向）
     - 子目录：`skills/` `prompts/` `themes/` `extensions/` `agents/` `tools/` `bin/` `sessions/`
     - 文件：`models.json` `auth.json` `settings.json` `{APP_NAME}-debug.log`
   - project scope：`{cwd}/.pi/`（`CONFIG_DIR_NAME` = `.pi`，编译期常量）
     - 子目录：`skills/` `prompts/` `themes/` `extensions/` `agents/`
   - 注：sessions 默认在 user scope 的 `agent/sessions/` 下，按 cwd 编码子目录名

2. **可禁用**：`--no-skills` / `--no-extensions` / `--no-prompt-templates` / `--no-themes` / `--no-context-files` / `--no-tools` / `--no-builtin-tools` 可禁用各资源自动发现

3. **全局配置项控制 .pi 目录创建**：无运行时配置项。`.pi` 目录名由 package.json `piConfig.configDir` 决定，是**编译期常量**。要改目录名必须 fork pi 改 package.json。

4. **环境变量/flag 重定向**：
   - `PI_CODING_AGENT_DIR` 环境变量 → 重定向整个 user scope agent dir（`~/.pi/agent/` → 任意路径）
   - `PI_CODING_AGENT_SESSION_DIR` 环境变量 / `--session-dir` flag → 重定向 session 存储
   - `PI_PACKAGE_DIR` 环境变量 → 重定向 package 目录（themes/docs/assets 解析根）
   - `--skill <path>` / `--extension <path>` / `--prompt-template <path>` / `--theme <path>` → 加载自定义路径资源（覆盖默认发现）

5. **project scope `.pi` 目录不可通过环境变量重定向**：`CONFIG_DIR_NAME` 是编译期常量，project scope 路径 `{cwd}/.pi/skills/` 等不可通过运行时配置改。但可通过 `--no-skills` 禁用 project scope 发现，再用 `--skill <path>` 指向 `.rick/skills/`。

6. **Y13 human 期望"删除 .pi 目录"的实现路径**：
   - 方案 A（fork）：fork pi，改 package.json `piConfig.configDir` 为 `.rick`，则 project scope 变为 `{cwd}/.rick/`
   - 方案 B（flag 重定向）：不 fork，用 `PI_CODING_AGENT_DIR=/dev/null` 或类似禁用 user scope，`--no-skills --no-extensions --no-prompt-templates --no-themes --no-context-files` 禁用 project scope 发现，再用 `--skill .rick/skills/` `--extension .rick/extensions/` 等显式加载
   - 方案 C（并存）：允许 .pi 目录存在但为空，所有资源从 .rick 加载（human 已确认"首次合并可以允许并存"）

## 疑问点

无。本节点事实清晰，源码三重交叉验证（config.ts + args.ts + resource-loader.ts + package.json）一致。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9（高，≥ 0.8 终止）
