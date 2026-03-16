# 技术规范 (Technical Specifications)

## 技术栈

- 语言: Go 1.21+
- 框架: 标准库为主，最小化外部依赖
- 文档工具: Markdown + Mermaid
- 测试框架: Go testing 标准库 + Python 测试脚本
- 版本控制: Git

## 架构设计

- 架构风格: 模块化单体架构，按功能划分模块
- 模块划分: cmd（命令处理）、workspace（工作空间）、parser（解析）、executor（执行引擎）、prompt（提示词管理）、git（Git 操作）、config（配置管理）、callcli（Claude Code CLI 交互）
- 接口设计: 模块间通过清晰的接口交互，避免循环依赖
- 数据流设计: Plan → Doing → Learning 的单向数据流

## 开发规范

- 代码风格: 遵循 Go 官方代码规范（gofmt, golint）
- 日志系统:
  - 使用 Go 标准库 log，文本格式
  - 最小化日志输出，避免干扰 AI Agent
  - 日志级别: ERROR（错误）、INFO（关键信息）
  - 避免 DEBUG 级别日志
- 测试要求:
  - 单元测试覆盖率 >= 70%（Go testing）
  - 集成测试: 使用 Shell 脚本（scripts/test_*.sh）
  - 任务测试: 使用 Python JSON 格式测试脚本
  - 每个任务必须有可自动化验证的测试脚本
  - 测试脚本必须提供清晰的错误信息（包含完整路径）
  - 测试脚本必须返回明确的成功/失败状态（exit code）
- 文档要求:
  - 每个模块必须有对应的 Wiki 文档
  - 文档使用 Markdown 格式
  - 文档必须包含：概述、核心类型、关键函数、类图、使用示例
  - 文档必须使用 Mermaid 图表辅助说明
- 路径规范:
  - **任务描述中必须使用绝对路径或明确的相对路径起点**
  - 项目工作空间: `.rick/`
  - Wiki 文档: `.rick/wiki/`
  - Job 目录: `.rick/jobs/job_n/`
  - 测试脚本: `.rick/jobs/job_n/doing/test_scripts/`
- 任务粒度规范:
  - 每个任务的预计执行时间: 5-10 分钟
  - 任务目标必须清晰、可衡量
  - 任务必须有明确的输入和输出
  - 任务的关键结果必须可自动化验证

## 工程实践

- 版本控制:
  - 使用 Git 管理代码和文档
  - 每个任务完成后自动 commit
  - Commit message 遵循 Conventional Commits 规范
  - Co-Authored-By 标注 AI 协作
- 持续集成:
  - 自动运行测试脚本
  - 自动验证文档质量
  - 自动检查代码规范
- 发布流程:
  - 使用 `scripts/build.sh` 构建
  - 使用 `scripts/install.sh` 安装
  - 支持生产版本和开发版本并行
- 任务执行流程:
  - Plan: 将需求分解为 DAG 任务图
  - Doing: 按拓扑排序串行执行任务
  - Learning: 总结经验，提取可复用知识
- DAG 设计规范:
  - 使用三层结构：基础层 → 生产层 → 验证层
  - 识别并行化机会，提高执行效率
  - 在阶段性成果完成后设置汇聚点
  - 避免循环依赖
  - 每个任务的依赖关系必须明确
