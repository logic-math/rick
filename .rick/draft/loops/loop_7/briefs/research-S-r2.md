# research-S-r2 简报

> 阶段：S 第 2 轮（loop_7）| 主题：Q5 模板耦合盘点 + R7-4 度量方案 + R7-6 dsh benchmark | 基准：2026-08-24

## ① Q5 模板耦合盘点（internal/prompt/templates/，10 主模板+20 skills）

- **① 显式 pi 触发语法（最大耦合面）**：关键字 128 处、19/30 文件；11 文件含可执行 JS——sense_loop.md 18 处最重（L26-60 独立章节「显式触发语法（pi subagent 工具）」）；plan.md L139 八-reviewer 并行派发 JS。`agent:'xxx'` 约 75 处、timeoutMs 27 处。agent 名两类：rick 自有 think/research/exporter + **pi-subagents 内置** worker/reviewer/researcher（agents.ts:38 BUILTIN_AGENT_NAMES）——第三方扩展依赖
- **② JSONL/事件流假设：2 处，均在 ctrl.md**（L43/L152，假设 pi JSONL 事件结构）
- **③ compaction/上下文行为假设：0 处（已消解）**——sense_loop.md 的 "compact" 实为 rick 自有术语「compact contract」（L38/L61），全 templates 无 pi compaction 假设
- **④ pi 路径约定：2 处，均在 ctrl.md**（L44/L47 硬编码 `~/.rick/pi/agent/sessions/--<cwd>--/<ts>_<sessionId>.jsonl`）
- **⑤ 其他生态耦合：~11 处/4 文件**——`.pi/subagents/artifacts/` 布局+meta.json/transcript.jsonl 格式（ctrl ×6、dream ×2、doing ×1、learning ×1）；doing.md L14 `{action:"status",view:"transcript"}` 管理语法。无 PI_* 环境变量、无 --mode json/agent_settled 在模板（后者仅在 Go 代码 internal/runtime/）

**切换规则评估：「templates 不改」不成立，须修订**。耦合三层：写侧触发语言（①，loop_6 亲手内嵌）、读侧产物格式（⑤）、会话路径（④）。修订方向：templates 分 runtime 版本，或 dsh 侧等价 subagent API+兼容产物布局。**缓解性事实**：compaction 假设为 0——耦合不涉上下文管理行为，仅触发语法+产物路径。

## ② R7-4 度量方案（结论：可行，纯离线脚本，无需新埋点）

**数据源（均已存在）**：
1. `.pi/subagents/artifacts/*_meta.json`（rick 仓库 296 文件=74 runs：worker 44+reviewer 30）+ `sessions/--cwd--/subagent-artifacts/`（session-scope，sense_loop runs 在此）。字段：runId/agent/task 全文/exitCode/usage{cost,turns}/model/durationMs/toolCount
2. `~/.rick/pi/agent/run-history.jsonl`：agent/taskHash/ts/status/duration（task redact）
3. 父会话 JSONL：message 事件含 toolName=subagent/subagent_wait+时间戳——实测 loop_7 父会话（session_id 可 join）含 2+1 次调用
4. `raw_session_coding.log`（pi --mode json）：turn/tool_execution/agent_settled 事件带时间戳
5. tasks.json（status/attempts）；Trace（runtime.go:27）仅 stdout（doing.go:140）未落盘

**覆盖**：doing→worker、plan/dream→reviewer、sense_loop→think/research/exporter，均可答。

**方案（指标→采集点）**：触发率=预期派发数（briefs/*.md 存在数）÷ 实际（父会话 JSONL subagent 计数）；启动率=调用数 vs artifacts run 数；延迟=meta.json durationMs 分桶 P50/P95；成功率/成本=exitCode/usage.cost。

**限制（4）**：run-history task redacted 难归因；session_id join 键仅 loop_7 有，历史靠 meta.json task 文本匹配（含路径，可行）；「预期派发数」无机器可读定义；Trace 未持久化。

## ③ R7-6 dsh benchmark 追查（有公开数据，更新第 1 轮认知）

1. **Terminal-Bench 2.1**（tbench.ai）：DeepSeek 条目 **82.70%**（V4 Flash 0731，dsh minimal mode; max effort）[phaseo.app 镜像]。**社区复测差异大**：Discussion #1107 Harbor eval 跑全 89 task 仅 **0.51**（对照 Clawcodex 0.75，未归因）；#1089 质疑官方未附 artifacts
2. **Composio Golden Eval**（2026-08，8 harness×30 task×V4 Flash）：**dsh 未入选**（发布晚 2 天）；Pi Agent 66.7% 最高、$0.028/task 最便宜，Claude Code 最快 122.7s [composio.dev；springbrand.ai]
3. **阿里云 AgentLoop**（TB2.1 10-task 子集）：dsh vs Codex 通过率持平、失败归因各异 [segmentfault.com/a/1190000048179545]
4. **单任务对比**：dsh vs Hermes（同 v4-pro）：121s/132.6K tokens vs 780s/1.11M [atlascloud.ai]；**dsh 延迟 issue**：自托管 web TTFT ~10s→成熟会话 median 49.7s/p90 142s（#3235）；zero prompt-cache hits+6.6K token schema 开销，每轮 3.5s→53s（#3304）
5. **dsh vs pi 直接受控对比：仍无**。第三方评测基建已现：dsh-eval/harbor PR#2835/arXiv 2608.16393（安全评估，非性能）

## ④ 仍未消解项

1. TB2.1 官方 82.70% vs 社区 0.51 未归因（配置差异+官方 artifacts 完整性存疑；DNS 受限经镜像确认）
2. dsh vs pi 直接受控对比不存在（pi 是否有 TB2.1 条目未核实）
3. R7-4「预期派发数」机器可读化是落地唯一前置
4. workflowScript→dsh ctx.subagents 等价性未实测（属 R7-5 spike）
