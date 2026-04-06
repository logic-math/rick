# Skills Index

本目录包含可在 doing 阶段调用的 Python 脚本工具。

## 可用 Skills

| 文件 | 描述 | 触发场景 |
|------|------|----------|
| build_and_get_rick_bin.py | 构建 rick 并返回本地二进制路径，测试脚本用此代替系统安装版 | |
| check_go_build.py | 检查 Go 项目编译 | |
| check_prompt_variables.py | 验证 rick dry-run 输出的 prompt 中是否包含指定的变量或关键词 | |
| check_variadic_api.py | 检查 Go 函数签名是否已改为 variadic（可变参数）形式，用于验证向后兼容性改造 | |
| mock_agent_testing.py | Mock AI agent for integration testing - simulates 11 claude scenarios without real API calls | |

## 调用方式

```bash
python3 .rick/skills/<filename>.py
```
