# Python 测试脚本生成任务

**YOU MUST declare at the start: "I will use skill:tdd and skill:testing-anti-patterns for test generation."**

## 核心 Skills（必须加载）

在开始任何工作之前，必须读取以下 skill 文件：

- skill:tdd（测试驱动开发）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/prompts/skill_tdd_zh.md`
- skill:testing-anti-patterns（测试反模式）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/prompts/skill_testing_anti_patterns_zh.md`

你需要根据任务的测试方法生成一个 Python 测试脚本。

## 任务信息

**Task ID**: task1
**Task Name**: 创建 debug_skill.md（三阶段调试 SOP + review debug agent 协议）
**Task Goal**: 在 `internal/prompt/templates/skills/debug_skill.md` 创建完整的调试技能文件，内聚三阶段 SOP（源码推理法→增量调试法→科学实验法）、review debug agent 协议（两个触发点）、运行时观察工具指引和 debug/ 目录约定，成为 Rick 调试能力的唯一核心 skill。

### Job OKR
# Job OKR: 实现 RFC-debugging，建立三阶段科学调试体系

## 目标 (Objective)
将 Rick 的调试能力从"盲目重试"升级为基于状态机理论的科学调试——三阶段 SOP（源码推理→增量调试→科学实验）+ review debug agent + 运行时工具指引，消除调试上下文的恶性循环。

## 关键结果 (Key Results)
- KR1: `internal/prompt/templates/skills/debug_skill.md` 存在，包含准备阶段、三阶段 SOP（含回滚约束、循环上限）、review debug agent 协议（两个触发点）、运行时观察工具指引、debug/ 目录文件格式
- KR2: `super-debugging-zh.md` 已删除；`doing.md` 和 `plan.md` 模板中所有 `super_debugging*` 引用替换为 `debug_skill_path`；doing.md 的 debug{N} 调试记录格式替换为 debug_skill 加载指令
- KR3: `doing_prompt.go`、`plan_prompt.go`、`easy_prompt.go` 的 WriteSkillFile/SetVariable 调用全部从 "super-debugging-zh"/"super_debugging_path"/"super_debugging_skill_path" 切换到 "debug_skill"/"debug_skill_path"；`go test ./internal/prompt/...` 全部通过
- KR4: `internal/executor/runner.go` 的重试上下文加载逻辑从仅读 `debug.md` 扩展为同时扫描 `debug/` 目录下所有 `bug*.md` 文件；`go test ./internal/executor/...` 全部通过


### 项目 SPEC
# SPEC

## 技术栈

- 语言: Go 1.21+（主程序），Python 3.8+（tools 脚本和测试脚本）
- 框架: Cobra（CLI 命令框架），Goldmark（Markdown 解析）
- 测试: Go testing 标准库，Python unittest，Bash integration tests
- 其他: Git（版本管理），Claude Code CLI（AI agent 集成）

## 架构设计

- 架构风格: 命令行工具，模块化分层架构（cmd → executor → prompt/workspace/git）
- 模块划分: cmd（命令处理）/ executor（任务执行引擎）/ prompt（提示词管理）/ workspace（路径管理）/ parser（内容解析）/ git（Git 操作）/ callcli（Claude 集成）/ agent（接口契约）/ actpath（act-path 生成）
- 工具链模块: `rick tools` 子命令体系，plan_check/doing_check/learning_check/dream_check 四个子命令
- 接口设计: check 命令统一输出格式（✅/❌ + 描述），exit code 0=pass / 1=fail
- human-loop 模块: `rick human-loop <topic>` 命令，通过 SENSE 方法论模板引导 Claude 对复杂主题进行深度分析，产出存入 `.rick/RFC/` 目录；三个 sub agent 模板通过 Go embed 编译进二进制，运行时写出到 tmp 文件，路径注入主控 prompt
- tools 模块: `.rick/tools/*.py` 存放确定性工具脚本，agent 通过 `python3 .rick/tools/<file>.py` 调用
- **agent 接口模块** (`internal/agent/`): 定义 `AgentSession` / `AgentExecutor` 接口契约和 `ToolCall` struct；`claudecode` 子包为唯一实现，只在 `doing.go` 组合根中实例化
- **act-path 生成模块** (`internal/actpath/`): `Generate(session AgentSession, outputFile string) error`，不 import 任何具体 agent 实现，输出含执行摘要/行为轨迹/Agent 最终输出三节
- **DIP 组合根模式**: `doing.go` 是唯一 import `internal/agent/claudecode` 的地方；runner/executor/actpath 仅依赖 `internal/agent` 接口，保证可单元测试；验证: `grep -r "claudecode" internal/executor/ internal/actpath/` 应为空
- **dream 模块**: `internal/cmd/dream.go` 实现 `rick dream` 命令，不生成 act-path，自动扫描 `.rick/jobs/*/doing/tasks.json` 发现已完成 jobs、与 `dream_run_*_log.md` 对比得出待处理列表；支持 `--background`/`-p` 背景模式（`--dangerously-skip-permissions`），限制修改范围为 `.rick/wiki/`/`.rick/tools/`/`.rick/SPEC.md`

## 开发规范

- 代码风格: Go 标准格式（gofmt），函数命名 camelCase，导出函数 PascalCase
- check 命令规范: 默认只报告问题，`--auto-fix` 标志才触发 Claude 修复，保持确定性
- **三层上下文结构**（`.rick/` 内部）:
  - `SPEC.md`：规范与约束，agent 上下文的唯一入口
  - `wiki/*.md`：系统原理文档 + 技能说明书，供人类阅读和 dream 阶段参考
  - `.rick/tools/*.py`：确定性工具脚本，原子化，单一职责，JSON 输出（`{"pass": bool, "errors": [...]}`），文件首行必须有 `# Description:` 注释，调用方式 `python3 .rick/tools/<file>.py`
- Tools 脚本规范: Python 文件，argparse 解析参数，JSON 输出结果（`{"pass": bool, "errors": [...]}`）
- 测试要求: 单元测试覆盖核心逻辑，集成测试覆盖 CLI 命令，mock_agent 替代真实 Claude 调用
- **Mock Agent 同步要求**: `tests/mock_agent/mock_agent.py` 和 `.rick/tools/mock_agent_testing.py` 的 mock 输出格式必须与 doing_check/learning_check 期望严格对齐；当 check 命令格式规范变更时，两个 mock_agent 文件需同步更新
- 路径规范: 测试脚本位于 `.rick/jobs/job_N/doing/tests/`，需要 6 次 dirname 到达项目根目录
- **测试脚本 binary 规范**: 测试脚本调用 rick 命令验证新实现的功能时，必须先调用 `.rick/tools/build_and_get_rick_bin.py` 构建本地 `./bin/rick` 并使用返回的 `bin_path`，不得直接调用系统安装版（系统版不含当前任务的新代码）
- **Cobra flag 定义规范**: 全局 flag（跨命令共享，如 `--job`、`--dry-run`）用 `rootCmd.PersistentFlags()`，在 `root.go` 定义；命令级 flag 用 `cmd.Flags()`，在对应命令文件定义；全局 flag 通过 `GetXxx()` 函数统一暴露
- Go variadic 改造模式: 当需要让现有必传参数变为可选时，使用 variadic（`...T`）而非新增无参构造函数，保持接口唯一性；调用方无需修改
- 包内函数共享: 同一 Go 包内的函数（如 `callClaudeCodeCLI`）可在多个文件中直接调用，不需要重新声明或导出
- Dry-run 规范: `--dry-run` 标志必须输出完整的 prompt 内容（而非占位消息），便于调试和验证上下文注入效果
- **测试断言精确性**: dry-run 输出包含大量上下文文本，断言需先定位 section（如 `## 可用的项目 Skills`）再检查内容，避免全文搜索误判
- **task.md 测试方法精确性**: task.md 中"测试方法"描述的命令行调用必须基于工具**实际存在的参数接口**，不得引用尚未实现的参数。plan 阶段生成测试脚本前应验证 `.rick/tools/` 下对应工具的 `--help` 输出
- **embed.FS 目录嵌入**: `//go:embed dir`（目录）必须绑定 `embed.FS` 类型；`//go:embed file`（单文件）可绑定 `string`；两者可在同一文件共存。`_ "embed"` 改为 `"embed"` 才能使用 `embed.FS`
- **JSON 输出编码约定**: 所有 Python 工具/测试脚本的 `json.dumps()` 调用必须加 `ensure_ascii=False`，避免中文字符被转义为 `\uXXXX` 导致字符串匹配失败
- **接口签名协商**: 并行 task 中若涉及接口定义和实现，接口 task 应先完成后实现 task 才开始；或在 plan 阶段明确接口签名（不含 context.Context，避免标准库强制依赖）
- **同包测试 mock 命名**: 同一 Go 包的多个测试文件共享命名空间；mock struct 应使用区分前缀（如 `runnerMockExecutor` vs `executorMockExecutor`）避免冲突

## 工程实践

- 版本控制: Git，每个任务完成后独立 commit（commit message 包含 task ID）
- 知识合并: learning 产出经人工审核后手动 `git merge --no-ff`（`rick tools merge` 命令尚未实现，见 RFC-005）
- 持续集成: `go test ./...` 覆盖单元测试，`bash tests/tools_integration_test.sh` 覆盖集成测试
- 发布流程: `./scripts/build.sh` 构建，`./scripts/install.sh` 安装到 `~/.rick/bin/rick`

## 路径约定

- `.rick/RFC/`: human-loop 会话产出文档目录，由 `GetRFCDir()` 管理，`rick human-loop` 执行时自动创建
- `.rick/jobs/job_N/`: 每次 job 的工作目录，包含 plan/doing/learning 三个子目录
- `.rick/jobs/job_N/plan/OKR.md`: job 级 OKR，由 plan 阶段 Claude 生成，doing/learning 阶段读取
- `.rick/wiki/`: 系统原理文档 + 技能说明书（`.md`），供人类阅读和 dream 阶段参考；`wiki/README.md` 为所有文档索引
- `.rick/dream/`: dream 目录，存放 `dream_run_*_log.md` 和 `prompts/`；待处理 jobs 由程序自动扫描 tasks.json 发现，无需手工维护索引文件
- `.rick/dream/run_log_{n}.md`: learning 阶段 Step 6 写入的度量文件，格式 `| Job | 模型 | 错误次数 | 工具调用轮次 | 备注 |`
- `.rick/tools/`: 确定性 Python 工具脚本（**只含 `.py` 文件**）；每个脚本首行必须有 `# Description:` 注释；调用方式 `python3 .rick/tools/<file>.py`
- `doing/tasks/{taskID}/act-path.md`: 任务执行后自动生成的行为轨迹文件，含工具调用、报错次数、执行时长
- `doing/tasks/{taskID}/raw_session.log`: Claude Code NDJSON 原始流式输出，每行一个 JSON 对象（非 JSON 行也写入）

## 命令规范

### rick doing（DIP 全链路）

- `doing.go` 是唯一 import `internal/agent/claudecode` 的地方（**组合根**）
- `runner.go` 和 `executor.go` 只依赖 `internal/agent` 接口，不 import claudecode
- `actpath.Generate(session, outputFile)` 在每个 task 的 `agentExecutor.Execute` 完成后调用
- session 为 nil 时跳过 act-path 生成（nil guard），不 panic

### rick dream

- 自动扫描 `.rick/jobs/*/doing/tasks.json` 发现所有 tasks 均 "success" 的 jobs（已完成）
- 对比 `.rick/dream/dream_run_*_log.md` 排除已处理 jobs，取最多 5 个待处理 jobs
- `--job_num <n>`：调整每次处理的 job 数量（默认 5）
- `--background`/`-p`：背景模式，使用 `--dangerously-skip-permissions` 非交互执行
- `--dry-run`：输出完整提示词（含 sense + evolve-skills core-skills），不调用 Claude
- **变更约束**: 仅允许修改 `.rick/wiki/`、`.rick/tools/`、`.rick/SPEC.md`，严禁修改业务代码

### NDJSON 解析规范

- Claude Code `--output-format stream-json` **必须加 `--verbose`**，否则报错退出
- `tool_use`/`tool_result` 嵌套在 `message.content[]` 内，不在顶层
- 非 JSON 行: `log.Printf("warn: skip non-json line %d: %s")` 后继续，不 panic
- 截断规范: Input/Output 截断 300 字符，FinalMessage 截断 200 字符，用 `[]rune` 处理 Unicode

### human-loop 规范

- dry-run 输出中 sub agent 路径为占位符格式（如 `<tmp>/human_loop_think_*.md`），不含真实 `/tmp/` 路径
- 三个 sub agent 模板（think/learn/express）通过 Go embed 编译进二进制，运行时写出到系统 tmp，路径注入主控 prompt
- 自动创建 `.rick/RFC/` 目录（MkdirAll，幂等）
- 复用 `callClaudeCodeCLI`（plan.go 中定义，同包内共享，不重复声明）
- 会话结束后 defer 清理所有 tmp 文件（主 prompt + 三个 sub agent）
- 验证 human-loop dry-run 输出：`python3 .rick/tools/check_prompt_variables.py --phase human-loop --topic '测试主题' --keywords human_loop_think`

### rick plan --job

- `--job <job_id>` 为全局 flag（定义在 root.go），plan.go 通过 `GetJobID()` 读取，不在 plan.go 中重复定义
- 指定 `--job` 时跳过 `NextJobID()`，直接复用已有 job 的 plan 目录
- plan 目录不存在时返回明确错误，不自动创建

### rick plan --dry-run

- 生成完整 plan prompt 并打印到 stdout（通过 `runPlanDryRun()` 函数）
- 不调用 Claude，不创建任何文件
- 输出包含所有注入内容：job_plan_dir、SPEC 路径等

### rick doing --dry-run

- 打印完整 doing prompt 内容到 stdout
- 不调用 Claude，不执行任何任务
- 展示第一个非 success 状态的任务（从 tasks.json 读取，不硬编码 task1）

### rick learning --dry-run

- 生成完整 learning prompt 并打印到 stdout（通过 `runLearningDryRun()` 函数）
- 不调用 Claude，不创建任何文件
- 输出包含所有注入内容：okr_content、task_md_content、debug 记录、act_path_content 等

### rick ctrl

- `--job <job_id>` 为必传参数，无默认值
- 调用 `GenerateCtrlPromptFile(jobID, rickDir)` 生成 prompt，写入 `doing/prompts/ctrl_prompt.md`，返回路径
- `callClaudeCodeCLI(cfg, promptFile)` 启动交互式 Claude 会话（与 plan/human-loop 共用同一函数）
- ctrl 与 doing 之间**仅通过文件通信**：reading tasks.json + raw_session_coding.log，writing tasks.json + plan/task<N>.md
- 场景 A（追加指令）：在 `plan/task<N>.md` 末尾追加 `## 干预指令 (Intervention)` 章节写入人类指令，通常同时执行场景 B
- 场景 B（重置 task）：将 `status` 改为 `"pending"`，清空 `error` 字段，更新 `updated_at`；若目标 task 正在运行（`running`），直接重置无效，需告知人类先 Ctrl+C 停止 doing
- **变更约束**：只能修改 `doing/` 和 `plan/` 下的文件，不得修改 `.rick/` 其他目录
- dry-run 输出完整 prompt（通过 `runCtrlDryRun()`），需指定 `--job` 否则报错退出


### 问题记录
暂无

## 测试方法

操作：`cat internal/prompt/templates/skills/debug_skill.md | head -5`
预期：输出包含 `name: debug-skill` 的 frontmatter
预期：全部输出 ✅
预期：输出 ≥ 4（阶段一建立假设、阶段二简化复现、阶段三简化复现、阶段三建立传播链）
预期：两行均输出 ✅
预期：✅ sense skill 路径声明存在
预期：两行均 ✅
预期：两行均 ✅

## 测试脚本路径

请创建测试脚本到: `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/tests/task1.py`

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
- 脚本应该可以直接运行: `python3 /Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/tests/task1.py`

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
python3 /Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/tests/task1.py
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
