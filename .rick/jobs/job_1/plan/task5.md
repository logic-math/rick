# 依赖关系
task4

# 任务名称
编写 DAG 执行引擎详解

# 任务目标
创建 `wiki/dag-execution.md`，深入讲解 Rick CLI 的任务执行引擎。包含 DAG 数据结构设计、拓扑排序算法（Kahn 算法）实现、环检测机制（DFS 算法）、任务执行流程、重试机制详解。使用类图和时序图展示执行流程，提供代码示例和最佳实践。

# 关键结果
1. 完成 `wiki/dag-execution.md` 文档创建
2. 详细说明 DAG 数据结构设计（Graph, Tasks 映射）
3. 讲解拓扑排序算法实现（Kahn 算法，包含伪代码）
4. 讲解环检测机制（DFS 算法，包含实现细节）
5. 详细描述任务执行流程（测试生成 → 执行 → 验证 → 提交）
6. 深入说明重试机制（RetryManager, debug.md 上下文传递）
7. 绘制完整的类图和时序图（使用 Mermaid）
8. 提供代码示例和最佳实践

# 测试方法
1. 验证文件已创建：`test -f wiki/dag-execution.md && echo "PASS" || echo "FAIL"`
2. 检查包含核心章节：`grep -q "## DAG 数据结构\|## 拓扑排序\|## 环检测\|## 任务执行流程\|## 重试机制" wiki/dag-execution.md && echo "PASS" || echo "FAIL"`
3. 验证包含 Mermaid 图表（至少 2 个）：`grep -c '```mermaid' wiki/dag-execution.md | awk '{if($1>=2) print "PASS"; else print "FAIL"}'`
4. 验证文档长度（至少 200 行）：`wc -l wiki/dag-execution.md | awk '{if($1>=200) print "PASS"; else print "FAIL"}'`
5. 检查包含算法关键词：`grep -q "Kahn\|DFS\|拓扑排序\|环检测\|RetryManager\|debug.md" wiki/dag-execution.md && echo "PASS" || echo "FAIL"`
