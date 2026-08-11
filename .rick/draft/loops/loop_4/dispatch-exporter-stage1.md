# 派发：exporter subagent — RFC 阶段一（大纲，先确认）

sense_loop 全流程完成（S→E→N→S-R→EC，EC 维持/良质通过）。派发 exporter 产出 RFC。**本阶段只做阶段一：生成大纲（章节骨架），不填具体推理，不写 RFC 文件**——大纲返回 sense_loop 呈现 human 确认后，再派阶段二填内容。

**先读**（角色与方法论）：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_4/prompts/skill_exporter.md`（UNDERSCORE 真实 repo）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_4/prompts/exporter.md`（UNDERSCORE）

## ⚠️ 路径地图（关键：本会话文件系统分裂为两路径）

- **我的最新 loop_4 工作（READ SOURCE，含全部 human 原话 + 13 份 brief）**：`/workdir/sunquan20/AICODING/rick/.rick/draft/loops/loop_4/`（**无下划线 AICODING**）—— `judgment.md` + `briefs/`（批判门禁-E-r{1..5}.md / 批判门禁-N1-r{1,2}.md / 批判门禁-N2.md / research-report-E-r{2,4}.md / research-report-N1.md / research-report-SR.md / research-report-EC.md）
- **现有 RFC（FORMAT 参考，READ）**：`/workdir/sunquan20/AI_CODING/rick/.rick/draft/rfc/rfc-升级-human-loop-使其更具批判性-2026-08-02.md`（**UNDERSCORE**）
- **RFC 输出（阶段二才写，UNDERSCORE 真实 repo）**：`/workdir/sunquan20/AI_CODING/rick/.rick/draft/rfc/rfc-[主题]-[日期].md`

> 注：无下划线路径不是 git repo、仅存 loop_4；下划线路径是真实 git repo（含全树 + .git + prompts + 现有 RFC）。读 briefs/judgment 用无下划线，读现有 RFC/写新 RFC 用下划线。

## 前序判断（sense_loop 核心结论，供大纲提取；完整原话见 judgment.md）

- **主题方向**（建议，可调）：rick→pi 迁移的价值基础与架构定位（sense_loop 推导）
- **核心假设**：∃G 外 G′（LLM 未见过+无法一次性解决）→ 需迭代/实验探索 → rick 存在（A7 有损+非确定 / A5 G′存在 / A15 zero-shot 不选 rick 编排 / A18 单轮不足多轮改善，全 CONFIRMED）
- **哲学基础**：LLM 参数权重=有损+非确定压缩（A7 内禀，与上下文长度正交）；智能有极限（全知≠全能）；扩展心智=确定性信息存储/提取
- **rick 价值论**（D3′重锚）：价值=弥补参数记忆有损+非确定（LLM 内禀，与训练成本正交，刚性）；手段=应对上下文熵增（EC nuance: 手段层部分弱化，价值主体锚定 A7+A15+human 判断者）；实现=确定性编排+强制执行+含判断的选择+迭代框架
- **架构定位**（D3）：rick=引导程序（引导人类正确模式[pi 不可内化]+引导 pi 加载系统提示词）；价值主体=rick；"可完全内嵌 pi"≠"功能不可替代"
- **核心价值**=有限迭代最大化改进（非单调，含回退/震荡/局部最优；失败模式管理强化 rick 价值）
- **主要矛盾**（M3，human 选定）=对模型输入的可控性（rick 确定性输入控制 vs LLM 内禀输出非确定）
- **控制手段**=治理上下文熵增
- **收敛机制**=有序上下文→最大化改进（非单调，含失败模式管理）
- **逆转逻辑**（S-R 三层，human 完全接受）：层 1 可控性转移（输出→输入侧）/ 层 2 非确定吸收（迭代+失败模式管理）/ 层 3 非确定转化（阻碍→推动力）
- **替代路径**（S-R P1–P6）：compaction 策略 / 迭代框架 / RAG vs 上下文工程 / skill 系统级 / subagent 递归 / 二进制部署
- **EC nuance**：A17 手段重锚（锚定 A7+A15+human 判断者，非"应对上下文熵增"）；A18 边界细化（对真正 G′ 单轮不足，对 bounded G′ 承认弱化）
- **遗留逻辑漏洞（R7/未决）**：M8 迭代策略优劣（rick sense+doing loop vs Self-Refine/Reflexion/重复采样 最优性未 benchmark）；A7 节点 A/B R7（LLM 内部机制源码不可访问，多源交叉印证但置信度未达高）；human 判断者不可替代性（N2 提示，未纳入主要矛盾）；M7 rick 长期形式（pi 社区若原生实现 rick 结构则 rick 形式消失，长期开放变量）

## 阶段一任务

按 exporter.md 阶段一 + 现有 RFC 格式，提取**章节骨架**（不填具体推理）：

```
# RFC — [主题] — 2026-08-10
## 主题 / 完成日期 / 哲学基础 / 核心假设 / rick 价值论 / 架构定位 / 主要矛盾 / 控制手段 / 收敛机制 / 逆转逻辑（三层）/ 替代路径 / EC nuance / 派生修订需求 / 遗留逻辑漏洞（R7）/ SENSE 流程记录
```

每章节仅标题 + 一句话定位（引用 human 原话要点），不展开推理。

## 交付标准

返回大纲全文（章节骨架）给 sense_loop 呈现 human 确认。**不写 RFC 文件**（阶段二 human 确认后才写）。

**禁止**：阶段一就填具体推理、替 human 决策 R7、补充未确认内容。

## 返回

大纲全文即为最终输出。
