修改一下 rick 主题配色，命令块中的命令行改成绿色，默认模型改成 deepseek-v4 pro 
---

## Grilling 澄清结论（2026-08-12，job_34）

**原始需求**：修改 rick 主题配色：命令块中的命令行改成绿色；默认模型改成 deepseek-v4 pro。

**Q1（命令块中的命令行 = 哪个 token）**：pi TUI 中 bash 命令块（工具执行块，背景 toolPendingBg/toolSuccessBg）的标题行 `$ <command>` 由 `toolTitle` 上色（源码 bash.js renderCall: `theme.fg("toolTitle", theme.bold("$ ..."))`）；`bashMode` 仅用于 bash 模式的实时输入行，不在"块"内。→ 结论：把 rick.json 的 `toolTitle` 从 gold 改为绿色。副作用：read/edit/write/ls/grep/find 等工具标题也变绿（pi 主题 token 无法按工具细分，这是唯一杠杆，与现有"工具标题统一配色"设计一致）。

**Q2（绿色取哪个色值）**：主题 vars 现有 `green`(#57ab5a) 与 `greenBright`(#6bc46d)。命令文本位于近黑背景 #0d1117 上，且 mdLink/syntaxString（链接/路径）已用 greenBright。→ 结论：用 `greenBright`（#6bc46d），亮且与既有绿色元素一致。

**Q3（默认模型改哪里）**：pi 对 deepseek provider 的默认模型本身就是 `deepseek-v4-pro`（model-resolver.js: `deepseek: "deepseek-v4-pro"`）；rick 的实际默认来自 `~/.rick/config.json` 的 `pi_extra_args`（当前显式 `--model deepseek-v4-flash` 覆盖了 pi 默认）。→ 结论：① `~/.rick/config.json` 的 `--model` 改为 `deepseek-v4-pro`（保持显式声明，不依赖 pi 隐式默认）；② README.md 配置示例 2 处 `deepseek-v4-flash` → `deepseek-v4-pro`（文档与默认一致）。

**Q4（改动范围）**：只改 rick 默认主题（rick.json），不动 gh-light / gh-dark-dimmed（GitHub Primer 风格主题，与 rick 主题无关）。→ 结论：仅 rick.json。

**实现要点**：
1. `internal/cmd/themes/rick.json`：`"toolTitle": "gold"` → `"toolTitle": "greenBright"`
2. `internal/cmd/tools_theme.go`：更新 rick 主题注释（"工具标题橙/bash 命令灰"已过时 → 工具标题/命令绿）
3. `~/.rick/config.json`：`--model deepseek-v4-flash` → `--model deepseek-v4-pro`
4. `README.md`：2 处配置示例同步为 deepseek-v4-pro
5. 重建 `bin/rick` 并重新激活主题（go:embed 陷阱：不重建会旧 embed 覆盖磁盘主题文件）

---

## Grilling 追问 2（2026-08-12，job_34，diff 高亮反馈迭代）

**用户反馈**：① 命令块/编辑 diff 里关键字被反显成红/绿背景块不符合预期；② 下划线方案也不要；③ 不要全局共享生效，只 ~/.rick/pi 独立生效。

**Q1（高亮样式）**：反显（背景块）被否、下划线被否 → 最终方案：**加粗**（`theme.bold`）。行级红绿对比保留，变更词加粗提示，无背景、无下划线。
→ 结论：patch 替换 `theme.inverse(` → `theme.bold(`（regexp 匹配任意单参，diff.js 2 处）。

**Q2（生效范围）**：patch 的是 pi 包代码（dist JS），主题 token 管不到；要"只 rick 生效"必须让 rick 拥有独立的 pi 副本。
→ 结论：**运行时自闭环**——init-pi 用 `npm install --prefix ~/.rick/pi/agent/runtime @earendil-works/pi-coding-agent@<全局版本>`（全局有则匹配版本，无则 latest）安装 rick 自己的 pi；`FindBinary`/`piPathOrDefault`/`piCommand`/Executor 默认优先托管运行时（cfg.PiPath 仍最高优先）；patch-pi-diff 只作用于托管副本；全局/独立 pi 完全不受影响（本机全局已恢复原样，实测验证）。

**Q3（幂等/健壮）**：patch 检测"无 theme.inverse( 则 no-op"（已 patch 或上游变更都不报错）；pinned 版本安装失败自动降级 latest。
→ 结论：均已实现并有单测覆盖。

**实现要点**：
1. `piagent/agentdir.go`：新增 `RuntimeDir()`/`RuntimeBin()`/`FileExists()`（runtime 在 agent dir 下，测试可用 RICK_PI_AGENT_DIR 隔离）
2. `piagent/cli.go` FindBinary/piPathOrDefault、`executor.go` piBin 默认：cfg.PiPath → 托管运行时 → PATH
3. `tools_init_pi.go`：ensurePI 优先托管、installManagedPI（npm --prefix，pinned 失败降级 latest）、piCommand 优先托管；删除旧 curl installPI
4. `tools_patch_pi_diff.go`：`theme.inverse(` → `theme.bold(`；检测逻辑简化为"无 inverse 则 no-op"
5. 全局 pi 恢复原样（sed 还原 theme.inverse）
6. README/commands.md 同步（加粗 + 自闭环）

---

## Grilling 追问 4（2026-08-12，job_34，撤销 patch 功能）

**用户决策**：删除 patch 功能（`rick tools patch-pi` / `patch-pi-diff`）——"破坏运行时的做法我们不需要现在引入，后面可能做到；只用基本的主题配置功能"。**但运行时自闭环保留**："运行时自闭环不需要撤出，~/.rick/pi 这个隔离逻辑一定要有"。

**Q1（patch 功能去留）**：→ 删除。撤销 diff 反显→加粗、diff 语法高亮、bash 命令语法高亮三个运行时代码级修改；托管运行时副本恢复为 stock（与全局 pi 逐字节一致，已验证）；删除 `tools_patch_pi_diff.go`/`_test.go`、tools.go 注册、init-pi Step 5、README/commands.md 的 patch 章节。

**Q2（运行时自闭环去留）**：→ **保留**。`~/.rick/pi` 隔离（agent 配置目录 + runtime 独立副本）是必须的基础设施：rick 的 pi 与全局/独立 pi 完全隔离、互不污染、可独立升级。保留 `RuntimeDir()`/`RuntimeBin()`/`FileExists()`、FindBinary/piPathOrDefault/piCommand/Executor 托管优先、ensurePI/installManagedPI。

**Q3（主题配置）**：UI 定制只走主题（`rick tools theme` + rick.json 51 token）——命令绿/链接蓝/路径蓝/标题金等已完成的主题改动全部保留。

**遗留**：pi 的 diff/命令渲染行为（反显高亮、语法高亮）主题 token 覆盖不到，只能改 pi 代码——已记录为"后续可能再做"（commands.md 设计权衡）。
