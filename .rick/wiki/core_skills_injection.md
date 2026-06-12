# core-skills 精准注入机制

## 概述

Rick 将核心 skill 文件通过 `embed.FS` 编译进二进制，在不同 SOP 阶段精准注入对应 skill，避免信息污染（如 doing 阶段不注入 gen-skill，plan 阶段不注入 debug-skill）。

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
├── sense.md                       # SENSE 框架 4 维度
├── tc.md                          # ⚠️ 待合并：内容（测试用例四要素）合并进 tdd-zh.md 后删除（见 RFC-refactor-2 §2.1）
├── tdd.md                         # ⚠️ 死代码：英文版已被 tdd-zh.md 替代，待删除（见 RFC-refactor-2）
├── tdd-zh.md                      # TDD 铁律（中文版，plan/doing/easy 注入）
├── testing.md                     # 测试执行规范（暂未注入）
├── testing-anti-patterns-zh.md    # 禁止伪测试反模式（plan/doing 注入）
├── debug_skill.md                 # 三阶段科学调试 SOP + review debug agent（doing/easy 注入）
├── write_spec.md                  # SPEC 撰写规范（plan 注入）
├── gen-skill.md                   # 从 act-path 生成 skill 格式（easy learning 注入）
├── evolve-skills.md               # 进化决策逻辑（dream 注入）
├── tdd/
│   └── testing-anti-patterns.md  # ⚠️ 死代码：英文版已被 testing-anti-patterns-zh.md 替代，待删除（见 RFC-refactor-2）
├── refactor-rfc.md                # 重构调查 RFC 模板（dream 注入）
└── source-context-consistency.md  # 源码与上下文一致性检查（dream 注入）
```

所有 skill 文件 description 遵循 CSO 规则：`"Use when..."` 开头，仅写触发条件，不超过 200 词。

### 精准注入映射

| SOP 阶段 | 注入 skills | 原因 |
|---------|------------|------|
| `plan` | sense、write_spec、tdd-zh、testing-anti-patterns-zh | 需求分析 + SPEC 撰写 + TDD 规范 |
| `doing` | tdd-zh、testing-anti-patterns-zh、sense、debug_skill | TDD 铁律 + 三阶段科学调试 |
| `easy`（会话） | tdd-zh、sense、debug_skill | 交互式会话的 TDD + 调试支持 |
| `easy`（learning） | gen-skill | 会话结束后从 act-path 提取技能 |
| `dream` | sense、evolve-skills、source-context-consistency、refactor-rfc | 反思 + 进化决策 + 一致性检查 + 重构 RFC |

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
# 验证 doing 阶段包含 debug-skill
python3 .rick/tools/check_prompt_variables.py --phase doing --keywords "debug-skill"
# {"pass": true, "errors": []}

# 验证 doing 阶段不含 gen-skill（无污染）
python3 .rick/tools/check_prompt_variables.py --phase doing --keywords "gen-skill"
# {"pass": false, ...}  ← 预期失败，说明未污染

# 查看编译进二进制的 skill 文件
go run . tools --help
```
