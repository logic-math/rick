# 写域
web/src/**（前端优化）
仅在证据指向 server 时最小改动 internal/web/**（须在回执中单独说明）

# 问题
用户实测：「底层 pi 返回消息的速度那么慢？使用 CLI 的时候为什么没有？」
——web 端文字出现/流式推进比 CLI 明显慢/卡。

# 已采集证据（勿重复，直接作为起点）
真实会话（8412，plan bootstrap，74 秒窗口）SSE 事件统计：
- message_update 1969 条（thinking_delta 1498 / toolcall_delta 322 / text_delta 114）
- **相邻 delta 间隔：中位 8ms、p90 75ms、最大 11s**（模型等待）
- agent 期间 session busy=true 持续 40s+
→ **pi 输出不慢**（8ms/帧 ≈ 125 事件/秒的密集流）。
CLI 侧无前端管线（终端增量渲染）；web 侧管线 = Go server 转发（已确认低开销）→ SSE →
events store（rAF 批处理 + 500 条环形缓冲）→ buildChatViewModel **每帧全量重放 envelopes**
→ React 全量 diff → DOM/滚动。
**待验证假设**：瓶颈在前端处理管线（每帧全量 replay + React 更新 + 长历史影响），
导致显示滞后/积压（用户感知「慢」）。

# 任务目标
用测量数据定位 web 端显示延迟的真实瓶颈（SSE 到达 → DOM 更新的各段耗时），做针对性
优化，使流式推进延迟接近 CLI 体验。

# 必读
- /workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_36/doing/prompts/skill_debug_skill.md（数据驱动，勿猜）
- debug/bug1/2/3（流式稳定性历史修复，勿回归：落定 id 稳定、打开块裁剪保护、thinking 折叠、贴底 rAF）

# 关键结果
1. **分段测量**（给出数字，p50/p95）：
   - SSE 事件到达（可达性能条目记录的到达时间）→ events store flush → vm 重建 → React commit → DOM 文本出现（MutationObserver 时间戳）的总延迟
   - buildChatViewModel 单帧耗时 vs envelopes 长度（500 条 live / 长 history 场景）
   - 事件积压：flush 批次大小分布、单帧处理的事件数、是否有队列增长
   - 对照：mock 高频流（如 8ms/帧 × 2000 帧）单独测前端纯成本（与真实会话分离）
2. **瓶颈定位**：明确指出哪一段占比最大（解析/flush/vm 重放/React diff/布局/滚动），附证据
3. **优化**（按影响排序，最小改动优先）——候选方向（择证据支持者）：
   a. **SSE 批次合并**：events store 的 rAF flush 已是批处理；若 vm 全量重放是瓶颈 →
      改为**增量重放**（复用上一帧 BuildState / 只重放 open block 段；history.items 已缓存）
   b. **服务端 delta 合并**（若前端解析/flush 本身是瓶颈）：同 open block 的连续
      message_update delta 合并为一帧（如 50ms 窗口）后转发——事件数降 10-50×；
      注意保持契约兼容（前端只做文本拼接）与 message_end 落定语义不变
   c. React 层：live 块渲染瘦身（避免整棵消息树 diff；已 memo 落定项）
   d. 滚动/布局：贴底 rAF 已有；检查是否有同步布局抖动（layout thrash）
4. **验证**：
   - 优化前后延迟数字对比（同场景、同事件量）
   - 不回归：消息顺序/折叠 override/落定 id 稳定/打开块裁剪保护/thinking 默认折叠/贴底跟随
   - tsc + build + go test（若动 server）+ playwright 桌面与 iPhone

# 环境
- 8412 = 用户真实实例（可用真实会话观测；**测试用完 close 会话**，勿破坏用户数据）
- 6173 = 隔离 verify（`/tmp/start-verify.sh`，HOME=/tmp/rick-e2e-home，token 取真实 HOME config 的 web_token）
- playwright 脚本在 /tmp/ui-verify/（repro10/11/12/13 可参考）；浏览器性能采样可用
  PerformanceObserver / performance.mark + MutationObserver
- 长会话 fixture：aed8d899-766d-4139-8840-0ea5432a495b（254 条历史）

# 纪律
不碰 git；回执：分段测量数据 + 瓶颈结论 + 优化 diff + 前后对比 + 回归结果
