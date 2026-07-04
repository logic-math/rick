# 依赖关系

（无依赖）

# 任务名称

添加 `.rick/draft/` 目录基础设施并注入 draft_dir 变量

# 任务目标

在 `workspace/paths.go` 中添加 `GetDraftDir()` 函数，在 `human_loop.go` 中创建 draft 目录结构，在 `human_loop_prompt.go` 中将 `{{draft_dir}}` 注入主模板和三个子 agent 模板，使 draft/ 目录可寻址并被所有 human-loop 模板引用。

# 关键结果

1. `workspace/paths.go` 新增 `DraftDirName = "draft"` 常量和 `GetDraftDir() (string, error)` 函数，返回 `{rickDir}/draft`
2. `internal/cmd/human_loop.go` 在 RFC dir 创建后创建三个目录：`draft/`、`draft/concepts/`、`draft/human-learning/`，幂等（MkdirAll）
3. `internal/prompt/human_loop_prompt.go` 中 `GenerateHumanLoopPromptFile` 和 `GenerateHumanLoopPrompt` 签名增加 `draftDir string` 参数，并将 `{{draft_dir}}` 注入主模板和 think/learn/express 三个子模板
4. `internal/cmd/human_loop.go` 正确传入 draftDir（dry-run 模式用占位符 `<draft>/`）
5. 所有现有测试继续通过，新增单元测试覆盖以下场景

# 测试方法

**前提：使用 skill:tdd，先写失败测试，再实现，再看绿。**

### 测试 1：GetDraftDir 返回正确路径

- 前置条件：`os.Chdir` 到 tmpDir，tmpDir 下有 `.rick/` 目录
- 输入：无参数
- 操作序列：调用 `workspace.GetDraftDir()`
- 预期输出：返回 `{tmpDir}/.rick/draft`，error 为 nil

```bash
go test ./internal/workspace/... -run TestGetDraftDir -v
```

### 测试 2：human-loop 命令创建 draft 目录结构

- 前置条件：workDir 下有 `.rick/` 和 `.rick/config.json`（mock claude），dry-run=false；使用 mock claude（exit 0）
- 输入：`rick human-loop "测试主题"`（mock claude）
- 操作序列：执行命令，检查目录存在性
- 预期输出：`{workDir}/.rick/draft/`、`draft/concepts/`、`draft/human-learning/` 均已创建（Stat 不返回 IsNotExist）

```bash
go test ./internal/cmd/... -run TestHumanLoopDraftDirsCreated -v -timeout 30s
```

### 测试 3：dry-run 输出包含 draft_dir（已替换，无 `{{` 残留）

- 前置条件：workDir 下有 `.rick/` 目录，dry-run=true
- 输入：`rick human-loop --dry-run "测试主题"`
- 操作序列：捕获 stdout
- 预期输出：输出中包含 `draft` 路径字符串，不含未替换变量 `{{draft_dir}}`

```bash
./bin/rick human-loop --dry-run '测试主题' | grep -v '{{draft_dir}}'
./bin/rick human-loop --dry-run '测试主题' | grep 'draft'
```

### 测试 4：GenerateHumanLoopPromptFile 注入 draft_dir 到子模板

- 前置条件：mock promptManager，tmpDir 作为 rfcDir 和 draftDir
- 输入：`draftDir = "/tmp/test-draft"`
- 操作序列：调用 `GenerateHumanLoopPromptFile("topic", rfcDir, "/tmp/test-draft", pm)`，读取 think/express 子模板文件内容
- 预期输出：think 和 express 文件内容包含 `/tmp/test-draft`，不含 `{{draft_dir}}`

```bash
go test ./internal/prompt/... -run TestGenerateHumanLoopPromptFileInjectsDraftDir -v
```

# 约束说明

1. `GetDraftDir()` 实现位置：`paths.go` 中 `GetRFCDir()` 函数之后，返回 `filepath.Join(rickDir, DraftDirName)`
2. **dry-run 占位符**：`human_loop.go` 中 dry-run 分支调用 `GenerateHumanLoopPrompt` 时传入 `"<draft>/"` 作为 draftDir 占位符
3. **两处调用方均需更新**：`human_loop.go` 第37行（dry-run 分支）和第52行（正常分支），两处均需加 draftDir 参数
4. dry-run 模式下**不**创建 draft 目录（仅打印 prompt，与现有 RFC dir 逻辑保持一致）

# 参考 Loops/Skills

- `tdd-red-green-refactor-loop`：Go 代码改动时触发，先写失败测试再实现
- `.rick/skills/verify_go_changes_skill/skill.md`：每次 Go 改动后构建验证
- `.rick/skills/mark_task_success_skill/skill.md`：task 完成后标记 success + commit

### 边界测试：draftDir 为空字符串

- 预期输出：不 panic，模板中 `{{draft_dir}}` 替换为空字符串（不影响执行）
