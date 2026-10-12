APPROVED: true

# Job job_18 执行总结

## 执行概述

**项目目标**: 在 `rick easy` 和 `rick plan` 中嵌入结构化 grilling 追问机制，替换 sense S→E→N 追问，提升需求澄清质量
**实际完成**: 4/4 任务全部成功，0 次重试，所有 KR 达成
**整体评价**: ⭐⭐⭐⭐⭐

## 关键成就

1. **新增 grilling skill 文件**: `internal/prompt/templates/skills/grilling.md`，通过 embed.FS 加载，内容包含"Interview me relentlessly"追问协议 + 终止条件
2. **plan.md 模板重构**: 移除 `## 三、思考方法`（sense S→E→N）及 Step 1/Step 7，新增 grilling 步骤，SOP 步骤连续无断号
3. **easy.md 模板增强**: 新增 grilling 步骤，grilling 结束后 append 澄清内容到 `requirement.md`（禁止覆写原始需求）
4. **全链路验证通过**: `./bin/rick plan --dry-run` 和 easy 生成 prompt 均含 `skill_grilling.md` 路径，无未替换占位符

## 问题与教训

### 问题1: 测试断言字符串与实际路径不匹配（task2）

**根本原因**: 断言写的是 `"grilling_skill"` 但 `WriteSkillFile` 写出文件名为 `skill_grilling.md`，顺序不同
**解决方案**: 改断言为 `strings.Contains(prompt, "skill_grilling.md")`
**经验教训**: 测试注入路径时，断言字符串应与 `WriteSkillFile(promptsDir, "skill_xxx.md", ...)` 的第二个参数（文件名）一致，写测试前先确认文件名格式

### 问题2: mark_task_success CLI 参数格式（task1）

**根本原因**: Agent 调用 `./bin/rick tools mark_task_success --job job_18 --task task1`，实际接口为位置参数 `job_18 task1`
**解决方案**: 直接 Edit `tasks.json` 手动更新状态
**经验教训**: 调用 rick tools 子命令前先运行 `--help` 确认参数接口，避免一次无效调用

### 问题3: easy.md 模板变量在测试中触发替换失败（task3）

**根本原因**: 模板中 `{{doing_dir}}` 变量出现在 grilling 指令说明文字里，测试构造的 builder 未设置该变量导致报错
**解决方案**: 将说明文字中的 `{{doing_dir}}` 改为 `<doing_dir>`（非变量语法）
**经验教训**: skill 注入步骤中涉及路径示例时，用 `<>` 包裹说明性路径，不用 `{{}}`，避免与模板变量系统冲突

## 知识沉淀清单

- [x] `.rick/wiki/adding_new_skill_to_templates.md` - 向 plan/easy 模板注入新 skill 的完整工作流（WriteSkillFile 模式）
- [x] `SPEC.md` - 技能列表新增 adding-new-skill-to-templates 条目
