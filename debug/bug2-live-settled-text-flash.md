---
summary: "流式（工具/思考）期间「上一条 AI 已落定消息」闪烁/消失——根因：① vm 落定条目 id 用 replay 局部计数（st.seq/items.length），buffers 环形裁剪（>500）平移 replay 起点 → 已渲染条目 key 变 → 重挂+enter 动画重放=闪；② 裁剪把结构事件（message_end 等落定）与增量帧一视同仁丢弃 → 已落定文字挤出 500 窗口 → 从 UI 消失（无 resync 不回来）=「闪没」。已修：① 落定条目 id 改用服务端全局 envelope.seq（裁剪无关）；② 环形裁剪保护结构事件、只丢最旧增量帧（message_update/tool_execution_update——落定由 message_end/tool_execution_end 承载）。"
status: "✅ 已解决"
---

# bug2-live-settled-text-flash.md

## Phase 1: 构建反馈回路

**用户报告**（真实会话 aed8d899，455 条历史）：resume 会话后流式进行中（工具调用/思考状态输出时），**上一条 AI 已返回的消息（assistant-text 落定块）持续一闪一闪/反复刷新**。非消息顺序问题（已修乐观 user 位置）。

**复现环境**：6173 verify 实例注入真实会话 aed8d899 注册（读用户 HOME 的 pi session JSONL，455 条真实 entries）。Playwright headless + 修正版 MockEventSource（ev.data 必须是 JSON 本身）。

**DOM 检测**（动画事件 + 节点 identity + DOM mutations，弃用像素差分）：
- 首屏历史挂载动画 rm-chat-in×219 + rm-portal-spin×120 集中 243ms（一次性挂载，非闪烁——**判定闪烁必须排除首屏窗口**，在基线完成后注入）。
- 挂载完成后注入单回合流式 → 新内容动画仅 5 次（新条目挂载），历史消息零重挂。
- 决定性复现：注入 780 条工具流（13×60 @100ms）逼 buffers 环形裁剪 → **已落定文字消息从 UI 消失**（gone=1，无 resync 不回来）。

## Phase 2: 复现最小化

**纯函数级最小复现（秒级）**：`esbuild viewModel.ts → node`，模拟 buffers 环形裁剪（slice 掉头部 N 条后重新 buildChatViewModel）：
- 裁剪头 6 条：同语义落定条目 id `text-8-1` → `text-2-0`（**key 变 = 重挂**）
- 基线对比同语义条目 id 一致性 = **false**

**store 级最小复现**：逐条 append + trimBuffer，观察 message_end（文字落定事件）是否随 780 条增量帧被挤出——旧 slice 逻辑：message_end 在 500 窗口外被丢 → vm replay 不再生成文字。

## Phase 3: 可证伪假设

### 假设 A（高优先级，✅ 确认）
- **观察证据**：vm 落定条目 id = `text-${st.seq}-${st.items.length}`（st.seq=replay 局部事件计数、items.length=构建时数组长）。
- **推断**：buffers 环形裁剪（500 上限）→ replay 起点平移 → 全部已渲染落定条目 key 变 → 卸载重挂 + rm-chat-in 重放 = 闪。
- **验证**：esbuild 单测，裁剪 6 条 → `text-8-1`→`text-2-0`（纯函数确认）；DOM 注入 780 条工具流观察。
- **结果**：✅ 确认。

### 假设 B（中，`❌ 未复现`——resync 跨源切换）
- **推断**：resync 重建 history 后，同内容从 vm（text-{seq}）切到 history（h-text-{entry.id}）→ key 不同 → 切换重挂。resync 周期 = 闪烁周期。
- **验证**：注入文字 → mock entries 第 2 次 fetch 含已落盘文字 → 触发 resync（onerror→指数退避重连→server_info→dispatch）→ 观察 ZZ 节点替换。
- **结果**：ZZ 节点 replaced=0（未复现）——fingerprint skip + React 批处理使切换在单 commit 完成，未观测到独立重挂。**存疑但非主因**（真实环境 resync 频率若高仍可能逐次轻微重挂，修复 A 后 id 统一为 envSeq 降低了切换时的 key 差）。

### 假设 C（✅ 确认——用户可见主因）
- **观察证据**：buffers 500 窗口在长工具回合（780 条 update 帧）下必然溢出 → slice 裁掉头部 → **文字落定的 message_end 结构事件被一并裁掉** → vm 不再生成文字 → **已显示的消息从 UI 消失**（无 resync 前不回来）。
- **推断**：用户看到「上一条 AI 消息一闪一闪」= 消息落定显示后，随后思考/工具流把它的结构事件挤出窗口 → 消失；若期间有 resync/history 重建 → 以 h-text 重新出现 → 反复 = 「总在刷新」。
- **验证**：DOM（repro8）文字 seen=1 后 gone=1、最终从正文消失。
- **结果**：✅ 确认。

## Phase 4: 插桩观察

- events store MAX_PER_SESSION=500 环形裁剪 `buf.slice(buf.length - 500)`——**不区分事件类型**，message_end（落定里程碑）与 message_update（增量帧）同样被裁。
- 增量帧语义：`message_update`（text/thinking delta）与 `tool_execution_update`（partialResult）是**纯增量**——完整事实分别由 `message_end` 与 `tool_execution_end` 承载。裁增量帧只损失「正在打的字中间帧」（不闪消息）；裁结构事件 = 已渲染内容消失（闪没）。
- envelope.seq（SSEEnvelope 顶层）= 服务端 Hub.Publish 全局单调——**裁剪/重放无关**的稳定身份源（vs st.seq replay 局部计数）。
- go:embed：web/dist 改动必须 `go build` 重新 embed 才生效（verify 实例曾 serve 旧 dist 导致修复验证假失败——**教训**）。

## Phase 5: 修复回归

**修复（两处，最小改动）**：
1. `viewModel.ts`：落定条目 id 改用 **envelope.seq**（服务端全局单调）：
   - `applyEvent(st, ev)` → `applyEvent(st, ev, env.seq)`；BuildState 增 `idSeq`。
   - message_end user `user-{envSeq}`；assistant text `text-{envSeq}`（同消息多 text block：`text-{envSeq}-{blockIdx}`）；thinking `think-{envSeq}`；notice `retry-/exterr-/abort-/err-{envSeq}`。
   - message_update liveText/liveThinking id `live-{envSeq}`。
   - **效果**：buffers 裁剪/重放起点平移不再改变已渲染条目 key → 无重挂。纯函数回归：裁剪 6 条 → id 一致 ✓（旧版 text-8-1→text-2-0）。
2. `stores/events.ts`：环形裁剪保护结构事件——`isDeltaFrame()`（message_update/tool_execution_update）+ `trimBuffer()`（优先丢最旧增量帧；仅当增量帧不足才退化为丢最旧结构帧）。**效果**：已落定文字/工具卡消息永不随裁剪消失。
   - 纯函数回归（逐条 append 模拟 flush）：665→500 条裁剪后 message_end/agent_start/tool start 全存活、尾部最新保留、顺序保持 ✓。

**DOM 回归（桌面 + iPhone 13 模拟）**：真实 455 条历史 + 注入文字落定 + 780 条工具流（必触发裁剪）：
- 文字节点 出现/替换/消失 = **1 / 0 / 0**（零替换零消失）✓
- 文字在正文 **true**（修复前 false——消失）✓
- 裁剪期 animationstart = **0**（5 个动画全部为初始挂载瞬间，其后 780 帧流式零重挂）✓
- 无 JS 错误 ✓

**全量回归**：`npx tsc --noEmit` ✓、`npm run build` ✓、`go build` ✓、`go test ./internal/...` ✓。

## Phase 6: 清理事后分析

- 诊断脚本留存 /tmp/ui-verify/（repro*.mjs，不入库）。
- 6173 verify registry 注入的 aed8d899 真实会话记录——**已恢复**（移除注入，保持验证环境隔离）。
- **防范模式（沉淀 bugs.md）**：
  1. **React list key 必须裁剪无关**：live 落定条目 id 不得用 replay 局部计数（st.seq/items.length）；用 envelope.seq（服务端全局）或稳定数据身份。
  2. **环形缓冲裁剪必须按事件语义分层**：纯增量帧（delta/update——完整事实由其 end 事件承载）可裁；结构事件（message_end/tool_execution_start+end/agent_*/回合边界）是已渲染内容的承载，裁掉 = 已显示消息消失 = 「闪没」。
  3. **mock 验证环境必须确认产物链**：rick web 静态资源 go:embed——改 web/dist 后必须 go build + 重启，否则验证的是旧产物（假失败/假通过）。
  4. 闪烁判定须排除首屏挂载窗口（animationstart 按注入基线分段统计）。

## 结论

「工具/思考流式时上一条 AI 消息闪烁」根因二元：
① 落定条目 key 不稳定（replay 局部计数）——裁剪后重挂闪（A）；
② 环形裁剪无差别丢结构事件——已落定文字被挤出 500 窗口而从 UI 消失，resync 后又回来 = 反复「闪没」（C，用户可见主因）。
修复 = id 稳定化（envelope.seq）+ 裁剪保护结构事件（只丢增量帧）。双端 DOM 回归：780 条工具流裁剪期间已落定文字 0 替换 0 消失 0 重挂。
