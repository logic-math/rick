APPROVED: true

# Job job_47 执行总结

## 执行概述

**项目目标**: 实现 INFER_RSI 参照的 super RFC——rick 的自改进搜索调度器（superopt 包：四 Agent Loop 调度 + 特征格精英档案 + 事件溯源 + 门禁体系），并以 13 轮实测迭代持续治理成本与可靠性
**实际完成**: 58 task/58 success/0 重派；13 轮迭代三段式（功能 1-3 → 治理 4-10 → 数据驱动 11-13）；521 测试全绿；env_11 深分析重校准度量口径并催生 R1-R5 五项根因修复。收尾的 release S5 提升连环失败（跨机事故链），由 job_49 接手根治——本 job 诚实记录失败，只沉淀 super 侧经验（release 教训已由 job_49 沉淀，不重复）
**整体评价**: ⭐⭐⭐⭐（4 星。理由：13 轮迭代全部「观察→假设→修复→量化验证」闭环、成本治理有真实对照数据（-20% tokens）、末期方法论升级（researcher 深读）产出密度全 job 最高；扣 1 星：①JSON hook 首版未真实验证即提交（9667d13，human 质询才补测）②release 收尾失败本可在 S3 阶段用全链路影子演练预防）

## 关键成就

1. **super RFC 全量落地 + 13 轮实测迭代**: superopt 包（events/archive/feature_space/sampler/budget/lineage + 四 Agent Loop：adjust/builder/coding/debug）从 RFC 到 521 测试全绿；29 个 gate 全部通过且 gate 自身的检查缺陷当轮修正并透明化记录（gate 修正模式贯穿全程）；58 task 零重派
2. **成本治理量化闭环（迭代 9→10）**: human 假设「debug 截断是浪费主因」→ worker 量化验证：tokens/迭代 2.18M→1.74M（**-20%**）、debug 截断 12/20 派发→**0/3**、it16 soft_gate 失败被 debug 救回（浪费的钱真的省到了）；归因 caveat 诚实标注（2 迭代未触发 adjust，保守估计 -10~-15%）
3. **env_11 事件流深分析（末期方法论转折点）**: 派独立 researcher 深读 18140 事件流，**重校准度量口径**（表面 20% 产出率/93-99% invalid 里 86% 是 API 配额空转——真实工作轮产出率 ≈61%）+ 挖出 5 项结构性根因（R1 配额熔断/R2 回合×协议失配/R3 幻影 elite/R4 处方-基座相位差/R5 停止判据），迭代 13 一轮全部落地（task57-59，gate29 全绿）
4. **agent 确定性输出（pi hook，三关验证后）**: superopt-json-guard 扩展（agent_end + deliverAs:"followUp" 的官方模式），同进程续跑修复 JSON 契约违例；修复上限 2 次防死循环、env 门控防误伤普通会话——三项真实验证（事件流时间序证据）通过
5. **agent 失败基础设施化治理（迭代 13 R1+R2）**: 三态 outcome 分类（infra/agent/真实无产出分开计）+ 配额熔断（连续 3 次→暂停+60s 探活恢复）+ 产物落盘抢救（「做完了但没说」≠「没做」：workdir 有产物直接 gate 入格）——消灭 env_11 类 68% 空转的机制基础

## 问题与教训

### 问题 1: JSON hook「写完+部署+提交」但从未真实跑过（9667d13）

**根本原因**: 「client 层能想到的兜底」和「运行时真实行为」之间隔着一层没有验证的想象——三个 bug（导出形态错/api.session?.messages 不存在/agent_settled 撞 stale-ctx 重入限制）全部是跑一次真实 pi 进程就能暴露的，第一项甚至是加载器当场报错
**解决方案**: human 质询后补真实测试：隔离 agent home 起真实 pi 进程，从 stderr 调试日志定位嵌套运行死因，读运行时源码（agent-loop.js followUp 队列注释 + _handlePostAgentRun 注释）找到官方模式，换 agent_end + deliverAs:"followUp" 三测通过（e2bfd6f）
**经验教训**: 外部运行时集成必须过三关（静态对照类型定义/真实进程加载/事件流行为断言）——已沉淀 runtime_hook_verify skill；写 hook 前先读运行时源码找官方模式，比从 API 表面猜组合快且稳

### 问题 2: 25M token 失控——观测性迟到 4 个迭代

**根本原因**: 失控发生在「跑的当时」（回合数实时可观测，NDJSON 流就在那）但初版无在线告警，只能事后从 usage 统计发现；回合预算值先拍脑袋（12）再两次实测 tune（20/24）——预算本该从产物复杂度反推（文件×动作数 ÷ 每回合动作数 = 预算下界）
**解决方案**: 迭代 5 落地 NDJSON 回合计数 + 优雅终止；迭代 12 补 turn_soft_warn 软告警（不截断）；治理段 4→5→9→10 四轮压缩了本可更短的路径
**经验教训**: 观测性先于治理——长时 agent 循环 V1 必须带「回合/token/时长三维实时计数 + 软告警事件」，事后统计是兜底不是主力（已写进 super-loop V1 最低配清单）

### 问题 3: debug 截断浪费的证据「早就有了，缺的是看的动作」

**根本原因**: 截断信号（truncated 字段）在迭代 9 之前就存在于 events.jsonl，但没人看——cost_report 的价值一半是工具，一半是「强制看」
**解决方案**: 迭代 9 建 cost_report.py 让浪费可见（12/20 截断）→ 迭代 10 修复 + 量化验证（-20%）；后续 outcome 五态分列把 infra 空转与真实失败彻底分开
**经验教训**: 事件流健康巡检例行化（每轮固定一条 cost_report 命令），把「浪费可见化」从专项 task 变成例行动作（已写进 super-loop Step 3）

### 问题 4: release S5 收尾连环失败（跨机事故链），本 job 未独立解决

**根本原因**: S3 演练绿但 S5 死——dry-run 按设计不覆盖 merge/promote/restart/health，「演练绿」的心理安全是虚假的；叠加环境迁移后「带历史注释的配置」全部过时（8413 vs 真实生产 6137）+ stopProd 快照断言漏杀 + 回滚点不可执行校验缺失——发布链路关键环节从未在真实跨机环境全链路演练过
**解决方案**: 本 job 修了 spawnDetached 相对路径 bug + 诚实交接事故现场；job_49 接手 P0 四项根治（promote 前平台校验/回滚点校验/stopProd 认领补强/health 诊断增强）+ 真实 promote 实战验证（stopped=[旧pid] skipped=[]）
**经验教训**: dry-run 覆盖不到的阶段需要「带暂停点的全链路影子演练」（隔离 state-dir + 临时端口跑真提升）；环境迁移后对带历史注释的配置项做实测校准（job_49 已沉淀 env_drift_probe skill，本 job 引用不重复）

## 知识沉淀清单

- [x] skills/runtime_hook_verify_skill/skill.md - 外部运行时 hook 三关验证协议（静态对照/加载/行为事件流断言；素材：9667d13→e2bfd6f 三 bug 链）
- [x] skills/super_event_autopsy_skill/skill.md - super 环境事件流深读四步法（重校准口径→失败分类带损失当量→结构性根因→ROI 排序；素材：env_11 R1-R5 + 61% vs 20% 重校准）
- [x] loops/super-loop.md - 三处增量：①观测性先于治理 V1 最低配清单（Step 1）②事件流健康巡检例行化 + outcome 五态（Step 3）③环境结束/跑挂后固定派 researcher 深读事件流成为硬性环节（Step 7）
- [x] domain/bugs.md - +2：pi 扩展导出形态错误（export default async function）；agent_settled 内 sendUserMessage 撞 stale-ctx（正确模式 agent_end + followUp）
- [x] domain/pi-runtime.md - +1 节 6 条：followUp 官方模式源码引用；ExtensionContext 无 sendUserMessage；消息追踪走 message_end/getEntries；handler async await 语义；导出形态；激活门控
- [x] domain/env.md - +1：darwin procfs Environ 不可用（KERN_PROCARGS2 env 不可靠）→ ownedByProd HOME 断言在 macOS 恒失效
- [x] domain/build.md - +2：pi 扩展激活门控用环境变量优于死代码函数；rsi 契约字段位置敏感（第三次复现）

## 遗留（下轮候选）

- super 调度器真实长跑验证：R1-R5 修复（task57-59）只有单测+旧事件流回归证据，未在真 pi 环境跑过完整 session 验证熔断/抢救/相位差修复的实际效果——建议下一个 super env 跑完后对比 outcome 分布
- outcome 五态分类的历史基线：旧事件流（env_11 之前）无 outcome 字段，cost_report 只能 fallback `_legacy` 桶——不影响新数据，但历史对比需要人工按分类判据重算（super_event_autopsy skill Step 1 的判据代码可直接用）
- JSON hook 的生产观测：superopt-json-guard 的修复触发次数（json_repair 计数）尚无跨环境统计——跑下一个 env 时从 events.jsonl 抽取「agent_end 后是否出现修复 user 消息」验证真实触发率
- pi hook 的 rsi 证据链：release.md 头部契约（version= 置首）在 job_49 已复证——本 job 的 release 证据归 job_49 的 S5 记录（47befe6），不重复沉淀
