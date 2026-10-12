# Python 测试脚本生成任务

**YOU MUST declare at the start: "I will use skill:tdd and skill:testing-anti-patterns for test generation."**

## 核心 Skills（必须加载）

在开始任何工作之前，必须读取以下 skill 文件：

- skill:tdd（测试驱动开发）：`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/prompts/skill_tdd_zh.md`
- skill:testing-anti-patterns（测试反模式）：`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/prompts/skill_testing_anti_patterns_zh.md`

你需要根据任务的测试方法生成一个 Python 测试脚本。

## 任务信息

**Task ID**: task7
**Task Name**: 完成 handler 覆盖 human-loop/ctrl/dream/learning 并让 cli 全量变薄
**Task Goal**: 按 spec（task2）完成 KR2.1：handler 调度聚合层覆盖剩余命令 human-loop、ctrl、dream、learning，`internal/cmd` 全部命令变薄为「路由 + 参数解析 + 调 handler」。所有命令统一走「env.Ensure → builder.Build（产 method+instance 两份）→ 调用 runtime」编排；交互命令保留 `CallCLI`（行为不变，method 经 `--append-system-prompt` 传给 `CallCLI`），`runtime.Run` 结构化签名仅 task8 的 doing 使用。

## 逐命令改动明细（调研结论）

### human-loop（不依赖 executor/parser/git/actpath/logging，迁移无编译断裂）
- cli 保留 `NewHumanLoopCmd`（topic 校验：空则报「topic is required」）；迁出 RunE 内编排 → `handler.HumanLoop`：`GetDraftDir` → `NextLoopID` → 建目录 → `builder.BuildHumanLoop`（产 method+instance）→ `env.Ensure` → `runtime.CallCLI`（交互，注入 method）→ 持久化 sessionID
- flags：无自有 flag；全局 `--dry-run`/`--job`/`-v`
- **task5 包迁移影响**：`human_loop_prompt.go` 依赖 prompt 包的 `WriteSkillFile`/`LoadCoreSkills`（读 skillsFS embed），prompt→builder 迁移后改经 builder 导出函数（`ReadEmbeddedSkill`）；task9 注册 think/research/exporter agent 后，`{{think_skill_path}}` 等 skill 路径变量改由 pi skills 机制承载

### ctrl（不直接 import executor/parser/git/logging/agent 接口/actpath，仅 piagent→runtime）
- cli 保留 `NewCtrlCmd`（`--job` 必传校验 + 调 handler）；迁出 `runCtrl`/`runCtrlDryRun` → `handler.Ctrl`/`handler.CtrlDryRun`
- flags：无自有 flag；全局 `--dry-run`/`--job`（代码强制必传）/`-v`
- **ctrl.md 模板 stale（重要）**：当前描述 claude code 的 NDJSON 格式（`type=system/assistant/user/result`），但 runtime 已用 pi JSONL（`session/agent_settled/message_end/tool_execution_start/tool_execution_end`，camelCase）——本 task 改写 ctrl.md 为 pi JSONL 语义；`act-path.md` 引用改为 runtime trace（task8 一并）
- builder：`GenerateCtrlPromptFile` 拆为 `BuildCtrl` → (method=ctrl.md 角色定义, instance=job_id/doing_dir/plan_dir/tasks_json 路径)，**注入路径而非内容**（去掉 tasks_json_content 快照，pi 自行 read）

### dream（唯一 executor 依赖 = LoadTasksJSON，line 169）
- cli 保留 `NewDreamCmd`（`-p/--background`、`--job_num int`(默认5)）；迁出 `dreamWorkflow`→`handler.Dream`、`runDreamDryRun`→`handler.DreamDryRun`、`selectPendingJobs`/`getDreamProcessedJobs`/`discoverCompletedJobs`/`jobNumber`（4 个确定性扫描过滤函数）
- **扫描过滤留在 Go（确定性输入过滤，决定哪些 job 要 dream）**：`discoverCompletedJobs` 依赖的 `executor.LoadTasksJSON` + `TasksJSON.GetAllTasks()` + `TaskState.Status` 迁 `workspace`（极薄读取器，task8 显式落地）
- `dream.md` Step 2 读 `act-path.md`、`dream_prompt.go` 的 `loadActPaths` 扫 `act-path.md` → task8 改扫 `trace.md`；Step 7 的 subagent_1~4 触发词 → task11 改 workflowScript+runs.run；Step 8 `tools dream_check` 引用保留（dream_check 存活）

### learning（依赖 executor + parser + actpath，task8 一并闭环）
- cli 保留 `NewLearningCmd`（`--job` flag）；迁出 `executeLearningWorkflow`→`handler.Learning`、`runLearningDryRun`→`handler.LearningDryRun`、`collectExecutionData`、`callAgentForAnalysis`、`buildLearningPrompt`（当前内联在 cmd 内，无独立 prompt/*.go → 迁 builder 的 `BuildLearning`）
- 依赖闭环（task8）：`executor.LoadTasksJSON`/`TasksJSON`/`TaskState` + `executor.LoadDebugContext` + `parser.ExtractBugFrontmatter` → 迁 workspace/prompt 极薄实现；`ActPathFiles` glob `act-path.md` → 改 glob `trace.md`
- 模板 `learning.md` 完成要求 `rick tools learning_check {{job_id}}`（learning_check 存活，task8 明确）

参考：domain/commands.md「human-loop」「ctrl」「dream」；skill `command_registration_verification_skill`、`verify_go_changes_skill`、`global_ref_sync_skill`。

### 问题记录


## 测试方法

正常路径：前置条件 = task6 完成；输入 = 无；操作 = `go build -o bin/rick ./cmd/rick && go test ./internal/cmd/... ./internal/handler/... -timeout 60s -v`；预期 = build 成功，测试全绿。
边界（human-loop dry-run + dream 扫描过滤 + ctrl --job 缺失）：前置条件 = build 成功 + 存在「全 success」与「未完成」的 job；输入 = `human-loop --dry-run '测试主题'`、`dream --dry-run`、`ctrl`（无 --job）；操作 = 依次运行；预期 = human-loop 输出含 `sense_loop`；dream 只列「完成且未 dream」的 job（排除未完成/已 dream，按 job 号升序截断）；ctrl 报 `--job flag is required` exit 非 0。
异常（learning 缺数据 + ctrl doing 目录不存在）：前置条件 = build 成功；输入 = `learning job_N`（doing/tasks.json 不存在）、`ctrl --dry-run --job job_N`（doing 目录不存在）；操作 = 运行；预期 = learning 报 `tasks.json not found` exit 非 0 不 panic；ctrl 报 `doing directory not found` exit 非 0。

## 测试脚本路径

请创建测试脚本到: `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task7.py`

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
- 脚本应该可以直接运行: `python3 /workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task7.py`

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
python3 /workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task7.py
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
