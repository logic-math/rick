# Tutorial 4: 自定义提示词模板

> 学习如何定制 Rick CLI 的提示词模板，优化 AI 编程效果

## 📋 目标

在本教程中，你将学习：
- 理解 Rick 的提示词管理机制
- 自定义各阶段的提示词模板
- 优化提示词以提升代码质量
- 创建特定领域的提示词模板

## 🎯 提示词模板架构

### 模板目录结构

```
internal/prompt/templates/
├── plan.md         # 规划阶段提示词
├── doing.md        # 执行阶段提示词
├── test.md         # 测试阶段提示词
└── learning.md     # 学习阶段提示词
```

### 提示词构建流程

```
1. 加载模板 (template.md)
2. 收集上下文 (context)
3. 构建提示词 (builder)
4. 发送给 Claude Code
```

---

## Step 1: 理解默认提示词模板

### 查看默认模板

```bash
# 查看规划阶段模板
cat internal/prompt/templates/plan.md

# 查看执行阶段模板
cat internal/prompt/templates/doing.md

# 查看学习阶段模板
cat internal/prompt/templates/learning.md
```

### 默认 doing.md 模板结构

```markdown
# Rick 项目执行阶段提示词

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

## 任务信息

**任务 ID**: {{task_id}}
**任务名称**: {{task_name}}
**重试次数**: {{retry_count}}

### 任务目标
{{task_goal}}

### 关键结果
{{key_results}}

### 测试方法
{{test_methods}}

## 项目背景

**项目名称**: {{project_name}}
**项目描述**: {{project_description}}

### 项目 SPEC
{{spec_content}}

### 项目架构
{{architecture_content}}

## 执行上下文

### 已完成的任务
{{completed_tasks}}

### 任务依赖
{{task_dependencies}}

{{#if retry_count > 0}}
### 前次执行的问题记录
{{debug_content}}
{{/if}}

## 执行要求

1. **理解需求**: 仔细阅读任务目标和关键结果
2. **设计方案**: 根据项目架构和现有代码，设计实现方案
3. **编写代码**: 实现所有必要的功能
4. **测试验证**: 按照测试方法验证功能的正确性
5. **提交代码**: 使用 git 提交代码，提交信息应该清晰明确

## 代码质量要求

- 遵循项目的代码风格规范
- 添加必要的注释和文档
- 确保代码可读性和可维护性
- 避免代码重复（DRY 原则）
- 处理错误情况
```

---

## Step 2: 创建自定义模板

### 场景 1: 为 Web 开发定制模板

```bash
# 创建自定义模板目录
mkdir -p ~/.rick/custom_templates/web

# 创建 Web 开发专用的 doing 模板
cat > ~/.rick/custom_templates/web/doing.md << 'EOF'
# Web 开发任务执行提示词

你是一个资深的 Web 开发工程师，精通前端和后端技术。

## 任务信息

**任务 ID**: {{task_id}}
**任务名称**: {{task_name}}

### 任务目标
{{task_goal}}

### 关键结果
{{key_results}}

## Web 开发最佳实践

### 前端开发
- 使用 React/Vue/Angular 现代框架
- 遵循组件化开发原则
- 确保响应式设计
- 优化性能（懒加载、代码分割）
- 确保可访问性（WCAG 2.1）

### 后端开发
- RESTful API 设计原则
- 输入验证和安全性
- 错误处理和日志记录
- 数据库查询优化
- 缓存策略

### 安全性
- 防止 SQL 注入
- 防止 XSS 攻击
- 防止 CSRF 攻击
- 使用 HTTPS
- 实施认证和授权

### 测试
- 单元测试（Jest/Mocha）
- 集成测试
- E2E 测试（Cypress/Playwright）
- 测试覆盖率 > 80%

## 执行要求

1. **分析需求**: 理解 Web 应用的用户需求
2. **设计 API**: 设计清晰的 API 接口
3. **实现前端**: 创建响应式 UI 组件
4. **实现后端**: 实现业务逻辑和数据处理
5. **编写测试**: 确保功能正确性
6. **性能优化**: 优化加载时间和响应速度

## 代码质量要求

- 遵循 ESLint/Prettier 代码规范
- 使用 TypeScript 提供类型安全
- 添加 JSDoc 注释
- 实施代码审查检查清单

{{#if retry_count > 0}}
### 前次执行的问题
{{debug_content}}
{{/if}}
EOF
```

### 场景 2: 为数据科学定制模板

```bash
mkdir -p ~/.rick/custom_templates/datascience

cat > ~/.rick/custom_templates/datascience/doing.md << 'EOF'
# 数据科学任务执行提示词

你是一个资深的数据科学家，精通机器学习和数据分析。

## 任务信息

**任务 ID**: {{task_id}}
**任务名称**: {{task_name}}

### 任务目标
{{task_goal}}

## 数据科学工作流

### 1. 数据探索（EDA）
- 数据加载和检查
- 描述性统计分析
- 数据可视化
- 识别缺失值和异常值

### 2. 数据预处理
- 数据清洗
- 特征工程
- 数据标准化/归一化
- 数据分割（训练集/验证集/测试集）

### 3. 模型开发
- 选择合适的算法
- 模型训练
- 超参数调优
- 交叉验证

### 4. 模型评估
- 评估指标（准确率、召回率、F1-score）
- 混淆矩阵
- ROC 曲线和 AUC
- 特征重要性分析

### 5. 模型部署
- 模型序列化
- API 接口设计
- 性能监控

## 代码质量要求

- 使用 Jupyter Notebook 进行探索
- 使用 scikit-learn/TensorFlow/PyTorch
- 添加详细的注释和 Markdown 说明
- 可重现性（设置随机种子）
- 版本控制（DVC）

{{#if retry_count > 0}}
### 前次执行的问题
{{debug_content}}
{{/if}}
EOF
```

---

## Step 3: 配置自定义模板

### 方法 1: 修改配置文件

```bash
# 编辑 Rick 配置
vim ~/.rick/config.json
```

```json
{
  "version": "0.1.0",
  "template_dir": "~/.rick/custom_templates/web",
  "templates": {
    "plan": "plan.md",
    "doing": "doing.md",
    "test": "test.md",
    "learning": "learning.md"
  }
}
```

### 方法 2: 使用环境变量

```bash
# 设置自定义模板目录
export RICK_TEMPLATE_DIR=~/.rick/custom_templates/web

# 使用自定义模板
rick plan "创建登录页面"
rick doing job_0
```

### 方法 3: 命令行参数（需要扩展 Rick）

```bash
# 使用特定模板
rick doing job_0 --template-dir ~/.rick/custom_templates/datascience
```

---

## Step 4: 优化提示词

### 优化技巧 1: 添加项目特定上下文

```markdown
## 项目特定约束

### 技术栈
- 后端: Go 1.21+
- 数据库: PostgreSQL 14
- 缓存: Redis 7
- 消息队列: RabbitMQ

### 代码规范
- 使用 gofmt 格式化代码
- 遵循 Effective Go 指南
- 错误处理使用 errors.Wrap
- 日志使用 slog 包

### 性能要求
- API 响应时间 < 100ms (P95)
- 数据库查询 < 50ms
- 并发支持 > 10000 QPS
```

### 优化技巧 2: 添加示例代码

```markdown
## 代码示例

### HTTP 处理器示例

\`\`\`go
func (h *Handler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("id")
    if userID == "" {
        http.Error(w, "missing user id", http.StatusBadRequest)
        return
    }

    user, err := h.userService.GetUser(r.Context(), userID)
    if err != nil {
        h.logger.Error("failed to get user", "error", err, "user_id", userID)
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(user)
}
\`\`\`

请遵循类似的模式实现新的处理器。
```

### 优化技巧 3: 添加检查清单

```markdown
## 代码审查检查清单

在提交代码前，请确保：

- [ ] 所有测试通过 (`go test ./...`)
- [ ] 代码已格式化 (`gofmt -w .`)
- [ ] 无 linter 警告 (`golangci-lint run`)
- [ ] 添加了必要的注释
- [ ] 更新了相关文档
- [ ] 错误处理完善
- [ ] 日志记录适当
- [ ] 性能测试通过
- [ ] 安全性检查通过
```

---

## Step 5: 测试自定义模板

### 测试流程

```bash
# 1. 设置自定义模板
export RICK_TEMPLATE_DIR=~/.rick/custom_templates/web

# 2. 创建测试任务
rick plan "创建用户注册 API"

# 3. 查看生成的提示词（调试模式）
rick doing job_0 --debug

# 4. 验证代码质量
go test ./...
golangci-lint run
```

### 对比效果

```bash
# 使用默认模板
rick plan "创建 API"
rick doing job_0

# 使用自定义模板
export RICK_TEMPLATE_DIR=~/.rick/custom_templates/web
rick plan "创建 API"
rick doing job_1

# 对比生成的代码
diff -r .rick/jobs/job_0/ .rick/jobs/job_1/
```

---

## 🎓 学习要点

### 1. 提示词设计原则

- **清晰性**: 提示词应该清晰明确，避免歧义
- **完整性**: 包含所有必要的上下文信息
- **结构化**: 使用标题、列表、代码块等结构化元素
- **示例性**: 提供具体的代码示例
- **可扩展性**: 支持动态内容插入

### 2. 上下文管理

| 上下文类型 | 来源 | 用途 |
|-----------|------|------|
| 项目信息 | SPEC.md, README.md | 理解项目背景 |
| 架构信息 | architecture.md | 理解系统设计 |
| 任务信息 | task.md | 理解具体任务 |
| 历史信息 | debug.md, learning/ | 学习历史经验 |
| 依赖信息 | tasks.json | 理解任务关系 |

### 3. 模板变量

```markdown
{{task_id}}           # 任务 ID
{{task_name}}         # 任务名称
{{task_goal}}         # 任务目标
{{key_results}}       # 关键结果
{{test_methods}}      # 测试方法
{{retry_count}}       # 重试次数
{{debug_content}}     # 调试信息
{{completed_tasks}}   # 已完成任务
{{task_dependencies}} # 任务依赖
{{spec_content}}      # SPEC 内容
{{architecture_content}} # 架构内容
```

---

## 💡 高级技巧

### 技巧 1: 多模板管理

```bash
# 创建模板库
mkdir -p ~/.rick/template_library/{web,datascience,devops,mobile}

# 创建模板切换脚本
cat > ~/.rick/switch_template.sh << 'EOF'
#!/bin/bash
TEMPLATE=$1
TEMPLATE_DIR=~/.rick/template_library/$TEMPLATE

if [ ! -d "$TEMPLATE_DIR" ]; then
    echo "Template not found: $TEMPLATE"
    exit 1
fi

echo "export RICK_TEMPLATE_DIR=$TEMPLATE_DIR" > ~/.rick/current_template
echo "Switched to template: $TEMPLATE"
EOF

chmod +x ~/.rick/switch_template.sh

# 使用
~/.rick/switch_template.sh web
source ~/.rick/current_template
rick plan "创建 Web 应用"
```

### 技巧 2: 模板继承

```bash
# 基础模板
cat > ~/.rick/custom_templates/base/doing.md << 'EOF'
# 基础执行提示词

## 任务信息
{{task_id}}: {{task_name}}

## 通用要求
- 代码质量
- 测试覆盖
- 文档完整
EOF

# Web 模板（继承基础模板）
cat > ~/.rick/custom_templates/web/doing.md << 'EOF'
{{include "base/doing.md"}}

## Web 特定要求
- 响应式设计
- 安全性
- 性能优化
EOF
```

### 技巧 3: 动态模板生成

```bash
# 创建模板生成器
cat > ~/.rick/generate_template.sh << 'EOF'
#!/bin/bash
PROJECT_TYPE=$1
OUTPUT_DIR=~/.rick/custom_templates/$PROJECT_TYPE

mkdir -p $OUTPUT_DIR

# 根据项目类型生成模板
case $PROJECT_TYPE in
    web)
        # 生成 Web 项目模板
        ;;
    api)
        # 生成 API 项目模板
        ;;
    cli)
        # 生成 CLI 项目模板
        ;;
esac
EOF
```

---

## 🚀 下一步

恭喜掌握提示词定制！接下来你可以：

1. **[Tutorial 5: CI/CD 集成](./tutorial-5-cicd-integration.md)** - 集成到 CI/CD 流程
2. **[最佳实践](../best-practices.md)** - 学习提示词设计的最佳实践
3. **[Prompt Manager 模块](../modules/prompt_manager.md)** - 深入理解提示词管理

---

*最后更新: 2026-03-14*
