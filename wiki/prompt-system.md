# Rick 提示词系统

## 概述

提示词系统负责为每个命令生成结构化、上下文丰富的 pi 提示词。核心原则是**上下文优先**：把方法层（rick 方法描述）、技能层（skills）、实例层（任务实例）三层分离注入，使 pi 在充分理解项目背景的基础上执行。

## 三层注入模型

| 层 | 内容 | 注入通道 |
|----|------|----------|
| 方法 | system prompt（rick 方法描述 + 命令 SOP） | `--append-system-prompt`（保留 pi 默认骨架） |
| 技能 | skills | pi skills 机制 |
| 实例 | user prompt（本次任务实例） | prompt 文件 |

## 系统组件

1. **PromptManager**：模板管理器，加载和缓存 go embed 内嵌模板
2. **PromptBuilder**：变量替换 + 上下文注入
3. **PIBuilder**：pi 统一入口组合子（builder 三件之 pibuilder），每个命令产出 method + instance 两份

## 上下文注入

| 注入项 | 来源 | 内容 |
|--------|------|------|
| `loops_context` | `.rick/loops/*.md` | loop 清单（name + trigger） |
| `skills_context` | `.rick/skills/*_skill/skill.md` | skill 清单（标题 + 触发场景） |
| `domain_dir` | `.rick/domain` | domain 事实目录路径 |
| `doing_loop_content` | embedded skill | doing loop 正文（内联） |
| `orchestration_section` | builder | pi workflowScript 编排（doing） |
| `rick_bin_path` | cwd | rick 二进制路径 |

## 单文件内聚

关键 skill/loop 内容在构建时内联进主产物（`### skill:<name>` 结构化段），单文件内聚，pi 无需读散落文件。缺失内联源返回 error（非 panic、非静默产出空内容）。

## dry-run 验证

```bash
# 输出完整 prompt，无未替换模板变量
./bin/rick doing job_N --dry-run | grep -c '{{'   # 应为 0
```
