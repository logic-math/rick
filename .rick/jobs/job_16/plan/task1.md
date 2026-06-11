# 依赖关系

（无依赖）

# 任务名称

创建 debug_skill.md（三阶段调试 SOP + review debug agent 协议）

# 任务目标

在 `internal/prompt/templates/skills/debug_skill.md` 创建完整的调试技能文件，内聚三阶段 SOP（源码推理法→增量调试法→科学实验法）、review debug agent 协议（两个触发点）、运行时观察工具指引和 debug/ 目录约定，成为 Rick 调试能力的唯一核心 skill。

# 关键结果

1. 文件路径：`internal/prompt/templates/skills/debug_skill.md`，frontmatter 包含 `name: debug-skill`，`description: 遇到任何 bug、测试失败或不符合预期的行为时加载`
2. 准备阶段：明确 debug/ 目录创建路径（当前工作目录下）、bug 编号规则（全局递增、不按 task 重置）、文件命名 `bug{n}-{简要描述}.md`；**bug*.md 文件格式规范**：每个文件必须以 YAML frontmatter 开头（`---` 包裹），包含两个必填字段：
   - `summary`：一句话描述根因 + 最终状态（如：`"并发竞态导致数组越界，已通过 sync.Mutex 修复"`）
   - `status`：当前状态（`"✅ 已解决"` / `"🔄 进行中"` / `"❌ 无法修复"`）
   - 示例：
     ```yaml
     ---
     summary: "并发竞态导致数组越界，已通过 sync.Mutex 修复"
     status: "✅ 已解决"
     ---
     ```
   - 这个 frontmatter 是 Rick 框架读取摘要的唯一来源，不得省略；框架通过解析两个 `---` 之间的内容提取 `summary` 和 `status`，不读取文件其余部分
   - **bug.md 有且仅有两种合法终止状态**，任何 bug.md 最终必须以其中一种结束，不得悬置：
     1. `status: "✅ 已解决"` —— bug 已修复，测试全部通过
     2. `status: "❌ 无法修复"` —— 有扎实论证说明 bug 在当前约束下无法修复，需要人类决策；论证内容记录在 `## 结论` 章节，不可空置
3. 阶段一（源码推理法）：review debug agent 在建立假设时触发（subagent 大范围读源码，输出假设列表后退出，不执行改动）；主 Agent 执行-回滚-记录循环；最多 3 次尝试；每次尝试前 `git diff` 必须干净
4. 阶段二（增量调试法）：review debug agent 在简化复现步骤时触发（subagent 读测试输出/源码，产出最小复现建议和基线方案）；基线判断逻辑（git log 历史 commit 或最小配置）；无基线则跳过，进入阶段三
5. 阶段三（科学实验法）：review debug agent 在简化复现时触发（同阶段二触发点）+ 在建立错误传播链假设时触发；运行时观察工具章节（指引 AI 自行安装使用 delve/pdb/gdb/profile 等，禁止 mock）；最多 5 次实验循环；**超限处理**：停止所有尝试，在 `bug{n}-xxx.md` 的 `## 结论` 章节写入「论证无法修复的原因」（包含：已尝试的方法、已排除的假设、当前约束与边界、建议人类决策的方向），同时将文件顶部 frontmatter 的 `status` 更新为 `"❌ 无法修复"`，`summary` 更新为一句话说明根因和无法修复的关键约束
6. review debug agent 协议章节：明确定义 subagent 的输入（相关源码+测试输出+已有 debug 记录）、输出格式（假设列表/复现建议，每条包含观察证据+推断+建议）、角色约束（只输出分析，不执行任何改动，不写文件，输出后即退出）；**必须内嵌 SENSE 思考方法**：协议中显式要求 review debug agent 在启动后立即加载 sense skill。debug_skill.md 是静态文件，不支持模板变量，路径描述方式为**硬编码相对路径**：`./skill_sense.md`（与 debug_skill.md 同目录，doing 阶段二者均由 WriteSkillFile 写入同一 prompts 目录）。agent 启动后读取此文件，按 S→E→N 三阶段结构化分析 bug 信息后再输出假设，保证假设有 SENSE 方法论支撑而非随机猜测
7. 完整 bug 文件结构示例：文件第一行必须是 YAML frontmatter（`---\nsummary: "..."\nstatus: "..."\n---`），其后是三个阶段的记录（含 `## 结论` 章节）；frontmatter 是 Rick 框架提取摘要的唯一来源，结论章节供人类阅读
8. 三阶段递进关系图（文字形式）：阶段一上限3次→阶段二（无基线可跳过）→阶段三上限5次→超限升级人工

# 测试方法

**前置条件**：项目根目录为 `/Users/sunquan/ai_coding/CODING/rick`，无同名文件存在

**测试1：文件存在且结构完整**
- 操作：`cat internal/prompt/templates/skills/debug_skill.md | head -5`
- 预期：输出包含 `name: debug-skill` 的 frontmatter

**测试2：关键章节存在**
```bash
for section in "准备阶段" "源码推理法" "增量调试法" "科学实验法" "review debug agent" "运行时观察工具" "三阶段递进"; do
  grep -q "$section" internal/prompt/templates/skills/debug_skill.md && echo "✅ $section" || echo "❌ 缺失: $section"
done
```
- 预期：全部输出 ✅

**测试3：review debug agent 两个触发点**
```bash
grep -c "触发" internal/prompt/templates/skills/debug_skill.md
```
- 预期：输出 ≥ 4（阶段一建立假设、阶段二简化复现、阶段三简化复现、阶段三建立传播链）

**测试4：三阶段上限约束**
```bash
grep "3 次" internal/prompt/templates/skills/debug_skill.md && echo "✅ 阶段一上限" || echo "❌"
grep "5 次" internal/prompt/templates/skills/debug_skill.md && echo "✅ 阶段三上限" || echo "❌"
```
- 预期：两行均输出 ✅

**测试5：review debug agent 协议中声明加载 sense skill（硬编码相对路径）**
```bash
grep -E "skill_sense\.md|\./skill_sense" internal/prompt/templates/skills/debug_skill.md && echo "✅ sense skill 路径声明存在" || echo "❌ 缺少 sense skill 路径"
```
- 预期：✅ sense skill 路径声明存在

**测试6：两种合法终止状态文字描述均存在**
```bash
grep -c "无法修复" internal/prompt/templates/skills/debug_skill.md | awk '{print ($1>=1) ? "✅ 含❌无法修复终止状态" : "❌ 缺少无法修复终止状态"}'
grep -c "已解决" internal/prompt/templates/skills/debug_skill.md | awk '{print ($1>=1) ? "✅ 含✅已解决终止状态" : "❌ 缺少已解决终止状态"}'
```
- 预期：两行均 ✅

**边界用例**：不得使用模板变量格式引用 sense，且无旧 super-debugging 残留
```bash
grep -i "super_debugging\|super-debugging" internal/prompt/templates/skills/debug_skill.md && echo "❌ 残留旧引用" || echo "✅ 无旧引用"
grep "{{sense_skill_path}}" internal/prompt/templates/skills/debug_skill.md && echo "❌ 错用模板变量（应为硬编码相对路径）" || echo "✅ 无错误模板变量"
```
- 预期：两行均 ✅

# 调试方法

遇到任何不符合预期的行为，必须加载并遵循 super-debugging skill：
`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_16/doing/prompts/skill_super_debugging_zh.md`

执行顺序：S（还原问题）→ E（视角分析）→ N（验证假设）→ 修复实现 → 3 次失败则停止找人类协作者
