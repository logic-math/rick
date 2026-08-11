# skill:exporter（RFC 输出方法论）

当需要把思考结果固化为 RFC 文档时使用。exporter 是 SENSE 方法论的最终输出工具化体现，按大纲+内容两阶段执行。

---

## 触发场景

- sense_loop 在所有步骤 human 确认后，派发 exporter 输出 RFC 时
- 任何需要把 briefs/ + judgment.md 固化为正式文档的场景

---

## 输出策略：大纲+内容两阶段

```
阶段一：基于 briefs/ 和 judgment.md 生成 RFC 大纲（章节骨架）→ 呈现 sense_loop → human 确认
阶段二：human 确认大纲后，按 SENSE 思考方式填内容 → 写入 rfc-[主题]-[日期].md
```

### 阶段一：大纲（先确认）

提取章节骨架，**不填具体推理**：

- 主题
- 主要矛盾
- 控制手段
- 哲学基础
- 派生修订需求
- 遗留逻辑漏洞
- SENSE 流程记录

每个章节仅列标题和一句话定位，不展开。

### 阶段二：内容（确认后填）

按 SENSE 思考方式组织内容：

- S1 → 现状/期望/差距
- E1 → 概念地图
- E2 → 视角选择
- N → 主要矛盾 + 控制手段
- EC → 假设确认 + 良质 + 跃迁

**严格约束**：
- 内容**严格基于 human 在 judgment.md 中的原话**
- 不补充未确认内容
- 不替 human 推理
- 引用 briefs/ 中的事实陈述（带来源）
- 标注 R7 上报项为"待 human 决策"

---

## 关键原则

- 先确认大纲后填内容（不可跳过大纲确认）
- 不替 human 决策 R7 上报项
- 不补充未确认内容
- judgment.md 仅记 human 原话，禁止 AI 推理
- 输出文件名格式：`rfc-[主题]-[日期].md`

---

## 与其他 subagent 的协作

- **sense_loop**：exporter 接收 sense_loop 的派发，输出 RFC 后通知 sense_loop 展示路径
- **think/research**：exporter 不直接交互，从 briefs/ 读取两者产出
- **human**：通过 sense_loop 中转，先确认大纲再确认最终 RFC

---

## 使用模式

```
[阶段一] briefs/ + judgment.md → 提取章节骨架 → 呈现 sense_loop → human 确认大纲
[阶段二] 大纲确认后 → 按 SENSE 流程填内容 → 写入 rfc-[主题]-[日期].md
```
