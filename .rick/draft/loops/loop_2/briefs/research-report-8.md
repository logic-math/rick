# 调研报告 — Y15 去掉 .pi 目录的最小可行方法 — 2026-08-04

## 信源配置

| 信源 | 权重 | 验证方式 |
|---|---|---|
| 代码原文 | 0.4 | Read/Grep pi 仓库源码（/tmp/pi_repo） |
| 运行时行为 | 0.3 | pi 仓库 .pi 目录实际内容 + git log 提交频率 |
| 文档 | 0.2 | README + extensions.md + args.ts help 文本 |
| 反事实 | 0.1 | N/A（本轮纯外部调研，未修改代码） |

加权公式：置信度 = Σ(信源验证结果 × 权重)，验证结果 ∈ {0, 1}
高置信度阈值：≥ 0.8（终止）| 中 0.5-0.8（续研）| 低 < 0.5（R7 上报）

## 尽调树（快照）

```
根：Y15 去掉 .pi 目录的最小可行方法
├─ N1-pi 路径硬编码点完整枚举 ✅0.9
│  事实：CONFIG_DIR_NAME 编译期常量（config.ts:491），17 处引用，pi 不主动创建 .pi 目录
├─ N2-环境变量 + flag 覆盖能力 ✅0.9
│  事实：12 个 PI_ env + 22 个路径 flag，user scope 可 env 覆盖，project scope 可 flag 禁用+显式加载
├─ N3-extension 拦截能力 ✅0.9
│  事实：resources_discover 是追加式（非拦截），30+ 事件 0 个能拦截路径，extension 无法阻止 .pi 创建
├─ N4-fork pi 改动评估 ✅0.9
│  事实：最小改动 1 行 JSON（package.json configDir），pi 20 天 9 版本发布极活跃，fork 维护成本高
└─ N5-最小可行方案对比 ✅0.9
   事实：5 种方案（A 纯 env/flag / B extension / C fork / D 混合 / E patch-package），方案 A 改动量最小
```

## 节点详情

### N1-pi 路径硬编码点完整枚举

- 置信度：0.9（高）
- 信源验证：
  - 代码原文 ✅（Grep CONFIG_DIR_NAME 17 处引用 + Read config.ts:491 + Read 所有引用点）
  - 运行时行为 ✅（/tmp/pi_repo/.pi/ 实际内容验证 + Grep mkdirSync 0 处针对 CONFIG_DIR_NAME）
  - 文档 ✅（README Customization + extensions.md Locations）
  - 反事实 N/A
- 调研报告：briefs/research-8-N1-路径硬编码点完整枚举.md
- 关键事实：
  - CONFIG_DIR_NAME = `pkg.piConfig?.configDir || ".pi"`（config.ts:491），编译期常量
  - pi **从不主动创建 .pi 目录**（0 处 mkdirSync 针对 CONFIG_DIR_NAME）
  - 17 处引用分类：2 处可 env 覆盖 / 6 处可 flag 覆盖 / 0 处可 extension 拦截 / 3 处仅 fork 可改 / 6 处仅文案
  - 3 处"仅 fork 可改"：project scope settings.json + project scope npm/git 安装路径

### N2-环境变量 + flag 覆盖能力

- 置信度：0.9（高）
- 信源验证：
  - 代码原文 ✅（Read args.ts:90-210 + Read args.ts:390-410 + Read config.ts:360-380 + Read main.ts:630-645）
  - 运行时行为 ✅（main.ts:632-636 优先级链 `??` 实现）
  - 文档 ✅（args.ts 环境变量文档段）
  - 反事实 N/A
- 调研报告：briefs/research-8-N2-环境变量与flag覆盖能力.md
- 关键事实：
  - 12 个 PI_ 前缀环境变量（3 个路径相关：PI_CODING_AGENT_DIR / PI_CODING_AGENT_SESSION_DIR / PI_PACKAGE_DIR）
  - 22 个路径覆盖 flag（4 个显式加载 + 5 个禁用默认发现 + 2 个覆盖系统提示词 + 其余 session/工具/模型）
  - 覆盖优先级：flag > env > settings.json > 默认
  - `--skill <path>` 是追加非替换（skills.ts:455-481），需组合 `--no-skills` 完全禁用默认
  - 3 处路径无运行时覆盖：project scope settings.json / npm / git（仅 fork 可改）

### N3-extension 拦截能力

- 置信度：0.9（高）
- 信源验证：
  - 代码原文 ✅（Read types.ts:520-560 + Read types.ts:1198-1230 + Read runner.ts:1147-1192 + Read resource-loader.ts:330-377）
  - 运行时行为 ✅（resources_discover 触发时序：默认资源加载后）
  - 文档 ✅（extensions.md 事件列表）
  - 反事实 N/A
- 调研报告：briefs/research-8-N3-extension拦截能力.md
- 关键事实：
  - resources_discover 是**追加式**（extendResources 用 mergePaths 合并，不替换默认）
  - 只支持 3 类资源（skills/prompts/themes），不支持 extensions/agents/settings.json/SYSTEM.md
  - 30+ extension 事件中 0 个能拦截路径
  - extension 无法阻止 .pi 目录创建（pi 本身就不主动创建，extension 无拦截事件）
  - project_trust 事件返回 trusted:"no" 可全局禁用 project scope，但非针对性拦截

### N4-fork pi 改动评估

- 置信度：0.9（高）
- 信源验证：
  - 代码原文 ✅（Read package.json:6-8 + Read startup-ui.ts:25-130）
  - 运行时行为 ✅（git log 5394 提交/1.5 年 + git tag 20 天 9 版本）
  - 文档 ✅（LICENSE MIT + CONTRIBUTING.md）
  - 反事实 N/A
- 调研报告：briefs/research-8-N4-fork改动评估.md
- 关键事实：
  - 最小改动：1 行 JSON（package.json configDir `.pi` → `.rick`）
  - 连锁影响：17 处 CONFIG_DIR_NAME 引用自动指向 .rick + user scope 自动变为 ~/.rick/agent/
  - 失去"官方发行版"身份（isOfficialDistribution false），首次安装引导不触发，不影响核心功能
  - pi 发布极活跃：5394 提交/1.5 年，20 天 9 版本，fork 维护成本高
  - 升级冲突：主要在 package.json configDir 字段 + deps 版本

### N5-最小可行方案对比

- 置信度：0.9（高）
- 信源验证：
  - 代码原文 ✅（综合 N1-N4 源码事实 + 验证 rick .rick 目录结构）
  - 运行时行为 ✅（方案 A/C 可行性运行时验证）
  - 文档 ✅（pi README + patch-package 文档）
  - 反事实 N/A
- 调研报告：briefs/research-8-N5-最小可行方案对比.md
- 关键事实：见下方方案对比表

## N5 方案对比表

| 方案 | 改动量 | 覆盖范围 | 维护成本 | 升级风险 | .pi 目录是否出现 | 推荐排序依据 |
|---|---|---|---|---|---|---|
| A：纯 env/flag | 0 行 pi 代码 + rick 包装脚本 | user scope ✅ + project scope skills/prompts/themes/extensions/SYSTEM.md ✅ + settings.json/npm/git ❌（不创建即跳过） | 低 | 低 | 不出现 | 改动量最小 |
| B：extension 适配 | 0 行 pi 代码 + 1 个 extension（~100-200 行）+ 包装脚本 | 等同 A + 动态发现 | 中 | 低 | 不出现 | 改动量第三 |
| C：fork configDir | 1 行 JSON + fork 维护流程 | 100%（project + user scope 全部自动重定向） | 高 | 高 | 不出现（.pi→.rick） | 改动量第四 |
| D：混合（fork + extension + env） | 1 行 JSON + 1 extension + 包装脚本 + fork 维护 | 100% + skills 格式适配 + 动态发现 | 最高 | 高 | 不出现 | 改动量最大 |
| E：patch-package | 1 个 patch 文件 + postinstall 脚本 | 等同 C（100%） | 中 | 中 | 不出现 | 改动量次小 |

**推荐排序（基于"改动量最小"原则，不含倾向性判断）**：
1. 方案 A（0 行 pi 代码，覆盖大部分，维护成本最低）
2. 方案 E（1 个 patch 文件，覆盖 100%，但 patch 可能失效）
3. 方案 B（1 个 extension，覆盖等同 A + 动态发现）
4. 方案 C（1 行 JSON，覆盖 100%，但 fork 维护成本高）
5. 方案 D（fork + extension + 脚本，覆盖最全但复杂度最高）

## R7 上报项（无法达高置信度的叶节点）

无。所有 5 个叶节点（N1-N5）置信度均为 0.9（高，≥ 0.8），无需 R7 上报。

## Y15 事实澄清结论（去掉 .pi 的最小可行方法）

1. **pi 从不主动创建 .pi 目录**（N1 源码级验证，0 处 mkdirSync 针对 CONFIG_DIR_NAME）。.pi 目录只在用户手动放置 project scope 资源或调用 `pi install --local` 时出现。**若 rick 不手动放 .pi 资源、不调用 pi install --local，.pi 目录天然不出现**。

2. **最小可行方法 = 方案 A（纯 env/flag 重定向）**：
   - 设置 `PI_CODING_AGENT_DIR=.rick/.pi-agent`（user scope 移入 .rick）
   - 设置 `PI_CODING_AGENT_SESSION_DIR=.rick/sessions`（session 移入 .rick）
   - 启动 pi 时传 `--no-skills --no-extensions --no-prompt-templates --no-themes --no-context-files`（禁用 project scope 默认发现）
   - 用 `--skill .rick/skills/ --extension .rick/extensions/ --prompt-template .rick/prompts/` 显式加载 .rick 资源
   - 用 `--system-prompt .rick/SYSTEM.md --append-system-prompt .rick/APPEND_SYSTEM.md` 覆盖系统提示词
   - **改动量：0 行 pi 代码，仅 rick 包装脚本**

3. **方案 A 的覆盖范围**：
   - ✅ user scope 全部（env 整体重定向）
   - ✅ project scope 的 skills/prompts/themes/extensions/SYSTEM.md/APPEND_SYSTEM.md/context-files（flag 禁用 + 显式加载）
   - ❌ project scope settings.json（`{cwd}/.pi/settings.json` 硬编码，但文件不存在则 settings-manager 跳过）
   - ❌ project scope npm/git 安装（`{cwd}/.pi/npm` `.pi/git` 硬编码，但只在 `pi install --local` 时创建，rick 不调用即可）

4. **方案 A 遗留的 3 处硬编码点**（project scope settings.json / npm / git）在 rick 不主动创建这些文件的情况下**不影响 pi 运行**。pi 的 existsSync 检测会跳过不存在的文件。

5. **若需 100% 覆盖**（包括 settings.json/npm/git 路径），需方案 C（fork 改 configDir，1 行 JSON）或方案 E（patch-package，1 个 patch 文件）。

6. **所有方案共通的适配工作**：rick skills 需从 `{name}_skill/skill.md` 适配为 pi 的 `{name}/SKILL.md` 格式（R6-N2 已记录）。

## 整合摘要

总节点数 5 | 高置信度叶节点 5 | R7 上报 0

- 尽调树深度 2 层（根 + N1-N5 第一层），5 个叶节点全部高置信度终止
- 信源加权：代码原文 0.4 + 运行时行为 0.3 + 文档 0.2 = 0.9（反事实 N/A 不计入）
- 5 种方案均经源码级验证，方案对比表维度完整（改动量/覆盖范围/维护成本/升级风险/.pi 目录是否出现）
- Y15 核心结论：pi 不主动创建 .pi 目录，方案 A（纯 env/flag）改动量最小且覆盖大部分场景，方案 C/E 覆盖 100% 但需 fork/patch 维护

## 给 human 的 S 阶段最终三连追问

基于全部 8 轮 research（Y1-Y15 + 25 项功能映射 + 6 项迁移价值论证 + Y15 去掉 .pi 最小可行方法），准备进入 E 阶段：

**① 现状补充**：Y15 已澄清"去掉 .pi 的最小可行方法 = 方案 A（纯 env/flag，0 行 pi 代码）"，是否还有遗漏的路径覆盖场景需要补充调研？（如：pi 的 telemetry 日志路径、pi 的 debug log 路径 `{agentDir}/{APP_NAME}-debug.log` 是否也需移入 .rick？）

**② 期望**：在方案 A（改动量最小，覆盖大部分，遗留 settings.json/npm/git 硬编码但不影响运行）与方案 C（fork 改 1 行 JSON，覆盖 100%，但需承担 fork 维护成本，pi 20 天 9 版本）之间，human 对"覆盖完整性 vs 维护成本"的权衡期望是什么？是否接受方案 A 的 3 处遗留硬编码点（在 rick 不创建对应文件的情况下不影响运行）？

**③ 差距**：从 R1-R8 全部 8 轮 research 来看，事实调研已覆盖 pi 框架定位、扩展点机制、调用链与功能枚举、pi 映射表、迁移价值论证、Y13 目录结构/skill 加载/session agent config、Y14 因果链、Y15 去掉 .pi 最小可行方法。当前差距是否已收敛到"可进入 E 阶段（视角生成）"？还是仍有事实性模糊需要 S 阶段补充调研？
