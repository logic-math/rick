# 依赖关系

task3

# 任务名称

全局替换 debug.md 读取：7 处上下文加载改为 debug/ 优先、回退 debug.md 的兼容策略

# 任务目标

项目中有 7 处生产代码读取 `debug.md` 作为上下文注入。统一改为新策略：**优先**扫描 `debug/bug*.md`（只读 frontmatter 中的 summary+status，不加载全文）；若 debug/ 为空或不存在，则**回退**读取 `debug.md`（旧格式，保障历史上下文不丢失）。回退逻辑用 TODO 注释标记，2026-08 后重构时删除。

## 7 处变更位置（执行前逐一阅读实际代码确认行号）

| # | 文件 | 大致位置 | 变更说明 |
|---|------|---------|---------|
| 1 | `internal/executor/retry.go` | `loadDebugContext()` 函数体 | 函数签名不变，内部逻辑改为：`return LoadDebugContext(filepath.Dir(debugFile))`（`filepath.Dir(debugFile)` = workspaceDir，因为 debugFile = `workspaceDir/debug.md`）；原有的 `os.ReadFile(debugFile)` 逻辑全部删除，由 `LoadDebugContext` 统一处理 |
| 2 | `internal/executor/runner.go` | `DebugContent:` 赋值处（TestGenContext） | 替换为 `LoadDebugContext(tr.config.WorkspaceDir)` |
| 3 | `internal/executor/runner.go` | `contextMgr.LoadDebugFromFile(debugMdPath)` 处 | 用 `builder.SetVariable("debug_context", LoadDebugContext(tr.config.WorkspaceDir))` 替换 `contextMgr.LoadDebugFromFile` + 后续的 `formatDebugContext` 调用；`doing.md` 模板无需改动 |
| 4 | `internal/cmd/learning.go` | L102-103（主路径，使用 `jobDir`） | 统一改为 `data.DebugContent = executor.LoadDebugContext(doingDir)`（`doingDir` 在 learning.go 中已定义为 `filepath.Join(jobDir, "doing")`，不得再用 `filepath.Join(jobDir, "doing")` 手动拼接） |
| 5 | `internal/cmd/learning.go` | L164-171（主执行 + dry-run，使用 `doingDir`） | `data.DebugContent = executor.LoadDebugContext(doingDir)`，删除原 `os.ReadFile(debugPath)` 逻辑 |
| 6 | `internal/prompt/easy_prompt.go` | `debugContent := readFileOrDefault(...)` 处 | 替换为 `debugContent := executor.LoadDebugContext(doingDir)`；**注意**：easy 模式下 `doingDir` 可能不存在，`LoadDebugContext` 内部需容错（目录不存在时返回空字符串，不 panic） |
| 7 | `internal/prompt/easy_prompt.go` | `buildEasyLearningPrompt` 的"数据来源"和"Step 1"文字 | 更新"读取 debug.md"的描述为"优先读取 debug/ 下的 bug*.md 摘要，若无则读取 debug.md" |

# 关键结果

1. **新增文件 `internal/executor/debug_dir.go`**，包含三个函数：

   - `extractBugFrontmatter(content string) (summary, status string)`（私有）：解析文件首段 YAML frontmatter（两个 `---` 之间），提取 `summary:` 和 `status:` 字段值（去掉引号和多余空格）；frontmatter 缺失或字段不存在时返回空字符串

   - `LoadDebugDirSummaries(workspaceDir string) string`（**导出**）：扫描 `{workspaceDir}/debug/`，按字典序读取所有 `bug*.md`，对每个文件调用 `extractBugFrontmatter`，返回格式：
     ```
     \n\n## Debug 目录（路径 + 摘要）\n- debug/bug1-xxx.md [✅ 已解决]: 根因一句话\n
     ```
     目录不存在/无 bug*.md 时返回**空字符串**；单文件失败时 warn log 后跳过

   - `LoadDebugContext(workspaceDir string) string`（**导出**，统一入口，所有调用方使用此函数）：
     ```go
     func LoadDebugContext(workspaceDir string) string {
         if workspaceDir == "" {
             return "" // easy 模式等 doingDir 不存在场景的容错
         }
         summaries := LoadDebugDirSummaries(workspaceDir)
         if summaries != "" {
             return summaries
         }
         // TODO: backwards compat with debug.md — remove after 2026-08 once debug/ is widely adopted
         content, err := os.ReadFile(filepath.Join(workspaceDir, "debug.md"))
         if err != nil {
             return "" // 目录不存在或文件不存在均静默返回空
         }
         return string(content)
     }
     ```
   - `workspaceDir` 为空或目录不存在时均返回空字符串，不 panic（easy 模式下 doingDir 可能尚未创建）

2. **上述 7 处全部替换**（见任务目标表格），每处仅保留一次 `LoadDebugContext(...)` 调用，不再直接读 debug.md

3. **更新受影响的测试**：
   - 新增 `TestExtractBugFrontmatter`：正常 frontmatter、缺失 frontmatter、字段缺失三种情况
   - 新增 `TestLoadDebugDirSummaries`：bug*.md 被读取、非 bug*.md 被跳过、目录不存在返回空
   - 新增 `TestLoadDebugContext_WithDebugDir`：debug/ 有内容时返回摘要（不回退）
   - 新增 `TestLoadDebugContext_Fallback`：debug/ 为空时回退读取 debug.md，返回其内容
   - `executor/retry_test.go` — `TestLoadDebugContext`：在 tmpDir 下创建 `debug/bug1-test.md`（含 frontmatter），验证返回摘要而非全文
   - `executor/runner_test.go` — `TestGenerateDoingPromptFile_WithDebugContext`：在 workspace 创建 `debug/bug1-test.md`，验证 prompt 含摘要不含 bug 正文；删除旧的 debug.md 写入逻辑（或保留作为回退测试）
   - `cmd/learning_test.go`：`DebugContent` 断言改为兼容 debug/ 摘要格式

4. `go build ./...` 无编译错误
5. `go test ./internal/executor/... ./internal/cmd/... ./internal/prompt/...` 全部通过

# 测试方法

**前置条件**：task3 已完成；`go test ./internal/prompt/...` 全部通过

**测试0：7 处变更全部使用 LoadDebugContext**
```bash
echo "=== 生产代码使用 LoadDebugContext ===" 
grep -rn "LoadDebugContext" internal/ --include="*.go" | grep -v "_test.go"
total=$(grep -rn "LoadDebugContext" internal/ --include="*.go" | grep -v "_test.go" | wc -l | tr -d ' ')
[ "$total" -ge 5 ] && echo "✅ 生产代码出现 $total 次（至少5处）" || echo "❌ 仅 $total 次，覆盖不足"
echo "=== 无残留的直接 debug.md 读取 ==="
grep -rn '"debug\.md"' internal/executor/ internal/cmd/ internal/prompt/ --include="*.go" | grep -v "_test.go" | grep -v "debug_dir.go" && echo "❌ 仍有直接读取" || echo "✅ 无残留"
```
- 预期：LoadDebugContext ≥ 5 次，无残留 ✅

**测试1：编译通过**
```bash
go build ./... && echo "✅" || echo "❌ 编译失败"
```
- 预期：✅

**测试2：全部单元测试通过**
```bash
go test ./internal/executor/... ./internal/cmd/... ./internal/prompt/... 2>&1 | grep -E "^(--- FAIL|FAIL|ok)" | tail -20
```
- 预期：无 FAIL 行

**测试3：TODO 注释存在（防止回退逻辑未标注）**
```bash
grep -n "TODO.*2026-08\|TODO.*backwards compat" internal/executor/debug_dir.go && echo "✅ TODO 注释存在" || echo "❌ 缺少 TODO 注释"
```
- 预期：✅ TODO 注释存在

**测试4：debug/ 优先策略——有 debug/ 时不读 debug.md**
```bash
python3 - <<'EOF'
import subprocess, json, os

result = subprocess.run(['python3', '.rick/tools/build_and_get_rick_bin.py'], capture_output=True, text=True)
data = json.loads(result.stdout)
if not data['pass']:
    print('❌ build failed'); exit(1)

doing_dir = '.rick/jobs/job_16/doing'
debug_dir = os.path.join(doing_dir, 'debug')
os.makedirs(debug_dir, exist_ok=True)

# 写 debug/bug1（新格式）
SUMMARY = 'NEW_FORMAT_SUMMARY_MARKER'
with open(os.path.join(debug_dir, 'bug1-test.md'), 'w') as f:
    f.write(f'---\nsummary: "{SUMMARY}"\nstatus: "✅ 已解决"\n---\n# bug1\n')

# 写 debug.md（旧格式）
OLD = 'OLD_FORMAT_DEBUG_MD_MARKER'
with open(os.path.join(doing_dir, 'debug.md'), 'w') as f:
    f.write(f'## task1: test\n{OLD}\n')

out = subprocess.run([data['bin_path'], 'doing', '--job', 'job_16', '--dry-run'], capture_output=True, text=True)
has_new = SUMMARY in out.stdout
has_old = OLD in out.stdout
if has_new and not has_old:
    print('✅ debug/ 优先，未回退读 debug.md')
elif has_old and not has_new:
    print('❌ 读了 debug.md 而不是 debug/')
else:
    print(f'❌ 状态异常: new={has_new} old={has_old}')

os.remove(os.path.join(debug_dir, 'bug1-test.md'))
os.rmdir(debug_dir)
os.remove(os.path.join(doing_dir, 'debug.md'))
EOF
```
- 预期：✅ debug/ 优先，未回退

**测试5：回退策略——debug/ 为空时回退读 debug.md**
```bash
python3 - <<'EOF'
import subprocess, json, os, shutil

result = subprocess.run(['python3', '.rick/tools/build_and_get_rick_bin.py'], capture_output=True, text=True)
bin_path = json.loads(result.stdout)['bin_path']

doing_dir = '.rick/jobs/job_16/doing'
debug_dir = os.path.join(doing_dir, 'debug')
shutil.rmtree(debug_dir, ignore_errors=True)  # 强制清理测试4的残留，保证隔离

OLD = 'FALLBACK_DEBUG_MD_MARKER'
with open(os.path.join(doing_dir, 'debug.md'), 'w') as f:
    f.write(f'## task1: test\n{OLD}\n')

out = subprocess.run([bin_path, 'doing', '--job', 'job_16', '--dry-run'], capture_output=True, text=True)
print('✅ 回退读取 debug.md 成功' if OLD in out.stdout else '❌ 回退失败，debug.md 内容未出现')

os.remove(os.path.join(doing_dir, 'debug.md'))
EOF
```
- 前置条件：debug/ 目录不存在
- 预期：✅ 回退读取 debug.md 成功

**测试6：learning dry-run 优先使用 debug/ 摘要**
```bash
python3 - <<'EOF'
import subprocess, json, os

result = subprocess.run(['python3', '.rick/tools/build_and_get_rick_bin.py'], capture_output=True, text=True)
bin_path = json.loads(result.stdout)['bin_path']

debug_dir = '.rick/jobs/job_16/doing/debug'
os.makedirs(debug_dir, exist_ok=True)
MARKER = 'LEARNING_SUMMARY_MARKER_XYZ'
with open(os.path.join(debug_dir, 'bug1-lr.md'), 'w') as f:
    f.write(f'---\nsummary: "{MARKER}"\nstatus: "✅ 已解决"\n---\n# bug1\n')

out = subprocess.run([bin_path, 'learning', '--job', 'job_16', '--dry-run'], capture_output=True, text=True)
print('✅ learning dry-run 含 debug/ 摘要' if MARKER in out.stdout else '❌ 摘要未出现')

os.remove(os.path.join(debug_dir, 'bug1-lr.md'))
if not os.listdir(debug_dir): os.rmdir(debug_dir)
EOF
```
- 预期：✅

**测试7：easy_prompt.go 文字已更新（学习提示词不再硬说"读取 debug.md"）**
```bash
grep "优先读取.*debug/\|debug/ 下.*bug.*摘要" internal/prompt/easy_prompt.go && echo "✅ easy 文字已更新" || echo "❌ easy_prompt.go 仍引用旧描述"
```
- 预期：✅ easy 文字已更新

**边界用例：debug/ 不存在且无 debug.md 时各阶段正常运行（返回空字符串）**
```bash
rm -rf .rick/jobs/job_16/doing/debug .rick/jobs/job_16/doing/debug.md 2>/dev/null || true
python3 -c "
import subprocess, json
b = json.loads(subprocess.run(['python3','.rick/tools/build_and_get_rick_bin.py'],capture_output=True,text=True).stdout)['bin_path']
for cmd in [['doing','--job','job_16','--dry-run'],['learning','--job','job_16','--dry-run']]:
    r = subprocess.run([b]+cmd, capture_output=True, text=True)
    print('✅' if r.returncode==0 else '❌', ' '.join(cmd))
"
```
- 预期：两个命令均 ✅

**异常路径A：无 frontmatter 的 bug*.md 不崩溃（仅验证不 panic）**
```bash
python3 - <<'EOF'
import subprocess, json, os, shutil

b = json.loads(subprocess.run(['python3','.rick/tools/build_and_get_rick_bin.py'],capture_output=True,text=True).stdout)['bin_path']
d = '.rick/jobs/job_16/doing/debug'
shutil.rmtree(d, ignore_errors=True)  # 确保前置状态干净
try:
    os.makedirs(d, exist_ok=True)
    open(os.path.join(d,'bug1-no-fm.md'),'w').write('# bug1\nno frontmatter\n')
    r = subprocess.run([b,'doing','--job','job_16','--dry-run'], capture_output=True, text=True)
    print('✅ 无 frontmatter 不崩溃' if r.returncode==0 else '❌ 崩溃: '+r.stderr[:200])
finally:
    shutil.rmtree(d, ignore_errors=True)
EOF
```
- 预期：✅

**异常路径B：bug*.md 存在（即使无 frontmatter）时不回退到 debug.md**
```bash
python3 - <<'EOF'
import subprocess, json, os, shutil

b = json.loads(subprocess.run(['python3','.rick/tools/build_and_get_rick_bin.py'],capture_output=True,text=True).stdout)['bin_path']
d = '.rick/jobs/job_16/doing/debug'
shutil.rmtree(d, ignore_errors=True)  # 确保前置状态干净
OLD = 'MUST_NOT_APPEAR_FROM_FALLBACK'
debug_md = '.rick/jobs/job_16/doing/debug.md'
try:
    os.makedirs(d, exist_ok=True)
    open(os.path.join(d,'bug1-no-fm.md'),'w').write('# bug1\nno frontmatter\n')
    with open(debug_md,'w') as df: df.write(f'## task1\n{OLD}\n')
    r = subprocess.run([b,'doing','--job','job_16','--dry-run'], capture_output=True, text=True)
    if OLD not in r.stdout:
        print('✅ bug*.md 存在时不回退到 debug.md')
    else:
        print('❌ 意外回退（bug*.md 存在时不应读 debug.md）')
finally:
    shutil.rmtree(d, ignore_errors=True)
    if os.path.exists(debug_md): os.remove(debug_md)
EOF
```
- 预期：✅ bug*.md 存在时不回退到 debug.md

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 super-debugging skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/prompts/skill_super_debugging_zh.md`

执行顺序：S（还原问题）→ E（视角分析）→ N（验证假设）→ 修复实现 → 3 次失败则停止找人类协作者
