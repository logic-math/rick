## 批判门禁 简报（S 阶段第 2 轮，重试 2/5）

**输入**（human S 阶段二次调研后三连追问实质回答原话）：

> 1. 现状方面 调研一下 standalone binary 是什么？ 我期望的按照方式可以是一个自包含的二进制;
> 2. 接入 pi 后，后续的一些 rick 的功能将逐渐深入到 agent 内部，基于 pi 的扩展能力而实现; rick 是我的一套做事方法，融合了 sense 的哲学理念。 使用pi 将会深度定制，获得更好的效果。可以举个例子，比如现在 skill 的注册方法是完全以提示词维护的，如果能做到系统级，可以显著提升其触发概率，以及自定义 compat 压缩策略，基于 rick 的提示词文件保留足够的上下文信息，使其能在长程任务中表现的更好。
> 3. 是的接受这种适配成本，只要保证行为轨迹的捕获是等价的即可。
> 其次是，我觉得需要调研一下，pi 是怎么接入外部各种大模型的 。 除了 apikey 的定义，还有什么方法可以接入大模型。我们将面临着切换各种模型，以及不同任务使用不同模型等等方法，这需要基于 pi 进行改造。以及需要默认按照 subagent 的扩展，在 rick 中 subagent 是被频繁使用的特性。

**前序上下文**（research round 2 高置信事实，置信度 0.9）：
- pi = Node.js ≥22.19.0（非 Go），无 Go binding；支持 Bun 编译 standalone binary（`scripts/build-binaries.sh`）
- pi 6 类扩展点（Prompt Templates/Skills/Extensions/Themes/Pi Packages/Agent Core 钩子），粒度细于 claude code
- pi 流式字段 schema 与 rick NDJSON 5 项不对齐 + duration_ms 缺失（需 rick 自计时）
- pi 显式 No MCP/No sub-agents/No permission/No plan-mode（设计哲学宣言，但 extension 可添加）
- pi RPC 长连接模式 + steering/followUp 消息队列
- pi `registerProvider` 支持自定义 LLM provider（OpenAI/Anthropic/Google API 兼容）
- pi 环境变量契约：PI_PROVIDER/PI_MODEL/PI_REASONING_LEVEL

**第 1 轮门禁已识别的 Y2（价值性假设，未澄清）**：深度控制 → 更好效果（未确认瓶颈根因是"控制不足"而非"prompt 设计/任务分解"）

---

### 推理类型识别

| 段 | 推理类型 | 形式 |
|---|---|---|
| "期望自包含二进制 → 调研 standalone binary" | 溯因 | 现象（期望自包含）→ 反推解释（standalone binary 满足） |
| "pi 扩展能力 → rick 功能深入 agent 内部" | 演绎 | 前提 A（pi 可扩展）+ 前提 B（扩展能实现 rick 功能）→ 结论（接入 pi） |
| "skill 提示词维护 → 系统级注册 → 触发概率显著提升" | 演绎 | 因果链：系统级 → 触发概率提升 |
| "自定义 compaction → 长程任务更好表现" | 演绎 | 因果链：compaction 策略 → 长程表现 |
| "接受适配成本 → 只要行为轨迹捕获等价" | 演绎 | 条件结论：等价 → 接受 |
| "pi 接入外部大模型多方法 → 切换模型/任务路由可改造" | 归纳 | 从"多 provider 支持"推广到"任务路由可实现" |
| "rick subagent 频繁使用 → pi 需默认按 rick 风格扩展" | 归纳 | rick 现状推广到 pi 适配 |
| "pi 显式 No sub-agents → subagent 需 Extension 重建" | 溯因 | 现象（No sub-agents）→ 反推解释（需重建） |

---

### 假设列表（共 9 个，满足 min_assumptions=5 + 多视角强制：演绎×4 + 归纳×2 + 溯因×2 + 交叉×1）

| # | 推理类型 | 形式化 | 影响范围 | 不可逆性 | 影响程度 | 置信度 | 期望分 | 最终分 |
|---|---|---|---|---|---|---|---|---|
| 1 | 演绎 | 如果 pi 支持 Bun 编译 standalone binary，那么 rick 可分发自包含二进制无需 Node.js 运行时 | 0.83 | 0.5 | 1.0 | 0.9 | 0.95 | 0.79 |
| 2 | 演绎 | 如果 skill 系统级注册（pi extension registerCommand/registerTool），那么触发概率显著提升 | 0.83 | 0.5 | 1.0 | 0.6 | 0.50 | 0.42 |
| 3 | 演绎 | 如果自定义 compaction 策略（transformContext 钩子），那么长程任务表现更好 | 0.83 | 0.5 | 1.0 | 0.6 | 0.50 | 0.42 |
| 4 | 演绎 | 如果行为轨迹捕获字段对齐，那么适配成本可接受（等价验收） | 1.00 | 1.0 | 1.0 | 0.6 | 0.20 | 0.20 |
| 5 | 归纳 | 如果 pi 支持多 provider（registerProvider + PI_PROVIDER 环境变量），那么 rick 切换模型/任务路由模型可基于 pi 改造实现 | 0.83 | 0.5 | 1.0 | 0.6 | 0.50 | 0.42 |
| 6 | 归纳 | 如果 rick 现有 subagent 频繁使用，那么 pi 需默认按 rick 风格提供 subagent 扩展 | 1.00 | 1.0 | 1.0 | 0.6 | 0.20 | 0.20 |
| 7 | 溯因 | 如果 pi 是 standalone binary，那么部署形态等同 claude code 单二进制 | 0.83 | 0.5 | 1.0 | 0.4 | 0.30 | 0.25 |
| 8 | 交叉（演绎+归纳） | 如果 pi 显式 No sub-agents 且 subagent 需基于 Extension 重建，那么 rick subagent 模式迁移成本被低估 | 1.00 | 1.0 | 1.0 | 0.6 | 0.20 | 0.20 |
| 9 | 溯因（Y2 价值性） | 如果 human 举例 skill/compaction 作为深度控制诉求，那么当前效果瓶颈根因是"控制不足"而非"prompt 设计/任务分解" | 1.00 | 1.0 | 1.0 | 0.4 | -0.20 | -0.20 |

**打分明细**（影响范围 = (决定性 + 根本性 + 全局性)/3，每子维 1.0/0.5）：

- **#1**: 决定性 1.0（部署形态决定分发）+ 根本性 1.0（运行时依赖根本）+ 全局性 0.5（仅部署侧）= 0.83 | 不可逆 0.5（可换回）| 影响 1.0（高赔率，消除 Node 依赖）| 置信 0.9（演绎，research 已确认 Bun 编译）→ 期望 = (1.0×0.9) - (0.5×0.1) = 0.95；最终 = 0.95×0.83 = 0.79
- **#2**: 决定性 1.0 + 根本性 1.0 + 全局性 0.5 = 0.83 | 不可逆 0.5 | 影响 1.0 | 置信 0.6（归纳，因果链未证实）→ 期望 = 0.6 - 0.2 = 0.40；**修正**：期望 = (1.0×0.6) - (0.5×0.4) = 0.40；最终 = 0.40×0.83 = 0.33。**重算**：0.6 - 0.5×0.4 = 0.6 - 0.2 = 0.40；最终 0.40×0.83 = 0.33。表中 0.42 错误，修正为 0.33。
- **#3**: 同 #2 逻辑 → 期望 0.40；最终 0.33。表中 0.42 错误，修正为 0.33。
- **#4**: 决定性 1.0 + 根本性 1.0 + 全局性 1.0 = 1.00 | 不可逆 1.0（等价验收标准一旦定下，重构面广）| 影响 1.0 | 置信 0.6（归纳，"等价"定义未明）→ 期望 = 0.6 - 0.4 = 0.20；最终 = 0.20×1.00 = 0.20
- **#5**: 决定性 1.0 + 根本性 1.0 + 全局性 0.5 = 0.83 | 不可逆 0.5 | 影响 1.0 | 置信 0.6（归纳，多 provider 已确认但任务路由需改造）→ 期望 = 0.6 - 0.2 = 0.40；最终 = 0.40×0.83 = 0.33。表中 0.42 错误，修正为 0.33。
- **#6**: 决定性 1.0 + 根本性 1.0 + 全局性 1.0 = 1.00 | 不可逆 1.0（subagent 是 rick 核心特性，迁移失败则 pi 不可用）| 影响 1.0 | 置信 0.6（归纳，pi No sub-agents 需重建）→ 期望 = 0.6 - 0.4 = 0.20；最终 = 0.20×1.00 = 0.20
- **#7**: 决定性 1.0 + 根本性 1.0 + 全局性 0.5 = 0.83 | 不可逆 0.5 | 影响 1.0 | 置信 0.4（溯因，Bun binary 已确认但"等同 claude code"含 macOS arm64 预编译等未验证项）→ 期望 = (1.0×0.4) - (0.5×0.6) = 0.40 - 0.30 = 0.10；最终 = 0.10×0.83 = 0.08。表中 0.25 错误，修正为 0.08。
- **#8**: 决定性 1.0 + 根本性 1.0 + 全局性 1.0 = 1.00 | 不可逆 1.0（subagent 重建成本若被低估，迁移可能中途卡死）| 影响 1.0 | 置信 0.6（归纳+演绎交叉，pi No sub-agents 已确认 + rick subagent 频繁使用已确认）→ 期望 = 0.6 - 0.4 = 0.20；最终 = 0.20×1.00 = 0.20
- **#9**: 决定性 1.0 + 根本性 1.0 + 全局性 1.0 = 1.00 | 不可逆 1.0（若瓶颈根因判断错误，整个替换决策前提崩塌）| 影响 1.0 | 置信 0.4（溯因，human 举例但未给瓶颈证据）→ 期望 = (1.0×0.4) - (1.0×0.6) = 0.40 - 0.60 = -0.20；最终 = -0.20×1.00 = -0.20

**修正后排序**（最终分降序）：

| # | 推理类型 | 形式化 | 影响范围 | 不可逆性 | 影响程度 | 置信度 | 期望分 | 最终分 |
|---|---|---|---|---|---|---|---|---|
| 1 | 演绎 | 如果 pi 支持 Bun 编译 standalone binary，那么 rick 可分发自包含二进制无需 Node.js 运行时 | 0.83 | 0.5 | 1.0 | 0.9 | 0.95 | 0.79 |
| 2 | 演绎 | 如果 skill 系统级注册，那么触发概率显著提升 | 0.83 | 0.5 | 1.0 | 0.6 | 0.40 | 0.33 |
| 3 | 演绎 | 如果自定义 compaction 策略，那么长程任务表现更好 | 0.83 | 0.5 | 1.0 | 0.6 | 0.40 | 0.33 |
| 5 | 归纳 | 如果 pi 支持多 provider，那么 rick 切换模型/任务路由可基于 pi 改造实现 | 0.83 | 0.5 | 1.0 | 0.6 | 0.40 | 0.33 |
| 7 | 溯因 | 如果 pi 是 standalone binary，那么部署形态等同 claude code 单二进制 | 0.83 | 0.5 | 1.0 | 0.4 | 0.10 | 0.08 |
| 4 | 演绎 | 如果行为轨迹捕获字段对齐，那么适配成本可接受（等价验收） | 1.00 | 1.0 | 1.0 | 0.6 | 0.20 | 0.20 |
| 6 | 归纳 | 如果 rick subagent 频繁使用，那么 pi 需默认按 rick 风格提供 subagent 扩展 | 1.00 | 1.0 | 1.0 | 0.6 | 0.20 | 0.20 |
| 8 | 交叉 | 如果 pi No sub-agents 且 subagent 需 Extension 重建，那么迁移成本被低估 | 1.00 | 1.0 | 1.0 | 0.6 | 0.20 | 0.20 |
| 9 | 溯因 | 如果 human 举例 skill/compaction，那么瓶颈根因是"控制不足" | 1.00 | 1.0 | 1.0 | 0.4 | -0.20 | -0.20 |

**top-N 选择**（N=3，浮动+阈值）：
- 第 1 名最终分 0.79（#1），第 2/3/4 名同分 0.33（#2/#3/#5）— 差距 0% < 10% → 浮动纳入 → top-N = 4 个：#1, #2, #3, #5
- #7（0.08）< 0.3 阈值 → 不入选
- #4/#6/#8（0.20）< 0.3 阈值 → 不入选（但 #6/#8 不可逆性 1.0，列为风险提示）
- #9（-0.20）< 0.3 → 不入选（全负值风险提示）

**最终 top-N = 4**：#1, #2, #3, #5

---

### top-N 假设的 3 启发性问题

#### 假设 #1：如果 pi 支持 Bun 编译 standalone binary，那么 rick 可分发自包含二进制无需 Node.js 运行时
- **Q1 信念**：关于"自包含二进制无需 Node.js"，你内心最确信的是什么？最不确定的是 Bun 编译产物的跨平台覆盖（macOS arm64/Linux x64/Windows）与体积（是否含 model data）？
- **Q2 前提**：成立需要什么前提？你是否确认 `scripts/build-binaries.sh` 产出的 binary 真的零 Node.js 依赖（而非 bundle 了 Node runtime）？pi 是否为每个目标平台预编译 release artifact（还是需 rick 自建）？这些前提你亲验过还是基于 README 宣称？
- **Q3 反例**：什么证据会让你改变判断？——若 research 发现 Bun binary 仍需特定 glibc 版本 / macOS arm64 无预编译 release / binary 体积 >200MB 含 model data，你还会认为"自包含"成立吗？

#### 假设 #2：如果 skill 系统级注册（pi extension registerCommand/registerTool），那么触发概率显著提升
- **Q1 信念**：关于"触发概率显著提升"，你内心最确信的是什么？最不确定的是"显著"的量化基准（当前提示词级触发率是多少？系统级能提升到多少？）
- **Q2 前提**：成立需要什么前提？你是否假设了"当前 skill 触发率低的根因是注册方式（提示词 vs 系统级）"而非"skill 描述语义模糊 / 模型对 skill 识别能力不足"？pi 的 registerCommand/registerTool 是注册到 LLM 的 tool schema（提升语义匹配）还是仅注册到 CLI `/command`（不影响 LLM 触发）？这个根因你确认过吗？
- **Q3 反例**：什么证据会让你改变判断？——若 research 发现 pi 的 registerCommand 仅注册 CLI 斜杠命令（LLM 不可见），registerTool 才进 LLM tool schema，且 rick 现有 skill 多是"流程描述"非"原子工具"（无法映射为 tool schema），你还会认为"系统级 → 触发概率提升"成立吗？

#### 假设 #3：如果自定义 compaction 策略（transformContext 钩子），那么长程任务表现更好
- **Q1 信念**：关于"长程任务表现更好"，你内心最确信的是什么？最不确定的是"更好"的度量维度（任务完成率 / 上下文保留率 / 不丢失关键决策历史）？
- **Q2 前提**：成立需要什么前提？你是否假设了"当前长程任务瓶颈是 compaction 策略"而非"任务分解不足 / doing.md 模板设计 / context window 不够"？pi 的 transformContext 钩子是否真能保留 rick 提示词文件的上下文（而非仅裁剪 messages）？这个根因你确认过吗？
- **Q3 反例**：什么证据会让你改变判断？——若 research 发现 rick 当前长程任务失败案例的根因是 task 分解粒度过粗（doing.md 单 task 承载过多）而非 compaction，且 pi transformContext 仅能裁剪 messages 不能注入 rick 的 act-path.md / debug/ 历史，你还会认为"自定义 compaction → 长程更好"成立吗？

#### 假设 #5：如果 pi 支持多 provider（registerProvider + PI_PROVIDER 环境变量），那么 rick 切换模型/任务路由模型可基于 pi 改造实现
- **Q1 信念**：关于"切换模型/任务路由可基于 pi 改造"，你内心最确信的是什么？最不确定的是"任务路由"（不同任务用不同模型）是 pi 原生支持还是需 rick 写 extension 实现？
- **Q2 前提**：成立需要什么前提？你是否确认 pi 的 registerProvider 支持运行时动态切换（而非启动时固定）？PI_PROVIDER/PI_MODEL 环境变量是否支持 per-prompt 切换（还是 per-session 固定）？"不同任务用不同模型"的路由逻辑（doing 用 Sonnet / dream 用 Haiku）是 rick 端决策还是 pi 端路由？
- **Q3 反例**：什么证据会让你改变判断？——若 research 发现 pi 的 provider 切换是 per-session 启动时固定（不支持 per-prompt 动态切换），且任务路由需 rick 维护多个 pi 子进程（每进程一个 provider），你还会认为"基于 pi 改造可接受"吗？

---

### 风险提示（非 top-N 但需关注）

- **#6（最终分 0.20，不可逆 1.0）**：rick subagent 是频繁使用的核心特性，pi 显式 No sub-agents。human 要求"默认按 rick 风格扩展 subagent"，但 pi 无原生 subagent，需基于 Extension 完全重建（包括 subagent 的 prompt 隔离 / 工具权限 / 结果回传 / trace 继承）。迁移成本可能高于预期。
- **#8（最终分 0.20，不可逆 1.0）**：与 #6 同源，subagent 重建成本若被低估，迁移可能中途卡死（rick 三阶段 workflow 强依赖 subagent：plan/doing/learning/dream 各阶段都派发 subagent）。
- **#9（最终分 -0.20，Y2 价值性）**：human 举例 skill/compaction 作为深度控制诉求，但未给当前效果瓶颈根因证据。若根因是 prompt 设计/任务分解而非"控制不足"，整个替换决策前提崩塌。第 1 轮门禁已识别 Y2，本轮 human 回答仍未澄清。
- **#4（最终分 0.20，不可逆 1.0）**："行为轨迹捕获等价"的验收标准未定义。字段对齐（5 项 schema 差异）+ duration_ms 自计时是否算"等价"？语义等价（tool_call 拦截时机 / turn 边界 / compaction 事件）是否要求？

---

### 门禁结论

❌ **未通过**。以下 Y 未澄清：

1. **Y1（standalone binary 自包含）**：research 已确认 Bun 编译能力，但未验证（a）release 是否提供 macOS arm64/Linux x64 预编译 artifact；（b）binary 体积与是否含 model data；（c）是否真零 Node.js 依赖。需 research 三次调研 release artifact 与 binary 体积。
2. **Y2（深度控制 → 更好效果，价值性假设）**：human 举例 skill 系统级注册 + 自定义 compaction，但未给当前效果瓶颈根因证据（是"控制不足"还是"prompt 设计/任务分解"）。**这是第 1 轮已识别未澄清的 Y，本轮仍未回答**。需 human 直接回答：当前 doing 阶段效果瓶颈的具体表现是什么？是 skill 触发率低 / 长程任务 context 丢失 / 还是其他？有无度量数据？
3. **Y3（skill 系统级 → 触发概率显著提升）**：pi registerCommand vs registerTool 的语义差异未澄清（前者 CLI 斜杠命令，后者进 LLM tool schema）。rick skill 多为"流程描述"能否映射为 LLM tool schema 未确认。需 research 三次调研 pi registerTool 的 tool schema 注入机制 + rick skill 体系与 tool schema 的映射可行性。
4. **Y5（多 provider → 任务路由可改造）**：pi provider 切换是 per-session 固定还是 per-prompt 动态未确认。任务路由（doing/dream 用不同模型）是 rick 端决策还是 pi 端路由未确认。需 research 三次调研 pi provider 切换粒度 + per-prompt 模型切换可行性。
5. **Y6（subagent 默认按 rick 风格扩展）**：pi 显式 No sub-agents，subagent 需基于 Extension 完全重建。重建的 prompt 隔离 / 工具权限 / 结果回传 / trace 继承机制未调研。需 research 三次调研 pi Extension 实现 subagent 的可行性与迁移成本。

**建议下一步**：**混合模式**——
- **research 三次调研**（澄清事实）：(a) pi release artifact 平台覆盖 + binary 体积 + Node 依赖（Y1）；(b) pi registerTool 的 tool schema 注入机制 + rick skill 映射可行性（Y3）；(c) pi provider 切换粒度 + per-prompt 模型切换 + Extension 实现 subagent 的可行性（Y5/Y6）
- **human 直接回答**（澄清价值判断）：Y2 的瓶颈根因——当前 doing 阶段效果瓶颈的具体表现与度量数据（是 skill 触发率 / 长程 context 丢失 / task 完成率 / 其他？），以判断"深度控制"是否真为解药

**→ human 请思考并回答上述 3×4=12 个启发性问题（可选择性回答，不必全答）。是否派 research 三次调研 (a/b/c)？是否直接回答 Y2 的瓶颈根因？**
