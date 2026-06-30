# SENSE 会话进度存档

> 日期：2026-06-28
> 主题：rick 上下文架构重设计 + RFC 草案

---

## 复杂度判断

**Level 3（复杂）**，完整 SENSE 五步。

---

## ✅ Subject 阶段（已完成）

### 现状（修正后）

- 当前架构：`SPEC.md → wiki → tools` 三层
- 弊端：skills 没有将上下文内聚，没有模块化
- 根因：当前 skills 默认 agent 思维方式 = 人类思维方式（这个假设是错的）
- 当前 skills 本质上是"给人看的"，而非"给 agent 看的"

### 期望

- 重构为 `SPEC.md → loops → skills`
- Skills 以 agent 为中心（agent 用实验性方式解决问题：观察-假设-实验-分析-结论）
- 给 agent 的任务必须是**确定性任务**（主要矛盾已定位，设计决策全部落实）
- Agent 核心能力 = **debug**（在确定性任务下达成目标）
- Rick 核心定位 = **治理 LLM 输入复杂性**，不过度干预 LLM 行为
- 人类角色 = 决策 + 引导；agent 角色 = 执行 + debug

### 差距

- 缺少 loops 抽象层（显式终止条件 + pipeline）
- Skills 未模块化 → 无法对外部 skills 学习内化
- agent-centric skill 格式尚待探索（开放性研究课题）
- 缺 learning 阶段的 loops/skills 格式模板

### 高风险假设澄清

| 假设 | 澄清结果 |
|------|----------|
| 循环对 agent 更好 | 有实验支撑：上下文污染 → 迭代不可避免 |
| learning 谁做抽象决策 | 人类决策，learning 提供候选 |
| SPEC.md 加载机制 | 直接注入，trust LLM 自行导航 |
| 按需引用 loops/skills | agent 自行从 SPEC.md 选 loop，可创新 |
| human vs agent 写权限 | human 决策引导；learning/dream 写 SPEC.md，人类确认 |

### Rick 设计哲学（关键）

- **human-loop** = 解决**可行性**问题（战略思考，产出 RFC）
- **plan** = 解决**确定性**问题（RFC → 设计树澄清决策 → 任务分解）
- **doing** = debug 执行
- **learning** = 候选产出，人类确认后写入
- **human 与 agent 分层设计**是 rick 的核心哲学

---

## ✅ Perspective 阶段（已完成）

### 概念地图

- debug = 治理 LLM 输入复杂性的手段（获取真实世界反馈 → 降低输入复杂性）
- human-loop → plan → doing → learning：可行性 → 确定性 → 执行 → 沉淀
- 当前 `wiki + tools` 承接了 skills 职责，无 loops 抽象

### 核心解释模型：Loop 作为辩证逆转

> **问题**：agent 执行本身具有不确定性，如何获得确定性的结果？
>
> **逆转**：不消除不确定性，而是给 agent 一个**结构化的迭代空间**——用过程的受控不确定，换取结果的最终确定。

### Loop 三要素（loop.md 核心字段）

1. **运行环境**：执行前解决软件/配置依赖，明确运行环境前置条件
2. **反馈信息**：终止条件 + bug 第一现场判断标准（推进 or 停下修 bug 的判断依据）
3. **工作流**：pipeline（步骤 + 调用的 skills/工具 + 成功描述 + debug 观察路径）

### 融贯性验证

- **自洽** ✅：A（收窄决策空间）是设计动机，C（允许迭代失败）是运行结果，两者是同一逆转的两面
- **他洽** ✅：有真实观察——skills 过多、工作流过长时，上下文遗忘导致低级错误反复，效率崩溃
- **续洽** ✅：直接指导 loop.md 三要素字段设计

---

## 🔄 Judgment 阶段（进行中）

### Step1 — 循环图

**正反馈环路**：skills 积累 → loop 覆盖场景更广 → doing 成功率更高 → learning 产出更多 skills
**天花板**：规模增长 → skills 之间信息矛盾、上下文混乱 → 打破正反馈

**负反馈触发**：debug skills 完善程度决定负反馈是否有效
- debug skills 不完善 → human 无法获得有效信号 → 无法有效介入调节

**核心张力**：plan 精细度 vs. doing 自由度
- 平衡点：设计树叶子节点的终止条件

### Step2 — 主要矛盾（已确认）

**两条关键链路，均为全局性、根本性、决定性：**

**链路 1（doing 阶段）**：
- debug skills 是否被严格遵守、是否执行有效
- 是有效收集真实信息反馈的关键

**链路 2（learning+dream 阶段）**：
- SPEC.md → loops → skills 的修复（负反馈链路）
- 针对 loops 修复运行环境、反馈信息、工作流
- 是对抗熵增的关键负反馈链路

### Step3 — 控制手段选择（进行中）

**链路 1 候选控制手段（待决策）：**
- A：在 doing prompt 中强制注入 debug skills 模板
- B：在 loop.md 的"反馈信息"字段中内嵌 debug 观察路径
- C：靠 agent 自己判断（trust LLM）
- D：doing 结束时强制 human review debug 日志

用户当前：调研 builder/skills 的 debug skills 方法作为信息黑箱参考（learn subagent 进行中）

**链路 2 候选控制手段（已倾向）：**
- learning + dream 保证（E+F 组合）
- learning 提供选项，不想介入则按推荐直接执行，人类可选择是否介入
- G（agent 自动更新）已排除——防止错误假设叠加

---

## ⚡ 关键洞见：Doing 的完备状态机

**三层循环结构**：
- **Micro loop**：TDD red-green-refactor（每个实现步骤内）
- **Step loop**：TDD_IMPL → STEP_CHECK → DEBUG_STEP → TDD_IMPL
- **Task loop**：TASK_VALIDATE → DEBUG_TASK → TDD_IMPL → TASK_VALIDATE

**完备状态机**：

```
[TASK_START]
     │
     ▼
[GEN_TEST]  ──── gen_test agent 生成验收脚本
     │
     ▼
[TDD_IMPL]  ◄──────────────────────────────┐
     │  RED → GREEN → REFACTOR             │
     ▼                                     │
[STEP_CHECK]                               │
     ├─── 符合预期 ──► 下一个 step ─────── ┘
     │
     └─── 出现第一现场
               │
               ▼
          [DEBUG_STEP]
               └─── bug 修复 ──────────────┘（回 TDD_IMPL）

     │（所有成果实现完毕）
     ▼
[TASK_VALIDATE]  ◄────────────────────────┐
     │                                    │
     └─── FAIL                            │
               │                          │
               ▼                          │
          [DEBUG_TASK]                    │
               └─── bug 修复 ────────────┘（回 TDD_IMPL）

     PASS ──► [TASK_DONE]  ← 唯一出口
```

**loop.md 三要素的映射**：
- 运行环境 → GEN_TEST 前的环境配置
- 反馈信息 → STEP_CHECK 结果 + TASK_VALIDATE 验收脚本（终止条件）
- 工作流 → TDD_IMPL → STEP_CHECK → TASK_VALIDATE（pipeline）

**执行者分工**：
- gen_test agent：生成验收脚本
- coding agent：TDD 实现 + step 检查 + 验收执行
- diagnosing-bugs skill：DEBUG_STEP 和 DEBUG_TASK 两处触发

---

## ✅ Judgment 阶段（已完成）

控制手段最终选定：
- 链路1：方案A（doing prompt 强制注入 debug skills）+ meta loop（TDD预防 + 验收触发 diagnosing-bugs）
- 链路2：E+F 组合（learning 产出候选 + dream 识别高频失效），可选人类介入

## ✅ Critique 阶段（已完成）

**逻辑漏洞澄清**：
- plan 确定性：由构建设计树保证，暂无专门检测机制
- learning 自举：人类控制 rick 自迭代修复
- 验收脚本有效性：plan 阶段设计树每个 step 有人类决策的可测试成功描述

**设计树终止条件（显式化）**：
1. 设计决策落实
2. 代码接口/实现确定
3. 文件结构确定
4. 模块交互与接口逻辑确认清晰

**人机协作哲学（核心）**：
> agent 负责执行，human 负责判断。判断决定执行的上限，二者相辅相成。
> 这是人类在人工智能时代的存在方法。

**良质确认**：doing 状态机是自然浮现的，符合良质 ✅

## 🔄 Express 阶段（进行中）

产出目录：`/Users/sunquan/ai_coding/CODING/rick/.rick/RFC/`

---

## 产出目录

`/Users/sunquan/ai_coding/CODING/rick/.rick/RFC/`
