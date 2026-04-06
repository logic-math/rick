# Skills Index

本目录包含可在 doing 阶段调用的 Python 脚本工具。

## 可用 Skills

| 文件 | 描述 | 触发场景 |
|------|------|----------|
| check_go_build.py | 检查 Go 项目编译 | 任何 Go 代码变更后，验证项目能否编译通过 |
| check_variadic_api.py | 检查 Go 函数签名是否已改为 variadic（可变参数）形式，用于验证向后兼容性改造 | 重构函数签名为 variadic 形式后，验证改造是否完整 |
| mock_agent_testing.py | Mock AI agent for integration testing - simulates 11 claude scenarios without real API calls | 需要测试 claude CLI 集成逻辑但不想消耗真实 API 调用时 |
| rick_tools_check_pattern.py | Pattern for implementing rick tools check commands (argparse + JSON output standard) | 实现新的 rick tools check 命令时，参考此模式保持一致性 |

## 调用方式

```bash
python3 .rick/skills/<filename>.py
```
