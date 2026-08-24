# Python 测试脚本生成任务

**YOU MUST declare at the start: "I will use skill:tdd and skill:testing-anti-patterns for test generation."**

## 核心 Skills（必须加载）

在开始任何工作之前，必须读取以下 skill 文件：

- skill:tdd（测试驱动开发）：`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/prompts/skill_tdd_zh.md`
- skill:testing-anti-patterns（测试反模式）：`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/prompts/skill_testing_anti_patterns_zh.md`

你需要根据任务的测试方法生成一个 Python 测试脚本。

## 任务信息

**Task ID**: task2
**Task Name**: 产出 rick 第一份 spec（四层架构 + 5 模块 + env 四职责契约）
**Task Goal**: 按 task1 定义的 spec 规范，产出 rick 项目第一份 spec（KR1.2），使 rick 拥有这份 spec（信息内核）。spec 覆盖收敛后的最终架构：
- **四层架构（调用逐级往下）**：
  - 第一层 入口：CLI / TUI / WEB-UI（路由命令、解析参数、交互呈现）
  - 第二层 调度聚合：handler（接受入口参数，编排 env/runtime/builder 完成功能）
  - 第三层 执行：env（pi/dsh 及扩展的检查/安装/配置/维护）+ runtime（pi/dsh 调用封装：参数解析+调用）+ builder（按入口拼接 pi/dsh 提示词产物）
  - 第四层 基础设施：pi（当前 runtime）/ dsh（预留 runtime）/ workspace（路径解析）/ config（~/.rick/config.json 加载）
- 调用关系：上层调下层（逐级往下），下层不回调上层；**例外一**：env ↔ dsh 相互调用（dsh 生态交互关系，非纯单向；不单列 dshRuntime/dshBuilder 节点，链接直接连到具体组件 env 与 dsh）；**例外二**：TUI / WEB-UI 跨层直连 pi/dsh（交互界面直接驱动 runtime，绕过 handler/env/runtime/builder）；**例外三**：组合根（cmd 的 RunE 懒加载实例化 piRuntime/piEnv/pibuilder 注入 handler）是 DIP 组合根模式，越级豁免；**例外四**：workspace/config 是跨层叶子基础设施（路径解析/配置加载），可被任意层直接使用，不参与功能调用链的「逐级往下」约束
- 5 模块职责与边界（含 env 四职责、runtime 职责、handler 职责、builder 三件）；`internal/prompt` = builder 三件中 templates 的承载包（L3）；L3 内部（env/runtime/builder）可复用共享路径工具（AgentDir/RuntimeDir/RuntimeBin/AgentEnv 等），不视为越级回调
- builder 三件契约（templates = go embed 内嵌提示词；pibuilder = pi 统一入口组合子 builder；xxxxbuilder = 扩展位）
- runtime 契约：拉起 pi + 内部校验 session 就绪 + 返回 (sessionID, 行为轨迹)
- 删除清单：executor（调度→pi）、parser（读/校验→pi）、actpath（轨迹→runtime）、logging（死代码）、git（→pi 脚本）、agent 接口（失去消费者）
- 验收标准：功能等价 = 通过所有功能验收；rick 做薄（dag 调度与门禁下沉 pi）；**单一 runtime（pi）为当前实现**——为将来 deepseek harness(dsh) 预留三扩展 seam：builder 的 `RuntimeBuilder`（= xxxxbuilder 转义层）、runtime 的 `Runtime`（`Name`/`Run`）、env 的 `RuntimeEnv`（`Ensure`/`DeployCustomizations`/`CheckReady`）+ config `runtime` 字段（默认 `pi`）；当前 pi 是唯一实现，不写 dsh 代码

依据：`.rick/draft/rfc/rfc-rick-三层架构重构与spec信息内核-2026-08-14.md` §4（目标架构）、§6 O1 KR1.2。此 spec 是 task3~11 重构的「契约」。

### 问题记录


## 测试方法

正常路径：前置条件 = `.rick/domain/spec.md`（task1）存在；输入 = rick-spec.md 正文；操作 = 写 `.rick/domain/rick-spec.md` + `git add`；预期 = `test -f .rick/domain/rick-spec.md` 返回 0。
边界（5 模块 + env 四职责 + 四层架构覆盖）：前置条件 = rick-spec.md 已写；输入 = 待写入正文；操作 = `for w in cli handler env builder runtime; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done` + `for w in 安装 生态扩展 定制 就绪; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done` + `for w in 第一层 第二层 第三层 第四层 CLI TUI WEB-UI; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done`；预期 = exit 0（5 模块名 + env 四职责 + 四层架构关键词各自命中）。
异常（与 RFC 一致 + 无变量泄漏 + 扩展 seam）：前置条件 = rick-spec.md 已写；操作 = `for w in dag 门禁 sessionID; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done` + `for w in RuntimeBuilder RuntimeEnv runtime; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done` + `grep -c '{{' .rick/domain/rick-spec.md`（=0）；预期 = exit 0（dag/门禁/sessionID + 三 seam 各自命中）且无 `{{`。

## 测试脚本路径

请创建测试脚本到: `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task2.py`

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
- 脚本应该可以直接运行: `python3 /workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task2.py`

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
python3 /workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tests/task2.py
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
