APPROVED: true

# Job job_50 执行总结

## 执行概述

**项目目标**: human-loop v7 重设计——实现「启发人类进行原创思考」的核心目标（从 ADR-human-loop-v7 出发，grilling 16 问全裁决）。
**实际完成**: v7 → v7.6.2 共九轮定向增量（每轮 human 在 dev 实跑一个 loop → 一句话反馈 → 定向增量 → build/test → dev 部署 → 下一轮实测）→ release 生产（源码+二进制同版）→ loop_22 思考记录迁移入生产 → draft loops 清理（20→5，-7.7MB）。
**整体评价**: ⭐⭐⭐⭐（4/5）——目标全达成且经 7 轮实测收敛；扣分项：探索流两版方案（混沌学园三动作/SENSE 状态机）实测被废造成两轮返工，release 门禁竞态耗 ~30 分钟。

## 关键成就

1. **human-loop v7.6.2 全量落地并发布生产**：五步教学循环（结构化提问→ZPD 生成→大纲→每课子循环→总结与统一探索流）+ 4 agent（research/syllabus-builder/lesson-writer/recall-review）+ 3 parent 内联 skill（zpd/feynman/exploration_stream）+ 核心教学原则（学生先自己推再对照作者=启发推导自己的 SENSE 思考）
2. **实测驱动的迭代形态**：7 轮「dev 实跑→一句话反馈→定向增量→部署→复测」≈15 分钟/轮，探索单点突破（输出纪律/费曼呈现纪律/靶向卡点调研）全部由会话实测病灶分析驱动（38 条 text 逐条分析定位编排叙述泄露）
3. **排查文化落地**：Squid 代理 403 断网根因定位（ACL 拒绝源 IP→直连裁决）、release 门禁竞态根因定位（buildEnv HOME=dev-home 与活动 dev web 会话共享 pi sessions），均沉淀为可复用知识
4. **loop_22 思考记录完整入生产**：22 课 infer RSI 学习全程（judgment 全量原创观点 #1–#28 + 费曼记录 + 终稿综述 + 探索流记录），后续思考可经新会话继承继续

## 问题与教训

### 问题1: release 门禁连挂 ×4（web 包 TempDir cleanup 竞态）

**根本原因**: 门禁 buildEnv 用 HOME=dev-home（release.go DevHomeOr），web 测试 spawn 的 worker 与在跑 dev web 实例（活动会话）共享该 HOME 的 pi runtime/sessions → worker 异步终止滞后于 t.TempDir 清理 → RemoveAll 竞态失败（单测/低载通过、门禁全量必挂）
**解决方案**: release 前 `rick tools dev-web down` 消除竞争（已沉淀 rick_release_ops_skill 第 8 条预检）；代码级根治（cleanup 前 wait worker 退出）记 domain/bugs.md backlog
**经验教训**: 测试与运行环境共享 HOME 的隔离假设不可靠；「规避解」须与「根治 backlog」同时留档

### 问题2: workflowScript task 文本转义连挂 ×3

**根本原因**: 长文本内嵌单引号 JS 字符串——JSON `\n` 解码为真实换行即 unterminated string、撇号破坏字符串；证据混合（部分含 `\n` 的 task 曾成功）说明与文本组合相关，无法靠局部规避
**解决方案**: task 超 1 行/含引号撇号换行一律文件化（write 任务书 + 单行指针 task）——本 job 后续 0 失败（已沉淀 pi_orchestration_syntax_skill 第 7 节）
**经验教训**: 不可靠的隐式约束要升级为显式纪律；文件化比转义可靠一个量级

### 问题3: 探索流「呈现质量」两版被废（呈现不启发）

**根本原因**: v7.4 内联 web_search 信息太薄、v7.5「相似异质性概念」太泛——启发的本质是**信息与学生自己思考的张力**（对准具体卡点），薄和泛都不产生启发；另：呈现不说人话（无概念定义）则理解失败，理解失败启发为零
**解决方案**: 方案 A 靶向卡点调研（进程判定=SENSE 阶段+具体卡点→靶向 research→张力呈现最小充分集+引导性问题）+ 费曼呈现纪律（初学者假设/最少概念/概念必附准确定义）
**经验教训**: 设计假设（探索流方法）必须经 human 实测检验；「呈现质量」的验收标准是「学生能读懂并被启发」，不是「信息正确」

## 知识沉淀清单

- [x] skills/dev_session_migration_skill/skill.md — dev 会话继承迁移模式（六次成功迁移：先建会话→拷产物→mission 注记→steer(message 字段)→验证首输出；禁止先预建 loop 目录）
- [x] skills/rick_release_ops_skill/skill.md（更新）— +第 8 条：release 前预检 dev-web down（门禁 HOME 竞争竞态根因+规避）
- [x] skills/pi_orchestration_syntax_skill/skill.md（更新）— +第 7 节：派发纪律（task 文件化 + runs.all 描述符对象形态）
- [x] loops/prompt-template-iteration-loop.md — prompt 模板实测反馈迭代（Go embed 生效链/agent 部署三连/会话快照与继承迁移/dry-run 冒烟；7 轮实测 ≈15 分钟/轮）
- [x] loops/web-feedback-iteration-loop.md（更新）— +分工路由头（模板层反馈→prompt-template-iteration-loop；协议大重构→protocol-redesign-loop）
- [x] domain 事实沉淀（Step 5）：env/bugs/commands 追加（门禁 HOME 竞争事实、web 测试竞态根治 backlog、steer 契约行）
