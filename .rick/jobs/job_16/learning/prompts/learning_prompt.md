# Rick Learning 阶段

你是一个资深的技术文档专家和知识管理专家。根据本次 job 执行过程，总结知识、经验和教训，沉淀可复用 skills。

## 执行上下文

**Job**: job_16

### OKR（任务目标）

# Job OKR: 实现 RFC-debugging，建立三阶段科学调试体系

## 目标 (Objective)
将 Rick 的调试能力从"盲目重试"升级为基于状态机理论的科学调试——三阶段 SOP（源码推理→增量调试→科学实验）+ review debug agent + 运行时工具指引，消除调试上下文的恶性循环。

## 关键结果 (Key Results)
- KR1: `internal/prompt/templates/skills/debug_skill.md` 存在，包含准备阶段、三阶段 SOP（含回滚约束、循环上限）、review debug agent 协议（两个触发点）、运行时观察工具指引、debug/ 目录文件格式
- KR2: `super-debugging-zh.md` 已删除；`doing.md` 和 `plan.md` 模板中所有 `super_debugging*` 引用替换为 `debug_skill_path`；doing.md 的 debug{N} 调试记录格式替换为 debug_skill 加载指令
- KR3: `doing_prompt.go`、`plan_prompt.go`、`easy_prompt.go` 的 WriteSkillFile/SetVariable 调用全部从 "super-debugging-zh"/"super_debugging_path"/"super_debugging_skill_path" 切换到 "debug_skill"/"debug_skill_path"；`go test ./internal/prompt/...` 全部通过
- KR4: `internal/executor/runner.go` 的重试上下文加载逻辑从仅读 `debug.md` 扩展为同时扫描 `debug/` 目录下所有 `bug*.md` 文件；`go test ./internal/executor/...` 全部通过


### debug.md（执行问题记录，已内嵌）

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

## debug1: TestExecuteDoingWorkflow 测试超时（task4 测试脚本 go test ./internal/... 超时）

**现象 (Phenomenon)**:
- `go test ./internal/...` 在 180s 内超时，`TestExecuteDoingWorkflow_ResumesFromTasksJSON` 和 `TestExecuteDoingWorkflow_WithMockClaude` 挂起

**复现 (Reproduction)**:
- `go test -timeout 60s ./internal/cmd/...` → 60s 后 FAIL，stack trace 卡在 `retry.go:123 time.Sleep`

**猜想 (Hypothesis)**:
- `~/.rick/config.json` 设置了 `max_retries: 16`，导致 retry sleep 累计 = 1+2+...+15 = 120s

**验证 (Verification)**:
- `cat ~/.rick/config.json` → `max_retries: 16` ✅ 确认

**修复 (Fix)**:
- 两个测试函数开头加 `t.Setenv("HOME", dir)` + 写入 `dir/.rick/config.json`（`{"max_retries":2}`）
- 测试从挂起 60s+ 降至 ~1s 完成

**进展 (Progress)**:
- 当前状态：✅ 已解决


### 参考资料路径（按需读取）

- **SPEC.md**: `/Users/sunquan/ai_coding/CODING/rick/.rick/SPEC.md`
- **任务详情**（task*.md）:
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/plan/task1.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/plan/task2.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/plan/task3.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/plan/task4.md`
- **执行轨迹**（act-path.md）:
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/tasks/task1/act-path.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/tasks/task2/act-path.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/tasks/task3/act-path.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/tasks/task4/act-path.md`

### 任务执行结果

| Task ID | 任务名称 | 状态 | Commit Hash | 重试次数 |
|---------|---------|------|-------------|----------|
| task1 | 创建 debug_skill.md（三阶段调试 SOP + review debug agent 协议） | success | db57d56f | 0 |
| task2 | 更新 doing.md / plan.md 模板引用，删除 super-debugging-zh.md | success | b8758693 | 0 |
| task3 | 更新 Go prompt 文件，将 super-debugging-zh 切换到 debug_skill | success | eab9121b | 0 |
| task4 | 全局替换 debug.md 读取：7 处上下文加载改为 debug/ 优先、回退 debug.md 的兼容策略 | failed | N/A | 0 |


---

## ⚠️ 必须严格按以下 7 步 SOP 执行

### Step 1：分析执行记录（必须完成，不可跳过）

**1a. 分析 debug.md**（内容已内嵌在上方，硬约束，SUMMARY.md 生成前必须完成）

分析上方"debug.md（执行问题记录）"内容：
- 每个 debug 条目的根因与解决过程
- 未解决的问题（进展状态为"未解决"的条目）

**1b. 还原完整执行轨迹**

读取上方列出的所有 act-path.md 文件，按任务顺序还原本次 job 的完整执行轨迹：
- 每个 task 的工具调用序列
- 报错次数与修复路径
- 执行耗时与关键决策点

输出格式：逐 task 列出轨迹摘要（1-3 句）。

---

### Step 2：评估更合理的 act-path

针对每个 task，评估：
1. 是否存在冗余工具调用（可合并或省略）？
2. 是否存在可预防的错误（通过前置检查或更好的顺序）？
3. 是否有更短的执行路径能达到同样目标？

为路径最长或报错最多的 1-2 个 task 输出改进建议。

---

### Step 3：提取 Tools

**YOU MUST declare: "I will use skill:gen-skill." Before writing any skill proposal.**

技能文件：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/learning/prompts/skill_gen_skill.md`

从 act-path 和 debug 中识别可复用模式，**优先判断**哪些逻辑值得提取为独立 Python 工具：
- ✅ 纯函数：确定性输入输出，无副作用
- ✅ 跨 task / 跨 job 通用
- ✅ 支持 `--test` 自验证

直接写入：`/Users/sunquan/ai_coding/CODING/rick/.rick/tools/*.py`

---

### Step 4：沉淀 Skills（wiki 文档）

基于 Step 3 识别出的 tools，为每个可复用模式生成 wiki 文档：

**wiki 文档格式**：触发场景 / 预期效果 / 使用方法

- **触发场景**：何时使用（具体信号词）
- **预期效果**：可量化的结果
- **使用方法**：
  - 有对应 tool → 只写工具路径 + 调用示例，**禁止内联实现代码**
  - 无对应 tool → 可写简短伪代码说明思路

直接写入：`/Users/sunquan/ai_coding/CODING/rick/.rick/wiki/*.md`

**原则：tools 承载 how，wiki 描述 what/when/why，不重复实现。**

---

### Step 5：更新 SPEC.md

直接更新 `/Users/sunquan/ai_coding/CODING/rick/.rick/SPEC.md`（in-place，无需生成副本）。

#### 5a. 将 Step 4 所有 wiki 文档注册到技能列表

**每一个 wiki 文档都必须在 `## 技能列表` 中有对应条目**，格式：

```markdown
| 名称 | 触发词 | 路径 |
|------|--------|------|
| rick-test-isolation | plan_check 错误被自动修复 | .rick/wiki/rick_test_isolation.md |
```

#### 5b. SPEC 内容瘦身（渐进式披露）

若 SPEC.md 某节内容过长（详细步骤、示例、背景说明），将其迁移到 wiki，SPEC 只保留一行摘要 + 链接：

```markdown
## 编译与运行方法

详见 → [编译与运行指南](wiki/build_and_run.md)
```

**原则：SPEC ≤ 512 行；超出部分卸载到 wiki，SPEC 保留入口链接。**

---

### Step 6：生成 SUMMARY.md

**⚠️ 前置检查**：确认已完成 Step 1a（分析 debug.md 内容）。未完成 Step 1a 禁止生成 SUMMARY.md。

在 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/learning` 下生成 `SUMMARY.md`：

```markdown
APPROVED: true

# Job job_16 执行总结

## 执行概述

**项目目标**: ...
**实际完成**: ...
**整体评价**: ⭐⭐⭐⭐⭐ (1-5 星)

## 关键成就

1. **成就1**: 描述和意义

## 问题与教训

### 问题1: 问题描述

**根本原因**: ...
**解决方案**: ...
**经验教训**: ...

## 知识沉淀清单

- [ ] skills/xxx.md - 技能描述
- [ ] SPEC.md - 变更说明（如有）
```

---

### Step 7：运行 learning_check 验证 SUMMARY.md

```bash
/Users/sunquan/ai_coding/CODING/rick/bin/rick tools learning_check job_16
```

失败则修复后重新运行，直至通过。

---

## ⚠️ 重要约束

1. **debug.md 内容已内嵌，必须在 SUMMARY.md 之前分析**：Step 1a 是硬约束，不可跳过
2. **Step 3 必须声明使用 gen-skill**：`"I will use skill:gen-skill."` 是硬约束
3. **wiki/tools/SPEC 直接写入 `.rick/`**：不要写到 learning 子目录再合并，直接操作 `/Users/sunquan/ai_coding/CODING/rick/.rick/wiki`、`/Users/sunquan/ai_coding/CODING/rick/.rick/tools`、`/Users/sunquan/ai_coding/CODING/rick/.rick/SPEC.md`
4. **SUMMARY.md 写入 learning 目录**：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/learning/SUMMARY.md` 作为本次执行记录
