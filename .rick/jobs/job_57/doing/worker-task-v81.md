# Worker 任务书：human-loop v8.1 增量（实测反馈修正）

## 工作目录

- `/Users/logic/workdir/rick-dev`（dev/self-evolve，当前 HEAD 9a1c8fe）
- 设计依据（只读）：`/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_57/doing/grilling/design-tree.md` 的「v8.1 迭代」节（human 三层修正裁决原文）

## 背景

v8（commit 9a1c8fe）已落地：SENSE 五阶段门禁 loop + orig agent + thought_path。orig 首次实测（检索「原创性思考/科研品味」）成功返回启发内容，human 实测反馈三项修正。本轮是 v8.1 增量：三个文件群改动。

## 改动清单

### 1. 新建教学润色 skill：`internal/prompt/templates/skills/teacher_polish.md`

内容（中文，简洁——参照目录内其他 skill 的篇幅与文风）：

- 定位：human-loop v8.1 呈现层方法论（main agent 会话内执行，非 agent）——orig 返回的启发内容经本 skill 润色后呈现给用户
- **科普视角写作**：把用户当作该内容所属领域的初学者
- **句子规范（参考 ASD-STE100 技术写作）**：短句；一句一义；主动语态；主语明确；少嵌套——不套用受控词典，不写长句
- **内容要求**：准确（不损失事实性与来源）；通俗；有逻辑；有趣味
- **必须标注三态**：已知结论（确认的）；争议（存在分歧的）；知识边界（未知/适用范围）
- 润色只改表达，不改事实与来源

### 2. 重写 orig 检索规则：`internal/prompt/templates/skills/orig.md` 的「调研方法」节

用「概念跳跃法」替换原「调研方法」节（其余节保留：唯一提示词/输入/返回/中间结构落盘/输出纪律）。新检索规则（human 裁决原文为准）：

- **检索链**：寻找主概念领域 → 找到该领域最具启发力的信息源 → 基于该信息源寻找其他更具启发力的信息源（链式推进）
- **检索关键词核心原则：差异性**——与核心概念存在差异、但有相似性
- **概念跳跃**：跳跃的关键 = 选择**同一概念的差异性**（同中求异）+ **差异概念的同一性**（异中求同）；不区分收敛/发散等结构化内容，只搜索信息、在概念之间跳跃
- **大量跳跃**收集概念与解释 → 以唯一标准筛选：**这些概念哪些是最能启发用户推进思考、通过当前门禁目标的？**

### 3. 重写五阶段 skill 的门禁标准（每个文件只改「门禁标准」节，human 原文为准）

- `sense_reconstruct.md`（问题还原）：期望、现状、差距三个结构化表述说清楚；能清晰地明确价值判断和事实判断
- `sense_perspective.md`（视角生产/选择）：明确描述选择这一视角所带来的应用价值；能够描述其内在的因果运动
- `sense_contradiction.md`（判断矛盾）：基于这一视角对问题进行解释；定义决定性/根本性/全局性的矛盾来描述其因果运动
- `sense_reversal.md`（辩证逆转）：符合逻辑地回答——当阻碍 X 是目标 Y 达成的必要条件时，应当如何？
- `sense_critique.md`（良质批判）：回顾思考过程所依赖的假设是否可证伪；推导过程的逻辑性

### 4. 主模板小改：`internal/prompt/templates/sense_loop.md`

- 「呈现启发」环节（核心循环第 3 步）：改为「orig 返回的启发内容**经教学润色 skill 润色后呈现**（科普视角：短句、一句一义、主动语态、主语明确、少嵌套；标注已知结论/争议/知识边界）+ 阶段 skill 规定的引导性问题」
- 顶部 skill 路径清单增加教学润色 skill 占位符 `{{teacher_polish_skill_path}}`
- 开始节相应加一句：读教学润色 skill（呈现前必读）

### 5. 渲染管线：占位符接线

- 找到 builder/prompt 渲染层（SaveHumanLoopPrompt / GenerateHumanLoopPrompt 的 skills 部署清单与占位符映射），新增 `teacher_polish_skill_path` 占位符 + teacher_polish skill 部署（八 skills：五阶段 + orig + sense + teacher_polish）

### 6. 测试同步

- 断言新关键字：`teacher_polish`（或「教学润色」「科普」）、「概念跳跃」（或「差异性」）、「期望」「现状」「差距」（问题还原门禁）、「因果运动」「可证伪」等核心词
- 保持 forbid 旧概念不变（ZPD/费曼/syllabus 等）
- dry-run 冒烟 `grep -c '{{'` = 0

## Commit

单 commit：`feat(human-loop): v8.1 实测反馈——教学润色呈现层 + orig 概念跳跃检索 + 五门禁验证规则`

## 验证

- [ ] `go build ./...`
- [ ] `go test ./internal/handler/ ./internal/cmd/ ./internal/env/ ./internal/builder/ ./internal/prompt/`
- [ ] `go run ./cmd/rick human-loop --dry-run <主题>`：占位符 0 残留；含教学润色/概念跳跃/新门禁关键字；无旧概念

## 完成回执

改动文件清单 + 验证结果 + 遗留问题。
