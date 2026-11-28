# skill:dev-session-migration（dev 会话继承迁移——loop 续思）

## 触发场景

human-loop 会话需要在新 loop 会话继续时使用：prompt 模板更新后旧会话仍用旧快照（每会话 prompts/ 是启动时渲染副本）、会话上下文过长、或 human 明示「继承/迁移到新 loop」。信号词：「继承 loop_N 到新的 loop_N+1」「迁移并继承进度，继续推进」「开启 loop_N+1 继续讨论，使用新的会话」。

## 预期效果

- 新会话从正确环节直接继续（跳过已完成步骤——继承注记指明进度边界与下一步起点）
- 产物全量继承（v8+：thought_path.md + orig/；以当前 loop 实际产物为准，不硬编码清单）
- 零内容丢失（job_50 六次 + job_57 四次实测全成功：loop_15→17→18→19→20→21→22 + loop_19→20→21→22）

## 核心内容

1. **先建会话**：`POST /api/sessions`（`{"workspace_id":"<ws>","type":"human-loop","params":{"topic":"<主题——继承 loop_N（进度边界）"}}`）→ 自动分配新 loop_N
2. **拷贝产物**：前 loop 目录的全部产物 → 新 loop_N/（全量；v7 时代是 mission/zpd/values/syllabus/judgment/lessons/research，v8+ 是 thought_path.md + orig/——按实际存在的产物拷，勿硬编码清单）
3. **继承注记**：thought_path.md（v7 时代 mission.md）顶部写「承接链（loop_19→…→loop_N）+ 当前进度边界（哪些阶段已完成、进行到哪一轮）+ 下一步起点」。**剥离旧注记规则（精确）**：替换策略 = 剥离到第一个正文标题行（如 `# 思考轨迹`）之前的**全部**注记块（不限层数——只剥一层会残留更早 loop 的注记头），再写新注记。python 片段：
   ```python
   lines = body.split('\n'); start = next((i for i,l in enumerate(lines) if l.startswith('# 思考轨迹')), 0)
   body = '\n'.join(lines[start:]) if start > 0 else body
   ```
4. **steer 就位**：`POST /api/sessions/{id}/steer`——**字段名 `message`（非 text）**；streaming 会话用 steer（prompt 会 409 use steer）——指示直接进入正确环节（「全部产物已迁移，按注记的下一步起点直接继续，先读 prompts/skill_*.md，不要重做已完成阶段」）。**硬规则：模板层已升级时禁止用 steer 传达新模板逻辑**——steer 只能传达一次性指令，不换 prompts 快照；模板变更生效的唯一路径 = 迁移到新 loop_N（新会话启动时渲染新模板）。suspended 会话：steer/prompt 都会 state_conflict——先 `POST /api/sessions/{id}/resume` 再 steer
5. **验证就位**：等首个 text 输出确认进入正确环节（模型 TTFB 可达数分钟，看 pi session transcript 的 assistant text）

## 禁止

- **先手工预建 loop 目录再建会话**——PrepareHumanLoopDirsIn 分配 next 编号，预建使会话分配偏移到 N+1（job_50 实测：备好 loop_16 产物，会话分配 loop_17）
- 拷 session_id 期望恢复旧会话——跨 HOME 的 pi 会话不可恢复，续思一律靠产物继承（本 skill）
- **用 steer 传达新模板逻辑**（见第 4 步硬规则——job_57 实测：steer 后 human 纠正「你需要新开 loop_N+1 才能触发新的 prompt 逻辑」）

## 关联

- prompt 模板层迭代节奏（commit→dev-web up→init-pi→新 loop 迁移）见 prompt-template-iteration-loop；job_57 实测 dev-web 管理命令须直跑 dev 树 `./bin/rick`（禁 go run——进程收割），status 的 fingerprint=MISMATCH 竞态以 health 端点 build_id 为真值
