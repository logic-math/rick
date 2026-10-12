# 零重试任务设计模式

## 技能描述

零重试任务设计模式是一种通过精心设计任务定义、上下文和测试方法，使 AI Agent 能够一次性成功完成任务的设计方法。在 Job 1 中，我们实现了 9/9 任务零重试的教科书级别成功，充分验证了这一模式的有效性。

**核心理念**: 通过提供完整的上下文、清晰的任务定义和明确的测试标准，让 AI Agent 在首次执行时就能准确理解任务目标并正确完成。

## 适用场景

- **复杂任务分解**: 将大型任务分解为多个小任务，每个任务都能一次性完成
- **文档生成任务**: 需要生成大量结构化文档的场景
- **代码生成任务**: 需要生成符合特定规范的代码的场景
- **自动化工作流**: 需要高可靠性的自动化任务执行
- **批量处理任务**: 需要处理大量相似任务的场景

## 实现模式

### 方案1: 五要素设计法

**核心思想**: 确保任务定义包含五个关键要素

**五要素**:
1. **合理的任务粒度**: 5-10 分钟可完成
2. **清晰的任务定义**: 目标、关键结果、测试方法
3. **完整的上下文**: 通过 DAG 依赖关系提供
4. **明确的测试标准**: 可自动化验证
5. **无歧义的路径和参数**: 使用绝对路径或明确的相对路径

**优点**:
- 系统化、可复制
- 覆盖任务设计的关键维度
- 易于检查和优化

**缺点**:
- 需要在 plan 阶段投入更多时间
- 对任务设计者的要求较高

**示例代码** (task.md 模板):

```markdown
# 依赖关系
task1, task2

# 任务名称
创建模块文档

# 任务目标
在 `/Users/sunquan/ai_coding/CODING/rick/.rick/wiki/modules/` 目录下创建 7 个核心模块的详细文档。

# 关键结果
1. 创建 `modules/` 目录
2. 生成 7 个模块文档：cmd.md, workspace.md, parser.md, executor.md, prompt.md, git.md, config.md
3. 每个文档至少 100 行，总计至少 500 行
4. 每个文档包含：模块职责、核心类型、关键函数、类图（Mermaid）、使用示例（至少 4 个）

# 测试方法
1. 检查 `modules/` 目录是否存在于 `/Users/sunquan/ai_coding/CODING/rick/.rick/wiki/modules/`
2. 检查 7 个文档是否都存在
3. 统计总行数是否 >= 500 行
4. 检查每个文档是否包含 Mermaid 类图（`\`\`\`mermaid`）
5. 检查每个文档是否包含至少 4 个代码示例（`\`\`\`go`）
```

**关键点**:
- ✅ **使用绝对路径**: `/Users/sunquan/ai_coding/CODING/rick/.rick/wiki/modules/` 而非 `wiki/modules/`
- ✅ **明确数量要求**: "7 个模块文档"、"至少 100 行"、"至少 4 个示例"
- ✅ **可验证的测试方法**: 每个测试步骤都可以通过脚本自动化验证
- ✅ **完整的关键结果**: 列出所有需要完成的子目标

### 方案2: 渐进式上下文构建法

**核心思想**: 通过 DAG 依赖关系，让每个任务都能利用前序任务的输出作为上下文

**实现步骤**:
1. **基础任务**: 创建目录结构和索引（task1）
2. **并行基础任务**: 创建架构和流程文档（task2, task3）
3. **汇聚任务**: 基于前序任务生成模块文档（task4）
4. **串行专题任务**: 基于所有前序任务生成专题文档（task5-8）
5. **验证任务**: 验证所有前序任务的输出（task9）

**优点**:
- 每个任务都有完整的上下文
- 避免重复劳动
- 确保文档的一致性

**缺点**:
- DAG 设计需要仔细规划
- 任务之间的依赖关系可能复杂

**示例** (DAG 设计):

```
         task1 (基础)
         /    \
     task2    task3 (并行基础)
        \    /
        task4 (汇聚点)
          |
     task5, task6 (并行专题)
          |
     task7, task8 (并行专题)
          |
        task9 (验证)
```

**关键点**:
- ✅ **汇聚点设计**: task4 依赖 task2 和 task3，确保基础完成后再批量生产
- ✅ **渐进式构建**: 每个任务都能利用前序任务的输出
- ✅ **最终验证**: task9 依赖所有前序任务，进行全面验证

### 方案3: 测试驱动设计法

**核心思想**: 先设计测试脚本，再定义任务，确保任务定义和测试标准完全一致

**实现步骤**:
1. **设计测试脚本**: 定义所有验证点
2. **编写任务定义**: 基于测试脚本编写任务目标和关键结果
3. **执行任务**: AI Agent 根据任务定义执行
4. **运行测试**: 自动化验证任务完成质量

**优点**:
- 测试标准明确、无歧义
- 任务定义和测试完全对齐
- 易于自动化验证

**缺点**:
- 需要提前设计测试脚本
- 测试脚本可能需要多次迭代

**示例代码** (测试脚本):

```python
#!/usr/bin/env python3
import os
import sys

def test_task4():
    """测试 task4: 创建模块文档"""
    base_dir = "/Users/sunquan/ai_coding/CODING/rick/.rick/wiki"
    modules_dir = os.path.join(base_dir, "modules")

    # 1. 检查目录是否存在
    if not os.path.exists(modules_dir):
        print(f"❌ modules directory does not exist at {modules_dir}")
        return False

    # 2. 检查 7 个文档是否存在
    required_files = ["cmd.md", "workspace.md", "parser.md", "executor.md",
                      "prompt.md", "git.md", "config.md"]
    for file in required_files:
        file_path = os.path.join(modules_dir, file)
        if not os.path.exists(file_path):
            print(f"❌ {file} does not exist at {file_path}")
            return False

    # 3. 统计总行数
    total_lines = 0
    for file in required_files:
        file_path = os.path.join(modules_dir, file)
        with open(file_path, 'r') as f:
            total_lines += len(f.readlines())

    if total_lines < 500:
        print(f"❌ Total line count is {total_lines}, expected at least 500 lines")
        return False

    # 4. 检查 Mermaid 类图
    has_mermaid = False
    for file in required_files:
        file_path = os.path.join(modules_dir, file)
        with open(file_path, 'r') as f:
            content = f.read()
            if "```mermaid" in content:
                has_mermaid = True
                break

    if not has_mermaid:
        print(f"❌ No module document contains a Mermaid class diagram")
        return False

    # 5. 检查代码示例
    for file in required_files:
        file_path = os.path.join(modules_dir, file)
        with open(file_path, 'r') as f:
            content = f.read()
            code_blocks = content.count("```go")
            if code_blocks < 4:
                print(f"❌ {file} contains only {code_blocks} code examples, expected at least 4")
                return False

    print("✅ All tests passed!")
    return True

if __name__ == "__main__":
    success = test_task4()
    sys.exit(0 if success else 1)
```

**关键点**:
- ✅ **使用绝对路径**: 测试脚本中使用完整的绝对路径
- ✅ **详细的错误信息**: 每个验证点失败时都提供清晰的错误信息
- ✅ **可执行的测试**: 测试脚本可以直接运行，返回明确的成功/失败状态

## 最佳实践

### 1. 任务粒度控制

**原则**: 每个任务的预计执行时间为 5-10 分钟

**理由**:
- 太小（< 5 分钟）: 任务过于琐碎，管理开销大
- 太大（> 10 分钟）: 任务复杂度高，失败风险大

**示例**:
- ✅ **合理粒度**: "创建 7 个模块文档，每个文档 100 行"（预计 10 分钟）
- ❌ **粒度过小**: "创建 cmd.md 文档"（预计 1 分钟）
- ❌ **粒度过大**: "创建所有 Wiki 文档"（预计 2 小时）

### 2. 路径规范

**原则**: 始终使用绝对路径或明确的相对路径起点

**理由**:
- 避免路径歧义
- 确保文件创建在正确位置
- 便于测试脚本验证

**示例**:
- ✅ **绝对路径**: `/Users/sunquan/ai_coding/CODING/rick/.rick/wiki/modules/cmd.md`
- ✅ **明确的相对路径**: "在项目根目录下的 `.rick/wiki/modules/` 目录中创建 cmd.md"
- ❌ **歧义路径**: "创建 `wiki/modules/cmd.md`"（不清楚起点）

### 3. 测试标准明确化

**原则**: 每个测试标准都应该可以通过脚本自动化验证

**理由**:
- 避免主观判断
- 提高验证效率
- 确保一致性

**示例**:
- ✅ **可验证**: "文档至少 100 行"（可通过 `wc -l` 验证）
- ✅ **可验证**: "包含 Mermaid 类图"（可通过 `grep "```mermaid"` 验证）
- ❌ **不可验证**: "文档质量高"（主观判断）

### 4. 上下文完整性

**原则**: 通过 DAG 依赖关系确保每个任务都有完整的上下文

**理由**:
- 避免重复劳动
- 确保文档一致性
- 提高执行效率

**示例**:
- ✅ **完整上下文**: task4 依赖 task2 和 task3，可以参考架构和流程文档
- ❌ **缺失上下文**: task4 不依赖任何任务，需要从零开始理解系统架构

### 5. 关键结果可衡量

**原则**: 每个关键结果都应该有明确的衡量标准

**理由**:
- 便于验证任务完成度
- 避免歧义
- 提高任务定义质量

**示例**:
- ✅ **可衡量**: "生成 7 个模块文档，总计至少 500 行"
- ✅ **可衡量**: "每个文档包含至少 4 个代码示例"
- ❌ **不可衡量**: "生成高质量的模块文档"

## 常见陷阱

### ❌ 陷阱1: 路径歧义

**问题**: 使用相对路径 `wiki/modules/` 而不明确起点

**后果**: 文件创建在错误位置，导致测试失败和重试

**避免方法**:
- 始终使用绝对路径
- 或明确说明相对路径的起点（如"在项目根目录下的..."）

### ❌ 陷阱2: 任务粒度过大

**问题**: 将多个子任务合并为一个大任务

**后果**: 任务复杂度高，失败风险大，难以定位问题

**避免方法**:
- 将大任务分解为多个小任务
- 每个任务的预计执行时间控制在 5-10 分钟

### ❌ 陷阱3: 测试标准不明确

**问题**: 使用主观判断标准（如"质量高"、"内容详实"）

**后果**: 无法自动化验证，依赖人工审查

**避免方法**:
- 使用可量化的标准（行数、文件数、代码示例数）
- 使用可自动化验证的标准（文件存在、格式正确）

### ❌ 陷阱4: 缺失上下文

**问题**: 任务定义中缺少必要的背景信息和参考资料

**后果**: AI Agent 需要猜测或假设，导致输出不符合预期

**避免方法**:
- 通过 DAG 依赖关系提供前序任务的输出
- 在任务定义中明确参考资料
- 提供示例或模板

### ❌ 陷阱5: 过度并行化

**问题**: 将有隐式依赖的任务设计为并行执行

**后果**: 任务之间可能产生冲突或不一致

**避免方法**:
- 仔细分析任务之间的依赖关系
- 只有在确实无依赖时才并行化
- 使用汇聚点设计确保阶段性成果的完整性

## 测试建议

### 1. 单元测试

为每个任务编写独立的测试脚本，验证任务完成质量。

**示例**:
```python
def test_task1():
    """测试 task1: 创建目录结构和索引"""
    # 验证目录是否存在
    # 验证 README.md 是否存在
    # 验证 README.md 内容是否符合要求
    pass
```

### 2. 集成测试

验证多个任务的组合效果，确保任务之间的依赖关系正确。

**示例**:
```python
def test_tasks_1_to_4():
    """测试 task1-4 的集成效果"""
    # 验证目录结构
    # 验证架构文档
    # 验证模块文档
    # 验证文档之间的引用关系
    pass
```

### 3. 端到端测试

验证整个任务流程的完整性，确保最终产出符合预期。

**示例**:
```bash
#!/bin/bash
# 端到端测试脚本
./validate_wiki.sh
```

## 相关技能

- **DAG 任务分解方法**: 如何设计合理的 DAG 任务图
- **文档工程三阶段法**: 如何高效生成大量文档
- **测试驱动设计**: 如何通过测试脚本驱动任务设计

## 参考资料

- Job 1 执行总结: `.rick/jobs/job_1/learning/SUMMARY.md`
- 任务定义模板: `.rick/jobs/job_1/plan/tasks/*.md`
- 测试脚本示例: `.rick/jobs/job_1/doing/test_scripts/*.py`

---

## 成功案例

**Job 1: Wiki 文档创建**
- 任务数量: 9 个
- 零重试率: 100% (9/9)
- 执行时长: ~2 小时
- 文档产出: 16 个文档，10,657 行，33 个图表

**关键成功因素**:
1. 合理的任务粒度（5-10 分钟）
2. 清晰的任务定义（目标、关键结果、测试方法）
3. 完整的上下文（通过 DAG 依赖关系）
4. 明确的测试标准（可自动化验证）
5. 无歧义的路径和参数（使用绝对路径）

这次成功充分验证了零重试任务设计模式的有效性。
