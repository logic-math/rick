# 依赖关系
task3

# 任务名称

迁移 doing prompt builder + doing.md 模板：移除 SPEC/OKR，注入 loops_context

# 任务目标

修改 `internal/prompt/doing_prompt.go` 和 `internal/prompt/templates/doing.md`：
1. 移除 `builder.SetVariable("spec_content", ...)` 和 `builder.SetVariable("job_okr_content", ...)` 两处注入（对应 `contextMgr.GetSPECRaw()` 和 `contextMgr.GetOKRRaw()` 调用）
2. 调用 `LoadLoopsContext(loopsDir)` 添加 `builder.SetVariable("loops_context", ...)` 注入，loopsDir 为 `{rickDir}/loops/`
3. 在 `doing.md` 模板中删除 `{{spec_content}}` 和 `{{job_okr_content}}` 变量，添加 `{{loops_context}}` 变量
4. 更新相关单元测试

关键代码路径：
- `internal/prompt/doing_prompt.go`：`GenerateDoingPrompt()` 函数，约 52-64 行（GetOKRRaw/GetSPECRaw 调用处）
- `internal/prompt/templates/doing.md`：模板文件
- `internal/prompt/doing_prompt_test.go`：相关测试

# 关键结果

1. `doing_prompt.go` 中不再有 `GetOKRRaw()` 和 `GetSPECRaw()` 调用，新增 `LoadLoopsContext()` 调用
2. `doing.md` 模板中不含 `{{spec_content}}` 和 `{{job_okr_content}}`，新增 `{{loops_context}}`；**修改范围仅限删除旧变量 + 添加 loops_context，不得添加其他变量占位符**（`{{loop_protocol_path}}` 由 task9 负责追加，task4 不触碰）
3. `loopsDir` 通过 `workspace.GetRickDir()` 在函数内部自动获取（不改变 `GenerateDoingPrompt()` 对外签名，避免破坏调用方）；实现前先执行 `grep -r "GenerateDoingPrompt" --include="*.go" .` 确认所有调用方，若签名确需变更则同步更新所有调用方
4. `go test ./internal/prompt/... -run TestDoing` 全部通过
5. `./bin/rick doing --job job_22 --dry-run` 输出包含 "可用的项目 Loops"，不包含 "SPEC" 或 "Job OKR"

# 测试方法

1. **正常路径 - dry-run 验证新变量**：
   - 前置条件：job_22 tasks.json 存在且有 pending task；`.rick/loops/` 目录存在（task1 已完成）；二进制已重新构建
   - 操作：`./bin/rick doing --job job_22 --dry-run 2>&1`
   - 预期输出：包含 "可用的项目 Loops"（loops_context 注入成功），不包含 "{{spec_content}}" 或 "{{job_okr_content}}" 字面量，不包含 "SPEC.md" 路径

2. **单元测试 - spec/okr 变量不在输出中**：
   - 前置条件：在 `doing_prompt_test.go` 中更新测试
   - 操作：`go test ./internal/prompt/... -run TestDoingPrompt -v`
   - 预期输出：测试中断言 generated prompt 不包含 "{{spec_content}}" 或 "{{job_okr_content}}"，且包含 "loops_context" 相关内容

3. **边界用例 - loops 目录为空时 fallback**：
   - 前置条件：`.rick/loops/` 目录存在但无 *.md 文件
   - 操作：`./bin/rick doing --job job_22 --dry-run 2>&1 | grep "暂无项目 Loop"`
   - 预期输出：包含 "暂无项目 Loop 记录"（fallback 文本正确显示）

4. **边界用例 - .rick/loops/ 目录不存在时不 panic**：
   - 前置条件：临时移除 `.rick/loops/` 目录（或使用不存在的路径测试 LoadLoopsContext）
   - 操作：`./bin/rick doing --job job_22 --dry-run`
   - 预期输出：命令正常运行，输出包含 "暂无项目 Loop 记录"，exit code 0

5. **编译验证**：
   - 操作：`./scripts/build.sh && echo "build ok"`
   - 预期输出：构建成功，无编译错误

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill Phase 1-6 执行，每阶段达上限后升级人工协作
