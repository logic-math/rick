# 调研简报: Agent loop 定制 × 稳定内核/插件故障隔离(loop_7 / extra-leaf-1)

## 子问题1: Agent 框架 loop 定制

1. **LangGraph**: 接口=显式 StateGraph, 节点为 State→Partial State 函数, 条件边决定下一节点; 自定义循环=手工编排节点+条件回边。checkpointer 支持 interrupt 中断恢复与 replay/fork 回放。代价: 图/状态 schema 变更即破坏历史 checkpoint 兼容, 旧线程恢复可报 TypeError。 [Graph API](https://docs.langchain.com/oss/python/langgraph/graph-api) · [Checkpointers](https://docs.langchain.com/oss/python/langgraph/checkpointers) · [Backward compat](https://docs.langchain.com/oss/python/langgraph/backward-compatibility) · [issue#8629](https://github.com/langchain-ai/langgraph/issues/8629)
2. **OpenHands**: 循环接口=每迭代调用 agent.step(State)→Action; append-only 事件流(Pydantic 类型化)兼作记忆与集成点, 订阅者含 agent_controller/runtime 等; 定制手段=子代理委派(spawn/consolidate)。已知痛点: response_to_actions 为模块级函数, 子类难覆写。 [agenthub README](https://github.com/OpenHands/OpenHands/blob/1.6.0/openhands/agenthub/README.md) · [Events](https://docs.openhands.dev/sdk/arch/events) · [Delegation](https://docs.openhands.dev/sdk/guides/agent-delegation) · [issue#8025](https://github.com/All-Hands-AI/OpenHands/issues/8025)
3. **AutoGPT**: 官方架构笔记解释弃用 legacy Plan-Act(REPL 式)原因: 老代码"缺抽象边界、全局状态封装差、难演进", 2023 选择破坏性重构为 Forge 组件化 agent; 失败模式=无限循环与 token 失控; 旧插件体系整体废止("legacy plugins no longer work")。 [ARCHITECTURE_NOTES.md](https://github.com/Significant-Gravitas/Auto-GPT/blob/v0.4.5/autogpt/core/ARCHITECTURE_NOTES.md) · [Re-arch #4770](https://github.com/Significant-Gravitas/AutoGPT/issues/4770) · [Component Agent](https://agpt.co/docs/classic/forge/component-agent-introduction)
4. **smolagents**: 单一 MultiStepAgent 循环, 每步 thought→action→observation; CodeAgent 以可执行 Python 代码为 action(本地或沙箱执行); ICML'24 论文实测代码动作优于 JSON 动作(arXiv:2402.01030)。 [smolagents ReAct](https://huggingface.co/docs/smolagents/main/en/conceptual_guides/react) · [CodeAct](https://proceedings.mlr.press/v235/wang24h.html)
5. **CrewAI**: 循环定制收敛在 Process 层而非循环体: Sequential 顺序链 / Hierarchical 自动创建 manager agent 负责分派与验收(manager LLM 须正确配置)。 [Processes](https://docs.crewai.com/edge/en/concepts/processes) · [FAQ](https://docs.crewai.com/enterprise/resources/frequently-asked-questions)

## 子问题2: 稳定内核 vs 可替换内核(插件故障隔离)

6. **微内核教训**: Chen&Bershad(1993)实测 Mach3+UNIX server 较单体内核 Ultrix 内存系统性能显著恶化; Liedtke 的 L4 把 IPC 降至 45–121 cycles, 证伪"微内核 IPC 必慢"; Torvalds 1992 论战主张单内核开发更快更稳。 [CB93](https://people.eecs.berkeley.edu/~prabal/resources/osprelim/CB93.pdf) · [L4 HotOS-VI](https://dl.acm.org/doi/10.5555/822075.822414) · [Torvalds 评论](https://www.osnews.com/story/14571/)
7. **Chromium**: 设计文档明言"渲染引擎不可能不崩溃", 以多进程切分故障域; "插件是浏览器不稳定主因", 方案=插件独立进程运行; 渲染器崩溃浏览器存活; MV3 把扩展后台迁至 service worker(移出主线程), 代价是 SW 挂起/事件超时可靠性坑。 [Multi-process](https://www.chromium.org/developers/design-documents/multi-process-architecture/) · [Plugin Architecture](https://www.chromium.org/developers/design-documents/plugin-architecture/) · [ESW lifetimes](https://developer.chrome.com/blog/longer-esw-lifetimes)
8. **Firefox**: 选择"稳定内核+受限扩展 API": 2017-11 起仅支持 WebExtensions(与多进程 e10s 兼容), 进程内旧插件(overlay/bootstrap/SDK)全部停用, 官方理由=更安全更稳定。 [MDN Add-ons](https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons) · [Mozilla blog](https://blog.mozilla.org/en/firefox/new-firefox-add-ons/)
9. **Linux 分级容错**: oops/BUG 行为由 panic_on_oops 定级: 0=尝试继续运行, 1=立即 panic; 模块异常只置 taint 标志、不阻断启动, 但 taint 内核的 bug 报告常被开发者忽略。 [sysctl 文档](https://docs.kernel.org/admin-guide/sysctl/kernel.html) · [Tainted kernels](https://kernel.org/doc/html/latest/admin-guide/tainted-kernels.html)
10. **VS Code**: 全部扩展运行于独立 Extension Host 进程(可多宿主 local/web/remote); 扩展宿主崩溃仅报"terminated unexpectedly", 编辑器存活; 官方提供 Extension Bisect 二分定位劣化扩展。 [Extension Host](https://code.visualstudio.com/api/advanced-topics/extension-host) · [Bisect](https://code.visualstudio.com/blogs/2021/02/16/extension-bisect) · [issue#111486](https://github.com/microsoft/vscode/issues/111486)
11. **OSGi/Eclipse(与 dsh 插件树最同构)**: 非可选 import 不满足时仅该 bundle 停在 INSTALLED, 框架与其余 bundle 照常启动; Bundle-ActivationPolicy: lazy 把激活推迟到首次类加载, 启动器按激活策略拉起, 规避"单插件阻塞全局启动"。 [OSGi Core 8 §3.2.1.1](https://docs.osgi.org/specification/osgi.core/8.0.0/framework.module.html) · [equinox-dev](https://www.eclipse.org/lists/equinox-dev/msg08241.html)

## 结论
1. Loop 定制业界收敛为显式图/事件流+检查点持久化, 而非改死循环代码。
2. 单插件阻塞整体启动被 OS/浏览器/IDE 一致视为设计缺陷, 标准解=optional 依赖+lazy 激活。
3. 故障隔离首选进程边界(Chromium/VS Code), 无法进程隔离则降级+污染继续运行(Linux taint)。
