# 依赖关系
task5, task8

# 任务名称
让 pibuilder 产出单文件内聚的 pi 定制化规范产物

# 任务目标
按 spec（task2）落地 KR3.1：pibuilder 产出 pi 定制化规范产物——pi agent 运行时所需的所有提示词内聚在单文件内，被 pi 以标准规范化定制开发语言引用使用。将「主 prompt 文件 + 多个 skill 文件散落（按路径引用）」改为「单文件内聚」：每个 cmd 的 pibuilder 产物把 skill/loop 内容内联进单个 .md 产物（而非散落文件 + 路径引用），可被 pi 直接加载执行。

本 task 只做结构内聚（产出形态），**不改模板触发语言内容**（task11）。--dry-run 输出完整单文件内容。

参考：RFC §4.4「上下文熵减」、§6 KR3.1「内聚在单文件内，被 pi 以标准规范化定制开发语言引用使用」；skill `verify_go_changes_skill`、`template_injection_skill`。

# 关键结果
1. pibuilder 的 BuildPlan/BuildDoing/BuildEasy/BuildHumanLoop 产出的主产物把关键 skill/loop 内容内联（内聚单文件），散落 skill 文件引用下降（关键 skill 内联而非仅路径）；**内聚实现 = 运行时经 prompt 包的导出函数 `ReadEmbeddedSkill(name)`（封装 `skillsFS.ReadFile`）读取（缺失返回 error），而非硬编码 Go 常量**——builder 是独立包、`skillsFS` 未导出，必须经导出函数跨包访问；**内聚范围 = 模板/skill 内容，job 级动态内容（task.md/debug/OKR）仍按 task5 走路径注入，pi 自行 read**；**「单文件内聚」指 method（system prompt）这一份自包含（关键 loop/skill 内容内联进 method），执行期按需技能（grilling/tdd 等）仍走 pi skills 机制（task5），二者不冲突**
2. --dry-run 输出完整单文件内容（`plan/doing/easy --dry-run` 输出自包含、无 `{{` 未替换）
3. 产物可被 pi 加载：单文件含 pi 可识别的结构化引用（agent frontmatter / skill 内联段），不依赖散落文件路径拼装
4. `go build` + `go test ./internal/builder/... ./internal/prompt/... -v` 全绿；dry-run 输出 `grep -c '{{'` = 0

# 测试方法
（本 task 测试用例设计遵循 skill:tdd + skill:testing-anti-patterns：先写失败测试；测试真实产物内容不 mock；dry-run 断言先定位 section。）

1. 正常路径：前置条件 = task5 完成；输入 = `plan --dry-run`；操作 = `./bin/rick plan --dry-run | wc -l` + `./bin/rick plan --dry-run | grep -c '{{'`；预期 = 输出为单个连贯 prompt（行数 > 0），`{{` 计数 0。
2. 边界（单文件内聚）：前置条件 = build 成功；输入 = `plan --dry-run`；操作 = `./bin/rick plan --dry-run | grep -cE 'Grilling|结构化追问'`；预期 = ≥1（grilling skill 内容内联进主产物，而非仅 `skill_grilling.md` 路径引用）。
3. 异常（缺 skill 内联源）：前置条件 = 删除源文件 `internal/prompt/templates/skills/grilling.md` 后重新 `go build`（templates 由 go:embed 编译期内嵌，运行时改名不影响二进制，必须重建）；输入 = 运行 `PIBuilder.BuildPlan`；操作 = 调用检查返回；预期 = 返回 error 且消息含 grilling（非 panic、非静默产出空内容）。
