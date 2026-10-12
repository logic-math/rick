# skill:gen-loop（从 runtime-trace 提取并生成 Loop）

从 runtime-trace 和 debug 日志中识别 job 内反复出现的循环模式，固化为可复用的 Loop 文件。

## Loop 文件格式

> 参考 `doing_loop.md` 的 Step 0-5 结构，每个项目 Loop 必须遵循以下模板。

每个 Loop 写入 `{{loops_dir}}/{name}-loop.md`，包含以下完整结构：

---

### frontmatter

```yaml
---
name: {name}-loop
trigger: "当...时触发（具体场景描述）"
scope: "doing / easy / 全局"
---
```

---

### Step 0：环境确认 + Domain 搜索

在执行任何步骤前，必须完成以下两项：

**0.1 依赖准备**（硬约束，缺失则报错停止）：

```markdown
| 依赖项 | 确认命令 | 要求 |
|--------|----------|------|
| {tool} | `which {tool}` | 已安装 |

环境安装（首次或缺失时执行）：
```bash
# 安装命令
pip install {package}
```
```

**0.2 Domain 搜索**：

根据当前任务，搜索 `.rick/domain/` 下的相关文件，获取已知约束和事实信息。
遇到任何问题，**优先搜索 `.rick/domain/bugs.md`**，再做其他尝试。

---

### Step 1：parent（编排者）确认全局目标

描述本 Loop 要达成的目标和成功标准（与 task 的 Key Results 对齐）：

- 目标：...
- 成功标准：测试全通过 + check pass + 所有 Key Results 达成

---

### Step 2：parent 读取上下文（压缩策略）

从 `doing/debug/` 目录读取已有信息：

- **bug\*.md** → 从每个文件的 frontmatter `summary` 字段提取摘要，避免重复踩坑
- **跨轮核心事实** → 任务目标 + Key Results 达成状态 + debug/ 摘要 + 当前迭代编号 N

---

### Step 3：启动 worker child 执行工作流

**每轮迭代由 parent 用 `runs.run` 启动一个独立 worker child，执行完整工作流后返回产出摘要。**

```
[parent 编排者]
   │
   ├─ runs.run 派发 worker child（agent:'worker'，携带：任务目标 + debug/摘要 + 迭代编号 N）
   │     │
   │     │  worker child 执行：
   │     │  [{Step A}] → [{Step B}] → [COMMIT]
   │     │      ↑              │
   │     │      └──[DEBUG]─────┘
   │     │
   │     └─ worker child 完成，输出产出摘要
   │
   └─ parent 执行 Step 4 产出评估
```

**worker child：{Step A 名称}**
- 加载 skill：`.rick/{skill_name}_skill/skill.md`
- **精确命令**（必须写到具体命令，不得模糊描述）：
  ```bash
  {exact_command_1}
  {exact_command_2}
  ```
- 产出：...

**worker child：{Step B 名称}**
- 加载 skill：`.rick/{skill_name}_skill/skill.md`
- **精确命令**：
  ```bash
  {exact_command}
  ```
- 产出：...

**worker child：COMMIT**
1. `git add` + `git commit`（含 task ID）
2. 运行 check 命令，循环直到 pass

---

### Step 4：parent 产出评估

**调用验证 skill**：`.rick/{verify_skill}_skill/skill.md`

| 检查项 | 验证方法 | 通过标准 |
|--------|----------|----------|
| {check_1} | `{command}` | {expected} |
| {check_2} | `{command}` | {expected} |

- **全部通过** → 进入 Step 5
- **存在失败** → 将失败原因附加到上下文，返回 Step 3 启动下一轮

---

### Step 5：parent 确认停止标准

**成功退出**：所有 Key Results 达成，check pass

**优雅退出**（任意一条触发）：
- 迭代次数达上限（默认 **3 轮**）
- 连续 2 轮产出相同错误
- 人类明确要求停止

**退出时**：parent 输出 Loop 执行摘要，等待人类决策。

---

## 从 runtime-trace 提取协议

识别以下信号，判断是否值得提取为 Loop：

```
1. 在多个 task 中反复出现的工具调用序列（同一模式 3+ 次）
2. 跨 task 的"出错→修复→验证"循环（相同类型错误的共同解法）
3. 有明确触发条件（什么情况下会进入这个循环）
4. 有可量化的完成标准（知道什么时候退出循环）
5. 涉及依赖安装或环境配置（首次运行需要准备）
```

**不值得提取为 Loop 的情况**：
- 只在单个 task 出现，无法泛化
- 步骤完全线性，无需迭代收敛（提取为 skill 更合适）

## 写入协议

```
1. 读取所有 runtime-trace.md 和 debug/bug*.md
2. 识别反复出现的循环模式
3. 检查 .rick/loops/ 中是否已有 trigger/scope 相似的 loop：
   - 有相似 loop → 优先升级已有 loop（补充新步骤、完善依赖或评估标准），不创建新文件
   - 无相似 loop → 按上述 Step 0-5 格式编写新 {name}-loop.md
4. Step 0 填写依赖准备（从 runtime-trace 的环境配置步骤提取）
5. Step 3 填写每个 worker child Step 引用的 skill 路径
6. Step 4 填写产出评估的验证 skill
7. 写入 {{loops_dir}}/{name}-loop.md
```

**相似性判断标准**：trigger 场景 80% 重叠，或整体步骤序列基本一致 → 视为相似，优先升级。

## 质量标准

- trigger 足够具体，能判断何时激活
- Step 0 依赖准备完整，新环境可直接运行
- **Step 3 每个 worker child Step 必须写明精确的 shell 命令**（`go test ./internal/...` 而非"运行测试"）
- Step 3 每个 worker child Step 明确引用对应 skill（`.rick/{name}_skill/skill.md`）
- Step 4 产出评估有具体验证 skill 和精确验证命令
- Step 5 停止标准可量化，不依赖主观判断
