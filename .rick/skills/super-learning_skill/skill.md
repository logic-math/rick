---
name: super-learning
description: super run 结束时的学习 subloop——学习整个会话执行过程（事件流/成本/ESQ/故障史），产出对 super-loop 与 super skills 的改进提案（经 human 确认落地）——super 自改进的学习闭环
execution_plane: local
---

# skill:super-learning（独立 subloop）

## 触发场景

**super run 结束时**（成功/失败/优雅退出——所有退出路径）加载：学习整个会话执行过程，改进 super-loop 与 super 的各种 skill。

（激活词：super 学习、run 复盘、自改进、super-learning）

## 职责边界

**只做学习**——从会话执行轨迹中提炼系统性模式，产出仓级知识改进（super-loop.md / .rick/skills/super-*/）；**不做执行**（不跑实验、不修 env、不巡检）。

| 归本 skill 管 | 不归本 skill 管 |
|---|---|
| 会话轨迹的模式提炼（好/坏） | env 级资产进化（→ super-loop Step 7） |
| super-loop / super skills 改进提案 | 实验失败归因（→ debug agent / event-autopsy） |
| 改进的证据链（数据支撑） | 基础设施故障修复（→ super-watchdog） |
| 人类确认后的知识落地 | 质量评分（→ super-eval） |

## 学习流程（四步）

### 第一步：采集执行轨迹（确定性数据）

```bash
# ① ESQ 四维报告（进化质量全景）
python3 /Users/logic/workdir/AI_CODING/rick/super/tools/esq_report.py <workspace>
# ② 成本报告（角色占比/昂贵迭代/上下文膨胀）
python3 /Users/logic/workdir/AI_CODING/rick/super/tools/cost_report.py --workdir <workspace>
# ③ 事件流统计（故障史/调整史/停机史）
sqlite3 <workspace>/super.db "SELECT event_type, count(*) FROM events GROUP BY event_type ORDER BY count(*) DESC;"
# ④ 会话全过程：super.log + decisions/ + domain_knowledge.md 的增量
```

### 第二步：对照 super-loop 与 skills 找差距（模式提炼）

学习对象 = **整个 super 生态的 loop+skill 资产**，逐项对照执行轨迹找「指引与现实的差距」：

| 资产 | 对照什么 |
|---|---|
| `.rick/loops/super-loop.md` | 实际执行流程与 loop 文本的偏差（哪步走不通/哪步多余） |
| `.rick/skills/super-watchdog_skill/` | 巡检发现的故障模式 vs playbook 覆盖度（哪类故障没指引） |
| `.rick/skills/super-eval_skill/` | ESQ 指标是否真实反映质量（哪个指标失真） |
| `.rick/skills/gen_env_skill/` | env 构建指引与实战的差距 |
| `.rick/skills/super_cost_analysis/` 等 | 成本浪费模式有没有新形态 |
| `super/prompts/*.md`（builder/coding/debug/adjust×7） | agent 行为与 prompt 指引的偏差（如 L1 预检遵守率） |

**提炼维度**：
- **好模式**（轨迹证明有效的指引——保持/强化）
- **坏模式**（系统性浪费/失败——跨 env 都会出现的那种：重复检索/上下文膨胀/无效 debug/巡检盲区）
- **盲区**（发生了但没有任何 loop/skill 指引覆盖的事件）

### 第三步：产出改进提案（证据链 + 落地动作）

每条提案必须带证据链（事件 seq / 成本数据 / ESQ 读数引用）：

```markdown
## super-learning 提案（run: <日期区间>）
### 提案 N：<改进点>
- 证据：<数据/事件引用>
- 差距：<哪个 loop/skill 的哪段指引缺失/失真>
- 落地：<改哪个文件哪一段，具体改法>
- 风险：<可能的副作用>
```

### 第四步：人类确认后落地（红线）

1. **所有仓级知识改动（super-loop.md / skills / super/prompts/）必须经 human 确认**——学习 skill 只产出提案，不直接改
2. **两级接通纪律**（super-loop 既有规则）：env 级问题改 env（该 run 的搜索 DNA），系统性问题才升到仓级（loop+skill）；不越界
3. **提案去重**：学习前先查 domain_knowledge / decisions/ 是否已有同类结论——增量学习不重复
4. **数据说话**：无证据链的直觉不算提案

## 回执格式

```
[super-learning] run <区间> 学习完成：N 个好模式确认 · M 个坏模式提案 · K 个盲区
提案清单：<一句一条，含证据引用>
待 human 确认后落地。
```

## 相关组件

- 上游：super-loop Step 7（成本复盘 + 两级接通——本 skill 是其学习闭环的 subloop 化）
- 数据源：esq_report / cost_report / super.db events / super.log / decisions/
- 下游：human 确认 → 落地到 loop/skill → 下一个 run 继承改进
