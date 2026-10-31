# grilling L3 批量提问——第 1 层（模块层）+ 根层确认

> 2026-10-31 ｜ 依据：ADR-human-loop-v7（D1–D10 已裁决）+ design-tree.md 第 0/1 层 + research-L1.md（⚠️ 离线整理，等级 C，网验推迟）
> 每问附：已调研上下文（一两行）→ 选项与权衡 → 推荐。

## Q0 根层 OKR 确认

O：human-loop v7 按 ADR 落地为 rick 可运行 loop+skill 架构，目标函数=启发 human 原创性思考，消除 v6 啰嗦/低效，未决项全部裁决闭环。
KR1 架构落地（五步 loop+全 skills）｜KR2 症状消除（三来源+篇幅硬约束）｜KR3 机制闭环（复习/掌握/原创三机制可操作）｜KR4 判断不吞（全部判断节点经 human 裁决）｜KR5 实跑验收（human 主动判断）。

**推荐**：确认此 OKR 集（充分性推导已自检）。

## Q1（J-arch）v7 的架构形态——源码真身在哪儿

调研：v6 现实现=Go 模板（internal/prompt/templates/sense_loop.md）→ `rick human-loop` 命令渲染到 loop_N/prompts + pi agents（~/.rick/pi/agent/agents/）。而 ADR D7（V12）说「按 rick 的 loop+skill 格式内化」。.rick/loops 里的 loop 是数据文件（name/trigger frontmatter，被 easy/plan prompt 注入发现），internal 模板是代码资产（改内容要重编译）。

选项：A 纯 .rick 数据（无 Go 改动，但 rick 无「按 loop 启动会话」的通用入口，KR1 的启动方式无解）；B 改写 internal 模板（最小 Go 改动，但与 D7 字面相悖、方法论存代码树）；**C 混合：.rick/loops/human-learning-loop.md + .rick/skills/* 为唯一真身，`rick human-loop` 命令读它们渲染实例 prompt（Go 改动=换渲染源，目录分配/session_id/resume 全保留）**。

**推荐 C**：方法论归 .rick（数据、可编辑、被 easy/plan 上下文发现），命令保留（KR5 需要一键启动+会话持久化+resume）。

## Q2（J-teach-agent）teach agent 去留

调研：ADR §4.3 明文「skill=注入当前上下文（元数据常驻）、agent=独立上下文只回摘要」——即 lesson-writer 等由会话本体执行，v6 的「fresh 重读 6 份 canon」成本消失；代价=主会话上下文随课程线性增长（5 课 ≈ 50–100KB，pi 可承受）。

选项：**A 按 ADR：无 teach 子代理，六 skill 全 in-context，仅 research 留 agent（独立上下文、回执返回）**；B 保留 teach agent（fresh 上下文但每课重读材料，v6 成本结构回潮）。

**推荐 A**：ADR 已裁（§4.3 是 D7/D10 的组合推论）；上下文增长用「lesson 落盘、会话内只留摘要+路径」控制。

## Q3（J-anchor，ADR §7 Q1）可重访锚的载体

调研：ALEKS 存离散知识状态+fringe（刚学会/可学前沿）+定期重评估建模遗忘；LLM 侧（MemGPT/Mem0）=结构化画像块+可检索事件库。v6 loop 目录已有 judgment.md（human 原话）、lessons/teach_N.md、values.md 可被读取。

选项：A lesson 卡点条目（分散在课文里，需回读全文）；B judgment 原话（有，但非按课组织）；**C 每课收尾时写 loop_N/anchors.md：每课 ≤5 条「关键词 + human 卡点原话 + 未解矛盾」，跨 loop 探针 grep 全部 loop 的 anchors.md 抽关键词**；D research 缺口清单（膨胀快，且缺 human 侧痕迹）。

**推荐 C**：锚=关键词+卡点+未解矛盾三元素（对齐 D4「从卡点条目抽，而非结论」）；mem0 式「决策 ADD/UPDATE/DELETE」不引入（D7 自研降级，不做记忆系统）。

## Q4（J-three-times，ADR §7 Q2）「三次不行」的计数对象

调研：ADR D9 判据=human 主动判断、连续三次不行即删；未定「三次什么」。信度检验（三次同类失败=噪声排除）vs 效度检验（三次异类失败=构念污染）。

**推荐：三次同类**（同一 skill 的同类失败，如「原创启发三次都没引出差异性想法」）→ 删该机制；异类失败记入 judgment 留痕但单独计数。理由：单 human 样本下先压噪声；构念污染靠 KR5 实跑观察。

## Q5（J-error-absorb，ADR §7 Q3）误差吸收规则

调研：Tulving & Pearlstone——部分线索能救回=提取失败；换措辞+多线索两轮仍败=真缺口；teach-back「二不过=缺口」；AutoTutor 提示梯度 pump→hint→prompt→assertion（级内失败升一级，最终直给+回退）。

选项：A 先给答案再讲；**B 二次探针链：首次答不出/答错→给部分线索（类别/语境）复测→仍败→换措辞→仍败=判「弱」并触发回退或补讲**；C 三探针交叉（成本高）。

**推荐 B**：两级探针（部分线索→换措辞），两轮不过=弱/缺口；「先讲 vs 先给答案」的取舍=仅当二次探针仍败才直给答案（AutoTutor assertion 层），护栏成本可控。

## Q6（J-v9-boundary，ADR §7 Q4）「想不起来=无能力」的适用范围

调研：ADR §6 已警告——提取失败≠不会，搬到即时判决会把「一时想不起来」误判为「不会」而倒退到已会的课（mission 抱怨的新增来源）。

**推荐：只用于跨 loop 探针（V9 原位）**；即时达标判决走 Q5 的二次探针链，二次探针仍败才降档。

## Q7（J-speak-out，ADR §7 Q5）「讲出来」的验收位置

调研：D10 说「让学生讲出来不需要（只需理解）」（skill 划分语境），D6（V11）说「讲出来可作达标判据」——矛盾的解释空间在「是不是独立 skill」。

**推荐：讲出来=验收动作而非独立 skill**——zpd-assessor 在判档探针里要求 human 阐述（teach-back 式），内容按事实符合度判档；不设独立「讲出来」skill（尊重 D10 划分），但保留 D6 的达标判据地位。

## Q8（J-mastery-boundary，ADR §7 Q6）强/中/弱三档的操作边界

调研：生产 LLM 导师无公开量化 rubric（判级=追问链+行为规范）；量化形态只有 ITS 传统（AutoTutor LSA 0.4–0.5）。E12：70/80/90 阈值属约定无实证。

选项：A 纯自然语言判级（无边界定义，判级漂移）；**B 自然语言 rubric+最低操作判据：强=无错误断言且关键概念关系齐（能自述「X 与 Y 的关系」）；中=无错误断言但覆盖不全，或 1 处次要偏差；弱=有错误断言或关键概念缺失，或二次探针不过**；C 量化计数（错误断言数/覆盖率阈值，E12 已示无实证基础）。

**推荐 B**：与 D6（阐述×事实符合度）同构，可核对（错误断言可指出、关键概念可点名），不引入伪精确。

## Q9（J-coldstart，ADR §7 Q7）首次 ZPD 初始化冷启动

调研：V9「领域相关性启发」无历史 loop 可抽时怎么办。GenMentor 用简历+skill gap 建初始 profile。

**推荐：mission 面试强制问「起点：你已经知道什么」（mission.md 起点字段）+ zpd-assessor 用领域核心概念出 2–3 个探针（从 research 首轮 facts 抽）校准自报起点**；探针结果直接作 ZPD 基线，不另建画像。

## Q10（J-length）lesson 篇幅硬约束数值

调研：v6 实测单课 15–21.5KB 被判啰嗦；ADR 附录：篇幅上限+事实层/推导层配比必须先于内容质量成为硬约束；LearnLM 回复 297.6 vs 423.0 tokens（更短且教学增益更高，E3）。

**推荐：lesson 落盘全文 ≤8KB（硬上限，超限拆课）；会话内呈现正文 ≤4KB（事实层摘要+推导层留白）；事实层:推导层配比 ≥2:1（推导层留白=human 补齐的空位，不是删内容）**。

## Q11（J-ascii）文本绘图的操作约束

调研：CJK 全角占 2 列+UAX #11 East-Asian Ambiguous 宽度→LLM 自绘含对齐的 ASCII 图几乎必错位；code-fence 内简单结构（线性流程/层级清单）可靠；Mermaid 更稳健但属 D8 否决的图文生成。

**推荐：文本绘图限 code-fence 内的简单结构（流程箭头链/缩进层级/表格），禁止 box-drawing 对齐图；教学图解以「结构化文本（分节+编号+关系动词）」为主**。

## Q12（J-accept）KR5 实跑验收方式

调研：D9=人的主动判断；desirable difficulties 风险（读着顺≠有用）已知并接受。

**推荐：v7 落地后 human 自选一个真实想学主题，单会话完整走五步（或明确中止并记录原因）；通过判据=human 主动判断「值得继续用」；judgment.md 留痕**。

## Q13（J-v6-fate）v6 资产处置

调研：v6=internal 模板（sense_loop/teach/research skills）+ pi agents（teach/research）+ Go 命令。历史 loop_1–12 目录是 v7 锚源（Q3 探针要读它们）。

选项：**A v7 替换 v6：sense_loop.md 模板重写为 v7（或 Go 改读 .rick loop 文件）、teach agent 下线（agents/teach.md + 模板移除或移 deprecated）、research 保留；历史 loop 目录不动（锚源）**；B v6/v7 并存（双入口，维护两套）。

**推荐 A**：ADR 明言「取代 v6」；并存违背单一事实源。legacy 锚兼容：zpd-assessor 探针读 anchors.md（v7 起存在）+ fallback 读旧 loop 的 judgment.md/lessons 标题。

## 附：research-L1 的置信度声明（更新）

最终版简报为**主线在线自验版**（代理宕机期间绕行 curl 直连 arXiv：2412.16429v3 LearnLM 全文、2501.15749v1 GenMentor 全文已读，另 11 篇 arXiv ID 核验；B/C 级项=AutoTutor 阶梯/LSA 阈值、AHRQ teach-back、Khan 四档分数线、Khanmigo 泄露词，均出自经典文献/公开报道但本轮未复核，见 R7 清单 8 条）。上述推荐不依赖任何单条未验结论；doing 开工前网验 R7 第 3/4/8 项即可。关键升级：LearnLM 不判学生（29 题 rubric 评模型）[A]；GenMentor U_t 递归画像 [A]；异质性排序跨基座>同基座>role-play [A-ID]；LLM 不可自评知识结构 [A]。
