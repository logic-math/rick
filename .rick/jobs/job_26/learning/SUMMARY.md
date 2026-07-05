APPROVED: true

# Job job_26 执行总结

## 执行概述

**项目目标**: 为 human_loop 工作流增加持续学习基础设施——draft 目录、judgment.md 判断捕获、ZPD 评价引导、draft_dir 变量注入全链路打通。

**实际完成**: 4/4 tasks 全部 success，0 次重试，所有测试通过，doing_check pass。

**整体评价**: ⭐⭐⭐⭐⭐

## 关键成就

1. **draft 目录基础设施**: 新增 `.rick/draft/` 目录结构，`draft_dir` 变量贯通 think/express/learning 三个模板阶段。
2. **判断捕获闭环**: think 模板新增「判断记录协议」，每个 SENSE 阶段结束自动追加到 `judgment.md`；express 模板「第零步」读取并清洗这些判断，形成完整捕获→复用闭环。
3. **ZPD 显式评价**: express 第五步引导用户回答 3 个 ZPD 问题，产出写入 `progress.md` 和 `loops.md`，让每次会话都有可沉淀的学习元数据。
4. **learning 阶段打通 draft_dir**: learning.go 注入 `{{draft_dir}}`，learning 模板可引用 draft 产出做同步，实现 think→express→learning 数据流通。

## 问题与教训

### 问题1: Python 集成测试 subprocess 未隔离 HOME 环境变量

**根本原因**: task1 的测试脚本在 `subprocess.run()` 时未显式传 `env=` 参数，子进程继承真实 HOME，`LoadConfig()` 读取真实 `~/.rick/config.json`，触发真实 Claude CLI 调用，导致测试超时。

**解决方案**: `env = os.environ.copy(); env["HOME"] = work_dir` 后传给 subprocess。

**经验教训**: 编写 subprocess 集成测试时，凡是涉及配置文件读取，必须先建立隔离的测试 HOME 目录并显式透传，这是防止测试"污染"生产环境的基础隔离手段。

### 问题2: task4 代码探索冗余（Explore subagent + 直接 Read 重复）

**根本原因**: 先启动 Explore subagent，返回后又直接 Read 了相同文件，造成 45 次工具调用（其他 task 约 22-30 次）。

**解决方案**: 对于明确知道需要读取的目标文件（`learning.go`、`builder.go`、`paths.go`），直接并行 Read，不需要先用 Explore subagent 预探索。

**经验教训**: Explore subagent 适合"不知道文件在哪"的开放式探索，目标明确时直接 Read 更高效。

## 知识沉淀清单

- [x] skills/subprocess_env_isolation_skill/skill.md — subprocess 测试 HOME 隔离模式
- [x] loops/tdd-red-green-refactor-loop.md — 升级：新增 embed.FS 模板变更附加步骤
- [x] domain/bugs.md — 追加 subprocess HOME 未透传导致真实 Claude 调用的已知问题
- [x] domain/architecture.md — 追加 human_loop think/express 扩展事实 + draft_dir 注入说明
