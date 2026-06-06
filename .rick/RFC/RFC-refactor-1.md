# RFC-refactor-1: 代码简洁性重构建议

## 背景

job_15 新增 ctrl 命令并将 `debug.md` skill 重命名为 `super-debugging-zh.md`。本次扫描聚焦 job_15 引入的变更，检查是否有遗留的过时引用。

## 发现的问题

### 死代码

无新增死代码。ctrl 命令和 super-debugging skill 均被正确引用。

### 过时测试引用

| 文件 | 行号 | 问题 | 说明 |
|------|------|------|------|
| `internal/prompt/manager_test.go` | 199 | `"debug"` skill 名称已过时 | skill 已重命名为 `"super-debugging-zh"`，但测试仍使用旧名称，导致 `LoadCoreSkills` 测试用例引用不存在的 skill 文件 |

### 重复逻辑

无。

### 冗余抽象

无。

## 优先级建议（Top 1）

1. **P0** `internal/prompt/manager_test.go:199` — 将 `"debug"` 替换为 `"super-debugging-zh"`，确保测试实际验证当前 skill 文件存在；删除成本为零，修复后测试可验证 super-debugging skill 注入的正确性

## 影响评估

- 修改范围：`internal/prompt/manager_test.go` 单文件单行
- 测试覆盖：修改本身就是测试文件修正，无需额外测试
- 风险：极低，只是字符串替换
