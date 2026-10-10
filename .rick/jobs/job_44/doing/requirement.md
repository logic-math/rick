# Job 44 需求（原始 + Grilling 澄清）

## 原始需求

迭代升级
1. web ui 的交互体验
2. human-loop 模式效果优化，升级架构与流程获得更好的执行成果

---

# Grilling 澄清结论（2026-10-05，设计树见 doing/grilling/design-tree.md）

## 全局 OKR（human 确认）

- **O**：rick 自改进升级——①web 新增 humans 学习页面 + 书写体验（草稿持久化）修复；②human-loop 从「机械五阶段引导」重构为「教学内容内化 SENSE 流程」的教师会话，教学能力可验证提升。
- **KR1（web UI）**：humans 页面上线（human-loop 产物 = 学习材料 + 学习进度，可浏览复习）；草稿不因页面刷新/重载/会话中断丢失。
- **KR2（human-loop）**：重构后学习成果质量可验证提升（5 维 rubric + 结构指标，dev 实例真实主题试运行评审）。
- **KR3（工程）**：三层门禁全绿 + RSI 流程发布生产（dev 工作区实现 → 人类确认 → merge）。

## 需求 1：web UI（human 裁决）

1. **新增 humans 工作区页面**（与 sessions/jobs/dreams 并列）：展示 human-loop 过程产物（学习材料 + 学习进度），human 直接查看复习即可。**明确不要**阶段进展条、判断点卡片。
2. **书写体验**：不得丢失未发送草稿。
   - **Grilling 事实修正（L4 回流）**：空闲回收（30min reap）**在当前代码中已禁用**（job_43：`internal/cmd/web.go resolveSupervisorConfig` → `IdleTimeout: runtime.IdleDisabled`，web 会话持久持有至 server 生命周期）。故「idle 回收配置化」无必要，原推荐作废。
   - **真实丢失路径**（代码核实）：①手动刷新页面 ②`frontend_reload` SSE 事件 → `web/src/api/sse.ts` `window.location.reload()`（每次前端产物变更自动触发）③路由切换/组件重挂载 ④会话挂起后重载。根因：ChatInput 的草稿是纯 React `useState`，无任何持久化。
   - **方案**：ChatInput 草稿按会话 id 持久化到 localStorage（挂载恢复、输入防抖保存、发送成功清除、localStorage 不可用时静默降级）。纯前端改动，无后端配置变更。

## 需求 2：human-loop 重构（human 裁决）

1. **架构**：只剩**一个主体提示词**（`sense_loop.md` 文件名保留，调度全局节奏）+ **两个 subagent**：
   - **research**：输入 = 历史上下文 + human 需求 → 输出事实材料 `research/research_{N}.md`（尽调树 + 信源等级 + ## Gaps）
   - **teach**（`think.md` 删除改名）：输入 = `research_{N}.md` + 历史对话 → 做 SENSE 思考并产出教学课程 `lessons/teach_{N}.md`；**所有给 human 呈现的内容都由 teach 产出**；教学内容须引导 human 进行 SENSE 思考
   - **exporter 整个删除**（模板 + 技能 + agent 定义 + builder 链）
2. **SENSE 简化**：只告知 SENSE 思考流程 = **还原事实、呈现视角、洞察矛盾、启发逆转、良质批判**，解释每个概念的**本质**，**不再阐述方法论**（怎么还原/呈现/洞察/逆转/批判全部删除）。
3. **去机械化**：删除阶段序列（S/E/N1/N2/S-R/EC）、阶段标签、固定追问集、硬性跳转约束、回流计数器、系统论描述符、三维打分、批判门禁重试流程。SENSE 内化于流畅教学叙事（隐含，不标阶段）。
4. **教学节奏**（参照 mattpocock teach skill 调研，见 research-L2.md）：文件状态机驱动（mission 缺 → 面试；材料缺 → 调研；每回合一课）；一课一事 + 一次有形收益 + working memory 内；每课引注事实材料并荐 primary source；认知强制（呈现分析前先请 human 表态）；teach 隐式检查 human 判断作为隐式 quiz。
5. **产物布局**：
   ```
   .rick/draft/loops/loop_N/
     mission.md               # 学习目标 + 考核标准 + 起点（面试产出）
     research/research_{N}.md # 事实材料（research 产出）
     lessons/teach_{N}.md     # 教学节（teach 产出，学习材料主体）
     judgment.md              # human 判断原话时间线（表达 = 学习进度）
     prompts/                 # sense_loop + teach + research（3 文件）
     session_id               # 恢复机制不变
   ```
6. **退出准则**：mission 学习目标达成 + human 明示完成 → 收尾（**不再产 RFC**）。需要总结时另起 easy job 消费 human-loop 工作文件夹；easy/plan 可直接消费该文件夹指导后续工作。
7. **保留**：session_id 恢复、judgment.md 禁写 AI 推理、research 尽调树 + 信源加权、派发显式语法（workflowScript + timeoutMs）与交付门禁（回执 + 文件校验）。

## 执行方式（human 裁决）

- **dev 工作区实现**（/workdir/sunquan20/rick-dev，dev/self-evolve 分支），门禁全绿 + dev 实例试运行 + human 确认后 `rick tools release --merge-source` 合并生产（RSI 流程 S0→S7）。
- 分层门禁：gate1 模板/agent/builder 链；gate2 web 后端 + 前端；gate3 dev 验证 + RSI 证据。
- 评估（KR2）：5 维 rubric（教学流畅度/SENSE 内化/事实锚定/提问启发性/学习成果质量）+ 结构指标（lessons ≥3、judgment ≥5、每课有引注），human 在 dev 实例试运行后评审。
