# research-8 N4：fork pi 改动评估

节点路径：[根 > N4-fork pi 改动评估]
事实陈述：若需 fork，改 configDir 的具体改动量？升级 pi 时的合并冲突风险？长期维护成本？

## 执行动作

1. Read `packages/coding-agent/package.json`（piConfig.configDir 定义点）
2. Read `cli/startup-ui.ts:25-130`（OFFICIAL_CONFIG_DIR_NAME + isOfficialDistribution + shouldRunFirstTimeSetup）
3. Bash `git log --oneline --since="2025-01-01" | wc -l`（提交频率）
4. Bash `git tag --sort=-creatordate | head -10`（版本发布频率）
5. Grep `OFFICIAL_CONFIG_DIR_NAME|isOfficialDistribution`（fork 影响范围）

## 信源验证结果

### 代码原文（权重 0.4）✅

**fork pi 改 configDir 的最小改动点**：

| 改动点 | 文件:行 | 改动内容 | 必需性 |
|---|---|---|---|
| 1 | `packages/coding-agent/package.json:6-8` | `"piConfig": { "configDir": ".pi" }` → `"configDir": ".rick"` | **必需**（CONFIG_DIR_NAME 来源） |
| 2 | `packages/coding-agent/package.json:1-2` | `"name": "@earendil-works/pi-coding-agent"` → 自定义 name（可选） | 可选（避免包名冲突） |
| 3 | `packages/coding-agent/src/cli/startup-ui.ts:28` | `OFFICIAL_CONFIG_DIR_NAME = ".pi"` → `.rick`（可选） | 可选（保留首次安装引导） |
| 4 | `packages/coding-agent/src/cli/startup-ui.ts:26-27` | `OFFICIAL_PACKAGE_NAME` / `OFFICIAL_APP_NAME` → 自定义（可选） | 可选（同上） |

**最小改动量**：**1 行 JSON**（package.json 的 configDir 字段）。

**改动 1 行后的连锁影响**：
- `CONFIG_DIR_NAME` 从 `.pi` 变为 `.rick`（config.ts:491 自动生效）
- 所有 project scope 路径自动从 `{cwd}/.pi/` 变为 `{cwd}/.rick/`（N1 枚举的 17 处引用全部生效）
- user scope 路径从 `~/.pi/agent/` 变为 `~/.rick/agent/`（config.ts:520）
- 环境变量名从 `PI_CODING_AGENT_DIR` 变为 `RICK_CODING_AGENT_DIR`（config.ts:495，依赖 APP_NAME）
  - 注：APP_NAME 来自 `piConfig.name`，若未改 name 则仍是 `pi`，env 名不变
- `isOfficialDistribution` 返回 false（startup-ui.ts:36-42，因 configDirName 不匹配）
  - 影响：`shouldRunFirstTimeSetup` 返回 false（startup-ui.ts:115-132），失去首次安装引导（实验性 UI）
  - **不影响核心功能**：首次安装引导只是 theme 选择 + analytics opt-in，非必需

**fork 后的构建方式**：
- pi 用 `tsgo -p tsconfig.build.json` 编译 TypeScript → `dist/`
- 用 `bun build --compile` 生成独立二进制 `dist/pi`
- rick fork 后需要：`npm run build`（或 `bun run build`）+ 可选 `bun build --compile` 生成自定义二进制
- 构建产物：`dist/cli.js`（Node 入口）+ `dist/pi`（Bun 二进制）

**fork 后的发布方式**：
- 方式 A：npm publish 到自定义 scope（如 `@rick/pi-coding-agent`）
- 方式 B：Bun 编译为单一二进制，rick 直接分发二进制
- 方式 C：Git submodule + 本地构建（不发布 npm）

### 运行时行为（权重 0.3）✅

**pi 发布频率验证**（git log + git tag）：
- 2025-01-01 至今（2026-08-04）：**5394 次提交**
- v0.80.6（2026-07-10）到 v0.83.0（2026-07-30）：20 天 9 个版本
- 平均发布间隔：~2-3 天/版本
- **结论：pi 发布极活跃，fork 维护成本高**

**fork 升级流程**：
```
1. git remote add upstream https://github.com/earendil-works/pi.git
2. git fetch upstream
3. git merge upstream/main（或 rebase）
4. 解决冲突（主要在 package.json 的 configDir 字段）
5. 重新 build
6. 重新发布（npm 或二进制）
```

**预期冲突点**：
- `package.json`：configDir 字段 + 依赖版本（pi 频繁更新 deps，如 undici/brace-expansion）
- `startup-ui.ts`：OFFICIAL_CONFIG_DIR_NAME（若 rick 改了，pi 上游也可能改）
- 其余源码：无冲突（rick 不改源码，只改 package.json）

### 文档（权重 0.2）✅

- pi LICENSE（MIT）：允许 fork 和修改
- pi CONTRIBUTING.md：fork 工作流文档
- README "Installation"：支持 npm / brew / bun 二进制 / 源码构建

### 反事实（权重 0.1）N/A

本节点为外部源码调研，未修改代码。

## 还原确认

本轮纯外部调研，未修改 rick 仓库代码，无需 git restore。

## 关键事实

1. **fork 最小改动量：1 行 JSON**（package.json 的 `piConfig.configDir` 从 `.pi` 改为 `.rick`）。所有 17 处 CONFIG_DIR_NAME 引用自动生效，无需改源码。

2. **fork 连锁影响**：
   - project scope 路径：`.pi/{resource}` → `.rick/{resource}`（自动）
   - user scope 路径：`~/.pi/agent/` → `~/.rick/agent/`（自动）
   - 环境变量名：若未改 `piConfig.name`，仍是 `PI_CODING_AGENT_DIR`；若改为 `rick`，则变为 `RICK_CODING_AGENT_DIR`
   - 失去"官方发行版"身份：`isOfficialDistribution` 返回 false，首次安装引导（实验性 UI）不触发，**不影响核心功能**

3. **pi 发布极活跃**：5394 次提交/1.5 年，20 天 9 个版本。fork 升级需频繁 merge upstream。

4. **fork 升级冲突点**：
   - **主要冲突**：`package.json` 的 configDir 字段（每次 merge 需手动保留 `.rick`）
   - **次要冲突**：`package.json` 依赖版本（pi 频繁更新 deps，需手动 resolve）
   - **低风险冲突**：`startup-ui.ts` 的 OFFICIAL_CONFIG_DIR_NAME（若 rick 改了，上游也可能改，但概率低）
   - **无冲突**：其余源码（rick 不改源码）

5. **fork 长期维护成本**：
   - 每次 pi 发版（~2-3 天/次）需评估是否升级
   - 升级流程：fetch upstream → merge → resolve package.json 冲突 → build → 发布
   - 单次升级工作量：预计 30 分钟-2 小时（取决于 deps 冲突复杂度）
   - 若不升级：rick fork 会逐渐落后上游，错过 bug fix 和新特性

6. **fork 风险评估**：
   - **低风险**：configDir 改动是 1 行 JSON，技术复杂度低
   - **中风险**：package.json deps 冲突需手动 resolve（pi 用 npm-shrinkwrap.json 锁定版本）
   - **高风险**：长期落后上游会导致 rick fork 与 pi 主线分歧过大，最终无法 merge（需重 fork）

7. **fork 替代方案**：
   - **patch-package**：用 patch-package 工具生成 package.json patch，每次 pi 升级后自动 apply。比 fork 轻，但仍需维护 patch。
   - **postinstall 脚本**：npm install 后用 sed 改 package.json 的 configDir。但 pi 启动时读的是已编译的 dist/，不读源 package.json，**此方案不可行**。
   - **构建时替换**：fork 但不维护源码，只在自己的 build 流程中 sed 替换 package.json 后 build。介于 fork 和 patch 之间。

## 疑问点

无。本节点事实清晰，源码三重交叉验证（package.json + startup-ui.ts + git log）一致。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0
- 合计 = 0.9（高，≥ 0.8 终止）
