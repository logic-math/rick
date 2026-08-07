# research-8 N1：pi 路径硬编码点完整枚举

节点路径：[根 > N1-pi 路径硬编码点完整枚举]
事实陈述：源码级调研 pi 仓库中所有 `.pi` / `CONFIG_DIR_NAME` / 默认路径硬编码点，分类：可 env 覆盖 / 可 flag 覆盖 / 可 extension 拦截 / 仅 fork 可改。

## 执行动作

1. Grep `CONFIG_DIR_NAME|configDir|\.pi` 全 src 目录（head_limit 100）
2. Grep `process\.env\[ENV_|process\.env\.PI_` 全 src 目录（head_limit 80）
3. Grep `mkdirSync|ensureDirSync|mkdirp` 验证 .pi 目录是否被主动创建
4. Grep `getContextFiles|--no-context-files` 验证 context files 加载
5. Read `config.ts:470-566`（CONFIG_DIR_NAME 定义 + user scope 路径）
6. Read `resource-loader.ts:800-880, 1000-1050`（project scope 默认路径 + SYSTEM.md/APPEND_SYSTEM.md）
7. Read `package-manager.ts:1990-2116`（.pi/npm .pi/git 安装路径）
8. Read `skills.ts:420-480`（project scope skills 路径）
9. Read `extensions/loader.ts:680-712`（project scope extensions 路径）
10. Read `settings-manager.ts:195-205`（project scope settings.json 路径）
11. Read `prompt-templates.ts:185-215`（project scope prompts 路径）
12. Read `trust-manager.ts:170-206`（.pi trust 检测）

## 信源验证结果

### 代码原文（权重 0.4）✅

**CONFIG_DIR_NAME 定义点**（config.ts:491）：
```ts
export const CONFIG_DIR_NAME: string = pkg.piConfig?.configDir || ".pi";
```
- 来源：`package.json` 的 `piConfig.configDir` 字段
- 性质：**编译期常量**（pi 启动时读 package.json 一次，运行时不可改）
- 默认值：`".pi"`

**CONFIG_DIR_NAME 引用点全枚举**（17 处源码引用，分类如下）：

| # | 文件:行 | 用途 | 路径形态 | 覆盖方式 |
|---|---|---|---|---|
| 1 | config.ts:520 | user scope agent dir 默认根 | `~/{CONFIG_DIR_NAME}/agent` | `PI_CODING_AGENT_DIR` env 覆盖 |
| 2 | migrations.ts:259 | 迁移旧 project dir | `{cwd}/{CONFIG_DIR_NAME}` | 无（迁移逻辑，一次性） |
| 3 | package-manager-cli.ts:99,103,118,140 | help 文本 | `${CONFIG_DIR_NAME}/settings.json` | 无（仅文案） |
| 4 | startup-ui.ts:28,40,120 | 官方发行版检测 | `OFFICIAL_CONFIG_DIR_NAME=".pi"` 硬编码 | 无（UI 检测，不影响路径） |
| 5 | project-trust.ts:25 | trust 提示文案 | `${CONFIG_DIR_NAME} settings and resources` | 无（仅文案） |
| 6 | settings-manager.ts:201 | **project scope settings.json** | `{cwd}/{CONFIG_DIR_NAME}/settings.json` | **仅 fork** |
| 7 | session-manager.ts:474,1517,1555,1635 | 注释（session 在 agentDir 下） | `~/{CONFIG_DIR_NAME}/agent/sessions/` | `PI_CODING_AGENT_SESSION_DIR` env / `--session-dir` flag |
| 8 | package-manager.ts:904,2004,2045,2084,2110 | **project scope npm/git 安装** | `{cwd}/{CONFIG_DIR_NAME}/npm` `.pi/git` | **仅 fork**（scope=project 时） |
| 9 | args.ts:356,402 | help 文本 | `~/{CONFIG_DIR_NAME}/agent/sessions/` | 无（仅文案） |
| 10 | interactive-mode.ts:319,3686 | trust 提示 + reload 检测 | `${CONFIG_DIR_NAME} resources` | 无（仅文案/检测） |
| 11 | prompt-templates.ts:191,203 | **project scope prompts** | `{cwd}/{CONFIG_DIR_NAME}/prompts` | `--no-prompt-templates` 禁用 + `--prompt-template <path>` 显式加载 |
| 12 | resource-loader.ts:818-821,874,1023,1037 | **project scope skills/prompts/themes/extensions + SYSTEM.md/APPEND_SYSTEM.md** | `{cwd}/{CONFIG_DIR_NAME}/{resource}` | `--no-skills/--no-extensions/--no-themes` 禁用 + `--skill/--extension/--theme <path>` 显式加载 |
| 13 | skills.ts:432,436 | **project scope skills** | `{cwd}/{CONFIG_DIR_NAME}/skills` | `--no-skills` 禁用 + `--skill <path>` 显式加载 |
| 14 | trust-manager.ts:189-190 | **trust 检测** | `{cwd}/{CONFIG_DIR_NAME}/{entry}` existsSync 检测 | 无（仅检测存在性，不创建） |
| 15 | extensions/loader.ts:686-687 | **project scope extensions** | `{cwd}/{CONFIG_DIR_NAME}/extensions` | `--no-extensions` 禁用 + `--extension <path>` 显式加载 |
| 16 | config-selector.ts:94,212,213,851 | UI 显示 + project scope base | `{CONFIG_DIR_NAME}/settings.json` | 无（UI 文案 + scope 判断） |
| 17 | sdk.ts:41 | 注释 | `~/.pi/agent` | 无（仅注释） |

**关键分类汇总**：

| 分类 | 数量 | 说明 |
|---|---|---|
| 可 env 覆盖 | 2 处 | user scope agent dir（config.ts:520）+ session dir（session-manager.ts） |
| 可 flag 覆盖 | 6 处 | project scope 的 skills/prompts/themes/extensions/settings.json/SYSTEM.md 都有对应 `--no-X` 禁用 + `--X <path>` 显式加载 |
| 可 extension 拦截 | 0 处 | extension 无法拦截路径**计算**，只能通过 `resources_discover` 追加路径 |
| 仅 fork 可改 | 3 处 | project scope settings.json（settings-manager.ts:201）+ project scope npm/git 安装（package-manager.ts:2004,2084）+ CONFIG_DIR_NAME 常量本身 |
| 仅文案/注释/UI | 6 处 | 不影响实际路径计算 |

**pi 是否主动创建 .pi 目录**：
- Grep `mkdirSync.*CONFIG_DIR_NAME|mkdirSync.*\.pi` → **0 匹配**
- 所有 mkdirSync 调用（17 处）都在 user scope（agentDir 下）或临时目录
- **结论：pi 从不主动创建 project scope `.pi` 目录**。.pi 目录只在用户手动放置资源（skills/extensions/agents/SYSTEM.md/settings.json）时才存在
- trust-manager.ts:189 只做 `existsSync` 检测，不创建

**user scope 路径硬编码点**（config.ts:514-566）：
- `getAgentDir()` → `~/{CONFIG_DIR_NAME}/agent`（env `PI_CODING_AGENT_DIR` 可覆盖）
- 子路径全部 `join(getAgentDir(), ...)`：themes/models.json/auth.json/settings.json/tools/bin/prompts/sessions/debug.log
- 这些子路径**不可单独覆盖**，只能通过覆盖 `getAgentDir()` 整体重定向

**project scope 路径硬编码点**（6 类资源）：
- skills: `{cwd}/{CONFIG_DIR_NAME}/skills`（skills.ts:432）
- prompts: `{cwd}/{CONFIG_DIR_NAME}/prompts`（prompt-templates.ts:203, resource-loader.ts:819）
- themes: `{cwd}/{CONFIG_DIR_NAME}/themes`（resource-loader.ts:820,874）
- extensions: `{cwd}/{CONFIG_DIR_NAME}/extensions`（extensions/loader.ts:687, resource-loader.ts:821）
- settings.json: `{cwd}/{CONFIG_DIR_NAME}/settings.json`（settings-manager.ts:201）
- SYSTEM.md/APPEND_SYSTEM.md: `{cwd}/{CONFIG_DIR_NAME}/SYSTEM.md` `.pi/APPEND_SYSTEM.md`（resource-loader.ts:1023,1037）
- npm/git 安装: `{cwd}/{CONFIG_DIR_NAME}/npm` `.pi/git`（package-manager.ts:2004,2084，仅 scope=project 时）
- agents: `{cwd}/{CONFIG_DIR_NAME}/agents`（subagent README 文档，源码在 subagent extension 内）

### 运行时行为（权重 0.3）✅

**pi 仓库自身 .pi 目录验证**（/tmp/pi_repo/.pi/）：
- 实际存在子目录：`extensions/` `git/` `npm/` `prompts/` `skills/`
- 其中 `git/` `npm/` 是 package-manager.ts 安装产物（pi 仓库自身作为开发项目被 pi 工具操作时创建）
- `skills/` `prompts/` `extensions/` 是 pi 仓库自带的 project scope 资源（开发时手动放置）
- **证明：.pi 目录是资源发现 + 包安装的产物，不是 pi 启动时强制创建**

### 文档（权重 0.2）✅

- README "Customization"：所有资源路径均可通过 flag/env 重定向
- extensions.md "Locations"：`~/.pi/agent/extensions/` 和 `.pi/extensions/` 是默认发现路径
- subagent README：agent 定义路径 `~/.pi/agent/agents/` 和 `.pi/agents/`

### 反事实（权重 0.1）N/A

本节点为外部源码调研，未修改代码，无反事实验证。

## 还原确认

本轮纯外部调研，未修改 rick 仓库代码，无需 git restore。

## 关键事实

1. **CONFIG_DIR_NAME 是编译期常量**：`pkg.piConfig?.configDir || ".pi"`（config.ts:491），来自 package.json，运行时不可改。要改 project scope 目录名必须 fork pi 改 package.json。

2. **pi 从不主动创建 .pi 目录**：源码中 0 处 `mkdirSync` 针对 CONFIG_DIR_NAME。.pi 目录只在以下场景出现：
   - 用户手动放置 project scope 资源（skills/extensions/agents/SYSTEM.md）
   - package-manager 安装 project scope 包时创建 `.pi/npm` `.pi/git`（仅 scope=project 且用户主动 `pi install` 时）
   - trust-manager 检测 `.pi/{entry}` 存在性（只 existsSync，不创建）

3. **user scope 路径可整体 env 覆盖**：`PI_CODING_AGENT_DIR` 覆盖 `getAgentDir()`，所有子路径（skills/prompts/themes/extensions/agents/sessions/tools/bin/auth.json/settings.json/models.json）随之移动。

4. **project scope 资源可 flag 禁用 + 显式加载**：
   - `--no-skills` / `--no-extensions` / `--no-prompt-templates` / `--no-themes` / `--no-context-files` 禁用默认发现
   - `--skill <path>` / `--extension <path>` / `--prompt-template <path>` / `--theme <path>` 显式加载任意路径
   - 但 `SYSTEM.md` / `APPEND_SYSTEM.md` / `settings.json`（project scope）**无 flag 禁用**，只能通过 `--system-prompt` / `--append-system-prompt` 覆盖内容（不读 .pi/SYSTEM.md）

5. **project scope 包安装路径仅 fork 可改**：`.pi/npm` `.pi/git`（package-manager.ts:2004,2084）硬编码 CONFIG_DIR_NAME，无 env/flag 覆盖。若 rick 不使用 pi 的包管理器安装 project scope 包，则这两个路径不会被创建。

6. **3 处真正"仅 fork 可改"的硬编码点**：
   - CONFIG_DIR_NAME 常量本身（config.ts:491）
   - project scope settings.json 路径（settings-manager.ts:201）
   - project scope npm/git 安装路径（package-manager.ts:2004,2084）
   - 其余 project scope 路径都有 flag 禁用 + 显式加载的覆盖能力

7. **trust 检测不创建目录**：trust-manager.ts:189 只 `existsSync` 检测 `.pi/{entry}`，不创建。若 .pi 不存在，trust 检测直接返回 false，pi 正常运行（只是不加载 project scope 资源）。

## 疑问点

无。本节点事实清晰，源码三重交叉验证（Grep 枚举 + Read 关键段 + .pi 目录实际内容）一致。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9（高，≥ 0.8 终止）
