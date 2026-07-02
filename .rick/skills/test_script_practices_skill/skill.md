# skill:test-script-practices（测试脚本编写规范）

## 触发场景

在 plan 或 doing 阶段编写/调试任务测试脚本（`.rick/jobs/job_N/doing/tests/taskN.py`）时使用，特别是：
- 测试脚本调用 `rick` 二进制命令
- 测试脚本需要定位项目根目录
- 测试脚本调用 `.rick/skills/` 下的辅助脚本
- 测试脚本验证文件路径或 dry-run 输出内容

## 预期效果

- 一次运行通过，不因路径、binary 版本、section 误判等问题多次重试
- 测试脚本输出标准 JSON：`{"pass": true, "errors": []}` 或 `{"pass": false, "errors": [...]}`

## 核心内容

### 必做：项目根路径（6 次 dirname）

测试脚本路径：`.rick/jobs/job_N/doing/tests/taskN.py` → 需要 **6 次** dirname 到达项目根：

```python
import os

def get_project_root():
    p = os.path.abspath(__file__)
    for _ in range(6):
        p = os.path.dirname(p)
    return p

project_root = get_project_root()
```

### 必做：使用本地构建的 rick binary

调用 `rick` 命令时必须先用辅助脚本构建本地版（系统版不含当前任务的新代码）：

```python
import subprocess, json

result = subprocess.run(
    ["python3", ".rick/skills/mark_task_success_skill/build_rick.py"],
    capture_output=True, text=True, cwd=project_root
)
build_info = json.loads(result.stdout)  # 注意：返回 JSON，不是纯文本路径
assert build_info["pass"], f"Build failed: {build_info}"
rick_bin = build_info["bin_path"]
```

### 必做：section 精准断言（不能全文搜索）

验证 dry-run 输出中某 section 的内容时，先定位 section 边界再检查：

```python
import re

match = re.search(r'## 可用的项目 Skills(.*?)(?=^##|\Z)', output, re.DOTALL | re.MULTILINE)
skills_section = match.group(1) if match else ""
assert ".py" not in skills_section  # 只检查 section 内容，不全文搜索
```

### 陷阱清单

| # | 现象 | 修复 |
|---|------|------|
| 1 | 系统 rick 不含新代码 | 用 `build_rick.py` 构建本地版，从 JSON 取 `bin_path` |
| 2 | 5 次 dirname 只到 `.rick/`，少一层 | 用 6 次 dirname |
| 3 | 引用不存在的工具参数 | 先 `tool --help` 验证参数存在 |
| 4 | `.rick/wiki/` vs `wiki/` 路径歧义 | 使用绝对路径，含 `.rick/` 前缀 |
| 5 | 否定引用导致字符串匹配误报 | 改用精确定位或修改源文件措辞 |
| 6 | 全文搜索被 tools section 误判 | 提取目标 section 范围后再断言 |
| 7 | `check_variadic_api.py` 不支持 method | 改用 grep 匹配 method 签名 |
| 8 | `~/.rick/config.json` 高 max_retries 导致测试超时 | 测试开头 `t.Setenv("HOME", t.TempDir())` + 写入 `{"max_retries":2}` |

### 标准测试脚本模板

```python
import os, subprocess, json, sys

def get_project_root():
    p = os.path.abspath(__file__)
    for _ in range(6):
        p = os.path.dirname(p)
    return p

project_root = get_project_root()

def build_rick():
    out = subprocess.run(
        ["python3", ".rick/skills/mark_task_success_skill/build_rick.py"],
        capture_output=True, text=True, cwd=project_root
    )
    info = json.loads(out.stdout)
    assert info["pass"], info
    return info["bin_path"]

def main():
    rick_bin = build_rick()
    result = subprocess.run(
        [rick_bin, "tools", "doing_check", "job_N"],
        capture_output=True, text=True, cwd=project_root
    )
    errors = []
    if result.returncode != 0:
        errors.append(result.stdout + result.stderr)
    print(json.dumps({"pass": len(errors) == 0, "errors": errors}, ensure_ascii=False))
    sys.exit(0 if not errors else 1)

if __name__ == "__main__":
    main()
```
