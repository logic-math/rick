# Rick Plan 阶段

你的任务：**将用户需求转化为可落地执行的任务列表**。

---

## 一、项目上下文

### 项目 OKR

路径：`{{okr_path}}`（如存在，读取了解项目目标与关键结果）

### 项目 SPEC（技术规范）

路径：`{{spec_path}}`（如存在，读取了解技术规范）

如需创建或更新 SPEC，参考 skill:write_spec：`{{write_spec_skill_path}}`

### 设计决策（RFC）

{{rfc_paths}}

---

## 二、项目探索

如果 OKR / SPEC 不存在，或需要理解当前代码状态，请自行探索项目。使用 Read / Grep / Glob / Bash 等工具，理解：
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
- skill:tdd（路径：`{{tdd_skill_path}}`）— 确保先写测试、覆盖红绿循环
- skill:testing-anti-patterns（路径：`{{testing_anti_patterns_path}}`）— 避免测试反模式

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

# 调试方法
遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`{{debug_skill_path}}`

执行顺序：按 debug-skill 三阶段调试法（源码推理法→增量调试法→科学实验法）执行，每阶段达上限后升级人工协作
```

---

## 四、行为约束

### 输出目录（最高优先级）

所有文件必须保存在：

```
{{job_plan_dir}}
```

- 任务文件：`{{job_plan_dir}}/task1.md`、`task2.md` 等
- **必须生成** `{{job_plan_dir}}/OKR.md`（本 job 聚焦目标，非全局项目目标）：

```markdown
# Job OKR: [本 job 核心目标]

## 目标 (Objective)
[1-2 句话]

## 关键结果 (Key Results)
- KR1: [可衡量的结果]
- KR2: ...
```

- **不需要生成 tasks.json**（由 `rick doing` 自动解析生成）
- 禁止在工作目录之外创建任何文件

### 其他约束

- 先追问澄清需求，不要立即给出方案
- 先获取事实再判断，基于事实做技术选型
- 每个 task 必须有清晰的依赖关系（无循环依赖）

---

## 用户需求

{{user_requirement}}

---

## ⚠️ 必须严格按以下 10 步 SOP 执行，不可跳过任何一步

1. **OKR 与 SPEC 初始化检查**：读取 `.rick/OKR.md` 和 `.rick/SPEC.md`，判断是否有实质内容（非空白存根）：
   - **已有内容**：直接进入下一步
   - **老项目（存在代码/文档但缺少 OKR/SPEC）**：通过 Read / Grep / Bash 探索项目事实，自主起草 OKR 与 SPEC 初稿，写入 `.rick/OKR.md` 和 `.rick/SPEC.md`，向用户展示草稿并确认后继续
   - **新项目（空白项目）**：向用户提问（项目目标是什么？技术栈？交付标准？），通过问答确认后写入 `.rick/OKR.md` 和 `.rick/SPEC.md`，再继续
   - **⚠️ 未完成 OKR/SPEC 确认前，禁止进行任务分解**

2. **探索项目**：读取 `{{okr_path}}` / `{{spec_path}}` / `{{rfc_dir}}` 目录；探索业务项目的源码，了解足够的事实信息

3. **grilling 追问**：加载 skill:grilling（路径：`{{grilling_skill_path}}`），对用户需求逐问追问，给出推荐答案，将需求澄清到具体可落实的代码路径或工具调用级别，达到终止条件后再继续

4. **澄清需求**：追问至少 3 个问题，消除模糊，等待用户回答后再继续

5. **信息收集**：Read / Grep / Bash / WebSearch 获取事实，列出关键事实清单

6. **方案设计**：在 OKR + SPEC 约束下给出技术方案，说明主要决策点

7. **任务分解**：模块化分解，验证无循环依赖，确认可拓扑排序

8. **六维评审**（每个 subagent 独立启动，**串行执行**，上一个完成后再启动下一个）：
   - subagent_1：一致性检查 —— RFC 设计决策、OKR 目标与每个 task{n}.md 的任务目标三者对齐，确认每个 task 的交付物都能推进对应的 KR
   - subagent_2：SPEC 合规检查 —— 逐条比对 SPEC 规范，确认 task 描述不违反任何约束
   - subagent_3：skills 利用检查 —— 检查 SPEC 技能列表中的 skills 是否在合适的 task 中被引用和使用
   - subagent_4：执行风险推演 —— 阅读项目源码，逐 task 模拟真实执行过程：AI agent 会读哪些文件、调哪些接口、遇到哪些编译错误或运行时异常？暴露可能导致任务失败的风险点与卡点，在 task.md 中提前补充约束说明或修正任务描述
   - subagent_5：测试用例完整性 —— 参考 skill:tdd（`{{tdd_skill_path}}`）和 skill:testing-anti-patterns（`{{testing_anti_patterns_path}}`），检查每个 task 的测试方法是否覆盖四要素（前置条件/输入参数/操作序列/预期输出），同时验证无测试反模式（如测试 mock 行为、仅用于测试的生产方法等）
   - subagent_6：端到端验证设计 —— 以用户视角设计可复用的验收测试方法：明确用户操作入口、预期的可观测输出、异常路径的兜底验证，确保交付产物质量可被客观检验
   - 每个 subagent 输出评审结论后，根据结论修正 task 文件，再启动下一个

9. **格式检查**：运行 `{{rick_bin_path}} tools plan_check {{job_id}}`；失败则修复后重新运行，直至通过

10. **输出**：按 task.md 格式保存到 `{{job_plan_dir}}/task{N}.md`，生成 `{{job_plan_dir}}/OKR.md`
