# Rick 项目执行阶段

## 角色定义

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

---

## 先验知识（执行前必读）

## 可用的项目 Loops

- **agent-runtime-bootstrap-loop**："当需要初始化/迁移/重装 rick 的 agent runtime（pi）及其扩展时触发（如 rick tools init-pi、版本升级、runtime 迁移）"
- **protocol-redesign-loop**："当需要重构 AI agent 的多阶段协议(如人机协作流程、任务执行流程),涉及阶段合并/拆分/反向回流/批判层重设计时触发"
- **readme-wiki-sync-loop**："当需要编写或重构 README.md、wiki/（用户面向文档）等描述性资料时触发"
- **tdd-red-green-refactor-loop**："当测试已存在且处于 FAIL 状态，需要通过迭代实现让其变绿时触发（前提：先写测试、当前测试 FAIL、目标是收敛到绿）"


## 可用的项目 Skills

- **check-mechanism**：learning_check / dream_check 命令失败，需要理解失败原因或扩展新检查规则时使用。（doing 门禁已下沉为 rick-gates 确定性脚本，plan_check/doing_check 已删除。）
- **command-registration-verification**：在文档（README、commands.md、学习文档等）中引用项目自身的 CLI 命令、flags、子命令关系时使用
- **dag-task-decomposition**：plan 阶段将复杂需求分解为多个相互依赖的 task 时使用，特别是
- **failure-feedback**：doing 阶段 task 失败重试时，需要理解或调整失败信息如何传递给下一轮 Agent 时使用。
- **fake-binary-script**：当 Go/Python 测试中用**假的可执行脚本**（fake pi、fake node 等）模拟真实二进制时使用
- **global-ref-sync**：修改一个在多个文件中被引用的核心名称/变量时
- **mark-task-success**：doing task 代码已提交（有 commit hash）但 rick-gates 门禁（helper.py）报错
- ****：当通过 `pi install npm:<pkg>` 或 `pi install <local-path>` 安装 pi 扩展（如 subagent、web-access、主题包）后使用。
- ****：当 rick 迁移或更换底层 agent runtime（如 claude code → pi，或 pi 版本大升级），且改动涉及 rick 调用 agent 的 CLI flags / 事件解析 / prompt 落盘路径时使用。
- **pi-theme-verification**：当需要验证/定制 pi 主题时使用
- **subprocess-env-isolation**：当集成测试中通过 subprocess 调用 rick CLI，测试本地通过但行为与预期不符时
- **template-injection**：需要在 `rick plan` 或 `rick easy` 会话中嵌入新的结构化行为时
- **test-script-practices**：在 plan 或 doing 阶段编写/调试任务测试脚本（`.rick/jobs/job_N/doing/tests/taskN.py`）时使用，特别是
- **verify-go-changes**：修改了 Go 源文件后，需要验证编译通过、单元测试和集成测试通过时使用。
- **zero-retry-task-design**：plan 阶段分解需求为多个 task.md 时使用，目标是让每个 task 在 doing 阶段一次性完成，无需重试。


---

## Job 上下文

暂无问题记录

---

## 任务信息

**任务 ID**: task12
**任务名称**: 三个 O 端到端验收 + README/wiki 文档同步

### 任务目标
以用户视角做端到端验收，确认三个 O 全部落地，并同步用户面向文档（README.md + wiki/，不修改 `.rick/domain/`）。

三个 O 验收清单（收敛版）：
- O1（spec 信息内核）：`.rick/domain/spec.md` + `.rick/domain/rick-spec.md` 存在且含四要素 + 验收标准
- O2（三层金字塔 + 做薄）：`internal/{cmd,handler,env,builder,runtime}` 存在且职责与 spec 对应；`rick` 全命令可用；已删除 `internal/{executor,parser,actpath,logging,git,agent}`
- O3（pibuilder pi 对齐）：模板中 `workflowScript`/`runs.run` 出现 >0；think/research/exporter 落盘为 pi agent；门禁由 rick-gates hook + 脚本承载；自然语言触发词下降

参考：loop `readme-wiki-sync-loop`（README/wiki 同步，**禁止修改 .rick/domain/**）；skill `verify_go_changes_skill`、`command_registration_verification_skill`、`check_mechanism_skill`（仅 learning_check 相关）；domain/testing-conventions.md「go test 范围精确性」。

### 关键结果
1. 三 O 验收逐项通过：
2. O1：`test -f .rick/domain/spec.md && test -f .rick/domain/rick-spec.md`；`for w in 模块边界 职责 接口契约 验收标准; do grep -q "$w" .rick/domain/rick-spec.md || exit 1; done`；`grep -q '功能等价' .rick/domain/rick-spec.md`；`grep -qE 'dry-run|go test|集成测试' .rick/domain/spec.md`（可操作判据被枚举）
3. O2：`ls internal/cmd internal/handler internal/env internal/builder internal/runtime` 均存在；`for d in executor parser actpath logging git agent; do test ! -d internal/$d || exit 1; done`（6 冗余包全删）
4. O3：`grep -rc workflowScript internal/prompt/templates/ | grep -v ':0' | wc -l` ≥1；显式 `export RICK_PI_AGENT_DIR=<temp>`（或改用 `~/.rick/pi/agent` 默认路径）后，`$RICK_PI_AGENT_DIR/agents/{think,research,exporter}.md` 3 文件存在 + `$RICK_PI_AGENT_DIR/extensions/rick-gates/` 已部署；`test -f .rick/skills/rick-gates/helper.py`；自然语言触发词计数下降（正则口径与 RFC §2.1 基线一致：含 bare `subagent`/`Sub Agent`/`SPAWN`/`子 Agent`，迁移前后用同一正则对比，迁移后 < 迁移前基线）
5. 全命令可用：`./bin/rick --help` + 8 子命令 `--help` 无 panic；`plan --dry-run`/`doing job_35 --dry-run`/`human-loop --dry-run`/`easy --dry-run`/`learning --dry-run`/`dream --dry-run`/`ctrl --dry-run --job job_N` 含 pi 触发语法且无 `{{`
6. 测试全绿：`go build ./...`（整仓编译兜底）+ `go test ./internal/config/... ./internal/env/... ./internal/runtime/... ./internal/builder/... ./internal/prompt/... ./internal/handler/... ./internal/cmd/... ./internal/workspace/... -timeout 60s`；**同步改写 `tests/tools_integration_test.sh`**（task8 已删 plan_check/doing_check，该脚本场景 1,2,3,4,5,6,11 依赖被删命令需删除或改写为 pi 侧门禁脚本验证；场景 7 只删 merge/branch 断言、保留 learning_check 断言）后再跑，并与 mock_agent 对齐
7. README.md + wiki/ 同步三层金字塔 + spec 信息内核 + env 四职责 + 下沉策略（只读引用 `.rick/domain/`，不写）；`git status .rick/domain/` 无本 task 引入的变更
8. 扩展点验收：`grep -q 'type Runtime interface' internal/runtime/*.go`、`grep -q 'type RuntimeEnv interface' internal/env/*.go`、`grep -q 'type RuntimeBuilder interface' internal/builder/*.go` 三接口就位；`grep -rniE 'type .*dsh|dshRuntime|dshEnv|dshBuilder|NewDsh' internal/ cmd/` = 0（无 dsh 类型/实现/构造；代码注释中的「dsh」豁免）；`grep -rn 'piRuntime' internal/handler/` 无命中（handler 依赖接口非具体实现）；`grep -q '"runtime"' internal/config/config.go` 字段就位
9. 真实运行验收（**硬门，功能等价的核心行为验收，不可 skip**）：`rick tools init-pi` 全 ✅ → `rick doing job_N` 按依赖执行 → `doing/tasks.json` 状态机正确、`doing/session_id` 落盘、门禁脚本在 `agent_settled` 后执行；无 pi/API key 环境**须由 supervisor 手动执行并记录结果**（结构 grep 不能替代行为等价）


### 测试方法
正常路径（三 O 验收）：前置条件 = task8~11 完成 + `rick tools init-pi` 已成功；输入 = 无；操作 = 依次执行 KR1 的三组断言（O1 两个 spec 文件 + 四要素关键词 + 功能等价；O2 5 目录存在 + 6 冗余包 `test ! -d`；O3 workflowScript 计数 ≥1 + 3 个 agent 文件 + rick-gates helper.py）；预期 = 全部通过。
边界（命令全可用）：前置条件 = build 成功；输入 = 各子命令 `--help`；操作 = `./bin/rick --help` + 8 个子命令 `--help`；预期 = 均 exit 0、无 panic、含 plan/doing/easy/learning/dream/tools/human-loop/ctrl。
异常（回滚兜底）：前置条件 = 某命令行为异常；输入 = 无；操作 = 记录当前 release commit（`git log --oneline -1`）后，在**副本 worktree** 验证回滚：`git worktree add /tmp/rick-rollback <release-commit>` + `cd /tmp/rick-rollback && ./scripts/build.sh && ./bin/rick --help`（exit 0）；预期 = 可回滚且 rick 仍可编译运行（不污染主工作区；`.rick/domain/` 无本 task 变更）。注意 task8 已删 `plan_check`/`doing_check` 命令，回滚验证改用 `./bin/rick --help`。




---

**你需要一步步执行以下操作，不可跳过任何步骤。**



## 第一步：执行 Doing Loop

# Doing Loop

> ⚠️ 以下是默认 loop 的执行步骤，也是 gen-loop 需要参考的 skill 模板！！

---

## Step 0：Domain 搜索 + Loop 匹配

**必须依次完成以下两项，再进入 Step 1：**

### 0.1 搜索 Domain（强制）

根据澄清的需求，读取 `/workdir/sunquan20/AI_CODING/rick/.rick/domain` 下的相关文件，获取足够的事实信息（环境配置、已知问题、接口约束、构建命令等），建立解决问题的基本视角。

- 由 AI 自行判断读取哪些文件，但**必须完成搜索动作**后再继续
- 遇到任何问题（编译报错 / 测试失败 / 行为异常），**必须优先搜索 `/workdir/sunquan20/AI_CODING/rick/.rick/domain/bugs.md` 和 `/workdir/sunquan20/AI_CODING/rick/.rick/domain/`**，再做其他尝试

### 0.2 匹配 Loop

在 Domain 搜索完毕后，读取 `loops_context`，按 trigger 字段匹配当前任务/需求：

- **有匹配** → 读取对应 Loop 文件，按其定义步骤执行（不再执行以下 Step 1–5）
- **无匹配** → 按以下 Step 1–5 执行默认 Loop

---

## Step 1：Main Agent 确认全局目标

确认以下内容全部清晰后才继续：

- task.md 中 `# 任务目标` 和 `# 关键结果` 已理解
- 成功标准已明确：测试脚本全通过 + check pass + 所有 Key Results 达成

---

## Step 2：Main Agent 读取上下文（压缩策略）

从 `doing/debug/` 目录读取已有信息，按以下方式压缩后传递给 Sub Agent：

- **bug\*.md** → 从每个文件的 frontmatter `summary` 字段提取摘要，避免重复踩坑
- **跨轮核心事实** → 任务目标 + Key Results 达成状态 + debug/ 摘要 + 当前迭代编号 N

---

## Step 3：启动 Sub Agent 执行工作流

**每轮迭代由 Main Agent 启动一个独立 Sub Agent，携带 Step 2 的上下文，执行完整工作流后返回产出摘要。**

```
[Main Agent]
   │
   ├─ SPAWN Sub Agent（携带：任务目标 + debug/摘要 + 迭代编号 N）
   │     │
   │     │  Sub Agent 执行：
   │     │  [ANALYZE] → [RED] → [GREEN] → [REFACTOR] → [COMMIT]
   │     │                 ↑        │
   │     │                 └──[DEBUG]┘
   │     │
   │     └─ Sub Agent 完成，输出产出摘要
   │
   └─ Main Agent 执行 Step 4 产出评估
```

### Sub Agent：ANALYZE（理解需求）
1. 声明：`"I will use skill:sense."`，按 S→E→N 分析（Symptoms / Evidence / Next）
2. 读取 debug/ 摘要，避免重复踩坑

### Sub Agent：RED（先写失败测试）
1. 声明：`"I will use skill:tdd for implementation."`
2. 针对 `# 测试方法` 中每个场景编写测试
3. 运行测试，**必须确认 FAIL**（证明测试有效，进入 GREEN 的前提）

### Sub Agent：GREEN（最小实现）
1. 编写让测试通过的最小实现代码（不超出 task scope）
2. 通过 → REFACTOR；失败 → DEBUG

### Sub Agent：DEBUG（遇红强制触发）

触发条件（任意一条）：测试 FAIL / 编译报错 / 行为与预期不符

1. **优先搜索 `/workdir/sunquan20/AI_CODING/rick/.rick/domain/bugs.md` 和 `/workdir/sunquan20/AI_CODING/rick/.rick/domain/`**，查看是否有精确解决方案
   - 有匹配 → 直接应用，记录引用来源
   - 无匹配 → 继续下方流程
2. 声明：`"I will use skill:debug-skill."`，加载 skill 文件：`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/prompts/skill_debug_skill.md`
3. 在 `doing/debug/` 下创建 `bug{N}-{描述}.md`，按 Phase 1-6 执行
4. Phase 4 上限 3 次，达上限后输出当前状态并升级人工协作
5. 修复后回到 GREEN

### Sub Agent：REFACTOR（代码改善）
1. 测试全绿后改善代码质量（命名、结构、去重）
2. 运行全量测试确认无回归；回归失败 → DEBUG

### Sub Agent：COMMIT（收尾提交）
1. `git add` + `git commit`（commit message 含 task ID）
2. 运行 check 命令（使用 prompt 上下文中的 rick_bin_path 和 job_id）：
   - doing 阶段：`<rick_bin_path> tools doing_check <job_id>`
   - easy 阶段：`<rick_bin_path> tools easy_check <job_id>`
3. check 失败 → 修复后重新运行，循环直到 pass
4. **Sub Agent 完成**：输出本轮产出摘要（完成了哪些 KR、遗留了哪些问题），通知 Main Agent 执行 Step 4

---

## Step 4：Main Agent 产出评估

Sub Agent 完成后，Main Agent 逐项检查：

| 检查项 | 判断方法 |
|--------|----------|
| check pass | 读取 doing_check / easy_check 输出，确认 ✅ |
| 测试全通过 | 确认测试脚本无 FAIL 输出 |
| Key Results 达成 | 逐条比对 task.md `# 关键结果` |

- **全部通过** → 进入 Step 5
- **存在失败** → 将失败原因附加到上下文，返回 Step 3 启动下一轮迭代

---

## Step 5：Main Agent 确认停止标准

**成功退出**：check pass + 测试全通过 + 所有 Key Results 达成

**优雅退出**（任意一条触发）：
- 迭代次数达上限（默认 **3 轮**）
- 连续 2 轮产出相同错误（判断无法自动收敛）
- 人类明确要求停止

**退出时**：Main Agent 输出 Loop 执行摘要（完成了哪些 KR、遗留了哪些问题），等待人类决策。





---

## 第二步：格式检查

`/workdir/sunquan20/AI_CODING/rick/bin/rick tools doing_check job_35`

check pass 后才算完成。




## Test Execution Feedback

**Previous test execution encountered errors. You may need to fix the test script.**

```
=== Attempt 1 ===
doing_check failed: tasks.json not found or invalid: failed to unmarshal tasks.json: parsing time "2026-08-17T06:01:04.789297" as "2006-01-02T15:04:05Z07:00": cannot parse "" as "Z07:00"

```
