# core-skills 精准注入机制

## 概述

Rick 将核心 skill 文件通过 `embed.FS` 编译进二进制，在不同 SOP 阶段精准注入对应 skill，避免信息污染（如 doing 阶段不注入 gen-skill，plan 阶段不注入 super-debugging）。

## 工作原理

### embed 声明（manager.go）

```go
import "embed"  // 从 _ "embed" 改为 "embed" 暴露包

//go:embed templates/plan.md
var planTemplate string  // 现有 string 变量不变

//go:embed templates/skills
var skillsFS embed.FS  // 新增：递归嵌入整个 skills 目录树
```

关键：`//go:embed templates/skills`（目录）必须绑定 `embed.FS`，不能绑定 `string`。

### core-skill 文件（注入阶段用）

```
internal/prompt/templates/skills/
├── sense.md                   # SENSE 框架 4 维度
├── tc.md                      # 测试用例四要素
├── tdd.md                     # RED→GREEN→REFACTOR 铁律
├── testing.md                 # 测试执行规范
├── super-debugging-zh.md      # super-debugging 系统化调试流程（中文）
├── gen-skill.md               # 从 act-path 生成 skill 格式
├── evolve-skills.md           # 进化决策逻辑
├── tdd/
│   └── testing-anti-patterns.md  # 禁止伪测试的反模式
├── refactor-rfc.md            # 重构调查 RFC 模板（dream 阶段用）
└── source-context-consistency.md  # 源码与上下文一致性检查（dream 阶段用）
```

所有 skill 文件 description 遵循 CSO 规则：`"Use when..."` 开头，仅写触发条件，不超过 200 词。

### 精准注入映射

| SOP 阶段 | 注入 skills | 原因 |
|---------|------------|------|
| `plan` | sense、tc | 需求分析 + 测试用例设计 |
| `doing`（coding agent） | tdd、tdd/testing-anti-patterns、super-debugging-zh | 实现阶段强制 TDD + 调试规范 |
| `doing`（testing agent） | tdd、testing、tc | 测试生成 + RED 验证 |
| `learning` | gen-skill | 从 act-path 提取技能 |
| `dream` | sense、evolve-skills | 反思 + skills 进化决策 |

### LoadCoreSkills 函数

```go
func LoadCoreSkills(names []string) string {
    // 遍历 names，从 skillsFS 读取
    // 特例：tdd/testing-anti-patterns → templates/skills/tdd/testing-anti-patterns.md
    // 拼接用 "\n\n---\n\n" 分隔
    // 文件不存在时 log.Printf warn，跳过，不 panic
}
```

## 如何控制/使用

1. **添加新 skill**: 在 `templates/skills/` 下创建 `.md` 文件，`skillsFS` 自动包含
2. **注入到新阶段**: 在对应 prompt 构建函数末尾调用 `LoadCoreSkills([]string{"skill-name"})`
3. **验证注入**: `python3 .rick/tools/check_prompt_variables.py --phase <phase> --keywords "skill:<name>"`
4. **验证无污染**: `python3 .rick/tools/check_prompt_variables.py --phase <other-phase> --keywords "<skill-content>"` 应返回未找到

## 示例

```bash
# 验证 doing 阶段包含 super-debugging skill
python3 .rick/tools/check_prompt_variables.py --phase doing --keywords "super-debugging"
# {"pass": true, "errors": []}

# 验证 doing 阶段不含 gen-skill（无污染）
python3 .rick/tools/check_prompt_variables.py --phase doing --keywords "gen-skill"
# {"pass": false, ...}  ← 预期失败，说明未污染

# 查看编译进二进制的 skill 文件
go run . tools --help
```
