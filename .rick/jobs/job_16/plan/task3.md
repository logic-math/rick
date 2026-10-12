# 依赖关系

task2

# 任务名称

更新 Go prompt 文件，将 super-debugging-zh 切换到 debug_skill

# 任务目标

修改 `doing_prompt.go`、`plan_prompt.go`、`easy_prompt.go` 中所有 WriteSkillFile / SetVariable 调用，将 "super-debugging-zh" 文件名和对应模板变量名全部切换为 "debug_skill"/"debug_skill_path"；同步更新 `manager_test.go` 相关测试用例，确保 `go test ./internal/prompt/...` 全部通过。

# 关键结果

1. `internal/prompt/doing_prompt.go`：
   - `WriteSkillFile(promptsDir, "skill_super_debugging_zh.md", "super-debugging-zh")` → `WriteSkillFile(promptsDir, "skill_debug_skill.md", "debug_skill")`（三个参数必须同步，filename 用于磁盘，skillName 用于 embed 检索）
   - `SetVariable("super_debugging_path", ...)` → `SetVariable("debug_skill_path", filepath.Join(promptsDir, "skill_debug_skill.md"))`
   - **新增**：`WriteSkillFile(promptsDir, "skill_sense.md", "sense")` —— debug_skill.md 中 review debug agent 需要加载 sense skill，必须确保该文件在 doing prompts 目录中存在
   - **新增**：`SetVariable("sense_skill_path", filepath.Join(promptsDir, "skill_sense.md"))` —— 供 debug_skill.md 模板引用 `{{sense_skill_path}}`
   - 错误消息（`WriteSkillFile` 失败提示）中的旧文件名同步更新
2. `internal/prompt/plan_prompt.go`：
   - `SetVariable("super_debugging_skill_path", ...)` → `SetVariable("debug_skill_path", ...)`
   - dry-run 占位值从 `"<doing-prompts>/skill_super_debugging_zh.md"` → `"<doing-prompts>/skill_debug_skill.md"`
   - 对应变量（`superDebuggingSkillPath`）重命名为 `debugSkillPath`，路径从 `skill_super_debugging_zh.md` → `skill_debug_skill.md`
3. `internal/prompt/easy_prompt.go`：
   - `WriteSkillFile` 和 `SetVariable` 参数同 doing_prompt.go 同步更新（变量名必须统一为 `debug_skill_path`）
   - **新增**：`WriteSkillFile(promptsDir, "skill_sense.md", "sense")` 和 `SetVariable("sense_skill_path", ...)` —— 与 doing_prompt.go 保持一致
   - 读 easy.md 模板确认其引用的占位符名称，确保 SetVariable 的 key 与模板中 `{{变量名}}` 完全一致
4. `internal/prompt/manager_test.go`：
   - 将测试列表中的 `"super-debugging-zh"` 替换为 `"debug_skill"`
   - 确认 LoadCoreSkills([]string{"debug_skill"}) 返回非空内容（依赖 task1 已完成且文件存在于 embed）
5. `go build ./...` 无编译错误
6. `go test ./internal/prompt/...` 全部通过

# 测试方法

**前置条件**：task2 已完成；项目可编译

**测试1：Go 代码无旧引用**
```bash
git grep "super_debugging\|super-debugging\|superDebugging" internal/prompt/ -- '*.go' && echo "❌ 有残留" || echo "✅ 无残留"
```
- 预期：✅ 无残留

**测试2：编译通过**
```bash
go build ./... && echo "✅ 编译通过" || echo "❌ 编译失败"
```
- 预期：✅ 编译通过

**测试3：prompt 包测试通过**
```bash
go test ./internal/prompt/... -v 2>&1 | tail -20
```
- 预期：所有测试 PASS，无 FAIL

**测试4：doing dry-run 注入正确变量**
```bash
python3 -c "
import subprocess, sys
# 构建 rick（使用 .rick/tools/build_and_get_rick_bin.py）
import json
result = subprocess.run(['python3', '.rick/tools/build_and_get_rick_bin.py'], capture_output=True, text=True)
data = json.loads(result.stdout)
if not data['pass']:
    print('❌ build failed:', data)
    sys.exit(1)
bin_path = data['bin_path']
out = subprocess.run([bin_path, 'doing', '--job', 'job_16', '--dry-run'], capture_output=True, text=True)
if 'debug_skill_path' in out.stdout or 'skill_debug_skill' in out.stdout:
    print('✅ doing dry-run 注入 debug_skill_path')
else:
    print('❌ doing dry-run 未找到 debug_skill_path')
    print(out.stdout[:500])
"
```
- 预期：✅ doing dry-run 注入 debug_skill_path

**边界用例：does doing_prompt write the correct skill file**
```bash
python3 -c "
import subprocess, json
result = subprocess.run(['python3', '.rick/tools/build_and_get_rick_bin.py'], capture_output=True, text=True)
bin_path = json.loads(result.stdout)['bin_path']
subprocess.run([bin_path, 'doing', '--job', 'job_16', '--dry-run'], capture_output=True, text=True)
import os
path = '.rick/jobs/job_16/doing/prompts/skill_debug_skill.md'
print('✅ skill file written' if os.path.exists(path) else '❌ skill file missing: ' + path)
"
```
- 预期：✅ skill file written

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 super-debugging skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/prompts/skill_super_debugging_zh.md`

执行顺序：S（还原问题）→ E（视角分析）→ N（验证假设）→ 修复实现 → 3 次失败则停止找人类协作者
