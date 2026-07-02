# Rick 项目执行阶段

## 角色定义

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

---

## 任务信息

**任务 ID**: task1
**任务名称**: 创建 `.rick/loops/` 和 `.rick/skills/` 目录，写入固化的格式规范和示例文件

### 任务目标
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

### 关键结果
1. `.rick/loops/README.md` 存在，正文包含 Loop Engineering 五要素章节标题（目标/上下文管理/可调用工具/产出评估/停止标准）
2. `.rick/skills/README.md` 存在，正文包含四要素章节（When to Use/Procedure/Pitfalls/Verification）
3. `.rick/loops/example_loop.md` 存在，frontmatter 含 name/trigger，正文包含完整五要素内容
4. 所有文件内容与上方规范一致，章节标题不得随意修改


### 测试方法
**文件存在性验证**：
操作：`ls .rick/loops/README.md .rick/loops/example_loop.md .rick/skills/README.md`
预期输出：三个文件均存在，exit code 0
**loop frontmatter 校验**：
操作：`python3 -c "import re; c=open('.rick/loops/example_loop.md').read(); fm=re.search(r'^---\n(.*?)\n---', c, re.DOTALL); assert fm, 'no frontmatter'; fields={l.split(':')[0].strip() for l in fm.group(1).splitlines() if ':' in l}; assert {'name','trigger'}.issubset(fields), f'missing: {fields}'"`
预期输出：exit code 0
**loop 五要素章节验证**：
操作：`grep -c "## 目标\|## 上下文管理\|## 可调用工具\|## 产出评估\|## 停止标准" .rick/loops/example_loop.md`
预期输出：5
**skill README 四要素验证**：
操作：`grep -c "When to Use\|Procedure\|Pitfalls\|Verification" .rick/skills/README.md`
预期输出：4
**幂等性验证**：
操作：重新写入相同内容后再次运行测试 1-4
预期输出：所有结果不变






## Loop 列表

## 可用的项目 Loops

- **candidate-loop-1**：when implementing new features
- **go-tdd-loop**："当需要对 Go 代码进行 TDD 迭代直到测试通过时触发"


---

## Job 上下文

暂无问题记录

---

## ⚠️ 执行 Loop（强制，不可跳过）

**YOU MUST execute the following loop. No exceptions. Skipping this loop means the task is NOT complete.**

# Doing Loop

## Step 0：匹配决策

读取 `loops_context`，按 trigger 字段匹配当前任务/需求：

- **有匹配** → 读取匹配的项目 Loop 文件，按其定义执行
- **无匹配** → 按以下 Doing Loop 执行

---

## 全局目标

实现 task.md（`# 任务目标` + `# 关键结果`）或需求文档中的全部交付物。

**成功标准**（全部满足时退出）：
- 测试脚本全部通过（无 FAIL 输出）
- doing_check / easy_check pass
- 所有 Key Results 均已达成（逐条可验证）

---

## 上下文管理（压缩策略）

每轮迭代的中间信息写入 `doing/debug/` 目录，遵循 debug_skill 写入规范：

- **遇到 bug** → 写 `bug{N}-{描述}.md`（frontmatter + Phase 1-6 + 结论）
- **跨轮传递的核心事实** → 从各 bug 文件的 frontmatter `summary` 字段提取（已压缩）

**父 Agent 启动下一轮子 Agent 时传递**：任务目标 + Key Results 达成状态 + debug/ 摘要 + 迭代编号 N

---

## 子 Agent 工作流

**每轮迭代由父 Agent 启动一个独立子 Agent 执行，子 Agent 完成 ANALYZE→COMMIT 全流程后，父 Agent 执行产出评估，决定继续迭代或退出。**

```
[父 Agent]
   │
   ├─ SPAWN 子 Agent（携带：任务目标 + debug/摘要 + 迭代编号 N）
   │     │
   │     │  子 Agent 执行：
   │     │  [ANALYZE] → [RED] → [GREEN] → [REFACTOR] → [COMMIT]
   │     │                 ↑        │
   │     │                 └──[DEBUG]┘
   │     │
   │     └─ 子 Agent 完成，输出产出摘要
   │
   └─ 父 Agent 产出评估 → 成功退出 / 继续迭代 / 优雅退出
```

**ANALYZE（理解需求）**
1. 声明：`"I will use skill:sense."`，按 S→E→N 分析（Symptoms / Evidence / Next）
2. 读取 debug/ 摘要，避免重复踩坑

**RED（先写失败测试）**
1. 声明：`"I will use skill:tdd for implementation."`
2. 针对 `# 测试方法` 中每个场景编写测试
3. 运行测试，**必须确认 FAIL**（证明测试有效，进入 GREEN 的前提）

**GREEN（最小实现）**
1. 编写让测试通过的最小实现代码（不超出 task scope）
2. 通过 → REFACTOR；失败 → DEBUG

**DEBUG（遇红强制触发）**
触发条件（任意一条）：测试 FAIL / 编译报错 / 行为与预期不符
1. 声明：`"I will use skill:debug-skill."`
2. 在 `doing/debug/` 下创建 `bug{N}-{描述}.md`，按 Phase 1-6 执行
3. Phase 4 上限 3 次，达上限后输出当前状态并升级人工协作
4. 修复后回到 GREEN

**REFACTOR（代码改善）**
1. 测试全绿后改善代码质量（命名、结构、去重）
2. 运行全量测试确认无回归；回归失败 → DEBUG

**COMMIT（收尾提交）**
1. `git add` + `git commit`（commit message 含 task ID）
2. 运行 check 命令（使用 prompt 上下文中的 rick_bin_path 和 job_id）：
   - doing 阶段：`<rick_bin_path> tools doing_check <job_id>`
   - easy 阶段：`<rick_bin_path> tools easy_check <job_id>`
3. check 失败 → 修复后重新运行，循环直到 pass
4. **子 Agent 完成**：输出本轮产出摘要（完成了哪些 KR、遗留了哪些问题），通知父 Agent 执行评估

---

## 产出评估（父 Agent 执行）

子 Agent 完成后，父 Agent 逐项检查：

| 检查项 | 判断方法 |
|--------|----------|
| check pass | 读取 doing_check / easy_check 输出，确认 ✅ |
| 测试全通过 | 确认测试脚本无 FAIL 输出 |
| Key Results 达成 | 逐条比对 task.md `# 关键结果` |

评估结论：**成功**（全部通过）或 **失败**（附具体原因，传递给下一轮）

---

## 停止标准

**成功退出**：check pass + 测试全通过 + 所有 Key Results 达成

**优雅退出**（任意一条触发）：
- 迭代次数达上限（默认 **3 轮**）
- 连续 2 轮产出相同错误（判断无法自动收敛）
- 人类明确要求停止

**退出时**：父 Agent 输出 Loop 执行摘要（完成了哪些 KR、遗留了哪些问题），等待人类决策。





---

## 完成要求

`/Users/sunquan/ai_coding/CODING/rick/bin/rick tools doing_check job_N`

check pass 后才算完成。


