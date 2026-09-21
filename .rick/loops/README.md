# Loops 格式规范

Loop 描述一个带评估机制的迭代控制流，供 agent 在需要反复执行直到收敛的场景中加载。

**Loop vs Skill 区别**：Skill 是静态上下文模块（执行一次）；Loop 是动态迭代控制流（执行直到收敛）。

## 目录结构

```
.rick/loops/
├── rick-rsi-loop.md                  # rick 自进化（RSI）：dev 隔离开发 → 门禁 → 人类确认 → release
├── agent-runtime-bootstrap-loop.md   # pi runtime 初始化/迁移/重装
├── go-refactor-migration-loop.md     # Go 大重构迁移（build+test 收敛到绿）
├── protocol-redesign-loop.md         # 多阶段协议重构
├── readme-wiki-sync-loop.md          # README/wiki 同步
├── tdd-red-green-refactor-loop.md    # TDD 迭代直到测试通过
└── deprecated/                       # 已淘汰（含 do-check-mark-success-loop）
```

> **硬规则：改 rick 自身必须走 `rick-rsi-loop`。**
> 任何修改 rick 源码/知识库并要让它生效到生产的迭代，都必须加载 `rick-rsi-loop`（RSI 会话类型会自动注入该 loop 全文）；
> 该 loop 的产出评估由 `rick tools rsi_check` 机器校验——缺证据即视为迭代未完成。

## Loop 文件格式（五要素）

```
---
name: {name}-loop
trigger: "当...时触发（具体场景）"
scope: "doing / easy / 全局"
---

## 依赖准备（硬约束，缺失则报错停止）
## 全局目标（成功标准）
## 上下文管理（保留/压缩/遗忘）
## 子 Agent 工作流（状态机：每轮一个子 Agent）
## 产出评估（验证 skill + 检查表）
## 停止标准（成功/失败/优雅退出）
```

**机器校验（必须通过）**：`rick tools loops_check`（默认校验 `<cwd>/.rick`，可用 `--dir` 指定）

- frontmatter 必需字段：`name`、`trigger`
- body 必需小节：`## 目标`、`## 上下文管理`、`## 可调用工具`、`## 产出评估`、`## 停止标准`
- `README.md` 跳过；`deprecated/` 目录不参与校验

> 说明：上文的「依赖准备 / 全局目标 / 子 Agent 工作流」是内容组织建议（推荐写法）；
> 校验器只强制上表列出的五个小节名。写 loop 时两者都要照顾到：
> 例如小节标题写成 `## 目标（Goal）`，既满足校验器又保留可读性。

## 淘汰标准

连续 3 次 dream 未被任何 job 触发的 loop → 移至 `deprecated/`。
