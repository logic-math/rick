# Rick 项目执行阶段

## 角色定义

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

---

## 先验知识（执行前必读）

## 可用的项目 Loops

- **agent-runtime-bootstrap-loop**："当需要初始化/迁移/重装 rick 的 agent runtime（pi）及其扩展时触发（如 rick tools init-pi、版本升级、runtime 迁移）"
- **do-check-mark-success-loop**："当 doing_check 报错 task status != success 或 commit_hash 缺失时触发"
- **protocol-redesign-loop**："当需要重构 AI agent 的多阶段协议(如人机协作流程、任务执行流程),涉及阶段合并/拆分/反向回流/批判层重设计时触发"
- **readme-wiki-sync-loop**："当需要编写或重构 README.md、wiki/（用户面向文档）等描述性资料时触发"
- **tdd-red-green-refactor-loop**："当测试已存在且处于 FAIL 状态，需要通过迭代实现让其变绿时触发（前提：先写测试、当前测试 FAIL、目标是收敛到绿）"


## 可用的项目 Skills

- **check-mechanism**：plan/doing/learning_check 命令失败，需要理解失败原因或扩展新检查规则时使用。
- **command-registration-verification**：在文档（README、commands.md、学习文档等）中引用项目自身的 CLI 命令、flags、子命令关系时使用
- **dag-task-decomposition**：plan 阶段将复杂需求分解为多个相互依赖的 task 时使用，特别是
- **failure-feedback**：doing 阶段 task 失败重试时，需要理解或调整失败信息如何传递给下一轮 Agent 时使用。
- **fake-binary-script**：当 Go/Python 测试中用**假的可执行脚本**（fake pi、fake node 等）模拟真实二进制时使用
- **global-ref-sync**：修改一个在多个文件中被引用的核心名称/变量时
- **mark-task-success**：doing task 代码已提交（有 commit hash）但 doing_check 报错
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

**任务 ID**: task4
**任务名称**: 重构 runtime 模块（pi 调用逻辑收口到 runtime 层）

### 任务目标
按 spec（task2）落地 KR2.4：把 pi 调用逻辑（参数解析 + 调用）收口到 runtime 层。将 `internal/agent/piagent` 迁移为 `internal/runtime`（pi 调用封装）。

**本 task 只做「包迁移 + session 就绪判定显式化」，不改 `Execute` 签名、不删 `internal/agent` 接口、不删 `internal/actpath`**——这些由 task8（做薄 cutover）与删 executor 同批完成；否则 executor（仍引用 agent/actpath/prompt）会编译断裂。

迁移内容（均在 `internal/agent/piagent/`）：`cli.go`（FindBinary/piPathOrDefault/buildArgs/CallCLI/mergeExtraArgs/CLIMode）、`executor.go`（Executor/parseStream/piEvent/piMessage/piSession）、`agentdir.go`（AgentDir/SettingsPath/RuntimeDir/RuntimeBin/FileExists/EnsureAgentDir/AgentEnv）。`internal/runtime` **继续实现 `internal/agent` 的 AgentExecutor/AgentSession**（保持 executor 可编译）。

session 就绪判定显式化：`Executor` 已从 pi JSONL 解析 session header 提取 sessionID、以 `agent_settled` 为终止信号；本 task 抽出「sessionID 非空 && settled」的判定函数并补单测（fake JSONL：有/无 `agent_settled` 两种）。注意：当前 `parseStream` 缺 `agent_settled` 时**不报错**只回退计时（`executor_test.go` 有对应断言），本 task **不改变该行为、不把 `isSessionReady` 接入 `Execute`**（只定义函数 + 单测，Execute 仍返回原行为）。

**扩展 seam（为将来 dsh runtime 留扩展位）**：`internal/runtime` 定义 `Runtime` 接口 `{ Name() string; Run(methodText string, promptFile string, cfg *config.Config) (sessionID string, trace *Trace, err error) }`——`methodText` 走 `--append-system-prompt` 注入（系统提示词、会话前注入），`promptFile` 走 user prompt（实例上下文）；`piRuntime` 为实现——handler 依赖此接口而非具体 piRuntime，将来新增 dsh 只需加 `dshRuntime` 实现并注册；`Run` 最终签名在 task8 与 `Execute→Run` 切换同批落地，本 task 先定义接口 + `piRuntime` 骨架（接口签名仅占位，以 task8 为准；同时保留 AgentExecutor 兼容 executor）。config 增加 `runtime` 字段（默认 `"pi"`），handler/env/builder 按它选实现。

参考：domain/architecture.md「DIP 组合根」；skill `verify_go_changes_skill`、`pi_runtime_verification_skill`、`fake_binary_script_skill`、`subprocess_env_isolation_skill`、`global_ref_sync_skill`；bugs.md「pi --session vs --session-id」「pi 解析器 message_end role==assistant」「托管运行时优先 PATH-fake 命中真实 pi」。

### 关键结果
1. 新建 `internal/runtime` 包，迁移 piagent 全部调用/解析逻辑；继续实现 `internal/agent` 接口（AgentExecutor/AgentSession），executor 编译不破
2. 抽出 session 就绪判定函数（`isSessionReady`：sessionID 非空 && settled），单测覆盖有/无 `agent_settled` 两种 fake JSONL
3. 所有 `internal/agent/piagent` import 改为 `internal/runtime`；`internal/agent` 接口 + `internal/actpath` **保留**（task8 删）
4. 迁移测试全绿：cli/executor/agentdir 测试随包迁移（改 package/import 路径）；真实 pi 冒烟测试（realpi/realds）在无 pi 时跳过
5. `go build` + `go test ./internal/runtime/... ./internal/cmd/... -timeout 60s` 全绿
6. 定义 `Runtime` 接口（`Name`/`Run(methodText, promptFile, cfg)`）+ `piRuntime` 骨架 + config `runtime` 字段（默认 `"pi"`）+ `Trace` 结构体（sessionID/toolCalls/finalMessage/rawLogPath/duration/settled，等价承载原 act-path + session 信息）；`Run` 启动 pi 时把 `methodText` **落盘临时文件、经 `--append-system-prompt <method文件路径>` 注入**（会话前注入系统提示词，保留 pi 默认骨架，避免长文本 inline 传参；**临时文件由 runtime 创建并 `defer` 清理、用完即删**），`promptFile` 作为 user prompt；**`CallCLI` 同样经 extraArgs 支持 `--append-system-prompt <methodFile>` 注入（交互命令 plan/easy/human-loop/ctrl 也注入 method）**；handler 依赖 `Runtime` 接口，组合根按 config 选实现（dsh 扩展位）；「`runtime` 空值 → `"pi"`」归一化落在 `LoadConfig`（`json.Unmarshal` 不回填默认值，补单测），**并同步在 `GetDefaultConfig()` 加 `Runtime:"pi"`（覆盖「无 config 文件」分支）**；**组合根落点 = `internal/cmd` 根命令的 RunE 内懒加载**（每次命令执行时读 `config.runtime` 实例化 piRuntime/piEnv/pibuilder 注入 handler，task6/7 落地）——**禁止在 `NewRootCmd`/`NewXxxCmd` 构造期 `LoadConfig`**（会触发 `~/.rick/config.json` 落盘副作用，破坏 `--help`/`--version`/测试）


### 测试方法
正常路径：前置条件 = task2 完成；输入 = 无；操作 = `go build -o bin/rick ./cmd/rick && go test ./internal/runtime/... -v`；预期 = build 成功，runtime 测试全绿（cli/executor/agentdir 迁移后仍绿）。
边界（session 就绪判定 + config 默认值）：前置条件 = 用 fake pi 输出一段含 `{"type":"session","id":"s123"}` + `{"type":"agent_settled"}` 的 JSONL；输入 = 该 JSONL；操作 = 调 `isSessionReady` 检查返回；预期 = 有 `agent_settled` 返回 true；去掉 `agent_settled` 返回 false（本 task 不改变 Execute 不报错行为）。另 `TestLoadConfig_RuntimeDefault`：空 config → `cfg.Runtime == "pi"`（归一化）。
异常（pi 缺失 + 接口保留）：前置条件 = `RICK_PI_AGENT_DIR` 指向空 temp 且 PATH 无 pi；输入 = `runtime.FindBinary(nil)`；操作 = 调用检查 error；预期 = 返回 error 含 `pi binary not found`。另 `grep -rn "internal/agent/piagent" internal/` 无残留（但 `internal/agent` 接口仍存在）。




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


