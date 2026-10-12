# 依赖关系
task2, task3

# 任务名称
构建验证：build + dry-run 输出检查 + 回归测试

# 任务目标
确认 task1-3 所有改动编译通过，plan/easy 的 dry-run 或生成文件正确包含 grilling skill 路径（无未替换占位符），现有 go test 不引入新的 FAIL，plan_check 通过。

注意：测试脚本必须先调用 `.rick/tools/build_and_get_rick_bin.py` 构建本地 `./bin/rick`，不得使用系统安装版（系统版不含本 job 新代码）。

# 重要约束
- `./scripts/build.sh` 是所有验证的前置步骤：grilling.md 通过 embed.FS 在编译时包含，不重新 build 则 LoadCoreSkills("grilling") 返回空字符串
- 检查现有 test 是否因 sense_skill_path 删除而失败（如 `TestGeneratePlanPrompt_NoUnreplacedVars`），需要同步修复受影响的测试用例

# 关键结果
1. `./scripts/build.sh` 构建成功，产出 `./bin/rick`
2. `./bin/rick plan --dry-run` 输出包含 `skill_grilling` 字样，不含 `sense_skill_path` 字样，不含 `{{grilling_skill_path}}` 未替换字符串
3. `go test ./internal/prompt/... -run .` 全部通过（含新增的 TestLoadCoreSkills_Grilling）
4. `go test ./internal/cmd/...` 全部通过（现有命令测试不回归）
5. `./bin/rick tools plan_check job_18` 退出码 0

# 测试方法
1. 正常路径（完整流程验证）：
   - 前置条件：task1/task2/task3 完成，`./scripts/build.sh` 已执行
   - 操作序列：
     ```bash
     ./scripts/build.sh
     # plan dry-run：含 grilling，无 sense，无未替换占位符
     ./bin/rick plan --dry-run 2>&1 | grep "skill_grilling"
     ./bin/rick plan --dry-run 2>&1 | grep -q "sense_skill_path" && echo FAIL || echo OK
     ./bin/rick plan --dry-run 2>&1 | grep -q "{{grilling_skill_path}}" && echo FAIL || echo OK
     # easy 模板文件包含 grilling_skill_path
     grep "grilling_skill_path" internal/prompt/templates/easy.md
     go test ./internal/prompt/... -v -run "TestLoadCoreSkills"
     ./bin/rick tools plan_check job_18
     ```
   - 预期输出：每步退出码 0；plan dry-run 含 grilling 路径；easy.md 含 grilling 变量；plan_check 输出 ✅
2. 边界用例（无未替换变量）：
   - 输入：`./bin/rick plan --dry-run 2>&1 | grep '{{grilling_skill_path}}'`
   - 预期输出：无匹配（退出码 1）
3. 异常路径（回归测试）：
   - 输入：`go test ./internal/prompt/... ./internal/cmd/... ./internal/executor/...`（精确范围，避免全量）
   - 预期输出：0 个 FAIL，新增测试全部 PASS

# 调试方法
遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_18/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill 三阶段调试法（源码推理法→增量调试法→科学实验法）执行，每阶段达上限后升级人工协作
