# 依赖关系

task3

# 任务名称

升级 plan.md 六步 SOP 并集成 Cialdini 说服原则到 doing/test_python 提示词

# 任务目标

合并两项模板升级工作（原 task4 + task6）：

1. **plan.md**：升级为 RFC 规定的 a-j 十步 SOP，在对应步骤显式引用 `sense` 和 `tc` skill
2. **doing.md / test_python.md**：集成 Cialdini 三原则（权威/承诺/稀缺），提升 agent 对 core-skills 的合规率

两者都只依赖 task3 提供的 `LoadCoreSkills`，且均为模板文件修改，不涉及 Go 代码逻辑。

# 关键结果

**plan.md 升级：**

1. 新增 `## Plan SOP（a-j 步）` section，步骤 a 起始处：
   `YOU MUST use skill:sense for steps a-f. Declare: "I will use skill:sense for requirement analysis."`
2. 步骤 h 明确 6 个 subagent 职责（每个独立启动）：
   - subagent_1：RFC 与 task{n}.md 一致性
   - subagent_2：SPEC 合规检查
   - subagent_3：skills 利用检查
   - subagent_4：代码事实模拟（提前识别风险）
   - subagent_5：测试用例完整性 → `YOU MUST use skill:tc`，检查四要素（前置条件/输入参数/操作序列/预期输出）
   - subagent_6：端到端流程验证
3. 步骤 i 调用 `rick tools plan_check job_{n}` 程序化格式检查

**doing.md 升级（Cialdini 三原则）：**

4. 权威：`YOU MUST follow TDD. No exceptions.` 置于执行步骤顶部
5. 承诺：在开始实现前声明 `"I will use skill:[skill-name]"`（由 task8 具体指定 TDD/debug）
6. 稀缺：`Before proceeding to next task, verify: all tests pass`；`Immediately after test failure, run systematic-debugging Phase 1`

**test_python.md 升级（Cialdini 三原则）：**

7. 权威：`YOU MUST generate a failing test first (RED phase). No exceptions.`
8. 承诺：`Declare: "I will use skill:tdd and skill:tc for test generation."`（由 task8 具体完善）
9. 稀缺：`Before writing any test, verify: you understand the acceptance criteria`

**约束：** 以上修改均不改变现有模板变量结构（`{{task_goal}}` 等），不破坏 Go 代码

# 测试方法

1. 编译：`python3 tools/build_and_get_rick_bin.py`
2. **plan.md SOP 全覆盖**（OKR KR4）：
   ```bash
   python3 tools/check_prompt_variables.py --phase plan --keywords "skill:sense"    # 承诺宣告
   python3 tools/check_prompt_variables.py --phase plan --keywords "subagent_6"     # 6个subagent都在
   python3 tools/check_prompt_variables.py --phase plan --keywords "skill:tc"       # subagent_5 tc 引用
   python3 tools/check_prompt_variables.py --phase plan --keywords "plan_check"     # 步骤 i 程序化检查
   ```
3. **doing.md Cialdini 三原则**：
   ```bash
   python3 tools/check_prompt_variables.py --phase doing --keywords "YOU MUST"           # 权威
   python3 tools/check_prompt_variables.py --phase doing --keywords "I will use skill"   # 承诺
   python3 tools/check_prompt_variables.py --phase doing --keywords "Before proceeding"  # 稀缺
   ```
4. **test_python.md Cialdini**：
   ```bash
   python3 tools/check_prompt_variables.py --phase testing --keywords "YOU MUST generate" # 权威
   python3 tools/check_prompt_variables.py --phase testing --keywords "I will use skill"   # 承诺
   ```
5. 完整测试：`go test ./...`，无新增失败
