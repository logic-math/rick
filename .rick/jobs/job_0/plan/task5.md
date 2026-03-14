# 依赖关系
task3, task4

# 任务名称
提取可复用技能并创建 Skills 库

# 任务目标
从现有代码库中识别可复用的技能模式，创建 Skills 库的目录结构和初始技能文档，为未来的 Context Loop 学习沉淀提供框架。

# 关键结果
1. 识别 5-10 个可复用技能，例如：
   - DAG 拓扑排序算法
   - Markdown 解析技巧
   - Go 嵌入式资源管理
   - 失败重试模式
   - Git 自动化操作
   - CLI 参数设计模式
   - 测试覆盖率优化
   - 模块化架构设计
2. 创建 Skills 目录结构：
   ```
   .rick/skills/
   ├── index.md
   ├── dag-topological-sort/
   │   ├── description.md
   │   ├── implementation.md
   │   └── examples/
   ├── markdown-parsing/
   ├── embedded-resources/
   ├── retry-pattern/
   ├── git-automation/
   └── ...
   ```
3. 为每个技能编写文档：
   - description.md：技能描述、使用场景、优缺点
   - implementation.md：实现细节、代码示例、最佳实践
   - examples/：实际应用案例
4. 创建 `skills/index.md` 索引，包含：
   - 技能分类（算法、工具、模式、架构）
   - 技能列表和简介
   - 使用指南

# 测试方法
1. 验证 `.rick/skills/` 目录结构已创建
2. 检查至少包含 5 个技能目录
3. 验证每个技能包含：
   - description.md（至少 200 字）
   - implementation.md（包含代码示例）
   - 至少 1 个实际应用案例
4. 检查 `index.md` 包含完整的技能导航
5. 验证技能文档与项目代码一致
6. 确保技能具有可复用性和通用性
