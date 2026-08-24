# 项目约定

## 路径约定

| 路径 | 说明 |
|------|------|
| `.rick/draft/` | 个人判断记录（价值维度）：`rfc/`（human-loop 会话 RFC 产出）、`loops/loop_N/`（每轮会话目录，含 prompts/ + briefs/ + judgment.md）、`concepts/`、`human-learning/`、`loops.md`/`progress.md`；由 `GetDraftDir()` 管理，`rick human-loop` 执行时自动创建 |
| `.rick/draft/rfc/` | human-loop 会话 RFC 产出（原 `.rick/RFC/` 已被 v2.10 迁移，`GetRFCDir()` 现返回此路径） |
| `.rick/jobs/job_N/` | 每次 job 的工作目录，包含 plan/doing/learning 三个子目录 |
| `.rick/jobs/job_N/plan/OKR.md` | job 级 OKR，由 plan 阶段 Claude 生成，doing/learning 阶段读取 |
| `.rick/loops/` | loop 文件（`{name}-loop.md`），dream 阶段产出和维护 |
| `.rick/skills/` | skill 目录（`{name}_skill/skill.md`），dream 阶段产出和维护 |
| `.rick/domain/` | 项目级别的领域知识文档（命令规范/架构/Go模式/测试约定） |
| `.rick/dream/` | dream_run_*_log.md 和 prompts/；待处理 jobs 由程序自动扫描 tasks.json 发现 |
| `.rick/dream/run_log_{n}.md` | learning 阶段 Step 6 写入的度量文件（Job/模型/错误次数/工具调用轮次） |
| `doing/debug/bug*.md` | 调试记录（新格式），YAML frontmatter 含摘要；`LoadDebugContext()` 优先读此目录 |
| `doing/debug/bug*.md` (fallback) | 无 debug/ 目录时，回退读取 `doing/debug.md` |
| `doing/tasks/{taskID}/act-path.md` | 任务执行后自动生成，含工具调用/报错次数/执行时长 |
| `doing/tasks/{taskID}/raw_session.log` | Claude Code NDJSON 原始流式输出，每行一个 JSON 对象 |
| `.rick/jobs/job_N/doing/tests/` | 测试脚本目录，从此出发需 **6 次 dirname** 到项目根 |

## debug 文件格式

`doing/debug/bug*.md` 的 YAML frontmatter 格式：

```markdown
---
title: "问题标题"
status: "✅ 已解决" | "🔄 进行中" | "❌ 无法修复"
summary: "一句话根因 + 状态"
---

正文：详细描述、复现步骤、解决方案
```

## 工程实践

### 构建与安装

```bash
./scripts/build.sh      # 构建 → ./bin/rick
./scripts/install.sh    # 安装到 ~/.rick/bin/rick（用户决定是否安装）
```

测试时直接用 `./bin/rick`，无需安装。

### 版本号规则

- 版本号定义在 `cmd/rick/main.go` 的 `const VERSION`，语义化三段式 `major.minor.patch`
- **每修复一个 fix，patch（最小段）就 +1**（如 4.0.0 → 4.0.1）
- 改完 `cmd/rick/main.go` 后必须 `go build -o bin/rick ./cmd/rick` 重新编译，否则 `rick --version` 仍是旧版本

### Git 工作流

- 每个 task 完成后独立 commit，message 包含 task ID（如 `feat(task3): ...`）
- **大改动 commit 后必须 `git status` 验证无遗漏文件**（job_28 教训：git add 参数列表中的文件可能未被 add 成功，只看 commit 成功消息会漏文件；think.md 因此漏 commit 一次）
- learning 产出经人工审核后手动合并到 `.rick/`（逐文件审核，`git add .rick/ && git commit`）

### 持续集成

```bash
go test ./...                          # 单元测试
bash tests/tools_integration_test.sh  # 集成测试
```

### check 命令 --auto-fix 规范

- 默认只**报告**问题，不自动修复（保持确定性）
- `--auto-fix` 标志才触发 Claude 修复（opt-in）

## 已知 Template 注入遗漏（job_23 发现）

`GenerateEasyPromptFile`（easy 会话启动时）**不写入** `skill_gen_domain.md`，也**不注入** `domain_dir`/`gen_domain_path` 到 `learning_loop.md`。

- 影响：easy 模式的 learning_loop.md 中 `{{gen_domain_path}}` 和 `{{domain_dir}}` 保持为字面量，Step 5 子 Agent 无法读取 gen-domain skill 路径
- 绕过方式：手动使用 `internal/prompt/templates/skills/gen-domain.md` 原始模板，domain 路径用 `.rick/domain/`
- 对比：`GenerateEasyLearningPromptFile`（会话结束后）已正确注入上述变量

**来源 Job**: job_23

## 新架构上下文结构（v2.9.0+）

```
.rick/
├── loops/          # 迭代控制流（{name}-loop.md）
├── skills/         # 原子能力单元（{name}_skill/skill.md）
├── domain/         # 项目知识文档（commands/architecture/go-patterns/testing/conventions）
├── RFC/            # human-loop 产出
├── jobs/           # job 工作目录
└── dream/          # dream 日志和 prompts
```

旧架构（wiki/tools/SPEC.md）已于 v2.9.0 迁移删除：
- wiki/ → skills/ + domain/
- tools/*.py → skills/\*/helper.py
- SPEC.md → domain/ (拆分) + skills/ (规范类)
- OKR.md → job_N/plan/OKR.md（job 级，由 plan 生成）
