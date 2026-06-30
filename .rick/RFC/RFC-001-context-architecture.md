# RFC-001：rick 上下文架构重设计

> **文档类型：** SENSE 思考记录 / RFC 草案
> **创建时间：** 2026-06-28
> **适用场景：** 作为 rick plan 的输入，供 plan agent 细化任务并分解，交给 doing 执行，learning 沉淀

---

## 1. 澄清问题（Subject）

### 现状

- 当前架构：`SPEC.md → wiki → tools` 三层
- skills 没有将上下文内聚，没有模块化
- 当前 skills 默认 agent 思维方式 = 人类思维方式（这个假设是错的）
- 当前 skills 本质上是"给人看的"，而非"给 agent 看的"

### 期望

- 重构为 `SPEC.md → loops → skills` 三级结构
- Skills 以 agent 为中心（agent 用实验性方式解决问题：观察-假设-实验-分析-结论）
- 给 agent 的任务必须是确定性任务（主要矛盾已定位，设计决策全部落实）
- Agent 核心能力 = debug（在确定性任务下达成目标）
- Rick 核心定位 = 治理 LLM 输入复杂性，不过度干预 LLM 行为
- 人类角色 = 决策 + 引导；agent 角色 = 执行 + debug

### 差距

- 缺少 loops 抽象层（显式终止条件 + pipeline）
- Skills 未模块化 → 无法对外部 skills 学习内化
- agent-centric skill 格式尚待探索（开放性研究课题）
- 缺 learning 阶段的 loops/skills 格式模板

### Rick 设计哲学

- human-loop = 解决可行性问题（产出 RFC）
- plan = 解决确定性问题（RFC → 构建设计树 → 任务分解）
- doing = debug 执行
- learning = 候选产出，人类确认后写入
- human 与 agent 分层设计是 rick 的核心哲学

**技能学习的含义**：直接拿已有 skills 整理使用、对外部信息内化、对自己工作经验总结，三者都算学习。

**plan 构建设计树终止条件（已显式化）**：
1. 设计决策落实
2. 代码接口/实现确定
3. 文件结构确定
4. 模块交互与接口逻辑确认清晰

---

## 2. 假设视角（Perspective）

### 核心假设

**Loop 作为辩证逆转**

> 问题：agent 执行本身具有不确定性，如何获得确定性的结果？
>
> 逆转：不消除不确定性，而是给 agent 一个结构化的迭代空间——用过程的受控不确定，换取结果的最终确定。

### 假设依据

**概念地图**：
- debug = 治理 LLM 输入复杂性的手段（获取真实世界反馈 → 降低输入复杂性）
- human-loop → plan → doing → learning：可行性 → 确定性 → 执行 → 沉淀
- 当前 wiki + tools 承接了 skills 职责，无 loops 抽象

**Loop 三要素（loop.md 核心字段）**：
1. 运行环境：执行前解决软件/配置依赖，明确运行环境前置条件
2. 反馈信息：终止条件 + bug 第一现场判断标准（推进 or 停下修 bug 的判断依据）
3. 工作流：pipeline（步骤 + 调用的 skills/工具 + 成功描述 + debug 观察路径）

### 融贯性检验

- 自洽：A（收窄决策空间）是设计动机，C（允许迭代失败）是运行结果，同一逆转的两面 ✅
- 他洽：有真实观察——skills 过多、工作流过长时，上下文遗忘导致低级错误反复，效率崩溃 ✅
- 续洽：直接指导 loop.md 三要素字段设计 ✅

---

## 3. 矛盾判断（Judgment）

### 主要矛盾

**系统循环图**：
- 正反馈：skills 积累 → loop 覆盖场景更广 → doing 成功率更高 → learning 产出更多 skills
- 天花板：规模增长 → skills 之间信息矛盾、上下文混乱 → 打破正反馈
- 负反馈触发：debug skills 完善程度决定负反馈是否有效

**两条关键链路（均为全局性、根本性、决定性）**：

链路 1（doing 阶段）：debug skills 是否被严格遵守、是否执行有效 → 有效收集真实信息反馈的关键

链路 2（learning+dream 阶段）：SPEC.md → loops → skills 的修复（负反馈链路）→ 对抗熵增的关键

### 控制手段

**链路1**：方案A（doing prompt 强制注入 debug skills）+ meta loop

**链路2**：E+F 组合（learning 产出候选 + dream 识别高频失效），可选人类介入，不强制

**排除的选项**：
- C（trust LLM，不注入）：agent 在没有明确输入约束时会走捷径、跳过 debug，被排除
- G（agent 自动更新 loop.md）：错误假设叠加风险，被排除

### Doing 完备状态机（Meta Loop）

**三层循环**：
- Micro loop：TDD red-green-refactor（每个实现步骤内）
- Step loop：TDD_IMPL → STEP_CHECK → DEBUG_STEP → TDD_IMPL
- Task loop：TASK_VALIDATE → DEBUG_TASK → TDD_IMPL → TASK_VALIDATE

**状态机**：

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

---

## 4. 解决方案

### 架构设计

重构目标架构：`SPEC.md → loops → skills` 三级结构

- SPEC.md：系统级设计哲学与全局约束
- loops：场景级工作流（含终止条件 + pipeline）
- skills：原子级能力单元（agent-centric 格式，模块化）

### loop.md 格式规范

三要素字段：

1. **运行环境**：执行前解决软件/配置依赖，明确运行环境前置条件
2. **反馈信息**：终止条件 + bug 第一现场判断标准（推进 or 停下修 bug 的判断依据）
3. **工作流**：pipeline（步骤 + 调用的 skills/工具 + 成功描述 + debug 观察路径）

### 执行者分工

- **gen_test agent**：生成验收脚本
- **coding agent**：TDD 实现 + step 检查 + 验收执行
- **diagnosing-bugs skill**（引自 builder/skills）：DEBUG_STEP 和 DEBUG_TASK 两处触发

**builder/skills 的 diagnosing-bugs 方法（已调研确认采用）**：
- Phase 1（最核心）：构建反馈回路（red-capable + deterministic + 秒级 + agent-runnable）
- Phase 2：复现 + 最小化
- Phase 3：生成 3-5 个有排名的可证伪假设
- Phase 4：插桩观察（调试器优先，禁止"记录所有然后 grep"）
- Phase 5：修复 + 回归测试
- Phase 6：清理 + 事后分析

---

## 5. 过程批判（Critique）

### 核心假设清单

1. agent 执行具有不确定性，结构化迭代空间可以换取结果确定性
2. debug skills 被严格遵守是有效反馈的前提
3. learning + dream 负反馈链路可以对抗系统熵增
4. plan 阶段设计树终止条件可以保证任务确定性

### 已知漏洞与处理

- **plan 确定性检测**：构建设计树保证，暂无专门检测节点
- **learning 自举问题**：人类控制 rick 自迭代修复 learning 的 loop 定义
- **验收脚本有效性**：plan 阶段设计树每个 step 有人类决策的可测试、可证伪成功描述

**真正的瓶颈（已识别）**：plan 阶段人类判断力的上限——构建设计树时，是否真的把任务分解到 agent 可确定性执行的粒度，依赖人类经验，是系统隐形天花板。

**应对方向**：人类不断优化 build design tree 阶段，完善设计树终止条件的判断依据。

### 良质确认

- doing 状态机是在不断使用和修改 rick 的过程中自然浮现的，不是妥协
- 人机协作哲学：agent 负责执行，human 负责判断。判断决定执行的上限，二者相辅相成。这是人类在人工智能时代的存在方法。

### 下一步行动

1. 将本 RFC 作为 `rick plan` 输入，构建设计树
2. 设计 loop.md 格式规范（三要素字段的具体模板）
3. 探索 agent-centric skill 格式（开放性研究课题）
4. 补充 learning 阶段的 loops/skills 格式模板
5. 将 diagnosing-bugs skill 从 builder/skills 引入并内化
