# skill:gen-loop（从 act-path 提取并生成 Loop）

从 act-path 和 debug 日志中识别 job 内反复出现的循环模式，固化为可复用的 Loop 文件。

## Loop 文件格式

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

### 依赖准备

在第一轮迭代开始前，子 Agent 确认以下依赖（硬约束，缺失则报错停止）：

```markdown
| 依赖项 | 确认命令 | 要求 |
|--------|----------|------|
| Go | `go version` | 1.21+ |
| Python | `python3 --version` | 3.8+ |
| {tool} | `which {tool}` | 已安装 |

环境安装（首次或缺失时执行）：
```bash
# 安装命令示例
pip install {package}
go install {tool}@latest
```
```

---

### 全局目标

描述本 Loop 要达成的目标和成功标准（与 task 的 Key Results 对齐）。

---

### 上下文管理

**压缩内容**：每轮迭代产生的中间状态

**写入目标**：`doing/debug/` 目录，遵循 debug_skill 写入规范

**压缩规则**：
- 已完成步骤的结论 → frontmatter `summary` 字段（一句话根因 + 状态）
- 未解决问题 → `status: "🔄 进行中"`
- 已解决问题 → `status: "✅ 已解决"`
- 跨轮传递的关键事实 → 父 Agent 从各文件 `summary` 字段提取，构成下一轮初始上下文

---

### 子 Agent 工作流

**每轮迭代父 Agent 启动一个子 Agent**，按以下状态机执行：

```
[Step 1] → [Step 2] → [Step N] → [COMMIT]
              ↑              │
              └──[DEBUG]─────┘  （任何 FAIL 时触发）
```

每个 Step 在执行前读取对应 skill：

**Step 1：{步骤名}**
- 加载 skill：`.rick/{skill_name}_skill/skill.md`
- 操作：...
- 产出：...

**Step 2：{步骤名}**
- 加载 skill：`.rick/{skill_name}_skill/skill.md`
- 操作：...
- 产出：...

**COMMIT**：
1. `git add` + `git commit`（含 task ID）
2. 运行 check 命令，循环直到 pass

---

### 产出评估

**调用验证 skill**：`.rick/{verify_skill}_skill/skill.md`

| 检查项 | 验证方法 | 通过标准 |
|--------|----------|----------|
| {check_1} | `{command}` | {expected} |
| {check_2} | `{command}` | {expected} |

---

### 停止标准

**成功退出**：所有 Key Results 达成，check pass

**优雅退出**（任意一条触发）：
- 迭代次数达上限（默认 **3 轮**）
- 连续 2 轮产出相同错误
- 人类明确要求停止

---

## 从 act-path 提取协议

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
1. 读取所有 act-path.md 和 debug/bug*.md
2. 识别反复出现的循环模式
3. 按上述格式编写 {name}-loop.md
4. 填写依赖准备（从 act-path 的环境配置步骤提取）
5. 填写每个 Step 引用的 skill 路径（.rick/{name}_skill/skill.md）
6. 填写产出评估的验证 skill
7. 写入 {{loops_dir}}/{name}-loop.md
```

## 质量标准

- trigger 足够具体，能判断何时激活
- 依赖准备完整，新环境可直接运行
- 每个 Step 明确引用对应 skill
- 产出评估有具体验证 skill（不只是人工判断）
- 停止标准可量化，不依赖主观判断
