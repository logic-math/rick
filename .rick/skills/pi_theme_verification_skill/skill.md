# skill:pi-theme-verification（pi 主题验证与定制）

## 触发场景

当需要验证/定制 pi 主题时使用：
- 从 npm 找 pi 主题包，需要确认包**实际提供哪些主题名**（不能只看描述）
- 需要确认 pi 能否发现某个主题（themes/ 目录结构 vs package.json 的 pi.themes 声明）
- 需要读懂/修改主题文件的 51 个 color token（哪个 token 控制哪个 UI 元素）
- 修改 rick 内置主题（go:embed）后重新激活

**问题信号**：「主题切换后没生效」「两个主题颜色共存」「列表里有但 pi 不认」「改完主题没变化」

## 预期效果

- 一次确认主题名/发现机制，不靠猜
- 精确定位"哪个 token 控制哪个 UI 元素"（工具标题/命令/标题/链接/背景）
- 防止"改主题→忘重建→旧 embed 覆盖新文件"的静默回退

## 核心内容

### 1. 确认 npm 包提供的主题名（不看描述，看文件）

```bash
cd /tmp && npm pack <pkg> --silent | tail -1 | xargs -I{} tar -tzf {} | grep -iE "themes/.*\.json"
# 主题文件也可能在包根目录（配 pi.themes 声明）：
npm pack <pkg> --silent | tail -1 | xargs -I{} tar -xzf {} -C . && cat package/package.json
# 看 "pi": {"themes": [...]} 字段——这是 pi 真正加载的清单
```

### 2. pi 主题发现机制（源码验证过的结论）

| 来源 | 规则 |
|---|---|
| 包内 `themes/` 目录 | **递归**收集所有 `*.json`（子目录也认，如 themes/pi/） |
| package.json `pi.themes` | manifest 显式声明（如 jellybeans 在包根目录） |
| `~/.rick/pi/agent/themes/*.json` | 全局自定义主题目录（隔离后跟随 agent dir） |

### 3. 主题文件结构：vars（调色板）+ colors（51 token 映射）

```json
{"name": "rick", "vars": {"gold": "#eac54f"}, "colors": {"mdHeading": "gold"}}
```
- `colors` 的 51 个 token 是**引用**：值可以是 `vars` 键名、`#hex`、256 色数字、或空串（用默认）
- 解析引用要**递归查 vars**（token → vars 键 → 可能还是引用）
- 关键 token 对照：`toolTitle` 工具标题 / `bashMode` 命令文本+实时块边框 / `mdHeading` markdown 标题 / `mdLink` 链接 / `muted` 次级文本（bash 输出、引用、注释共用）/ `toolPendingBg|toolSuccessBg|toolErrorBg` 工具块背景 / `text` 正式回复

### 4. 验证"token 到底控制哪个元素"——查 pi 组件源码

```bash
grep -rn "toolTitle\|bashMode" ~/.local/lib/node_modules/@earendil-works/pi-coding-agent/dist/modes/interactive/components/*.js | grep -v ".map"
```
限制速查：bash 输出= `theme.fg("muted", ...)`（**无法单独改色**，只能改 muted）；实时执行块背景= 无（透明，pi 组件硬编码）；历史工具块背景= toolPendingBg 三兄弟。

### 5. 修改 rick 内置主题（go:embed）后必须重建（防回退陷阱）

```bash
go build -o bin/rick ./cmd/rick && ./bin/rick tools theme <name>
```
⚠️ 嵌入的主题文件**不重建就激活 = 旧 embed 覆盖磁盘新文件**（rick tools theme 会重写主题文件），表现是"改了没生效"。验证安装副本：
```bash
python3 -c "import json; print(json.load(open('$HOME/.rick/pi/agent/themes/rick.json'))['colors']['mdHeading'])"
```

### 6. 主题试色/切换工作流（job_34 沉淀）

用户要"试试别的配色"时的标准流程（**目标：10 分钟内完成一轮试色**）：

```bash
# ① 直接改运行时激活副本 —— pi 对 custom theme 文件热重载，运行中会话立即刷新
vim ~/.rick/pi/agent/themes/rick.json
# ② 满意后同步仓库源头 + 重建（go:embed 陷阱，见第 5 节）
cp ~/.rick/pi/agent/themes/rick.json internal/cmd/themes/rick.json
go build -o bin/rick ./cmd/rick && ./bin/rick tools theme rick && ./scripts/install.sh
```

**试色只改 vars**（调色板），colors 的 51 个 token 映射保持不动——换方案 = 换一组 vars 值。经典方案 vars 速查（job_34 已实践）：

| 方案 | bg | accent/强调 | 语法风格 |
|------|----|------------|---------|
| 极客绿(Matrix) | `#070a07` | `#00ff41` 霓虹绿 | keyword 红 / string 绿 / number 青 |
| Dracula | `#282a36` | `#ff79c6` 粉 | keyword 粉 / string 黄 / number 紫 / type 青 |
| One Dark | `#282c34` | `#61afef` 蓝 | keyword 紫 / string 绿 / number 橙 |
| Tokyo Night | `#1a1b26` | `#7aa2f7` 蓝 | keyword 粉红 / string 绿 / operator 青 |
| VSCode Dark+ | `#1e1e1e` | `#007acc` | keyword 紫 / string 橙 / comment 绿 `#6a9955` |

**展示预览**（让用户判断）用 python 24-bit ANSI 按实际 token 渲染模拟场景（命令块/diff/markdown/语法），比贴 JSON 直观得多。

### 7. 边界：渲染行为不可主题化（job_34 核心结论）

**主题只有 51 个颜色 token，只能决定"哪个元素用什么颜色"，管不了"渲染行为"**：

| 想要的效果 | 属于 | 主题能改吗 |
|-----------|------|-----------|
| 命令块/工具标题颜色、链接色、语法色 | 颜色 | ✅ 改 token |
| diff 变更词反显（`\x1b[7m` 背景红/绿块） | 渲染行为（pi 写死在 diff.js `theme.inverse`） | ❌ |
| diff 内容语法高亮、命令行语法高亮 | 渲染行为（需调用 highlight.js） | ❌ |

改渲染行为 = 改 pi 运行时代码（patch dist JS）——**job_34 用户决策：不引入**（破坏运行时副本的做法留待后续）。确认边界后再承诺需求，别把"主题能改"说成能改一切。
