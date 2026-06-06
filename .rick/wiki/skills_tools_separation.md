# .rick/ 三层上下文结构

## 概述

Rick 的知识体系在 `.rick/` 目录内形成三层结构：`SPEC.md → wiki/ → tools/`。

```
.rick/
  SPEC.md       ← 规范与约束，agent 上下文入口
  wiki/         ← 系统原理文档 + 技能说明书（.md）
  tools/        ← 确定性工具脚本（.py）
```

## 三层职责

### SPEC.md（第一层）
- Agent 的主要上下文来源，描述架构约束、开发规范、路径约定
- Dream 阶段只精简不堆砌，保持 ≤ 500 行
- 变更需经过 job 流程，不直接随意修改

### wiki/（第二层）
- **系统原理文档**：how-things-work，供人类理解系统运行机制
- **技能说明书**：actionable guides，描述在特定场景下的操作步骤
- `wiki/README.md` 为所有文档的统一索引
- Dream 阶段可直接修改，更新过时内容、添加新技能

### tools/（第三层）
- 确定性 Python 脚本，单一职责，原子化操作
- 每个脚本首行必须有 `# Description:` 注释
- 统一 JSON 输出格式：`{"pass": bool, "errors": [...]}`
- 调用方式：`python3 .rick/tools/<file>.py`

## 如何使用

### 调用工具

```bash
# 构建本地 rick binary
python3 .rick/tools/build_and_get_rick_bin.py

# 检查 Go 编译
python3 .rick/tools/check_go_build.py

# 验证 prompt 变量
python3 .rick/tools/check_prompt_variables.py --phase doing --keywords "skill:tdd"
```

### 添加新工具

1. 在 `.rick/tools/` 创建 `.py` 文件
2. 首行写 `# Description: 一句话描述`
3. 输出 `{"pass": bool, "errors": [...]}` JSON，`sys.exit(0/1)`

### 添加新技能说明书

1. 在 `.rick/wiki/` 创建 `.md` 文件
2. 包含：触发场景、使用的 Tools、执行步骤
3. 在 `wiki/README.md` 添加索引条目

## 当前工具列表

| 文件 | 描述 |
|------|------|
| `build_and_get_rick_bin.py` | 构建本地 rick binary，返回 `{"pass": true, "bin_path": "..."}` |
| `check_go_build.py` | 检查 Go 项目是否能成功编译 |
| `check_prompt_variables.py` | 验证 rick dry-run 输出包含指定关键词或变量 |
| `check_variadic_api.py` | 检查 Go 函数签名是否已改为 variadic 形式（仅支持 standalone function） |
| `mock_agent_testing.py` | Mock Claude Code Agent，模拟多种场景，支持集成测试 |
