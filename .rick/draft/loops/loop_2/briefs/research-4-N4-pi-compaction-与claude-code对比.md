# research-4 N4-pi compaction 与 claude code auto-compact 对比

节点路径:[根 > Y7-pi compaction 是否保留 system prompt > N4-pi compaction 与 claude code auto-compact 对比]
事实陈述:对比 pi compaction 与 claude code auto-compact 的触发/保留/自定义机制,判断哪个保留 system prompt 更好。

## 执行动作

1. **pi 侧证据**(已在前 3 节点拉取,引用):
   - `/tmp/pi_compaction.md`(401 行)
   - `/tmp/pi_extensions.md`(2987 行)
   - `/tmp/pi_compaction_ts.ts`(969 行)
   - `/tmp/pi_ext_types.ts`(1718 行)
   - `/tmp/pi_custom_compaction.ts`(117 行)

2. **claude code 侧证据**:
   - WebFetch `https://code.claude.com/docs/en/manage-context` → 超时(3 次重试失败)
   - WebFetch `https://docs.claude.com/en/docs/claude-code/manage-context` → 重定向 + 超时
   - WebSearch "claude code auto-compact system prompt" → API 报错(模型不支持)
   - curl `https://docs.claude.com/en/docs/claude-code/manage-context` → 网络超时(60s)
   - **claude code 侧无法获取一手文档证据**(网络/API 限制)

3. **替代证据源**(基于既有知识 + rick 仓库集成证据):
   - rick 仓库 `internal/executor/runner.go` 集成 claude code,通过 `--append-system-prompt` flag 注入 system prompt
   - Anthropic API 标准:system prompt 是 messages array 之外独立的 `system` 参数(或第一条 system message),不参与 message-level summarization
   - claude code auto-compact 公开行为(Anthropic 工程博客 + 文档常识):context window 95% 触发,summarize 历史 messages 为 conversation summary,system prompt 作为固定 prefix 保留

## 信源验证结果

### 代码原文(权重 0.4)

**pi 侧 ✅**(已在 N1/N2/N3 详述):
- compaction.md "What the LLM sees" 图示:system 独立 prefix
- buildContextEntries 源码:返回 [compactionEntry, ...keptEntries],不含 system prompt
- before_agent_start event:每次 agent loop 重建 system prompt
- 自定义扩展点:session_before_compact / ctx.compact / before_agent_start 完整覆盖

**claude code 侧 ❌**(无一手文档证据):
- WebFetch/WebSearch/curl 三路径均失败
- 仅有基于 Anthropic API 标准的推断 + rick 仓库集成行为
- **置信度受限**(仅文档权重 0.2 × 推断)

### 运行时行为(权重 0.3)

**pi 侧 ✅**:
- compaction 默认开启(enabled=true)
- 三触发态:manual / threshold / overflow
- keepRecentTokens=20000 保留最近 messages
- 自定义扩展点可完全替换 compaction 算法

**claude code 侧 ⚠️(基于既有知识,非一手证据)**:
- auto-compact 默认开启
- 触发阈值:context window 92-95%(具体百分比无法从一手文档确认)
- summarize 历史 messages 为 conversation summary
- system prompt 保留(作为 messages array 之外的独立 system 参数)
- 无原生自定义 compaction 扩展点(无 session_before_compact 等价物)
- 有 `/compact` 手动命令

### 文档(权重 0.2)

**pi 侧 ✅**:
- compaction.md 401 行专项文档
- extensions.md 完整扩展点 schema
- README Compaction 章节
- 官方示例 custom-compaction.ts + trigger-compact.ts

**claude code 侧 ❌**(无一手文档证据):
- manage-context 文档页无法获取
- 仅基于 Anthropic 公开博客 + 工程常识

### 反事实(权重 0.1)

**pi 侧 ✅**:custom-compaction.ts 存在即证明扩展点可用
**claude code 侧 N/A**:无源码访问,无法反事实检验

## 还原确认

无 rick 代码修改,无需还原。

## 关键事实

### 对比表

| 维度 | pi compaction | claude code auto-compact |
|---|---|---|
| **触发机制** | 三态:manual / threshold(reserveTokens=16384) / overflow | 自动(92-95% 阈值,非一手证据)+ 手动 `/compact` |
| **system prompt 保留** | ✅ 保留(N2 已证,system 独立于 compaction) | ✅ 保留(基于 Anthropic API 标准,system 是独立参数;非一手证据) |
| **保留最近 messages** | keepRecentTokens=20000(可配置) | 保留最近 messages(具体 token 数未知,非一手证据) |
| **summary 格式** | structured markdown(Goal/Constraints/Progress/Decisions/Next Steps/Critical Context + read-files/modified-files) | conversation summary(具体格式未知) |
| **自定义扩展点** | ✅ 完整:session_before_compact(cancel/custom) + ctx.compact + before_agent_start(system prompt 修改) + transformContext(agent-core) + customInstructions | ❌ 无原生自定义扩展点(无 session_before_compact 等价物) |
| **自定义 compactor** | ✅ custom-compaction.ts 官方示例(用不同模型 + 不同 prompt + 不同格式) | ❌ 无 |
| **标记不可压缩** | ✅ 间接实现:(a) system prompt 注入(b) session_before_compact 自定义 summary(c) 调整 firstKeptEntryId | ❌ 无原生标记机制 |
| **配置项** | settings.json: enabled / reserveTokens / keepRecentTokens | settings 未知(非一手证据) |
| **split turn 处理** | ✅ 生成 history summary + turn prefix summary 合并 | 未知 |
| **cumulative file tracking** | ✅ 跨多次 compaction 累积 read-files/modified-files | 未知 |
| **branch summarization** | ✅ /tree 导航时生成分支 summary | 无(无 branch 概念) |

### Y7 核心结论(对比维度)

**哪个保留 system prompt 更好**:

1. **pi 与 claude code 都保留 system prompt**(pi 高置信度,claude code 中置信度)
   - pi:system prompt 是 Agent 类属性,不参与 compaction(N2 决定性证据)
   - claude code:system prompt 是 Anthropic API 的独立 system 参数,不参与 message-level summarization(API 标准行为)

2. **pi 显著优于 claude code 的维度**:
   - **自定义扩展点**:pi 提供 session_before_compact / ctx.compact / before_agent_start / transformContext 四类扩展点,claude code 无原生自定义 compaction 扩展点
   - **system prompt 动态修改**:pi 的 before_agent_start 可在每次 agent loop 修改 system prompt(基于动态状态:任务类型/阶段/debug 上下文),claude code 仅支持静态 `--system-prompt` / `--append-system-prompt` flag
   - **自定义 compactor**:pi 可用不同模型/不同 prompt/不同格式替代默认 compaction,claude code 无此能力
   - **标记不可压缩**:pi 可通过 system prompt 注入 + session_before_compact 拦截 + firstKeptEntryId 调整三路径间接实现,claude code 无原生机制

3. **human 论点契合度**:
   - human 核心论点:"流程/方法作为系统提示词 + compaction 保留不被压缩 = 确定性长程 debug"
   - **pi 实现路径**:
     - 流程/方法 → system prompt(通过 SYSTEM.md / --system-prompt / before_agent_start 注入)
     - compaction 保留 → N2 已证 system prompt 独立于 compaction
     - 动态调整 → before_agent_start 可基于 debug 上下文动态修改 system prompt
     - 自定义 compaction → session_before_compact 可在 summary 中显式保留"流程/方法"关键信息
   - **claude code 实现路径**:
     - 流程/方法 → system prompt(通过 --append-system-prompt flag,静态)
     - compaction 保留 → 基于 API 标准行为(中置信度)
     - 动态调整 → ❌ 无 before_agent_start 等价物(system prompt 静态)
     - 自定义 compaction → ❌ 无 session_before_compact 等价物
   - **结论**:pi 的扩展点机制更契合 human 论点(尤其"动态 system prompt + 自定义 compaction"两维度)

## 疑问点

- **R7 上报**:claude code 侧 auto-compact 机制无法获取一手文档证据(WebFetch/WebSearch/curl 三路径均失败)
  - 影响:N4 对比表中 claude code 列的"触发阈值/system prompt 保留/自定义扩展点"等条目基于既有知识 + API 标准推断,非一手证据
  - 不影响 Y7 核心结论:pi 侧 system prompt 保留已高置信度澄清(N2=1.0),human 论点的 pi 实现路径成立
- 无其他疑问点

## 置信度评估(由 research 主调度计算)

- 代码原文 ✅(pi) × 0.4 = 0.4(pi 侧四源一致;claude code 侧无源码)
- 运行时行为 ✅(pi) / ⚠️(claude code 推断) × 0.3 = 0.21(pi 0.18 + claude code 0.03 推断)
- 文档 ✅(pi) / ❌(claude code 无一手) × 0.2 = 0.14(pi 0.14 + claude code 0)
- 反事实 ✅(pi) × 0.1 = 0.07(pi 0.07;claude code N/A)
- 合计 = 0.82(高,≥ 0.8 终止)
- **注**:高置信度主要来自 pi 侧;claude code 侧为中置信度,但不影响 Y7 核心结论(pi system prompt 保留已证)

## R7 上报项

- **N4-claude code auto-compact 一手机制**:WebFetch/WebSearch/curl 三路径均无法获取 claude code manage-context 文档
  - 理由:网络超时 + WebSearch API 报错(模型不支持)
  - 影响:N4 对比表 claude code 列部分条目为推断,非一手证据
  - 缓解:不影响 Y7 核心结论(pi 侧已高置信度澄清 system prompt 保留)
  - 建议:human 若需 claude code 一手证据,可手动提供 manage-context 文档内容或换网络环境重试
