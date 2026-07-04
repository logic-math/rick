# human-loop 升级：判断力强化学习设计方案

**日期**：2026-07-02
**类型**：RFC（Request for Comments）
**状态**：草稿

---

## Subject（澄清问题）

### 现状

- human 在启动 loop 前不做任何复杂度判断，直接进入追问流程
- 追问进行到一半才发现问题超出范围
- 无跨会话记忆；无结构化领域知识底座
- human 的判断过程从未被捕获到信息流里

### 期望

- `.rick/draft/` 机制：
  - `concepts/`（嵌套概念地图，human 主动下钻，AI 标记建议展开节点写入 loops.md 作为执行信号）
  - `progress.md`（动态学习大纲，基于 ZPD 划分，每次 loop 结束后更新）
  - `human-learning/judgment.md`（human SENSE 过程中的关键判断，原话保留）
- AI 主导推荐，human 做最终选择
- Push Right + Brief 执行哲学：AI 做最大量工作，human 只在价值性决策点介入
- 跨会话推进，每个 loop 有独立子目录，存放搜索结果等中间产物
- learning agent 在 learning 阶段将 domain 事实同步到 draft

### 差距

- draft 机制不存在
- human 判断从未被单独存储
- ZPD 个性化判断未实现
- 无概念地图嵌套结构
- 无多 loop 调度机制

### 边界

- draft = 预测/学习轨迹（loop 阶段产出，事前）
- domain = 实验后事实记录（learning 阶段产出，事后）
- 时间维度严格区分，两者不覆盖，learning agent 负责事实同步方向（domain → draft）

---

## Perspective（假设视角）

### 核心视角

学习系统的核心状态是学习轨迹（而非知识点覆盖），行为信号驱动 ZPD 迭代，Push Right + Brief 是执行哲学——三者构成可自我修正的闭环。

### 三个核心假设

1. **信息假设**：思考 = 工作记忆中有效信息的排列组合，概念涌现新概念。AI 基于学习轨迹 + 概念地图判断什么是有效信息，提供给 human 激发原创性思考
2. **能力假设**：Push Right 成立的前提是 AI 有足够的事实调研能力；调研能力失效则 Brief 残缺，判断质量下降
3. **迭代假设**：ZPD 初期判断可能不准，但通过行为信号（追问通过率、原创想法数量、认可频率）持续反馈，系统逐步逼近真实 ZPD；loop 结束时引导 human 显式评价加速收敛

### 融贯性检验

- 自洽 ✅：行为信号是客观的，不依赖主观自我报告
- 他洽 ✅：与 Mastery Learning（80-90% 通过率）、ZPD 动态评估、强化学习理论一致
- 续洽 ✅：可预测「ZPD 收敛」，加速方法是 loop 结束时引导 human 显式评价
- 开放风险：信息过载临界点待观察，暂由系统迭代处理

---

## Judgment（矛盾判断）

### 系统核心增强循环

概念地图越完善 → 学习轨迹越丰富 → 有效信息越多 → 原创性判断越多 → 有效反馈越多 → 更完善的概念地图与行为轨迹 → 对项目的控制力越来越强

### 冷启动解决方案

AI 基于广泛事实搜索给出初始概念地图推荐，human 做最终选择

### Push Right 边界

价值排序由 human 决定，事实选择由 AI 完成；当 Brief 从「这是你需要的信息」变成「这是你应该选的答案」，Push Right 就越界了

### 主要矛盾

human 的认知/判断过程没有被捕获进信息流（learning agent 调研能力不足只是症状）

### 矛盾分析

「human 判断的缺失」和「系统进化需要判断信号」之间的张力——系统无法通过 ZPD 迭代优化，因为关键输入信号从未被记录；行为输出是原始数据（噪声大），判断记录是经 human 验证的压缩信号（跨会话可直接使用），两者不可替代

### 选择的控制手段

- think agent 实时捕获 human 关键判断 → `draft/human-learning/judgment.md`
- express agent 在 loop 结束时 review，删除无效/混乱条目
- 判断闭环：`SENSE → judgment.md → doing → learning → domain → dream → 更好 ZPD → 更好 SENSE`

### 排除的选项

- human 手动填写（依赖自律，容易丢失）
- 纯 express 回溯（上下文压缩导致遗漏，且判断尚未回收）

---

## Reverse（逆转）

未触发——常规控制手段已有效，无需逆转思维。

---

## Critique（批判）

### 核心假设清单

| 假设 | 如果不成立，结论会…… |
|------|---------------------|
| AI 能准确区分价值性 vs 事实性决策 | Push Right 越界，human 的判断空间被挤压 |
| 行为信号能有效逼近 ZPD | 迭代方向发散，系统越来越偏离 human 真实能力边界 |
| judgment.md 的判断记录质量足够高 | 强化学习信号失真，优化方向错误 |
| human 愿意在 loop 结束时给出显式评价 | ZPD 收敛加速机制失效，退化为纯隐式反馈 |

### 良质确认

用户确认：这个升级方案让属于 human 的判断力被放大了。核心哲学——AI 只能完成 human 做对关键判断的任务——与做 rick 的内在驱动力合一：不是 AI 替代人，而是人的判断力被放大、被留存、被传递下去。

### 下一步行动

- [ ] 实现 `.rick/draft/` 目录结构（concepts/ + progress.md + human-learning/judgment.md）
- [ ] 升级 human-loop 模板：think agent 增加实时捕获判断的机制
- [ ] 升级 express 模板：增加 judgment.md review + 清洗步骤
- [ ] 升级 learning 阶段：learning agent 将 domain 事实同步到 draft
- [ ] 设计 loops.md 结构化字段（做什么/难度感受/前置依赖/掌握程度）
- [ ] loop 结束时增加 ZPD 显式评价引导
- [ ] 待验证：信息过载临界点的处理方式
