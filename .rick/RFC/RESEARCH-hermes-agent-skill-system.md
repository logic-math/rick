# Hermes Agent Skill 系统技术调研报告

> 调研日期：2026-04-13
> 调研对象：NousResearch/hermes-agent
> 调研目标：理解 skill 自动生成机制，评估对 Rick 的参考价值

---

## 一、背景与定位

Hermes Agent 是 NousResearch（Hermes 系列模型的开发团队）于 2026 年 2-3 月发布的开源 AI Agent 框架。

- GitHub Stars：~28k（上线约 1 个月）
- 开源协议：MIT
- 核心定位："The agent that grows with you"——与用户共同成长的 Agent
- 前身：OpenClaw（提供一键迁移工具）

**关键背景**：这个框架是模型训练团队自己用的工具，内置了 Atropos RL 环境、批量轨迹生成、轨迹压缩等研究向功能，用于训练下一代工具调用模型。这意味着 skill 机制的设计目标不只是"好用"，还要"能产生训练数据"。

---

## 二、Skill 触发时机

### 触发条件（满足任一即触发）

来源：`agent/prompt_builder.py` 中的 `SKILLS_GUIDANCE` 常量（阿里云开发者社区文章直接引用源码）：

```python
SKILLS_GUIDANCE = (
    "After completing a complex task (5+ tool calls), fixing a tricky error, "
    "or discovering a non-trivial workflow, save the approach as a "
    "skill with skill_manage so you can reuse it next time.\n"
    "When using a skill and finding it outdated, incomplete, or wrong, "
    "patch it immediately with skill_manage(action=..."
)
```

| 触发条件 | 说明 |
|---------|------|
| 工具调用 ≥ 5 次 | 认为是"复杂任务" |
| 棘手错误后成功恢复 | "fixing a tricky error" |
| 发现非平凡工作流 | 走了一条不明显但有效的路径 |
| 用户主动纠正 | Agent 做法被用户纠正后 |

### 定期复盘（来源：CSDN，可信度中）

每完成约 15 个任务，Agent 自动复盘：检查哪些 Skill 可以合并、优化或删除冗余步骤。

### 设计哲学

System Prompt 不只是"允许"使用 skill，而是**强制规范**：
- 任务前：先扫描可用 skill
- 命中则：加载完整内容
- 复杂任务后：主动保存
- 发现问题时：立刻 patch

这使得 skill 维护是被系统行为规范**持续强化**的行为，而非偶然行为。

---

## 三、Skill 生成逻辑

### 生成流程

```
失败/复杂执行
    → LLM 自主判断触发条件
    → 调用 skill_manage(action="create")
    → 安全扫描
    → 原子写入磁盘（临时文件 + os.replace()）
    → 清除系统提示缓存
    → 立即生效
```

### LLM 的参与方式

LLM **直接生成** Skill 的全部内容。没有专门的"生成 skill 的 prompt"，而是通过 System Prompt 中的 `SKILLS_GUIDANCE` 指令，让 LLM 在适当时机自主决策调用 `skill_manage` 工具。

LLM 生成的内容包括：
- 执行步骤（Procedure）
- 已知陷阱（Pitfalls）
- 验证方法（Verification）

### `_create_skill` 内部实现步骤

1. 验证 name 格式（小写字母、数字、连字符，最长 64 字符）
2. 验证 frontmatter 结构（必须包含 `name` 和 `description` 字段）
3. 检查是否已存在同名技能（跨本地和外部目录）
4. 创建目录
5. **原子写入**：使用临时文件 + `os.replace()` 防止写入中断导致损坏
6. 执行安全扫描（与从 Hub 安装技能相同的安全扫描器）
7. 清除系统提示缓存，使新技能立即生效

---

## 四、Skill 存储结构

### 存储位置

- 用户自动生成：`~/.hermes/skills/`
- 内置技能：项目源码 `skills/` 目录（内置约 105 个）
- 外部目录：通过 `config.yaml` 的 `external_dirs` 字段配置（只读，本地版本优先）

### 目录结构（agentskills.io 官方规范）

```
skill-name/
├── SKILL.md          # 必需：元数据 + 指令
├── scripts/          # 可选：可执行脚本（Python/Bash/JS）
├── references/       # 可选：技术参考文档
├── assets/           # 可选：模板、资源
└── ...               # 其他支持文件
```

### SKILL.md 格式

```markdown
---
name: skill-name           # 必须：1-64字符，小写+数字+连字符
description: ...           # 必须：1-1024字符，描述功能和使用场景
license: Apache-2.0        # 可选
compatibility: ...         # 可选：环境要求说明（最长500字符）
metadata:                  # 可选：任意键值对
  author: example-org
  version: "1.0"
allowed-tools: Bash(git:*) Read   # 可选（实验性）：预批准工具
---

# 技能标题

## When to Use
...

## Procedure
（步骤说明）

## Pitfalls
（已知陷阱）

## Verification
（验证方法）
```

Hermes 私有扩展字段（不在官方规范中）：
- `metadata.hermes.tags`：标签
- `fallback_for_toolsets`：当某工具集不可用时才显示该 skill
- `requires_toolsets`：条件激活逻辑
- `config`：配置参数

---

## 五、Skill 复用机制

### 渐进式加载（Progressive Disclosure）

三层按需加载，最小化 token 消耗：

| Level | 工具调用 | 内容 | token 量 | 加载时机 |
|-------|---------|------|---------|---------|
| 0 | `skills_list()` | 所有 skill 的 name + description | ~3k tokens 总量 | 每次会话开始 |
| 1 | `skill_view(name)` | 特定 skill 的完整 SKILL.md | 按需 | Agent 判断匹配时 |
| 2 | `skill_view(name, path)` | skill 内部特定文件 | 按需 | 需要具体资源时 |

### 检索触发时机

System Prompt 要求 Agent 在收到用户任务后：
1. 先调用 `skills_list()` 扫描所有 skill 的 description
2. 如果 description 与当前任务语义匹配，主动调用 `skill_view(name)` 加载完整内容
3. 匹配方式：依赖 LLM 语义理解（无向量搜索，无 BM25）

---

## 六、Skill 优化机制

### 持续 patch 机制

当 Agent 使用某个 skill 时发现问题，System Prompt 要求立即调用：

```
skill_manage(action="patch", old_string=..., new_string=...)
```

**patch 的设计特点：**
- 只传入旧字符串和替换字符串，只修改变化的部分
- 内置 `fuzzy_find_and_replace` 引擎：容许轻微空白符/缩进差异
- 匹配失败时返回文件前 500 字符预览，供 Agent 自我纠正，无需人工介入

**为什么用 patch 而不是全量 edit：**
- 正确性：全量重写有风险破坏已正常工作的部分
- 效率：patch 的 token 消耗远低于全量重写

### `skill_manage` 完整操作列表

| 操作 | 说明 |
|------|------|
| `create` | 新建 skill（参数：name, content, 可选 category） |
| `patch` | 目标修复（old_string → new_string） |
| `edit` | 结构重写（完整 SKILL.md 替换，不推荐日常使用） |
| `delete` | 删除 skill |
| `write_file` | 写入 skill 内的支持文件 |
| `remove_file` | 删除 skill 内的支持文件 |

---

## 七、agentskills.io 开放标准

- **发起方**：Anthropic
- **首发**：2025-10-16（Claude 内部），2025-12-18（开放标准正式发布）
- **官方地址**：https://agentskills.io/specification
- **GitHub**：https://github.com/agentskills/agentskills
- **验证工具**：`skills-ref validate ./my-skill`

**已兼容平台（30+）**：Claude Code、OpenClaw、OpenAI Codex CLI、Gemini CLI、Cursor、VS Code、GitHub Copilot、Windsurf 等。

**渐进式披露三阶段：**
1. 元数据阶段（~100 tokens）：name + description，启动时加载所有 skill 的元数据
2. 指令阶段（< 5000 tokens 推荐）：完整 SKILL.md 正文，激活时才加载
3. 资源阶段（按需）：scripts/、references/、assets/ 下的文件，需要时才读取

---

## 八、四层记忆架构

Skill 是 Hermes 四层记忆体系中的一层：

| 层级 | 存储位置 | 内容 | 加载时机 |
|------|---------|------|---------|
| MEMORY.md | `~/.hermes/memories/MEMORY.md`（~800 tokens） | 环境配置、约定、经验教训 | 每次会话启动时注入 |
| USER.md | `~/.hermes/memories/USER.md`（~500 tokens） | 用户偏好、沟通风格 | 每次会话启动时注入 |
| Skills | `~/.hermes/skills/` | 程序性知识（怎么做） | 按需（Level 0-2 渐进式） |
| Session Search | `~/.hermes/state.db`（SQLite + FTS5） | 全部历史会话 | 按需检索，LLM 摘要后注入 |

三者分工：
- MEMORY.md = "用户喜欢什么"
- Session Search = "之前怎么处理过类似问题"
- Skills = "以后遇到这种任务怎么标准化处理"

---

## 九、源码目录结构（部分确认）

```
hermes/
├── agent/
│   ├── prompt_builder.py      # 包含 SKILLS_GUIDANCE 常量（确认）
│   ├── memory_manager.py      # 记忆管理器（确认类名）
│   └── context_compressor.py  # 上下文压缩（确认类名）
└── tools/
    └── skills_tool.py         # Skill 系统实现（确认文件名）
        # 包含：_create_skill、fuzzy_find_and_replace
        # 暴露工具：skills_list、skill_view、skill_manage
```

---

## 十、尚未确认的问题

以下几点未能从公开资料中直接确认，属于推测：

1. `skills_list()` 的匹配算法：是纯 LLM 语义判断，还是有 keyword 预过滤？
2. "每 15 个任务复盘一次"的具体实现：计数器触发？还是 cron？
3. Session Search 与 Skill 之间是否有"合并提升"逻辑（发现历史中的好模式，自动生成 skill）？
4. 安全扫描的具体实现（检查哪些类型的威胁）。
5. 内置 105 个技能的来源和更新机制。

---

## 十一、信息来源

| 来源 | 可信度 | 内容 |
|------|--------|------|
| 官方文档 hermes-agent.nousresearch.com/docs | 高 | 功能描述、存储结构 |
| agentskills.io/specification | 高 | SKILL.md 格式规范 |
| 阿里云开发者社区（含源码片段） | 高 | `prompt_builder.py` 源码、`_create_skill` 实现 |
| CSDN liyou125 | 中 | 定期复盘机制、四层记忆 |
| 36kr、今日头条、腾讯网等技术文章 | 中 | 功能描述（多源印证） |
