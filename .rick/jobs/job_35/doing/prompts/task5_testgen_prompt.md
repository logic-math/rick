# Python 测试脚本生成任务

**YOU MUST declare at the start: "I will use skill:tdd and skill:testing-anti-patterns for test generation."**

## 核心 Skills（必须加载）

在开始任何工作之前，必须读取以下 skill 文件：

- skill:tdd（测试驱动开发）：`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/prompts/skill_tdd_zh.md`
- skill:testing-anti-patterns（测试反模式）：`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/prompts/skill_testing_anti_patterns_zh.md`

你需要根据任务的测试方法生成一个 Python 测试脚本。

## 任务信息

**Task ID**: task5
**Task Name**: 重构 builder 三件（templates + pibuilder + xxxxbuilder），注入路径而非内容
**Task Goal**: 按 spec（task2）落地 KR2.3：将 `internal/prompt` 重构为 builder 三件——templates（go `embed` 内嵌现有模板）+ pibuilder（pi 统一入口，组合 plan/doing/easy/human-loop 子 builder）+ xxxxbuilder（扩展位）。本 task 只做结构重构，**不改模板内容**（触发语言迁移在 task11，单文件内聚在 task10）。

关键方向：**builder 从「注入内容」改为「注入路径」**——rick 不再把 task.md/debug/OKR/SPEC 的内容解析进提示词，而是把 `job_dir`/`plan_dir`/`loops_dir`/`skills_dir`/`domain_dir` 路径注入模板，让 pi 在运行时自己 read。这使 `internal/parser`（读/校验内容）的消费者**大幅减少**，为 task8 删除 parser 铺路（parser 的 executor/prompt 消费点在 task8 与删 executor 同批解耦）。

**三层注入（方法/技能/实例分离，对齐「方法/实现隔离」+ 上下文熵减）**：每个 cmd 的 builder 产出**两份产物**——`method`（命令特定方法：plan 9 步 SOP / doing 角色+doing_loop / SENSE 5 阶段 → 走 system prompt，pi 的 `--append-system-prompt` 注入，免于被 compaction summarize）+ `instance`（job 上下文/路径 → 走 user prompt 文件）；rick 方法论 skills 走 pi skills 机制加载（不塞 system prompt）。

映射：现有 `internal/prompt/templates/`（顶层 10 个 .md = 9 个 loop + test_python.md，skills/ 19 个，go:embed）→ templates；`PromptBuilder`/`PromptManager` + `plan_prompt.go`/`doing_prompt.go`/`easy_prompt.go`/`human_loop_prompt.go`/`ctrl_prompt.go` 生成器 → 子 builder，由新建 pibuilder 统一入口组合；新增 `xxxxbuilder.go` 定义 `RuntimeBuilder` 接口（扩展位，当前无 pi 之外实现）。

参考：domain/go-patterns.md「embed.FS 目录嵌入」「包内函数共享」；skill `verify_go_changes_skill`、`global_ref_sync_skill`、`template_injection_skill`；RFC §4.2「builder 三件」。

### 问题记录


## 测试方法

正常路径：前置条件 = task2 完成；输入 = 无；操作 = `go build -o bin/rick ./cmd/rick && go test ./internal/builder/... ./internal/prompt/... -v`；预期 = build 成功，builder/prompt 测试全绿。
边界（模板零改动 + 注入路径）：前置条件 = 重构完成；输入 = 无；操作 = `git diff --stat internal/prompt/templates/`（预期无 diff）+ `./bin/rick plan --dry-run | grep -cE 'plan/task|doing/debug|/jobs/|/domain'`（预期 ≥1；`task_info_section`/`debug_context` 变量值已变真实路径片段，非 `plan_dir` 等变量名字面量）；预期 = 模板无 diff 且 task/debug 路径注入命中。
异常（builder 缺参数）：前置条件 = 重构完成；输入 = `PIBuilder.BuildPlan("")`（requirement 为空字符串，其余参数为空）；操作 = 调用检查 error；预期 = 返回 error 含 `requirement cannot be empty`。

## 测试脚本路径

请创建测试脚本到: `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task5.py`

## **CRITICAL**: JSON 输出格式要求

测试脚本**必须**输出**恰好一行**有效的 JSON 到 stdout：

### 成功情况
```json
{"pass": true, "errors": []}
```

### 失败情况
```json
{"pass": false, "errors": ["error message 1", "error message 2"]}
```

### JSON 格式规范

1. **`pass`**: 布尔值
   - `true`: 所有测试通过
   - `false`: 至少有一个测试失败

2. **`errors`**: 字符串数组
   - 如果 `pass=true`，必须是空数组 `[]`
   - 如果 `pass=false`，包含所有错误信息

3. **输出规则**:
   - 使用 `print(json.dumps(result))` 输出 JSON
   - **不要**向 stdout 输出其他任何内容
   - 调试信息请输出到 stderr

4. **退出码**:
   - `pass=true` → 退出码 0
   - `pass=false` → 退出码 1

## 测试脚本模板

**请严格遵循以下结构**：

```python
#!/usr/bin/env python3
import json
import sys
import os

def main():
    errors = []

    # Test step 1: 检查文件是否存在
    if not os.path.exists('expected_file.txt'):
        errors.append('expected_file.txt does not exist')

    # Test step 2: 验证文件内容
    try:
        with open('expected_file.txt', 'r') as f:
            content = f.read()
            if 'expected_content' not in content:
                errors.append('expected_file.txt missing expected content')
    except Exception as e:
        errors.append(f'Failed to read expected_file.txt: {str(e)}')

    # Test step 3: 检查其他条件
    # 添加更多测试步骤...

    # 构建结果 JSON
    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    # 输出 JSON (CRITICAL: 只有这一行输出到 stdout)
    print(json.dumps(result))

    # 使用合适的退出码
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
```

## 测试脚本编写要求

### 1. 实现所有测试步骤
- 根据上面的"测试方法"实现每个测试步骤
- 每个步骤都要有清晰的注释

### 2. 错误收集
- 使用 `errors.append()` 收集所有测试失败
- 不要在第一个错误时就退出
- 收集所有错误后一次性返回

### 3. 异常处理
- 使用 try-except 捕获可能的异常
- 将异常信息添加到 errors 数组
- 示例：`errors.append(f'操作失败: {str(e)}')`

### 4. 路径处理
- **必须使用绝对路径**检查文件
- 使用 `os.path.abspath()` 或 `os.getcwd()` 获取绝对路径
- 示例：`os.path.join(os.getcwd(), 'file.txt')`

### 5. 可执行性
- 添加 shebang: `#!/usr/bin/env python3`
- 脚本应该可以直接运行: `python3 /workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task5.py`

## ✅ DO（必须做）

- ✅ 使用 `print(json.dumps(result))` 输出 JSON
- ✅ 使用 `errors.append()` 收集所有失败
- ✅ `pass=true` 时退出码为 0，`pass=false` 时退出码为 1
- ✅ 使用绝对路径检查文件
- ✅ 使用 try-except 处理异常
- ✅ 实现测试方法中的所有步骤

## ❌ DON'T（禁止做）

- ❌ 向 stdout 输出调试信息（使用 stderr 代替）
- ❌ 输出多个 JSON 对象
- ❌ 返回无效的 JSON 格式
- ❌ 使用相对路径（容易出错）
- ❌ 在第一个错误时就退出（应该收集所有错误）
- ❌ 忘记实现某个测试步骤

## 示例：完整的测试脚本

```python
#!/usr/bin/env python3
import json
import sys
import os

def main():
    errors = []

    # 获取项目根目录（假设测试脚本在 tests/ 目录下）
    project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

    # Test 1: 检查配置文件
    config_file = os.path.join(project_root, 'config.json')
    if not os.path.exists(config_file):
        errors.append('config.json does not exist')
    else:
        try:
            with open(config_file, 'r') as f:
                import json as json_lib
                config = json_lib.load(f)
                if 'api_key' not in config:
                    errors.append('config.json missing api_key field')
        except Exception as e:
            errors.append(f'Failed to parse config.json: {str(e)}')

    # Test 2: 检查日志目录
    log_dir = os.path.join(project_root, 'logs')
    if not os.path.isdir(log_dir):
        errors.append('logs directory does not exist')

    # Test 3: 检查可执行文件
    binary = os.path.join(project_root, 'bin', 'app')
    if not os.path.exists(binary):
        errors.append('bin/app does not exist')
    elif not os.access(binary, os.X_OK):
        errors.append('bin/app is not executable')

    # 构建结果
    result = {
        'pass': len(errors) == 0,
        'errors': errors
    }

    # 输出 JSON
    print(json.dumps(result))

    # 退出
    sys.exit(0 if result['pass'] else 1)

if __name__ == '__main__':
    main()
```

## Cialdini 合规原则

### 权威（Authority）

**YOU MUST generate a failing test first (RED phase). No exceptions.**

测试脚本生成必须覆盖全部验收条件，不得遗漏任何测试步骤。

### 承诺（Commitment）

在开始生成测试脚本前，声明你将使用的 skills：

```
Declare: "I will use skill:tdd and skill:tc for test generation."
```

使用 `skill:tc` 时，必须检查四要素：前置条件 / 输入参数 / 操作序列 / 预期输出。

### 稀缺（Scarcity）

**Before writing any test, verify: you understand the acceptance criteria.**

每个测试步骤都必须对应明确的验收标准，未理解验收条件不得开始编写。

---

## 测试质量自检（强制）

生成测试脚本后，**必须立即运行以下命令**：

```bash
python3 /workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task5.py
```

**根据运行结果判断**：

- 输出 `"pass": false` → 符合预期，测试正确覆盖了待实现的功能
- 输出 `"pass": true` → 需要判断原因：
  - ✅ **可接受**：该功能已被前面的 task 顺带实现，测试通过是合理的
  - ❌ **需重写**：功能尚未实现但测试已通过，说明测试逻辑有缺陷（如断言过弱、检查对象错误），**必须重新编写测试脚本**

**你负责判断**，不依赖程序的硬性检查。判断依据：查看当前代码库，确认被测功能是否已存在。

---

## 重要提醒

1. **只生成测试脚本，不要执行任务本身**
2. **严格遵循 JSON 输出格式**，否则测试框架无法解析结果
3. **收集所有错误**，不要在第一个错误时就停止
4. **使用绝对路径**，避免路径相关的错误
5. **测试脚本应该是幂等的**，多次运行应该得到相同结果

现在请生成测试脚本。
