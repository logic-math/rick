---
name: import-ctx
description: 上下文继承 Loop；将父级 .rick 的 loops/skills 迁移到当前项目，适配环境后生成候选文件
---

## Step 0：触发条件

仅在 `--ctx` 模式下触发：本次会话继承了父级 .rick 上下文 `{{ctx_path}}`。

在开始处理用户需求之前，先启动一个子 Agent 完成上下文初始化。

---

## 全局目标

将父级 `{{ctx_path}}` 中的 loops/skills 知识迁移到当前项目（`{{local_rick_dir}}`），生成适配当前环境的候选文件，供人工审核后正式使用。

**成功标准**：
- 父级 loops/ 中有价值的 loop 文件已生成对应候选文件
- 父级 skills/ 中有价值的 skill 文件已生成对应候选文件
- 所有环境强相关内容已替换为 TODO 占位符

---

## 上下文管理

**读取**：
- `{{ctx_path}}/loops/` — 所有 loop 文件（含 frontmatter: name/trigger/scope）
- `{{ctx_path}}/skills/` — 所有 skill 文件

**写入**：
- `{{local_rick_dir}}/loops/candidate_loop_*.md` — 适配当前项目的候选 loop
- `{{local_rick_dir}}/skills/candidate_{name}_skill/skill.md` — 适配当前项目的候选 skill

人工确认候选目录后，去掉 `candidate_` 前缀即为正式目录（如 `candidate_go_build_skill/` → `go_build_skill/`）。

---

## 子 Agent 工作流

```
[READ_PARENT] → [FILTER] → [ADAPT] → [WRITE_CANDIDATES]
```

**READ_PARENT（读取父级上下文）**
- 逐一读取 `{{ctx_path}}/loops/*.md` 和 `{{ctx_path}}/skills/*.md`
- 提取每个文件的 trigger/scope/功能描述，判断与当前项目的相关性

**FILTER（过滤相关内容）**
- 保留：架构知识、流程经验、通用工作流
- 跳过：高度项目特定、与当前需求无关的内容

**ADAPT（适配当前环境）**

⚠️ 以下内容**不得直接复制**，必须替换：

| 内容类型 | 处理方式 |
|----------|----------|
| IP 地址、域名、端口 | 替换为 `TODO: 填写当前环境地址` |
| 密码、密钥、Token | 替换为 `TODO: 从当前环境获取` |
| 文件系统路径 | 替换为 `TODO: 确认当前路径` |
| 服务账号、用户名 | 替换为 `TODO: 确认当前账号` |

**WRITE_CANDIDATES（写入候选文件）**
- 每个保留的 loop → `{{local_rick_dir}}/loops/candidate_loop_{name}.md`
- 每个保留的 skill → `{{local_rick_dir}}/skills/candidate_{name}_skill/skill.md`

---

## 产出评估

| 检查项 | 判断方法 |
|--------|----------|
| 候选文件已创建 | `ls {{local_rick_dir}}/loops/candidate_*.md {{local_rick_dir}}/skills/candidate_*.md` |
| 环境隔离 | 文件内无裸露 IP/路径/密钥，仅含 TODO 占位符 |
| frontmatter 完整 | 每个候选文件含 name/trigger/description 字段 |

---

## 停止标准

**完成退出**：所有父级文件已遍历，候选文件已写入。

**原则**：迁移的是架构知识和流程经验，不是环境配置。保持环境隔离。

---

## 知识查询规则

会话过程中遇到模糊信息时：
1. 先查当前 `.rick/loops/` 和 `.rick/skills/`（已迁移的上下文）
2. 若无答案，查父级 ctx：`{{ctx_path}}`
3. 找到后适配当前环境使用
