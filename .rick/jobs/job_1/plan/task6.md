# 依赖关系
task4

# 任务名称
编写提示词管理系统文档

# 任务目标
创建 `wiki/prompt-system.md`，详细说明 Rick CLI 的独特模块——提示词管理系统。包含提示词管理架构（PromptManager, PromptBuilder, ContextManager）、模板系统设计、上下文加载机制、提示词生成流程。使用类图和流程图展示系统设计，说明如何扩展新模板和最佳实践。

# 关键结果
1. 完成 `wiki/prompt-system.md` 文档创建
2. 详细说明提示词管理架构（PromptManager, PromptBuilder, ContextManager）
3. 讲解模板系统设计（plan.md, doing.md, test.md, learning.md）
4. 说明上下文加载机制（OKR, SPEC, debug.md）
5. 描述提示词生成流程（模板加载 → 变量替换 → 上下文注入）
6. 绘制类图和流程图（使用 Mermaid）
7. 说明如何扩展新模板的方法
8. 提供最佳实践和使用示例

# 测试方法
1. 验证文件已创建：`test -f wiki/prompt-system.md && echo "PASS" || echo "FAIL"`
2. 检查包含核心章节：`grep -q "## 提示词管理架构\|## 模板系统\|## 上下文加载\|## 提示词生成流程\|## 扩展新模板" wiki/prompt-system.md && echo "PASS" || echo "FAIL"`
3. 验证包含 Mermaid 图表（至少 2 个）：`grep -c '```mermaid' wiki/prompt-system.md | awk '{if($1>=2) print "PASS"; else print "FAIL"}'`
4. 验证文档长度（至少 150 行）：`wc -l wiki/prompt-system.md | awk '{if($1>=150) print "PASS"; else print "FAIL"}'`
5. 检查包含关键组件名称：`grep -q "PromptManager\|PromptBuilder\|ContextManager\|模板系统\|变量替换" wiki/prompt-system.md && echo "PASS" || echo "FAIL"`
