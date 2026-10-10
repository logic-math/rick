# research-L1 简报：web UI 交互体验业界基准（job_44 web UI 线）
> grilling L1｜2025-2026 agent 编排/会话产品 web UI 交互基准+可落地改进模式清单｜2026-10-05｜4 叶子（researcher）+本人 7 组交叉验证｜信源：A=官方文档/changelog/工程博客；B=核心人物复盘/49k 样本调查；C=三方

## 事实性结论（12 条）

1. **进度感知＝阶段化计划直播+成本/置信度可见**：Devin plan/playbook 呈现分步进度（含成功判据+护栏），2.1 加🟢🟡🔴完成置信度、ACU 单会话成本可见；Cursor 2.0 侧栏统一管理多 agents+plans 支持并行计划评审；Anthropic 主张 to-do/进度文件+文件系统记忆。（A：Cognition/Cursor/Anthropic 官方）
2. **中断/恢复是一等公民**：LangGraph interrupt()＝断点持久化+无限期等待外部输入后 resume；Claude Code 每条用户消息前自动 checkpoint（留 100 个），/rewind 回滚、/resume 跨会话恢复、/branch 分叉。前提：需持久化层支撑。（A：LangChain/Claude Code 官方文档）
3. **逐轮审批弹窗已被官方证伪**：Anthropic 遥测：~93% 权限提示被批准→审批疲劳失效；官方转向 auto mode（分类器决策）+沙箱+分级权限。前提：高频工具调用场景。（A：Anthropic 工程博客，本人交叉验证）
4. **敏感操作＝人工接管而非弹窗**：ChatGPT agent 高危操作需确认、敏感站点强制 watch mode、登录页暂停并 Take over browser 接管（停截屏护密码），交还后原状态续跑。（A：OpenAI 帮助中心）
5. **可纠偏（steering）是核心交互**：Claude Code Esc＝打断且保留已完成工作；Cursor ⌘+Enter 插话（即时生效）+消息排队；Devin /btw 侧聊不打断主任务。（A：Claude Code/Cursor/Devin 官方）
6. **后台并行+完成通知**：Cursor Background Agent 0.50→1.0 GA→1.5 OS 原生通知，2.0 单提示最多 8 agent 并行（git worktree 隔离），Cloud Agents 专用 VM+Slack/web/移动端管理；ChatGPT 云任务关机续跑+Scheduled 页+push/email 通知；Devin managed Devins 并行。（A：Cursor/OpenAI/Devin 官方）
7. **编排可视化＝DAG+节点状态+时间旅行**：LangGraph Studio Graph Mode 可视化 agent 图逐节点交互，每节点执行后自动 checkpoint，支持 replay/fork（改状态分叉重跑，"调试器+撤销+审计日志"）；Dify 节点状态着色+节点级 retry/错误处理+运行历史；n8n 执行数据按节点查看。（A：LangChain 文档/Dify 源码/n8n 文档）
8. **错误恢复＝error workflow+断点重试+机器验证兜底**：n8n 每工作流可配 error workflow（失败触发告警链）+从失败节点重试（已开放 API）；Dify 节点 retry-on-error 保留重试详情展示；93% 盲批准（结论 3）证明人审兜底不可靠，需测试/门禁机器验证。（A：n8n/Dify 官方）
9. **产物管理＝版本化+并排视图**：Claude Artifacts 每次编辑自动存新版本+侧栏 Artifacts 空间+与会话并排+Files 可搜索网格；ChatGPT Canvas 双栏协作+选中片段内联操作+版本箭头+Show changes；Claude Code 会话产物可发布为 live 页面。（A：Anthropic/OpenAI 官方）
10. **导航与信息密度＝项目聚合+全局搜索+归档**：ChatGPT Projects 聚合会话+文件+项目指令，Cmd/Ctrl+K 全局搜索，归档≠删除（可恢复）；Claude Projects＝工作区（历史+知识库+指令）。（A：OpenAI/Anthropic 官方）
11. **上下文用量可见+主动压缩**：Cursor 1.3 显示上下文窗口用量；Claude Code /context 可视化占用+/compact 阈值压缩；反例：ChatGPT 触顶仅提示"开新对话"无自动续接。（A/C：Cursor/Claude Code 官方+三方）
12. **反面教训**：Copilot Workspace 2025-05-30 下线（未 GA）：web 端无法 build/test 验证+重 review 负担，转向 VS Code Agent Mode；Cursor 2025-02 合并 Chat/Composer/Agent 为单一自适应界面（模式泛滥致困惑）。启示：web agent 界面须内嵌可验证性（门禁/测试结果是一等公民）。（B+/A：GitHub Next gist、dsyme.net、Cursor changelog）

## R7 上报项
1. Cursor/Anthropic 未公开 plan mode/checkpoint/后台 agent 使用率，"有效"结论依赖官方持续投入信号。
2. n8n「retry with currently saved workflow」有已知 bug（#12957：重试未用新版工作流），修复状态未确认。
3. Devin 会话回放（replay）粒度（逐帧/可编辑性）官方文档未述。
4. 「一次性大段输出不可中断」无官方弃用声明，属竞品全面流式化下的间接证伪。
5. Dify/n8n 的 UI 模式多由代码/文档推断，缺官方 UX 设计动机阐述；ChatGPT Tasks 无使用数据公开（未弃用已确认）。

## 可落地模式清单
- **有效**：流式+随时可打断（保留已完成）；diff-first 审查+任务后自动复查；plan 先行（计划可编辑）；后台/云任务+OS 通知；checkpoint+时间线回滚/fork；工具调用卡片（完成后折叠、可重试）；分级权限；项目聚合+Cmd+K+归档可恢复；产物版本化+并排预览；上下文用量可见+主动压缩；error workflow+失败节点重试；敏感操作接管；多 agent 并行（worktree）+结果自动评判。
- **弃用/证伪**：逐轮权限弹窗（93% 盲批准）；无验证闭环的大 diff review（Copilot Workspace）；模式泛滥（Cursor 三模式合并）；上下文触顶事后报错；自动全量接受为默认。
- **开发者要素（证据强度序）**：可验证准确性（SO 2025：49k 人，46% 不信任 vs 33% 信任，87% 担忧 agent 准确性）＞可打断纠偏＞可回滚＞后台并行低打扰＞用量/成本透明。

## 叶子文件
- research-L1-leaf-1.md（LangGraph/n8n/Dify）；leaf-2.md（ChatGPT/Claude）；leaf-3.md（Devin/Cursor/Copilot Workspace）；leaf-4.md（模式有效性证据清单）。含来源 URL/信源等级。
