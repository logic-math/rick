# 依赖关系


# 任务名称

实现 `LoadLoopsContext()` 函数，遍历 `.rick/loops/` 提取触发点列表

# 任务目标

在 `internal/prompt/manager.go`（或 `context_helpers.go`）中实现 `LoadLoopsContext(loopsDir string) string` 函数，遍历指定目录下所有 `*.md` 文件，解析 YAML frontmatter 提取 `trigger` 字段，拼接为格式化的 loops 触发列表字符串，供各 phase prompt builder 调用。

函数签名：`func LoadLoopsContext(loopsDir string) string`

输出格式（当目录非空时）：
```
## 可用的项目 Loops

- **{name}**：{trigger}
- **{name}**：{trigger}
```

输出格式（当目录为空或不存在时）：
```
## 可用的项目 Loops

（暂无项目 Loop 记录）
```

# 关键结果

1. `LoadLoopsContext(loopsDir string) string` 函数在 `internal/prompt/context_helpers.go` 中实现（若文件已存在则追加，否则新建）；先用 `grep -r "LoadLoopsContext" internal/prompt/` 确认无重复声明
2. 函数使用 `strings.Split`、`strings.TrimSpace` 等标准库解析 YAML frontmatter（`---\nkey: value\n---` 格式），**禁止引入任何 YAML 外部库**（当前 go.mod 无 yaml 依赖），解析逻辑：按行遍历 frontmatter 块，匹配 `key: value` 格式提取 `name` 和 `trigger`
3. 目录不存在或为空时返回"暂无项目 Loop 记录"占位文本
4. 文件无 frontmatter 或缺少 `trigger` 字段时跳过该文件（不 panic，log warning）
5. `internal/prompt/` 包单元测试覆盖以下场景：空目录、单文件、多文件、无 frontmatter 文件、缺 trigger 字段文件
6. `go test ./internal/prompt/... -run TestLoadLoopsContext` 全部通过

# 测试方法

1. **正常路径 - 多文件遍历**：
   - 前置条件：创建临时目录，写入两个 loop.md 文件，各有完整 frontmatter（name + trigger）
   - 输入：`LoadLoopsContext(tmpDir)`
   - 操作：在测试文件 `internal/prompt/context_helpers_test.go` 中创建 `TestLoadLoopsContext_MultipleFiles`
   - 预期输出：返回字符串包含 "## 可用的项目 Loops"，每个 name 和 trigger 均以 `- **name**：trigger` 格式出现

2. **边界用例 - 空目录**：
   - 前置条件：空临时目录
   - 输入：`LoadLoopsContext(emptyDir)`
   - 预期输出：返回字符串包含 "暂无项目 Loop 记录"

3. **边界用例 - 目录不存在**：
   - 前置条件：传入不存在的路径
   - 输入：`LoadLoopsContext("/nonexistent/path")`
   - 预期输出：返回字符串包含 "暂无项目 Loop 记录"，函数不 panic

4. **边界用例 - 文件缺 trigger 字段**：
   - 前置条件：临时目录内有一个 loop.md，frontmatter 只有 name 无 trigger；另一个有完整字段
   - 输入：`LoadLoopsContext(tmpDir)`
   - 预期输出：只有完整字段的 loop 出现在结果中，无 trigger 的文件被跳过，函数不 panic

5. **边界用例 - 非 .md 文件被忽略**：
   - 前置条件：临时目录内有 `loop.md`（有效）和 `README.txt`（无效）
   - 预期输出：结果中只包含 loop.md 的 trigger，.txt 文件不出现

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill Phase 1-6 执行，每阶段达上限后升级人工协作
