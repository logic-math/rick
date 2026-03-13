# Plan: prompt_manager

## 模块概述

**模块职责**: 实现提示词管理模块，支持提示词模板管理、动态构建、上下文管理，是 Rick CLI 的核心创新模块

**对应 Research**:
- `.morty/research/使用_golang_开发_rick_命令行程序.md` - 提示词管理模块设计
- `.morty/research/DEVELOPMENT_GUIDE.md` - 代码组织规范

**现有实现参考**: 无

**依赖模块**: infrastructure, parser

**被依赖模块**: dag_executor, cli_commands

## 接口定义

### 输入接口
- 提示词模板文件（plan.md, doing.md, test.md, learning.md）
- Task 结构体
- 执行上下文（debug.md, OKR.md, SPEC.md）
- 模板变量（{{variable}} 格式）

### 输出接口
- 完整的提示词字符串
- 提示词构建器实例
- 上下文管理器实例

## 数据模型

### PromptTemplate 结构体
```go
type PromptTemplate struct {
    Name      string
    Content   string
    Variables []string
}
```

### PromptBuilder 结构体
```go
type PromptBuilder struct {
    Template  *PromptTemplate
    Context   map[string]interface{}
    Variables map[string]string
}
```

### ContextManager 结构体
```go
type ContextManager struct {
    Task      *Task
    Debug     *DebugInfo
    OKR       string
    SPEC      string
    History   []string
}
```

## Jobs

---

### Job 1: 提示词模板管理器

#### 目标

实现提示词模板管理系统，支持模板的加载、存储、缓存

#### 前置条件

- infrastructure:job_3 - 工作空间管理系统完成

#### Tasks

- [ ] Task 1: 创建 internal/prompt/manager.go，实现 PromptManager 类型
- [ ] Task 2: 创建提示词模板目录 internal/prompt/templates/
- [ ] Task 3: 创建 plan.md 模板，用于规划阶段
- [ ] Task 4: 创建 doing.md 模板，用于执行阶段
- [ ] Task 5: 创建 test.md 模板，用于测试脚本生成
- [ ] Task 6: 创建 learning.md 模板，用于学习阶段
- [ ] Task 7: 实现 LoadTemplate(name) 函数，加载模板文件
- [ ] Task 8: 实现模板缓存机制，避免重复加载
- [ ] Task 9: 编写单元测试，覆盖模板加载

#### 验证器

- 所有模板文件都存在且格式正确
- LoadTemplate() 能正确加载模板
- 缓存机制正常工作
- 模板内容包含必要的变量占位符
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 2: 提示词构建器

#### 目标

实现提示词构建器，支持动态构建完整的提示词

#### 前置条件

- job_1 - 提示词模板管理器完成

#### Tasks

- [ ] Task 1: 创建 internal/prompt/builder.go，实现 PromptBuilder 类型
- [ ] Task 2: 实现 NewPromptBuilder(template) 函数
- [ ] Task 3: 实现 SetVariable(key, value) 方法，设置模板变量
- [ ] Task 4: 实现 SetContext(key, value) 方法，设置上下文
- [ ] Task 5: 实现 Build() 方法，构建最终提示词
- [ ] Task 6: 实现变量替换逻辑（{{variable}} 格式）
- [ ] Task 7: 实现上下文注入逻辑
- [ ] Task 8: 编写单元测试，覆盖提示词构建流程

#### 验证器

- NewPromptBuilder() 能正确创建构建器实例
- SetVariable() 和 SetContext() 能正确设置值
- Build() 返回格式正确的提示词
- 变量替换正确（所有 {{variable}} 被替换）
- 上下文信息正确注入
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 3: 上下文管理器

#### 目标

实现上下文管理器，支持从各种源加载和管理执行上下文

#### 前置条件

- parser:job_5 - 多文件解析协调器完成

#### Tasks

- [ ] Task 1: 创建 internal/prompt/context.go，实现 ContextManager 类型
- [ ] Task 2: 实现 NewContextManager(jobID) 函数
- [ ] Task 3: 实现 LoadTask(task) 方法，加载任务信息
- [ ] Task 4: 实现 LoadDebug(debugInfo) 方法，加载问题记录
- [ ] Task 5: 实现 LoadOKR(okrContent) 方法，加载 OKR
- [ ] Task 6: 实现 LoadSPEC(specContent) 方法，加载 SPEC
- [ ] Task 7: 实现 LoadHistory(jobID) 方法，加载执行历史
- [ ] Task 8: 编写单元测试，覆盖上下文加载

#### 验证器

- NewContextManager() 能正确创建实例
- LoadTask() 能正确加载任务信息
- LoadDebug() 能正确加载问题记录
- LoadOKR() 和 LoadSPEC() 能正确加载文件
- LoadHistory() 能正确加载执行历史
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 4: 规划阶段提示词生成

#### 目标

实现规划阶段的提示词生成，支持从用户需求生成 task.md

#### 前置条件

- job_2 - 提示词构建器完成
- job_3 - 上下文管理器完成

#### Tasks

- [ ] Task 1: 创建 internal/prompt/plan_prompt.go，实现规划提示词生成
- [ ] Task 2: 实现 GeneratePlanPrompt(requirement) 函数
- [ ] Task 3: 实现 GeneratePlanPrompt 包含项目 OKR 和 SPEC 上下文
- [ ] Task 4: 实现提示词包含任务格式规范（task.md 格式）
- [ ] Task 5: 实现提示词包含依赖关系说明
- [ ] Task 6: 编写单元测试，覆盖规划提示词生成

#### 验证器

- GeneratePlanPrompt() 返回格式正确的提示词
- 提示词包含项目上下文
- 提示词包含任务格式规范
- 提示词清晰明确
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 5: 执行阶段提示词生成

#### 目标

实现执行阶段的提示词生成，支持从 task.md 生成完整的执行提示词

#### 前置条件

- job_4 - 规划阶段提示词生成完成

#### Tasks

- [ ] Task 1: 创建 internal/prompt/doing_prompt.go，实现执行提示词生成
- [ ] Task 2: 实现 GenerateDoingPrompt(task, retryCount) 函数
- [ ] Task 3: 实现提示词包含任务目标和关键结果
- [ ] Task 4: 实现提示词包含测试方法
- [ ] Task 5: 如果是重试，加载 debug.md 作为额外上下文
- [ ] Task 6: 实现提示词包含项目 SPEC 作为背景
- [ ] Task 7: 编写单元测试，覆盖执行提示词生成

#### 验证器

- GenerateDoingPrompt() 返回格式正确的提示词
- 提示词包含完整的任务信息
- 提示词包含测试方法
- 重试时包含问题记录上下文
- 提示词清晰明确
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 6: 测试脚本生成提示词

#### 目标

实现测试脚本生成的提示词，支持生成有效的测试脚本

#### 前置条件

- job_5 - 执行阶段提示词生成完成

#### Tasks

- [ ] Task 1: 创建 internal/prompt/test_prompt.go，实现测试提示词生成
- [ ] Task 2: 实现 GenerateTestPrompt(task, code) 函数
- [ ] Task 3: 实现提示词包含任务的测试方法
- [ ] Task 4: 实现提示词包含生成的代码
- [ ] Task 5: 实现提示词包含测试脚本格式规范
- [ ] Task 6: 编写单元测试，覆盖测试提示词生成

#### 验证器

- GenerateTestPrompt() 返回格式正确的提示词
- 提示词包含测试方法
- 提示词包含代码上下文
- 提示词清晰明确
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 7: 学习阶段提示词生成

#### 目标

实现学习阶段的提示词生成，支持从执行结果生成知识总结

#### 前置条件

- job_6 - 测试脚本生成提示词完成

#### Tasks

- [ ] Task 1: 创建 internal/prompt/learning_prompt.go，实现学习提示词生成
- [ ] Task 2: 实现 GenerateLearningPrompt(jobID) 函数
- [ ] Task 3: 实现提示词包含所有任务的执行结果
- [ ] Task 4: 实现提示词包含 debug.md 中的问题记录
- [ ] Task 5: 实现提示词包含 git 历史提交
- [ ] Task 6: 编写单元测试，覆盖学习提示词生成

#### 验证器

- GenerateLearningPrompt() 返回格式正确的提示词
- 提示词包含完整的执行历史
- 提示词包含问题记录
- 提示词包含 git 历史
- 提示词清晰明确
- 单元测试覆盖率 >= 80%

#### 调试日志

无

#### 完成状态

⏳ 待开始

---

### Job 8: 集成测试

#### 目标

验证 prompt_manager 模块所有组件协同工作正确，能正确生成各阶段的提示词

#### 前置条件

- job_1 - 提示词模板管理器完成
- job_2 - 提示词构建器完成
- job_3 - 上下文管理器完成
- job_4 - 规划阶段提示词生成完成
- job_5 - 执行阶段提示词生成完成
- job_6 - 测试脚本生成提示词完成
- job_7 - 学习阶段提示词生成完成

#### Tasks

- [ ] Task 1: 验证所有模板都能正确加载
- [ ] Task 2: 验证提示词构建器能正确构建提示词
- [ ] Task 3: 验证上下文管理器能正确加载上下文
- [ ] Task 4: 验证规划、执行、测试、学习提示词都能生成
- [ ] Task 5: 验证提示词包含所有必要的上下文信息
- [ ] Task 6: 验证重试时提示词包含问题记录
- [ ] Task 7: 编写集成测试脚本，覆盖完整提示词生成流程

#### 验证器

- 所有模板都能正确加载
- 提示词构建器能正确工作
- 上下文管理器能正确工作
- 所有阶段的提示词都能生成
- 提示词格式正确且完整
- 集成测试脚本通过

#### 调试日志

无

#### 完成状态

⏳ 待开始

