# RFC-001: 将上下文管理重构为信息网络流

**状态**: 草案  
**作者**: 人类（通过 SENSE Human Loop 思考产出）  
**日期**: 2026-04-06  
**AI 不可修改**

---

## 一、问题定义（Subject）

### 现状
rick 每次新 job 都重复解决旧问题，token 消耗大，确定性差，多个 job 之间不一致。具体表现：
- learning 沉淀的 skills 没有被后续 job 使用
- 多次迭代后，rick 对项目的控制能力没有提升
- 上下文在多次多人变更中无法保持一致性

### 期望
上下文在多次多人变更中保持一致性和有效性，逼近 AI 模型能力上限。

### 差距
没有机制将旧问题的解法沉淀为可复用的工具，导致知识无法跨 job 传递。

---

## 二、核心视角（Perspective）

### 理论基础
上下文管理本质上是一个**信息网络流**问题：

- **节点**：agent（生产和消费信息）
- **边**：信息传递的通道（文件、prompt）
- **容量**：信息的确定性与有效性
- **最小割**：系统中信息流通的最大瓶颈

根据 Max-Flow Min-Cut 定理：系统的上限由最薄弱的环节决定。优化系统 = 找瓶颈 + 消除瓶颈。

### 最小信息单元：DoD
上下文以 **DoD（完成定义）** 为最小信息传递单元，由三要素构成：

| 要素 | 内容 | 文件 |
|------|------|------|
| 目标（OKR） | 这个 job 要达成什么 | OKR.md |
| 约束（SPEC） | 全局规范，所有 agent 必须遵循 | SPEC.md |
| 标准（task） | 关键结果 + 测试方法 | task.md |

### 闭环结构
```
Milestone / RFC
      ↓ 输入
    plan（生成 OKR / SPEC / task）
      ↓
    doing（执行，产出 code / debug.md）
      ↓
    learning（压缩，产出 wiki / skills / tools / SPEC 更新）
      ↓
    人类阅读 wiki
      ↓
    builder-loop（思考，产出 RFC / Milestone）
      ↑ 闭环
```

---

## 三、主要矛盾（Judgment）

### 主要矛盾
**learning 层断裂**是当前最根本的问题。它导致：
- job 产出的知识无法融入全局上下文
- 后续 job 无法继承已有经验
- skills 无法被 plan 和 doing 感知和使用

### 三个并行修复点

**1. 重构 learning 的输入**
- 当前：读取 git 历史（低效，噪声多）
- 应该：直接读取 OKR / task.md / debug.md
- 原则：渐进式披露，agent 自行判断是否需要进一步读取源码

**2. 重构 skills 格式**
- skills 是对 tools（确定性 Python 脚本）的组合运用方法
- 必须有 index 文件，描述每个 skill 的触发点
- index 在 plan 和 doing 阶段注入 prompt

**3. 重构 plan 和 doing 的 prompt**
- plan 和 doing 必须感知 skills index
- prompt 中强制要求 agent 优先考虑用已有 skills 解决问题

---

## 四、上下文层次结构

### 四层上下文模型

#### 层 1：doing 级上下文（job 内 agent 间同步）

| 文件 | 作用 | 生命周期 |
|------|------|----------|
| OKR.md | 本 job 的目标，仅描述当前 job 所需目标信息 | plan 创建，job 结束后归档 |
| SPEC.md | 全局唯一规范，验证环境控制方法 | 全局共享，learning 可更新 |
| task.md | 单个 agent 的完成标准（关键结果 + 测试方法） | plan 创建，doing 消费 |
| debug.md | job 内多个 coding agent 的信息共享文件 | doing 期间持续写入，learning 消费 |

**注意**：OKR 应为 job 级而非全局级。agent 只需知道完成当前 job 所需的目标信息，不需要了解整个项目的全局进展。

#### 层 2：job 级上下文（全局项目知识）

learning agent 将 job 产出压缩融合到全局上下文：

| 文件/目录 | 作用 | 修改者 |
|-----------|------|--------|
| SPEC.md | 项目规范，agent 控制方法 | learning agent |
| skills/ | 技能库，tools 的组合运用方法 + index | learning agent |
| tools/ | 确定性 Python 工具，存项目根目录 | learning agent / coding agent |
| wiki/ | 给人类阅读的事实性知识文档 | learning agent |
| RFC/ | 项目规划与讨论，关乎未来发展 | **人类，AI 不可修改** |
| Milestone.md | 项目最终愿景与实现路径 | **人类，AI 不可修改** |
| README.md | 项目简介、安装、快速入门 | learning agent |

#### 层 3：builder 级上下文（个人战略思考）
- builder-loop：人类独立思考，产出 RFC 与 Milestone
- 人类通过阅读 wiki 理解系统现状
- **后续构建**

#### 层 4：builders 级上下文（多人协作对齐）
- Milestone：构建者们讨论出的确定性规划结论
- issue/：记录宽泛讨论问题，暂无结论
- **后续构建**

---

## 五、skills 设计规范

### tools（确定性工具）
- 位置：项目根目录 `tools/` 目录
- 格式：Python 脚本
- 规范：脚本顶部注释必须包含工具描述（供 rick 扫描注入 prompt）
- 特性：确定性、可反复调用、可被 agent 直接执行

### skills（组合技能）
- 位置：`.rick/skills/` 目录
- 格式：Markdown 文件
- 内容：描述如何组合使用 tools 完成复杂任务
- **必须有 `index.md`**：列出所有 skill 的名称和触发场景

### 注入机制
```
rick plan / rick doing
    ↓ 扫描
tools/ 目录（读取每个 .py 文件的顶部注释）
.rick/skills/index.md
    ↓ 注入
coding agent 的 prompt
    ↓ 强制要求
优先使用已有 skills 和 tools 解决问题
```

---

## 六、learning agent 重构规范

### 输入（渐进式披露原则）
1. 读取 `OKR.md`、`task.md`、`debug.md`（必读）
2. 根据上述内容的索引，自行判断是否需要读取具体源码（按需读取）
3. **不读取 git 历史**（噪声过多，效率低）

### 输出（四个子任务，可用 subagent 并行生成）
1. **wiki 更新**：新增或更新给人类阅读的知识文档
2. **skills 更新**：沉淀新的组合技能，更新 index.md
3. **tools 生成**：将确定性操作封装为 Python 脚本
4. **SPEC 更新**：更新项目规范

### 触发方式
- 当前：手动触发（`rick learning job_n`）
- 后续：CI/CD 定期自动触发（后续扩展）

---

## 七、关键假设清单

| 假设 | 验证状态 |
|------|----------|
| DoD 是有效的最小信息单元 | ✓ 已验证 |
| learning 层断裂是主要矛盾 | ✓ 确认 |
| skills 沉淀后通过 prompt 强制 agent 优先使用 | ✓ 确认 |
| plan 输入灵活，不限制内容格式 | ✓ 确认 |
| builder/builders 层后续构建，当前不阻塞 | ✓ 确认 |
| learning 手动触发，CI/CD 定期触发后续扩展 | ✓ 确认 |

---

## 八、实施优先级

### 现在（核心闭环）
1. 重构 learning agent 的输入逻辑
2. 建立 skills/index.md 格式规范
3. 重构 plan 和 doing 的 prompt（注入 tools 描述 + skills index）
4. OKR 改为 job 级而非全局级

### 后续
5. builder-loop 命令（人类思考引导）
6. Milestone 机制
7. issue 快速记录方法
8. CI/CD 定期触发 learning

---

## 九、rick 的终极目标

> **使人类成长，使 AI 交付成果。**

rick 是人与 AI 之间的上下文对齐框架。它的本质是：在多次迭代、多人协作的过程中，保持事实信息的有效性，让 AI 始终在正确的信息基础上工作，让人类始终在清晰的认知基础上决策。
