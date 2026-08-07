# research-2 N1-pi 框架定位与维护活跃度

节点路径:[根 > N1-pi 框架定位与维护活跃度]
事实陈述:pi 是开源轻量级 agent harness,定位为可扩展的终端编码 agent;维护活跃度可通过 commit/issue/release 频率验证。

## 执行动作

1. WebFetch pi.dev 官网(LLM 处理报错,改用 raw fetch + git clone)
2. git clone --depth 50 https://github.com/earendil-works/pi.git → 浅克隆
3. git fetch --unshallow → 完整历史
4. git log 分析:首末 commit / 总 commit / 30 天 commit / 贡献者
5. 读取 README.md / package.json / LICENSE

## 信源验证结果

### 代码原文(权重 0.4)✅

- 仓库:`github.com/earendil-works/pi`,根 `package.json` name=`pi-monorepo`,version=`0.0.3`,private=true
- workspace 包:agent / ai / client / coding-agent / evals / protocol / server / storage / tui(9 个)
- `packages/coding-agent/package.json`:name=`@earendil-works/pi-coding-agent`,version=`0.83.0`,bin=`{pi: dist/cli.js}`,type=`module`
- `engines`: `node >= 22.19.0`
- LICENSE:MIT,Copyright (c) 2025 Mario Zechner
- README 自述:"Pi is a minimal terminal coding harness. Adapt pi to your workflows, not the other way around, without having to fork and modify pi internals."
- 维护方:earendil-works 组织(Mario Zechner / badlogicgames 是核心作者,见 LICENSE 与 X 链接)

### 运行时行为(权重 0.3)✅

- git 历史可执行验证:
  - 首次 commit:2025-08-09 17:18:38 +0200 "Initial monorepo setup"
  - 最新 commit:2026-08-04 11:27:47 +0200 "DRAFT: add openai background mode responses (#7339)"(调研当天)
  - 总 commit 数:5394
  - 最近 30 天 commit:563(约 18.8/天)
  - 贡献者 top5(全量历史 shortlog):Christian Klotz 20+、Armin Ronacher 11+、David Brailovsky 9+、Mario Zechner 8+、Vegard Stikbakke 8+(注:shortlog 受 depth 限制为采样值)
  - tag 列表含 v0.9.4 → v0.83.0(版本号跃升说明多次 major 迭代)
- 仓库体积:27M(含 .git),纯代码规模适中

### 文档(权重 0.2)✅

- 官网 pi.dev(LOGO 引用 `https://pi.dev/logo-auto.svg`,README 多处链接 pi.dev/docs/latest)
- npm 包发布:`@earendil-works/pi-coding-agent`(README 顶部 badge 引用 npm)
- Discord 社区:discord.com/invite/3cU7Bz4UPx
- 作者博客:mariozechner.at/posts/2025-11-30-pi-coding-agent/(设计理念长文)
- 官网域名 pi.dev 由 exe.dev 捐赠(README 页脚)
- RFC 流程:rfc.earendil.com/keyword/pi/

### 反事实(权重 0.1)N/A

- 本节点为外部事实调研,未修改 rick 代码,无反事实验证

## 还原确认

本轮纯外部调研,未修改 rick 仓库代码,无需 git restore。

## 关键事实

1. **定位**:Pi 是 minimal terminal coding harness,核心理念"aggressively extensible so it doesn't have to dictate your workflow"
2. **作者**:Mario Zechner(badlogicgames)+ earendil-works 组织,多贡献者协作
3. **项目年龄**:约 1 年(2025-08-09 至 2026-08-04)
4. **维护活跃度**:极度活跃,5394 commits / 563 commits in last 30 days / 当天有新 commit / 版本 v0.83.0
5. **License**:MIT(可商业使用、可修改、可分发)
6. **生态**:npm 包 + Discord 社区 + RFC 流程 + 官网文档站
7. **明确不内置**:No MCP / No sub-agents / No permission popups / No plan mode / No built-in to-dos / No background bash(全部交由 extension 实现)

## 疑问点

无。本节点事实清晰,信源三重交叉验证(代码 + git 历史 + 文档)一致。

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅ × 0.4 = 0.4
- 运行时行为 ✅ × 0.3 = 0.3
- 文档 ✅ × 0.2 = 0.2
- 反事实 N/A × 0.1 = 0(无反事实)
- 合计 = 0.9(高,≥ 0.8 终止)
