# 文档工程三阶段法

## 技能描述

文档工程三阶段法是一种系统化的文档生成方法论，通过"建立标准 → 批量生产 → 质量保证"三个阶段，高效生成大量高质量的技术文档。在 Job 1 中，我们使用这一方法成功生成了 16 个文档（10,657 行），零重试率达到 100%。

**核心理念**: 先建立文档标准和模板，然后基于标准批量生产文档，最后通过自动化工具进行质量保证。

## 适用场景

- **大规模文档生成**: 需要生成数十个甚至上百个文档的场景
- **多模块系统文档**: 需要为多个模块生成结构相似的文档
- **API 文档生成**: 需要为多个 API 生成标准化文档
- **项目文档体系建设**: 从零开始建立完整的项目文档体系
- **文档标准化改造**: 将现有的非标准文档改造为标准化文档

## 实现模式

### 阶段1: 建立标准（Foundation Phase）

**目标**: 创建文档标准、模板和基础文档

**关键活动**:
1. **创建目录结构**: 定义文档的组织方式
2. **编写索引文档**: 提供文档导航和概览
3. **建立文档模板**: 定义文档的标准结构和格式
4. **生成基础文档**: 创建架构、流程等全局性文档

**Job 1 示例**:
- Task 1: 创建 Wiki 目录结构和索引文件（README.md, 150 行）
- Task 2: 编写架构概览文档（architecture.md, 741 行）
- Task 3: 编写运行时流程文档（runtime-flow.md, 900 行）

**产出**:
- 文档目录结构
- 文档索引和导航
- 文档模板和标准
- 2-3 个基础文档

**关键成功因素**:
- ✅ **标准先行**: 在批量生产前建立清晰的标准
- ✅ **模板化**: 定义可复用的文档模板
- ✅ **全局视图**: 提供系统架构和流程的全局视图

**示例** (文档模板):

```markdown
# 模块名称

## 模块职责

简要描述模块的核心职责和定位。

## 核心类型

### 类型1: 名称

**定义**:
\`\`\`go
type Example struct {
    Field1 string
    Field2 int
}
\`\`\`

**说明**: 类型的用途和关键字段

### 类型2: 名称

...

## 关键函数

### 函数1: 名称

**签名**:
\`\`\`go
func Example(param1 string, param2 int) (result string, err error)
\`\`\`

**功能**: 函数的功能描述

**参数**:
- `param1`: 参数说明
- `param2`: 参数说明

**返回值**:
- `result`: 返回值说明
- `err`: 错误说明

**示例**:
\`\`\`go
result, err := Example("test", 123)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result)
\`\`\`

### 函数2: 名称

...

## 类图

\`\`\`mermaid
classDiagram
    class Example {
        +Field1 string
        +Field2 int
        +Method1() error
        +Method2() string
    }
\`\`\`

## 使用示例

### 示例1: 场景描述

\`\`\`go
// 示例代码
\`\`\`

说明...

### 示例2: 场景描述

...

## 注意事项

- 注意事项1
- 注意事项2

## 相关模块

- 模块1: 关系说明
- 模块2: 关系说明
```

### 阶段2: 批量生产（Production Phase）

**目标**: 基于标准和模板，批量生成大量文档

**关键活动**:
1. **并行生成相似文档**: 使用模板批量生成结构相似的文档
2. **串行生成依赖文档**: 基于前序文档生成有依赖关系的文档
3. **保持一致性**: 确保所有文档遵循相同的标准和格式

**Job 1 示例**:
- Task 4: 编写核心模块文档（7 个模块，3,329 行）
- Task 5: 编写 DAG 执行引擎详解（1,092 行）
- Task 6: 编写提示词管理系统文档（1,383 行）
- Task 7: 编写测试与验证文档（1,339 行）
- Task 8: 编写安装与部署文档（1,475 行）

**产出**:
- 10+ 个模块/专题文档
- 5,000+ 行文档内容
- 20+ 个图表
- 100+ 个代码示例

**关键成功因素**:
- ✅ **模板复用**: 使用统一的模板生成相似文档
- ✅ **上下文继承**: 后续文档可以引用前序文档的内容
- ✅ **并行化**: 无依赖的文档可以并行生成

**示例** (批量生成任务):

```markdown
# 任务名称
编写核心模块文档

# 任务目标
在 `.rick/wiki/modules/` 目录下创建 7 个核心模块的详细文档。

# 关键结果
1. 创建 `modules/` 目录
2. 生成 7 个模块文档：cmd.md, workspace.md, parser.md, executor.md, prompt.md, git.md, config.md
3. 每个文档至少 100 行，总计至少 500 行
4. 每个文档包含：
   - 模块职责
   - 核心类型（Go 代码）
   - 关键函数（Go 代码 + 示例）
   - 类图（Mermaid）
   - 使用示例（至少 4 个）

# 测试方法
1. 检查 `modules/` 目录是否存在
2. 检查 7 个文档是否都存在
3. 统计总行数是否 >= 500 行
4. 检查每个文档是否包含 Mermaid 类图
5. 检查每个文档是否包含至少 4 个代码示例
```

**关键点**:
- ✅ **批量生成**: 一次任务生成 7 个相似文档
- ✅ **统一标准**: 所有文档遵循相同的结构
- ✅ **可验证**: 所有要求都可以自动化验证

### 阶段3: 质量保证（Quality Assurance Phase）

**目标**: 验证文档质量，修复问题，建立持续改进机制

**关键活动**:
1. **创建验证工具**: 开发自动化验证脚本
2. **全面验证**: 检查文档完整性、格式、链接、图表等
3. **生成验证报告**: 提供详细的质量报告
4. **修复问题**: 根据验证结果修复问题
5. **建立贡献指南**: 为后续文档维护提供指导

**Job 1 示例**:
- Task 9: 验证和完善 Wiki 文档
  - 创建 validate_wiki.sh（186 行，6 个验证模块）
  - 生成 VALIDATION_REPORT.md（283 行）
  - 创建 CONTRIBUTING.md（447 行）
  - 修复链接和格式问题

**产出**:
- 自动化验证工具
- 详细的验证报告
- 贡献指南
- 修复后的高质量文档

**关键成功因素**:
- ✅ **自动化验证**: 使用脚本自动检查文档质量
- ✅ **全面覆盖**: 检查多个维度（完整性、格式、链接、图表）
- ✅ **持续改进**: 建立贡献指南，支持后续维护

**示例** (验证脚本):

```bash
#!/bin/bash

# 验证 Wiki 文档质量

set -e

WIKI_DIR="wiki"
REPORT_FILE="wiki/VALIDATION_REPORT.md"

echo "# Wiki 文档验证报告" > "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "生成时间: $(date)" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# 1. 文档完整性检查
echo "## 1. 文档完整性检查" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

required_files=(
    "README.md"
    "architecture.md"
    "runtime-flow.md"
    "modules/cmd.md"
    "modules/workspace.md"
    "modules/parser.md"
    "modules/executor.md"
    "modules/prompt.md"
    "modules/git.md"
    "modules/config.md"
    "dag-engine.md"
    "prompt-system.md"
    "testing.md"
    "installation.md"
    "CONTRIBUTING.md"
)

missing_files=()
for file in "${required_files[@]}"; do
    if [ ! -f "$WIKI_DIR/$file" ]; then
        missing_files+=("$file")
    fi
done

if [ ${#missing_files[@]} -eq 0 ]; then
    echo "✅ 所有必需文档都存在" >> "$REPORT_FILE"
else
    echo "❌ 缺失文档: ${missing_files[*]}" >> "$REPORT_FILE"
    exit 1
fi

echo "" >> "$REPORT_FILE"

# 2. 文档行数统计
echo "## 2. 文档行数统计" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

total_lines=0
for file in "${required_files[@]}"; do
    if [ -f "$WIKI_DIR/$file" ]; then
        lines=$(wc -l < "$WIKI_DIR/$file")
        total_lines=$((total_lines + lines))
        echo "- $file: $lines 行" >> "$REPORT_FILE"
    fi
done

echo "" >> "$REPORT_FILE"
echo "**总计**: $total_lines 行" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

if [ $total_lines -lt 1500 ]; then
    echo "❌ 总行数不足 1,500 行（当前: $total_lines）" >> "$REPORT_FILE"
    exit 1
else
    echo "✅ 总行数达标（当前: $total_lines, 要求: >= 1,500）" >> "$REPORT_FILE"
fi

echo "" >> "$REPORT_FILE"

# 3. Mermaid 图表统计
echo "## 3. Mermaid 图表统计" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

total_diagrams=0
for file in "${required_files[@]}"; do
    if [ -f "$WIKI_DIR/$file" ]; then
        diagrams=$(grep -c '```mermaid' "$WIKI_DIR/$file" || true)
        if [ $diagrams -gt 0 ]; then
            total_diagrams=$((total_diagrams + diagrams))
            echo "- $file: $diagrams 个图表" >> "$REPORT_FILE"
        fi
    fi
done

echo "" >> "$REPORT_FILE"
echo "**总计**: $total_diagrams 个图表" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

if [ $total_diagrams -lt 10 ]; then
    echo "❌ 图表数量不足 10 个（当前: $total_diagrams）" >> "$REPORT_FILE"
    exit 1
else
    echo "✅ 图表数量达标（当前: $total_diagrams, 要求: >= 10）" >> "$REPORT_FILE"
fi

echo "" >> "$REPORT_FILE"

# 4. 代码示例统计
echo "## 4. 代码示例统计" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

total_examples=0
for file in "${required_files[@]}"; do
    if [ -f "$WIKI_DIR/$file" ]; then
        examples=$(grep -c '```go\|```bash\|```python\|```json' "$WIKI_DIR/$file" || true)
        if [ $examples -gt 0 ]; then
            total_examples=$((total_examples + examples))
            echo "- $file: $examples 个代码示例" >> "$REPORT_FILE"
        fi
    fi
done

echo "" >> "$REPORT_FILE"
echo "**总计**: $total_examples 个代码示例" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# 5. 链接检查
echo "## 5. 链接检查" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

broken_links=()
for file in "${required_files[@]}"; do
    if [ -f "$WIKI_DIR/$file" ]; then
        # 提取 Markdown 链接
        links=$(grep -oP '\[.*?\]\(\K[^)]+' "$WIKI_DIR/$file" || true)
        for link in $links; do
            # 检查内部链接
            if [[ $link == *.md* ]] && [[ $link != http* ]]; then
                # 移除锚点
                link_file=$(echo "$link" | cut -d'#' -f1)
                if [ ! -f "$WIKI_DIR/$link_file" ]; then
                    broken_links+=("$file -> $link")
                fi
            fi
        done
    fi
done

if [ ${#broken_links[@]} -eq 0 ]; then
    echo "✅ 所有内部链接都有效" >> "$REPORT_FILE"
else
    echo "❌ 发现断链:" >> "$REPORT_FILE"
    for link in "${broken_links[@]}"; do
        echo "  - $link" >> "$REPORT_FILE"
    done
    exit 1
fi

echo "" >> "$REPORT_FILE"

# 6. 格式检查
echo "## 6. 格式检查" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

format_issues=()
for file in "${required_files[@]}"; do
    if [ -f "$WIKI_DIR/$file" ]; then
        # 检查是否有标题
        if ! grep -q '^# ' "$WIKI_DIR/$file"; then
            format_issues+=("$file: 缺少一级标题")
        fi

        # 检查是否有空行分隔
        if grep -q '^##[^#]' "$WIKI_DIR/$file" && ! grep -q '^$' "$WIKI_DIR/$file"; then
            format_issues+=("$file: 缺少空行分隔")
        fi
    fi
done

if [ ${#format_issues[@]} -eq 0 ]; then
    echo "✅ 所有文档格式正确" >> "$REPORT_FILE"
else
    echo "⚠️ 发现格式问题:" >> "$REPORT_FILE"
    for issue in "${format_issues[@]}"; do
        echo "  - $issue" >> "$REPORT_FILE"
    done
fi

echo "" >> "$REPORT_FILE"

# 总结
echo "## 总结" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "- 文档数量: ${#required_files[@]}" >> "$REPORT_FILE"
echo "- 总行数: $total_lines" >> "$REPORT_FILE"
echo "- 图表数量: $total_diagrams" >> "$REPORT_FILE"
echo "- 代码示例: $total_examples" >> "$REPORT_FILE"
echo "- 断链数量: ${#broken_links[@]}" >> "$REPORT_FILE"
echo "- 格式问题: ${#format_issues[@]}" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

if [ ${#broken_links[@]} -eq 0 ] && [ ${#format_issues[@]} -eq 0 ]; then
    echo "✅ **质量评级: 优秀**" >> "$REPORT_FILE"
elif [ ${#broken_links[@]} -eq 0 ] && [ ${#format_issues[@]} -le 5 ]; then
    echo "✅ **质量评级: 良好**" >> "$REPORT_FILE"
else
    echo "⚠️ **质量评级: 需要改进**" >> "$REPORT_FILE"
fi

echo "" >> "$REPORT_FILE"
echo "验证完成！" >> "$REPORT_FILE"

cat "$REPORT_FILE"
```

## 最佳实践

### 1. 标准先行

**原则**: 在批量生产前建立清晰的文档标准

**理由**:
- 避免后期大规模返工
- 确保文档一致性
- 提高批量生产效率

**实践**:
1. 创建文档模板
2. 编写 2-3 个示例文档
3. 基于示例文档提炼标准
4. 在批量生产前验证标准的有效性

### 2. 模板化

**原则**: 为相似文档创建可复用的模板

**理由**:
- 提高生成效率
- 确保结构一致性
- 降低维护成本

**实践**:
1. 识别相似文档的共同结构
2. 提取可变部分（如模块名称、函数签名）
3. 创建模板，使用占位符表示可变部分
4. 在批量生产时替换占位符

### 3. 渐进式验证

**原则**: 在每个阶段结束时进行验证，而不是等到最后

**理由**:
- 及早发现问题
- 降低修复成本
- 避免问题累积

**实践**:
1. 阶段1结束时验证标准和模板
2. 阶段2中定期验证已生成的文档
3. 阶段3进行全面验证和修复

### 4. 自动化优先

**原则**: 优先使用自动化工具进行验证和生成

**理由**:
- 提高效率
- 确保一致性
- 降低人工成本

**实践**:
1. 创建自动化验证脚本
2. 使用模板引擎生成文档
3. 集成到 CI/CD 流程

### 5. 持续改进

**原则**: 建立文档维护和改进机制

**理由**:
- 文档需要随着项目演进而更新
- 避免文档过时
- 提高文档长期价值

**实践**:
1. 创建贡献指南
2. 定期审查和更新文档
3. 收集用户反馈
4. 持续优化文档标准

## 常见陷阱

### ❌ 陷阱1: 跳过标准建立阶段

**问题**: 直接进入批量生产，没有建立标准和模板

**后果**: 文档结构不一致，后期需要大规模返工

**避免方法**:
- 严格遵循三阶段法
- 在批量生产前建立清晰的标准
- 通过示例文档验证标准的有效性

### ❌ 陷阱2: 过度并行化

**问题**: 将有依赖关系的文档设计为并行生成

**后果**: 文档之间可能产生不一致或冲突

**避免方法**:
- 仔细分析文档之间的依赖关系
- 只有在确实无依赖时才并行化
- 使用汇聚点设计确保阶段性成果的完整性

### ❌ 陷阱3: 忽视质量保证

**问题**: 批量生产后不进行验证，直接发布

**后果**: 文档存在大量错误（断链、格式问题、内容不完整）

**避免方法**:
- 创建自动化验证工具
- 在发布前进行全面验证
- 修复所有发现的问题

### ❌ 陷阱4: 缺乏持续改进机制

**问题**: 文档生成后不再维护，逐渐过时

**后果**: 文档失去价值，用户无法信任

**避免方法**:
- 创建贡献指南
- 定期审查和更新文档
- 建立用户反馈机制

### ❌ 陷阱5: 过度追求数量

**问题**: 只关注文档数量和行数，忽视质量

**后果**: 文档冗长、可读性差、实用性低

**避免方法**:
- 在标准中明确"简洁性"要求
- 在验证阶段检查文档质量
- 优先保证文档的实用性和准确性

## 测试建议

### 1. 标准验证

在阶段1结束时，验证文档标准和模板的有效性。

**验证内容**:
- 模板结构是否清晰
- 示例文档是否符合标准
- 标准是否可复制

### 2. 批量验证

在阶段2中，定期验证已生成的文档。

**验证内容**:
- 文档结构是否一致
- 文档内容是否完整
- 文档是否符合标准

### 3. 全面验证

在阶段3中，进行全面的质量验证。

**验证内容**:
- 文档完整性
- 文档格式
- 链接有效性
- 图表质量
- 代码示例正确性

## 相关技能

- **零重试任务设计模式**: 如何设计高成功率的任务
- **DAG 任务分解方法**: 如何设计合理的 DAG 任务图
- **自动化验证**: 如何创建高效的验证工具

## 参考资料

- Job 1 执行总结: `.rick/jobs/job_1/learning/SUMMARY.md`
- 文档模板: `.rick/jobs/job_1/plan/tasks/*.md`
- 验证脚本: `wiki/validate_wiki.sh`

---

## 成功案例

**Job 1: Wiki 文档创建**

**规模**:
- 16 个文档
- 10,657 行
- 33 个 Mermaid 图表
- 150+ 代码示例

**执行效率**:
- 零重试率: 100% (9/9)
- 执行时长: ~2 小时
- 平均每个文档: 10-15 分钟

**质量评级**: ⭐⭐⭐⭐⭐ 优秀

**关键成功因素**:
1. **阶段1**: 创建清晰的目录结构和索引，建立文档标准
2. **阶段2**: 基于标准批量生成模块和专题文档
3. **阶段3**: 创建自动化验证工具，全面验证文档质量

这次成功充分验证了文档工程三阶段法的有效性。
