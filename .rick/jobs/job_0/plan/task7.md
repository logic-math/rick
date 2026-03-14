# 依赖关系
task1, task2, task3, task4, task5, task6

# 任务名称
验证全局上下文完整性并生成总结报告

# 任务目标
全面验证已生成的 OKR、SPEC、Wiki、Skills 等全局上下文文件的完整性、一致性和可用性，生成 job_0 的总结报告。

# 关键结果
1. 验证文件完整性：
   - 检查所有必需文件已创建
   - 验证文件格式正确（Markdown）
   - 检查文件大小合理（非空文件）
2. 验证内容一致性：
   - OKR 与项目实际功能对齐
   - SPEC 与现有代码规范一致
   - Wiki 文档间交叉引用正确
   - Skills 与代码实现一致
3. 验证可用性：
   - 文档可读性良好（清晰的结构、目录）
   - 示例代码可执行
   - 链接有效（内部链接、外部链接）
   - 搜索友好（关键词覆盖）
4. 生成总结报告 `.rick/jobs/job_0/learning/summary.md`，包含：
   - Job 0 执行概览
   - 生成的文件清单
   - 关键发现和洞察
   - 待改进事项
   - 下一步建议

# 测试方法
1. 运行验证脚本，检查：
   ```bash
   # 验证文件存在
   test -f .rick/OKR.md
   test -f .rick/SPEC.md
   test -f .rick/wiki/index.md
   test -f .rick/skills/index.md

   # 验证文件非空
   test -s .rick/OKR.md
   test -s .rick/SPEC.md

   # 验证目录结构
   test -d .rick/wiki/modules
   test -d .rick/skills
   test -d .rick/wiki/tutorials
   ```
2. 手动审查每个文档的质量
3. 验证总结报告包含：
   - 执行时间线
   - 生成的文件列表（至少 20 个文件）
   - 关键统计数据（字数、文件数、模块数）
   - 质量评估（完整性、一致性、可用性）
4. 确认所有任务的关键结果已达成
5. 验证 job_0 可作为后续 job 的参考基准
