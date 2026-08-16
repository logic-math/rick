# Rick CLI Wiki

欢迎来到 Rick CLI 的知识库！这是一个 Context-First AI Coding Framework 的文档中心。

> 事实来源：`.rick/domain/` 是 agent 内部知识库（spec 规范 / rick-spec 实例 / 命令规范 / 架构事实），本 Wiki 是面向用户的学习文档，两者并行存在，互不迁移。本 Wiki 只**读** `.rick/domain/`，不写。

## 目录结构

```
wiki/
├── README.md                    # 本文件：Wiki 索引和导航
├── architecture.md              # 三层金字塔架构 + spec 信息内核 + 下沉策略
├── runtime-flow.md              # 运行时流程（env → builder → runtime）
├── prompt-system.md             # 提示词系统（方法/技能/实例三层注入）
├── testing.md                   # 测试与验证
├── installation.md              # 安装部署指南（init-pi）
├── CONTRIBUTING.md              # 贡献指南
└── modules/                     # 模块详细文档
    ├── cmd.md                   # 入口层（Cobra 命令）
    ├── handler.md               # 调度聚合层
    ├── env.md                   # 执行层：环境四职责
    ├── builder.md               # 执行层：提示词拼接（templates/pibuilder/xxxxbuilder）
    ├── runtime.md               # 执行层：拉起 pi + 采集轨迹
    ├── prompt.md                # 提示词模板与管理
    ├── workspace.md             # 路径管理
    ├── config.md                # 配置管理
    └── human-loop.md            # SENSE 深度思考
```

## 文档导航

### 🚀 新手入门

1. **[安装部署指南](installation.md)** - 安装、配置（`rick tools init-pi`）
2. **[系统架构设计](architecture.md)** - 三层金字塔 + spec 信息内核
3. **[运行时流程详解](runtime-flow.md)** - env → builder → runtime 全链路

### 📚 深入理解

1. **[提示词系统详解](prompt-system.md)** - 方法/技能/实例三层注入
2. **[测试与验证](testing.md)** - 单元测试、集成测试、门禁
3. **[模块详细文档](modules/)** - 各核心模块说明

### 🔧 模块文档

- **[命令处理模块](modules/cmd.md)** - 入口层：路由命令、解析参数
- **[调度聚合模块](modules/handler.md)** - 编排 env → builder → runtime
- **[环境模块](modules/env.md)** - env 四职责
- **[提示词拼接模块](modules/builder.md)** - templates + pibuilder + xxxxbuilder
- **[运行时模块](modules/runtime.md)** - 拉起 pi + 采集行为轨迹
- **[提示词管理模块](modules/prompt.md)** - 模板管理、上下文注入
- **[工作空间管理](modules/workspace.md)** - 路径解析
- **[配置管理](modules/config.md)** - `~/.rick/config.json`
- **[human-loop 模块](modules/human-loop.md)** - SENSE 深度思考

## 核心命令速查

```bash
# 初始化 pi（rick 的 agent runtime）+ 扩展 + 主题
rick tools init-pi

# 规划任务
rick plan "任务描述"

# 执行任务（dag 调度下沉 pi workflowScript，门禁下沉 rick-gates）
rick doing job_n

# 知识积累
rick learning job_n

# 全局反思
rick dream
```

## 版本信息

- **当前版本**：v3.2.0（三层金字塔重构：rick 做薄，dag 调度/门禁/agent 下沉 pi）
- **Go 版本要求**：>= 1.21
- **支持平台**：macOS, Linux

## 相关资源

- **项目仓库**：[GitHub](https://github.com/sunquan/rick)
- **spec 信息内核**：`.rick/domain/spec.md`（规范）+ `.rick/domain/rick-spec.md`（实例）
- **Skills 库**：`.rick/skills/` - 可复用技能库
- **Loops 库**：`.rick/loops/` - 可复用工作流
- **Learning 记录**：`.rick/jobs/*/learning/` - 历史项目的知识积累
