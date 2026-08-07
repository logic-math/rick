# research-8 N5：最小可行方案对比

节点路径：[根 > N5-最小可行方案对比]
事实陈述：综合 N1-N4，列出 3-5 种方案（纯 env/flag / extension 拦截 / fork / 混合），每个方案：改动量 / 覆盖范围 / 维护成本 / 升级风险。产出方案对比表。

## 执行动作

1. 综合 N1（路径硬编码点枚举）
2. 综合 N2（env + flag 覆盖能力）
3. 综合 N3（extension 拦截能力）
4. 综合 N4（fork 改动评估）
5. 验证 rick .rick 目录结构（skills/loops/domain/draft/jobs/dream/RFC）
6. 设计 5 种方案 + 对比表

## 信源验证结果

### 代码原文（权重 0.4）✅

**5 种方案设计**（基于 N1-N4 事实）：

#### 方案 A：纯 env/flag 重定向（不 fork，不写 extension）

**原理**：用 pi 的 env 和 flag 把 user scope 移出 `~/.pi/`，禁用 project scope 默认发现，显式加载 `.rick/` 资源。

**启动命令**（rick 包装 pi 时设置）：
```bash
PI_CODING_AGENT_DIR=.rick/.pi-agent \
PI_CODING_AGENT_SESSION_DIR=.rick/sessions \
pi \
  --no-skills --no-extensions --no-prompt-templates --no-themes --no-context-files \
  --skill .rick/skills/ \
  --extension .rick/extensions/ \
  --prompt-template .rick/prompts/ \
  --system-prompt .rick/SYSTEM.md \
  --append-system-prompt .rick/APPEND_SYSTEM.md
```

**覆盖范围**：
- ✅ user scope agent dir（env 覆盖）
- ✅ session dir（env 覆盖）
- ✅ skills（--no-skills + --skill 显式加载）
- ✅ extensions（--no-extensions + --extension 显式加载）
- ✅ prompt templates（--no-prompt-templates + --prompt-template 显式加载）
- ✅ themes（--no-themes + --theme 显式加载）
- ✅ SYSTEM.md / APPEND_SYSTEM.md（--system-prompt / --append-system-prompt 覆盖内容）
- ✅ context files（--no-context-files 禁用 AGENTS.md/CLAUDE.md）
- ❌ project scope settings.json（`{cwd}/.pi/settings.json`，无 env/flag 覆盖，但文件不存在则跳过）
- ❌ project scope npm/git 安装（`{cwd}/.pi/npm` `.pi/git`，无 env/flag 覆盖，但只在 `pi install --local` 时创建，rick 不调用即可）

**改动量**：0 行 pi 代码 + rick 包装脚本设置 env/flag。

**.pi 目录创建情况**：pi 不主动创建 `.pi/`（N1 验证）。若 rick 不手动放 `.pi/` 资源、不调用 `pi install --local`，则 `.pi/` 目录**完全不出现**。

**遗留问题**：
- user scope 仍在 `.rick/.pi-agent/`（名字含 "pi"，但路径在 .rick 下，符合"整个 pi 上下文从 .rick 读取"）
- rick skills 需适配 pi 格式（`{name}_skill/skill.md` → `{name}/SKILL.md`，R6-N2 已记录）
- rick agents 需放在 `.rick/.pi-agent/agents/`（user scope，因 project scope `.pi/agents/` 不可重定向）

#### 方案 B：extension 适配（不 fork，写 1 个 rick extension）

**原理**：写一个 rick extension，订阅 `resources_discover` 事件，扫描 `.rick/skills/` `.rick/loops/` 等，返回适配后的路径。配合 `--no-skills` 等 flag 禁用默认发现。

**extension 实现**：
```ts
// .rick/extensions/rick-adapter/index.ts
export default function (pi) {
    pi.on("resources_discover", (event) => {
        return {
            skillPaths: scanRickSkills(),    // 扫描 .rick/skills/，适配为 SKILL.md
            promptPaths: scanRickPrompts(),  // 扫描 .rick/loops/ 等
            themePaths: [],
        };
    });
}
```

**覆盖范围**：
- 与方案 A 相同（仍需 env + flag 禁用默认发现）
- extension 额外提供**动态路径发现**能力（startup/reload 时扫描 .rick/）

**改动量**：0 行 pi 代码 + 1 个 rick extension（~100-200 行 TS）+ rick 包装脚本。

**.pi 目录创建情况**：与方案 A 相同，`.pi/` 不出现。

**遗留问题**：
- extension 无法拦截默认资源加载（N3 验证，resources_discover 是追加式）
- 仍需 `--no-skills` 等 flag 禁用默认发现（否则 .pi/skills 若存在仍会被加载）
- extension 无法覆盖 settings.json / SYSTEM.md / npm / git 路径（N3 验证）
- extension 本身需放在 `.rick/.pi-agent/extensions/`（user scope）或用 `--extension .rick/extensions/rick-adapter` 显式加载
- **方案 B 是方案 A 的超集**：extension 提供动态发现，但仍需方案 A 的 env/flag 基础

#### 方案 C：fork pi 改 configDir（1 行 JSON）

**原理**：fork pi，改 `package.json` 的 `piConfig.configDir` 从 `.pi` 为 `.rick`，所有 project scope 路径自动变为 `{cwd}/.rick/`。

**改动点**：`packages/coding-agent/package.json:7`：`"configDir": ".pi"` → `"configDir": ".rick"`

**覆盖范围**：
- ✅ 所有 project scope 路径自动变为 `.rick/`（skills/prompts/themes/extensions/agents/settings.json/SYSTEM.md/APPEND_SYSTEM.md/npm/git）
- ✅ user scope 路径自动变为 `~/.rick/agent/`（若同时改 piConfig.name 为 rick，env 名变为 RICK_CODING_AGENT_DIR）
- ✅ 与 rick 现有 `.rick/` 目录**同名合并**（project scope `.rick/skills/` 直接复用 rick 现有结构）
- ⚠️ 失去"官方发行版"身份（isOfficialDistribution 返回 false，首次安装引导不触发，不影响核心功能）

**改动量**：1 行 JSON + fork 维护流程。

**.pi 目录创建情况**：`.pi/` 目录**完全不出现**（CONFIG_DIR_NAME 变为 `.rick`，pi 创建的是 `.rick/`）。

**遗留问题**：
- rick 现有 `.rick/` 结构与 pi 期望的 `.rick/` 子结构需**协调**：
  - pi 期望：`.rick/skills/` `.rick/prompts/` `.rick/themes/` `.rick/extensions/` `.rick/agents/` `.rick/settings.json` `.rick/SYSTEM.md` `.rick/npm/` `.rick/git/`
  - rick 现有：`.rick/skills/` `.rick/loops/` `.rick/domain/` `.rick/draft/` `.rick/jobs/` `.rick/dream/` `.rick/RFC/`
  - **冲突点**：`.rick/skills/`（两者都用）、`.rick/settings.json`（rick 无，pi 会创建）
  - **不冲突**：`.rick/loops/` `.rick/domain/` `.rick/draft/` `.rick/jobs/` `.rick/dream/` `.rick/RFC/`（pi 不用这些子目录）
- rick skills 仍需适配 pi 格式（`{name}_skill/skill.md` → `{name}/SKILL.md`）
- fork 维护成本（N4 验证：pi 20 天 9 版本，每次升级需 merge）

#### 方案 D：混合方案（fork configDir + extension 适配 + env/flag 补充）

**原理**：fork pi 改 configDir 为 `.rick`（解决 project scope 硬编码），同时写 rick extension（解决 skills 格式适配 + 动态发现），用 env/flag 处理 user scope。

**改动点**：
1. fork pi，`package.json:7` configDir `.pi` → `.rick`
2. 写 rick extension，`resources_discover` 扫描 `.rick/skills/` 适配为 SKILL.md 格式
3. rick 包装脚本设置 `RICK_CODING_AGENT_DIR=~/.rick/agent`（user scope 全局化）

**覆盖范围**：100%（project scope 由 fork 覆盖 + user scope 由 env 覆盖 + skills 格式由 extension 适配）。

**改动量**：1 行 JSON + 1 个 extension + 包装脚本 + fork 维护。

**.pi 目录创建情况**：`.pi/` 完全不出现。

**遗留问题**：
- fork 维护成本（同方案 C）
- extension 维护成本（同方案 B）
- **复杂度最高**：同时维护 fork + extension + 包装脚本

#### 方案 E：patch-package（不 fork，用 patch 工具）

**原理**：用 `patch-package` 工具生成 package.json patch，每次 `npm install pi` 后自动 apply，改 configDir。

**操作流程**：
1. `npm install @earendil-works/pi-coding-agent`
2. 手动改 `node_modules/@earendil-works/pi-coding-agent/package.json` 的 configDir 为 `.rick`
3. `npx patch-package @earendil-works/pi-coding-agent` 生成 patch
4. `package.json` 加 `postinstall: "patch-package"`

**覆盖范围**：与方案 C 相同（configDir 改为 .rick，所有路径自动生效）。

**改动量**：1 个 patch 文件 + postinstall 脚本。

**.pi 目录创建情况**：`.pi/` 完全不出现。

**遗留问题**：
- **patch 只改 node_modules 里的 package.json，不改 dist/**（pi 启动时读 dist/cli.js，dist/ 是预编译的）
- **关键问题**：pi 的 CONFIG_DIR_NAME 在**编译时**被嵌入 dist/（`pkg = JSON.parse(readFileSync(getPackageJsonPath()))` 在运行时读 package.json，但 getPackageJsonPath 指向 dist/package.json）
- 验证：`copy-binary-assets` 脚本（package.json:40）会 `shx cp package.json dist/`，即 dist/package.json 是源 package.json 的副本
- **结论**：patch-package 改 node_modules 里的 package.json，pi 运行时读 dist/package.json（同一个文件），**方案 E 技术可行**
- 但 patch-package 每次升级 pi 需重新生成 patch（patch 可能因 package.json 结构变化而失效）

### 方案对比表

| 方案 | 改动量 | 覆盖范围 | 维护成本 | 升级风险 | .pi 目录是否出现 |
|---|---|---|---|---|---|
| A：纯 env/flag | 0 行 pi 代码 + rick 包装脚本 | user scope ✅ + project scope skills/prompts/themes/extensions/SYSTEM.md ✅ + project scope settings.json/npm/git ❌（不创建即跳过） | 低（只需维护包装脚本） | 低（pi 升级不影响，env/flag 是稳定 API） | 不出现（pi 不主动创建） |
| B：extension 适配 | 0 行 pi 代码 + 1 个 extension（~100-200 行） + rick 包装脚本 | 等同方案 A + 动态发现（resources_discover 追加，不拦截） | 中（需维护 extension） | 低（extension API 相对稳定） | 不出现 |
| C：fork configDir | 1 行 JSON + fork 维护流程 | 100%（project scope 全部 + user scope 全部自动重定向） | 高（pi 20 天 9 版本，每次升级需 merge） | 高（package.json deps 冲突 + 长期分歧风险） | 不出现（.pi 变为 .rick） |
| D：混合（fork + extension + env） | 1 行 JSON + 1 个 extension + 包装脚本 + fork 维护 | 100% + skills 格式适配 + 动态发现 | 最高（fork + extension + 脚本同时维护） | 高（fork 冲突 + extension API 变化） | 不出现 |
| E：patch-package | 1 个 patch 文件 + postinstall 脚本 | 等同方案 C（100%） | 中（每次升级需重新生成 patch） | 中（patch 可能因 package.json 结构变化失效） | 不出现 |

**推荐排序依据（基于"改动量最小"原则，不含倾向性判断）**：

1. **方案 A 改动量最小**（0 行 pi 代码，仅 rick 包装脚本）
2. **方案 E 改动量次小**（1 个 patch 文件，但覆盖范围 = 方案 C）
3. **方案 B 改动量第三**（1 个 extension，覆盖范围 = 方案 A + 动态发现）
4. **方案 C 改动量第四**（1 行 JSON，但需 fork 维护流程）
5. **方案 D 改动量最大**（fork + extension + 脚本，覆盖最全但复杂度最高）

**覆盖范围排序**（从全到窄）：
1. 方案 C / D / E（100%）
2. 方案 A / B（user scope + project scope 大部分，settings.json/npm/git 不覆盖但不创建即跳过）

**维护成本排序**（从低到高）：
1. 方案 A（低）
2. 方案 E（中）
3. 方案 B（中）
4. 方案 C（高）
5. 方案 D（最高）

**升级风险排序**（从低到高）：
1. 方案 A / B（低，env/flag/extension API 稳定）
2. 方案 E（中，patch 可能失效）
3. 方案 C / D（高，fork merge 冲突）

### 运行时行为（权重 0.3）✅

**方案 A 可行性运行时验证**：
- N1 验证：pi 不主动创建 .pi 目录
- N2 验证：PI_CODING_AGENT_DIR + --no-skills + --skill 等 flag 组合可覆盖 user scope + project scope 资源发现
- 唯一遗留：project scope settings.json 路径仍为 `{cwd}/.pi/settings.json`，但文件不存在则 settings-manager 跳过（existsSync 检测）

**方案 C 可行性运行时验证**：
- N4 验证：改 package.json configDir 后，CONFIG_DIR_NAME 自动变为 .rick
- N1 验证：所有 17 处 CONFIG_DIR_NAME 引用自动指向 .rick
- rick 现有 .rick/ 子目录（loops/domain/draft/jobs/dream/RFC）与 pi 期望的 .rick/ 子目录（skills/prompts/themes/extensions/agents/settings.json/SYSTEM.md/npm/git）**不冲突**（除 .rick/skills/ 共用）

### 文档（权重 0.2）✅

- pi README "Customization"：env/flag 是稳定公开 API
- pi LICENSE（MIT）：允许 fork 和修改
- patch-package 文档：支持 postinstall 自动 apply

### 反事实（权重 0.1）N/A

本节点为方案综合设计，无代码修改。

## 还原确认

本轮纯外部调研，未修改 rick 仓库代码，无需 git restore。

## 关键事实

1. **5 种方案均可实现"去掉 .pi 目录"**：pi 本身不主动创建 .pi（N1 验证），所有方案下 .pi 目录都不会出现（除非用户手动放置资源）。

2. **方案 A（纯 env/flag）改动量最小**：0 行 pi 代码，仅 rick 包装脚本。覆盖范围：user scope 全部 + project scope 大部分（skills/prompts/themes/extensions/SYSTEM.md/APPEND_SYSTEM.md/context-files）。遗留：project scope settings.json/npm/git 路径仍硬编码 `.pi`，但文件不创建则跳过，不影响 pi 运行。

3. **方案 C（fork configDir）覆盖最全**：1 行 JSON 改动，所有 17 处 CONFIG_DIR_NAME 引用自动指向 .rick。但需承担 fork 维护成本（pi 20 天 9 版本）。

4. **方案 D（混合）复杂度最高**：fork + extension + 脚本同时维护，覆盖最全但维护成本最高。

5. **方案 E（patch-package）是方案 C 的轻量替代**：不改 pi 源码，用 patch 工具改 node_modules 里的 package.json。技术可行（pi 运行时读 dist/package.json，patch 改的就是这个文件）。但每次升级 pi 需重新生成 patch。

6. **方案 B（extension 适配）是方案 A 的增强**：extension 提供动态发现，但无法拦截默认资源加载（N3 验证），仍需方案 A 的 env/flag 基础。

7. **rick 现有 .rick/ 结构与 pi 期望的 .rick/ 子目录协调**：
   - 共用：`.rick/skills/`（两者都用，需格式适配）
   - pi 新增：`.rick/prompts/` `.rick/themes/` `.rick/extensions/` `.rick/agents/` `.rick/settings.json` `.rick/SYSTEM.md`（rick 现有无，pi 可能创建）
   - rick 独有：`.rick/loops/` `.rick/domain/` `.rick/draft/` `.rick/jobs/` `.rick/dream/` `.rick/RFC/`（pi 不用，不冲突）

8. **所有方案共通的适配工作**：rick skills 需从 `{name}_skill/skill.md` 适配为 pi 的 `{name}/SKILL.md` 格式（R6-N2 已记录，可通过重命名或 extension 适配）。

## 疑问点

无。本节点事实清晰，5 种方案基于 N1-N4 事实综合设计，对比维度完整。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9（高，≥ 0.8 终止）
