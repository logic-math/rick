# Job_57 Step 3：Skill 提取报告（2026-11-29）

按 gen-skill 协议评估三个候选技能——**结论：0 新建，2 升级既有 skill，1 评估不并入不新建（记事实）**。

## 新建清单

（无——三个候选全部命中「相似 skill 优先升级」或「查询型事实非流程型 skill」路径）

## 升级清单（已直接修订，git diff 可审）

### 1. `dev_session_migration_skill/skill.md`（升级，候选①）

- **相似性判定**：触发场景与五步法与既有 skill 完全一致（>80% 重叠）——job_57 的贡献是 v8 产物结构适配 + 三个硬规则修正，走升级路径（gen-skill 协议第 5 条）
- **修订点**：
  1. 拷贝产物清单去硬编码（v7 教学产物清单 → 「当前 loop 全部产物，按实际存在拷」——避免每次模板演进改 skill）
  2. 继承注记精确剥离规则（剥到第一个正文标题前的**全部**注记块，不限层数 + python 片段——修复 loop_22 残留 loop_21 注记头的剥离 bug）
  3. **硬规则：模板层升级后禁止用 steer 传达新模板逻辑**（steer 只传一次性指令；模板变更唯一生效路径 = 迁移新 loop_N）——job_57 实测 human 纠正的直接产物
  4. suspended 会话恢复链（resume API → steer）与 TTFB 等待提示补入
  5. 实测记录更新（job_50 六次 + job_57 四次）

### 2. `rick_release_ops_skill/skill.md`（升级，候选③）

- **相似性判定**：触发场景完全重叠（release 提升操作）——是既有 skill 的预检清单与惯例增强
- **修订点**：
  1. 预检第 0 项：生产工作树干净（`git status --porcelain` 为空；job 产物先 commit——job_57 实测 merge 阶段中止浪费一次完整构建）
  2. 显式版本号惯例条：一律 `--version <semver>`，禁止裸跑默认 sha7（版本链 8.1.x → 8.2.0）；漏打补打方式（`--version <ver> --no-merge-source`）；--prod-repo 共享机必传
  3. backlog 记录：门禁顺序（merge 检查应前置）+ 命令级 semver 护栏

## 评估不并入不新建（候选②：dev-web 运维排障）

- **判定**：不并入 `dev_web_fingerprint_skill`（触发场景重叠 < 80%：fingerprint = 「验证前指纹校验」（构建一致性），本 job 坑 = 「进程起死/状态误报/会话挂起恢复」（运维排障））；不新建 skill
- **理由**：本 job 的 dev-web 运维坑（①go run 收割子进程 ②status fingerprint=MISMATCH 竞态、health 为真值 ③suspended resume 链 ④up 静默无输出）复用形态是**查询型事实**（bugs.md 条目，一次查得）而非**流程型技能**（多步流程）；且 ①② 已在升级后的 dev_session_migration_skill「关联」节与 prompt-template-iteration-loop 工具表（Step 4 落点）有承载
- **落点**：bugs.md 追加（Step 5 执行）——「dev-web 管理命令禁 go run（进程收割）+ status MISMATCH 竞态以 health build_id 为真值」一条两症状

## 模板源码待办（记 report，不属 skill 沉淀范围——模板改动走 doing/模板迭代，非 learning）

1. `internal/prompt/templates/skills/grilling.md` 派发语法节追加：task 含换行/超 1 行 → 先 write 派发单文件、task 参数单行指针（job_57 两次「系统异常」根因，第 4 次派发改文件化后一次成功；pi_orchestration_syntax_skill 已有纪律但 grilling 模板自身诱导内联）
2. `internal/prompt/templates/skills/orig.md` 检索规则节追加：全盘 find（`find ~`/`find /`）禁止，按派发单路径清单或 ls/grep 定向探索（job_57 research 卡死 5min+ 根因）

（此两项属 rick 源码模板，修改需走下一轮模板迭代 + release；本次 learning 不动源码，已在 report 留痕供后续消费）
