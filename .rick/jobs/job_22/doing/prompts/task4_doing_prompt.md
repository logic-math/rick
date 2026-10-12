# Rick 项目执行阶段提示词

**YOU MUST declare at the start: "I will use skill:tdd for implementation. I will use skill:debug-skill for any unexpected behavior."**

## 核心 Skills（必须加载）

在开始任何工作之前，必须读取以下 skill 文件：

- skill:tdd（测试驱动开发）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_tdd_zh.md`
- skill:testing-anti-patterns（测试反模式）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_testing_anti_patterns_zh.md`
- skill:debug-skill（调试技能）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`
- skill:sense（系统化思维，供 review debug agent 使用）：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_sense.md`

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

## 任务信息

**任务 ID**: task4
**任务名称**: 迁移 doing prompt builder + doing.md 模板：移除 SPEC/OKR，注入 loops_context

### 任务目标
修改 `internal/prompt/doing_prompt.go` 和 `internal/prompt/templates/doing.md`：
1. 移除 `builder.SetVariable("spec_content", ...)` 和 `builder.SetVariable("job_okr_content", ...)` 两处注入（对应 `contextMgr.GetSPECRaw()` 和 `contextMgr.GetOKRRaw()` 调用）
2. 调用 `LoadLoopsContext(loopsDir)` 添加 `builder.SetVariable("loops_context", ...)` 注入，loopsDir 为 `{rickDir}/loops/`
3. 在 `doing.md` 模板中删除 `{{spec_content}}` 和 `{{job_okr_content}}` 变量，添加 `## 可用的项目 Loops

- **candidate-loop-1**：when implementing new features
- **go-tdd-loop**："当需要对 Go 代码进行 TDD 迭代直到测试通过时触发"
` 变量
4. 更新相关单元测试

关键代码路径：
- `internal/prompt/doing_prompt.go`：`GenerateDoingPrompt()` 函数，约 52-64 行（GetOKRRaw/GetSPECRaw 调用处）
- `internal/prompt/templates/doing.md`：模板文件
- `internal/prompt/doing_prompt_test.go`：相关测试

### 关键结果
1. `doing_prompt.go` 中不再有 `GetOKRRaw()` 和 `GetSPECRaw()` 调用，新增 `LoadLoopsContext()` 调用
2. `doing.md` 模板中不含 `{{spec_content}}` 和 `{{job_okr_content}}`，新增 `## 可用的项目 Loops

- **candidate-loop-1**：when implementing new features
- **go-tdd-loop**："当需要对 Go 代码进行 TDD 迭代直到测试通过时触发"
`；**修改范围仅限删除旧变量 + 添加 loops_context，不得添加其他变量占位符**（`{{loop_protocol_path}}` 由 task9 负责追加，task4 不触碰）
3. `loopsDir` 通过 `workspace.GetRickDir()` 在函数内部自动获取（不改变 `GenerateDoingPrompt()` 对外签名，避免破坏调用方）；实现前先执行 `grep -r "GenerateDoingPrompt" --include="*.go" .` 确认所有调用方，若签名确需变更则同步更新所有调用方
4. `go test ./internal/prompt/... -run TestDoing` 全部通过
5. `./bin/rick doing --job job_22 --dry-run` 输出包含 "可用的项目 Loops"，不包含 "SPEC" 或 "Job OKR"


### 测试方法
**正常路径 - dry-run 验证新变量**：
前置条件：job_22 tasks.json 存在且有 pending task；`.rick/loops/` 目录存在（task1 已完成）；二进制已重新构建
操作：`./bin/rick doing --job job_22 --dry-run 2>&1`
预期输出：包含 "可用的项目 Loops"（loops_context 注入成功），不包含 "{{spec_content}}" 或 "{{job_okr_content}}" 字面量，不包含 "SPEC.md" 路径
**单元测试 - spec/okr 变量不在输出中**：
前置条件：在 `doing_prompt_test.go` 中更新测试
操作：`go test ./internal/prompt/... -run TestDoingPrompt -v`
预期输出：测试中断言 generated prompt 不包含 "{{spec_content}}" 或 "{{job_okr_content}}"，且包含 "loops_context" 相关内容
**边界用例 - loops 目录为空时 fallback**：
前置条件：`.rick/loops/` 目录存在但无 *.md 文件
操作：`./bin/rick doing --job job_22 --dry-run 2>&1 | grep "暂无项目 Loop"`
预期输出：包含 "暂无项目 Loop 记录"（fallback 文本正确显示）
**边界用例 - .rick/loops/ 目录不存在时不 panic**：
前置条件：临时移除 `.rick/loops/` 目录（或使用不存在的路径测试 LoadLoopsContext）
操作：`./bin/rick doing --job job_22 --dry-run`
预期输出：命令正常运行，输出包含 "暂无项目 Loop 记录"，exit code 0
**编译验证**：
操作：`./scripts/build.sh && echo "build ok"`
预期输出：构建成功，无编译错误

## 项目背景

**项目名称**: rick
**项目描述**: Context-First AI Coding Framework

## 可用的项目 Loops

- **candidate-loop-1**：when implementing new features
- **go-tdd-loop**："当需要对 Go 代码进行 TDD 迭代直到测试通过时触发"


### 项目架构
Rick 项目采用模块化架构设计：

**核心模块**:
- infrastructure: 基础设施模块（Go 项目初始化、CLI、工作空间、配置、日志）
- parser: 内容解析模块（Markdown、task.md、debug.md、OKR/SPEC 解析）
- dag_executor: DAG 执行模块（DAG 构建、拓扑排序、任务执行、重试机制）
- prompt_manager: 提示词管理模块（模板、构建、上下文、各阶段提示词生成）
- cli_commands: 命令处理模块（init、plan、doing、learning 命令）

**关键设计**:
- 使用 Go 标准库为主，最小化外部依赖
- 提示词管理是核心创新，支持多阶段提示词生成
- 任务执行采用 DAG 拓扑排序，支持并行和串行执行
- 失败重试机制，超过限制后需人工干预

## 执行上下文

### 已完成的任务
暂无已完成的任务

### 任务依赖
该任务依赖以下任务的完成：
- task3


### 问题记录

以下是执行过程中遇到的问题记录，请重点关注避免重复错误：

暂无问题记录

## Cialdini 合规原则

### 权威（Authority）

**YOU MUST follow TDD. No exceptions.**

在开始任何实现之前，必须先编写失败的测试（RED phase）。这是不可协商的工程规范。

#### TDD 铁律（Three Laws）

1. **RED（先红）**: 先运行测试，确认测试失败（证明测试有效）
2. **GREEN（再绿）**: 编写最少代码让测试通过
3. **REFACTOR（再重构）**: 在测试通过的前提下改善代码质量

**不得跳过任何阶段。** 未经 RED 验证直接写实现，视为违反 TDD 铁律。

#### DEBUG 铁律

**所有代码都是 debug 出来的。RED 阶段测试失败 = 遇到 bug，必须触发 debug-skill，无一例外。**

> RED 不是"预期中的失败"，而是发现了系统与预期的差距——这正是 bug 的定义。
> 跳过 debug-skill 直接修改代码 = 随机修复 = 制造下一个 bug。

**触发条件（以下任意一条即触发，不得跳过）**：
- 运行测试出现 FAIL / 错误输出
- 代码行为与预期不符
- 编译报错（编译错误也是 bug）

**触发后必须执行**：
1. 声明 `"I will use skill:debug-skill."`
2. 在 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/debug/` 下创建 `bug{N}-{描述}.md`，**严格按以下格式**（doing_check 逐行校验，格式错误 = check 失败）：

```markdown
---
summary: "一句话描述根因 + 最终状态"
status: "✅ 已解决"
---

## Phase 1: 构建反馈回路

（复现步骤 + 最小化路径，获得秒级可重复运行的测试）

## Phase 2: 复现最小化

（进一步裁剪到最小可复现单元）

## Phase 3: 可证伪假设

（3-5 个有优先级排序的假设列表）

## Phase 4: 插桩观察

（log/gdb/delve/pprof 等插桩，选最合适的插入关键路径）

## Phase 5: 修复回归

（基于确认根因的最小改动修复 + 全量测试通过）

## Phase 6: 清理事后分析

（移除临时桩，提炼防范模式）

## 结论

根因：...  修复：...
```

**格式铁律（doing_check 严格校验）**：
- 文件名：`bug{n}-{描述}.md`（n 为正整数，描述非空）
- 必须包含七个 `##` 二级标题：`## Phase 1: 构建反馈回路`、`## Phase 2: 复现最小化`、`## Phase 3: 可证伪假设`、`## Phase 4: 插桩观察`、`## Phase 5: 修复回归`、`## Phase 6: 清理事后分析`、`## 结论`
- frontmatter 必须有 `status:` 字段，且最终状态不得为 `"🔄 进行中"`

3. 加载 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`，严格按 Phase 1-6 执行（Phase 4 上限 3 次）
4. 不得随机修改代码（no random fixes）

**doing_check 校验 debug/bug*.md 格式，格式不合规 = check 失败 = 任务未完成。**

### 承诺（Commitment）

在开始实现前，声明你将使用的 skills：

```
I will use skill:tdd for implementation.
I will use skill:debug-skill for any unexpected behavior.
```

明确的承诺能提升 skill 合规率，防止任务执行过程中遗忘关键工程实践。

### 稀缺（Scarcity）

**Before proceeding to next task, verify: all tests pass.**

**Immediately after test failure: 声明 "I will use skill:debug-skill."，在 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/debug/` 创建 `bug{N}-{描述}.md`，按 Phase 1: 构建反馈回路 → Phase 2: 复现最小化 → Phase 3: 可证伪假设 → Phase 4: 插桩观察 → Phase 5: 修复回归 → Phase 6: 清理事后分析 顺序调试，不可跳过。**

每次推进都有且仅有一次机会通过检查。未通过则必须先修复，不可跳过。

---

## 做事方法

1. **理解需求**: 仔细阅读任务目标和关键结果
2. **设计方案**: 根据项目架构和现有代码，设计实现方案
3. **实现代码**: 实现所有必要的功能
4. **测试验证**: 按照测试方法验证功能的正确性
5. **提交代码**: 使用 git 提交代码，提交信息应该清晰明确


## 具体步骤

请按照以下步骤执行任务：

1. **分析**: 基于目标和关键结果彻底分析既有事实现状
2. **设计**: 针对目标和关键结果规划实现方案
3. **实现**: 完全具体实现工作
4. **测试**: 根据测试方法对交付的结果进行测试,代码必须能在生产环境正确工作
5. **提交**: 使用 git 将这次任务变更进行提交,务必遵循项目规范进行提交

## 行为约束

1. **测试通过**: 确保所有测试都通过后才能提交代码
2. **bug 强制记录**: 每次测试失败，必须在 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/debug/bug{N}-{描述}.md` 创建调试记录，不可跳过
3. **生产就绪**: 代码应该能够在生产环境中正确运行
3. **优先使用 tools**: 如果项目根目录存在 `tools/` 目录，优先使用其中的 Python 工具脚本完成任务（tools 列表会在 prompt 末尾动态注入）
4. **强制 doing check**: 在 git commit 之后，**必须**运行以下命令验证产出：
   ```bash
   /Users/sunquan/ai_coding/CODING/rick/bin/rick tools doing_check job_N
   ```
   如果 check 失败，根据错误信息修复（如解决 zombie 任务等），修复后重新运行，循环直到 check 通过。**check 通过后才算任务完成**，不可跳过。
