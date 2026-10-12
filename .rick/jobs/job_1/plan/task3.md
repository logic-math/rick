# 依赖关系
task2

# 任务名称
编写运行时流程文档

# 任务目标
创建 `wiki/runtime-flow.md`，详细描述 Rick CLI 的运行时流程。包含 Plan、Doing、Learning 三个阶段的完整流程，使用 Mermaid 时序图和流程图展示关键流程，说明每个阶段的输入输出和关键决策点。

# 关键结果
1. 完成 `wiki/runtime-flow.md` 文档创建
2. 详细描述 Plan 阶段流程（需求 → 追问 → 任务分解 → task.md 生成）
3. 详细描述 Doing 阶段流程（DAG 构建 → 拓扑排序 → 串行执行 → 测试验证 → Git 提交）
4. 详细描述 Learning 阶段流程（知识提取 → 更新 OKR/SPEC/Wiki/Skills）
5. 绘制完整的时序图和流程图（使用 Mermaid）
6. 说明每个阶段的关键决策点和错误处理机制

# 测试方法
1. 验证文件已创建：`test -f wiki/runtime-flow.md && echo "PASS" || echo "FAIL"`
2. 检查包含三个阶段章节：`grep -q "## Plan 阶段\|## Doing 阶段\|## Learning 阶段" wiki/runtime-flow.md && echo "PASS" || echo "FAIL"`
3. 验证包含 Mermaid 图表（至少 2 个）：`grep -c '```mermaid' wiki/runtime-flow.md | awk '{if($1>=2) print "PASS"; else print "FAIL"}'`
4. 验证文档长度（至少 150 行）：`wc -l wiki/runtime-flow.md | awk '{if($1>=150) print "PASS"; else print "FAIL"}'`
5. 检查包含关键流程术语：`grep -q "拓扑排序\|重试机制\|测试验证\|Git 提交\|知识提取" wiki/runtime-flow.md && echo "PASS" || echo "FAIL"`
