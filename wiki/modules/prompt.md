# 提示词管理模块（internal/prompt）

## 职责

`internal/prompt` 是 builder 三件中 **templates** 的承载包（第三层「执行」）。它提供模板管理、变量替换、上下文注入，经 go embed 把提示词模板编译进二进制。它不直接调用 pi、不管理环境依赖。

## 核心组件

- **PromptManager**：加载/缓存内嵌模板（go embed）
- **PromptBuilder**：变量替换（`{{var}}` → 值）+ 上下文注入
- **Context loader**：`LoadLoopsContext` / `LoadSkillsContext` 对称注入 loops/skills 清单

## 三层注入模型

prompt 注入按三层分离，各走各的通道：

| 层 | 内容 | 注入通道 |
|----|------|----------|
| 方法 | system prompt（rick 方法描述） | `--append-system-prompt`（保留 pi 默认骨架） |
| 技能 | skills | pi skills 机制 |
| 实例 | user prompt（本次任务实例） | prompt 文件 |

## 单文件内聚

关键 skill/loop 内容（doing_loop、grilling、tdd-zh 等）在构建时**内联**进主产物（`### skill:<name>` 结构化段），单文件内聚，pi 无需读散落文件。

## 模板清单

| 模板 | 用途 |
|------|------|
| `templates/plan.md` / `doing.md` / `easy.md` | 对应命令主模板 |
| `templates/ctrl.md` / `dream.md` / `learning.md` | 对应命令主模板 |
| `templates/sense_loop.md` / `think.md` / `research.md` / `exporter.md` | human-loop 四文件 |
| `templates/skills/*.md` | 内联 skill 源（grilling/tdd-zh/doing_loop/debug_skill 等） |

## dry-run 约定

dry-run 输出完整 prompt 内容（非占位消息）。验证模板变量已替换：

```bash
./bin/rick <cmd> --dry-run | grep -c '{{'   # 应为 0
```
