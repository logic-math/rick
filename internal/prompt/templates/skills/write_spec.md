# skill:write_spec（沉淀知识到 loops/skills）

学习阶段或 dream 阶段发现了可复用的知识点，需要将其沉淀到 `.rick/loops/` 或 `.rick/skills/` 中时使用。

## 触发场景

- 发现跨 task/job 重复出现的工作流 → 创建或升级 loop
- 发现可复用的操作步骤或避坑经验 → 创建或升级 skill
- 现有 skill 描述不完整或有陷阱遗漏 → 升级已有 skill

## 判断：loop 还是 skill？

| 特征 | 选择 |
|------|------|
| 需要迭代直到收敛（有条件循环） | **loop** |
| 一次执行完成（静态步骤） | **skill** |
| 有明确触发条件和停止标准 | **loop** |
| 有触发场景和操作步骤 | **skill** |

## 创建 Skill

路径：`.rick/skills/{name}_skill/skill.md`

```markdown
# skill:{name}（一句话描述）

## 触发场景
什么情况下使用（具体，避免"调试时"这种模糊描述）

## 预期效果
可量化的结果

## 核心内容
可直接执行的步骤/命令，含具体示例
```

**命名规则**：目录以 `_skill` 结尾，如 `mark_task_success_skill/`。

## 创建/升级 Loop

路径：`.rick/loops/{name}-loop.md`

必须包含五个节：`## 目标（Goal）`、`## 可调用工具（Tool Access）`、`## 上下文管理（Context Management）`、`## 产出评估（Output Evaluation）`、`## 停止标准（Termination Condition）`。

参考 `.rick/loops/README.md` 中的完整格式。

## 升级优先于新建

升级已有 skill/loop（优先）：
1. 读取已有 skill.md，在"核心内容"中补充新步骤/陷阱
2. 更新触发场景（如有新场景）

新建（仅当无相似 skill/loop 时）：
1. 检查 `.rick/skills/` 中是否有触发场景 80% 重叠的 skill
2. 无相似 → 新建目录和 skill.md

## 质量标准

- skill.md 100-300 字（简洁但完整）
- 触发场景具体（能判断何时应用）
- 核心内容可操作（步骤含具体命令）
- 效果可观测（能判断是否生效）
