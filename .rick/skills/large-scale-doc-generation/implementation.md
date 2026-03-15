# 大规模文档生成 - 实现细节

> 基于 Job 0 实践的完整实现指南

## 实现架构

### 整体流程

```
Phase 1: 定义标准
  ├─ task1: 生成 OKR.md
  └─ task2: 生成 SPEC.md
       ↓
Phase 2: 统一模板
  ├─ task3: 创建 Wiki 索引
  ├─ task4: 创建模块文档
  └─ task6: 创建示例文档
       ↓
Phase 3: 提取技能
  └─ task5: 提取 Skills 库
       ↓
Phase 4: 自动验证
  └─ task7: 验证完整性
```

## Phase 1: 定义标准

### Step 1.1: 生成 OKR.md

**目标**: 定义项目目标和关键结果

**实现代码**（task.md）:

```markdown
# 依赖关系
（无）

# 任务名称
分析项目架构并生成 OKR.md

# 任务目标
为 Rick CLI 项目创建完整的 OKR 文档，明确项目目标和关键结果

# 关键结果
1. 生成 `.rick/OKR.md` 文件（≥ 400 行）
2. 包含项目愿景（200+ 字）
3. 包含 5 个核心目标（O1-O5）
4. 每个目标包含 5 个关键结果（KR1.1-KR1.5）
5. 包含成功指标、时间线、风险分析

# 测试方法
1. 检查文件存在
   ```bash
   test -f .rick/OKR.md || exit 1
   ```

2. 检查文档长度 ≥ 400 行
   ```bash
   lines=$(wc -l < .rick/OKR.md)
   test $lines -ge 400 || exit 1
   ```

3. 检查包含 5 个核心目标
   ```bash
   grep -c "^### O[1-5]:" .rick/OKR.md | grep -q "5" || exit 1
   ```

4. 检查每个目标有关键结果
   ```bash
   grep -c "^- \*\*KR" .rick/OKR.md | grep -q "[2-3][0-9]" || exit 1
   ```
```

**OKR.md 模板**:

```markdown
# 项目名称 - OKR

> 项目愿景（200+ 字）

## 项目概述
- 项目名称
- 项目定位
- 核心理念
- 技术栈
- 当前规模
- 版本

## 项目愿景（Vision）
详细描述项目的长期愿景（200+ 字）

## 核心目标（Objectives）

### O1: 目标1（时间线）
**优先级**: P0/P1/P2
**负责人**: Team Name
**时间线**: YYYY-MM 至 YYYY-MM

#### 关键结果（Key Results）
- **KR1.1**: 关键结果1
  - 指标1
  - 指标2
  - **目标**: 具体目标值
  - **当前进度**: 当前状态

- **KR1.2**: 关键结果2
  ...

### O2-O5: 其他目标
...

## 成功指标（Success Metrics）
### 核心指标（North Star Metrics）
### 质量指标（Quality Metrics）
### 技术指标（Technical Metrics）

## 时间线（Timeline）
### 2026 Q1-Q4
...

## 核心价值主张（Value Propositions）
### 1. 价值主张1
### 2. 价值主张2
...

## 风险与挑战（Risks & Challenges）
### 技术风险
### 产品风险
### 团队风险

## 下一步行动（Next Actions）
### 短期（1-3 个月）
### 中期（3-6 个月）
### 长期（6-12 个月）
```

### Step 1.2: 生成 SPEC.md

**目标**: 定义技术规范和架构设计

**实现代码**（task.md）:

```markdown
# 依赖关系
（无）

# 任务名称
分析代码规范并生成 SPEC.md

# 任务目标
为 Rick CLI 项目创建完整的技术规范文档

# 关键结果
1. 生成 `.rick/SPEC.md` 文件（≥ 800 行）
2. 包含架构设计（模块划分、数据流）
3. 包含技术选型（语言、框架、库）
4. 包含开发规范（代码风格、测试、Git）
5. 包含性能、安全、CI/CD 指南

# 测试方法
1. 检查文件存在
   ```bash
   test -f .rick/SPEC.md || exit 1
   ```

2. 检查文档长度 ≥ 800 行
   ```bash
   lines=$(wc -l < .rick/SPEC.md)
   test $lines -ge 800 || exit 1
   ```

3. 检查关键章节存在
   ```bash
   grep -q "## 架构设计" .rick/SPEC.md || exit 1
   grep -q "## 技术选型" .rick/SPEC.md || exit 1
   grep -q "## 开发规范" .rick/SPEC.md || exit 1
   ```
```

**SPEC.md 模板**:

```markdown
# 项目名称 - 技术规范

> 项目技术规范和架构设计文档

## 项目概述
- 项目简介
- 技术栈
- 依赖关系

## 架构设计

### 系统架构
#### 整体架构图
#### 模块划分
#### 数据流

### 目录结构
```
project/
├── cmd/
├── internal/
└── scripts/
```

### 核心模块

#### 模块1
- 职责
- 核心类型
- 主要接口

#### 模块2-N
...

## 技术选型

### 编程语言
### 框架和库
### 工具链

## 开发规范

### 代码风格
#### 命名规范
#### 注释规范
#### 错误处理

### 测试标准
#### 单元测试
#### 集成测试
#### E2E 测试

### Git 工作流
#### 分支策略
#### Commit 规范
#### PR 流程

## 文档规范
## 发布流程
## 性能指南
## 安全指南
## CI/CD 流程
## 代码审查
```

## Phase 2: 统一模板

### Step 2.1: 创建 Wiki 索引

**目标**: 创建 Wiki 知识库的索引和架构文档

**实现代码**（task.md）:

```markdown
# 依赖关系
task1

# 任务名称
创建 Wiki 知识库索引和架构文档

# 任务目标
创建 Wiki 知识库的索引、架构文档、核心概念文档

# 关键结果
1. 创建 `.rick/wiki/index.md`（主索引）
2. 创建 `.rick/wiki/architecture.md`（架构文档）
3. 创建 `.rick/wiki/core-concepts.md`（核心概念）
4. 创建 `.rick/wiki/README.md`（Wiki 介绍）
5. 所有文档遵循 OKR 定义的目标

# 测试方法
1. 检查文件存在
   ```bash
   test -f .rick/wiki/index.md || exit 1
   test -f .rick/wiki/architecture.md || exit 1
   test -f .rick/wiki/core-concepts.md || exit 1
   ```

2. 检查索引包含所有模块链接
   ```bash
   grep -c "\[.*\](.*/modules/.*\.md)" .rick/wiki/index.md | grep -q "[8-9]" || exit 1
   ```
```

**Wiki 索引模板**（index.md）:

```markdown
# 项目名称 Wiki

> 完整的项目知识库

## 快速开始
- [快速入门](./getting-started.md)
- [核心概念](./core-concepts.md)
- [架构设计](./architecture.md)

## 核心模块

### 基础设施
- [Infrastructure 模块](./modules/infrastructure.md)
- [Config 模块](./modules/config.md)
- [Logging 模块](./modules/logging.md)

### 核心功能
- [Parser 模块](./modules/parser.md)
- [DAG Executor 模块](./modules/dag_executor.md)
- [Prompt Manager 模块](./modules/prompt_manager.md)

### 集成
- [Git 模块](./modules/git.md)
- [CallCLI 模块](./modules/callcli.md)

## 教程和示例
- [教程1: 简单项目](./tutorials/tutorial-1.md)
- [教程2: 自我重构](./tutorials/tutorial-2.md)
...

## 最佳实践
- [任务设计最佳实践](./task-design-best-practices.md)
- [提示词工程](./prompt-engineering.md)

## 参考资料
- [OKR](../OKR.md)
- [SPEC](../SPEC.md)
- [Skills](../skills/index.md)
```

### Step 2.2: 创建模块文档

**目标**: 为所有核心模块创建文档

**实现代码**（task.md）:

```markdown
# 依赖关系
task2

# 任务名称
分析核心模块并完善 Wiki 模块文档

# 任务目标
为 8 个核心模块创建完整的 Wiki 文档

# 关键结果
1. 创建 8 个模块文档（每个 ≥ 200 行）
2. 每个文档包含：概述、核心类型、主要函数、使用示例、常见问题
3. 每个文档包含 ≥ 5 个代码示例
4. 所有文档遵循 SPEC 定义的规范

# 测试方法
1. 检查模块文档数量
   ```bash
   count=$(ls -1 .rick/wiki/modules/*.md | wc -l)
   test $count -ge 8 || exit 1
   ```

2. 检查每个文档长度 ≥ 200 行
   ```bash
   for file in .rick/wiki/modules/*.md; do
     lines=$(wc -l < "$file")
     test $lines -ge 200 || exit 1
   done
   ```

3. 检查代码示例数量
   ```bash
   grep -c '```go' .rick/wiki/modules/*.md | grep -q "[5-9][0-9]" || exit 1
   ```
```

**模块文档模板**（modules/module-name.md）:

```markdown
# 模块名称

> 模块简介（1-2 句话）

## 目录
- [概述](#概述)
- [核心类型](#核心类型)
- [主要函数](#主要函数)
- [使用示例](#使用示例)
- [常见问题](#常见问题)
- [相关模块](#相关模块)

## 概述

### 模块职责
- 职责1
- 职责2
- 职责3

### 模块位置
```
internal/
└── module-name/
    ├── type1.go
    ├── type2.go
    └── util.go
```

### 模块依赖
- 依赖模块1
- 依赖模块2

## 核心类型

### 类型1

```go
type Type1 struct {
    Field1 string
    Field2 int
}
```

**字段说明**:
- `Field1`: 字段1说明
- `Field2`: 字段2说明

### 类型2-N
...

## 主要函数

### 函数1

```go
func Function1(arg1 string, arg2 int) (result string, err error)
```

**参数**:
- `arg1`: 参数1说明
- `arg2`: 参数2说明

**返回值**:
- `result`: 返回值说明
- `err`: 错误信息

**功能说明**:
详细描述函数功能

### 函数2-N
...

## 使用示例

### 示例1: 基本使用

```go
package main

import "internal/module-name"

func main() {
    // 示例代码
}
```

**说明**: 示例说明

### 示例2-N
...

## 常见问题

### Q1: 问题1？
A: 答案1

### Q2: 问题2？
A: 答案2

## 相关模块
- [相关模块1](./related-module1.md)
- [相关模块2](./related-module2.md)

## 参考资料
- [SPEC](../../SPEC.md#相关章节)
- [Skills](../../skills/related-skill/description.md)

---

**文档版本**: v1.0
**最后更新**: YYYY-MM-DD
**维护者**: Team Name
```

## Phase 3: 提取技能

### Step 3.1: 提取 Skills 库

**目标**: 从代码和文档中提取可复用技能

**实现代码**（task.md）:

```markdown
# 依赖关系
task3, task4

# 任务名称
提取可复用技能并创建 Skills 库

# 任务目标
从 Rick CLI 代码和 Wiki 文档中提取 8 个可复用技能

# 关键结果
1. 创建 `.rick/skills/index.md`（技能索引）
2. 提取 8 个技能（每个包含 description.md + implementation.md + examples/）
3. 每个技能的 description.md ≥ 100 行
4. 每个技能的 implementation.md ≥ 100 行
5. 每个技能至少包含 1 个实际应用案例

# 测试方法
1. 检查技能数量
   ```bash
   count=$(ls -1d .rick/skills/*/ | wc -l)
   test $count -ge 8 || exit 1
   ```

2. 检查每个技能的文件完整性
   ```bash
   for skill in .rick/skills/*/; do
     test -f "$skill/description.md" || exit 1
     test -f "$skill/implementation.md" || exit 1
     test -d "$skill/examples" || exit 1
   done
   ```

3. 检查文档长度
   ```bash
   for skill in .rick/skills/*/; do
     lines=$(wc -l < "$skill/description.md")
     test $lines -ge 100 || exit 1
   done
   ```
```

**Skills 索引模板**（skills/index.md）:

```markdown
# Skills 库

> 可复用技能集合

## 技能分类

### 算法与数据结构
- [DAG 拓扑排序](./dag-topological-sort/description.md)

### 设计模式
- [重试模式](./retry-pattern/description.md)
- [工作空间管理](./workspace-management/description.md)

### 解析与处理
- [Markdown 解析](./markdown-parsing/description.md)
- [模板变量提取](./template-variable-extraction/description.md)

### 资源管理
- [Go 资源嵌入](./go-embed-resources/description.md)

### 版本控制
- [Git 自动化](./git-automation/description.md)

### 调试与分析
- [错误分析](./error-analysis/description.md)

## 技能索引表

| 技能名称 | 难度 | 类别 | 应用场景 |
|---------|------|------|---------|
| DAG 拓扑排序 | ⭐⭐⭐ | 算法 | 任务依赖调度 |
| 重试模式 | ⭐⭐ | 设计模式 | 失败重试 |
| Markdown 解析 | ⭐⭐ | 解析 | 文档解析 |
| ... | ... | ... | ... |

## 使用指南

### 如何选择技能
1. 根据场景查找合适的技能
2. 阅读技能描述，了解适用场景和局限
3. 查看实现细节和代码示例
4. 参考实际应用案例

### 如何贡献技能
1. 从实际项目中提取可复用模式
2. 编写 description.md（描述、场景、优缺点）
3. 编写 implementation.md（实现细节、代码示例）
4. 提供至少 1 个实际应用案例
5. 提交 PR 并通过审核
```

**技能文档模板**（skills/skill-name/description.md）:

```markdown
# 技能名称

> 技能简介（1-2 句话）

## 概述
详细描述技能的核心思想和价值

## 使用场景

### 适用场景 ✅
1. 场景1
2. 场景2
3. 场景3

### 不适用场景 ❌
1. 场景1
2. 场景2

## 技能优势
1. 优势1
2. 优势2
3. 优势3

## 技能局限
1. 局限1
2. 局限2

## 实现步骤
1. 步骤1
2. 步骤2
3. 步骤3

## 成功案例
### 案例1: 案例名称
- 背景
- 实施
- 结果

## 最佳实践
### DO ✅
- 实践1
- 实践2

### DON'T ❌
- 反模式1
- 反模式2

## 相关技能
- [相关技能1](../related-skill1/description.md)
- [相关技能2](../related-skill2/description.md)

## 参考资料
- 参考1
- 参考2

---

**技能版本**: v1.0
**最后更新**: YYYY-MM-DD
**验证状态**: ✅ 已验证
```

## Phase 4: 自动验证

### Step 4.1: 编写验证脚本

**目标**: 编写自动化验证脚本

**实现代码**（validate_context.sh）:

```bash
#!/bin/bash

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 计数器
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNINGS=0

# 辅助函数
check_pass() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED_CHECKS++))
    ((TOTAL_CHECKS++))
}

check_fail() {
    echo -e "${RED}✗${NC} $1"
    ((FAILED_CHECKS++))
    ((TOTAL_CHECKS++))
}

check_warn() {
    echo -e "${YELLOW}⚠${NC} $1"
    ((WARNINGS++))
}

echo "======================================"
echo "全局上下文完整性验证"
echo "======================================"
echo ""

# 1. 核心文档验证
echo "1. 核心文档验证"
echo "--------------------------------------"

# 1.1 文件存在性
if [ -f ".rick/OKR.md" ]; then
    check_pass "OKR.md 存在"
else
    check_fail "OKR.md 不存在"
fi

if [ -f ".rick/SPEC.md" ]; then
    check_pass "SPEC.md 存在"
else
    check_fail "SPEC.md 不存在"
fi

# 1.2 文件完整性
if [ -s ".rick/OKR.md" ]; then
    lines=$(wc -l < .rick/OKR.md)
    if [ $lines -ge 400 ]; then
        check_pass "OKR.md 长度符合要求（$lines 行）"
    else
        check_fail "OKR.md 长度不足（$lines < 400 行）"
    fi
else
    check_fail "OKR.md 为空"
fi

# 1.3 Markdown 结构
if grep -q "^# " .rick/OKR.md; then
    check_pass "OKR.md 包含标题"
else
    check_fail "OKR.md 缺少标题"
fi

# 2. Skills 库验证
echo ""
echo "2. Skills 库验证"
echo "--------------------------------------"

# 2.1 Skills 数量
skill_count=$(ls -1d .rick/skills/*/ 2>/dev/null | wc -l)
if [ $skill_count -ge 5 ]; then
    check_pass "Skills 数量符合要求（$skill_count ≥ 5）"
else
    check_fail "Skills 数量不足（$skill_count < 5）"
fi

# 2.2 Skills 文件完整性
for skill in .rick/skills/*/; do
    skill_name=$(basename "$skill")

    if [ -f "$skill/description.md" ]; then
        check_pass "$skill_name: description.md 存在"
    else
        check_fail "$skill_name: description.md 不存在"
    fi

    if [ -f "$skill/implementation.md" ]; then
        check_pass "$skill_name: implementation.md 存在"
    else
        check_fail "$skill_name: implementation.md 不存在"
    fi

    if [ -d "$skill/examples" ]; then
        check_pass "$skill_name: examples/ 目录存在"
    else
        check_warn "$skill_name: examples/ 目录不存在"
    fi
done

# 3. Wiki 知识库验证
echo ""
echo "3. Wiki 知识库验证"
echo "--------------------------------------"

# 3.1 Wiki 文档数量
wiki_count=$(find .rick/wiki -name "*.md" | wc -l)
if [ $wiki_count -ge 8 ]; then
    check_pass "Wiki 文档数量符合要求（$wiki_count ≥ 8）"
else
    check_fail "Wiki 文档数量不足（$wiki_count < 8）"
fi

# 3.2 核心模块文档
modules=("infrastructure" "parser" "dag_executor" "prompt_manager" "cli_commands" "workspace")
for module in "${modules[@]}"; do
    if [ -f ".rick/wiki/modules/$module.md" ]; then
        lines=$(wc -l < ".rick/wiki/modules/$module.md")
        if [ $lines -ge 200 ]; then
            check_pass "$module.md 存在且长度符合要求（$lines ≥ 200）"
        else
            check_warn "$module.md 长度不足（$lines < 200）"
        fi
    else
        check_fail "$module.md 不存在"
    fi
done

# 4. 代码示例验证
echo ""
echo "4. 代码示例验证"
echo "--------------------------------------"

go_examples=$(grep -r '```go' .rick/ | wc -l)
if [ $go_examples -ge 10 ]; then
    check_pass "Go 代码示例数量符合要求（$go_examples ≥ 10）"
else
    check_warn "Go 代码示例数量不足（$go_examples < 10）"
fi

# 5. 关键词覆盖验证
echo ""
echo "5. 关键词覆盖验证"
echo "--------------------------------------"

keywords=("Rick" "Context" "AI Coding" "DAG" "Prompt" "Task")
for keyword in "${keywords[@]}"; do
    count=$(grep -ri "$keyword" .rick/ | wc -l)
    if [ $count -ge 3 ]; then
        check_pass "关键词 '$keyword' 覆盖充分（$count 次）"
    else
        check_warn "关键词 '$keyword' 覆盖不足（$count < 3）"
    fi
done

# 总结
echo ""
echo "======================================"
echo "验证总结"
echo "======================================"
echo "总检查项: $TOTAL_CHECKS"
echo -e "${GREEN}通过: $PASSED_CHECKS${NC}"
echo -e "${RED}失败: $FAILED_CHECKS${NC}"
echo -e "${YELLOW}警告: $WARNINGS${NC}"

if [ $FAILED_CHECKS -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ 验证通过！${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}✗ 验证失败！${NC}"
    exit 1
fi
```

## 总结

### 核心要点

1. **标准先行**: OKR/SPEC 定义标准和规范
2. **统一模板**: 所有文档遵循相同结构
3. **自动验证**: 多维度验证确保质量
4. **任务分解**: 合理的任务粒度和依赖关系

### 成功公式

```
OKR/SPEC 先行 + 统一模板 + 自动验证 = 高质量文档
```

### 参考资料

- [Job 0 执行分析](../../jobs/job_0/learning/summary.md)
- [任务设计最佳实践](../../wiki/task-design-best-practices.md)

---

**文档版本**: v1.0
**最后更新**: 2026-03-15
**作者**: Rick Team
