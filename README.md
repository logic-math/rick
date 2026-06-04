# Rick CLI

**对抗上下文熵增的 AI Coding 控制框架**

随着项目迭代，软件复杂性逐渐升高，这种复杂性表现为上下文的熵增。Rick 通过模块化与自进化机制控制上下文的生命周期，保持其整洁性，以实现最佳的 AI Coding 效果。

> 一切以上下文为中心构建的 AI Coding 控制框架。

核心公式：

```
AICoding = Humans + Agents
Agents   = Models + Harness（上下文）
```

每次执行都在积累上下文（`wiki/`、`tools/`、`SPEC.md`），由 dream 定期维护熵减，让后续的 AI agent 越跑越准。

---

## 安装

```bash
./scripts/install.sh
```

---

## 工作流

Rick 2.0 支持两种执行模式：

```
标准模式：plan → doing → learning → dream
简单模式：doing --easy → (auto learning) → dream
```

### 标准模式（适合有明确任务分解的需求）

```bash
# 1. 规划：AI 将需求分解为 task 列表
rick plan "为用户系统添加 JWT 认证"

# 2. 执行：自动逐任务执行，每个任务通过测试后自动 git commit
rick doing job_1

# 3. 积累：提取经验，更新 .rick/wiki/、.rick/tools/、.rick/SPEC.md
rick learning job_1

# 4. 全局反思：跨 job 知识进化（见下方）
rick dream
```

### Easy 模式（适合探索性任务、快速修复、对话式开发）

跳过 plan，直接与 Claude 交互式对话完成任务。

```bash
# 新建 easy 会话（交互式）
rick doing --easy "帮我修复登录 bug"
rick doing --easy          # 不带需求，进入后输入

# 恢复中断的 easy 会话
rick doing --easy --job job_5
```

Easy 模式特点：
- **无需 plan**：直接进入 Claude 交互式对话
- **会话续接**：中断后可通过 `--job` 恢复同一对话（`session_id` 已保存）
- **强制 debug 记录**：每解决一个问题，agent 必须写入 `doing/debug.md`（dream 的分析依据）
- **自动 learning**：会话退出后自动在后台运行 learning，直接更新 `.rick/wiki/`、`.rick/tools/`、`.rick/SPEC.md`

---

## Dream — 跨 Job 知识进化

Dream 定期运行，对已完成的 job 进行全局反思，维护 `.rick/` 知识体系的质量。

```bash
# 交互式 dream（查看 AI 思考过程）
rick dream

# 后台自动化 dream（无需人工干预）
rick dream -p
```

Dream 会自动：
1. 扫描未处理的已完成 job（标准模式 + easy 模式均支持）
2. 加载 `debug.md`、`tasks.json`、`act-path.md` 等行为轨迹
3. 提取优化信号，进化 skills，精简 SPEC.md（≤ 500 行）
4. 运行四维质量验证（引用链、冗余清理、运行仿真、路径推演）
5. 写入 `dream_run_{job_id}_log.md` 作为记录

> **Dream 是 learning 的兜底**：即使 learning 写出了问题，dream 也会在后续修复，无需追求 learning 的完美一致性。

### 典型运行频率

```bash
# 每完成 3-5 个 job 后运行一次
rick dream -p
```

---

## 工具命令（AI agent 使用）

```bash
rick tools plan_check job_1      # 验证 plan 目录结构
rick tools doing_check job_1     # 验证 doing 执行结果
rick tools learning_check job_1  # 验证 SUMMARY.md 已生成
rick tools dream_check           # 验证 dream log 文件格式
rick tools merge job_1           # 手动合并（一般无需，learning 直接写入）
```

---

## 完整示例

### 标准模式

```bash
# 第一个需求
rick plan "添加用户注册功能"
rick doing job_1
rick learning job_1

# 第二个需求（自动继承上次积累的上下文）
rick plan "添加 JWT 登录"
rick doing job_2
rick learning job_2

# 定期全局反思
rick dream -p
```

### Easy 模式

```bash
# 快速修复一个 bug
rick doing --easy "Redis 连接池泄漏，帮我排查修复"

# 会话中直接与 Claude 对话：
# - Claude 调试、修复问题
# - 每个问题自动记录到 debug.md
# - 退出后自动触发 learning

# 如果中途断开，恢复会话
rick doing --easy --job job_3

# 定期 dream 整合所有 easy job 的经验
rick dream -p
```

### 混合使用

```bash
# 复杂需求用标准模式，临时修复用 easy 模式
rick plan "重构认证模块"     # 标准
rick doing job_1
rick learning job_1

rick doing --easy "修复线上 500 错误"   # easy
rick doing --easy "优化查询性能"        # easy

rick dream -p   # 统一整合所有 job 的经验
```

---

## .rick/ 目录结构

```
.rick/
├── OKR.md            # 项目目标
├── SPEC.md           # 技术规范（≤ 500 行，dream 维护）
├── wiki/             # 知识文档（learning/dream 直接写入）
├── tools/            # 可复用 Python 工具（learning/dream 直接写入）
├── skills/           # .rick/skills/ 自定义 skills
├── dream/            # dream 运行日志
│   └── dream_run_{job_id}_log.md
└── jobs/
    └── job_N/
        ├── plan/               # 标准模式：task*.md、OKR.md
        └── doing/
            ├── debug.md        # 问题记录（easy/标准均有）
            ├── tasks.json      # 任务状态（标准模式 = 多任务；easy = 单条 easy_session）
            ├── session_id      # easy 模式：Claude 会话 ID
            └── prompts/        # 本次生成的所有提示词文件（持久化）
```

---

## 配置

**配置文件**: `~/.rick/config.json`

```json
{
  "max_retries": 5,
  "claude_code_path": "",
  "git": {
    "user_name": "Your Name",
    "user_email": "your.email@example.com"
  }
}
```

| 配置项 | 说明 |
|--------|------|
| `max_retries` | 标准模式任务失败最大重试次数（默认 5） |
| `claude_code_path` | Claude CLI 路径（空则使用 PATH 中的 `claude`） |
| `git.user_name/email` | 自动 commit 时使用的 Git 用户信息 |

---

## 版本

| 版本 | 日期 | 说明 |
|------|------|------|
| 2.0.0 | 2026-06-04 | Easy 模式、Dream 命令、prompts 持久化、learning 直写 .rick/ |
| 1.1.2 | 2026-03 | 标准 plan/doing/learning 流程 |
