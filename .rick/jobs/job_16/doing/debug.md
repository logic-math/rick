## task1: 创建 debug_skill.md（三阶段调试 SOP + review debug agent 协议）

**分析过程 (Analysis)**:
- 阅读了现有 `internal/prompt/templates/skills/super-debugging-zh.md` 了解已有调试技能格式
- 阅读了 `internal/prompt/templates/skills/sense.md` 了解 SENSE 方法论结构
- 确认 `internal/prompt/templates/skills/` 目录已存在，直接在其中创建新文件
- 任务要求内聚三阶段 SOP、review debug agent 协议、bug 文件格式规范、SENSE 方法集成

**实现步骤 (Implementation)**:
1. 设计文件结构：frontmatter → 铁律 → 准备阶段 → 阶段一 → 阶段二 → 阶段三 → review debug agent 协议 → 流程图 → 完整示例 → 反模式
2. 在准备阶段定义 debug/ 目录约定、bug 编号规则、YAML frontmatter 格式规范、两种合法终止状态
3. 在阶段一（源码推理法）定义 review debug agent 触发点（建立假设时）、主 Agent 执行-回滚-记录循环、上限 3 次
4. 在阶段二（增量调试法）定义 review debug agent 触发点（简化复现时）、基线判断逻辑、无基线跳过规则
5. 在阶段三（科学实验法）定义两个触发点（简化复现 + 传播链假设）、运行时工具列表（delve/pprof/pdb/strace）、上限 5 次、超限处理流程
6. 在 review debug agent 协议章节明确输入/输出格式/角色约束，并内嵌 SENSE 方法（./skill_sense.md 硬编码路径）
7. 添加三阶段递进文字流程图和完整 bug 文件示例

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`python3 .rick/jobs/job_16/doing/tests/task1.py`
- 测试输出：
  ```
  {"pass": true, "errors": []}
  ```
- 结论：✅ 通过

## task2: 更新 doing.md / plan.md 模板引用，删除 super-debugging-zh.md

**分析过程 (Analysis)**:
- 阅读了 `internal/prompt/templates/doing.md`、`plan.md`、`easy.md` 确认所有 super_debugging 引用位置
- 确认 `internal/prompt/templates/skills/super-debugging-zh.md` 存在，需用 `git rm` 删除
- doing.md 中 DEBUG 铁律、skill 列表、Commitment、Scarcity 章节均需更新；debug{N} 格式整节需删除
- plan.md 中 task.md 格式的调试方法章节需替换 super_debugging_skill_path → debug_skill_path
- easy.md 中 skill 列表第2条需替换为 debug-skill（测试也检查了此文件）

**实现步骤 (Implementation)**:
1. `git rm internal/prompt/templates/skills/super-debugging-zh.md`
2. doing.md：Line 3 声明改为 skill:debug-skill
3. doing.md：skill 列表第3行替换为 debug-skill，新增 sense_skill_path 行
4. doing.md：DEBUG 铁律章节替换为三阶段调试法
5. doing.md：Commitment 块改为 debug-skill
6. doing.md：Scarcity 章节更新 Phase 1 描述
7. doing.md：删除"遇到问题时的详细记录"debug{N} 整节，替换为单行 bug{n}-{描述}.md 指引
8. plan.md：调试方法章节替换为 debug_skill_path + 三阶段调试法描述
9. easy.md：skill:super-debugging 条目替换为 skill:debug-skill

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`python3 .rick/jobs/job_16/doing/tests/task2.py`
- 测试输出：
  ```
  {"pass": true, "errors": []}
  ```
- 结论：✅ 通过

## task3: 更新 Go prompt 文件，将 super-debugging-zh 切换到 debug_skill

**分析过程 (Analysis)**:
- 阅读了 `doing_prompt.go`、`plan_prompt.go`、`easy_prompt.go`、`manager_test.go` 确认所有旧引用位置
- 检查 `easy.md`、`plan.md`、`doing.md` 模板确认新变量名（`debug_skill_path`、`sense_skill_path`）
- 确认 `templates/skills/debug_skill.md` 和 `templates/skills/sense.md` 均已存在于 embed

**实现步骤 (Implementation)**:
1. `doing_prompt.go`：WriteSkillFile `super-debugging-zh` → `debug_skill`；新增 `sense` WriteSkillFile；SetVariable `super_debugging_path` → `debug_skill_path`；新增 `sense_skill_path`
2. `plan_prompt.go`：dry-run `super_debugging_skill_path` → `debug_skill_path`；GeneratePlanPromptFile 中变量名 `superDebuggingSkillPath` → `debugSkillPath`，路径 `skill_super_debugging_zh.md` → `skill_debug_skill.md`
3. `easy_prompt.go`：WriteSkillFile `super-debugging-zh` → `debug_skill`；新增 `sense` WriteSkillFile；SetVariable `super_debugging_path` → `debug_skill_path`；新增 `sense_skill_path`
4. `manager_test.go`：skills 列表 `"super-debugging-zh"` → `"debug_skill"`

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`go build ./... && go test ./internal/prompt/...`
- 测试输出：
  ```
  ok  	github.com/sunquan/rick/internal/prompt	0.462s
  ```
- 结论：✅ 通过

## task4: 全局替换 debug.md 读取为 debug/ 优先、回退 debug.md 的兼容策略

**分析过程 (Analysis)**:
- 阅读了 7 处变更位置的实际代码：retry.go、runner.go（×2）、learning.go（×2）、easy_prompt.go（×2）
- 确认 `executor` 包导入 `prompt`，`prompt` 不能反向导入 `executor`（循环依赖），因此 easy_prompt.go 需本地 `loadDebugContextLocal` 函数复制同等逻辑
- runner.go location 3 需要新增 `SetDebugRaw/GetDebugRaw` 到 ContextManager，并在 doing_prompt.go 的 `formatDebugContext` 中加回退逻辑，以保持旧测试（`LoadDebugFromContent`）兼容

**实现步骤 (Implementation)**:
1. 新建 `internal/executor/debug_dir.go`：`extractBugFrontmatter`、`LoadDebugDirSummaries`、`LoadDebugContext`（含 TODO 2026-08 回退注释）
2. `retry.go`：`loadDebugContext` 改为 `return LoadDebugContext(filepath.Dir(debugFile))`，移除 `os` 依赖改用 `path/filepath`
3. `runner.go`：location 2 `DebugContent: LoadDebugContext(tr.config.WorkspaceDir)`；location 3 `contextMgr.SetDebugRaw(LoadDebugContext(tr.config.WorkspaceDir))`
4. `context.go`：新增 `debugRaw` 字段 + `SetDebugRaw/GetDebugRaw` 方法
5. `doing_prompt.go`：`formatDebugContext(contextMgr.GetDebug())` → `GetDebugRaw()` 优先，空时回退 `formatDebugContext`（保持旧测试兼容）
6. `learning.go`：location 4 `doingDir` + `executor.LoadDebugContext(doingDir)`；location 5 同理，删除旧 os.ReadFile 模式
7. `easy_prompt.go`：`loadDebugContextLocal` 复制逻辑；更新 Step 1 和数据来源描述
8. 新建 `debug_dir_test.go`（6 个测试）；更新 `retry_test.go`、`runner_test.go`（debug/ 路径）

**遇到的问题 (Issues)**:
- 初次运行测试失败：`TestGenerateDoingPrompt_WithRetry` 和 `TestIntegration_RetryPromptsIncludeDebug` 均使用 `LoadDebugFromContent` 路径（旧 API），但 `GetDebugRaw()` 返回 ""
- 修复：在 `doing_prompt.go` 中，`GetDebugRaw()` 为空时回退至 `formatDebugContext(contextMgr.GetDebug())`，兼容旧测试

**验证结果 (Verification)**:
- 测试命令：`go build ./... && go test ./internal/executor/... ./internal/cmd/... ./internal/prompt/...`
- 测试输出：
  ```
  ok  	github.com/sunquan/rick/internal/executor	1.529s
  ok  	github.com/sunquan/rick/internal/cmd	63.679s
  ok  	github.com/sunquan/rick/internal/prompt	(cached)
  ```
- 结论：✅ 通过
