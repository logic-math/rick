# 依赖关系

task1

# 任务名称

升级 express agent 模板：添加 judgment.md review 清洗步骤和 ZPD 显式评价引导

# 任务目标

修改 `internal/prompt/templates/human_loop_express.md`，在现有 Step 1 之前增加"第零步：judgment.md review"（读取并清洗 judgment.md，删除无效/混乱条目），在现有 Step 4（文档审核）之后增加"第五步：ZPD 显式评价"（引导 human 显式评价本次 loop 的难度/收获，将结果写入 `{{draft_dir}}/progress.md`，并将下次建议写入 `{{draft_dir}}/loops.md`）。

# 关键结果

1. `human_loop_express.md` 在"第一步：快速确认"之前新增"第零步"：读取 `{{draft_dir}}/human-learning/judgment.md`，删除逻辑混乱/自相矛盾/无实质内容的条目，保留经 human 明确确认的判断（原话），不补充新内容
2. `human_loop_express.md` 在"第四步：保存说明"之后新增"第五步：ZPD 显式评价"：向 human 提问三个方向（① 难度感受——作为迭代方向信号；② 核心收获——作为原创性思考信号，直接推进 ZPD；③ 还缺什么——作为能力边界识别信号，用于下次 loop 重新评估 ZPD），将回答结构化写入 `{{draft_dir}}/progress.md`（追加本次 loop 评价记录），并将下次建议条目追加到 `{{draft_dir}}/loops.md`（遵循 task2 定义的四字段格式）
3. `progress.md` 追加格式（固定）：
   ```markdown
   ## [Loop 主题 / 时间戳]
   ### 难度感受
   [用户回答]
   ### 核心收获（原创性思考信号）
   [用户回答]
   ### 还缺什么（能力边界信号）
   [用户回答]
   ```
4. 第零步判断文件存在性：`{{draft_dir}}/human-learning/judgment.md` 不存在时直接跳过，不报错
5. 两个新步骤均引用 `{{draft_dir}}` 变量（由 task1 注入真实路径）
4. 新增单元测试验证模板内容

# 测试方法

**前提：使用 skill:tdd，先写失败测试，再修改模板，再看绿。**

### 测试 1：express 模板包含 judgment.md review 步骤

- 前置条件：内嵌模板已加载
- 输入：`pm.LoadTemplate("human_loop_express")`
- 操作序列：检查返回内容
- 预期输出：包含 `judgment.md`、`清洗` 或 `review`、`{{draft_dir}}`

```bash
go test ./internal/prompt/... -run TestExpressTemplateContainsJudgmentReview -v
```

### 测试 2：express 模板包含 ZPD 显式评价步骤

- 前置条件：内嵌模板已加载
- 输入：`pm.LoadTemplate("human_loop_express")`
- 操作序列：检查返回内容
- 预期输出：包含 `progress.md`、`ZPD`（或 `难度感受`）、`loops.md`

```bash
go test ./internal/prompt/... -run TestExpressTemplateContainsZPDEvaluation -v
```

### 测试 3：构建后 express 文件内容不含未替换变量

- 前置条件：`draftDir = "/tmp/test-draft"`
- 输入：调用 `GenerateHumanLoopPromptFile("topic", rfcDir, "/tmp/test-draft", pm)`
- 操作序列：读取生成的 express 文件内容
- 预期输出：文件内容包含 `/tmp/test-draft`，不包含 `{{draft_dir}}`

```bash
go test ./internal/prompt/... -run TestExpressFileNoUnresolvedVars -v
```

# 参考 Loops/Skills

- `tdd-red-green-refactor-loop`：先写模板内容测试（RED），再修改模板（GREEN）
- `.rick/skills/verify_go_changes_skill/skill.md`：模板改后重新 build，确认测试通过
- `.rick/skills/mark_task_success_skill/skill.md`：task 完成后标记 success

### 边界测试：judgment.md 不存在时第零步不 panic

- 此为模板行为规范，描述"如 judgment.md 不存在则跳过"；不需要 Go 层面的特殊处理
