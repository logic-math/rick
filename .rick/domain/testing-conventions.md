# 测试约定

## go test 范围精确性

**禁止**跑全量 `go test ./internal/...`，会混入依赖真实 API key 的无关测试。

```bash
# ✅ 精确匹配改动包
go test ./internal/executor/... -v
go test ./internal/prompt/... -v
go test ./internal/cmd/... -timeout 60s -v

# ❌ 全量（会混入无关测试）
go test ./internal/...
```

## 同包测试 mock 命名

同一 Go 包的多个测试文件共享命名空间，mock struct 必须使用区分前缀：

```go
// runner_test.go
type runnerMockExecutor struct { ... }

// executor_test.go
type executorMockExecutor struct { ... }

// ❌ 两个文件都叫 mockAgentExecutor → 编译错误
```

## Mock Agent 同步要求

`tests/mock_agent/mock_agent.py` 和 `.rick/skills/check_mechanism_skill/mock_agent_testing.py` 的 mock 输出格式必须与 doing_check/learning_check 期望**严格对齐**。

当 check 命令格式规范变更时，两个 mock_agent 文件需**同步更新**，否则集成测试会产生假阳性。

验证：
```bash
bash tests/tools_integration_test.sh
```

## JSON 输出编码约定

所有 Python 工具/测试脚本的 `json.dumps()` 必须加 `ensure_ascii=False`：

```python
# ✅ 正确
print(json.dumps({"pass": True, "errors": ["中文错误"]}, ensure_ascii=False))

# ❌ 错误（中文变为 \uXXXX，字符串匹配失败）
print(json.dumps({"pass": True, "errors": ["中文错误"]}))
```

## Python 工具脚本规范

- argparse 解析参数
- JSON 输出结果：`{"pass": bool, "errors": [...]}`
- 文件首行必须有 `# Description:` 注释
- 调用方式：`python3 .rick/skills/{name}_skill/helper.py [args]`

## 测试断言精确性（dry-run 输出）

dry-run 输出包含大量上下文，断言**必须先定位 section** 再检查内容，避免全文搜索误判：

```python
import re, subprocess

output = subprocess.run([rick_bin, "doing", job_id, "--dry-run"], ...).stdout

# ✅ 定位 section 后断言
match = re.search(r'## 可用的项目 Skills(.*?)(?=^##|\Z)', output, re.DOTALL | re.MULTILINE)
skills_section = match.group(1) if match else ""
assert "my-skill" in skills_section

# ❌ 全文搜索（tools section 也含 .py，永远匹配）
assert ".py" not in output
```

## task.md 测试方法精确性

task.md 中"测试方法"描述的命令调用必须基于**实际存在的参数接口**：

```bash
# 先验证工具参数接口
python3 .rick/skills/verify_go_changes_skill/check_go_build.py --help

# 只使用 --help 输出中列出的参数，不臆测
```

**不使用**工具不存在的参数（plan 阶段凭想象写 `--command`/`--variables` 等）。

## 集成测试

```bash
bash tests/tools_integration_test.sh
```

单元测试覆盖核心逻辑，集成测试覆盖 CLI 命令，mock_agent 替代真实 Claude 调用。
