---
name: doing-loop
description: Doing Loop 执行协议；有匹配项目 Loop 时执行项目 Loop，否则执行内置 Doing Loop
---

# Doing Loop

> ⚠️ 以下是默认 loop 的执行步骤，也是 gen-loop 需要参考的 skill 模板！！

---

## Step 0：Domain 搜索 + Loop 匹配

**必须依次完成以下两项，再进入 Step 1：**

### 0.1 搜索 Domain（强制）

根据澄清的需求，读取 `{{domain_dir}}` 下的相关文件，获取足够的事实信息（环境配置、已知问题、接口约束、构建命令等），建立解决问题的基本视角。

- 由 AI 自行判断读取哪些文件，但**必须完成搜索动作**后再继续
- 遇到任何问题（编译报错 / 测试失败 / 行为异常），**必须优先搜索 `{{domain_dir}}/bugs.md` 和 `{{domain_dir}}/`**，再做其他尝试

### 0.2 匹配 Loop

在 Domain 搜索完毕后，读取 `loops_context`，按 trigger 字段匹配当前任务/需求：

- **有匹配** → 读取对应 Loop 文件，按其定义步骤执行（不再执行以下 Step 1–5）
- **无匹配** → 按以下 Step 1–5 执行默认 Loop

---

## Step 1：parent（编排者）确认全局目标

确认以下内容全部清晰后才继续：

- task.md 中 `# 任务目标` 和 `# 关键结果` 已理解
- 成功标准已明确：测试脚本全通过 + 门禁通过（rick-gates helper 校验）+ 所有 Key Results 达成

---

## Step 2：parent 读取上下文（压缩策略）

从 `doing/debug/` 目录读取已有信息，按以下方式压缩后传递给 worker child：

- **bug\*.md** → 从每个文件的 frontmatter `summary` 字段提取摘要，避免重复踩坑
- **跨轮核心事实** → 任务目标 + Key Results 达成状态 + debug/ 摘要 + 当前迭代编号 N

---

## Step 3：启动 worker child 执行工作流

**每轮迭代由 parent 用 `runs.run` 启动一个独立 worker child（`agent:'worker'`），携带 Step 2 的上下文，执行完整工作流后返回产出摘要。**

```
[parent 编排者]
   │
   ├─ runs.run 派发 worker child（agent:'worker'，携带：任务目标 + debug/摘要 + 迭代编号 N）
   │     │
   │     │  worker child 执行：
   │     │  [ANALYZE] → [RED] → [GREEN] → [REFACTOR] → [COMMIT]
   │     │                 ↑        │
   │     │                 └──[DEBUG]┘
   │     │
   │     └─ worker child 完成，输出产出摘要
   │
   └─ parent 执行 Step 4 产出评估
```

触发语法（单写者：同一 cwd 只允许一个 worker child 写代码；默认 `async: true`；`context: "fork"` 继承父会话；必须带 `timeoutMs: 3600000`）：
```text
subagent({ workflowScript: "return runs.run('doing-N', { agent: 'worker', task: '<任务目标 + debug/摘要 + 迭代编号 N；完成时调 task_complete 工具>' })", async: true, context: "fork", timeoutMs: 3600000 })
```

### worker child：ANALYZE（理解需求）
1. 声明：`"I will use skill:sense."`，按 S→E→N 分析（Symptoms / Evidence / Next）
2. 读取 debug/ 摘要，避免重复踩坑

### worker child：自测驱动实现（TDD 方法，过程性）
1. 声明：`"I will use skill:tdd for implementation."`
2. 按 `# 测试方法`（自测指引）驱动实现：可先写自测再实现（RED→GREEN），自测代码写在写域内随交付或跑通即弃——**不落盘共享测试目录，不生成专门测试脚本**（层验收由门禁的模块集成测试承担）
3. 自测全绿 → REFACTOR；失败 → DEBUG

### worker child：DEBUG（遇红强制触发）

触发条件（任意一条）：测试 FAIL / 编译报错 / 行为与预期不符

1. **优先搜索 `{{domain_dir}}/bugs.md` 和 `{{domain_dir}}/`**，查看是否有精确解决方案
   - 有匹配 → 直接应用，记录引用来源
   - 无匹配 → 继续下方流程
2. 声明：`"I will use skill:debug-skill."`，加载 skill 文件：`{{debug_skill_path}}`
3. 在 `doing/debug/` 下创建 `bug{N}-{描述}.md`，按 Phase 1-6 执行
4. Phase 4 上限 3 次，达上限后输出当前状态并升级人工协作
5. 修复后回到 GREEN

### worker child：REFACTOR（代码改善）
1. 测试全绿后改善代码质量（命名、结构、去重）
2. 运行全量测试确认无回归；回归失败 → DEBUG

### worker child：完成回执（worker 不碰 git）
1. 自测全绿后**输出回执**：改动文件清单（限写域内）+ 自测结果摘要 + 遗留问题——**不执行任何 git 操作、不调用提交工具**
2. 提交由 parent 在层检查点统一执行（`level_complete`：跑 human 确认的 gate{N}.py 模块集成测试 → 绿 → 单次 commit → tasks.json 批量写）
3. rick 侧门禁（helper.py，会话结束后兜底校验 tasks.json 可解析/无 zombie/success 有 commit_hash）保持不变
4. **worker child 完成**：回执即完成，通知 parent 执行 Step 4

---

## Step 4：parent 产出评估

worker child 完成后，parent 逐项检查：

| 检查项 | 判断方法 |
|--------|----------|
| 门禁通过 | 读取 rick-gates helper 输出，确认 exit 0 |
| 测试全通过 | 确认测试脚本无 FAIL 输出 |
| Key Results 达成 | 逐条比对 task.md `# 关键结果` |

- **全部通过** → 进入 Step 5
- **存在失败** → 将失败原因附加到上下文，返回 Step 3 启动下一轮迭代

---

## Step 5：parent 确认停止标准

**成功退出**：门禁通过（rick-gates）+ 测试全通过 + 所有 Key Results 达成

**优雅退出**（任意一条触发）：
- 迭代次数达上限（默认 **3 轮**）
- 连续 2 轮产出相同错误（判断无法自动收敛）
- 人类明确要求停止

**退出时**：parent 输出 Loop 执行摘要（完成了哪些 KR、遗留了哪些问题），等待人类决策。
