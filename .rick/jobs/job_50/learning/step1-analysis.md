# Learning Step 1 — job_50 执行记录分析（human-loop v7 重设计→发布全程）

> 素材：git log 12 个 job_50 提交 + doing/requirement.md（grilling 16 问 + v7.1–v7.6.2 增量裁决）+ doing/grilling/design-tree.md（四层设计树）。本 job 无 debug/ 与 runtime-trace（交互式设计-实现-发布会话，非任务流水线）。

## ① 执行轨迹还原（每个增量：触发 → 改动面 → 验证）

| # | 提交 | 触发（human 实测反馈） | 改动面 | 验证 |
|---|---|---|---|---|
| 1 | bb2a23e v7 | grilling 16 问全裁决（ADR D1–D10 落地：架构=Go 模板、六 skill→agent、anchor/二次探针/三档 rubric/篇幅 ZPD 驱动） | sense_loop.md 重写 + 5 skill 模板 + teach×2/research task 模板删除 + rickAgents 7→8 + 渲染/测试 | build + 4 包测试 + dry-run 冒烟 |
| 2 | 3241210 | 部署后 grep 发现错字 | zpd-assessor description 判揃→判断 | build+test |
| 3 | 640758b v7.1 | 「② ZPD 不要突兀问价值判断；费曼演绎选点建档；ZPD/费曼抽 skill；① 结构化提问」 | ② 五步流水重写 + feynman 第 8 agent + zpd_assessor select/consolidate + ① 固定三字段一次成表 | build+test+dry-run+dev 部署 |
| 4 | f5bbc7e v7.2 | 「每课费曼太复杂」+ 探索流 skill 定义 + 推进权归 human | 费曼 3 步 parent 内联 + exploration_stream skill + zpd.md 档案方法论 + agent 8→4 | build+test+dry-run+dev 部署 |
| 5 | 9a0df67 v7.3 | loop_17 实测：「大量思考内容/无关信息」——38 条 text 里编排叙述+10KB 英文翻译 | 输出纪律（沉默的编排者）：text 只允许七类 + 病灶反例写入 + skills 纯正文契约 | 会话实测分析（38 text block 逐条）+build+test |
| 6 | 48e937a v7.4 | 「课程无效信息；调研信息要结构化；SENSE 还原作者思考过程；探索流移到全部学完后统一一次；调研混沌学园探索流」 | lesson_writer 思考还原链 + 探索流出④入⑤ + 混沌学园三动作方法体（research-L2） | build+test+dry-run+dev 部署+loop_19 实测 |
| 7 | 15a098a v7.5 | 「探索流还是不行」→ SENSE 状态机重设计 | exploration_stream 重写：五步状态机+各阶段达标判据 | build+test+dry-run+loop_20 实测 |
| 8 | de57334 v7.6 | 「还是不行」→ 出三方案 human 选 A | 方案 A 靶向卡点调研：进程判定+靶向 research+张力呈现+推进判定 | build+test+dry-run+loop_21 实测 |
| 9 | 4f6e667 v7.6.1 | 「呈现的信息不够说人话」 | 费曼呈现纪律（初学者假设+最少概念+概念必附准确定义）×3 层（探索流/research 契约/lesson_writer） | build+test+dry-run+loop_22 实测 |
| 10 | 618b6ea v7.6.2 | 「教学内容核心原则=启发学生推导自己的 SENSE 思考过程」 | 脚手架原则（学生先推再对照作者）+课末「你的 SENSE 痕迹」五条 | build+test+dry-run+loop_22 实测 |
| 11 | 69f95e9 | 「阶段满意→release 到生产 + loop_22 继承到生产作思考记录」 | dev 树教学记录存档（loop_14 补全+loop_15/17-22+zai 配图）→ release --merge-source 随源码同版进生产 | release 门禁+生产 health build_id+agents 部署+dry-run |
| 12 | 2594f18 | 「humans 里无用 loop 删除省 token，只留 loop_22+架构决策思考 loop」 | 删 15 个 loop（-7.7MB）：实验/验证/调优/论文学习/采购/infer RSI 链；保留 loop_22/12/7/2/4 | git ls 确认 5 loop 留存 |

**迭代形态**：v7 初始 → 7 轮实测反馈增量（每轮：human 在 dev 实跑一个 loop → 反馈一句话 → 定向增量 → build/test → dev 部署 → 下一轮实测），与 web-feedback-iteration-loop 同构但对象是 prompt 模板层。

## ② 问题清单（根因 + 当时解法 + 是否根治）

| # | 问题 | 根因 | 当时解法 | 根治？ |
|---|---|---|---|---|
| 1 | workflowScript task 文本转义错误 ×3（research-L1 派发两连挂：撇号 `devil\'s`、`\n` 解码为真实换行破坏 JS 单引号字符串；runs.all 再挂 invalid key） | task 文本内嵌单引号 JS 字符串，JSON `\n`→真实换行即 unterminated string；撇号需转义 | 改用**文件式任务书**（write research-L1-brief.md）+ 单行指针 task——此后 0 失败 | **未根治**（工具层未做校验/转义；靠纪律规避。证据混合：doing-1..4 含 `\n` 的 task 曾成功，说明失败还与文本组合有关） |
| 2 | worker 回执被传输层截断 ×2（doing-1/doing-2 回执断句但工作已完成） | 长回执超传输层上限 | parent 不信回执信**产物**（自行 build/test/dry-run 复核） | **未根治**（传输限制仍在；parent 侧复核模式已固化为本会话惯例） |
| 3 | runs.all 传 promise 报 invalid key（第 4 轮 research+worker 并行派发） | runs.all 期望描述符对象而非已启动的 run promise；混用导致 workflow 失败且 research 子运行被 SIGTERM | 单独 runs.run 重派 research；孤儿 doing-5 child 幸存继续完成，挂 subagent_wait 订阅收割 | **未根治**（API 用法纪律；孤儿运行收割是补救不是机制） |
| 4 | release 门禁连挂 ×4（web 包 TempDir RemoveAll cleanup 竞态：三个不同测试轮换失败；单测/低载通过、门禁全量必挂） | 门禁 buildEnv 用 **HOME=dev-home**（DevHomeOr）→ web 测试 spawn 的 worker 与**在跑的 dev web 实例**（loop_22 会话）共享 dev HOME 的 pi runtime/sessions → worker 异步终止滞后于 t.TempDir 清理；-p 1 串行重负载放大 | 诊断链（HOME 对照实验→-p 1 复现→单测通过定位竞态）→ **release 前 dev-web down** 消除竞争后 release 成功 | **未根治**（测试基建竞态仍在：worker 终止与 TempDir 清理无同步；dev-web down 是规避不是修复） |
| 5 | Squid 代理 403 断网（web_search/Exa fetch failed、npm E403，research agent 全程受阻） | 公司 Squid（hk02-llm-spider-proxy37）ACL 拒绝当前 VPN 源 IP；代理 env 来自 start-web.sh 旧注释「机器无直连外网」已过时 | 对照实验（代理 403 vs 直连 200）→ human 裁决搜索走直连 → start-web.sh 删 proxy export + dev 干净 env 重启 + domain/env.md 记录回滚条件 | **已根治**（env 配置修复+文档化；research 后续 web 工具直连可用） |
| 6 | web 会话分配编号与手工预建 loop 目录错位（备好 loop_16 产物，会话分配 loop_17） | PrepareHumanLoopDirsIn 分配 next loop_N，预建目录使分配偏移 | 先建会话再拷产物进分配目录（loop_17/18/19/20/21/22 六次迁移全部成功） | **模式级根治**（顺序约定固化；无代码改动需求） |
| 7 | steer/prompt API 试错（{"text"}→409 use steer；steer {"text"}→message must not be empty） | 会话 streaming 时须用 steer；字段名是 message 非 text（web/sessions.go 契约） | 读 routes.go/sessions.go 源码确认字段 | **已根治**（用法知识，一次性） |

**复发风险最高**：问题 4（release 门禁竞态——下次 release 若 dev web 有活动会话还会挂）与问题 1（workflowScript 转义——派发长任务文本时仍可能踩）。
