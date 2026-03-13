# Plan: e2e_test

## 模块概述

**模块职责**: 验证整个 Rick CLI 系统的端到端功能、性能和稳定性，覆盖完整工作流和部署验证

**对应 Research**:
- `.morty/research/使用_golang_开发_rick_命令行程序.md` - 完整工作流、潜在挑战
- `.morty/research/DEVELOPMENT_GUIDE.md` - 开发工作流示例

**现有实现参考**: 无

**依赖模块**: __ALL__

**被依赖模块**: 无

## 接口定义

### 输入接口
- 完整的系统部署环境
- 所有功能模块已完成并通过集成测试

### 输出接口
- 端到端测试报告
- 性能测试结果
- 生产环境验证结果

## 数据模型

### E2ETestResult 结构体
```go
type E2ETestResult struct {
    TestName     string
    Status       string // pass, fail
    Duration     time.Duration
    ErrorMessage string
}
```

## Jobs

---

### Job 1: 开发环境启动验证

#### 目标

确保开发环境正确启动且等价于生产环境

#### 前置条件

- infrastructure:job_7 - 基础设施集成测试完成
- parser:job_6 - parser 集成测试完成
- dag_executor:job_7 - dag_executor 集成测试完成
- prompt_manager:job_8 - prompt_manager 集成测试完成
- git_integration:job_5 - git_integration 集成测试完成
- cli_commands:job_7 - cli_commands 集成测试完成
- installation:job_7 - installation 集成测试完成

#### Tasks

- [ ] Task 1: 构建完整的 Rick CLI 二进制
- [ ] Task 2: 运行 `rick --version` 验证版本号
- [ ] Task 3: 运行 `rick --help` 验证帮助信息
- [ ] Task 4: 验证 ~/.rick/config.json 配置文件
- [ ] Task 5: 验证所有依赖库都正确导入
- [ ] Task 6: 验证日志系统正常工作
- [ ] Task 7: 编写启动验证脚本

#### 验证器

- Rick CLI 二进制成功编译
- --version 显示正确的版本号
- --help 显示完整的帮助信息
- 配置文件格式正确
- 所有依赖库都可用
- 日志系统正常工作
- 启动验证脚本通过

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 2: 完整工作流测试

#### 目标

验证完整的 Rick CLI 工作流（init → plan → doing → learning）

#### 前置条件

- job_1 - 开发环境启动验证完成

#### Tasks

- [x] Task 1: 创建测试项目目录
- [x] Task 2: 运行 `rick init` 初始化项目
- [x] Task 3: 验证 .rick 目录结构正确创建
- [ ] Task 4: 运行 `rick plan "测试需求"` 规划任务
- [ ] Task 5: 验证 task.md 和 tasks.json 生成正确
- [ ] Task 6: 运行 `rick doing job_1` 执行任务
- [ ] Task 7: 验证任务执行和自动提交正确
- [ ] Task 8: 运行 `rick learning job_1` 进行学习
- [ ] Task 9: 验证 OKR.md 和 SPEC.md 更新正确

#### 验证器

- `rick init` 成功初始化项目
- .rick 目录结构完整
- `rick plan` 生成正确的 task.md 和 tasks.json
- `rick doing` 成功执行任务
- 任务完成后自动提交
- `rick learning` 成功更新知识库
- 完整工作流无错误

#### 调试日志

- debug1: rick plan 命令失败, 执行 `rick plan "测试"` 时报错 "open plan.md: no such file or directory", 猜想: 1)模板文件未被安装到 ~/.rick/templates 2)安装脚本未复制 templates 目录, 验证: 检查 install.sh 是否需要复制 templates, 修复: 在 install.sh 中添加复制 internal/prompt/templates 到 ~/.rick/templates 的逻辑, 已修复
- debug2: 修复 templates 后, 执行 `rick plan` 报错 "Claude Code cannot be launched inside another Claude Code session", 猜想: 1)无法在 Claude Code 内嵌套调用 Claude CLI 2)Rick 的设计假设可以调用 `claude` CLI, 验证: unset CLAUDECODE 环境变量后, 继续报错 "no tasks found in Claude output", 修复: Rick 的实现有设计缺陷 - 假设可以调用 `claude` 作为 CLI 工具但 Claude Code 不支持这种用法, 待重新设计

#### 完成状态

🔴 BLOCKED - 设计缺陷：Rick 的 `plan` 命令实现假设可以调用 `claude` CLI 工具，但 Claude Code 不支持这种用法

---

### Job 3: 并行版本管理测试

#### 目标

验证生产版本（rick）和开发版本（rick_dev）能并行运行

#### 前置条件

- job_2 - 完整工作流测试完成

#### Tasks

- [ ] Task 1: 安装生产版本 `./install.sh`
- [ ] Task 2: 验证 `rick --version` 可执行
- [ ] Task 3: 安装开发版本 `./install.sh --source --dev`
- [ ] Task 4: 验证 `rick_dev --version` 可执行
- [ ] Task 5: 验证两个版本使用不同的配置文件
- [ ] Task 6: 验证两个版本使用不同的工作空间
- [ ] Task 7: 同时运行 rick 和 rick_dev，验证不冲突

#### 验证器

- 生产版本和开发版本都能安装
- 两个版本命令都可执行
- 两个版本使用不同的配置
- 两个版本可以同时运行
- 版本间不存在冲突

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 4: 失败重试和恢复测试

#### 目标

验证失败重试机制和人工干预恢复流程

#### 前置条件

- job_3 - 并行版本管理测试完成

#### Tasks

- [ ] Task 1: 创建会失败的 task（模拟失败）
- [ ] Task 2: 运行 `rick doing job_n`，观察重试机制
- [ ] Task 3: 验证 debug.md 正确记录问题
- [ ] Task 4: 修改 task.md，修复问题
- [ ] Task 5: 重新运行 `rick doing job_n`，验证恢复
- [ ] Task 6: 验证最终任务成功完成

#### 验证器

- 重试机制正常工作（重试次数正确）
- debug.md 正确记录每次失败
- 修改 task.md 后能正确恢复
- 任务最终成功完成
- 自动提交记录完整

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 5: 性能和稳定性测试

#### 目标

验证 Rick CLI 在正常负载下的性能和稳定性

#### 前置条件

- job_4 - 失败重试和恢复测试完成

#### Tasks

- [ ] Task 1: 创建包含 20+ 个任务的 job
- [ ] Task 2: 运行 `rick doing job_n`，测量执行时间
- [ ] Task 3: 验证内存使用在合理范围内
- [ ] Task 4: 验证磁盘 I/O 在合理范围内
- [ ] Task 5: 运行多次完整工作流，验证稳定性
- [ ] Task 6: 验证日志文件大小在合理范围内

#### 验证器

- 执行时间在预期范围内
- 内存使用不超过 500MB
- 磁盘 I/O 合理
- 多次运行无异常
- 日志文件大小合理

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 6: 自我重构能力验证

#### 目标

验证 Rick CLI 能够使用自身进行重构

#### 前置条件

- job_5 - 性能和稳定性测试完成

#### Tasks

- [ ] Task 1: 使用生产版本 rick 规划重构任务
- [ ] Task 2: 安装开发版本 rick_dev
- [ ] Task 3: 使用生产版本 rick 执行重构任务
- [ ] Task 4: 使用开发版本 rick_dev 验证重构
- [ ] Task 5: 更新生产版本 `./update.sh`
- [ ] Task 6: 验证新版本功能正常
- [ ] Task 7: 卸载开发版本

#### 验证器

- 规划、执行、验证、更新流程完整
- 自我重构成功完成
- 新版本功能正常
- 版本更新无错误

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 7: 文档和示例验证

#### 目标

验证文档完整性和示例代码正确性

#### 前置条件

- job_6 - 自我重构能力验证完成

#### Tasks

- [ ] Task 1: 验证 README.md 文档完整
- [ ] Task 2: 验证 DEVELOPMENT_GUIDE.md 文档正确
- [ ] Task 3: 验证 QUICK_REFERENCE.md 快速参考有用
- [ ] Task 4: 运行 README 中的示例代码
- [ ] Task 5: 验证所有脚本都有使用说明
- [ ] Task 6: 验证错误消息清晰有帮助
- [ ] Task 7: 生成测试报告

#### 验证器

- 文档完整且准确
- 所有示例代码可运行
- 脚本使用说明清晰
- 错误消息有帮助
- 测试报告生成成功

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 8: 集成测试

#### 目标

验证整个系统的端到端集成正确性

#### 前置条件

- job_1 - 开发环境启动验证完成
- job_2 - 完整工作流测试完成
- job_3 - 并行版本管理测试完成
- job_4 - 失败重试和恢复测试完成
- job_5 - 性能和稳定性测试完成
- job_6 - 自我重构能力验证完成
- job_7 - 文档和示例验证完成

#### Tasks

- [ ] Task 1: 验证所有模块协同工作正常
- [ ] Task 2: 验证系统在压力下的稳定性
- [ ] Task 3: 验证生产环境配置正确
- [ ] Task 4: 运行完整的 E2E 测试套件
- [ ] Task 5: 验证所有 CLI 命令都可用
- [ ] Task 6: 验证所有错误处理机制正常
- [ ] Task 7: 生成最终的 E2E 测试报告

#### 验证器

- 所有模块协同工作产生正确结果
- 系统在压力测试下保持稳定
- 生产环境配置验证通过
- E2E 测试套件全部通过
- 所有 CLI 命令都可用
- 错误处理机制正常工作
- 测试报告生成完整

#### 调试日志

无

#### 完成状态

⏳ 待开始

