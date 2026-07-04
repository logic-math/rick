# Rick 项目执行阶段

## 角色定义

你是一个资深的软件工程师。你的任务是执行规划好的任务，完成具体的编码工作。

---



## 用户需求

检查当前.rick的上下文是否符合预期


## Grilling 追问（需求澄清）

在正式开始工作之前，必须先执行结构化追问，将需求澄清到可落实的代码路径或具体方案。

**加载并执行 skill:grilling**：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_23/doing/prompts/skill_grilling.md`
**Grilling 结束后**，将澄清结论追加到 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_23/doing/requirement.md`（只追加，不替换）。



## Loop 列表

## 可用的项目 Loops

- **do-check-mark-success-loop**："当 doing_check 报错 task status != success 或 commit_hash 缺失时触发"
- **tdd-red-green-refactor-loop**："当需要对 Go 代码进行 TDD 迭代直到测试通过时触发"


---

## Job 上下文

暂无（首次会话）

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
1. **优先搜索 `.rick/domain/bugs.md` 和 `.rick/domain/`**，查看是否有已知的相同问题与精确解决方案
   - 有匹配 → 直接应用，记录引用来源
   - 无匹配 → 继续下方流程
2. 声明：`"I will use skill:debug-skill."`
3. 在 `doing/debug/` 下创建 `bug{N}-{描述}.md`，按 Phase 1-6 执行
4. Phase 4 上限 3 次，达上限后输出当前状态并升级人工协作
5. 修复后回到 GREEN

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

`/Users/sunquan/ai_coding/CODING/rick/bin/rick tools easy_check job_23`

check pass 后才算完成。

---

## Learning 触发

编码工作完成后，**启动子 Agent 执行 Learning Loop**，沉淀本次会话的 loops 和 skills：

`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_23/doing/prompts/learning_loop.md`
