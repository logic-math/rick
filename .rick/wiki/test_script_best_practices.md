# test_script_best_practices

## 触发场景

在 plan 或 doing 阶段编写/调试测试脚本（`tasks/{taskID}/tests/taskN.py`）时使用，特别是：
- 测试脚本调用 `rick` 二进制命令
- 测试脚本需要定位项目根目录
- 测试脚本调用 `.rick/tools/` 下的 Python 工具
- 测试脚本验证文件路径是否存在

## 使用的 Tools

- `.rick/tools/build_and_get_rick_bin.py` — 构建本地 rick 二进制，返回 `{"pass": true, "bin_path": "..."}`
- `.rick/tools/check_go_build.py` — 确认 Go 编译正常

## 陷阱清单与修复

### 陷阱 1：使用系统 rick 而非本地构建版

**现象**：测试调用 `rick tools doing_check job_N`，但系统安装版不含当前任务新增的代码。

**修复**：
```python
import subprocess, json, os

# 先构建本地 rick
result = subprocess.run(
    ["python3", ".rick/tools/build_and_get_rick_bin.py"],
    capture_output=True, text=True, cwd=project_root
)
build_info = json.loads(result.stdout)  # 注意：返回 JSON，不是纯文本路径
assert build_info["pass"], f"Build failed: {build_info}"
rick_bin = build_info["bin_path"]

# 使用本地 binary
result = subprocess.run(
    [rick_bin, "tools", "doing_check", job_id],
    capture_output=True, text=True, cwd=project_root
)
```

---

### 陷阱 2：dirname 次数不足，无法定位项目根目录

**现象**：测试脚本从 `.rick/jobs/job_N/doing/tests/taskN.py` 出发，用 5 次 `os.path.dirname` 只到了 `.rick/`，缺少第 6 次才能到 `projectRoot`。

**修复**：
```python
import os

# 正确的 6 次 dirname
project_root = os.path.dirname(  # 6: .rick/ → projectRoot
    os.path.dirname(             # 5: tests/ → doing/
        os.path.dirname(         # 4: doing/ → job_N/
            os.path.dirname(     # 3: job_N/ → jobs/
                os.path.dirname( # 2: jobs/ → .rick/
                    os.path.dirname(os.path.abspath(__file__))  # 1: file → tests/
                )
            )
        )
    )
)
```

---

### 陷阱 3：引用不存在的工具参数接口

**现象**：task.md 写的测试方法引用 `--command`/`--variables` 等理想化参数，但工具实际未实现这些参数。

**修复**：在 task.md 描述测试方法时，必须先验证工具实际接口：
```bash
python3 .rick/tools/check_prompt_variables.py --help
# 仅使用 --help 输出中列出的参数
```

test.py 中优先使用 dry-run 输出内容检查关键词，而非调用不确定的工具参数：
```python
result = subprocess.run([rick_bin, "human-loop", "--dry-run", topic], ...)
assert "human_loop_think" in result.stdout  # 检查关键词，不依赖特定参数
```

---

### 陷阱 4：路径歧义（`.rick/wiki/` vs `wiki/`）

**现象**：Agent 将文件写入 `.rick/wiki/`，测试检查 `wiki/testing.md`（项目根相对路径），两者不同。

**修复**：task.md 中明确使用绝对路径或 `.rick/` 前缀：
```markdown
# 测试方法
1. 检查 `.rick/wiki/testing.md` 是否存在（注意：路径从项目根开始，含 .rick/）
```

测试脚本中：
```python
assert os.path.exists(os.path.join(project_root, ".rick", "wiki", "testing.md"))
```

---

### 陷阱 5：字符串匹配误报（negated phrase）

**现象**：测试检查文件中"不含某段文字"，但文件中有以否定形式引用该文字的句子（如："这不是'遇到问题才记录'的可选项"）。

**修复**：改用更精确的正则或通过上下文判断，或修改源文件中的措辞避免引用原文。

---

### 陷阱 6：dry-run 全文搜索导致 section 误判

**现象**：测试检查 `".py" in output` 来验证 skills section 不含 `.py`，但 tools section 合法包含 `.py` 路径，导致永远失败。

**修复**：提取目标 section 的内容范围后再做断言：
```python
# 提取 "## 可用的项目 Skills" 到下一个 "##" 之间的内容
import re
match = re.search(r'## 可用的项目 Skills(.*?)(?=^##|\Z)', output, re.DOTALL | re.MULTILINE)
skills_section = match.group(1) if match else ""
assert ".py" not in skills_section, "Skills section should not contain .py files"
```

---

### 陷阱 7：check_variadic_api.py 不支持 Go methods

**现象**：`.rick/tools/check_variadic_api.py` 使用 `func\s+{func_name}\s*\(` 正则，无法匹配 Go method（如 `func (tr *TaskRunner) buildTestGenerationPromptFile(...)`），返回 "Function not found"。

**修复**：验证 method 的 variadic 签名时，改用 grep 直接查找：
```python
result = subprocess.run(
    ["grep", "-n", "...TestGenContext", "internal/executor/runner.go"],
    capture_output=True, text=True, cwd=project_root
)
assert result.returncode == 0, "variadic signature not found"
```

---

### 陷阱 8：全局 ~/.rick/config.json 污染测试导致超时

**现象**：`go test ./internal/cmd/...` 在 60s+ 后超时，stack trace 卡在 `retry.go time.Sleep`；本地 `~/.rick/config.json` 设置了高 `max_retries`（如 16），导致 retry sleep 累计超出 test timeout。

**修复**：在每个涉及 retry/config 的 Go 测试函数开头注入临时 HOME，覆盖全局配置：

```go
func TestXxx(t *testing.T) {
    dir := t.TempDir()
    t.Setenv("HOME", dir)
    // 写入低 max_retries 的本地 config
    cfg := `{"max_retries":2}`
    _ = os.MkdirAll(filepath.Join(dir, ".rick"), 0755)
    _ = os.WriteFile(filepath.Join(dir, ".rick", "config.json"), []byte(cfg), 0644)
    // ... 测试逻辑
}
```

**因果链**：`max_retries:16` → retry sleep = 1+2+...+15 = 120s，远超 `-timeout 60s`；CI 无 `~/.rick/config.json` 故不受影响。

**识别信号**：本地运行时测试挂起 > 30s，stack trace 卡在 `retry.go:xxx time.Sleep`，在 CI 中正常。

---

## 执行步骤

1. **确认项目根路径**：使用 6 次 dirname，或 `os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../../../.."))` 等等价写法
2. **构建本地 binary**：`python3 .rick/tools/build_and_get_rick_bin.py`，解析 JSON 中的 `bin_path` 字段
3. **验证工具接口**：调用 `tool --help` 确认参数名称，不臆测
4. **路径使用 `.rick/` 前缀**：涉及 `.rick/` 内文件时明确写全路径
5. **section 精准断言**：验证某个注入 section 的内容时，先定位 section 边界再检查
6. **运行测试**：`python3 .rick/jobs/job_N/doing/tests/taskN.py`，确认 `{"pass": true}`

## 示例

```python
import os, subprocess, json, re

def get_project_root():
    """6 dirname calls from .rick/jobs/job_N/doing/tests/taskN.py"""
    p = os.path.abspath(__file__)
    for _ in range(6):
        p = os.path.dirname(p)
    return p

project_root = get_project_root()

# Build local rick (returns JSON, not plain text)
out = subprocess.run(
    ["python3", ".rick/tools/build_and_get_rick_bin.py"],
    capture_output=True, text=True, cwd=project_root
)
info = json.loads(out.stdout)
assert info["pass"], info
rick_bin = info["bin_path"]

# Test with local binary
result = subprocess.run(
    [rick_bin, "tools", "doing_check", "job_5"],
    capture_output=True, text=True, cwd=project_root
)
assert result.returncode == 0, result.stdout + result.stderr
print(json.dumps({"pass": True, "errors": []}, ensure_ascii=False))
```
