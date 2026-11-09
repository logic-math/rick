# skill:dev-session-migration（dev 会话继承迁移——loop 续学/续思）

## 触发场景

human-loop 教学会话需要在新 loop 会话继续时使用：prompt 模板更新后旧会话仍用旧快照（每会话 prompts/ 是启动时渲染副本）、会话上下文过长、或 human 明示「继承/迁移到新 loop」。信号词：「继承 loop_N 到新的 loop_N+1」「迁移并继承进度，继续推进」。

## 预期效果

- 新会话从正确环节直接继续（跳过已完成步骤——mission 继承注记指明进度边界与下一步起点）
- 产物全量继承：mission/zpd/values/syllabus/judgment/lessons/research
- 零内容丢失（job_50 六次实测全成功：loop_15→17→18→19→20→21→22）

## 核心内容

1. **先建会话**：`POST /api/sessions`（`{"workspace_id":"<ws>","type":"human-loop","params":{"topic":"<主题——继承 loop_N（进度边界）"}}`）→ 自动分配新 loop_N
2. **拷贝产物**：前 loop 的 mission.md / zpd.md / values.md / syllabus.md / judgment.md / lessons/、research/ → 新 loop_N/（全量；judgment 的原创记录是探索流原料）
3. **继承注记**：mission.md 顶部写「承接链（loop_14→…→loop_N）+ 当前进度边界（哪些课/环节已完成、费曼到哪）+ 下一步起点」
4. **steer 就位**：`POST /api/sessions/{id}/steer`——**字段名 `message`（非 text）**；streaming 会话用 steer（prompt 会 409 use steer）——指示直接进入正确环节（「全部产物已迁移，按 human 指令直接进入 <环节>，先读 prompts/skill_*.md，按输出纪律执行」）
5. **验证就位**：等首个 text 输出确认进入正确环节（如呈现原创信息点清单、发出费曼引导）

## 禁止

- **先手工预建 loop 目录再建会话**——PrepareHumanLoopDirsIn 分配 next 编号，预建使会话分配偏移到 N+1（job_50 实测：备好 loop_16 产物，会话分配 loop_17）
- 拷 session_id 期望恢复旧会话——跨 HOME 的 pi 会话不可恢复，续学一律靠产物继承（本 skill）

## 关联

- 迁移产物含 anchors.md/zpd.md 时，新会话 ZPD 初始化直接读档案建档（跳过冷启动探针）
- prompt 模板层迭代节奏（commit→merge→dev-web restart→init-pi）见 step2 M3（候选 `rick tools dev-web sync` 命令化）
