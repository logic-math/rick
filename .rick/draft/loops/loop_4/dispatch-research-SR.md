# 派发：research subagent — S-R 辩证逆转（逆转逻辑尽调 + 可选项）

N2 已通过（human 选定主要矛盾=对模型输入的可控性，M3）。进入 S-R。S-R 核心追问："如果 X 是必然发生的前提，要想实现 Y，我们应当如何？" research 对逆转逻辑做尽调，为 human 给出可选项。

**先读**（如未在上下文）：
- `loop_4/prompts/skill_research.md`、`loop_4/prompts/research.md`
- `loop_4/briefs/research-report-N1.md`（系统描述符 node/input/output/inner/edge + 稳态 A→B）
- `loop_4/briefs/批判门禁-N2.md`（M1–M8 打分 + human 选定 M3）
- `loop_4/briefs/批判门禁-E-r5.md`（E 收敛结论）
- `loop_2/briefs/research-report-{3,4,5,7}.md`（rick/pi 已尽调事实，复用）

无 `.rick/config.json`，信源权重默认，高置信 ≥0.8。运行时适配同前：你即调研执行者，可直接用 Read/Grep/Bash/WebFetch/WebSearch，保留尽调树/MECE/加权/R7/落盘/`git restore` 全部约束。

---

## 前序判断

- 主要矛盾（human 选定）：**对模型输入的可控性**（M3）——rick 确定性提取/强制执行（输入可控）vs LLM 参数记忆有损+非确定（A7 CONFIRMED 内禀，输出不可控）。控制手段=治理上下文熵增；输出侧机制=失败模式管理（doing_loop DEBUG/check/sense）。
- 核心价值=有限迭代最大化改进（非单调，含回退/震荡/局部最优）。
- 系统描述符：node=human/rick/pi/LLM/外部存储；edge=human↔rick、rick↔pi（系统提示词注入）、rick↔外部存储（确定性提取）、pi↔LLM（compaction）、LLM↔外部存储（skill 注册）、pi↔human（简报）。
- 稳态 A（rick+ai_cli+claude code）→ B（rick+pi+深度定制：二进制/skill 系统级/自定义 compaction/subagent 递归）。

## 任务

### 1. 阻碍识别（基于系统描述符 node/edge）

- **X（必然前提/阻碍）= LLM 输出有损+非确定**（A7 CONFIRMED，LLM node 内禀，不可消除）——对应 LLM node + pi→LLM→output edge。
- **Y（期望）= 有限迭代最大化改进/可靠解决 G′**。

### 2. 逆转逻辑

"若 [LLM 输出有损+非确定（A7 内禀不可消除）] 是 [有限迭代最大化改进解决 G′] 的必然前提，则 [可靠解决 G′] 应当 ___"

尽调 rick 现有机制如何填补逆转（基于代码 + loop_2 已尽调）：确定性提取（rick↔外部存储 edge：ContextManager+GenerateDoingPromptFile）/ 强制执行（doing.md 不可跳过+doing_loop Step 0-5）/ 迭代框架（doing_loop 3 轮+DEBUG Phase 1-6）/ 失败模式管理（check 门禁+sense 批判门禁+判断反馈→管理回退/震荡/局部最优）/ compaction 抗熵增（pi 自定义 compaction 保留 system prompt）。

### 3. 替代路径（research 调研的可选项，供 human 选择，含利弊，不替 human 推荐）

在 X（输出非确定）必然前提下，实现 Y（可靠解决 G′）的替代路径：
- 不同 compaction 策略（保留 system prompt+自定义 firstKeptEntryId vs 默认 auto-compact）
- 不同迭代框架对比（sense+doing loop vs Self-Refine/Reflexion vs 重复采样——复用 E-r4 节点 D 文献）
- RAG vs 上下文工程（确定性提取的不同实现路径）
- skill 系统级注册（提升确定性触发概率）
- subagent 递归（分层迭代，doing_loop Step 3 Main→Sub）
- 二进制部署脱离 node（V0，控制手段的部署形态）

### 4. human 启发性追问（照 sense_loop S-R 格式）

- 如果 [LLM 输出非确定] 是不可避免的前提，实现 [可靠解决 G′] 的最意想不到的路径是什么？
- 什么看似阻碍的力量（输出非确定/回退/震荡），其实可以转化为推动力？
- 在 [输出非确定] 必然的前提下，[可靠解决 G′] 的"逆向工程"是什么？

## 交付标准

按 S-R 简报格式：阻碍（node/edge）+ 逆转逻辑 + 替代路径可选项（含利弊，不推荐）+ human 启发性追问 + R7 上报。

**禁止**：简报含倾向性（不推荐某路径）、替 human 选择、无事实支撑构建选项。

## 产物写入

主报告：`loop_4/briefs/research-report-SR.md`。

## 返回

S-R 简报（阻碍+逆转逻辑+可选项+追问）即为最终输出。
