# 依赖关系


# 任务名称
提取公共 frontmatter 解析函数到 internal/parser 包

# 任务目标
在 `internal/parser/frontmatter.go` 中实现 `ExtractBugFrontmatter(content string) (summary, status string)` 导出函数，消除 `internal/executor/debug_dir.go` 和 `internal/prompt/easy_prompt.go` 中两份相同的私有 frontmatter 解析逻辑重复。

注：`internal/parser` 包已存在（含 context.go、coordinator.go 等），无需创建新包。

# 关键结果
1. `internal/parser/frontmatter.go` 新增 `ExtractBugFrontmatter` 函数，逻辑与现有两份实现完全一致
2. `internal/executor/debug_dir.go` 删除私有 `extractBugFrontmatter` 函数，改为调用 `parser.ExtractBugFrontmatter`
3. `internal/prompt/easy_prompt.go` 中 `loadDebugContextLocal` 内的内联解析逻辑替换为调用 `parser.ExtractBugFrontmatter`；同时在 import 块新增 `"github.com/sunquan/rick/internal/parser"`（当前 easy_prompt.go 无此 import，遗漏会导致编译报 undefined: parser）（注：easy_prompt.go 将在 task3 中删除，此处修改确保算法实现在删除前已迁移至 parser 包，避免逻辑丢失）
4. `internal/executor/debug_dir_test.go`：只删除 `TestExtractBugFrontmatter` 整个函数块，其他测试函数（`TestLoadDebugDirSummaries`、`TestLoadDebugContext_*` 等）完整保留；等价覆盖由 `internal/parser/frontmatter_test.go` 新增的测试承担
5. `go test ./internal/executor/... ./internal/parser/...` 通过
6. `go build ./...` 通过，无循环导入

# 测试方法
1. **parser.ExtractBugFrontmatter 正常解析**
   - 前置条件：`internal/parser/frontmatter_test.go` 新建
   - 输入：包含 `---\nsummary: "修复 nil 指针"\nstatus: resolved\n---\n正文` 的字符串
   - 操作：调用 `parser.ExtractBugFrontmatter(content)`
   - 预期输出：`summary == "修复 nil 指针"`，`status == "resolved"`

2. **无 frontmatter 时返回空**
   - 输入：`"只有正文，没有 frontmatter"` 的字符串
   - 预期输出：`summary == ""`，`status == ""`

3. **单引号包裹值**
   - 输入：`---\nsummary: '单引号值'\n---\n`
   - 预期输出：`summary == "单引号值"`（引号被 Trim）

4. **executor 单元测试无断裂**
   - 前置条件：task2 所有修改完成
   - 操作：`go test ./internal/executor/... ./internal/parser/...`
   - 预期输出：两个包均 PASS，exit code 0

5. **easy_prompt.go 内联解析已替换，parser import 已添加**
   - 操作 A：`grep -n "strings.HasPrefix.*summary\|strings.HasPrefix.*status" internal/prompt/easy_prompt.go`；预期：无输出
   - 操作 B：`grep -n '"github.com/sunquan/rick/internal/parser"' internal/prompt/easy_prompt.go`；预期：1 行匹配（import 存在，否则 task3 构建会报 undefined: parser）

6. **无循环导入验证**
   - 操作：`go build ./...`
   - 预期输出：exit code 0，无 `import cycle` 错误

# 调试方法
遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_17/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill 三阶段调试法（源码推理法→增量调试法→科学实验法）执行，每阶段达上限后升级人工协作
