# Job_57 Step 2：改进建议清单（冗余/可预防错误/更短路径）

> 输入：step1-analysis.md + grilling/design-tree.md + 相关 skill 现状核查（pi_orchestration_syntax / rick_release_ops / dev_session_migration / prompt-template-iteration）。按四个评估重点逐项给出根因 → 预防 → 落点。

## ① research 派发三次失败的预防

### 失败-1：workflowScript 内联 task 含换行 → JS 语法错误（两次「系统异常」部分同源）

- **根因**：skill_grilling.md 的显式派发模板把 task 写成内联长文本——模板本身诱导内联；而 pi_orchestration_syntax_skill 第 70 行已有「task 文本文件化」纪律（job_50 沉淀），**本 job 重复踩坑**。
- **改进**：skill_grilling.md 的派发语法节追加一行约束：「task 含换行/超 1 行 → 一律先 write 派发单文件（如 `<grilling>/dispatch-research-L{N}.md`），task 参数只放单行指针（读该文件并执行）——见 pi_orchestration_syntax_skill」。第 4 次派发（research-L1-r2c）改用文件化后一次成功，实证有效。
- **落点**：`internal/prompt/templates/skills/grilling.md`（源码树）+ `.rick/jobs/job_57` 无需——属模板层改进，进 Step 3/4 沉淀范围或直接 bugs.md。

### 失败-2：research agent 卡死 `find ~`（5min+ 0% CPU 低效全盘扫描）

- **根因**：research skill 无「文件系统探索边界」约束；agent 自行决定搜索路径时默认全盘 find（共享开发机 home 含数十万文件 + 远程挂载）。
- **改进**：派发单预置「已知路径清单」——凡涉及代码事实，直接给出目标文件路径（本 job 后续派发均如此，0 再犯）；research/orig 类 agent 的 skill 可加一条通用约束：「全盘 find（`find ~`/`find /`）禁止；按派发单路径清单或 ls/grep 定向探索」。
- **落点**：`internal/prompt/templates/skills/orig.md`（检索规则节追加一条，随 Step 3-4 修订一并做）或仅记 bugs.md。

### 失败-3：两次「系统异常」（上游模型/基础设施瞬时失败，简报中途丢失）

- **根因**：基础设施层不可根除；损失源是「简报在 agent 内存中未落盘」——第三次重派时前 12KB 已落盘，失败损失从「全丢」降为「尾部丢失」。
- **改进**：已验证有效的纪律是「write 首块 + bash 分批追加」（orig.md 中间结构落盘节已有）；**强化点**：所有长产出 agent 的任务书显式带「先落盘骨架（节标题），每完成一节立即追加落盘，禁止最后一次性写」——research 失败时 resume/重派可从已落盘内容续写，不从头再来。
- **落点**：bugs.md 追加（v4.0.6 已有「先骨架后填充」条目——本条是其「agent 自落盘」变体的再验证，追加一行实证即可）。

## ② dev-web up 用 go run 起进程即死

- **根因**：`go run ./cmd/rick tools dev-web up` 中 go run 作为父进程管理生命周期，其退出/收割导致 dev web 服务进程（即使 Setsid）随之死亡；实测同命令换 `./bin/rick`（dev 树已构建二进制）后稳定存活（pid 13402/28672/29107/41044/4330 多次验证）。
- **改进**：**管理命令一律直跑二进制**——`cd <dev-tree> && RICK_DEV_HOME=<dev-home> ./bin/rick tools dev-web <up|down|status>`；prompt-template-iteration-loop 的工具表已用 `<dev-bin>` 记法但未显式禁止 go run。
- **附带发现（status 判定盲区）**：dev-web up/restart 后 status 可能报 `fingerprint=MISMATCH + build_id 空`（状态文件 stale 竞态），此时以 **health 端点 build_id 为真值**（curl :8414/api/health）——本 job 三次 status 误报、health 全部正常。bugs.md 已有「health build_id 对照法」（up --no-build 条），MISMATCH-竞态变体值得追加一行。
- **落点**：bugs.md 新条目（go run 收割 + status MISMATCH 竞态，一条两症状）；prompt-template-iteration-loop 工具表加一行「dev-web 管理命令禁 go run，直跑 dev 树 bin/rick」。

## ③ release 两次返工

### 返工-1：生产工作树不干净 → merge 阶段中止（已完成一次完整构建被浪费）

- **根因**：release ops skill 的预检清单（落点可写性体检）不包含「生产工作树干净」检查；job 产物（rsi/approval.md）是 release 前刚写入的，天然处于未提交态。
- **改进**：release 前预检加第 0 项：`git -C <prod-repo> status --porcelain` 必须为空——非空时先 commit 产物（approval 留痕等）再 release。理想形态（backlog）：release 命令在 build 之前先做 merge 门禁检查（现状是 build 完成才在 merge 阶段失败，浪费构建时间）。
- **落点**：rick_release_ops_skill 追加预检项 + bugs.md（backlog：release 门禁顺序 build→merge 倒置浪费）。

### 返工-2：漏打显式版本号（默认 sha7-时间戳，破坏 8.1.x semver 链）

- **根因**：release 命令默认 sha7-时间戳合法（RELEASE_OK 一次成功），但本仓库生产版本链是显式 semver（prev=8.1.12）——惯例未沉淀进 skill/命令提示。
- **改进**：①rick_release_ops_skill 加一条惯例：「本仓库 release 一律 `--version <semver>`（major.minor.patch），禁止裸跑默认 sha7 版本」；②（backlog/命令级）release 无 --version 且检测到 prev 为 semver 时，交互提示「上一版本 8.1.12 是显式 semver，确认用默认 sha7-时间戳？」——机器守门优于人的记忆。
- **落点**：rick_release_ops_skill 追加 + bugs.md（backlog 记命令级护栏）。

## ④ loop 连环迁移的流程化（loop_19→20→21→22 四轮）

### 发现-1：模板升级后 steer 旧会话 ≠ 换模板（本 job 实际犯过一次）

- **根因**：v8.2 升级后我 steer loop_20 期望其用新呈现风格，human 纠正「你需要新开 loop_21 才能触发新的 prompt 逻辑」。dev_session_migration_skill 触发场景里写了「prompt 模板更新后旧会话仍用旧快照」，但**没有硬规则禁止「steer 传达新模板逻辑」**——执行者容易把 steer 当成模板热更新。
- **改进**：skill 第 4 步（steer 就位）追加硬规则：「**模板层已升级时禁止用 steer 传达新模板逻辑**——steer 只能传达一次性指令，不换 prompts 快照；模板变更生效的唯一路径 = 迁移到新 loop_N（新会话启动时渲染新模板）」。
- **落点**：`.rick/skills/dev_session_migration_skill/skill.md` 修订。

### 发现-2：拷贝产物清单过时（v7 教学产物 → v8 thought_path/orig）

- **根因**：skill 第 2 步写「拷 mission/zpd/values/syllabus/judgment/lessons/research」——v8 已删除全部教学产物，新产物是 thought_path.md + orig/（本 job 四轮迁移实际拷的就是这两样）。
- **改进**：skill 第 2 步改为「拷贝当前 loop 的全部产物目录内容（v8+：thought_path.md + orig/；随版本演进以实际产物为准）」——写成「全量拷贝 loop 目录产物」而非硬编码清单，避免每次模板演进都要改 skill。
- **落点**：同上 skill 修订。

### 发现-3：继承注记剥离 bug（残留旧注记头）

- **根因**：迁移脚本按「剥前 8 行」处理旧注记，只剥一层——loop_22 里残留 loop_21 的注记块（step1 已知遗留）。
- **改进**：skill 第 3 步（继承注记）给出精确剥离规则：「替换策略 = 剥离到第一个 `# 思考轨迹` 正文标题行之前的**全部**注记块（不限层数），再写新注记」；或 skill 直接内嵌可复制的 python 剥离脚本（从首个正文标题截取）。
- **落点**：同上 skill 修订（脚本片段）。

### 发现-4（正面模式）：「grilling 叶子层落实 + worker 任务书单行指针」四轮 worker 一次通过

- 不是改进对象而是验证：叶子层四维度（代码/文件/工具/环境）+ 派发单文件 + 单行指针 task 的组合，使 4 个 coding worker 全部一次通过（0 返工）。此模式无需新沉淀（doing_loop/gen-skill 体系已有），记入 SUMMARY 经验节即可。

## 汇总：改进落点矩阵

| # | 改进 | 落点 | 类型 |
|---|---|---|---|
| ①-1 | grilling 派发模板加「task 文件化」约束 | skills/grilling.md（模板源码） | 模板层 |
| ①-2 | agent 全盘 find 禁令 + 派发单路径清单 | skills/orig.md 或 bugs.md | skill/事实 |
| ①-3 | 「先骨架后分批落盘」agent 自落盘强化 | bugs.md 追加一行实证 | 事实追加 |
| ②-1 | dev-web 管理命令禁 go run（直跑 bin/rick） | bugs.md 新条目 + prompt-template-iteration-loop | 事实+loop |
| ②-2 | status MISMATCH 竞态 → 以 health 为真值 | bugs.md 追加 | 事实追加 |
| ③-1 | release 预检加「生产树干净」 | rick_release_ops_skill + bugs.md（backlog：门禁顺序） | skill+事实 |
| ③-2 | release 惯例 --version semver + 命令级护栏 backlog | rick_release_ops_skill + bugs.md（backlog） | skill+事实 |
| ④-1 | steer 禁代模板热更新（硬规则） | dev_session_migration_skill | skill 修订 |
| ④-2 | 拷贝清单「全量产物」化（去硬编码） | dev_session_migration_skill | skill 修订 |
| ④-3 | 继承注记精确剥离规则（含脚本） | dev_session_migration_skill | skill 修订 |
| ④-4 | 叶子层+任务书纪律正面验证 | SUMMARY 经验节 | 记录 |
