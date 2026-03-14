# 依赖关系
task1, task2, task3, task4, task5

# 任务名称
创建项目使用示例和最佳实践文档

# 任务目标
基于已建立的全局上下文，创建完整的使用示例和最佳实践文档，补充到 Wiki 中，确保新用户能够快速上手。

# 关键结果
1. 创建使用示例文档：
   - `.rick/wiki/getting-started.md` - 快速入门指南
   - `.rick/wiki/tutorials/` - 教程目录
   - `.rick/wiki/best-practices.md` - 最佳实践
2. 编写快速入门指南，包含：
   - 安装步骤（生产版 + 开发版）
   - 第一个 Job 示例
   - 常用命令速查
   - 故障排查
3. 创建 3-5 个教程：
   - Tutorial 1: 使用 Rick 管理简单项目
   - Tutorial 2: 使用 Rick 重构 Rick（自我重构）
   - Tutorial 3: 并行版本管理（rick + rick_dev）
   - Tutorial 4: 自定义提示词模板
   - Tutorial 5: 集成到 CI/CD 流程
4. 编写最佳实践文档，包含：
   - 任务分解原则
   - 依赖关系设计
   - 测试方法编写
   - 失败重试策略
   - Learning 阶段审核

# 测试方法
1. 验证文档已创建：
   ```
   .rick/wiki/
   ├── getting-started.md
   ├── best-practices.md
   └── tutorials/
       ├── tutorial-1-simple-project.md
       ├── tutorial-2-self-refactoring.md
       ├── tutorial-3-parallel-versions.md
       ├── tutorial-4-custom-prompts.md
       └── tutorial-5-ci-cd-integration.md
   ```
2. 检查快速入门指南可操作性（按步骤执行无错误）
3. 验证每个教程包含：
   - 目标说明
   - 完整步骤
   - 预期结果
   - 常见问题
4. 确保最佳实践与 SPEC.md 和 OKR.md 对齐
5. 验证所有示例代码可运行
