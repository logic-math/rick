# Rick Doing 阶段修复总结

## 问题描述

Rick 的 `doing` 命令执行时，所有任务立即标记为成功，但**没有真正调用 Claude Code CLI 来执行任务**。

### 原始实现的问题

**`internal/executor/runner.go`** 的 `GenerateTestScript` 函数：
```go
// 只生成一个假的测试脚本
scriptContent.WriteString("echo \"Status: PASS\"\n")  // ❌ 直接返回 PASS
```

导致：
- ✅ 任务立即标记为成功
- ❌ **没有调用 Claude Code CLI**
- ❌ **没有真正执行任务**
- ❌ **没有生成代码**

## 修复方案

### 1. 修改 `RunTask` 函数流程

**之前（错误）**：
```
生成测试脚本 → 运行测试 → 标记成功
```

**现在（正确）**：
```
生成 Doing 提示词 → 调用 Claude CLI → 生成测试脚本 → 运行测试 → 根据结果决定
```

### 2. 新增函数

#### `GenerateDoingPrompt(task *parser.Task) (string, error)`
- 使用 `prompt.GenerateDoingPrompt` 生成提示词
- 加载 OKR.md 和 SPEC.md 作为上下文
- 包含任务的目标、关键结果、测试方法

#### `CallClaudeCodeCLI(promptContent string) (string, error)`
- **非交互式模式**：使用管道 + `--dangerously-skip-permissions`
- 参考 Morty 的实现：`cat prompt | claude --dangerously-skip-permissions`
- 支持超时控制（默认 10 分钟）
- 捕获 stdout 和 stderr

#### `convertTestStepToCommand(step string) string`
- 将测试步骤描述转换为可执行命令
- 支持常见模式：文件验证、目录检查、运行测试等

### 3. 改进 `GenerateTestScript`

**之前**：
```bash
echo "Status: PASS"  # 假的
```

**现在**：
```bash
# 执行真实的测试步骤
TEST_PASSED=true

# Step 1: 验证文件存在
test -f file.txt || TEST_PASSED=false

# Step 2: 验证内容
grep "expected" file.txt || TEST_PASSED=false

# 返回测试结果
if [ "$TEST_PASSED" = true ]; then
  echo "Status: PASS"
  exit 0
else
  echo "Status: FAIL"
  exit 1
fi
```

## 完整执行流程

```
┌─────────────────────────────────────────────────────────────┐
│ Rick Doing 阶段 - 完整流程                                    │
└─────────────────────────────────────────────────────────────┘

1. 加载任务
   ├─ 从 .rick/jobs/job_X/plan/ 加载 task.md 文件
   ├─ 解析依赖关系（过滤"无"）
   └─ 构建 DAG 并拓扑排序

2. 对每个任务（按顺序）：

   2.1 生成 Doing 提示词
       ├─ 加载 OKR.md 和 SPEC.md
       ├─ 加载 debug.md（如果存在）
       ├─ 使用 prompt.GenerateDoingPrompt()
       └─ 包含：任务目标、关键结果、测试方法、上下文

   2.2 调用 Claude Code CLI ✨ 新增
       ├─ 命令: echo prompt | claude --dangerously-skip-permissions
       ├─ Claude 分析任务并生成代码
       ├─ Claude 执行必要的文件操作
       └─ 捕获输出和错误

   2.3 生成测试脚本 ✨ 改进
       ├─ 基于 task.TestMethod 生成可执行测试
       ├─ 转换测试步骤为 bash 命令
       └─ 创建 test_<task_id>.sh

   2.4 运行测试脚本
       ├─ 执行 bash test_<task_id>.sh
       ├─ 捕获输出
       └─ 解析测试结果（PASS/FAIL）

   2.5 根据结果决定
       ├─ 测试通过：
       │   ├─ git add 相关文件
       │   ├─ git commit -m "task: <task_id> completed"
       │   └─ 标记任务为 done
       │
       └─ 测试失败：
           ├─ 记录失败信息到 debug.md
           ├─ 重试（最多 MaxRetries 次，默认 5）
           └─ 超过限制：退出，等待人工干预

3. 所有任务完成
   └─ 输出执行摘要
```

## 代码改动

| 文件 | 改动 | 状态 |
|------|------|------|
| `internal/executor/runner.go` | 添加 `GenerateDoingPrompt()` | ✅ |
| `internal/executor/runner.go` | 添加 `CallClaudeCodeCLI()` | ✅ |
| `internal/executor/runner.go` | 修改 `RunTask()` 流程 | ✅ |
| `internal/executor/runner.go` | 改进 `GenerateTestScript()` | ✅ |
| `internal/executor/runner.go` | 添加 `convertTestStepToCommand()` | ✅ |
| `internal/executor/runner.go` | 添加 `extractFilePath()` | ✅ |

## 关键技术点

### 1. 非交互式调用 Claude CLI

```go
cmd := exec.Command("claude", "--dangerously-skip-permissions")
stdin, _ := cmd.StdinPipe()
cmd.Start()
stdin.Write([]byte(promptContent))
stdin.Close()
cmd.Wait()
```

**为什么使用 `--dangerously-skip-permissions`？**
- 允许 Claude 自动执行所有操作（文件读写、命令执行）
- 避免交互式权限提示
- 适合自动化流程
- ⚠️ 仅在受信任的环境使用

### 2. 提示词生成

```go
contextMgr := prompt.NewContextManager("doing")
contextMgr.LoadOKRFromFile(okriPath)
contextMgr.LoadSPECFromFile(specPath)

promptMgr := prompt.NewPromptManager("")  // 使用内嵌模板
doingPrompt, _ := prompt.GenerateDoingPrompt(task, 0, contextMgr, promptMgr)
```

### 3. 测试脚本生成

从描述性文本：
```
1. 验证文件 test_output.txt 存在
2. 验证文件内容包含 "Hello from Rick"
```

转换为可执行命令：
```bash
test -f test_output.txt || TEST_PASSED=false
grep "Hello from Rick" test_output.txt || TEST_PASSED=false
```

## 测试验证

### 单元测试
```bash
go test ./internal/executor/... -v
```

### 集成测试
```bash
# 1. 创建测试任务
rick plan "创建测试文件"

# 2. 执行任务（现在会真正调用 Claude）
rick doing job_X

# 3. 验证结果
# - Claude 输出日志
# - 测试脚本执行结果
# - Git 提交记录
```

## 与 Morty 的对比

| 特性 | Morty | Rick (修复后) |
|------|-------|---------------|
| 调用 Claude CLI | ✅ `cat prompt \| ai_cli --dangerously-skip-permissions` | ✅ `echo prompt \| claude --dangerously-skip-permissions` |
| 生成提示词 | ✅ 使用模板文件 | ✅ 使用内嵌模板 + PromptManager |
| 测试脚本 | ✅ 基于测试方法生成 | ✅ 基于测试方法生成 + 智能转换 |
| 重试机制 | ✅ 最多 50 次 | ✅ 最多 5 次（可配置） |
| Debug 记录 | ✅ debug.md | ✅ debug.md |
| Git 提交 | ✅ 自动提交 | ✅ 自动提交 |

## 下一步优化

1. **改进测试步骤转换**
   - 当前的 `convertTestStepToCommand` 是简单的启发式
   - 可以让 Claude 在执行任务时同时生成测试脚本

2. **增强错误处理**
   - 更详细的错误分类
   - 更智能的重试策略

3. **并行执行**
   - 当前是串行执行
   - 可以支持 DAG 并行执行无依赖的任务

4. **进度显示**
   - 实时显示 Claude 的执行进度
   - 流式输出

## 总结

✅ **修复完成**：Rick doing 阶段现在会真正调用 Claude Code CLI 来执行任务

✅ **完整流程**：提示词生成 → Claude 执行 → 测试验证 → 提交/重试

✅ **符合设计**：与 MEMORY.md 中的设计文档一致

✅ **参考 Morty**：借鉴了 Morty 的最佳实践

---

**修复日期**: 2026-03-14
**修复人**: Claude Opus 4.6
