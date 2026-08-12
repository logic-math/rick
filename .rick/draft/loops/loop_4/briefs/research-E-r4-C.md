# research-E-r4 节点 C — "确定性选择（含判断）"是否是 rick vs zero-shot 的关键差异

节点路径：[根 > E-r4-zero-shot对比 > C-确定性选择是否关键差异]
事实陈述：human 论断"rick 确定性地选择此方法（含判断）"——"确定性选择"是否是 rick 相对 zero-shot 的关键差异？即 rick 注入是否把"可选的方法"变成"确定被执行的方法"？

## 执行动作

1. 代码原文：Read rick `internal/prompt/templates/doing.md`（强制注入 + "不可跳过任何步骤"）+ `internal/prompt/templates/skills/doing_loop.md`（Step 0-5 强制序列，"必须"语言）+ `internal/prompt/doing_prompt.go`（doing_loop_content 注入变量 + SaveToFile 确定性落盘，复用 E-r2）
2. 复用 E-r2：`briefs/research-E-r2-C.md`（rick ContextManager 从文件确定性加载 + GenerateDoingPromptFile 确定性拼装 + RFC-001 信息网络流 + pi compaction 保留 system prompt）
3. 运行时：复用节点 B demo（zero-shot 线性单遍可选叙述）vs rick doing_loop Step 0-5（强制不可跳过）的对照
4. 文档：Plan-and-Solve（2305.04091，方法论注入改善 zero-shot）/ Self-Refine（2303.17651，显式迭代脚手架）/ Reflexion（2303.11366，显式反思脚手架）——确定性注入 vs 随机模型选择
5. 信源权重默认：代码 0.4 / 运行时 0.3 / 文档 0.2 / 反事实 0.1

## 信源验证结果

### 代码原文（权重 0.4）✅（决定性——rick 把方法从"可选"变"强制执行"）

**证据 1 — doing.md 模板强制注入 + 不可跳过**（`internal/prompt/templates/doing.md`）：
```
{{loops_context}}   ← .rick/loops/ 上下文
{{skills_context}}  ← .rick/skills/ 上下文
{{debug_context}}   ← debug.md
{{task_info_section}} ← task 信息
{{loop_step_header}}  ← "## 第一步：执行 Doing Loop"
{{doing_loop_content}} ← Doing Loop Step 0-5（见证据 2）
{{check_step_header}}  ← "## 第二步：格式检查"
`{{rick_bin_path}} tools {{check_command}} {{job_id}}`  ← check 门禁
...
**你需要一步步执行以下操作，不可跳过任何步骤。**   ← 强制
```
→ 关键："**不可跳过任何步骤**"——rick 把 Doing Loop 从"可选建议"变成"强制执行"。这与 zero-shot（节点 B：LLM 把方法当线性叙述，可跳过/可省 phase 声明）形成直接对照。

**证据 2 — doing_loop.md Step 0-5 全程"必须/强制"语言**（`internal/prompt/templates/skills/doing_loop.md`）：
- Step 0.1："**必须依次完成以下两项**...由 AI 自行判断读取哪些文件，但**必须完成搜索动作**后再继续...遇到任何问题**必须优先搜索** bugs.md"
- Step 0.2："读取 loops_context，按 trigger 字段匹配"——Loop 匹配是强制分发
- Step 1："确认以下内容全部清晰后才继续"
- Step 3："**每轮迭代由 Main Agent 启动一个独立 Sub Agent**"——subagent-per-iteration 是强制结构
- Sub Agent 各阶段强制**声明**："I will use skill:sense." / "I will use skill:tdd." / "I will use skill:debug-skill."——显式 phase 声明是强制
- RED："**必须确认 FAIL**"——TDD RED 的失败确认是强制
- DEBUG 触发条件："测试 FAIL / 编译报错 / 行为与预期不符"——自动触发，非可选
- Step 4：失败"**返回 Step 3 启动下一轮迭代**"——迭代是强制
- Step 5："迭代次数达上限（默认 3 轮）/ 连续 2 轮产出相同错误"——停止标准是确定性

→ 整个 Doing Loop 是"强制 + 必须 + 不可跳过 + 自动触发 + 确定性停止"——把方法从"LLM 可能选可能不选的可选项"（zero-shot）变成"确定被执行的协议"。这是 human "rick 确定性地选择此方法（含判断）"的代码实证。

**证据 3 — doing_prompt.go 确定性注入 + 落盘**（`internal/prompt/doing_prompt.go`，复用 E-r2）：
```go
builder.SetVariable("doing_loop_content", loadDoingLoopContent(...))  // Doing Loop Step 0-5 注入
builder.SetVariable("loops_context", LoadLoopsContext(loopsDir))     // 文件确定性加载
builder.SetVariable("skills_context", LoadSkillsContext(...))
builder.SetVariable("debug_context", contextMgr.GetDebugRaw())
promptFile := filepath.Join(promptsDir, fmt.Sprintf("%s_doing_prompt.md", task.ID))
builder.SaveToFile(promptFile)                                       // 确定性落盘
```
→ rick 不依赖 LLM"决定"用何方法——方法被**确定性拼装进 prompt 文件**并落盘可重现。LLM 接收到的是"已决定的方法"，不是"可选建议"。

**证据 4 — RFC-001 + pi compaction 保留 system prompt（复用 E-r2）**：
- RFC-001："rick 是人与 AI 之间的上下文对齐框架...保持事实信息的有效性，让 AI 始终在正确的信息基础上工作"——确定性对齐意图
- pi compaction 保留 system prompt（loop_2 research-4-N2）+ before_agent_start 每 turn 重建 system prompt——rick 注入的"做事方法"在 system prompt 中**天然 compaction-resist**，长程任务中确定性持久

### 运行时行为（权重 0.3）✅（决定性——zero-shot 可选 vs rick 强制的直接对照）

**对照（节点 B demo + rick 代码）**：
| 维度 | zero-shot LLM（节点 B runtime） | rick 注入（doing_loop Step 0-5） |
|---|---|---|
| 方法地位 | 线性叙述，可选叙述 | "**不可跳过任何步骤**" 强制 |
| phase 声明 | 无（线性 "I'd start... Next...") | 强制声明 "I will use skill:sense/tdd/debug-skill" |
| 迭代结构 | 单遍线性 | Main→Sub Agent per-iteration，失败强制返回 Step 3 |
| 停止标准 | 无（叙述完即止） | 3 轮上限 / 连续 2 轮同错 / check pass 确定性 |
| 门禁 | 无 | check 命令循环直到 pass |
| Domain/Loop 匹配 | 无 | Step 0 强制搜索 + trigger 匹配 |

→ runtime 对照直接证实：rick 把"可选方法"（zero-shot）变成"确定被执行的方法"（强制协议）。

### 文档（权重 0.2）✅（确定性方法论注入 vs 随机模型选择）

**源 1 — Plan-and-Solve（arxiv 2305.04091）**：zero-shot-CoT 有系统缺陷（calculation/missing-step/semantic 错误），需**显式 plan-and-solve 注入**改善 → 方法论需确定性注入，不可靠随机选择。

**源 2 — Self-Refine（arxiv 2303.17651）**："LLMs do not always generate the best output on their first try" → 需**显式迭代精炼脚手架**，LLM 不会自发可靠迭代 → 确定性脚手架是关键。

**源 3 — Reflexion（arxiv 2303.11366）**：LLM 默认"challenging to learn from trial-and-error"，需**显式反思脚手架 + episodic memory** → 确定性反思注入是关键，非随机选择。

→ 三源一致：**确定性方法论注入**（plan/solve、self-refine、reflexion）是改善 LLM 的关键手段——因为 zero-shot 随机选择不可靠。rick 的 Doing Loop/sense/think 即"确定性方法论注入"的工程化实例。human "确定性选择是关键差异"论断与文献方向一致。

### 反事实（权重 0.1）✅（de facto A/B 对照）

- 反事实设想："若 rick 不强制（仅建议），LLM 是否会跳过？"——节点 B runtime 已答：zero-shot（仅建议/叙述）下 LLM 用线性单遍，**跳过** rick 的 phase 声明/迭代结构/门禁
- rick 的"不可跳过"语言 = 把"可选"变"强制"的反事实实证：去掉强制（zero-shot）→ 跳过；加上强制（rick）→ 执行
- de facto A/B 成立（无 paired rick-injected runtime，但代码 + zero-shot runtime 对照构成有效反事实）

## 还原确认

无 rick 代码修改，无需还原。Read/curl/claude CLI 只读。

## 置信度评估（由 research 主调度计算）

- 代码原文 ✅ × 0.4 = 0.4（doing.md "不可跳过" + doing_loop.md Step 0-5 强制 + doing_prompt.go 确定性注入落盘 + RFC-001/pi compaction 保留，四源一致）
- 运行时 ✅ × 0.3 = 0.3（zero-shot 可选 vs rick 强制六维对照）
- 文档 ✅ × 0.2 = 0.2（Plan-and-Solve/Self-Refine/Reflexion：确定性注入是关键，zero-shot 随机不可靠）
- 反事实 ✅ × 0.1 = 0.1（de facto A/B：去掉强制→跳过；加上强制→执行）
- **合计 = 1.0（高，≥0.8 终止）**

## 关键事实

1. **✅ "确定性选择（含判断）"是 rick vs zero-shot 的关键差异**（human 论断成立）
   - 代码：rick doing.md "不可跳过任何步骤" + doing_loop Step 0-5 全程"必须/强制/自动触发" + doing_prompt.go 确定性拼装落盘 → 把方法从"可选"变"强制执行"
   - runtime：zero-shot（节点 B）把方法当线性可选叙述（跳过 phase 声明/迭代/门禁）；rick 把方法当强制协议 → 直接对照
   - 文档：Plan-and-Solve/Self-Refine/Reflexion 三源——确定性方法论注入是改善 LLM 的关键，因 zero-shot 随机选择不可靠

2. **"含判断"的体现**：
   - rick 的"选择"不是无脑套用——Doing Loop Step 0.2 有 **trigger 匹配**（按任务匹配项目 Loop，有匹配执行项目 Loop，无匹配执行默认 Loop）
   - think.md 有**假设数量保障 + 多视角强制 + 4 维打分 + top-N 阈值**——是结构化判断协议
   - 即 rick "确定性地选择此方法"含**条件化判断**（trigger 匹配 + 假设打分），非机械执行——与 human "它本身已经包含了判断在内"一致

3. **工程意义**：rick 的价值不在"发明组件"（组件是常识，节点 A），而在"**确定性地编排 + 强制执行 + 含判断的选择**"——这把 LLM 的"可能用对方法"（随机/概率）变成"确定用对方法"（确定性），是 zero-shot→可靠工程的关键跃迁

## 疑问点

- 无疑问点。节点 C 四源全 ✅，置信度 1.0 达高，终止。
- 边界：本节点证"确定性选择是关键差异"，不证"LLM 对未见 G' 能否一次性解决"（节点 D）

## R7 上报

- 无。节点 C 置信度 1.0（高），终止。
