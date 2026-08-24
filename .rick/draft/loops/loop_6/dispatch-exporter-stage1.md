# 派发：exporter subagent — 阶段一（RFC 大纲，OKR 结构）

**先读**：
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/prompts/exporter.md` + `skill_exporter.md`
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/judgment.md`（S/E/N/S-R/EC 全部 human 判断原话）
- `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_6/briefs/` 下全部简报（尤其 research-report-S-bestpractice.md、S-reasons-agent.md、SR-architecture.md、debate-dsh-vs-pi.md）

---

## 阶段一：RFC 大纲（先确认，不填具体推理）

**主题**：subagent 在 pi runtime 下触发概率低，优化提示词（使触发确定性提升到上限内最高）。

**human 已确认的三个目标（O，原话要点，来自 EC 判断）**：
- **O1**：将 rick 的方法层描述为一套尽可能无歧义的自然语言，作为 rick 的信息内核，以便于生成开发计划得到任意实现描述，加速后续实现层重构速度。
- **O2**：将 rick 现有的提示词触发语言与 pi coding agent 的触发语言完成等价迁移，遵循 pi 的定制开发规范以获得更好的执行效果。
- **O3**：实现 rick 所依赖的 pi 及 pi 的生态组件无感知升级。

**human 对 RFC 的要求（原话要点）**：
- RFC 可细化为一个开发计划。
- 基于 rick 当前代码现实 + pi 情况，针对三个 O 给出**具体的关键实现方法**。
- 每个 O 列举**关键结果（KR）**，描述要交付的关键结果。
- 确保所有 KR 完成后目标一定完成，**逻辑闭环**。

**任务**：
1. 读 rick 当前代码，摸清实现现状（供 KR 设计用，只读不改）：
   - `internal/prompt/templates/`（顶层 loop 文件 + skills/ 子目录，243 处自然语言 subagent 触发词、0 处 pi 语法/agent 名）
   - `internal/prompt/human_loop_prompt.go`（think/research/exporter prompt 生成与落盘）
   - `internal/prompt/*_prompt.go`（plan/dream/doing/easy/ctrl 的 prompt 生成）
   - `internal/agent/piagent/`（pi 执行器）
   - `internal/cmd/tools_init_pi.go`（pi 扩展注册 requiredExtensions=["pi-subagents","pi-web-access"]）
   - `~/.rick/pi/agent/settings.json`（当前配置）
2. 读 pi 情况（供 KR 设计用）：BP-1~BP-9（触发机制）、B1~B4（自定义 agent 机制）、skill+loop 抽象可行性、pi/dsh 辩论结论（均已在 briefs/）。
3. 产出 **RFC 大纲**（只列骨架 + 每项一句话定位，**不填具体实现细节**）：

```
## RFC 大纲 — subagent 触发确定性提升（优化提示词）— [日期]

### 主题
[一句话]

### 背景与哲学基础
[方法/实现隔离 + 核心假设（自然语言无歧义验收标准 ⇒ 等价一致开发计划，等价=功能等价）+ 协议对齐视角]

### 主要矛盾与辩证逆转
[K4 非主要矛盾；核心=工程化方法描述；深度定制 与 独立迭代 的辩证逆转]

### 三个目标（O）与关键结果（KR）骨架
### O1 [无歧义自然语言方法描述（rick 信息内核）]
  - KR1.1 [一句话：待交付关键结果]
  - KR1.2 ...
### O2 [rick↔pi 触发语言等价迁移]
  - KR2.1 ...
  - KR2.2 ...
### O3 [pi 及生态组件无感知升级]
  - KR3.1 ...
  - KR3.2 ...

### 闭环逻辑说明
[说明：所有 KR 完成 ⇒ 各 O 完成 ⇒ 触发确定性提升到上限内最高 的论证链条，指出当前链条中的假设/缺口]

### 派生修订需求
[human 各步确认的修订点]

### 遗留逻辑漏洞（R7 上报项）
[模型能力上限/触发概率量化/因果归属/转义层切换成本——待实测]

### SENSE 流程记录
[每步通过/未通过客观记录]
```

**KR 设计要求**（写进大纲的骨架）：
- 每个 KR 必须是「可交付、可验收的关键结果」，基于 rick 当前代码 + pi 机制的现实。
- KR 之间 + KR→O 的逻辑要能构成闭环（若发现某个 O 缺少闭环所需 KR，需在大纲中标注缺口）。
- 不填具体实现细节，只给骨架 + 一句话定位。

## 交付

大纲全文返回 sense_loop，落盘 `.../loop_6/briefs/rfc-outline.md`。**不进入阶段二**（等 human 确认大纲）。

**禁止**：阶段一填具体推理、补充未确认内容、替 human 决策 R7 项、judgment.md 写入 AI 推理。

## 返回

RFC 大纲全文作为最终输出返回 sense_loop。
