APPROVED: true

# Job job_34 执行总结

## 执行概述

**项目目标**: ① rick 主题配色定制（命令块命令行绿色 → 后续迭代为 md 链接/路径蓝色 → 多方案对比最终定为 VSCode Dark+）；② 默认模型改为 deepseek-v4 pro；③ 用户尝试期间提出 diff 反显背景问题 → 探索运行时 patch 方案后被否决（破坏运行时不引入，只留主题配置）；④ 自闭环 pi 运行时隔离（~/.rick/pi 必须保留）落地。
**实际完成**: 全部完成 —— 主题体系迭代 6 轮（命令绿/链接蓝 → 极客绿 → Dracula → One Dark → Tokyo Night → VSCode Dark+ 定稿）；默认模型 deepseek-v4-pro；自闭环 pi 运行时（npm --prefix 独立副本 + 全链路托管优先解析）；运行时 patch 功能完整实现后按用户决策撤销（运行时副本恢复 stock）；版本 3.1.5；已推送远程。
**整体评价**: ⭐⭐⭐⭐⭐ (5/5 —— 6 轮主题方案快速试色闭环、复杂需求（隔离/撤销）理解准确、测试隔离陷阱发现并根治、最终自闭环交付)

## 关键成就

1. **主题试色工作流（6 轮 10 分钟级闭环）**: 建立"改 vars 换方案 + 热重载 + ANSI 模拟场景预览 + 满意后同步仓库重建"的标准流程；沉淀 5 套经典方案 vars 速查（极客绿/Dracula/One Dark/Tokyo Night/VSCode Dark+）到 theme skill。用户从"命令绿"一路试到 Dark+，每次都给可视预览判断，需求闭环极快。
2. **pi 渲染行为边界确认**: 源码级验证 pi 只有 51 个颜色 token、渲染行为（diff 反显 `\x1b[7m`、语法高亮）不可主题化——避免向用户承诺主题做不到的事；patch 方案（inverse→bold、diff/命令语法高亮）完整实现并验证后按用户决策整体撤销，运行时副本恢复 stock，全局 pi 全程零污染。
3. **自闭环 pi 运行时（~/.rick/pi 隔离保留）**: `npm install --prefix ~/.rick/pi/agent/runtime` 独立副本 + `cfg.PiPath → RuntimeBin() → PATH` 全链路解析优先级；版本匹配全局、pinned 失败降级 latest；与全局/独立 pi 完全隔离。
4. **测试隔离陷阱根治**: piCommand/FindBinary 托管优先后，PATH-fake 测试静默命中真实 pi 导致全套件 10 分钟超时——用 HOME/RICK_PI_AGENT_DIR 隔离修复，测试从 265s+ 降到 5s 全绿；沉淀精确解决步骤到 subprocess_env_isolation skill 与 bugs.md。

## 问题与教训

### 问题1: 测试全量挂死 10 分钟（panic 无具体失败）

**根本原因**: 引入"托管运行时优先"解析后，`TestPiListContains`/`TestVerifyExtensions`/plan·learning workflow mock 测试仍只设 PATH fake，真实托管 pi 优先命中（甚至拉起交互会话等输入）→ 全套件超时。
**解决方案**: 这些测试补 `t.Setenv("RICK_PI_AGENT_DIR"/"HOME", t.TempDir())` 隔离；plan/learning workflow 测试的 mock 配置原本写到共享 `$TMPDIR/.rick/config.json`（从不被读到），改为写入隔离 `$HOME/.rick/config.json` 并显式 `pi_path`。
**经验教训**: 改解析优先级 = 全量审计依赖旧解析的测试；超时 panic 的 "running tests:" 是定位利器；反复整跑 6 分钟套件是调试大敌，用 `-run` 过滤 + 短 `-timeout`。

### 问题2: patch 方案的反复（下划线 → 加粗 → 整体撤销）

**根本原因**: 最初用下划线替代反显被否 → 改加粗 → 用户进一步决定"破坏运行时的做法不引入，只留主题配置"（且明确 ~/.rick/pi 隔离必须保留）。
**解决方案**: 完整实现加粗 + 语法高亮（清单式幂等 patch）后，按决策整体撤销：删除命令/代码/文档，运行时副本用全局 stock 版逐字节还原，保留隔离基础设施；沉淀"渲染行为不可主题化"边界到 domain/skills。
**经验教训**: 用户对"改运行时代码"有明确边界——先确认方案是否触碰运行时再动手；隔离基础设施（自闭环运行时）即使功能撤销也应保留，作为未来做 patch 的地基。

### 问题3: 幂等 patch 锚点陷阱 + Go raw string 嵌 JS 反引号

**根本原因**: helper 插入的锚点（函数签名）替换后仍存在 → 二次运行重复插入；Go raw string 无法包含反引号，复刻 JS 模板字符串 fixture 时截断报错。
**解决方案**: 整函数替换（锚点被消费）+ "raw 段+解释型段"拼接 fixture。
**经验教训**: 字符串 patch 幂等性靠"old 被消费"保证；跨语言 fixture 复刻注意引号语义（详见 bugs.md）。

## 知识沉淀清单

- [x] skills/pi_theme_verification_skill/skill.md - 升级：主题试色工作流 + 5 套经典方案 vars 速查 + 渲染行为不可主题化边界
- [x] skills/pi_runtime_verification_skill/skill.md - 升级：自闭环运行时验证（安装/优先级/版本匹配/测试隔离）
- [x] skills/subprocess_env_isolation_skill/skill.md - 升级：托管二进制优先解析陷阱 + 幂等 patch 锚点陷阱
- [x] loops/ - 无新 loop（主题迭代=线性模式，job_33 已定论；patch→撤销=一次性决策，非循环模式）
- [x] domain/bugs.md - 追加 3 条：托管优先命中真实 pi / raw string 嵌反引号 / 幂等锚点陷阱
- [x] domain/pi-runtime.md - 追加：自闭环运行时事实 + 渲染行为 vs 主题 token 边界
- [x] domain/env.md - 追加：pi 托管目录结构 + 主题最终状态
