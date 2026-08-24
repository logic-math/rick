# Python 测试脚本生成任务

**YOU MUST declare at the start: "I will use skill:tdd and skill:testing-anti-patterns for test generation."**

## 核心 Skills（必须加载）

在开始任何工作之前，必须读取以下 skill 文件：

- skill:tdd（测试驱动开发）：`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/prompts/skill_tdd_zh.md`
- skill:testing-anti-patterns（测试反模式）：`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/prompts/skill_testing_anti_patterns_zh.md`

你需要根据任务的测试方法生成一个 Python 测试脚本。

## 任务信息

**Task ID**: task3
**Task Name**: 落地 env 模块（四职责：pi + 生态扩展 + rick 定制 + 就绪 check）
**Task Goal**: 按 spec（task2）落地 KR2.2 并升级为 env 四职责。新建 `internal/env` 包，把 pi 相关能力收口为「保证 pi 正确启动」的统一管理器，职责：
1. **安装并更新 pi agent**：迁移 `ensurePI`/`installManagedPI`/`requireNodeForPiInstall`/`piVersion`（来自 tools_init_pi.go）
2. **安装并更新 pi 生态扩展/插件/skill**：迁移 `ensureNpmExtension`/`piListContains`/`verifyExtensions`（pi-subagents/pi-web-access/主题）
3. **安装并更新 rick 自有定制（hook/skill/agent）**：新增 `DeployRickCustomizations()`，把 rick-gates hook 扩展、think/research/exporter agent frontmatter、rick skills 落盘到 `~/.rick/pi/agent/`（agents/、extensions/、skills/）——rick 全局方法（「你是 rick 的 agent，遵循 loops/skills/domain 体系」）作为 builder method 的固定前缀（task5），不单独落盘 `APPEND_SYSTEM.md`（避免与 `--append-system-prompt` 覆盖冲突）
4. **就绪 check 函数**：新增 `IsPIReady`/`CheckPIInstalled`/`CheckEcosystemExtensions`/`CheckRickAgents`/`CheckRickHooks`（纯「功能点就绪」，不含 session）

**扩展 seam（为将来 dsh runtime 留扩展位）**：env 四职责按 `RuntimeEnv` 接口组织 `{ Ensure() error; DeployCustomizations() error; CheckReady() []string }`，pi 实现 = `piEnv`；将来 dsh = `dshEnv`（安装方式/扩展机制/定制落盘格式/就绪 check 各自实现），cli/handler 不改。

`internal/cmd/tools_init_pi.go` 变薄为 Cobra 入口，调用 env 导出函数；init-pi 行为不变（幂等、全 ✅）。env 从 `internal/runtime` import `AgentEnv`/`AgentDir`/`RuntimeDir`/`RuntimeBin`/`SettingsPath`/`EnsureAgentDir`/`FileExists` 注入 `PI_CODING_AGENT_DIR=~/.rick/pi/agent` 保持配置隔离（尊重 RICK_PI_AGENT_DIR/HOME）——依赖 task4 先落地 runtime，避免与 piagent 改名冲突。theme 相关（`embeddedThemes` go:embed + `ensureRickTheme`/`setTheme`/`currentTheme`/`purgeTokyoNight`/`piSettingsPath`/`ensureHideThinkingBlock`）随 env 迁移：`internal/cmd/themes/*.json` 移到 `internal/env/themes/`，go:embed 指令随之更新；`tools_theme.go` 变薄调用 env。

参考：loop `agent-runtime-bootstrap-loop`；skill `verify_go_changes_skill`、`global_ref_sync_skill`、`pi_extension_install_verification_skill`、`pi_runtime_verification_skill`、`fake_binary_script_skill`、`subprocess_env_isolation_skill`、`pi_theme_verification_skill`；bugs.md「fake pi PATH 替换」「pi 扩展假成功」「RICK_PI_AGENT_DIR 隔离」；pi docs/extensions.md（hook 扩展入口）。

### 问题记录


## 测试方法

正常路径：前置条件 = task2、task4 完成、隔离 HOME（`t.Setenv("HOME", t.TempDir())`，PATH 指向含 fake pi/node/npm/npx 的目录，或先预置 managed runtime 使 node 检查跳过）；输入 = 无；操作 = `go build -o bin/rick ./cmd/rick && ./bin/rick tools init-pi`；预期 = exit 0 且 stdout 含 `✅ pi environment ready`。
边界（幂等 + 四职责 check）：前置条件 = init-pi 已成功一次；输入 = 无；操作 = 再次 `./bin/rick tools init-pi` + `go test ./internal/env/... -run TestIsPIReady -v`；预期 = 第二次仍 exit 0 且无 `newly installed`；`IsPIReady()` 返回 ok=true、missing 为空，且 `CheckPIInstalled`/`CheckEcosystemExtensions`/`CheckRickAgents`/`CheckRickHooks` 各自就绪（返回 nil/空切片）。
异常（缺 node/npm + check 报告缺失）：前置条件 = runtime 未装；输入 = 无；操作 = `HOME=$(mktemp -d) PATH=$(mktemp -d) ./bin/rick tools init-pi`（PATH 指向空目录，确保 `exec.LookPath("node")` 必然失败；保留 HOME 避免 config 加载失败干扰）；预期 = stderr 含 `requires Node.js`，exit 1，不 panic。另测 `CheckEcosystemExtensions()` 在某扩展缺失时返回非空切片（不就绪即列出）。

## 测试脚本路径

请创建测试脚本到: `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task3.py`

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
- 脚本应该可以直接运行: `python3 /workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task3.py`

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
python3 /workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task3.py
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
