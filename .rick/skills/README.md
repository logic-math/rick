# Skill 格式规范

skill.md 描述一个原子级能力单元，agent 在遇到触发条件时按需加载并执行一次。
格式参考 agentskills.io 标准（When to Use / Procedure / Pitfalls / Verification），
内容面向 agent 而非人类（步骤可直接执行，命令可直接复制）。

由 learning/dream 阶段产出候选（命名为 `candidate_skill_N.md`），人工审核后移入 `.rick/skills/`。

## Frontmatter 字段规范

```yaml
---
name: skill-name               # 必须：小写字母+数字+连字符，最长 64 字符
description: "触发场景：能做什么"  # 必须：供 skills_context 索引检索
---
```

## 正文四要素

### When to Use
遇到什么情况时加载本 skill（触发词或触发场景，要具体）。

### Procedure
分步骤的执行方法，每步包含：
- 具体命令（可直接执行）
- 预期输出（可量化断言）

### Pitfalls
已知坑点，含具体反例：
- ❌ 不要……（原因）
- ✅ 应该……

### Verification
如何确认 skill 执行成功：
- `[命令]` 输出 `[预期内容]` 即为成功
