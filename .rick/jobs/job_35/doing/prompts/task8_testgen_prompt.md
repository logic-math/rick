# Python 测试脚本生成任务

**YOU MUST declare at the start: "I will use skill:tdd and skill:testing-anti-patterns for test generation."**

## 核心 Skills（必须加载）

在开始任何工作之前，必须读取以下 skill 文件：

- skill:tdd（测试驱动开发）：`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/prompts/skill_tdd_zh.md`
- skill:testing-anti-patterns（测试反模式）：`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/prompts/skill_testing_anti_patterns_zh.md`

你需要根据任务的测试方法生成一个 Python 测试脚本。

## 任务信息

**Task ID**: task8
**Task Name**: 做薄 cutover：下沉 doing 调度与门禁到 pi，并删除全部冗余 Go 包
**Task Goal**: 按 spec（task2）落地 KR2.5（rick 做薄，同时覆盖 KR3.2 的 dag/门禁下沉部分）：这是「做薄」的原子切换点，一次性完成调度下沉与冗余包删除（此前 task4/6 刻意保留 executor/agent/actpath/parser/git/logging，避免中间态编译断裂）。

1. **dag 调度 → pi workflowScript（顺序确定性）+ rick 生成期过滤（跳过已完成）**：doing 提示词产出 `workflowScript` + `runs.run` 编排（按 task 依赖拓扑，被依赖 task 先执行，`await` 强制顺序；编排权在 parent、单写者）。**「跳过已完成」= rick 生成期过滤**：rick 在拼提示词时读 `doing/tasks.json`（确定性 Go），过滤掉 `status=success` 的 task，对剩余 pending task 算剩余拓扑，只把 pending task 写进 workflowScript——重试时已完成 task 天然不在编排里（workflowScript 沙箱无文件系统、读不了 tasks.json，这一步只能在生成期做）。**触发假设（基线前提，非风险）**：workflowScript 由 main agent 调 `subagent` 触发——这是「模型遵循明确、无歧义指令」的基线假设，是 AI coding 成立的前提（同 RFC §2.4 核心假设）。若模型连如此明确的触发指令都无法遵循，则模型不足以支撑 AI coding，无需在架构层纠结「是否 100% 触发」。一旦触发，内部顺序 100% 确定（await 强制 + 未 await 报错）；agent_settled 门禁 + rick 薄重试循环仅作为兜底安全网（检测模型偶发失败/部分完成、支持断点续跑），非正确性前提
2. **门禁 → rick 侧确定性脚本（runtime.Run 在 pi 会话结束后调用）+ 可选 pi hook 记录**：tasks.json 可解析 / 无 zombie running / success 有 commit_hash 的门禁语义，由 `runtime.Run` 在解析到 `agent_settled`（pi 会话结束）后**直接调用 `python3 .rick/skills/rick-gates/helper.py <doing_dir>` 校验**（确定性脚本，exit 非 0 = 门禁失败）；不再由 Go 检查——注意 pi 的 extension hook 是「通知」语义而非「拦截」，且 Python 脚本不会被 pi 作为 extension 加载，故门禁判定+重试收敛在 rick 侧，`agent_settled` 只作为「会话结束」的确定性信号；rick-gates hook 扩展（TS 包装）仅作可选的记录/通知
3. **runtime 签名切换**：runtime 的 `Execute` 改为 `Run` 返回 `(sessionID, trace, err)`（返回 sessionID 即成功；未解析出 sessionID 或未收 `agent_settled` 返回 error）；**删除 `internal/agent` 接口**（失去 executor 消费者）与 `internal/actpath`（轨迹由 runtime 的 trace 承载）
4. **删除冗余 Go 包**：`internal/executor`（dag/topological/runner/executor/retry/tasks_json/doing_check/debug_dir）、`internal/parser`、`internal/git`（commit 下沉 pi 脚本）、`internal/logging`（死代码）、`internal/cmd/tools_doing_check.go`、`internal/cmd/tools_plan_check.go`，以及引用它们的测试文件（`tools_test.go`/`doing_test.go`/`learning_test.go` 中相关断言）
5. **调用点迁移**：`handler.Doing` 从 `executor.ExecuteJob` 改为「builder 产 workflowScript 编排 + runtime.Run」；`dream.go` 的 `discoverCompletedJobs`（依赖 `executor.LoadTasksJSON`+`TasksJSON.GetAllTasks`+`TaskState.Status`）与 `learning.go` 的 `collectExecutionData`（依赖 `executor.LoadDebugContext`/`LoadTasksJSON`/`TasksJSON` + `parser.ExtractBugFrontmatter`）迁到 `workspace`（极薄读取器：定义 `TasksJSON`/`TaskState`/`LoadTasksJSON`/`LoadDebugContext` 类型 + `ExtractBugFrontmatter` 极简 frontmatter 提取）或 pi 侧脚本；commit 由 pi 在每个 task 成功后立即执行（复用 mark_task_success_skill 模式）；**prompt（builder 底层）对 parser 的解耦**：`GenerateDoingPromptFile`/`ContextManager`/`loadDebugContextLocal`/`formatTaskInfoSection`/`formatDebugContext` 移除对 `parser.Task`/`parser.DebugInfo`/`parser.ContextInfo` 的 import，改为接收字符串/路径；**同时删除 `context_helpers.go` 的 `formatOKRContent`/`formatSPECContent` 与 `ContextManager` 的 OKR/SPEC 解析方法（生产代码零调用的死代码）及其测试**；**runtime 测试清理**：删除/改写 `internal/runtime/executor_e2e_test.go` 对 `internal/actpath`/`internal/agent` 的依赖及所有 `Execute(...) (agent.AgentSession, error)` 调用点/`agent.ToolCall`/`AgentSession` 断言
6. **act-path 语义由 runtime trace 承接 + doing 新执行流语义等价**：删除 `internal/actpath` 的同时改写模板中所有 `act-path.md` 引用（`learning.md`/`learning_loop.md`/`gen-skill.md`/`gen-loop.md`/`dream.md`/`ctrl.md`）为 runtime trace（raw_session_coding.log + Trace）；清理 `easy_prompt.go`/`doing_prompt.go` 中 `check_command`/`check_step_header` 的死代码设置；`executor_realpi_smoke_test.go` 同步改 `Run` 签名（或声明弃用）；doing 新执行流的 workflowScript 拓扑、per-task commit+commit_hash 时序、tasks.json 写入者与状态机、断点续跑（--resume）、失败汇总输出，逐条对齐原 executor（跳过已完成/running→success/retry/partial/failed）
7. **doing 状态机协议 + trace 产物契约**：明确 parent 单写者、per-task「running→success→commit→回传 commit_hash→parent 写 tasks.json」时序、失败/partial 语义、tasks.json 字段级 schema 与门禁 helper.py 对齐；runtime.Run 落盘 trace 文件（如 `doing/tasks/<id>/trace.md`）供 learning/dream/ctrl 模板引用（替代 act-path.md），6 个模板的 act-path 引用改为该文件；`parser.ExtractBugFrontmatter` 在 workspace（或 prompt）落地极薄实现（删 parser 前 grep 全量 `parser.` 消费点）
8. **skill/loop 维护（删除/下沉后的知识库收敛）**：更新 `.rick/skills/check_mechanism_skill/skill.md`（删除 doing_check/plan_check 段，保留 learning_check 段）、`.rick/skills/mark_task_success_skill/skill.md`（删除 doing_check 段）、`.rick/skills/failure_feedback_skill/skill.md`（删除 internal/executor/retry.go 引用或淘汰）；`.rick/loops/do-check-mark-success-loop.md` 迁 `loops/deprecated/`（或标注失效）；同步更新 `loops/README.md`、`skills/README.md` 索引

依据：RFC §3.2「dag 调度与门禁不再由 rick 维护，利用 pi 能力直接实现」；research-report-S-bestpractice.md BP-1/BP-6/BP-8；pi docs/extensions.md（hook 生命周期）。

参考：loop `tdd-red-green-refactor-loop`；skill `verify_go_changes_skill`、`check_mechanism_skill`（本 job 删除 doing_check/plan_check 后需更新，保留 learning_check 段）、`mark_task_success_skill`（commit 模式复用，需更新删除 doing_check 段）、`dag_task_decomposition_skill`（循环依赖/拓扑排序参考）、`global_ref_sync_skill`、`pi_runtime_verification_skill`、`pi_extension_install_verification_skill`；`failure_feedback_skill`（仅概念参考，本 job 删除 internal/executor/retry.go 后需更新或淘汰）；`do-check-mark-success-loop` 本 job 删除 doing_check 后失效、不再引用。

### 问题记录


## 测试方法

正常路径：前置条件 = task6/7 完成 + 一个含 3 task（task2 依赖 task1）的 job；输入 = `doing job_N --dry-run`；操作 = `./bin/rick doing job_N --dry-run | grep -cE 'workflowScript|runs\.run'`；预期 = ≥1（doing 提示词含 pi 编排语法）。
边界（dag 拓扑 + 跳过已完成 + 门禁脚本 + 冗余包已删）：前置条件 = 同上 job，且 tasks.json 中 task1 已标记 `success`；输入 = `doing job_N --dry-run`；操作 = `test -f .rick/skills/rick-gates/helper.py` + `for d in executor parser actpath logging git agent; do test ! -d internal/$d || exit 1; done` + `grep -oE "runs\.run\('task[0-9]+'" <doing dry-run 输出>` 提取编排的 task 序号序列，断言「task1 已 success → 序列不含 task1；否则 task1 在 task2 之前」；预期 = 门禁脚本已部署、6 冗余包已删、依赖顺序正确、已完成 task 被跳过。
异常（门禁语义不丢 + runtime 签名 + 重试收敛）：前置条件 = 某 task status=success 但 commit_hash 空；输入 = `python3 .rick/skills/rick-gates/helper.py <doing_dir>`；操作 = 跑该脚本；预期 = 报 `missing commit_hash` 退出非 0。另 runtime `Run` 在 fake JSONL 缺 `agent_settled` 时返回 error（未就绪）；门禁检测到「workflow 未触发/未完成」→ handler 重试（重新生成只含剩余 pending 的编排，上限 max_retries）。

## 测试脚本路径

请创建测试脚本到: `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task8.py`

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
- 脚本应该可以直接运行: `python3 /workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task8.py`

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
python3 /workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task8.py
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
