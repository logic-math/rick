# 依赖关系
task1

# 任务名称

在 learning_check 和 dream_check 中集成 loops/skills 格式校验

# 任务目标

在 `internal/cmd/` 包中新建共享校验文件 `tools_loops_skills_check.go`，实现 `runLoopsAndSkillsCheck(rickDir string) []string` 函数，校验 `.rick/loops/` 和 `.rick/skills/` 目录下所有候选文件的格式合规性，并分别在 `runLearningCheck()` 和 `runDreamCheck()` 末尾调用，确保 learning/dream 阶段写出的候选文件在 check 时被一并校验。

## 校验规则

**loops 校验**（对 `.rick/loops/` 下所有 `*.md`，跳过 `README.md`）：
- frontmatter 存在（文件以 `---` 开头，有闭合 `---`）
- frontmatter 包含 `name` 字段（非空）
- frontmatter 包含 `trigger` 字段（非空）
- 正文包含 `## 目标` 章节（Goal）
- 正文包含 `## 上下文管理` 章节（Context Management）
- 正文包含 `## 可调用工具` 章节（Tool Access）
- 正文包含 `## 产出评估` 章节（Output Evaluation）
- 正文包含 `## 停止标准` 章节（Termination Condition）

**skills 校验**（对 `.rick/skills/` 下所有 `*.md`，跳过 `README.md`）：
- frontmatter 存在
- frontmatter 包含 `name` 字段（非空）
- frontmatter 包含 `description` 字段（非空）
- 正文包含 `## When to Use` 章节
- 正文包含 `## Procedure` 章节
- 正文包含 `## Pitfalls` 章节
- 正文包含 `## Verification` 章节

**目录不存在时**：跳过校验，不报错（loops/skills 初期可为空）

**错误输出格式**（逐文件列出，供 AI 定位修复）：
```
loops/skills check errors:
  - loops/candidate_loop_1.md: missing 'trigger' field in frontmatter
  - loops/candidate_loop_1.md: missing section '## 产出评估'
  - skills/candidate_skill_1.md: missing 'description' field in frontmatter
  - skills/candidate_skill_1.md: missing section '## Procedure'
```

## 函数签名

```go
// runLoopsAndSkillsCheck validates .rick/loops/*.md and .rick/skills/*.md format.
// Returns a list of error strings (empty = pass). Skips README.md in each dir.
// Directories that don't exist are silently skipped.
func runLoopsAndSkillsCheck(rickDir string) []string
```

## 集成方式

**`runLearningCheck(learningDir string) error`** 末尾追加：
```go
// 使用 workspace.GetRickDir() 而非 filepath.Dir 层级推导，
// 避免依赖目录层数假设（learningDir 层级为 .rick/jobs/job_N/learning，需三层 Dir 才到 .rick/）
rickDir, err := workspace.GetRickDir()
if err != nil {
    return fmt.Errorf("failed to resolve rick dir for loops/skills check: %w", err)
}
errs := runLoopsAndSkillsCheck(rickDir)
if len(errs) > 0 {
    return fmt.Errorf("loops/skills check errors:\n  - %s", strings.Join(errs, "\n  - "))
}
```

**`runDreamCheck(rickDir string) error`** 末尾追加：
```go
errs := runLoopsAndSkillsCheck(rickDir)
if len(errs) > 0 {
    return fmt.Errorf("loops/skills check errors:\n  - %s", strings.Join(errs, "\n  - "))
}
```

## 实现约束

- 使用 `strings.Split`/`strings.TrimSpace` 解析 frontmatter，**不引入任何外部 YAML 库**（与 task3 保持一致）
- frontmatter 解析：找第一个 `---` 和第二个 `---` 之间的内容，按行遍历匹配 `key: value`
- 章节检测：用 `strings.Contains(content, "## 章节名")` 检查
- 文件名过滤：用 `filepath.Base(path) == "README.md"` 跳过说明文件
- 错误信息格式：`"loops/filename.md: <具体原因>"`（含目录前缀，便于 AI 定位文件）

# 关键结果

1. `internal/cmd/tools_loops_skills_check.go` 文件存在，包含 `runLoopsAndSkillsCheck(rickDir string) []string` 函数
2. `runLearningCheck()` 末尾调用 `runLoopsAndSkillsCheck`，有错误时返回包含具体文件路径的 error
3. `runDreamCheck()` 末尾同样调用，有错误时返回包含具体文件路径的 error
4. `NewLearningCheckCmd()` 的 Long description 更新，在 "Checks performed" 列表中补充：
   ```
   - .rick/loops/*.md: frontmatter (name, trigger) + 5 sections (目标/上下文管理/可调用工具/产出评估/停止标准)
   - .rick/skills/*.md: frontmatter (name, description) + 4 sections (When to Use/Procedure/Pitfalls/Verification)
   - README.md in each dir is skipped
   ```
5. `NewDreamCheckCmd()` 的 Long description 同步更新，补充相同的 loops/skills 校验说明
6. `tools.go` 的 Long description 中 learning_check 和 dream_check 的行内说明更新，提及 loops/skills 格式校验
7. `go test ./internal/cmd/... -run TestLoopsSkillsCheck` 全部通过
8. `./scripts/build.sh` 成功

# 测试方法

1. **正常路径 - loops/skills 格式正确时 learning_check 通过**：
   - 前置条件：手动执行 `mkdir -p .rick/jobs/job_22/learning && echo "# Job job_22 SUMMARY" > .rick/jobs/job_22/learning/SUMMARY.md` 创建占位文件；`.rick/loops/candidate_loop_1.md` 格式合规（含 frontmatter name/trigger + 五章节）；`.rick/skills/candidate_skill_1.md` 格式合规（含 frontmatter name/description + 四章节）
   - 操作：`./bin/rick tools learning_check job_22`
   - 预期输出：`✅ learning check passed`，exit code 0

2. **异常路径 - loop 文件缺 trigger 时 learning_check 报错**：
   - 前置条件：创建 `.rick/loops/bad_loop.md`，frontmatter 只有 name 无 trigger
   - 操作：`./bin/rick tools learning_check job_22 2>&1`
   - 预期输出：包含 `loops/bad_loop.md: missing 'trigger' field`，exit code 1
   - 清理：删除 bad_loop.md

3. **异常路径 - skill 文件缺章节时 dream_check 报错**：
   - 前置条件：创建 `.rick/skills/bad_skill.md`，有 frontmatter 但缺 `## Procedure` 章节
   - 操作：`./bin/rick tools dream_check 2>&1`
   - 预期输出：包含 `skills/bad_skill.md: missing section '## Procedure'`，exit code 1
   - 清理：删除 bad_skill.md

4. **边界用例 - loops/skills 目录不存在时不报错**：
   - 前置条件：`.rick/loops/` 和 `.rick/skills/` 均不存在
   - 操作：`./bin/rick tools learning_check job_22 2>&1`
   - 预期输出：`✅ learning check passed`（跳过不存在的目录），exit code 0

5. **边界用例 - README.md 被跳过**：
   - 前置条件：`.rick/loops/README.md` 存在（格式规范文件，不含五要素章节）
   - 操作：`./bin/rick tools learning_check job_22 2>&1`
   - 预期输出：不报告 README.md 相关错误，check 正常通过

6. **单元测试覆盖**：
   - 操作：`go test ./internal/cmd/... -run TestLoopsSkillsCheck -v`
   - 预期输出：覆盖以下场景（缺一不可）：
     - 合规 loop 文件 → 返回空切片
     - 缺 trigger 字段 → 返回含文件名和 "trigger" 的错误
     - 缺 `## 产出评估` 章节 → 返回含章节名的错误
     - 合规 skill 文件 → 返回空切片
     - skill 缺 description 字段 → 返回含 "description" 的错误
     - skill 缺 `## Procedure` 章节 → 返回含章节名的错误
     - 目录不存在 → 返回空切片（不报错）
     - README.md 被跳过 → 返回空切片（不误报）
     - 单文件多错误 → 全部报出（非 fail-fast）

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill Phase 1-6 执行，每阶段达上限后升级人工协作
