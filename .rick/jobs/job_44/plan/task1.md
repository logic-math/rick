# task1：human-loop v4 协议重构（教师会话）

## 任务目标

把 human-loop 从「机械五阶段引导（S/E/N1/N2/S-R/EC + 系统论描述符 + 三维打分 + 批判门禁）」重构为「**教师会话**」：单个主体提示词调度全局节奏，research 沉淀事实材料，teach 做 SENSE 思考并产出教学课程；教学叙事内化 SENSE（不标阶段、不讲方法论），引导 human 高质量原创判断。exporter 全链删除，不再产 RFC。

## 关键结果

- **KR1a 主体提示词重写** `internal/prompt/templates/sense_loop.md`：文件状态机驱动（`mission.md` 缺 → 面试 human 定**学习目标 + 考核标准 + 起点**；事实材料缺 → 派 research；每回合派 teach 产教学节 → 呈现 human → 记录 judgment）；认知强制（呈现分析前先请 human 表态直觉）；退出准则（mission 目标达成 + human 明示完成）；**删除**阶段序列/阶段标签/固定追问集/硬性跳转约束/回流计数器/系统论描述符/三维打分/批判门禁重试/exporter 派发/RFC 产出
- **KR1b teach 新建** `internal/prompt/templates/teach.md` + `internal/prompt/templates/skills/teach.md`：SENSE 五关注面**仅概念本质**（还原事实/呈现视角/洞察矛盾/启发逆转/良质批判）+ 教学纪律（一课一事 + 一次有形收益 + working memory 内；知识阶段难度是敌人；每课引注事实材料并荐一个 primary source；**隐式检查 human 判断（隐式 quiz）**；提问引导 human 做 SENSE 思考）+ 产出 `lessons/teach_{N}.md`
- **KR1c research 调整** `internal/prompt/templates/research.md` + `skills/research.md`：输出 `research/research_{N}.md`（尽调树 + 信源等级 A/B/C + `## Gaps`），保留信源加权方法论
- **KR1d 删除** `templates/think.md`、`templates/skills/think.md`、`templates/exporter.md`、`templates/skills/exporter.md`
- **KR1e agent 注册** `internal/env/agents.go`：`think` spec → `teach` spec（description 重写为「SENSE 思考 + 教学课程产出」）；删除 `exporter` spec
- **KR1f prompt 生成链** `internal/prompt/human_loop_prompt.go`：变量 `think_agent_path`→`teach_agent_path`、删 `exporter_agent_path`；`WriteSkillFile` 写 `skill_teach.md`（不再写 skill_think/skill_exporter）；`LoadTemplate("teach")`；`GenerateHumanLoopPromptFile` 返回值 4 文件（main/think/research/exporter）→ **3 文件（main/teach/research）**；主 prompt 变量注入学习材料路径（mission/research/lessons/judgment）
- **KR1g 调用链适配** `internal/builder/pibuilder.go`（`SaveHumanLoopPrompt` 签名）+ `internal/handler/human_loop.go`（返回值适配）
- **KR1h 测试适配**：`internal/prompt/human_loop_prompt_test.go`、`internal/env/env_test.go`、`internal/env/update_test.go` 及编译报错的其它测试全绿
- **KR1i 版本号** `cmd/rick/main.go`：`VERSION` 5.0.13 → 5.0.14

# 写域

- internal/prompt/templates/sense_loop.md
- internal/prompt/templates/teach.md（新建）
- internal/prompt/templates/think.md（删除）
- internal/prompt/templates/exporter.md（删除）
- internal/prompt/templates/research.md
- internal/prompt/templates/skills/teach.md（新建）
- internal/prompt/templates/skills/think.md（删除）
- internal/prompt/templates/skills/exporter.md（删除）
- internal/prompt/templates/skills/research.md
- internal/prompt/manager.go（go:embed 指令与 LoadTemplate 分支：think→teach、删 exporter）
- internal/prompt/human_loop_prompt.go
- internal/prompt/human_loop_prompt_test.go
- internal/builder/pibuilder.go
- internal/builder/（其它因签名变更需适配处 + 测试）
- internal/handler/human_loop.go
- internal/env/agents.go
- internal/env/env_test.go
- internal/env/update_test.go
- cmd/rick/main.go（仅 VERSION 一行）
- internal/web/sessions.go（**仅编译适配**：SaveHumanLoopPrompt 返回值解构 `mainFile, _, _, method, err :=` 一行；零行为变更——parent 批准的范围例外，2026-10-05）

# 依赖关系

- 无（首层；task2/task3 与本 task 写域不相交，并行）

## 实现要点

- **产物布局**（模板内写死路径约定）：`{{loop_dir}}/mission.md`、`{{loop_dir}}/research/research_{N}.md`、`{{loop_dir}}/lessons/teach_{N}.md`、`{{loop_dir}}/judgment.md`、`{{loop_dir}}/prompts/`
- **派发语法**：主体 prompt 内保留显式语法 `subagent({ workflowScript: "return runs.run('research-<轮>', { agent: 'research', task: '...' })", timeoutMs: 3600000 })`；teach 同理（串行：research → teach）
- **交付门禁**：保留「回执 + 文件校验（`wc -c` + `grep -c '^## '`）」纪律，写入主体 prompt
- **judgment.md**：仅写 human 原话，禁写 AI 推理（保留）
- **删除 `{{max_retries}}`/`{{max_backflows}}` 等已废变量注入**（config 字段本体保留不删——无其它消费者，避免扩大爆炸半径；新模板不再引用）
- prepareHumanLoopDirs 逻辑保留（draft/rfc/concepts/human-learning/loops 目录；rfc 目录留给历史 RFC 展示）
- 旧 loop 目录（briefs/teach-*/judgment.md）不做迁移——humans 页面通用树原样展示

## 测试方法

- `go build ./...` 绿
- `go test ./internal/prompt/... ./internal/env/... ./internal/builder/... ./internal/handler/...` 绿
- `go build -o /tmp/rick-v4 ./cmd/rick && cd $(mktemp -d) && /tmp/rick-v4 human-loop --dry-run '自检主题'`：
  - 输出含 `teach`（agent 路径 + 角色）与 `research`，**不含** `exporter`
  - 输出含 `mission.md`、`lessons/`、`research/`、`judgment.md` 路径约定
  - 输出**不含** 阶段机程序列（`阶段 1:S`、`N1`、`反向回流`、`三维打分`、`系统论描述符`）
  - 无未替换变量残留（不含 `{{`）
- `internal/prompt/templates/skills/teach.md` 含五个概念名（还原事实/呈现视角/洞察矛盾/启发逆转/良质批判）与各自「本质」说明；不含方法论章节（怎么还原/呈现/洞察/逆转/批判 的步骤化方法）
- 全链 exporter 痕迹清除：`grep -rn exporter internal/prompt internal/env internal/builder internal/handler` 无输出（排除 *_test.go 中显式断言「exporter 不存在」的场景——若测试需断言则应改写到 gate，不在本包测试中断言字符串）

## 禁止事项

- 不得修改 `internal/web/**`、`web/src/**`（task2/task3 写域）
- 不得删除 config 字段（ThinkTopN/ThinkMinAssumptions/SenseMaxBackflows）——保留向后兼容
- 不得保留任何 exporter/think agent 定义或模板引用
- 不得把「方法论」写回 teach/sense_loop（human 明确要求只讲概念本质）
