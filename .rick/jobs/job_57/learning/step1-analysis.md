# Job_57 Step 1：执行记录分析（human-loop v8 重构）

## 1a. Debug 分析（本 job 无 debug/ 目录——以四轮实测反馈链与运维坑代替）

### 实测驱动的三轮呈现层迭代（核心反馈链）

| 轮次 | human 反馈（原话要点） | 根因诊断 | 修复（commit） |
|---|---|---|---|
| v8→v8.1 | 「信息的呈现不够优秀」+ 指示调研写作 skill（ASD-STE100 句子规范：短句/一句一义/主动语态/主语明确/少嵌套） | v8 呈现无润色层——orig 返回原样呈现，领域词汇无解释 | 03ab25d：新增 teacher_polish skill + orig 概念跳跃检索 + 五门禁验证规则（human 给的原文） |
| v8.1→v8.2 | 「不要有什么知识边界、已知结论、张力、引导性问题，不要这些结构化的描述」+「每个概念，按科普风格描述呈现清楚即可」 | v8.1 把「三态标注/张力标注/引导性问题」做成了显式标签——信息对了但形式错了（标签打断阅读，科普要求自然行文） | 24c1cfb：呈现去标签化——确定性状态融入行文（「这一点已被反复验证」），核心要求收敛为「每个概念科普讲清楚」 |
| v8.2→v8.3 | human 主动理论化升级：orig 本质 = 概念空间随机游走（同中有异/异中有同）；递归解释关键原则（零领域专用词汇，递归落通识层）；概念地图收尾引导 | v8.2 缺收尾机制——概念讲完就停，无「发散→收敛」转折；orig 定位缺理论统一（跳跃/启发/门禁服务未串成一体） | 3c31089：orig 本质重述 + 递归解释原则 + 收尾引导（串联概念→绘制概念地图→用自己的话表达→门禁判定） |

**教训**：呈现层的迭代不是一次设计能到位的——「讲清楚」的标准（短句规范→去标签→递归解释→概念地图）只能由 human 实测反馈逐轮收敛。dev 实测-反馈-修复循环（loop_19→22 四轮迁移）是呈现层设计的必要路径，不是可选优化。

### 实测中的概念补讲轮（v8.3 机制的实战验证）

loop_21 会话内 human 问「条件码长我不理解是什么意思」——会话正确执行「概念补讲轮」（无门禁，专讲概念，递归解释），证明递归解释原则被真实触发。这是 v8.3 机制的第一个实战证据。

### 运维坑（grilling→doing→release 全链路）

1. **research 派发三次故障**（grilling 阶段）：① workflowScript 内联 task 含换行 → JS 语法错误（修复：task 落盘派发单文件，task 参数单行引用）；② research agent 卡死 `find ~`（5min+ 0% CPU，修复：kill 进程 + subagent_supervisor steer 纠偏）；③ 两次「系统异常」失败（修复：fresh 重派 + 任务书附「分批落盘」教训——研究简报中途丢失是最大损失源）。
2. **dev-web restart 后 status 报 fingerprint=MISMATCH**：状态文件 stale 竞态——health 端点（build_id 实时）才是真值，status 判定会误报。验证 dev web 以 health 为准。
3. **go run 启动 dev-web 的进程收割**：go run 父进程退出带走 dev web 子进程（up 后进程即死）。修复：用 `./bin/rick tools dev-web up` 直接跑二进制。
4. **suspended 会话恢复链**：steer/prompt 都要求 active；suspended 先 `POST /api/sessions/{id}/resume` 再 steer。
5. **release 的 --prod-repo 必传**：共享开发机无法自动确定生产仓库路径（exit=4 提示）。
6. **release merge 门禁：生产工作树必须干净**：job 产物（rsi/approval.md 等）未提交会阻塞 merge——产物先 commit 再 release。
7. **显式版本号惯例**：release 默认 sha7-时间戳版本，但本仓库惯例是显式 semver（prev=8.1.12）。第一次 release（5e59492-…）漏打版本号，human 提醒后 `--version 8.2.0 --no-merge-source` 补打（main 已 merge，无需再动源码）。

### 已知遗留

- **loop 迁移的双继承注记**：thought_path.md 迁移脚本只剥一层继承注记，loop_22 文件内残留 loop_21 的注记头（不影响会话运行——会话正确读取最新注记继续）。后续迁移应剥到「# 思考轨迹」正文标题为止的全部注记块。
- loop_22 阶段 2 第 2 轮 human 回答仍在等待中（发布时会话进行中）——v8.3 的完整五阶段走通验证待 human 后续实跑。

## 1b. 完整执行轨迹摘要

### grilling 阶段（2026-11-25 15:00 → 11-26，L1→L5 全流程）

- **L1 调研**：派 research 两轮（research-L1 22 条结论：认知科学原创机制/AI 先例/CPS 映射；research-L1-r2 24 条：skill 生态先例/writing-to-think/生成性学习）+ 自查 v7 全资产（sense_loop 模板/四 agent/八 skills/handler/builder/loop_24 实跑证据）。
- **L3 第一轮 9 问**（定位/SENSE 去留/状态机/防锚定/挣扎强度/硬约束/兼容性/验证标准/命名，每问附调研上下文+推荐）。
- **human 第一轮裁决（彻底推翻）**：全部教学机制删除；SENSE 后续集成；无状态机；极简；本质=通过学习提升自己；不参考历史；完全重新思考「原创性的学习、思考、表达」；指示调研外部写作 skill。
- **L4 回流调研 + L3 收敛轮 7 问** → **human 第二轮裁决（v8 范式定型）**：SENSE 五阶段 = 思考门禁制 loop；每阶段一个 skill（启发+验证双目标）；一个 orig agent（唯一提示词）；thought_path.md 唯一产出；推进权在 human。
- **叶子层落实**（五阶段 skill/orig.md/sense_loop 重写/删除清单/代码改动/验证链）+ P1 裁决（orig 中间结构 stage{N}_{seq}.md）→ grilling_gate ✅ 通过。

### doing 阶段（四轮 worker 派发，全部 dev 树执行）

- **doing-1（v8，commit 9a1c8fe）**：25 文件 +589/−1036——删教学六 skills+四 agent+旧模板，新建五阶段 skill+orig.md+极薄 sense_loop 主模板+agents.go 单 orig+handler/builder/测试全链改造。验证：build ✓ 五包测试绿 ✓ dry-run 占位符 0 ✓ 旧概念 forbid 0 ✓。
- **orig 首启实测**（doing/orig-test/stage0_1.md，21KB 素材）：四分检索法（方法/现象/局限/跨领域）实测有效——跨领域两检产出对门禁设计最有杀伤力的张力 → 触发 human 呈现反馈。
- **doing-2（v8.1，03ab25d）**：18 文件 +148/−48——teacher_polish skill（ASD-STE100）+ orig 概念跳跃检索规则 + 五门禁验证规则（human 原文）+ 渲染管线接线。
- **doing-3（v8.2，24c1cfb）**：9 文件 +27/−33——呈现全面去标签化，核心收敛为「每个概念科普风格讲清楚」。
- **doing-4（v8.3，3c31089）**：9 文件 +33/−11——orig 本质理论化（概念空间随机游走）+ 递归解释关键原则 + 概念地图收尾引导。

### dev 实测迁移链（loop_19→20→21→22，dev_session_migration skill 五步法）

- **loop_19**（11-25 22:05 起）：阶段 1 两轮通过（ABCD 架构主张成型）→ 阶段 2 三领域选择 + orig 两轮深挖（stage2_1/2_2）。
- **loop_20**（11-27 17:57，v8.1 模板）：迁移后 human 实测发现呈现问题 → v8.1/v8.2 反馈在此产生。
- **loop_21**（11-27，v8.2）：继续讨论（阶段 2 第 3 轮 + 概念补讲「条件码长」）→ human 提出 v8.3 理论升级。
- **loop_22**（11-28 00:08，v8.3）：继承重呈现（素材复用未重派 orig），递归解释呈现验证通过 → human 确认「很好，已经满足我的要求」。
- 每轮迁移：POST /api/sessions 建新 loop（禁止预建目录）→ 拷 thought_path+orig → 顶部继承注记（承接链/进度边界/下一步起点）→ steer 就位 → 首 text 输出验证。

### release 阶段（2026-11-28 深夜）

- 预检：approval.md 留痕（human 原话）→ dev-web down（防门禁竞态）→ 落点可写性体检全绿（本机 bin/releases 属主可写，无需重定向）。
- 第一次 release（--merge-source）：exit=4 生产工作树不干净（rsi/ 未提交）→ 产物 commit 9bfbc5a → 重跑 RELEASE_OK（但版本号为默认 sha7）。
- human 提醒显式版本号惯例 → `--version 8.2.0 --no-merge-source` 补打 → RELEASE_OK version=8.2.0，生产 web（port 6137）health build_id=8.2.0 ✓，dev-web up --no-build 恢复 ✓。

### 工具调用与耗时特征

- parent 会话：grilling 全流程 + 四轮 worker 派发 + 四轮 loop 迁移 + release 两次，全程 bash/read/write + subagent 派发，无编辑器类工具滥用。
- research worker：三次失败三次恢复（故障→教训→fresh 重派），最终两份简报共 46 条结论全部落盘。
- coding worker：四轮全部一次通过（任务书+回执模式，无返工）——验证「grilling 叶子层落实 + worker 任务书单行指针」的派发纪律有效。
