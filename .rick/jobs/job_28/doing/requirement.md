实现 RFC: /Users/sunquan/ai_coding/CODING/rick/.rick/draft/rfc/rfc-升级-human-loop-使其更具批判性-2026-08-02.md

---

## Grilling 澄清结论(2026-08-02)

### Layer 1 — 文件架构层

| 模块 | 决策 |
|------|------|
| sense_loop.md 角色 | 替代 human_loop.md 成为 main agent 协议(原文删除) |
| 旧文件处理 | sense_subagent.md + human_loop.md 完全删除 |
| 命令名 | `rick human-loop` 保留,内部加载新协议 |
| subagent 文件位置 | templates/ 根目录下,与现有模板同级 |

### Layer 2 — 模板内容层

| 模块 | 决策 |
|------|------|
| D-R2 引擎落地 | 拆分:think 识别+分类价值性 / research 调研事实性 |
| think pipeline | 识别(隐含)→ 打分(按失败风险)→ 选最高风险 |
| research 遍历策略 | BFS+全量收集,队列空=事实模糊性归零(L1 落地) |
| exporter 形式 | 大纲+内容两阶段(先确认大纲后填内容) |
| 派发要素 | 简化为 5 项:主题+草稿路径+前序判断+任务派发+结果核验 |
| 重试上限 | 经验值 5 次,可配置(L6 落地) |
| 简报落盘 | 简化为 2 产物:briefs/ + judgment.md(移除 loops.md+progress.md) |
| think/research 契约 | 串行管道:think 选最高风险 → research 调研该风险 |
| learning.md 同步 | 改写 learning.md,删除步骤一 progress.md 同步和步骤二 loops.md 孤儿检查 |
| D-R1 落地 | 两处都改:sense_loop.md S1 段落 + skills/sense.md S1 B 段落 |
| L2 良质判定 | sense_loop 提议+human 显式确认 |
| 重试配置载体 | .rick/config.json 新增 HumanLoop struct 嵌套字段 |

### Layer 3 — Go 代码集成层

| 模块 | 决策 |
|------|------|
| embed 加载 | 新增 4 个 //go:embed + 4 个 case 分支,删除 human_loop + sense_subagent |
| 函数签名 | `GenerateHumanLoopPromptFile` 返回 (mainFile, thinkFile, researchFile, exporterFile, error) |
| 模板变量名 | 删除 human_loop/sense_subagent,新增 sense_loop/think/research/exporter |
| Config struct | 嵌套 `HumanLoop{ MaxRetries int }`,默认 5 |

### Layer 4 — 测试与迁移层

| 模块 | 决策 |
|------|------|
| 测试更新 | 重写 human_loop_prompt_test.go 为新架构(8 处旧名断言全部更新) |
| 历史 loop_1 | 不动,作为历史记录保留 |
| skill 文件 | 为 think/research/exporter 各写一个 skill 文件(templates/skills/ 下) |
| RFC/learning 同步 | 同步更新 learning.md(不动 RFC 文件,作为历史决策记录) |

### Layer 5 — 叶子层(L1-L6 + 关键代码路径)

| 模块 | 决策 |
|------|------|
| L1 死循环保障 | 信任 BFS 自然收敛,不加轮次上限(⚠️ 风险:新假设不断涌现可能死循环,learning 阶段观察) |
| D-R3 think 落地 | 删除显式"识别假设"职责陈述,保留"分析→打分→选最高风险"流程,识别作为隐含前置 |
| D-R4 R7 描述 | 简化为两步:"research 调研失败的假设标注为事实性假设,加入上报列表" |
| 文件路径 | templates/{sense_loop,think,research,exporter}.md + templates/skills/{sense,think,research,exporter}.md |
| L3 边界 | sense_loop 调度,think 主动识别假设(X5 延续已隐含确认) |
| L4 替换路径 | go:embed 指令替换 + getEmbeddedTemplate case 替换 |
| L5 派生修订载体 | D-R1 双文件 / D-R2 拆分 think+research / D-R3 think.md / D-R4 research.md |
| L6 5 次依据 | 经验值,可配置(HumanLoop.MaxRetries 默认 5) |

### 三层复核机制

- 第一层:research agent(事实性假设澄清,BFS+全量收集)
- 第二层:sense_loop(main agent 通用假设澄清,二层门禁)
- 第三层:human(R7 上报项 + 良质跃迁确认)

### 阶段门禁推进条件

- 该阶段所有假设(事实性+价值性)被澄清
- 良质跃迁由 sense_loop 提议 + human 显式确认(F1 修正)

### 整体结束条件

- 良质跃迁(human 确认)
- 所有假设澄清(BFS 队列空 + R7 上报项已决策)
- human 确认 exporter 形式与大纲

### 遗留观察项(供 learning 阶段复盘)

1. ⚠️ L1 风险:BFS+全量收集无上限,新假设不断涌现可能死循环。learning 阶段需观察实战是否收敛
2. ⚠️ 简化为 2 产物(briefs+judgment)后,learning.md 的概念地图来源被切断,需用 briefs/E1.md 替代 loops.md
3. ⚠️ 重试 5 次经验值的合理性需在实战中观察

### 不在本次 job 范围

- 历史 loop_1/prompts/ 不迁移
- .rick/RFC/human-loop-upgrade-rfc.md 不动(历史决策记录)
- dream 阶段的 loops/skills 进化逻辑暂不涉及
