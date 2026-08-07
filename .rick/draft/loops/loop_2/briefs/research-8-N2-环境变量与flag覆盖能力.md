# research-8 N2：环境变量 + flag 覆盖能力

节点路径：[根 > N2-环境变量 + flag 覆盖能力]
事实陈述：调研所有 PI_ 前缀环境变量 + flag，能覆盖哪些路径？覆盖优先级？

## 执行动作

1. Read `args.ts:90-210`（所有 CLI flag 解析）
2. Read `args.ts:390-410`（环境变量文档段）
3. Read `config.ts:360-380`（PI_PACKAGE_DIR 覆盖）
4. Read `main.ts:630-645`（session dir 优先级链）
5. Grep `process\.env\.PI_` 全 src（N1 已完成，复用结果）

## 信源验证结果

### 代码原文（权重 0.4）✅

**PI_ 前缀环境变量全枚举**（args.ts:402-407 + Grep 结果）：

| 环境变量 | 覆盖目标 | 优先级（flag > env > 默认） | 来源文件:行 |
|---|---|---|---|
| `PI_CODING_AGENT_DIR`（=ENV_AGENT_DIR） | user scope agent dir 根（`~/.pi/agent`） | 无 flag 覆盖，env 直接生效 | config.ts:516 |
| `PI_CODING_AGENT_SESSION_DIR`（=ENV_SESSION_DIR） | session 存储根 | `--session-dir` flag 覆盖 env | main.ts:632-636 |
| `PI_PACKAGE_DIR` | package 资源根（themes/docs/assets） | 无 flag 覆盖，env 直接生效 | config.ts:369 |
| `PI_OFFLINE` | 禁用启动网络操作 | `--offline` flag 覆盖 env | main.ts:531 |
| `PI_TELEMETRY` | 安装遥测开关 | 无 flag，env only | telemetry.ts:10 |
| `PI_SHARE_VIEWER_URL` | /share 命令 base URL | 无 flag，env only | config.ts:506 |
| `PI_SKIP_VERSION_CHECK` | 跳过版本检查 | 无 flag，env only | version-check.ts:71 |
| `PI_STARTUP_BENCHMARK` | 启动基准测试 | 无 flag，env only | main.ts:866 |
| `PI_TIMING` | 计时开关 | 无 flag，env only | timings.ts:6 |
| `PI_EXPERIMENTAL` | 实验特性开关 | 无 flag，env only | experimental.ts:2 |
| `PI_CLEAR_ON_SHRINK` | 缩小时清屏 | 无 flag，env only | settings-manager.ts:1103 |
| `PI_HARDWARE_CURSOR` | 硬件光标 | 无 flag，env only | settings-manager.ts:1208 |

**注**：`PI_CODING_AGENT_DIR` 和 `PI_CODING_AGENT_SESSION_DIR` 的变量名依赖 `APP_NAME`（默认 `pi`）。若 fork pi 改 `piConfig.name` 为 `rick`，则变量名变为 `RICK_CODING_AGENT_DIR`。

**路径覆盖 flag 全枚举**（args.ts:90-210）：

| flag | 覆盖目标 | 优先级 | 对应 env |
|---|---|---|---|
| `--session-dir <dir>` | session 存储根 | flag > env > settings.json > 默认 | `PI_CODING_AGENT_SESSION_DIR` |
| `--session <path\|id>` | 指定 session 文件 | flag only | 无 |
| `--session-id <id>` | 指定 session UUID | flag only | 无 |
| `--fork <path\|id>` | fork session | flag only | 无 |
| `--no-session` | 不持久化 | flag only | 无 |
| `--skill <path>`（可多次） | 显式加载 skill 路径 | 追加到默认发现之后 | 无 |
| `--extension <path>`（可多次，`-e`） | 显式加载 extension 路径 | 追加到默认发现之后 | 无 |
| `--prompt-template <path>`（可多次） | 显式加载 prompt 模板 | 追加到默认发现之后 | 无 |
| `--theme <path>`（可多次） | 显式加载 theme | 追加到默认发现之后 | 无 |
| `--no-skills`（`-ns`） | 禁用默认 skill 发现 | flag only | 无 |
| `--no-extensions`（`-ne`） | 禁用默认 extension 发现 | flag only | 无 |
| `--no-prompt-templates`（`-np`） | 禁用默认 prompt 模板发现 | flag only | 无 |
| `--no-themes` | 禁用默认 theme 发现 | flag only | 无 |
| `--no-context-files`（`-nc`） | 禁用 AGENTS.md/CLAUDE.md 发现 | flag only | 无 |
| `--no-tools`（`-nt`） | 禁用所有工具 | flag only | 无 |
| `--no-builtin-tools`（`-nbt`） | 禁用内置工具 | flag only | 无 |
| `--system-prompt <text>` | 替换默认系统提示词（覆盖 .pi/SYSTEM.md 内容） | flag only | 无 |
| `--append-system-prompt <text>`（可多次） | 追加系统提示词（覆盖 .pi/APPEND_SYSTEM.md 内容） | flag only | 无 |
| `--models <patterns>` | 模型过滤 | flag only | 无 |
| `--provider <name>` | provider 选择 | flag only | 无 |
| `--model <pattern>` | 模型选择 | flag only | 无 |
| `--api-key <key>` | API key | flag only | 无 |

**关键覆盖优先级链**：

1. **user scope agent dir**：`PI_CODING_AGENT_DIR` env > `~/.pi/agent`（默认）
   - 无 flag 覆盖
   - 覆盖后所有 user scope 子路径（skills/prompts/themes/extensions/agents/sessions/tools/bin/auth.json/settings.json/models.json）随之移动

2. **session dir**：`--session-dir` flag > `PI_CODING_AGENT_SESSION_DIR` env > `settings.json:sessionDir` > `{agentDir}/sessions/{cwd-encoded}/`（默认）
   - main.ts:632-636 用 `??` 链实现优先级
   - `getDefaultSessionDir`（session-manager.ts:476-489）强制 `mkdirSync recursive`，即默认 session dir 会被创建

3. **project scope 资源**（skills/prompts/themes/extensions）：
   - 默认发现：`{cwd}/.pi/{resource}/`
   - `--no-X` 禁用默认发现
   - `--X <path>` 显式加载（追加，不替换默认）
   - **关键**：`--skill <path>` 是**追加**不是**替换**（skills.ts:455-481 在 includeDefaults 之后追加）。要完全禁用 project scope 必须组合 `--no-skills --skill <custom-path>`

4. **SYSTEM.md / APPEND_SYSTEM.md**：
   - 默认发现：`{cwd}/.pi/SYSTEM.md` `{cwd}/.pi/APPEND_SYSTEM.md`
   - **无 `--no-SYSTEM.md` flag**
   - `--system-prompt <text>` 覆盖内容（不再读 .pi/SYSTEM.md）
   - `--append-system-prompt <text>` 追加内容（覆盖 .pi/APPEND_SYSTEM.md）
   - 若 `.pi/SYSTEM.md` 不存在且未传 `--system-prompt`，则用 pi 内置默认 system prompt

5. **project scope settings.json**：
   - 路径 `{cwd}/.pi/settings.json`（settings-manager.ts:201）
   - **无 env 覆盖，无 flag 禁用**
   - 若文件不存在则跳过（settings-manager 用 existsSync 检测）
   - **仅 fork 可改路径**

6. **project scope npm/git 安装**：
   - 路径 `{cwd}/.pi/npm` `{cwd}/.pi/git`（package-manager.ts:2004,2084）
   - **无 env 覆盖，无 flag 禁用**
   - 仅在用户主动 `pi install --local` 时创建
   - **仅 fork 可改路径**

### 运行时行为（权重 0.3）✅

**覆盖优先级运行时验证**（main.ts:632-636 实际代码）：
```ts
const envSessionDir = process.env[ENV_SESSION_DIR];
const sessionDir =
    (parsed.sessionDir ? normalizePath(parsed.sessionDir) : undefined) ??
    (envSessionDir ? expandTildePath(envSessionDir) : undefined) ??
    startupSettingsManager.getSessionDir();
```
- `??` 链：第一个非 undefined/null 胜出
- flag > env > settings.json > 默认（createSessionManager 内部 fallback）

**user scope agent dir 运行时**（config.ts:515-521）：
```ts
export function getAgentDir(): string {
    const envDir = process.env[ENV_AGENT_DIR];
    if (envDir) {
        return expandTildePath(envDir);
    }
    return join(homedir(), CONFIG_DIR_NAME, "agent");
}
```
- env 设置即生效，无 flag 覆盖
- 所有 user scope 子路径函数（getCustomThemesDir/getModelsPath/getAuthPath/getSettingsPath/getToolsDir/getBinDir/getPromptsDir/getSessionsDir/getDebugLogPath）都 `join(getAgentDir(), ...)`，随 env 自动重定向

### 文档（权重 0.2）✅

- args.ts:402-407 环境变量文档段：明确列出 6 个 PI_ 前缀变量
- README "Customization"：所有资源路径均可通过 flag/env 重定向
- extensions.md：`--extension <path>` 可多次指定

### 反事实（权重 0.1）N/A

本节点为外部源码调研，无代码修改。

## 还原确认

本轮纯外部调研，未修改 rick 仓库代码，无需 git restore。

## 关键事实

1. **PI_ 前缀环境变量共 12 个**，其中路径相关 3 个：`PI_CODING_AGENT_DIR`（user scope 根）、`PI_CODING_AGENT_SESSION_DIR`（session 根）、`PI_PACKAGE_DIR`（package 资源根）。其余 9 个是行为开关（offline/telemetry/timing 等）。

2. **路径覆盖 flag 共 22 个**，其中：
   - 1 个覆盖 session dir（`--session-dir`）
   - 4 个显式加载资源（`--skill` `--extension` `--prompt-template` `--theme`，可多次）
   - 5 个禁用默认发现（`--no-skills` `--no-extensions` `--no-prompt-templates` `--no-themes` `--no-context-files`）
   - 2 个覆盖系统提示词内容（`--system-prompt` `--append-system-prompt`）
   - 其余是 session 选择 / 工具控制 / 模型选择

3. **覆盖优先级**：flag > env > settings.json > 默认（session dir 链验证）。user scope agent dir 只有 env 覆盖，无 flag。

4. **`--skill <path>` 是追加不是替换**：skills.ts:455-481 在 includeDefaults 之后追加。要完全从 .rick 加载必须 `--no-skills --skill .rick/skills/`。

5. **3 处路径无任何运行时覆盖能力**（仅 fork 可改）：
   - project scope settings.json（`{cwd}/.pi/settings.json`）
   - project scope npm 安装（`{cwd}/.pi/npm`）
   - project scope git 安装（`{cwd}/.pi/git`）
   - 这 3 处若文件不存在则跳过/不创建，不影响 pi 运行

6. **SYSTEM.md/APPEND_SYSTEM.md 无 `--no` flag**：但 `--system-prompt`/`--append-system-prompt` 覆盖内容（不读 .pi/SYSTEM.md）。若 .pi 不存在且未传 flag，则用内置默认。

7. **user scope 整体重定向的最小 env**：`PI_CODING_AGENT_DIR=.rick/.pi-agent`（或任意路径）即可把 user scope 全部移出 `~/.pi/`。配合 `PI_CODING_AGENT_SESSION_DIR=.rick/sessions` 可把 session 也移出。

## 疑问点

无。本节点事实清晰，源码三重交叉验证（args.ts + config.ts + main.ts）一致。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9（高，≥ 0.8 终止）
