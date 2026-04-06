# Skills Index

本目录包含可在 doing 阶段调用的 Python 脚本工具。

## 可用 Skills

| 文件 | 描述 | 触发场景 |
|------|------|----------|
| check_go_build.py | 检查 Go 项目编译 | |
| check_prompt_variables.py | 验证 rick dry-run 输出的 prompt 中是否包含指定的变量或关键词 | |
| check_variadic_api.py | 检查 Go 函数签名是否已改为 variadic（可变参数）形式，用于验证向后兼容性改造 | |
| mock_agent_testing.py | Mock AI agent for integration testing - simulates 11 claude scenarios without real API calls | |
| resolve_local_binary.py | 解析本地构建的二进制路径，优先用 ./bin/<name>，fallback 到系统安装版 | |
| rick_tools_check_pattern.py | Pattern for implementing rick tools check commands (argparse + JSON output standard) | |
| verify_template_variables.py | 验证 prompt 模板变量是否被正确替换（不含未替换的 {{placeholder}} 占位符） | |

## 调用方式

```bash
python3 .rick/skills/<filename>.py
```
