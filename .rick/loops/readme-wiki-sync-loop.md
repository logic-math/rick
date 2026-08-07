---
name: readme-wiki-sync-loop
trigger: "当需要编写或重构 README.md、wiki/（用户面向文档）等描述性资料时触发"
scope: "doing / easy / 全局"
---

# Loop：README + Wiki 文档同步

## 目标（Goal）

产出与代码事实一致的 README.md + wiki/ 用户文档,让读者理解 rick 解决什么问题、怎么用。

**范围限定**:本 Loop 只产出 `README.md` 和 `wiki/`,**不修改 `.rick/domain/`**(domain 由 learning_loop / dream 维护)。

**成功标准**:
- 文档中所有命令、flags、目录结构、配置项与代码事实一致
- 无过时或无用记录残留(旧架构概念、废弃 flag、已删除目录、过时版本号)
- README.md 与 wiki/ 之间无内容矛盾(README 是入口简版,wiki/ 是详细展开)
- `<rick_bin_path> tools easy_check <job_id>` / `doing_check` / `learning_check` pass

## 上下文管理（Context Management）

**输入**:
- 待重构文档当前内容:`README.md`(项目根)、`wiki/*.md` 和 `wiki/modules/*.md`(用户文档)
- `.rick/domain/*.md`(agent 知识库,**只读**,作为事实来源)
- `.rick/jobs/<job_id>/doing/requirement.md`(含 Grilling 澄清结论)
- `.rick/jobs/<job_id>/doing/debug/bug*.md` 摘要(如有),避免重复踩坑
- 跨轮核心事实:当前 rick 版本(`cmd/rick/main.go` 的 `VERSION` 常量)+ 三层知识体系状态

**输出**:
- `README.md` — 项目门面,简明扼要
- `wiki/*.md` 和 `wiki/modules/*.md` — 详细用户文档
- `.rick/jobs/<job_id>/doing/requirement.md` — 追加 Grilling 结论(如未存在)

## 可调用工具（Tool Access）

- **Read**:读 `README.md` / `wiki/` / `.rick/domain/` / `cmd/rick/main.go` / `internal/cmd/root.go` / `internal/config/config.go`
- **Grep**:搜索命令注册(`grep AddCommand`)、flags(`grep Flags()`)、过时引用(`grep ".rick/wiki/"` 等)
- **Write**:写 `README.md` / `wiki/*.md` / `wiki/modules/*.md`(**禁止** Write `.rick/domain/` 下任何文件)
- **Bash**:`ls wiki/` / `git log --oneline` / `./bin/rick tools easy_check` / `./bin/rick tools doing_check` / `./bin/rick tools learning_check`
- **Agent**:派发 Sub Agent 执行 GRILLING / FACT-INVESTIGATION / DRAFTING / CLEANUP / CONSISTENCY-GATE

## 产出评估（Output Evaluation）

| 检查项 | 验证方法 | 通过标准 |
|--------|----------|----------|
| easy_check / doing_check pass | `<rick_bin_path> tools easy_check <job_id>` | `✅ easy check passed` |
| 一致性门禁通过 | CONSISTENCY-GATE subagent 报告 | 0 个不一致问题 |
| 无过时记录残留 | `grep ".rick/wiki/\\.rick/SPEC.md\\.rick/OKR.md\\|v1.0.0-dev\\|TODO 待删除" README.md wiki/*.md wiki/modules/*.md` | 0 行(除非明确作为历史说明) |
| README 与 wiki 无矛盾 | 人工比对同一概念在两处的描述 | 一致 |
| 文档覆盖 requirement.md 所有决策 | 人工比对 | 全部覆盖 |
| **未修改 .rick/domain/** | `git status .rick/domain/` | 无变更(若有变更则终止 Loop) |

## 停止标准（Termination Condition）

**成功退出**:
- check pass
- CONSISTENCY-GATE 0 问题
- 无过时记录残留
- 人类确认文档可读且覆盖需求

**优雅退出**(任意一条触发):
- 迭代次数达上限(默认 **3 轮**)
- 连续 2 轮 CONSISTENCY-GATE 报告相同问题(无法自动收敛)
- 人类明确要求停止

**退出时**:Main Agent 输出 Loop 执行摘要(文档覆盖了哪些主题、清除了哪些过时记录、一致性门禁结果),等待人类决策。

---

## ⚠️ 不可变约束（硬性，违反即终止 Loop）

**本 Loop 禁止修改 `.rick/domain/` 目录下的任何文件。**

- `.rick/domain/` 是 agent 内部知识库,由 learning_loop / dream 维护,不属于本 Loop 的产出范围
- 本 Loop 只能**读取** `.rick/domain/` 作为事实来源,不得写入、修改、删除其中任何文件
- 若 DRAFTING 或 CLEANUP 试图修改 `.rick/domain/`,立即终止并报错

## 文档分层（必须区分）

| 文档 | 路径 | 面向 | 性质 | 本 Loop 权限 |
|------|------|------|------|--------------|
| README.md | 项目根 `README.md` | 用户(GitHub 门面) | 入口文档,简明扼要 | ✅ 可读写 |
| wiki/ | 项目根 `wiki/` 目录 | 用户(深入学习) | 详细架构、模块、运行时文档 | ✅ 可读写 |
| .rick/domain/ | `.rick/domain/` | agent(内部知识库) | 代码事实描述 | ❌ 只读,禁止修改 |

**关键约束**:三者职责不同,不可混淆。`wiki/` 不是 `.rick/domain/`,也不存在迁移关系——`wiki/` 是用户文档,`.rick/domain/` 是 agent 知识库,两者并行存在。

## 工作流（5 Step）

### Step 0:环境确认 + Domain 搜索

#### 0.1 依赖准备(硬约束)

| 依赖项 | 确认命令 | 要求 |
|--------|----------|------|
| rick binary | `ls ./bin/rick` | 已构建(缺失则 `./scripts/build.sh`) |
| git | `which git` | 已安装 |
| Mermaid 渲染 | 无需本地安装(GitHub 原生渲染) | - |

#### 0.2 Domain 搜索(只读,禁止修改)

读取 `.rick/domain/` 获取已知约束:
- `.rick/domain/commands.md` — 命令规范
- `.rick/domain/architecture.md` — 技术栈、模块划分、DIP 组合根
- `.rick/domain/project-conventions.md` — 路径约定、构建/发布流程
- `.rick/domain/bugs.md` — 已知问题与精确解决命令

#### 0.3 Wiki 现状搜索(用户文档)

```bash
ls wiki/
grep -l "v1.0.0-dev\|SPEC.md\|OKR.md\|wiki/" wiki/*.md wiki/modules/*.md
```

### Step 1:Main Agent 确认全局目标

见上方"## 目标（Goal）"section。

### Step 2:Main Agent 读取上下文

见上方"## 上下文管理（Context Management）"section。

### Step 3:启动 Sub Agent 执行工作流

```
[Main Agent]
   │
   ├─ SPAWN Sub Agent → [GRILLING] → [FACT-INVESTIGATION] → [DRAFTING] → [CLEANUP] → [CONSISTENCY-GATE] → [COMMIT]
   │                                                                    ↑                        │
   │                                                                    └────[如发现不一致]──────┘
   │
   └─ Main Agent 执行 Step 4 产出评估
```

#### Sub Agent:GRILLING(需求澄清)

- 加载 skill:`.rick/jobs/<job_id>/doing/prompts/skill_grilling.md`
- 设计树逐层追问,每问必附推荐答案
- 产出:`.rick/jobs/<job_id>/doing/requirement.md`(追加 Grilling 结论)
- **硬约束**:未完成 Grilling 禁止进入 FACT-INVESTIGATION

#### Sub Agent:FACT-INVESTIGATION(代码事实调研)

- 加载 skill:`.rick/skills/command_registration_verification_skill/skill.md`
- **精确命令**(禁止跳过):
  ```bash
  grep -n "AddCommand" internal/cmd/root.go
  grep -n "Use:\|Long:\|Flags()" internal/cmd/<cmd>.go
  grep -n "VERSION" cmd/rick/main.go
  cat internal/config/config.go
  ls .rick/domain/ .rick/loops/ .rick/skills/ .rick/draft/
  ls wiki/ wiki/modules/
  cat wiki/README.md
  git log --oneline -20
  ```
- 产出:事实清单(命令清单 + flags + 目录结构 + 版本演进 + 配置字段 + wiki 现状 + domain 现状)
- **关键原则**:尽可能深入调查事实,宁可多读代码也不要凭记忆推断

#### Sub Agent:DRAFTING(基于事实编写文档)

- 加载 skill:`.rick/skills/template-injection_skill/skill.md`(如需注入变量)
- 基于事实清单和 Grilling 结论编写文档
- **职责分工**:
  - `README.md`:项目门面,简明扼要
  - `wiki/`:详细展开
  - `.rick/domain/`:**禁止修改**
- **硬约束**:若 sub agent 试图修改 `.rick/domain/`,立即终止并报错

#### Sub Agent:CLEANUP(清除过时/无用记录)

- 检查 README.md 和 wiki/ 中是否引用已删除的旧架构概念
- 检查文档中是否引用已废弃的 flag 或命令
- 检查 wiki/ 下是否有重复或过时文件,直接删除(不留 TODO)
- **⚠️ 禁止执行**:rm / mv / Write 任何 `.rick/domain/` 下的文件

#### Sub Agent:CONSISTENCY-GATE(一致性门禁)

- 启动一个独立 subagent,专门检查文档与代码事实的一致性
- 检查清单:命令清单完整 / flags 准确 / 目录结构存在 / 配置字段一致 / 版本号正确 / 无旧架构引用 / 命令关系正确 / README 与 wiki 无矛盾 / wiki 内部无重复 / **未修改 .rick/domain/**
- **门禁规则**:
  - 发现任何不一致 → 返回 DRAFTING 修复
  - 发现 `.rick/domain/` 被修改 → 立即终止 Loop,回滚变更并报错
  - 全部通过 → 进入 COMMIT

#### Sub Agent:COMMIT

1. `git add README.md wiki/ .rick/jobs/<job_id>/`
   - ⚠️ **禁止 `git add .rick/domain/`**
2. `git commit -m "docs(<job_id>): <概述>"`
3. 运行 check 命令,循环直到 pass

### Step 4:Main Agent 产出评估

见上方"## 产出评估（Output Evaluation）"section。

- **全部通过** → 进入 Step 5
- **存在失败** → 将失败原因附加到上下文,返回 Step 3 启动下一轮

### Step 5:Main Agent 确认停止标准

见上方"## 停止标准（Termination Condition）"section。

---

## 关键原则

1. **禁止修改 `.rick/domain/`**:本 Loop 只产出 README.md 和 wiki/,`.rick/domain/` 由 learning_loop / dream 维护,本 Loop 只读不写
2. **区分文档层次**:`README.md`(入口简版)/ `wiki/`(用户详细文档)/ `.rick/domain/`(agent 内部知识库)三者职责不同,不可混淆,也不存在迁移关系
3. **事实优先于记忆**:文档中每一个具体陈述都必须有代码出处,凭记忆写文档是幻觉的主要来源
4. **完全清除过时记录**:不要留着过时的 wiki 或 README 内容,不要标记 TODO 待删除,直接删除
5. **独立一致性门禁**:CONSISTENCY-GATE 必须是独立 subagent,不能由 DRAFTING 的 agent 自检(避免自我确认偏差)
6. **Grilling 是硬约束**:未完成需求澄清禁止进入事实调研,否则文档会偏离用户真实意图
