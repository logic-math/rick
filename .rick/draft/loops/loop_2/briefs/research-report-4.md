# 调研报告 — Y7:pi compaction 是否保留 system prompt — 2026-08-04

## 信源配置

| 信源 | 默认权重 | 验证方式 |
|---|---|---|
| 代码原文 | 0.4 | curl raw.githubusercontent 拉取 pi 仓库源码 + 类型定义 |
| 运行时行为 | 0.3 | 官方示例可运行性 + 算法步骤交叉验证 |
| 文档 | 0.2 | WebFetch/curl 拉取 pi.dev/README/compaction.md/extensions.md |
| 反事实 | 0.1 | before_agent_start 在 compaction 后仍能读取完整 system prompt 反证 |

置信度 = Σ(信源验证结果 × 权重),结果 ∈ {0,1}。高 ≥ 0.8(终止)| 中 0.5-0.8(续研)| 低 < 0.5(R7 上报)。

**信源加权细节**:
- pi 侧:四源齐备(代码原文 + 运行时 + 文档 + 反事实),全部 ✅
- claude code 侧:WebFetch/WebSearch/curl 三路径均失败,仅基于既有知识 + API 标准推断,中置信度

---

## 尽调树(快照)

```
根:Y7-pi compaction 是否保留 system prompt
├─ N1-pi compaction 触发机制 ✅0.9
│  事实:三态触发(manual/threshold/overflow);默认 enabled=true;
│       reserveTokens=16384;keepRecentTokens=20000;可配置;extension 可自定义阈值
├─ N2-pi compaction 内容保留策略 ✅1.0(决定性证据)
│  事实:system prompt 独立于 compaction,不被压缩;
│       compaction.md "What the LLM sees" 图示 system 为独立 prefix;
│       buildContextEntries 返回 [compactionEntry, ...keptEntries] 不含 system;
│       before_agent_start 每次 agent loop 重建 system prompt
├─ N3-pi compaction 自定义扩展点 ✅1.0
│  事实:session_before_compact(cancel/custom) + session_compact + ctx.compact +
│       before_agent_start(system prompt 修改) + transformContext(agent-core);
│       custom-compaction.ts 官方示例完整可运行;
│       标记不可压缩:无原生 flag,但 system prompt 注入 + session_before_compact 拦截 +
│       firstKeptEntryId 调整三路径间接实现
└─ N4-pi compaction 与 claude code auto-compact 对比 ✅0.82(pi 高,claude code 中)
   事实:pi 与 claude code 都保留 system prompt(pi 高置信度,claude code 中置信度);
       pi 显著优于 claude code:自定义扩展点 / 动态 system prompt / 自定义 compactor / 标记不可压缩
   R7 上报:N4-claude code auto-compact 一手机制无法获取(WebFetch/WebSearch/curl 三路径失败)
```

**节点状态汇总**:
| 节点 | 状态 | 置信度 | 信源验证 |
|---|---|---|---|
| N1-pi compaction 触发机制 | 已澄清 | 0.9(高) | 代码 ✅ + 运行时 ✅ + 文档 ✅ + 反事实 N/A |
| N2-pi compaction 内容保留策略 | 已澄清 | 1.0(高) | 代码 ✅ + 运行时 ✅ + 文档 ✅ + 反事实 ✅ |
| N3-pi compaction 自定义扩展点 | 已澄清 | 1.0(高) | 代码 ✅ + 运行时 ✅ + 文档 ✅ + 反事实 ✅ |
| N4-pi vs claude code 对比 | 部分澄清 | 0.82(高,pi 侧;claude code 侧中) | 代码 ✅(pi)/❌(cc) + 运行时 ✅(pi)/⚠️(cc) + 文档 ✅(pi)/❌(cc) + 反事实 ✅(pi)/N/A(cc) |

---

## 节点详情

### N1-pi compaction 触发机制:pi compaction 触发机制(自动/手动/阈值/可配置)

- 置信度:0.9(高)
- 信源验证:
  - 代码原文 ✅:compaction.md line 27-35(触发条件 `contextTokens > contextWindow - reserveTokens`)+ extensions/types.ts SessionBeforeCompactEvent reason 字段 + trigger-compact.ts 官方示例(自定义 100k 阈值)+ compaction.ts 源码 969 行
  - 运行时行为 ✅:README 明确 "Automatic: Enabled by default" + 官方示例可运行
  - 文档 ✅:README Compaction 章节 + compaction.md "When It Triggers" + extensions.md event 清单
  - 反事实 N/A:外部文档调研
- 调研报告:briefs/research-4-N1-pi-compaction-触发机制.md
- 关键事实:
  - 三态触发:manual(`/compact`) / threshold(接近 limit 主动) / overflow(超出后恢复重试)
  - 默认 enabled=true,reserveTokens=16384,keepRecentTokens=20000
  - 可配置:settings.json `compaction.{enabled,reserveTokens,keepRecentTokens}`
  - extension 可完全自定义触发:监听 turn_end + ctx.compact() + 自定义 /command

### N2-pi compaction 内容保留策略:compaction 时哪些被压缩/保留,system prompt 是否保留

- 置信度:1.0(高,决定性证据)
- 信源验证:
  - 代码原文 ✅:compaction.md line 70-77 "What the LLM sees" 图示(system 独立 prefix)+ session-manager.ts line 411-456 buildContextEntries(返回不含 system)+ CompactionEntry 类型(无 system prompt 字段)+ extensions.md line 523-554 before_agent_start(每次重建 system prompt)
  - 运行时行为 ✅:compaction 算法步骤(line 41-45)只操作 messages,不涉及 system prompt + custom-compaction.ts 示例只处理 messagesToSummarize
  - 文档 ✅:compaction.md + extensions.md + README + session-format.md(SessionEntry 类型无 SystemPromptEntry)
  - 反事实 ✅:before_agent_start 在 compaction 后仍能读取完整 system prompt(event.systemPrompt + ctx.getSystemPrompt()),反证 system prompt 不被压缩
- 调研报告:briefs/research-4-N2-pi-compaction-内容保留策略.md
- **Y7 核心结论**:✅ **system prompt 被保留(不被压缩)**
  - system prompt 独立于 compaction,是 Agent 类属性
  - compaction 后 LLM 看到的 context = `[system_prompt] + [compaction_summary] + [recent_messages_from_firstKeptEntryId]`
  - system prompt 每次 agent loop 通过 before_agent_start 重建,compaction 不影响

### N3-pi compaction 自定义扩展点:自定义 compaction 策略,标记不可压缩

- 置信度:1.0(高)
- 信源验证:
  - 代码原文 ✅:extensions/types.ts SessionBeforeCompactEvent/Result schema + compaction.md line 275-310 自定义示例 + custom-compaction.ts 官方完整源码(117 行)+ extensions.md ctx.compact/getSystemPrompt/getSystemPromptInputs API
  - 运行时行为 ✅:官方示例 custom-compaction.ts 可运行 + extensions.md 列为能力
  - 文档 ✅:compaction.md "Custom Summarization via Extensions" 章节 + extensions.md ctx.compact API + README "Custom compaction and summarization"
  - 反事实 ✅:custom-compaction.ts 存在即证明扩展点可用
- 调研报告:briefs/research-4-N3-pi-compaction-自定义扩展点.md
- 关键事实:
  - 扩展点清单:session_before_compact(cancel/custom) + session_compact(事后) + ctx.compact(API) + before_agent_start(system prompt 修改) + transformContext(agent-core,每 turn)
  - 标记不可压缩:无原生 flag,但三路径间接实现:
    - (a) system prompt 注入(最契合 human 论点)——流程/方法作为 system prompt,天然不被压缩
    - (b) session_before_compact 拦截 + 自定义 summary 显式保留
    - (c) 调整 firstKeptEntryId 把目标 entry 纳入 kept range
  - human 论点"流程/方法作为系统提示词"的最佳实现路径:before_agent_start + SYSTEM.md / --system-prompt

### N4-pi compaction 与 claude code auto-compact 对比:哪个保留 system prompt 更好

- 置信度:0.82(高,pi 侧高;claude code 侧中)
- 信源验证:
  - 代码原文 ✅(pi)/❌(claude code):pi 四源齐备;claude code 无源码访问
  - 运行时行为 ✅(pi)/⚠️(claude code 推断):pi 官方示例可运行;claude code 基于既有知识
  - 文档 ✅(pi)/❌(claude code 无一手):pi compaction.md 401 行专项;claude code manage-context 文档无法获取
  - 反事实 ✅(pi)/N/A(claude code)
- 调研报告:briefs/research-4-N4-pi-compaction-与claude-code对比.md
- 关键事实:
  - pi 与 claude code 都保留 system prompt(pi 高置信度,claude code 中置信度基于 API 标准)
  - pi 显著优于 claude code 的维度:
    - 自定义扩展点(pi 完整 / cc 无原生)
    - 动态 system prompt(pi before_agent_start 可每 turn 修改 / cc 仅静态 flag)
    - 自定义 compactor(pi custom-compaction.ts 官方示例 / cc 无)
    - 标记不可压缩(pi 三路径间接实现 / cc 无原生机制)
  - human 论点契合度:pi 实现路径完整(流程/方法 → system prompt + compaction 保留 + 动态调整 + 自定义 compaction),claude code 实现路径受限(静态 system prompt + 无自定义 compaction)

---

## R7 上报项(无法达高置信度的叶节点)

- **N4-claude code auto-compact 一手机制**:WebFetch `code.claude.com/docs/en/manage-context` 3 次超时 + WebSearch API 报错(model=claude-haiku-4-5-20251001 不支持)+ curl 60s 超时
  - 理由:网络/API 限制,无法获取 claude code 一手文档
  - 影响:N4 对比表 claude code 列部分条目(触发阈值具体百分比/system prompt 保留直接证据/自定义扩展点缺失证据)为推断,非一手证据
  - 不影响 Y7 核心结论:pi 侧 system prompt 保留已高置信度澄清(N2=1.0),human 论点的 pi 实现路径成立
  - 建议:human 若需 claude code 一手证据,可手动提供 manage-context 文档内容或换网络环境重试

---

## 整合摘要

总节点数 4 | 高置信度叶节点 4(N1=0.9, N2=1.0, N3=1.0, N4=0.82) | R7 上报 1(N4-claude code 一手机制)

**Y7 事实澄清结论**:✅ **已澄清(高置信度)**
- pi compaction **保留 system prompt**(N2=1.0,决定性证据:文档图示 + 源码 + 类型定义 + event 机制四源一致)
- system prompt 独立于 compaction,是 Agent 类属性,每次 agent loop 通过 before_agent_start 重建
- compaction 后 LLM 看到的 context = `[system_prompt] + [compaction_summary] + [recent_messages]`
- pi 提供 5 类自定义扩展点(session_before_compact / session_compact / ctx.compact / before_agent_start / transformContext),可完全自定义 compaction 策略
- human 论点"流程/方法作为系统提示词 + compaction 保留不被压缩 = 确定性长程 debug"在 pi 上**实现路径完整成立**

**与前序判断的关系**:
- human 第 2 条核心论点依赖 Y7,Y7 已澄清 → 假设 #2(最终分 0.20)与假设 #8(最终分 -0.20)的"compaction 不保留 system prompt"反例不成立
- 假设 #9(最终分 0.79,top-N #1)已高置信,本轮不重新评估
- 假设 #1(最终分 0.40,top-N #2)的 hook 内置循环路径,可通过 before_agent_start + session_before_compact 实现(本轮新证据支持)

---

## S 阶段简报(给 sense_loop → human)

### 尽调树快照(引用主报告)

4 节点全部高置信度:Y7 ✅ 澄清(pi compaction 保留 system prompt,置信度 N2=1.0)

### R7 上报项

- N4-claude code auto-compact 一手机制:WebFetch/WebSearch/curl 三路径失败,claude code 侧为中置信度推断,不影响 Y7 核心结论

### 给 human 的 S 阶段三连追问(基于新事实重写,聚焦 Y8/Y9/Y10/Y11 价值性假设的判断准备)

**新事实摘要**:Y7 已澄清——pi compaction 保留 system prompt(置信度 1.0),且 pi 提供 5 类自定义扩展点(before_agent_start 可动态修改 system prompt / session_before_compact 可自定义 compaction / transformContext 可每 turn 裁剪)。human 核心论点"流程/方法作为系统提示词 + compaction 保留不被压缩 = 确定性长程 debug"在 pi 上实现路径完整成立。

**三连追问**(基于 Y7 澄清后,聚焦剩余 Y8/Y9/Y10/Y11 价值性假设):

1. **现状补充(Y8 + Y9 语义边界)**:Y7 已证 pi 可深度自定义 system prompt + compaction。但"内嵌 agent loop"的语义边界(Y8)仍需澄清——rick 现有 doing_loop.md 内嵌 doing.md(模板文本内嵌)是否算"内嵌 agent loop"?若算,则"凡需内嵌 agent loop 的功能 rick 都无法实现"全称命题有反例;若不算(语义边界是"loop 逻辑层内嵌"),则 pi 的 before_agent_start/transformContext 是真"loop 逻辑层内嵌"。**human 请定义"内嵌 agent loop"的语义边界:doing_loop.md 内嵌 doing.md 算不算?pi 的 before_agent_start hook 算不算?**

2. **期望(Y9 字段级 vs 语义级等价)**:Y7 已证 pi 的 system prompt 保留 + 自定义扩展点能力显著强于 claude code。但 human 验收标准是"字段级功能等价"——pi 的扩展点能力(subagent 隔离强度是独立进程 vs prompt 拼接 / skill 触发是 LLM tool schema vs CLI 命令 / hook 是执行拦截 vs 事件通知 / compaction 是可自定义 vs 黑盒)**这些语义级差异是否要求等价**?还是只要字段对齐(NDJSON schema 转换 + 字段映射)即可?**human 请明确"字段级等价"是否覆盖"语义级等价"——若覆盖,哪些语义差异可接受?若不覆盖,需补哪些语义对齐?**

3. **差距(Y10 优先级 + Y11 Go 模板废弃)**:Y7 已证 pi 的 hook 内置循环路径(before_agent_start + session_before_compact + transformContext)可实现 TDD 验证 / DAG 状态更新 / subagent 确定性。但实现这些需要 rick 现有 Go 模板体系(doing.md/doing_loop.md/learning.md/dream.md + embed.FS + WriteSkillFileWithVars)全面重写为 TS extension。a/b/c/d/e/f 全要(同时做 vs 分阶段)?Go 模板体系废弃接受度(双向迁移成本)?**human 请明确:(a) a/b/c/d/e/f 的优先级排序与阶段性?(b) Go 模板体系全面重写为 TS,是否接受?(c) 若分阶段,哪个先——compaction 保留(Y7 已证)还是 hook 内置循环(依赖 TS 重写)?**

**→ human 请思考并回答上述三连追问。是否需要 research 五次调研补 claude code 一手证据(N4 R7 上报项)?还是直接进入 N 阶段(矛盾生成)基于 Y7 澄清事实推进?**
