---
name: doing-loop
description: Doing Loop 执行协议；有匹配项目 Loop 时执行项目 Loop，否则执行内置 Doing Loop
---

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
3. 按需读取 `.rick/domain/` 下所有文件，提取与本任务相关的已知约束和事实

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

