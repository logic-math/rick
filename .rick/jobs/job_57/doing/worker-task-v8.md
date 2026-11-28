# Worker 任务书：human-loop v8 重构实现（dev 树）

## 工作目录与环境

- **工作目录**：`/Users/logic/workdir/rick-dev`（dev/self-evolve 分支，已同步 main@15c2505）——所有改动在此树
- 参考设计文档（只读，位于生产树）：
  - 设计树（唯一设计依据）：`/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_57/doing/grilling/design-tree.md`
  - 需求澄清结论：`/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_57/doing/requirement.md`
  - 调研简报（设计依据，按需读）：同目录 research-L1.md / research-L1-r2.md
- 遇到编译/测试失败先搜 `/Users/logic/workdir/AI_CODING/rick/.rick/domain/bugs.md` 与 domain/ 目录

## 任务目标

human-loop 彻底重写为 v8「SENSE 五阶段思考门禁制 loop」（human 已裁决设计，见设计树——**不重新设计，按设计落实**）：

- 目标：启发 human 进行思考；本质 = 通过学习提升自己
- SENSE 五阶段（问题还原 → 视角生产 → 判断矛盾 → 辩证逆转 → 良质批判），每阶段 = 一个 skill（①启发用户深入思考的内容 ②验证用户的思考是否足以通过思考门禁），通过门禁才进下一阶段，main agent 内联判定门禁
- 一个 orig agent：基于用户当前思考进度搜索调研信息、返回启发内容（不是答案）；唯一提示词原则：「给用户返回怎样的内容才能引导他完成本阶段的思考目标？」
- 未过门禁：AI 一句话指出缺口 + 重调 orig 针对性再启发，无硬性重试上限，推进权在 human
- 唯一产出物：`loop_N/thought_path.md`（human 回答 + AI 启发的时间线追加式）+ `loop_N/orig/stage{N}_{seq}.md`（orig 中间结构）

## 改动清单（全部执行）

### 1. 新建五个阶段 skill（internal/prompt/templates/skills/）

- `sense_reconstruct.md`（阶段 1 问题还原）
- `sense_perspective.md`（阶段 2 视角生产）
- `sense_contradiction.md`（阶段 3 判断矛盾）
- `sense_reversal.md`（阶段 4 辩证逆转）
- `sense_critique.md`（阶段 5 良质批判）

每个文件统一结构（内容从 internal/prompt/templates/skills/sense.md 该步的权威文本与可操作定义提炼——先读 sense.md）：
1. **本阶段思考目标**（sense.md 该步权威文本，一小段）
2. **orig 派发契约**（task 模板：当前阶段 + human 思考进度摘要 + 缺口 → orig 的唯一提示词「给用户返回怎样的内容才能引导他完成本阶段的思考目标？」；落盘 `{loop_dir}/orig/stage{N}_{seq}.md`）
3. **启发呈现纪律**（呈现 orig 返回的启发内容：事实/张力/引导性问题，**不供答案**——只引导 human 自己完成本阶段思考；呈现后留一个引导性问题，human 的回答是产出）
4. **门禁标准**（从 sense.md 该步可操作定义提取的可判定条件，如：问题还原 = 期望与现状分清 + 事实与价值分离；视角 = 强/弱相关框架清单及理由；矛盾 = 主要矛盾定位（全局性/根本性/决定性）多备选；逆转 = X 阻碍变 Y 前提的多可行方案；批判 = 假设置信自检 + 缺口显式）
5. **未过门禁行为**（一句话指出缺口 + 重调 orig 针对缺口再启发；human 可随时说「推进」——推进权在 human）

### 2. 新建 orig agent skill（internal/prompt/templates/skills/orig.md）

内容：
1. **唯一提示词（核心原则）**：「给用户返回怎样的内容才能引导他完成本阶段的思考目标？」——你的一切调研与返回都为回答这个问题服务
2. **输入**：main agent 传入的阶段 + human 思考进度摘要 + 缺口
3. **调研方法**：web_search 多 provider / fetch_content / OpenAlex 等 API（提示：不要依赖单一搜索源）
4. **返回**：启发内容（事实/张力/引导性问题——**不是答案**，不替 human 完成思考）
5. **中间结构落盘**：调研素材写 `{loop_dir}/orig/stage{N}_{seq}.md`（write 首块 + bash 分批追加，立即可交付性优先）

### 3. 重写主模板（internal/prompt/templates/sense_loop.md，旧内容全部删除）

新内容（极薄编排层）：
- 标题与占位符：主题 {{topic}}、会话目录 {{loop_dir}}、思考轨迹 {{thought_path_path}}、orig 中间结构目录 {{orig_dir}}、五个阶段 skill 路径 + orig skill 路径 + sense canon 路径（占位符名以 builder 实际传递为准，见第 5 条）
- **核心循环**（五阶段，每阶段）：①读对应阶段 skill → ②派 orig（按派发契约）→ ③呈现启发内容 + 引导性问题 → ④human 思考回答 → ⑤main agent 按门禁标准判定 → 通过：宣告推进下一阶段；未过：一句话指出缺口 + 重调 orig 再启发（无上限，human 可说「推进」跳过）
- **thought_path.md 写入契约**：每回合工具层追加（阶段标记 + human 原话 + AI 启发内容引用），不在对话中播报写入动作
- **结束**：五阶段全部通过 → 简短回顾（思考轨迹要点）→ 会话结束；thought_path.md 是唯一产出物
- **输出纪律（极简版）**：编排动作不打字；human 说什么语言回什么语言
- **开始**：复述主题 → 直接进入阶段 1（读 sense_reconstruct.md 开始）

### 4. agents.go（internal/env/agents.go）

- rickAgents 删除四旧（research / syllabus-builder / lesson-writer / recall-review）
- 新增 orig：name=orig, skill=orig（skillSource() 默认 name 一致，skills/orig.md）, tools=fullTools, defaultContext=fresh, thinking=high, timeoutMs=3600000, 注释说明 v8 定位
- 更新顶部注释（v8 单 agent：orig）

### 5. handler + builder（internal/handler/human_loop.go、internal/builder/pibuilder.go、internal/prompt/ 渲染代码）

- handler：入口/dry-run/resume 保留；PrepareHumanLoopDirsIn 的 loop_N/ 目录下新增 orig/ 子目录创建 + thought_path.md（空文件或标题行初始化）
- 渲染（找到 GenerateHumanLoopPrompt / SaveHumanLoopPrompt 的实现，internal/prompt/ 下）：占位符更新为新模板所需（thought_path/orig 目录/五阶段 skill 路径/orig skill 路径）；七 skills 部署清单更新（五阶段 + orig + sense；删除旧七 skills）
- **关键检查**：dream 的渲染不受影响（dream 引用 sense skill——sense.md 保留；确认 dream 不引用被删 skills）；web 层（task11 引用 PrepareHumanLoopDirsIn）编译通过

### 6. 删除旧文件

- internal/prompt/templates/skills/：删 zpd.md、feynman.md、exploration_stream.md、syllabus_builder.md、lesson_writer.md、recall_review.md、research.md（**保留 sense.md**——dream 共用）
- 不删历史 loop 目录（.rick/draft/loops/ 是数据）

### 7. 测试更新（TDD：先改测试断言——RED，再实现——GREEN）

- internal/cmd/human_loop_test.go + internal/handler/human_loop_test.go：
  - 断言新关键字（必含）：`thought_path`、`orig`、门禁/思考门禁、五阶段名（问题还原/视角生产/判断矛盾/辩证逆转/良质批判）
  - 禁止旧概念残留（forbid）：`ZPD`、`费曼`、`课程`、`syllabus`、`lesson-writer`、`recall-review`、`教学大纲`、`价值裁决`（values.md 机制）、`mission`（学习使命三字段机制）
  - 注意断言粒度：核心关键字即可，不强约束具体表述（protocol-redesign-loop 硬性约束 4）
- 检查 internal/env/ 的 agent 部署测试（如有）同步更新为单 orig
- dry-run 冒烟：`go run ./cmd/rick human-loop --dry-run <topic>` 输出 `grep -c '{{'` = 0（占位符全渲染）

## 执行流程（easy doing_loop）

ANALYZE（读设计树/sense.md/相关代码）→ RED（先改测试断言，跑红）→ GREEN（实现全部改动，跑绿）→ REFACTOR（命名/去重/注释精炼）→ COMMIT

## Commit 纪律（分多个 commit，每个可独立 revert）

1. `refactor(human-loop): v8 骨架——删旧教学 skills 与四 agent，agents.go 单 orig`（删除 + agents.go + orig.md + 五阶段 skills 骨架）
2. `feat(human-loop): v8 主模板与渲染管线——sense_loop 极薄编排层 + thought_path/orig 产物`（模板 + handler/builder + 占位符）
3. `test(human-loop): v8 断言——新关键字必含 + 旧概念 forbid`（测试更新与收尾修正）

（顺序可按实际依赖调整，但保持每个 commit 前后 build/test 不挂）

## 验证（全部通过才算完成）

- [ ] `go build ./...` 成功
- [ ] `go test ./internal/...` 全绿（至少 handler/cmd/env/builder/prompt 相关包）
- [ ] `go run ./cmd/rick human-loop --dry-run <任一主题>` 渲染冒烟：无未渲染占位符（`grep -c '{{'` = 0）、含五阶段与门禁关键字、无旧概念
- [ ] git log 显示分步 commits

## 完成回执

输出：改动文件清单（新建/修改/删除）+ 验证结果摘要 + 遗留问题（如有）。
