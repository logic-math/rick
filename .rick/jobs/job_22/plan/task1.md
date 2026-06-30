# 依赖关系


# 任务名称

创建 `.rick/loops/` 和 `.rick/skills/` 目录，写入固化的格式规范和示例文件

# 任务目标

建立 RFC-001 新架构的两个核心目录，并将 loop.md 和 skill.md 的完整模板格式以文件形式固化，供 learning/dream 阶段的 agent 按规范产出候选文件。

目录定位：
- `.rick/loops/`：项目级 loop 文件，描述带评估机制的迭代控制流（由 learning/dream 产出候选，人工审核后合并）
- `.rick/skills/`：项目级 skill 文件，描述原子级能力模块（由 learning/dream 产出候选，人工审核后合并）

**Loop 与 Skill 的本质区别**：
- Skill = 静态上下文模块，agent 加载后执行一次，无迭代语义
- Loop = 带评估机制的动态迭代控制流，agent 需判断每轮进展、管理跨轮上下文、知道何时收敛

**⚠️ 以下模板内容必须严格按照规范写入文件，不得自由发挥。**

---

## 必须创建的文件及其内容

### 文件 1：`.rick/loops/README.md`

**完整内容**（逐字写入，章节和字段名不得修改）：

````markdown
# Loop 格式规范

loop.md 描述一个带评估机制的迭代控制流（Loop Engineering），供 agent 在需要反复执行直到收敛的场景中加载。
Loop 与 Skill 的本质区别：Skill 是静态上下文模块（执行一次），Loop 是动态迭代控制流（执行直到收敛）。

由 learning/dream 阶段产出候选（命名为 `candidate_loop_N.md`），人工审核后重命名为正式文件。

## Frontmatter 字段规范

```yaml
---
name: loop-name        # 必须：小写字母+数字+连字符，最长 64 字符
trigger: "一句话描述"  # 必须：什么场景下激活本 loop（供 loops_context 注入）
---
```

## 正文五要素（Loop Engineering）

### 目标（Goal）
可验证的结果定义，必须是 agent 自己能判断是否达成的，不依赖人的主观评估。

- 成功标准：[具体可量化指标，如"所有目标测试通过且 exit code 为 0"]
- 自评命令：`[agent 用来判断目标是否达成的命令]`
- 自评输出：[命令返回什么时视为达成]

### 上下文管理（Context Management）
每轮迭代后的上下文更新策略。决定 agent 在长循环中是越跑越准还是越跑越乱。

- 保留：[跨迭代必须保留的信息，如 bug 记录、已尝试的方案列表、当前最佳结果]
- 压缩：[可摘要化的信息，如历史迭代的详细输出 → 只保留结论和数据]
- 遗忘：[可丢弃的信息，如临时调试日志、已回滚的代码片段]

### 可调用工具（Tool Access）
本 loop 中 agent 可使用的工具及权限边界。

- `[工具/命令]`：[用途描述] —— 约束：[允许或禁止的操作]
- 权限边界：[明确列出本 loop 中绝对禁止的操作，如"禁止直接 push 到远端"]

### 产出评估（Output Evaluation）
每轮迭代后判断进展的机制。这是 Loop 与普通脚本的本质区别——没有评估，循环无法收敛。

- 评估类型：客观（运行测试/编译）/ 主观（LLM 评分）/ 混合（diff 对比 + 测试）
- 评估命令：`[可直接执行的评估命令]`
- 进展判断：[如何从评估结果判断本轮是否有实质进展，如"错误数量减少视为进展"]
- 退步判断：[如何判断本轮比上轮更差，触发回滚或策略调整]

### 停止标准（Termination Condition）
明确的双出口：成功收敛 + 失败退出，缺一不可。

- **成功退出**：当 [可量化条件] 时停止，视为收敛成功
- **失败退出**：达到最大迭代次数 [N] 次，或出现 [硬性失败条件，如连续 N 轮无进展]
- **优雅退出**：[失败时的收尾动作，如记录当前最优状态、将未解决问题写入 debug 文件、标记需要人工介入]
````

---

### 文件 2：`.rick/skills/README.md`

**完整内容**（逐字写入）：

````markdown
# Skill 格式规范

skill.md 描述一个原子级能力单元，agent 在遇到触发条件时按需加载并执行一次。
格式参考 agentskills.io 标准（When to Use / Procedure / Pitfalls / Verification），
内容面向 agent 而非人类（步骤可直接执行，命令可直接复制）。

由 learning/dream 阶段产出候选（命名为 `candidate_skill_N.md`），人工审核后移入 `.rick/skills/`。

## Frontmatter 字段规范

```yaml
---
name: skill-name               # 必须：小写字母+数字+连字符，最长 64 字符
description: "触发场景：能做什么"  # 必须：供 skills_context 索引检索
---
```

## 正文四要素

### When to Use
遇到什么情况时加载本 skill（触发词或触发场景，要具体）。

### Procedure
分步骤的执行方法，每步包含：
- 具体命令（可直接执行）
- 预期输出（可量化断言）

### Pitfalls
已知坑点，含具体反例：
- ❌ 不要……（原因）
- ✅ 应该……

### Verification
如何确认 skill 执行成功：
- `[命令]` 输出 `[预期内容]` 即为成功
````

---

### 文件 3：`.rick/loops/example_loop.md`

**完整内容**（通用示例，不绑定具体项目）：

````markdown
---
name: go-tdd-loop
trigger: "当需要对 Go 代码进行 TDD 迭代直到测试通过时触发"
---

# Loop: Go TDD 迭代循环

## 目标（Goal）

让目标测试从失败状态收敛到通过状态，agent 自己可判断是否达成。

- 成功标准：目标测试全部通过，无 FAIL 输出，exit code 为 0
- 自评命令：`go test ./[package]/... -run [TestName] -v 2>&1`
- 自评输出：最后一行包含 `ok  [package]` 且无 `--- FAIL:`

## 上下文管理（Context Management）

- 保留：每轮的测试失败信息（具体 error message）、已尝试的修改方案摘要、当前失败用例列表
- 压缩：上一轮的完整代码 diff → 只保留"修改了哪个函数、结果如何"一句话
- 遗忘：已回滚的代码改动、临时加入的 fmt.Println 调试语句、通过的测试的详细输出

## 可调用工具（Tool Access）

- `go test`：运行单元测试，判断目标是否达成 —— 约束：只跑目标测试，不跑全量（`-run` 精确匹配）
- `Read / Edit / Write`：读写源码文件 —— 约束：只修改与失败测试直接相关的文件
- `git diff`：确认工作区状态 —— 约束：每轮修改前必须工作区干净
- 权限边界：禁止在迭代过程中 `git commit` 或修改测试文件本身（测试是 spec，不是实现）

## 产出评估（Output Evaluation）

- 评估类型：客观（运行目标测试，结果确定性）
- 评估命令：`go test ./[package]/... -run [TestName] -v 2>&1 | tail -5`
- 进展判断：本轮失败用例数量 < 上轮失败用例数量，视为有实质进展
- 退步判断：本轮失败用例数量 ≥ 上轮，或引入新的编译错误，立即用 `git checkout .` 回滚本轮改动

## 停止标准（Termination Condition）

- **成功退出**：`go test` 输出 `ok` 且 exit code 为 0，目标达成，退出循环
- **失败退出**：连续 3 轮无进展（失败数不减少），或累计迭代超过 5 轮
- **优雅退出**：回滚到本 loop 启动前的 git 状态（`git stash`），将当前失败信息写入 `debug/bug{n}-tdd-stuck.md`，frontmatter status 标为 `"❌ 无法修复"`，等待人工介入
````

# 关键结果

1. `.rick/loops/README.md` 存在，正文包含 Loop Engineering 五要素章节标题（目标/上下文管理/可调用工具/产出评估/停止标准）
2. `.rick/skills/README.md` 存在，正文包含四要素章节（When to Use/Procedure/Pitfalls/Verification）
3. `.rick/loops/example_loop.md` 存在，frontmatter 含 name/trigger，正文包含完整五要素内容
4. 所有文件内容与上方规范一致，章节标题不得随意修改

# 测试方法

1. **文件存在性验证**：
   - 操作：`ls .rick/loops/README.md .rick/loops/example_loop.md .rick/skills/README.md`
   - 预期输出：三个文件均存在，exit code 0

2. **loop frontmatter 校验**：
   - 操作：`python3 -c "import re; c=open('.rick/loops/example_loop.md').read(); fm=re.search(r'^---\n(.*?)\n---', c, re.DOTALL); assert fm, 'no frontmatter'; fields={l.split(':')[0].strip() for l in fm.group(1).splitlines() if ':' in l}; assert {'name','trigger'}.issubset(fields), f'missing: {fields}'"`
   - 预期输出：exit code 0

3. **loop 五要素章节验证**：
   - 操作：`grep -c "## 目标\|## 上下文管理\|## 可调用工具\|## 产出评估\|## 停止标准" .rick/loops/example_loop.md`
   - 预期输出：5

4. **skill README 四要素验证**：
   - 操作：`grep -c "When to Use\|Procedure\|Pitfalls\|Verification" .rick/skills/README.md`
   - 预期输出：4

5. **幂等性验证**：
   - 操作：重新写入相同内容后再次运行测试 1-4
   - 预期输出：所有结果不变

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 debug-skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_22/doing/prompts/skill_debug_skill.md`

执行顺序：按 debug-skill Phase 1-6 执行，每阶段达上限后升级人工协作
