# 调研：LangGraph Studio / n8n / Dify Web UI 交互基准（2025–2026）

## ① 长任务进度与中断恢复

**LangGraph Studio**
- 进度：Graph Mode 实时渲染执行；前端 graph execution cards 为每节点渲染一张卡片：显示状态、实时流式内容、跟踪整体完成度，用户可见当前处于哪一步。[官方=高](https://docs.langchain.com/oss/python/langgraph/frontend/graph-execution)
- 中断恢复：点 Interrupt → 选节点与暂停时机（before/after）→ 在 thread log 中点 Continue 恢复；`interrupt()` 触发时状态持久化、中断值呈现给客户端，可先编辑 state 再恢复。[官方=高](https://docs.langchain.com/langsmith/use-studio)、[官方=高](https://docs.langchain.com/oss/python/langgraph/interrupts)

**n8n**
- 进度：2025 年新增常驻画布底部 Logs 面板，执行一开始（含 AI 助手触发）即自动打开，逐步实时显示执行（覆盖循环/子工作流/AI agent），详情面板看每步输入输出。[官方=高](https://docs.n8n.io/changelog/release-notes)、[官方=高](https://github.com/n8n-io/n8n/pull/13715)
- 中断：执行状态枚举含 waiting；未见 Studio 式断点续跑。[官方=高](https://docs.n8n.io/connect/n8n-api/execution)

**Dify**
- 进度：Run History 的 Tracing 展示节点执行顺序、单节点耗时与数据流向；API 返回 elapsed_time/total_steps/total_tokens。[官方=高](https://docs.dify.ai/en/cloud/use-dify/debug/history-and-logs)
- 中断恢复：v1.13.0（2025）Human Input 节点：执行暂停 → 自定义表单送达指定人 → 填字段+点决策按钮 → 恢复（批准/改值/改路由）。[官方=高](https://github.com/langgenius/dify/releases/tag/1.13.0)、[工程博客=中](https://dify.ai/blog/the-human-input-node-bringing-human-judgment-into-automated-workflows)

三款均未见「预计剩余时间（ETA）」设计。

## ③ 多 Agent 编排可视化

**LangGraph Studio**
- Graph Mode 自动渲染节点/边；threads 持久化全部状态历史=运行历史；Time travel 可检查任意 checkpoint 的确切状态，并从该点恢复或分叉（fork）探索替代路径。[官方=高](https://docs.langchain.com/langsmith/use-studio)、[官方=高](https://docs.langchain.com/oss/python/langgraph/frontend/time-travel)
- 子图：仅当子图可被静态发现（作为节点加入或节点内直接调用）才能检查其状态；在工具函数内调用（subagents 模式）不可见/不可展开。[官方=高](https://docs.langchain.com/oss/python/langgraph/use-subgraphs)、[三方=低](https://github.com/langchain-ai/langgraph/issues/3372)

**n8n**
- 画布节点有执行状态视觉（running/success/error 等），group 标题栏聚合成员状态，优先级 waiting>running>error>issues>warning>success。[官方=高](https://github.com/n8n-io/n8n/pull/31594)
- 运行历史与重放：Executions 列表按状态过滤；可把历史执行 Copy to editor 载入画布调试并重跑。[官方=高](https://docs.n8n.io/build/understanding-workflows/understanding-executions/view-executions-for-a-single-workflow.md)

**Dify**
- 节点边框按运行状态着色（running/success/failure/exception）。[三方源码分析=低](https://deepwiki.com/langgenius/dify/10.3-workflow-canvas-and-node-ui-components)
- 迭代/循环/重试均有明细弹窗。[官方源码=高](https://github.com/langgenius/dify/blob/a9da8edbde40c6fc9ad14818d0f75b109783d51e/web/app/components/workflow/run/node.tsx)
- 重放：Stream Workflow Events 带 include_state_snapshot=true 可重放进行中运行已执行节点的状态。[官方=高](https://docs.dify.ai/en/api-reference/workflow-runs/list-workflow-logs)

## ⑤ 错误与恢复

**LangGraph（LangSmith）**
- Trace「Details」视图下钻单个 run：输入/输出/耗时/token 数/错误与元数据，嵌套 run tree 定位失败步骤；恢复=从 checkpoint time travel 分叉重跑。[官方=高](https://docs.langchain.com/langsmith/view-traces)

**n8n**
- 节点 Settings：Retry On Fail（Max Tries、Wait Between Tries）、On Error=Stop/Continue/Continue using error output、Always Output Data；可设 Error Workflow 单独接收错误数据。[官方=高](https://docs.n8n.io/flow-logic/error/)
- 失败执行重试：2025-09 起新增公开 API POST /executions/:id/retry。[官方=高](https://github.com/n8n-io/n8n/pull/19132)、[官方=高](https://docs.n8n.io/changelog/release-notes-1.x)

**Dify**
- 失败三行为：None（默认，全停并返回原始错误）/ Default Value（备份值继续运行）/ Fail Branch（画布橙色高亮分支，可走备用 API、通知、记日志）。[官方=高](https://docs.dify.ai/en/cloud/use-dify/build/predefined-error-handling-logic)
- 节点级 Retry on Failure：默认 3 次、间隔 1000ms；上限 10 次、5000ms。[官方=高](https://github.com/langgenius/dify/pull/12788)
- 错误详情：单节点测试后点 Last run 查看输入/输出/耗时/错误信息。[官方=高](https://docs.dify.ai/en/cloud/use-dify/debug/step-run)

## 结论 3 条
1. LangGraph/n8n/Dify 均无 ETA：进度=节点状态+流式日志+耗时统计。
2. 中断恢复标杆：LangGraph Studio 断点 Continue+改 state；Dify 1.13 Human Input 表单决策。
3. 断点恢复最强是 LangGraph checkpoint/time travel 分叉重跑；n8n、Dify 靠节点重试+失败分支。
