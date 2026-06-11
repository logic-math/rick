## task1: 创建 debug_skill.md（三阶段调试 SOP + review debug agent 协议）

**分析过程 (Analysis)**:
- 阅读了现有 `internal/prompt/templates/skills/super-debugging-zh.md` 了解已有调试技能格式
- 阅读了 `internal/prompt/templates/skills/sense.md` 了解 SENSE 方法论结构
- 确认 `internal/prompt/templates/skills/` 目录已存在，直接在其中创建新文件
- 任务要求内聚三阶段 SOP、review debug agent 协议、bug 文件格式规范、SENSE 方法集成

**实现步骤 (Implementation)**:
1. 设计文件结构：frontmatter → 铁律 → 准备阶段 → 阶段一 → 阶段二 → 阶段三 → review debug agent 协议 → 流程图 → 完整示例 → 反模式
2. 在准备阶段定义 debug/ 目录约定、bug 编号规则、YAML frontmatter 格式规范、两种合法终止状态
3. 在阶段一（源码推理法）定义 review debug agent 触发点（建立假设时）、主 Agent 执行-回滚-记录循环、上限 3 次
4. 在阶段二（增量调试法）定义 review debug agent 触发点（简化复现时）、基线判断逻辑、无基线跳过规则
5. 在阶段三（科学实验法）定义两个触发点（简化复现 + 传播链假设）、运行时工具列表（delve/pprof/pdb/strace）、上限 5 次、超限处理流程
6. 在 review debug agent 协议章节明确输入/输出格式/角色约束，并内嵌 SENSE 方法（./skill_sense.md 硬编码路径）
7. 添加三阶段递进文字流程图和完整 bug 文件示例

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`python3 .rick/jobs/job_16/doing/tests/task1.py`
- 测试输出：
  ```
  {"pass": true, "errors": []}
  ```
- 结论：✅ 通过

## task2: 更新 doing.md / plan.md 模板引用，删除 super-debugging-zh.md

**分析过程 (Analysis)**:
- 阅读了 `internal/prompt/templates/doing.md`、`plan.md`、`easy.md` 确认所有 super_debugging 引用位置
- 确认 `internal/prompt/templates/skills/super-debugging-zh.md` 存在，需用 `git rm` 删除
- doing.md 中 DEBUG 铁律、skill 列表、Commitment、Scarcity 章节均需更新；debug{N} 格式整节需删除
- plan.md 中 task.md 格式的调试方法章节需替换 super_debugging_skill_path → debug_skill_path
- easy.md 中 skill 列表第2条需替换为 debug-skill（测试也检查了此文件）

**实现步骤 (Implementation)**:
1. `git rm internal/prompt/templates/skills/super-debugging-zh.md`
2. doing.md：Line 3 声明改为 skill:debug-skill
3. doing.md：skill 列表第3行替换为 debug-skill，新增 sense_skill_path 行
4. doing.md：DEBUG 铁律章节替换为三阶段调试法
5. doing.md：Commitment 块改为 debug-skill
6. doing.md：Scarcity 章节更新 Phase 1 描述
7. doing.md：删除"遇到问题时的详细记录"debug{N} 整节，替换为单行 bug{n}-{描述}.md 指引
8. plan.md：调试方法章节替换为 debug_skill_path + 三阶段调试法描述
9. easy.md：skill:super-debugging 条目替换为 skill:debug-skill

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`python3 .rick/jobs/job_16/doing/tests/task2.py`
- 测试输出：
  ```
  {"pass": true, "errors": []}
  ```
- 结论：✅ 通过
