# Chapter 15: Fixing the Defect — 事实总结

## 核心概念

- **Defect location（缺陷位置）**：包含错误代码的具体源文件和行号
- **Infected value（感染值）**：在某执行点被感染（取了错误值）的变量
- **Origin（来源）**：感染值的上一个赋值来源（数据依赖链的前驱）
- **Workaround（变通方案）**：不修改 defect，而是防止 failure 出现的临时方案
- **Regression test（回归测试）**：修复后运行，确保修复不引入新问题
- **Debugging into existence（调试出存在）**：通过反复修改直到测试通过，但不理解修复原因——书中明确命名为反模式

## 主要内容

### 15.1 定位 Defect 的循环

**从感染值出发，逐步向源头追溯**：

```
while (还没找到 defect):
    在当前感染点找到感染变量 v
    找到 v 的最后一次赋值（origin）
    if origin 是 sane（正常的）:
        检查赋值语句——这里可能是 defect
        如果赋值语句本身错误 → 找到 defect
        如果赋值语句正确 → 继续追溯 origin 的 origin
    if origin 是 infected（感染的）:
        将追踪焦点移到该 origin
        继续循环
```

**关键原则**：每次追溯必须找到"所有 origins 均为 sane 但 v 却是 infected"的点——这就是 infection 的起始点，defect 在这附近。

### 15.2 五类优先聚焦策略

按优先级从高到低：

1. **Infections（感染点）**：已知感染的变量值，直接向其追溯来源（最精确）
2. **Causes（因果）**：Chapter 12-14 找到的 failure cause，直接指向问题区域
3. **Anomalies（异常）**：Chapter 11 检测到的异常行为，可能是 infection 的信号
4. **Code smells（代码异味）**：Chapter 7 静态分析发现的可疑模式
5. **Dependences（依赖）**：Backward slice（后向切片），最宽泛，最后使用

**原则**：越精确的信息优先使用，粗糙信息作为兜底。

### 15.3 双重验证要求

找到候选 defect 位置后，必须同时满足两个条件才能确认：

1. **条件一（感染）**：该位置的代码确实产生了感染值
2. **条件二（因果）**：该感染是 failure 的实际原因（不只是相关）

**"Debugging into existence" 反模式（书中明确命名）**：
- 定义：修改代码直到测试通过，但不理解为何有效
- 危害：可能掩盖真正的 defect，引入新 bug
- 书中明确说明这是应该避免的反模式

**验证因果的方法（见 Chapter 12）**：
- 替换感染值为正确值（注入 sane 值）
- 如果 failure 消失，说明该 infection 是 failure 的 actual cause
- 如果 failure 不消失，继续追溯

### 15.4 修复后四步检查

1. **Failure 消失**：运行原始失败测试用例，确认 failure 不再出现
2. **无新问题**：
   - Peer review（同行代码审查）：人工检查修复的代码是否引入新问题
   - Regression testing（回归测试）：运行完整测试套件
3. **无同类错误**：在代码库中搜索相同的错误模式（如相同的 off-by-one 逻辑），修复所有实例
4. **流程完整**：
   - Issue tracker 中更新问题状态（RESOLVED/FIXED）
   - 提交修复 commit，引用 issue 号
   - 添加回归测试到测试套件

### 15.5 Workaround（变通方案）

**定义**：保留 defect 在代码中，但阻止其产生 failure。

**适用场景**：
- Defect 在第三方库中，无法修改
- 修复代价太高，或修复风险太大
- 紧急情况下的临时处置

**要求**：
- Issue tracker 中 defect 的 issue 必须保持**开放状态**（不能关闭）
- 明确标记为 workaround，注明待修复的 defect 位置
- 不能将 workaround 误认为真正的修复

## 修复策略与步骤

**完整修复流程**（书中 Section 15.4）：

| 步骤 | 操作 | 工具 |
|------|------|------|
| 1. 追溯感染链 | 从失败点向源头追溯 | GDB / 动态切片 |
| 2. 找到 infection 起始点 | "所有 origins sane，但 v infected" | 手动 + 工具辅助 |
| 3. 验证双重条件 | 感染 + 因果 | 注入 sane 值实验 |
| 4. 实施修复 | 修改代码 | 编辑器 |
| 5. 四步检查 | failure消失/无新问题/无同类/流程完整 | 测试套件 + 代码审查 |
| 6. 若无法修复 | 实施 workaround，保持 issue 开放 | Issue tracker |

## 对四个追问的回答

**1. 「实验报告」vs「解决问题」**
调试记录（logbook）服务于问题解决，两者不对立。Section 15.4 的"流程完整"步骤明确要求更新 issue tracker 和提交 commit——这些记录是修复流程的一部分，不是额外负担。

**2. 「探针不影响系统状态」**
Chapter 15 未直接讨论 Heisenbug。但 Section 15.3 的"注入 sane 值"验证方式是主动改变程序状态——这是"实验"的一部分（见 Chapter 12），而非观察。

**3. 「学习、设计、验证正交」**
学习（追溯感染链）、设计（设计注入实验）、验证（执行实验确认因果）三者在书中是顺序依赖的，不是正交独立的。但"一次只修复一个 defect"（Section 15.4 隐含原则）保持了验证的独立性。

**4. 「理解状态转移是否足以覆盖所有bug」**
Chapter 14.7 明确论证：自动确定 defect 理论上不可能，因为需要完整规约，而完整规约等价于已有正确程序。**finding defect 永远是手动活动**。状态转移的理解是必要但不充分条件。

## 关键引用

- "Debugging into existence: changing the code until the tests pass, without understanding why. This is an anti-pattern." （Section 15.3）
- "For a fix to be complete, four conditions must be met: failure is gone, no new failures, no similar bugs, process complete." （Section 15.4）
- "A workaround keeps the defect but prevents the failure. The issue must remain open." （Section 15.5）
- "The fix should be as local as possible: change only what is necessary."

## 与AI Agent调试的关联

- **感染链追溯**：rick 的 debug.md 应记录"从失败症状向上追溯的推理过程"，对应 Section 15.1 的循环
- **"Debugging into existence" 警告**：AI Agent 可能通过随机修改代码直到测试通过，对应此反模式——rick 应要求 AI 在修改前给出因果解释
- **双重验证**：rick 的测试脚本只验证"failure 消失"（条件一），还需要验证"没有引入新问题"（条件二）——对应 Section 15.4 的回归测试
- **Workaround 标记**：当 AI Agent 采用 workaround 而非真正修复时，rick 应在 debug.md 中明确标记，保持问题跟踪
