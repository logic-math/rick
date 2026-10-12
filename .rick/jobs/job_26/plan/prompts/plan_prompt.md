# Rick Plan 阶段

你的任务：**将用户需求转化为可落地执行的任务列表**。

---

## 一、项目上下文

### 项目 Domain（事实性知识库）

路径：`/Users/sunquan/ai_coding/CODING/rick/.rick/domain`

> Domain 存储项目的客观事实：环境配置、已知问题与解法、接口规范、构建命令等。规划前必须优先读取。

### 项目 Loops（已有工作流模式）

## 可用的项目 Loops

- **do-check-mark-success-loop**："当 doing_check 报错 task status != success 或 commit_hash 缺失时触发"
- **tdd-red-green-refactor-loop**："当测试已存在且处于 FAIL 状态，需要通过迭代实现让其变绿时触发（前提：先写测试、当前测试 FAIL、目标是收敛到绿）"


---

## 二、项目探索

如果需要理解当前代码状态，请自行探索项目。使用 Read / Grep / Glob / Bash 等工具，理解：
- 项目目标（从 README、代码注释、测试用例推断）
- 技术栈与架构（从依赖文件、目录结构推断）
- 当前状态与待解决的问题

探索完成后，向用户确认你的理解，再开始任务规划。

---

## 三、任务分解原则

1. **模块化**：每个 task 是独立功能单元，可独立开发和验证
2. **粒度**：单个任务工作量 0.5–2 天；太大则拆分，太小则合并
3. **可验证**：每个 task 必须有明确的测试方法（可自动化，结果为 pass/fail）
4. **依赖关系**：明确列出技术必要依赖，最小化依赖，优先并行，无循环依赖

### 测试用例设计规范

**YOU MUST declare: "I will use skill:tdd and skill:testing-anti-patterns for test case design." before writing any test methods.**

设计 `# 测试方法` 章节时，必须参考：
- skill:tdd（路径：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/plan/prompts/skill_tdd_zh.md`）— 确保先写测试、覆盖红绿循环
- skill:testing-anti-patterns（路径：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/plan/prompts/skill_testing_anti_patterns_zh.md`）— 避免测试反模式

每个测试用例必须覆盖四要素：
1. **前置条件**：执行前系统需满足的状态
2. **输入参数**：调用的具体入参（含边界值：空值、最大值、非法值）
3. **操作序列**：精确的执行步骤
4. **预期输出**：可量化的断言（含正常路径 + 异常路径）

### task.md 格式

```markdown
# 依赖关系
task1, task2  （无依赖则留空）

# 任务名称
[动词开头，一句话]

# 任务目标
[具体描述要实现什么]

# 关键结果
1. [可验证的交付物]
2. ...

# 测试方法
1. [正常路径测试：前置条件 + 输入 + 操作 + 预期输出]
2. [边界用例：空输入/最大值/非法值等边界情况]
3. [异常路径：错误处理、回滚行为验证]
```

---

## 四、行为约束

### 输出目录（最高优先级）

所有文件必须保存在：

```
/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/plan
```

- 任务文件：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/plan/task1.md`、`task2.md` 等
- **不需要生成 tasks.json**（由 `rick doing` 自动解析生成）
- 禁止在工作目录之外创建任何文件

### 其他约束

- 先获取事实再判断，基于事实做技术选型
- 每个 task 必须有清晰的依赖关系（无循环依赖）

---

## 用户需求

实现: .rick/RFC/human-loop-upgrade-rfc.md

---

## ⚠️ 必须严格按以下 9 步 SOP 执行，不可跳过任何一步

1. **Domain 加载（优先）**：读取 `/Users/sunquan/ai_coding/CODING/rick/.rick/domain/` 下所有 `.md` 文件，提取与本次需求相关的事实（已知问题、构建命令、环境配置、接口规范等）。这些事实将直接影响任务分解和风险预判。

2. **Loops 上下文初始化**：读取上方 `## 可用的项目 Loops

- **do-check-mark-success-loop**："当 doing_check 报错 task status != success 或 commit_hash 缺失时触发"
- **tdd-red-green-refactor-loop**："当测试已存在且处于 FAIL 状态，需要通过迭代实现让其变绿时触发（前提：先写测试、当前测试 FAIL、目标是收敛到绿）"
` 中列出的项目已有 loop 模式，了解项目积累的工作流经验；如有相关 loop 文件，读取触发条件与范围，判断本次需求是否匹配某个已有 loop

3. **探索项目**：探索业务项目的源码，了解足够的事实信息；读取 `.rick/loops/` 和 `.rick/skills/` 了解项目已有工作流和技能

4. **grilling 追问**：加载 skill:grilling（路径：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/plan/prompts/skill_grilling.md`），对用户需求逐问追问，给出推荐答案，将需求澄清到具体可落实的代码路径或工具调用级别，达到终止条件后再继续

5. **方案设计**：在已有 domain 事实、loops/skills 约束下给出技术方案，说明主要决策点

6. **任务分解**：模块化分解，验证无循环依赖，确认可拓扑排序

7. **六维评审**（每个 subagent 独立启动，**串行执行**，上一个完成后再启动下一个）：
   - subagent_1：一致性检查 —— 任务目标与每个 task{n}.md 的任务目标对齐，确认每个 task 的交付物都能推进对应的 KR
   - subagent_2：loops/skills 利用检查 —— 检查 `.rick/loops/` 和 `.rick/skills/` 中已有的工作流与技能是否在合适的 task 中被引用和使用
   - subagent_3：依赖关系完整性 —— 确认所有 task 依赖的库、接口、数据结构在项目中已存在或在前置 task 中会被创建
   - subagent_4：执行风险推演 —— 阅读项目源码，逐 task 模拟真实执行过程：AI agent 会读哪些文件、调哪些接口、遇到哪些编译错误或运行时异常？暴露可能导致任务失败的风险点与卡点，在 task.md 中提前补充约束说明或修正任务描述
   - subagent_5：测试用例完整性 —— 参考 skill:tdd（`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/plan/prompts/skill_tdd_zh.md`）和 skill:testing-anti-patterns（`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/plan/prompts/skill_testing_anti_patterns_zh.md`），检查每个 task 的测试方法是否覆盖四要素（前置条件/输入参数/操作序列/预期输出），同时验证无测试反模式（如测试 mock 行为、仅用于测试的生产方法等）
   - subagent_6：端到端验证设计 —— 以用户视角设计可复用的验收测试方法：明确用户操作入口、预期的可观测输出、异常路径的兜底验证，确保交付产物质量可被客观检验
   - 每个 subagent 输出评审结论后，根据结论修正 task 文件，再启动下一个

8. **格式检查**：运行 `/Users/sunquan/ai_coding/CODING/rick/bin/rick tools plan_check job_26`；失败则修复后重新运行，直至通过

9. **输出**：按 task.md 格式保存到 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/plan/task{N}.md`
