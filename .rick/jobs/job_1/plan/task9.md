# 依赖关系
task3, task4, task5, task6, task7, task8

# 任务名称
验证和完善 Wiki 文档

# 任务目标
检查所有 Wiki 文档的完整性和一致性，确保文档质量。验证所有链接正确、Mermaid 图表语法正确、代码示例准确、文档风格统一。创建贡献指南，更新主 README.md 添加 Wiki 导航链接，生成最终的 Wiki 验证报告。

# 关键结果
1. 验证所有内部链接正确（文档间引用、代码引用）
2. 确保所有 Mermaid 图表语法正确（可以渲染）
3. 检查所有代码示例的准确性（引用真实代码）
4. 统一文档风格和术语（一致的标题层级、术语使用）
5. 创建 `wiki/CONTRIBUTING.md` 贡献指南
6. 更新主 `README.md`，添加 Wiki 导航链接
7. 生成 Wiki 验证报告（包含文档统计、图表数量、链接检查结果）

# 测试方法
1. 验证所有计划的文档都已创建：`test -f wiki/README.md && test -f wiki/architecture.md && test -f wiki/runtime-flow.md && test -f wiki/dag-execution.md && test -f wiki/prompt-system.md && test -f wiki/testing.md && test -f wiki/installation.md && test -d wiki/modules && echo "PASS" || echo "FAIL"`
2. 验证 CONTRIBUTING.md 已创建：`test -f wiki/CONTRIBUTING.md && echo "PASS" || echo "FAIL"`
3. 验证主 README.md 包含 Wiki 链接：`grep -q "wiki/README.md\|Wiki 文档\|文档导航" README.md && echo "PASS" || echo "FAIL"`
4. 检查 Mermaid 语法（统计所有 mermaid 代码块）：`grep -r '```mermaid' wiki/ | wc -l | awk '{if($1>=10) print "PASS"; else print "FAIL"}'`
5. 验证文档总量（至少 1500 行）：`find wiki/ -name "*.md" -exec wc -l {} + | tail -1 | awk '{if($1>=1500) print "PASS"; else print "FAIL"}'`
