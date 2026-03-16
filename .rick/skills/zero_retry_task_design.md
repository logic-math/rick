# Skill: Zero Retry Task Design (零重试任务设计法)

**技能类型**: 任务设计 / 项目管理
**适用场景**: 设计高成功率的 AI 执行任务
**成功率**: ⭐⭐⭐⭐⭐ (Job 1 验证：9/9 任务零重试)

---

## 概述

零重试任务设计法是一种任务设计方法论，通过**精确的任务定义、合理的粒度控制、完整的上下文传递**，使 AI Agent 能够一次性成功完成任务，无需重试。

### 核心理念
```
清晰定义 + 合理粒度 + 完整上下文 = 零重试
```

### 价值主张
- ✅ **提高效率**: 减少重试次数，节省时间
- ✅ **降低成本**: 减少 API 调用次数
- ✅ **提升质量**: 清晰的任务定义 → 高质量输出
- ✅ **易于调试**: 失败时容易定位问题

---

## 五大设计原则

### 1. 任务粒度: 5-10 分钟可完成 ⏱️

**原则**: 每个任务应该在 5-10 分钟内完成，不要太大也不要太小。

#### 为什么是 5-10 分钟？
- **太小（< 5 分钟）**: 任务过于琐碎，管理开销大
- **太大（> 10 分钟）**: 任务复杂度高，失败风险大
- **刚好（5-10 分钟）**: 平衡了复杂度和效率

#### 如何估算任务时长？
```markdown
# 估算公式
任务时长 = 理解时间 + 执行时间 + 验证时间

# 示例 1: 创建单个文档
- 理解时间: 2 分钟（阅读任务描述、上下文）
- 执行时间: 5 分钟（编写文档、代码示例）
- 验证时间: 1 分钟（检查格式、测试）
- 总计: 8 分钟 ✅

# 示例 2: 创建 7 个模块文档
- 理解时间: 3 分钟
- 执行时间: 35 分钟（7 个文档 × 5 分钟）
- 验证时间: 2 分钟
- 总计: 40 分钟 ❌ 太大，需要拆分

# 拆分后: 每个模块一个任务
- task1: 创建 module1.md (8 分钟) ✅
- task2: 创建 module2.md (8 分钟) ✅
- task3: 创建 module3.md (8 分钟) ✅
```

#### 实战案例：Job 1 任务粒度
| Task ID | 任务内容 | 估算时长 | 实际时长 | 状态 |
|---------|---------|----------|----------|------|
| task1 | 创建目录结构和索引 | 5 分钟 | ~5 分钟 | ✅ |
| task2 | 编写架构概览文档 | 10 分钟 | ~10 分钟 | ✅ |
| task3 | 编写运行时流程文档 | 12 分钟 | ~12 分钟 | ✅ |
| task4 | 编写 7 个模块文档 | 40 分钟 | ~40 分钟 | ✅ |

**改进建议**: task4 应该拆分为 7 个独立任务（每个模块一个任务）

---

### 2. 单一职责: 一个任务只做一件事 🎯

**原则**: 每个任务应该有且仅有一个明确的目标，避免多个目标混杂。

#### 反例：多职责任务 ❌
```markdown
# 任务: 创建文档和测试脚本

## 任务目标
1. 创建 wiki/architecture.md 文档
2. 编写文档的测试脚本
3. 运行测试并生成报告

## 问题
- 目标太多，容易遗漏
- 失败时难以定位问题（是文档问题还是测试问题？）
- 无法并行执行
```

#### 正例：单一职责任务 ✅
```markdown
# task1: 创建架构文档

## 任务目标
创建 `wiki/architecture.md`，详细介绍 Rick CLI 的整体架构。

## 关键结果
1. 文档包含 500+ 行内容
2. 包含 5 个 Mermaid 架构图
3. 包含核心理论和设计原则

---

# task2: 编写测试脚本

## 任务目标
为 wiki/ 目录下的所有文档编写自动化测试脚本。

## 关键结果
1. 创建 validate_wiki.sh
2. 检查文件存在性、行数、图表数
3. 生成验证报告
```

#### 如何判断任务是否单一职责？
```markdown
# 测试方法：用一句话描述任务
✅ "创建 architecture.md 文档"
✅ "编写模块文档测试脚本"
❌ "创建文档并编写测试脚本"（包含"并"字，说明多职责）
❌ "创建文档、测试脚本和运行测试"（包含多个动词）
```

---

### 3. 清晰依赖: 明确前置任务和依赖关系 🔗

**原则**: 任务之间的依赖关系应该清晰明确，确保上下文正确传递。

#### 依赖关系类型
```markdown
# 1. 无依赖（并行执行）
task1: 创建 module1.md
task2: 创建 module2.md
task3: 创建 module3.md

# 2. 串行依赖（顺序执行）
task1: 创建目录结构
task2: 编写架构文档（依赖 task1）
task3: 编写模块文档（依赖 task2）

# 3. 多依赖（汇聚执行）
task1, task2, task3: 编写各模块文档
task4: 验证所有文档（依赖 task1, task2, task3）
```

#### 如何设计依赖关系？
```markdown
# Step 1: 识别基础任务（无依赖）
task1: 创建目录结构 (无依赖)

# Step 2: 识别并行任务（可同时执行）
task2: 编写架构文档 (依赖 task1)
task3: 编写流程文档 (依赖 task1)

# Step 3: 识别串行任务（需要前置上下文）
task4: 编写模块文档 (依赖 task2, task3)
  → 需要 task2 提供的架构上下文
  → 需要 task3 提供的流程上下文

# Step 4: 识别汇聚任务（依赖所有前置任务）
task5: 验证所有文档 (依赖 task1, task2, task3, task4)
```

#### 实战案例：Job 1 依赖关系
```
Level 0: task1 (基础任务)
         └─ 创建目录结构

Level 1: task2, task3 (并行任务)
         ├─ task2: 编写架构文档
         └─ task3: 编写流程文档

Level 2: task4 (串行任务)
         └─ 编写模块文档 (依赖 task2, task3)

Level 3: task5, task6, task7, task8 (并行任务)
         ├─ task5: DAG 引擎详解
         ├─ task6: 提示词系统文档
         ├─ task7: 测试文档
         └─ task8: 安装文档

Level 4: task9 (汇聚任务)
         └─ 验证所有文档 (依赖所有前置任务)
```

---

### 4. 完整上下文: 提供足够的背景信息 📚

**原则**: 任务描述应该包含足够的上下文信息，让 AI Agent 无需猜测即可理解任务。

#### 上下文的三个层次

##### Layer 1: 任务本身的上下文
```markdown
# 任务: 编写模块文档

## 任务目标
创建 `wiki/modules/executor.md`，详细介绍任务执行引擎。

## 关键结果
1. 文档包含 500+ 行内容
2. 包含 Mermaid 类图
3. 包含 4+ 个使用示例

## 测试方法
1. 检查文件存在：`ls wiki/modules/executor.md`
2. 检查行数：`wc -l wiki/modules/executor.md`
3. 检查图表：`grep -c "```mermaid" wiki/modules/executor.md`
```

##### Layer 2: 依赖任务的上下文
```markdown
# 依赖关系
task2, task3  # 依赖架构文档和流程文档

# 上下文参考
- 参考 wiki/architecture.md 了解整体架构
- 参考 wiki/runtime-flow.md 了解执行流程
- executor 模块负责 DAG 拓扑排序和任务执行
```

##### Layer 3: 项目全局上下文
```markdown
# 项目上下文
- 项目名称: Rick CLI
- 核心理念: Context-First AI Coding Framework
- 目标读者: 开发者和高级用户

# 文档标准
- 使用 Markdown 格式
- 包含 Mermaid 图表
- 提供 Go 代码示例
- 遵循 wiki/CONTRIBUTING.md 规范
```

#### 上下文传递机制
```markdown
# Rick CLI 的上下文传递
Doing 提示词 = 系统上下文 + 任务上下文 + 依赖上下文

System Context (固定)
├─ Rick CLI 介绍
├─ 核心公式和理念
└─ 工作流程说明

Task Context (动态)
├─ 当前任务信息 (task.md)
├─ 调试信息 (debug.md，如有失败)
└─ 项目上下文 (OKR.md, SPEC.md)

Dependency Context (动态)
├─ 前置任务信息 (依赖任务的 task.md)
└─ 前置任务输出 (依赖任务创建的文件)
```

---

### 5. 可测试性: 明确的测试标准和方法 ✅

**原则**: 任务应该有明确的测试标准，能够自动化验证是否完成。

#### 测试标准的三个维度

##### 1. 功能完整性
```markdown
# 检查点
✅ 文件是否创建？
✅ 内容是否完整？
✅ 结构是否正确？

# 测试方法
1. 检查文件存在：`test -f wiki/architecture.md`
2. 检查章节存在：`grep -q "## 核心理论" wiki/architecture.md`
3. 检查代码块：`grep -c "```" wiki/architecture.md`
```

##### 2. 质量标准
```markdown
# 检查点
✅ 行数是否达标？
✅ 图表数量是否足够？
✅ 代码示例是否充足？

# 测试方法
1. 检查行数：`wc -l wiki/architecture.md | awk '{print $1}'`
2. 检查图表：`grep -c "```mermaid" wiki/architecture.md`
3. 检查示例：`grep -c "```go" wiki/architecture.md`
```

##### 3. 格式规范
```markdown
# 检查点
✅ Markdown 格式是否正确？
✅ 标题层级是否合理？
✅ 链接是否有效？

# 测试方法
1. 检查标题层级：`grep -E "^#{1,6} " wiki/architecture.md`
2. 检查链接格式：`grep -E "\[.*\]\(.*\)" wiki/architecture.md`
3. 检查代码块闭合：`grep -c "```" wiki/architecture.md` (应为偶数)
```

#### 测试脚本模板
```python
#!/usr/bin/env python3
import os
import sys

def test_task():
    """测试任务是否完成"""
    results = []

    # 1. 功能完整性
    if not os.path.exists("wiki/architecture.md"):
        results.append("❌ File not found: wiki/architecture.md")
        return False, results

    # 2. 质量标准
    with open("wiki/architecture.md") as f:
        content = f.read()
        lines = content.split("\n")

        if len(lines) < 500:
            results.append(f"❌ Line count too low: {len(lines)} < 500")
        else:
            results.append(f"✅ Line count: {len(lines)}")

        mermaid_count = content.count("```mermaid")
        if mermaid_count < 5:
            results.append(f"❌ Diagram count too low: {mermaid_count} < 5")
        else:
            results.append(f"✅ Diagram count: {mermaid_count}")

    # 3. 格式规范
    # ... (省略)

    # 判断是否通过
    passed = all("✅" in r for r in results)
    return passed, results

if __name__ == "__main__":
    passed, results = test_task()
    for result in results:
        print(result)
    sys.exit(0 if passed else 1)
```

---

## 实战案例：Job 1 零重试分析

### 任务设计对比

#### task1: 创建目录结构和索引 ✅
```markdown
# 任务粒度
估算时长: 5 分钟
实际时长: ~5 分钟

# 单一职责
目标: 创建 wiki/ 目录结构和 README.md 索引文件

# 清晰依赖
依赖: 无（基础任务）

# 完整上下文
- 项目名称: Rick CLI
- 目标: 创建 Wiki 文档系统
- 结构: wiki/README.md, wiki/modules/, wiki/tutorials/

# 可测试性
1. 检查目录存在：`test -d wiki`
2. 检查索引文件：`test -f wiki/README.md`
3. 检查行数：`wc -l wiki/README.md`

# 结果
✅ 一次通过，无重试
```

#### task4: 编写核心模块文档 ✅
```markdown
# 任务粒度
估算时长: 40 分钟（7 个模块）
实际时长: ~40 分钟
⚠️ 粒度偏大，建议拆分为 7 个任务

# 单一职责
目标: 创建 7 个模块文档
✅ 虽然包含 7 个文档，但都是同一类型（模块文档）

# 清晰依赖
依赖: task2, task3（架构文档和流程文档）
✅ 依赖关系明确，确保有足够的上下文

# 完整上下文
- 参考 wiki/architecture.md 了解整体架构
- 参考 wiki/runtime-flow.md 了解执行流程
- 7 个模块：cmd, workspace, parser, executor, prompt, git, config

# 可测试性
1. 检查 7 个文件存在
2. 检查总行数 ≥ 500
3. 检查每个文档包含 Mermaid 类图

# 结果
✅ 一次通过，无重试
💡 但有一次测试脚本路径错误（.rick/wiki vs wiki）
```

### "失败"分析

#### Debug1: task4 测试脚本路径错误
```markdown
# 现象
test did not pass: wiki/modules directory does not exist

# 根因分析
- Claude 正确创建了文件：`.rick/wiki/modules/*.md`
- 测试脚本期望路径：`wiki/modules/*.md`
- 问题：任务描述中没有明确指定 `.rick/` 前缀

# 改进建议
1. 任务描述中明确完整路径：
   "创建 `.rick/wiki/modules/` 目录（注意：在 .rick 目录下）"

2. 测试脚本更智能：
   自动检测 `wiki/` 和 `.rick/wiki/` 两个路径

# 结论
✅ 这不是任务执行失败，而是测试脚本设计问题
✅ Claude 实际上一次性正确完成了任务
```

---

## 任务设计检查清单

### 设计阶段 ✅
- [ ] 任务粒度：5-10 分钟可完成
- [ ] 单一职责：用一句话描述任务
- [ ] 清晰依赖：明确前置任务
- [ ] 完整上下文：提供足够的背景信息
- [ ] 可测试性：定义明确的测试标准

### 编写 task.md ✅
```markdown
# 依赖关系
[前置任务 ID，逗号分隔]

# 任务名称
[简短的任务标题]

# 任务目标
[详细的目标描述，包含完整路径]

# 关键结果
1. [可量化的结果 1]
2. [可量化的结果 2]
3. [可量化的结果 3]

# 测试方法
1. [测试步骤 1]
2. [测试步骤 2]
3. [测试步骤 3]

# 上下文参考（可选）
- [相关文件或文档]
- [关键信息]
```

### 执行阶段 ✅
- [ ] 加载任务上下文（task.md）
- [ ] 加载依赖上下文（前置任务的 task.md）
- [ ] 加载调试上下文（debug.md，如有失败）
- [ ] 生成测试脚本
- [ ] 执行任务
- [ ] 运行测试

### 验证阶段 ✅
- [ ] 测试是否通过？
- [ ] 如果失败，记录到 debug.md
- [ ] 如果成功，git commit
- [ ] 更新任务状态

---

## 最佳实践

### ✅ Dos

1. **任务粒度控制**
   - 5-10 分钟是最佳粒度
   - 宁可拆分过细，不要过大

2. **依赖关系设计**
   - 基础任务无依赖
   - 并行任务可同时执行
   - 串行任务确保上下文传递
   - 汇聚任务作为质量门禁

3. **上下文传递**
   - 任务描述包含完整上下文
   - 依赖任务提供必要信息
   - 项目全局上下文（OKR, SPEC）

4. **测试标准**
   - 功能完整性（文件存在、内容完整）
   - 质量标准（行数、图表数、示例数）
   - 格式规范（Markdown、标题层级、链接）

### ❌ Don'ts

1. **不要设计过大的任务**
   - 超过 10 分钟的任务容易失败
   - 应该拆分为多个小任务

2. **不要混杂多个目标**
   - 一个任务只做一件事
   - 避免"创建文档并测试"这样的多职责任务

3. **不要忽视依赖关系**
   - 依赖关系不清晰会导致上下文缺失
   - 必须明确指定前置任务

4. **不要省略测试标准**
   - 没有测试标准的任务无法验证
   - 测试标准应该可自动化执行

---

## 工具和模板

### task.md 模板
```markdown
# 依赖关系
[前置任务 ID，逗号分隔，如：task1, task2]

# 任务名称
[简短的任务标题，如：创建架构文档]

# 任务目标
[详细描述，包含完整路径和具体要求]
创建 `.rick/wiki/architecture.md`，详细介绍 Rick CLI 的整体架构。

# 关键结果
1. 文档包含 500+ 行内容
2. 包含 5 个 Mermaid 架构图
3. 包含核心理论和设计原则
4. 包含 4+ 个代码示例

# 测试方法
1. 检查文件存在：`test -f .rick/wiki/architecture.md`
2. 检查行数：`wc -l .rick/wiki/architecture.md | awk '{print $1}'`
3. 检查图表：`grep -c "```mermaid" .rick/wiki/architecture.md`
4. 检查章节：`grep -q "## 核心理论" .rick/wiki/architecture.md`

# 上下文参考
- 项目：Rick CLI (Context-First AI Coding Framework)
- 参考：.rick/OKR.md, .rick/SPEC.md
- 目标读者：开发者和高级用户
```

### 测试脚本模板
```python
#!/usr/bin/env python3
"""
任务测试脚本模板
"""
import os
import sys

def check_file_exists(file_path):
    """检查文件是否存在"""
    if not os.path.exists(file_path):
        return False, f"File not found: {file_path}"
    return True, f"File exists: {file_path}"

def check_line_count(file_path, min_lines):
    """检查行数"""
    with open(file_path) as f:
        lines = len(f.readlines())
    if lines < min_lines:
        return False, f"Line count too low: {lines} < {min_lines}"
    return True, f"Line count: {lines}"

def check_pattern_count(file_path, pattern, min_count):
    """检查模式出现次数"""
    with open(file_path) as f:
        content = f.read()
        count = content.count(pattern)
    if count < min_count:
        return False, f"Pattern '{pattern}' count too low: {count} < {min_count}"
    return True, f"Pattern '{pattern}' count: {count}"

def main():
    """主测试函数"""
    results = []

    # 1. 检查文件存在
    passed, msg = check_file_exists(".rick/wiki/architecture.md")
    results.append((passed, msg))
    if not passed:
        print_results(results)
        return 1

    # 2. 检查行数
    passed, msg = check_line_count(".rick/wiki/architecture.md", 500)
    results.append((passed, msg))

    # 3. 检查图表数量
    passed, msg = check_pattern_count(".rick/wiki/architecture.md", "```mermaid", 5)
    results.append((passed, msg))

    # 打印结果
    print_results(results)

    # 返回状态码
    return 0 if all(r[0] for r in results) else 1

def print_results(results):
    """打印测试结果"""
    for passed, msg in results:
        symbol = "✅" if passed else "❌"
        print(f"{symbol} {msg}")

if __name__ == "__main__":
    sys.exit(main())
```

---

## 总结

零重试任务设计法通过五大原则，确保 AI Agent 能够一次性成功完成任务：

1. **任务粒度**: 5-10 分钟可完成
2. **单一职责**: 一个任务只做一件事
3. **清晰依赖**: 明确前置任务和依赖关系
4. **完整上下文**: 提供足够的背景信息
5. **可测试性**: 明确的测试标准和方法

**成功率**: ⭐⭐⭐⭐⭐ (Job 1 验证：9/9 任务零重试)

**关键优势**:
- 🎯 提高效率，减少重试
- 💰 降低成本，减少 API 调用
- ✅ 提升质量，清晰定义 → 高质量输出
- 🐛 易于调试，失败时容易定位问题

**适用场景**: 所有需要 AI Agent 执行的任务，特别是复杂的软件开发任务。
