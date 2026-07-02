---
name: default-doing-loop
description: 通用任务执行 Loop，父 Agent 控制迭代，子 Agent 执行编码。无匹配项目 Loop 时使用。
trigger: "loops_context 中无匹配 Loop 时自动使用；或在 easy 会话中执行具体编码任务时触发"
scope: "单个 task 或需求文档的完整实现周期（编码 + 测试 + 调试 + 提交）"
---

# Default Doing Loop（通用任务执行 Loop）

> **使用前**：先检查 `loops_context`，若有匹配当前任务 trigger 的项目 Loop，优先用 skill:loop-protocol 执行它。无匹配时，父 Agent 按本 Loop 状态机控制迭代，子 Agent 执行编码。

---

## 五要素

### 1. 全局目标

实现 task.md（`# 任务目标` + `# 关键结果`）或需求文档中的全部交付物。

**成功标准**（全部满足时退出）：
- 测试脚本全部通过（无 FAIL 输出）
- doing_check / easy_check pass
- 所有 Key Results 均已达成（逐条可验证）

---

### 2. 上下文管理（压缩策略）

每轮迭代的中间信息写入 `doing/debug/` 目录，格式遵循 debug_skill 写入规范：

- **遇到 bug** → 写 `bug{N}-{描述}.md`（frontmatter + Phase 1-6 + 结论）
- **跨轮传递的核心事实** → 从各 bug 文件的 frontmatter `summary` 字段提取（已压缩）

**父 Agent 启动下一轮子 Agent 时传递**：
1. 任务目标 + 当前 Key Results 达成状态
2. debug/ 摘要（各 bug 的 summary 字段）
3. 迭代编号（第 N 轮）

---

### 3. 子 Agent 工作流

每轮迭代启动一个子 Agent，按以下状态机执行：

```
[ANALYZE] → [RED] → [GREEN] → [REFACTOR] → [COMMIT]
               ↑        │
               └──[DEBUG]┘  （任何 FAIL / 报错时触发）
```

#### ANALYZE（理解需求）

1. 声明：`"I will use skill:sense."`，按 S→E→N 分析：
   - **S (Symptoms)**：任务目标 + Key Results 的具体含义
   - **E (Evidence)**：读相关代码，了解现有实现的事实
   - **N (Next)**：一句话实现方案
2. 读取 debug/ 摘要，避免重复踩坑

#### RED（先写失败测试）

1. 声明：`"I will use skill:tdd for implementation."`
2. 针对 `# 测试方法` 中每个场景编写测试
3. 运行测试，**必须确认 FAIL**（证明测试有效，这是进入 GREEN 的前提）

#### GREEN（最小实现）

1. 编写让测试通过的最小实现代码（不超出 task scope）
2. 运行测试：通过 → 进入 REFACTOR；失败 → 触发 DEBUG

#### DEBUG（遇红强制触发）

触发条件（任意一条）：测试 FAIL / 编译报错 / 行为与预期不符

1. 声明：`"I will use skill:debug-skill."`
2. 在 `doing/debug/` 下创建 `bug{N}-{描述}.md`，按 Phase 1-6 执行
3. Phase 4（插桩观察）上限 3 次，达上限后输出当前状态并升级人工协作
4. 根因确认后修复，回到 GREEN 重新运行测试

#### REFACTOR（代码改善）

1. 测试全绿后改善代码质量（命名、结构、去重）
2. 运行全量测试确认无回归；回归失败 → 触发 DEBUG

#### COMMIT（收尾提交）

1. `git add` + `git commit`（commit message 含 task ID）
2. 运行 check 命令（使用父 prompt 上下文中的 rick_bin_path 和 job_id）：
   - doing 阶段：`<rick_bin_path> tools doing_check <job_id>`
   - easy 阶段：`<rick_bin_path> tools easy_check <job_id>`
3. check 失败 → 根据错误修复，重新运行，循环直到 pass

---

### 4. 产出评估（父 Agent 执行）

子 Agent 完成后，父 Agent 执行以下检查：

| 检查项 | 判断方法 |
|--------|----------|
| check pass | 读取 doing_check / easy_check 输出，确认 ✅ |
| 测试全通过 | 确认测试脚本无 FAIL 输出 |
| Key Results 达成 | 逐条比对 task.md `# 关键结果`，判断每条是否已实现 |

评估结论：**成功**（全部通过）或 **失败**（附具体失败原因，传递给下一轮）

---

### 5. 停止标准

**成功退出**（全部满足）：
- check pass + 测试全通过 + 所有 Key Results 已达成

**优雅退出**（任意一条触发）：
- 迭代次数达上限（默认 **3 轮**）
- 连续 2 轮产出相同错误（判断无法自动收敛）
- 人类明确要求停止

**退出时**：父 Agent 输出 Loop 执行摘要（完成了哪些 KR、遗留了哪些问题），等待人类决策。

---

## Loop 状态机（父 Agent 视角）

```
[INIT]
  │
  ▼
[SPAWN_SUBAGENT] ← 携带：任务目标 + debug/摘要 + 迭代编号 N
  │
  │  子 Agent 执行：ANALYZE → RED → GREEN/DEBUG → REFACTOR → COMMIT
  │
  ▼
[EVALUATE] ← 父 Agent：check 输出 + 测试状态 + KR 达成检查
  │
  ├── 成功 ──────────────────────── [DONE] ✅
  │
  └── 失败 → [STOP_CHECK]
                │
                ├── 达到上限 / 无法收敛 ── [GRACEFUL_EXIT] ⚠️
                │
                └── 未达上限 → 压缩上下文 → [SPAWN_SUBAGENT] N+1
```
