---
name: readme-wiki-sync-loop
trigger: "当需要编写或重构 README.md、wiki/（用户面向文档）等描述性资料时触发"
scope: "doing / easy / 全局"
---

# Loop：README + Wiki 文档同步

## ⚠️ 不可变约束（硬性，违反即终止 Loop）

**本 Loop 禁止修改 `.rick/domain/` 目录下的任何文件。**

- `.rick/domain/` 是 agent 内部知识库，由 learning_loop / dream 维护，不属于本 Loop 的产出范围
- 本 Loop 只能**读取** `.rick/domain/` 作为事实来源，不得写入、修改、删除其中任何文件
- 若 DRAFTING 或 CLEANUP 试图修改 `.rick/domain/`，立即终止并报错

## 文档分层（必须区分）

| 文档 | 路径 | 面向 | 性质 | 本 Loop 权限 |
|------|------|------|------|--------------|
| README.md | 项目根 `README.md` | 用户（GitHub 门面） | 入口文档，简明扼要 | ✅ 可读写 |
| wiki/ | 项目根 `wiki/` 目录 | 用户（深入学习） | 详细架构、模块、运行时文档 | ✅ 可读写 |
| .rick/domain/ | `.rick/domain/` | agent（内部知识库） | 代码事实描述 | ❌ 只读，禁止修改 |

**关键约束**：三者职责不同，不可混淆。`wiki/` 不是 `.rick/domain/`，也不存在迁移关系——`wiki/` 是用户文档，`.rick/domain/` 是 agent 知识库，两者并行存在。

## Step 0：环境确认 + Domain 搜索

### 0.1 依赖准备（硬约束）

| 依赖项 | 确认命令 | 要求 |
|--------|----------|------|
| rick binary | `ls ./bin/rick` | 已构建（缺失则 `./scripts/build.sh`） |
| git | `which git` | 已安装 |
| Mermaid 渲染 | 无需本地安装（GitHub 原生渲染） | - |

### 0.2 Domain 搜索（只读，禁止修改）

读取 `.rick/domain/` 获取已知约束（这是 agent 内部知识库，与 `wiki/` 不同，**只读不写**）：

- `.rick/domain/commands.md` — 命令规范（所有 rick 命令的代码层约束）
- `.rick/domain/architecture.md` — 技术栈、模块划分、DIP 组合根
- `.rick/domain/project-conventions.md` — 路径约定、构建/发布流程
- `.rick/domain/bugs.md` — 已知问题与精确解决命令

### 0.3 Wiki 现状搜索（用户文档）

读取 `wiki/` 目录，确认现有用户文档覆盖范围：

```bash
ls wiki/
# 检查每个文件的版本标记和过时内容
grep -l "v1.0.0-dev\|SPEC.md\|OKR.md\|wiki/" wiki/*.md wiki/modules/*.md
```

---

## Step 1：Main Agent 确认全局目标

- **目标**：产出与代码事实一致的 README.md + wiki/ 用户文档，让读者理解 rick 解决什么问题、怎么用
- **范围限定**：本 Loop 只产出 `README.md` 和 `wiki/`，**不修改 `.rick/domain/`**（domain 由 learning_loop / dream 维护）
- **成功标准**：
  - 文档中所有命令、flags、目录结构、配置项与代码事实一致
  - 无过时或无用记录残留（旧架构概念、废弃 flag、已删除目录、过时版本号）
  - README.md 与 wiki/ 之间无内容矛盾（README 是入口简版，wiki/ 是详细展开）
  - `<rick_bin_path> tools easy_check <job_id>` / `doing_check` / `learning_check` pass

---

## Step 2：Main Agent 读取上下文（压缩策略）

- 读取待重构文档当前内容：
  - `README.md`（项目根）
  - `wiki/*.md` 和 `wiki/modules/*.md`（用户文档）
  - `.rick/domain/*.md`（agent 知识库，**只读**，作为事实来源）
- 读取 `.rick/jobs/<job_id>/doing/requirement.md`（含 Grilling 澄清结论）
- 读取 `.rick/jobs/<job_id>/doing/debug/bug*.md` 摘要（如有），避免重复踩坑
- 跨轮核心事实：当前 rick 版本（`cmd/rick/main.go` 的 `VERSION` 常量）+ 三层知识体系状态

---

## Step 3：启动 Sub Agent 执行工作流

```
[Main Agent]
   │
   ├─ SPAWN Sub Agent → [GRILLING] → [FACT-INVESTIGATION] → [DRAFTING] → [CLEANUP] → [CONSISTENCY-GATE] → [COMMIT]
   │                                                                    ↑                        │
   │                                                                    └────[如发现不一致]──────┘
   │
   └─ Main Agent 执行 Step 4 产出评估
```

### Sub Agent：GRILLING（需求澄清）

- 加载 skill：`.rick/jobs/<job_id>/doing/prompts/skill_grilling.md`
- 设计树逐层追问，每问必附推荐答案
- 产出：`.rick/jobs/<job_id>/doing/requirement.md`（追加 Grilling 结论）
- **硬约束**：未完成 Grilling 禁止进入 FACT-INVESTIGATION

### Sub Agent：FACT-INVESTIGATION（代码事实调研）

- 加载 skill：`.rick/skills/command_registration_verification_skill/skill.md`
- **精确命令**（禁止跳过）：
  ```bash
  # 1. 读 root.go 的 AddCommand 清单，确认命令注册
  grep -n "AddCommand" internal/cmd/root.go

  # 2. 读每个命令源文件的 cobra.Command 定义和 flags
  grep -n "Use:\|Long:\|Flags()" internal/cmd/<cmd>.go

  # 3. 读 VERSION 常量
  grep -n "VERSION" cmd/rick/main.go

  # 4. 读配置结构
  cat internal/config/config.go

  # 5. 读三层知识体系现状（.rick/domain/ 只读，不修改）
  ls .rick/domain/ .rick/loops/ .rick/skills/ .rick/draft/

  # 6. 读 wiki 现状（用户文档）
  ls wiki/ wiki/modules/
  cat wiki/README.md

  # 7. 读 git log 获取版本演进
  git log --oneline -20
  ```
- 产出：事实清单（命令清单 + flags + 目录结构 + 版本演进 + 配置字段 + wiki 现状 + domain 现状）
- **关键原则**：尽可能深入调查事实，宁可多读代码也不要凭记忆推断。文档中每一个具体陈述都必须有代码出处。

### Sub Agent：DRAFTING（基于事实编写文档）

- 加载 skill：`.rick/skills/template-injection_skill/skill.md`（如需注入变量）
- 基于事实清单和 Grilling 结论编写文档
- **职责分工**：
  - `README.md`：项目门面，简明扼要，覆盖核心概念 + 命令速查 + 入门示例
  - `wiki/`：详细展开，architecture.md / runtime-flow.md / modules/ 等深入文档
  - `.rick/domain/`：**禁止修改**（只读作为事实来源）
- **精确命令**：
  ```bash
  # 写入文档（使用 Write 工具，不使用 echo）
  Write README.md
  Write wiki/<topic>.md
  Write wiki/modules/<module>.md
  # ⚠️ 禁止执行：Write .rick/domain/<topic>.md
  ```
- 产出：完整的 README.md + wiki/ 文档
- **硬约束**：若 sub agent 试图修改 `.rick/domain/`，立即终止并报错

### Sub Agent：CLEANUP（清除过时/无用记录）

- **精确命令**：
  ```bash
  # 1. 检查 README.md 和 wiki/ 中是否引用已删除的旧架构概念
  grep -rn "\.rick/wiki/\|\.rick/tools/\|\.rick/SPEC.md\|\.rick/OKR.md" README.md wiki/
  # 若有引用，必须删除或替换为新架构（loops/skills/domain/draft）
  # 注意：wiki/ 本身是合法目录，不算过时；过时的是 .rick/wiki/（已迁移为 .rick/domain/）

  # 2. 检查文档中是否引用已废弃的 flag 或命令
  # 对照 FACT-INVESTIGATION 的事实清单，删除文档中不存在的命令/flag

  # 3. 检查 wiki/ 下是否有重复或过时文件
  ls wiki/ wiki/modules/
  # 若某文件内容已被其他文件覆盖或已过时（如版本标记 v1.0.0-dev），直接删除（rm，不留 TODO）

  # ⚠️ 禁止执行：rm / mv / Write 任何 .rick/domain/ 下的文件
  # .rick/domain/ 的清理由 dream 负责，本 Loop 无权处理
  ```
- 产出：清洁的 README.md + wiki/ 文档（无过时记录、无冗余、无 TODO 残留、无 v1.0.0-dev 等过时版本标记）
- **关键原则**：完全清除，不要留着。过时的 wiki 内容或 README 内容必须删除，不允许"标记 TODO 待删除"。但 `.rick/domain/` 不在本 Loop 的清理范围内。

### Sub Agent：CONSISTENCY-GATE（一致性门禁）

- **启动一个独立 subagent**，专门检查文档与代码事实的一致性
- 输入：DRAFTING + CLEANUP 产出的 README.md + wiki/
- 检查清单：
  | 检查项 | 验证方法 | 通过标准 |
  |--------|----------|----------|
  | 命令清单完整 | 文档列出的命令 = `grep AddCommand root.go` 结果 | 一一对应，无遗漏无多余 |
  | flags 准确 | 文档中每个 flag = `grep Flags() cmd/*.go` 结果 | 名称/类型/默认值一致 |
  | 目录结构存在 | 文档中每个目录 = 实际 `ls .rick/ wiki/` 结果 | 无虚构目录 |
  | 配置字段一致 | 文档配置表 = `config.go` 结构体字段 | 字段名/类型一致 |
  | 版本号正确 | 文档版本 = `VERSION` 常量 | 一致（无 v1.0.0-dev 等过时标记） |
  | 无旧架构引用 | `grep "\.rick/wiki/\|\.rick/SPEC.md\|\.rick/OKR.md" README.md wiki/` | 0 行（除非明确作为历史说明） |
  | 命令关系正确 | 文档中"A 是 B 的子模块"等关系 = 代码中函数调用关系 | 有代码出处 |
  | README 与 wiki 无矛盾 | 同一概念在两处描述一致 | 无矛盾 |
  | wiki 内部无重复 | `ls wiki/ wiki/modules/` | 无内容重叠的文件 |
  | **未修改 .rick/domain/** | `git status .rick/domain/` | 无变更（若有变更则终止 Loop） |
- 产出：一致性检查报告（列出所有发现的问题）
- **门禁规则**：
  - 发现任何不一致 → 返回 DRAFTING 修复，重新进入 CONSISTENCY-GATE
  - 发现 `.rick/domain/` 被修改 → 立即终止 Loop，回滚变更并报错
  - 全部通过 → 进入 COMMIT

### Sub Agent：COMMIT

1. `git add README.md wiki/ .rick/jobs/<job_id>/`
   - ⚠️ **禁止 `git add .rick/domain/`**（若 CONSISTENCY-GATE 已通过，此处应无 domain 变更）
2. `git commit -m "docs(<job_id>): <概述>"`
3. 运行 check 命令：
   ```bash
   <rick_bin_path> tools easy_check <job_id>
   # 或 doing_check / learning_check，取决于模式
   ```
4. check 失败 → 修复后重新运行，循环直到 pass

---

## Step 4：Main Agent 产出评估

**调用验证 skill**：`.rick/skills/command_registration_verification_skill/skill.md`

| 检查项 | 验证方法 | 通过标准 |
|--------|----------|----------|
| easy_check / doing_check pass | `<rick_bin_path> tools easy_check <job_id>` | `✅ easy check passed` |
| 一致性门禁通过 | CONSISTENCY-GATE subagent 报告 | 0 个不一致问题 |
| 无过时记录残留 | `grep "\.rick/wiki/\|\.rick/SPEC.md\|\.rick/OKR.md\|v1.0.0-dev\|TODO 待删除" README.md wiki/*.md wiki/modules/*.md` | 0 行（除非明确作为历史说明） |
| README 与 wiki 无矛盾 | 人工比对同一概念在两处的描述 | 一致 |
| 文档覆盖 requirement.md 所有决策 | 人工比对 | 全部覆盖 |

- **全部通过** → 进入 Step 5
- **存在失败** → 将失败原因附加到上下文，返回 Step 3 启动下一轮

---

## Step 5：Main Agent 确认停止标准

**成功退出**：
- check pass
- CONSISTENCY-GATE 0 问题
- 无过时记录残留
- 人类确认文档可读且覆盖需求

**优雅退出**（任意一条触发）：
- 迭代次数达上限（默认 **3 轮**）
- 连续 2 轮 CONSISTENCY-GATE 报告相同问题（无法自动收敛）
- 人类明确要求停止

**退出时**：Main Agent 输出 Loop 执行摘要（文档覆盖了哪些主题、清除了哪些过时记录、一致性门禁结果），等待人类决策。

---

## 关键原则

1. **禁止修改 `.rick/domain/`**：本 Loop 只产出 README.md 和 wiki/，`.rick/domain/` 由 learning_loop / dream 维护，本 Loop 只读不写
2. **区分文档层次**：`README.md`（入口简版）/ `wiki/`（用户详细文档）/ `.rick/domain/`（agent 内部知识库）三者职责不同，不可混淆，也不存在迁移关系
3. **事实优先于记忆**：文档中每一个具体陈述都必须有代码出处，凭记忆写文档是幻觉的主要来源
4. **完全清除过时记录**：不要留着过时的 wiki 或 README 内容，不要标记 TODO 待删除，直接删除
5. **独立一致性门禁**：CONSISTENCY-GATE 必须是独立 subagent，不能由 DRAFTING 的 agent 自检（避免自我确认偏差）
6. **Grilling 是硬约束**：未完成需求澄清禁止进入事实调研，否则文档会偏离用户真实意图
