# 依赖关系
task3

# 任务名称

迁移 plan prompt builder + plan.md 模板：移除 OKR/SPEC/RFC 路径，注入 loops_context

# 任务目标

修改 `internal/prompt/plan_prompt.go` 和 `internal/prompt/templates/plan.md`：
1. 移除 `loadOKRPath()`、`loadSpecPath()`、`loadRFCDir()`、`loadRFCPaths()` 四处注入，对应变量 `{{okr_path}}`、`{{spec_path}}`、`{{rfc_dir}}`、`{{rfc_paths}}`
2. 调用 `LoadLoopsContext(loopsDir)` 添加 `{{loops_context}}` 注入
3. plan.md 模板中的 8 步 SOP 移除"读取 OKR/SPEC/RFC"相关步骤（步骤 1 和 2 中的 OKR/SPEC/RFC 读取指令），改为"读取 loops_context 了解项目已有 loop 模式"
4. plan agent 不再生成 `plan/OKR.md`（模板中移除 OKR.md 生成指令）
5. 更新 `plan_prompt_test.go` 中的相关测试（**先读取 `plan_prompt_test.go` 全文，再修改**，避免遗漏现有断言）
6. 移除 `loadOKRPath`/`loadSpecPath`/`loadRFCDir`/`loadRFCPaths` 调用后，若这些函数无其他引用可直接删除；移除后必须运行 `go build ./...` 确认无 "imported and not used" 编译错误

关键代码路径：
- `internal/prompt/plan_prompt.go`：`GeneratePlanPrompt()` 函数，约 49-60 行（loadOKRPath/loadSpecPath/loadRFCPaths 调用处）
- `internal/prompt/templates/plan.md`：模板文件（输出目录约束中的 OKR.md 生成要求）
- `internal/prompt/plan_prompt_test.go`：相关测试

# 关键结果

1. `plan_prompt.go` 不再调用 `loadOKRPath()`、`loadSpecPath()`、`loadRFCDir()`、`loadRFCPaths()`，新增 `LoadLoopsContext()` 调用
2. `plan.md` 模板不含 `{{okr_path}}`、`{{spec_path}}`、`{{rfc_dir}}`、`{{rfc_paths}}`，新增 `{{loops_context}}`
3. `plan.md` 模板中"必须生成 OKR.md"相关约束已移除
4. `go test ./internal/prompt/... -run TestPlan` 全部通过
5. `./bin/rick plan --dry-run` 输出包含 "可用的项目 Loops"，不包含 "SPEC.md" 路径或 "OKR.md 生成"指令

# 测试方法

1. **正常路径 - dry-run 验证变量替换**：
   - 前置条件：二进制已重新构建，`.rick/loops/` 目录存在（task1 已完成）
   - 操作：`./bin/rick plan --dry-run 2>&1`
   - 预期输出：包含 "可用的项目 Loops"，不包含 "{{okr_path}}" 或 "{{spec_path}}" 字面量，不包含 "必须生成.*OKR.md" 文本

2. **单元测试 - 旧变量不在输出中**：
   - 前置条件：在 `plan_prompt_test.go` 中更新/新增测试
   - 操作：`go test ./internal/prompt/... -run TestPlanPrompt -v`
   - 预期输出：测试断言 generated prompt 不包含 "okr_path" 或 "spec_path"，包含 "loops_context"

3. **格式检查 - plan_check 通过**：
   - 前置条件：task1-3 已完成
   - 操作：`./bin/rick tools plan_check job_22`
   - 预期输出：`✅ plan check passed`（plan_check 不检查 OKR.md，不影响通过）

4. **边界用例 - 旧辅助函数删除不影响编译**：
   - 如果 `loadOKRPath` 等函数在其他地方无引用，可直接删除；否则保留但不调用
   - 操作：`go build ./...`
   - 预期输出：编译无 "declared but not used" 错误

5. **异常路径 - plan 不生成 OKR.md**：
   - 前置条件：删除 job_22/plan/OKR.md（如存在）
   - 操作：确认 `plan.md` 模板正文中无"生成 OKR.md"指令
   - 预期输出：`grep -c "OKR.md" internal/prompt/templates/plan.md` 输出 0

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill Phase 1-6 执行，每阶段达上限后升级人工协作
