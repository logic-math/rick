# Loops 格式规范

Loop 描述一个带评估机制的迭代控制流，供 agent 在需要反复执行直到收敛的场景中加载。

**Loop vs Skill 区别**：Skill 是静态上下文模块（执行一次）；Loop 是动态迭代控制流（执行直到收敛）。

## 目录结构

```
.rick/loops/
├── tdd-red-green-refactor-loop.md    # TDD 迭代直到测试通过
├── do-check-mark-success-loop.md     # doing_check 失败修复
└── deprecated/                       # 已淘汰（连续3次dream未被触发）
```

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

## 淘汰标准

连续 3 次 dream 未被任何 job 触发的 loop → 移至 `deprecated/`。
