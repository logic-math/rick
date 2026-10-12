# 依赖关系

task1, task3

# 任务名称

升级 learning 六步 SOP，注入 gen-skill，生成 run_log 度量文件

# 任务目标

将 `internal/prompt/templates/learning.md` 升级为 RFC 六步 SOP，在 Step 2 显式引用 `skill:gen-skill`，并更新 `learning.go` 注入 act-path 内容、写入 `.rick/dream/run_log_{n}.md`。

依赖 task1（act-path 文件路径约定 `doing/tasks/{taskID}/act-path.md`）和 task3（gen-skill 注入）。

**learning SOP 七步映射（严格对应 RFC）：**

| 步骤 | 内容 | skill |
|------|------|-------|
| Step 0 | 加载 OKR/SPEC/debug/act-path | 无 |
| Step 1 | 读取 act-path，还原完整执行轨迹 | 无 |
| Step 2 | **评估更合理 act-path**：能否用更短路径（更少工具调用/更低报错）完成同样目标，产出改进建议 | 无 |
| Step 3 | 沉淀 skills → gen-skill 格式（触发场景/预期效果/核心内容） | **skill:gen-skill** |
| Step 4 | 识别 tools 候选（可复用 + 纯函数 + 清晰 I/O） | 无 |
| Step 5 | 更新 SPEC.md skills 列表 | 无 |
| Step 6 | 写入 `.rick/dream/run_log_{n}.md` | 无 |

**Step 2 是 RFC 明确要求的独立步骤**，不能与 Step 1 合并——评估"更优轨迹"是 learning 产生优化信号的核心，必须显式触发。

# 关键结果

1. **`learning.md` 重构为七步（严格对应 RFC）**：
   - Step 0：加载 `{{act_path_content}}`（由 learning.go 注入）
   - Step 1：读取 act-path，还原完整执行轨迹
   - Step 2：评估更合理 act-path——`Analyze: Could this task have been completed with fewer tool calls or fewer errors? Output: [Better Path Proposal]`
   - Step 3 开头：`YOU MUST declare: "I will use skill:gen-skill." Before writing any skill proposal.`，按格式输出（触发场景/预期效果/核心内容）
   - Step 4：识别可转化为 py 工具的 skill（可复用 + 纯函数 + 清晰 I/O）
   - Step 5：更新 SPEC.md skills 列表
   - Step 6：写入 `.rick/dream/run_log_{n}.md`，格式：`| Job | 模型 | 错误次数 | 工具调用轮次 | 备注 |`
   - 仅含 `gen-skill`（不含 tdd/debug/sense 等）

2. **`learning.go` 新增 `collectActPathContent(doingDir string) string`**：
   - 遍历 `doingDir/tasks/*/act-path.md`（使用 `filepath.Glob`）
   - 拼接全部内容，用 `\n\n---\n\n` 分隔
   - 注入模板变量 `{{act_path_content}}`

3. **`.rick/dream/run_log_{n}.md` 生成**：
   - n = 当前 `.rick/dream/` 目录下 `run_log_*.md` 文件数量 + 1
   - 由 learning agent 在 Step 6 写入（不由 Go 代码生成，由 prompt 指示 agent 操作）

# 测试方法

1. 编译：`python3 tools/build_and_get_rick_bin.py`
2. **`collectActPathContent` 单元测试**（本 task 新增于 `learning_test.go` 或 `cmd_test.go`）：
   - 在 tmpDir 下创建 `doing/tasks/task1/act-path.md`（内容 "# act-path task1"）和 `doing/tasks/task2/act-path.md`（内容 "# act-path task2"）
   - 调用 `collectActPathContent(doingDir)`
   - 断言返回值同时包含 "act-path task1" 和 "act-path task2"，用 "---" 分隔
   ```
   go test ./internal/cmd/... -v -run TestCollectActPathContent
   ```
3. **KR2 learning 提示词验证**（OKR KR2）：
   ```bash
   python3 tools/check_prompt_variables.py --phase learning --keywords "skill:gen-skill"    # gen-skill 承诺
   python3 tools/check_prompt_variables.py --phase learning --keywords "run_log"             # run_log 写入指令
   python3 tools/check_prompt_variables.py --phase learning --keywords "act_path_content"    # act-path 变量
   python3 tools/check_prompt_variables.py --phase learning --keywords "Better Path Proposal" # Step2 RFC要求的独立评估步骤
   ```
4. 无污染：`python3 tools/check_prompt_variables.py --phase learning --keywords "skill:tdd"` → "关键词未找到"
5. workspace 目录：`go test ./internal/workspace/... -v -run TestDreamDir`
6. 完整测试：`go test ./...`，无新增失败
