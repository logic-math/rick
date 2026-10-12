# 依赖关系

task1

# 任务名称

升级 think agent 模板：每个 SENSE 阶段结束时捕获关键判断到 judgment.md，Perspective 阶段写入 draft/loops.md

# 任务目标

修改 `internal/prompt/templates/human_loop_think.md`，增加"判断记录协议"章节：在每个 SENSE 阶段的推进条件满足后，提取 1-3 条关键判断（保留原话）写入 `{{draft_dir}}/human-learning/judgment.md`；在 Perspective 阶段识别出值得展开的概念节点时，将建议写入 `{{draft_dir}}/loops.md`（四字段结构）。

# 关键结果

1. `human_loop_think.md` 新增"## 判断记录协议"章节，明确：触发时机（每个阶段推进条件满足后）、捕获数量（1-3 条）、格式（## [阶段名] 关键判断 + 原话）、写入路径（`{{draft_dir}}/human-learning/judgment.md`）
2. `human_loop_think.md` 在 Perspective 阶段执行步骤后新增"概念展开标记"说明：当识别到值得深挖的概念节点时，将该建议追加到 `{{draft_dir}}/loops.md`，使用以下固定 Markdown 格式：
   ```markdown
   ## [概念节点名称]
   - 做什么: [具体要探索的内容]
   - 难度感受: [预估：容易/中等/困难]
   - 前置依赖: [需要先掌握的概念，无则填"无"]
   - 掌握程度: [当前状态：未接触/了解/熟悉/掌握]
   ```
3. `draft/loops.md` 格式规范在 task2 中统一定义，task3（express）写入 loops.md 时必须遵循同一格式
3. `{{draft_dir}}` 在模板中被正确引用（由 task1 的 Go 代码注入真实路径）
4. 新增单元测试验证模板内容

# 测试方法

**前提：使用 skill:tdd，先写失败测试，再修改模板，再看绿。**

### 测试 1：think 模板包含判断记录协议章节

- 前置条件：内嵌模板已加载
- 输入：`pm.LoadTemplate("human_loop_think")`
- 操作序列：检查返回内容
- 预期输出：内容包含 `judgment.md`、`{{draft_dir}}`、`判断记录协议` 等关键字

```go
// 验证关键词存在
assert strings.Contains(content, "judgment.md")
assert strings.Contains(content, "{{draft_dir}}")
assert strings.Contains(content, "判断记录协议") || strings.Contains(content, "关键判断")
```

```bash
go test ./internal/prompt/... -run TestThinkTemplateContainsJudgmentCapture -v
```

### 测试 2：think 模板包含 loops.md 概念展开标记说明

- 前置条件：内嵌模板已加载
- 输入：`pm.LoadTemplate("human_loop_think")`
- 操作序列：检查返回内容
- 预期输出：内容包含 `loops.md`、`做什么`（或 `难度感受`）、`前置依赖`、`掌握程度`

```bash
go test ./internal/prompt/... -run TestThinkTemplateContainsLoopsMarkup -v
```

### 测试 3：构建后 think 文件内容不含未替换变量

- 前置条件：`draftDir = "/tmp/test-draft"`，通过 task1 实现的注入机制
- 输入：调用 `GenerateHumanLoopPromptFile("topic", rfcDir, "/tmp/test-draft", pm)`
- 操作序列：读取生成的 think 文件内容
- 预期输出：文件内容包含 `/tmp/test-draft`，不包含字符串 `{{draft_dir}}`

```bash
go test ./internal/prompt/... -run TestThinkFileNoUnresolvedVars -v
```

# 参考 Loops/Skills

- `tdd-red-green-refactor-loop`：先写模板内容测试（RED），再修改模板（GREEN）
- `.rick/skills/verify_go_changes_skill/skill.md`：模板改后重新 build（embed.FS），确认测试通过
- `.rick/skills/mark_task_success_skill/skill.md`：task 完成后标记 success

### 边界测试：Perspective 阶段未识别到展开节点时不写入 loops.md

- 此为模板行为规范（"如果识别到"），不需要额外 Go 测试，通过模板语言描述清晰即可
