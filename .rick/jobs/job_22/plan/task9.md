# 依赖关系
task1, task4, task7

# 任务名称

新建 loop_protocol.md 模板，通过 embed.FS 内嵌并注入 doing/easy prompt

# 任务目标

将 Loop 执行协议提取为独立模板文件 `internal/prompt/templates/skills/loop_protocol.md`，通过 embed.FS 内嵌进二进制，在 doing 和 easy 的 prompt builder 中用 `WriteSkillFile` 写出到磁盘，路径注入为 `{{loop_protocol_path}}`。doing.md 和 easy.md 只引用该路径，不内联协议内容，变更点统一维护。

实现模式与 `debug_skill.md` + `{{debug_skill_path}}` 完全一致：

```
loop_protocol.md（embed.FS）
    │
    ├── doing_prompt.go → WriteSkillFile(promptsDir, "loop_protocol.md", "loop_protocol")
    │       → builder.SetVariable("loop_protocol_path", path)
    │
    └── easy_prompt.go → 同上
```

---

## 必须创建的文件及其内容

### 文件：`internal/prompt/templates/skills/loop_protocol.md`

**完整内容**（逐字写入）：

```markdown
---
name: loop-protocol
description: doing/easy 阶段父 agent 执行项目 Loop 的迭代控制协议
---

# Loop 执行协议

当执行任务步骤时，如遇到以下任一情形，检查可用项目 Loops 列表中是否有匹配的 loop：
- 需要反复尝试直到达成可量化目标
- 任务描述中包含 loop 触发词
- 当前操作属于"迭代收敛"类工作（如：测试从失败到通过、指标从不达标到达标）

**分工铁律（不得违反）**：
- subagent 职责：按 loop.md 工作流执行一轮操作后退出，**不评估、不决策继续与否**
- 父 agent 职责：读取 loop.md、管理迭代状态、运行评估命令、决定继续或停止

---

## Step 1：加载 Loop 定义

读取 `.rick/loops/[name].md`，获取五要素：目标（Goal）、上下文管理、可调用工具、产出评估、停止标准。

## Step 2：执行一次迭代（subagent）

以 subagent 方式执行一次迭代，向 subagent 传入：

```
## 当前 Loop：[loop-name]（第 N 轮迭代）

[loop.md 完整内容]

## 当前任务上下文
[task 描述 + 关键约束]

## 上轮迭代保留上下文（第 1 轮此项为空）
[按 loop.md 上下文管理策略压缩后的上轮结果]
```

subagent 完成后退出，不评估，不决策。

## Step 3：父 agent 执行评估

运行 loop.md `## 产出评估` 章节中的评估命令，获取本轮客观结果。

## Step 4：父 agent 判断下一步

```
IF 满足 loop.md `## 目标` 的成功标准：
    → 成功退出 loop，继续 task 后续步骤

ELSE IF 本轮相比上轮有实质进展（按产出评估的进展判断标准）：
    → 按 loop.md `## 上下文管理` 策略压缩本轮上下文
    → 回到 Step 2，启动第 N+1 轮 subagent

ELSE（无进展 or 达到停止标准的最大迭代次数）：
    → 按 loop.md `## 停止标准` 的优雅退出操作执行
    → 记录当前状态，退出 loop，等待人工介入
```

父 agent 自行维护迭代轮次 N（从 1 开始），每次 Step 2 前 +1。
```

---

## doing.md 和 easy.md 中的引用方式

在两个模板的任务执行主流程之前，添加以下引用（`{{loop_protocol_path}}` 由 builder 注入）：

```markdown
## Loop 执行协议

如当前任务步骤匹配某个项目 Loop，必须先读取并遵循以下协议文件：

`{{loop_protocol_path}}`
```

---

## Go 代码变更

### `doing_prompt.go`（`GenerateDoingPromptFile` 中，与其他 WriteSkillFile 并列）

```go
loopProtocolFile, err := manager.WriteSkillFile(promptsDir, "loop_protocol.md", "loop_protocol")
if err != nil {
    return "", fmt.Errorf("failed to write loop protocol skill: %w", err)
}
builder.SetVariable("loop_protocol_path", loopProtocolFile)
```

**⚠️ doing dry-run 分支**：检查 `doing_prompt.go` 或 `doing.go` 中是否有独立的 dry-run 路径（如 `GenerateDoingPrompt` 或 `runDoingDryRun`）。若存在且该分支不调用 `WriteSkillFile`，则必须在该分支中补加：
```go
builder.SetVariable("loop_protocol_path", filepath.Join(promptsDir, "loop_protocol.md"))
```
否则 `--dry-run` 输出会包含未替换的 `{{loop_protocol_path}}` 字面量，违反 dry-run 规范。

### `easy_prompt.go`（`GenerateEasyPromptFile` 中，与其他 WriteSkillFile 并列）

```go
loopProtocolFile, err := manager.WriteSkillFile(promptsDir, "loop_protocol.md", "loop_protocol")
if err != nil {
    return "", fmt.Errorf("failed to write loop protocol skill: %w", err)
}
builder.SetVariable("loop_protocol_path", loopProtocolFile)
skillFiles = append(skillFiles, loopProtocolFile)  // ⚠️ 必须追加到 skillFiles，否则调用方文件列表不完整
```

**⚠️ easy dry-run 分支**：检查 `easy_prompt.go` 中 `GenerateEasyPrompt`（dry-run 路径）是否已为所有 skill 设置占位路径。必须补加：
```go
builder.SetVariable("loop_protocol_path", filepath.Join(promptsDir, "skill_loop_protocol.md"))
```

### 模板变量注册

- `doing.md`：新增 `{{loop_protocol_path}}` 变量
- `easy.md`：新增 `{{loop_protocol_path}}` 变量

---

## 实现约束

- `loop_protocol.md` 放在 `internal/prompt/templates/skills/` 下，与 `debug_skill.md`、`tdd-zh.md` 同目录，embed.FS 自动包含
- 不使用 `WriteSkillFileWithVars`（协议内容无需变量替换）
- `WriteSkillFile` 的第三个参数（skillName）为 `"loop_protocol"`，对应文件路径 `templates/skills/loop_protocol.md`
- 参考 `manager.go` 中 `WriteSkillFile` 的实现确认路径映射规则

# 关键结果

1. `internal/prompt/templates/skills/loop_protocol.md` 存在，frontmatter 含 name/description，正文含 Step 1-4 和分工铁律
2. `doing_prompt.go` 调用 `WriteSkillFile` 写出 loop_protocol.md，并 `SetVariable("loop_protocol_path", ...)`
3. `easy_prompt.go` 同上
4. `doing.md` 模板含 `{{loop_protocol_path}}` 变量，不内联协议正文
5. `easy.md` 模板含 `{{loop_protocol_path}}` 变量，不内联协议正文
6. `go test ./internal/prompt/... -run "TestLoadEmbeddedTemplate|TestCoreSkillsEmbed" -v` 通过（`-run TestEmbedded` 是前缀匹配，实际函数名为 `TestLoadEmbeddedTemplate` 等，确认 loop_protocol.md 可被 embed.FS 读取）
7. `./bin/rick doing --job job_22 --dry-run 2>&1 | grep "loop_protocol"` 有输出且不含 `{{loop_protocol_path}}` 字面量
8. `./scripts/build.sh` 成功

# 测试方法

1. **embed.FS 可加载验证**：
   - 操作：`go test ./internal/prompt/... -run TestEmbedded -v 2>&1 | grep -i "loop_protocol\|PASS\|FAIL"`
   - 预期输出：包含 PASS，无 FAIL

2. **doing dry-run 路径注入验证**：
   - 前置条件：`./scripts/build.sh` 已执行；job_22 plan 目录存在即可（dry-run 不需要 tasks.json）
   - 操作：`./bin/rick doing --job job_22 --dry-run 2>&1 | grep "loop_protocol"`
   - 预期输出：路径字符串（如 `.../prompts/loop_protocol.md`），不含 `{{loop_protocol_path}}` 字面量

3. **easy 路径注入验证**：
   - 操作：在 `easy_prompt_test.go` 中新建 `TestGenerateEasyPromptFile_LoopProtocolInjected`，参照现有 `TestGenerateEasyPromptFile_GrillingSkillInjected` 的结构，断言生成的 prompt 内容包含 `loop_protocol.md` 路径，不含 `{{loop_protocol_path}}` 字面量
   - 执行：`go test ./internal/prompt/... -run TestGenerateEasyPromptFile_LoopProtocolInjected -v`
   - 预期输出：PASS

4. **doing.md 变量双向验证**：
   - 操作（正向）：`grep -c "{{loop_protocol_path}}" internal/prompt/templates/doing.md`
   - 预期输出：1（变量存在）
   - 操作（负向）：`grep -c "Step 1：加载 Loop\|Step 2：执行一次迭代" internal/prompt/templates/doing.md`
   - 预期输出：0（协议正文不内联）
   - 操作（回归）：`grep -c "{{loops_context}}" internal/prompt/templates/doing.md`
   - 预期输出：≥1（task4 的产出未被覆盖）

5. **easy.md 变量双向验证**：
   - 操作（正向）：`grep -c "{{loop_protocol_path}}" internal/prompt/templates/easy.md`
   - 预期输出：1
   - 操作（负向）：`grep -c "Step 1：加载 Loop\|Step 2：执行一次迭代" internal/prompt/templates/easy.md`
   - 预期输出：0
   - 操作（回归）：`grep -c "{{loops_context}}" internal/prompt/templates/easy.md`
   - 预期输出：≥1（task7 的产出未被覆盖）

6. **单一变更点验证**：
   - 操作：`grep -r "Step 1：加载 Loop" internal/prompt/templates/`
   - 预期输出：只有 `internal/prompt/templates/skills/loop_protocol.md` 一个文件（协议只在一处维护）

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill Phase 1-6 执行，每阶段达上限后升级人工协作
