# grilling L3 批量提问——第 2 层（模块内部层）

> 2026-10-31 ｜ 你的第 1 层裁决已回填 design-tree.md。第 2 层自查已完成（agent 部署/渲染机制已核实），仅剩 3 个判断节点。附推荐。

## Q14（J2-1）mission-builder 的归属

你裁决了「每个 skill 都升级为子代理」。但 mission 面试必须由 parent 与你交互（子代理无法直接对话）——mission-builder 若为 agent，只能做「面试原话的格式化转写」（把 parent 手里的对话整理成 mission.md，多一次派发往返 ≈1–2 分钟）。

**推荐 A：parent 面试并直接落盘 mission.md**（mission=你原话的结构化，无重计算；面试纪律写入 loop prompt；这是「全 skill 子代理化」的唯一例外）。备选 B：parent 面试 → mission-builder agent 转写（严格字面一致）。

## Q15（J2-2）学习知识记录的文件形态

你定的功能：费曼过程记录=学习知识记录 → 知道你差哪/不差哪 → 下次详略得当。

**推荐**：两个文件分工——
- `loop_N/zpd.md`（zpd-assessor 独写，每次费曼/探针判断**追加一节**：日期+课次+档位+对错条目+缺口清单+下一课详略建议）= 学习知识记录的细档
- `loop_N/anchors.md`（recall-review 在 ⑤ 收口时蒸馏：每课 ≤5 条「关键词+你的卡点原话+未解矛盾」）= 跨 loop 探针源（你 Q3 已裁）

下次会话 ② ZPD 初始化 = 读历史 loop 的 anchors.md 抽关键词出探针 + zpd.md 提供基线档位。

## Q16（J2-3）originality-sparker 的调研方式

它需要「真实分歧来源 + 异领域类比」（跨基座>同基座>role-play，research-L1 [A-ID]）。

**推荐 B：sparker 自带 web 检索**（它的工具本就含 web_search/fetch_content）——task 带：你的费曼原话 + 当前课概念 + research_N 路径（基础事实）；它定向检索「与你观点存在真实分歧的来源 + 跨领域结构类比」，产出差异化启发内容。需要深度尽调时 fallback：请 parent 派 research。
备选：A 串行 research→sparker（深但慢）；C 只消费已有 research（快但异质性弱）。

---

**其余 L2 细节已按你的裁决固化**（不需再问）：agent 契约表、存储布局、parent 状态机、节奏规则、teach 模板删除/SENSE 升级（问题还原）、六 agent 经 `rickAgents` 清单部署。
