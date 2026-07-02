# skill:source-context-consistency（源码与 loops/skills 一致性检查）

深度对照项目源码，发现并修正 `.rick/loops/` 和 `.rick/skills/` 中与源码事实不符的内容，确保 loops/skills 永远如实反映代码现状。

---

## 触发场景

在 dream 阶段执行一致性审核时使用：源码已重构、函数签名已改、模块已删除，但 `.rick/skills/` 中的 skill.md 仍描述旧实现，或 `.rick/loops/` 中的 loop 引用了已删除的工具/命令。

---

## 执行 SOP

### 1. 加载当前上下文

读取以下文件，提取所有陈述性事实：
- `.rick/skills/*/skill.md`：各 skill 中的命令、函数名、路径
- `.rick/loops/*.md`：各 loop 中的工具调用、命令、step 引用

### 2. 逐条源码验证

对每一条上下文陈述，使用 Read / Grep / Glob 对照实际源码（`internal/`、`cmd/`）验证：

| 陈述类型 | 验证方法 |
|----------|----------|
| 函数/方法签名 | Grep 对应函数名，Read 实际定义 |
| CLI 命令/参数 | `./bin/rick <cmd> --help` 验证 |
| 文件路径 | Glob 验证路径是否存在 |
| Python 脚本参数 | `python3 script.py --help` 验证 |
| 接口/依赖关系 | Grep 调用方，确认依赖仍存在 |

### 3. 分类不一致项

将发现的不一致分为三类：
- **已删除**：skill/loop 描述的函数/命令/路径在源码中已不存在
- **已变更**：签名、行为、参数已改变，描述过时
- **已新增**：源码中有重要实现，skill/loop 完全未提及

### 4. 输出不一致清单

对每项不一致输出：
```
- 文件: .rick/skills/mark_task_success_skill/skill.md
  描述: "python3 .rick/tools/mark_task_success.py"
  实际: 脚本已移至 .rick/skills/mark_task_success_skill/mark_task_success.py
  类型: 已变更
```

### 5. 修复

直接更新 `.rick/loops/*.md` 和 `.rick/skills/*/skill.md` 中不实的内容：
- **已删除项**：删除对应步骤，或更新为新路径
- **已变更项**：用当前事实替换旧描述
- **已新增项**：在合适 skill 的核心内容中补充说明

## 约束

⚠️ **允许**：Read、Grep、Glob 查阅任意源码；写入 `.rick/loops/`、`.rick/skills/`
⚠️ **禁止**：修改任何业务源代码（`internal/`、`cmd/` 等）
