# Chapter 5: Simplifying Problems — 事实总结

## 核心概念

- **Configuration（配置）**：可以被分解为子集的一组元素（如输入字符串的字符集、代码变更的集合）
- **1-minimal（1-最小）**：移除任意单个元素后测试就通过的配置，即不存在更小的失败配置
- **ddmin 算法**：自动找到 1-minimal 失败配置的算法
- **Delta Debugging**：通过自动二分搜索简化失败测试用例的技术（Zeller, 1999）
- **UNRESOLVED**：测试结果既非通过也非失败的中间状态（如超时、崩溃）
- **Adaptive testing（自适应测试）**：根据已有测试结果动态选择下一个测试的策略

## 主要内容

### 5.1 简化的目的
简化测试用例的两个价值：
1. **定位**：简化后的最小测试用例往往直接指向 defect
2. **效率**：更小的输入意味着更短的调试时间

### 5.2 手动简化
逐步删除输入的非关键部分，验证失败是否仍然存在。问题：手动过程耗时且容易遗漏。

### 5.3–5.4 ddmin 算法
**形式化定义**：
```
ddmin(c_fail):
  假设 test(c_fail) = FAIL
  返回 1-minimal 的 c' ⊆ c_fail 使得 test(c') = FAIL
```

**三种分支（递归）**：
1. 某子集本身失败 → 对该子集递归
2. 某子集的补集失败 → 对补集递归
3. 都不失败 → 增加分割粒度，继续搜索

**Python 实现**（书中 Example 5.4-5.7）：
```python
def ddmin(circumstances):
    # 从 n=2 开始分割
    # 三种情况的递归处理
    # 直到子集数量等于元素数量（1-minimal）
```

**实际效果**：Mozilla HTML 案例，从完整 HTML 文件经 48 次测试简化到单个 `<SELECT>` 标签。

### 5.5 加速技术
- **缓存**：避免重复测试相同配置
- **提前停止**：一旦找到 1-minimal 立即停止
- **语法感知简化**：对树结构（AST）按语法节点而非字符进行简化（HDD，Hierarchical Delta Debugging）
- **隔离差异**：不仅简化"失败输入"，也可简化"两个配置间的差异"

### 5.6 UNRESOLVED 处理
当 `test()` 返回 UNRESOLVED（非 PASS/FAIL）时，可以：
- 忽略（跳过该子集）
- 用 fork 隔离（避免崩溃影响测试进程）

### 5.7–5.11 Concepts/Tools/Further Reading
工具：
- `delta`：通用命令行 delta debugging 工具
- `CReduce`：针对 C 程序的语法感知简化工具
- `Picireny`：基于 ANTLR 语法的 HDD 实现

## 调试方法/技术（重点：Delta Debugging算法步骤）

**ddmin 算法完整步骤**：
1. 设置初始配置 `c = c_fail`，分割数 `n = 2`
2. 将 `c` 分割为 `n` 个子集 `{c₁, c₂, ..., cₙ}`
3. 对每个子集 `cᵢ`：
   - 若 `test(cᵢ) = FAIL`：令 `c = cᵢ`，`n = 2`，跳回步骤2
   - 若 `test(complement(cᵢ)) = FAIL`：令 `c = complement(cᵢ)`，`n = n-1`，跳回步骤2
4. 若所有子集和补集都 PASS：
   - 若 `n < |c|`：令 `n = min(2n, |c|)`，跳回步骤2
   - 若 `n ≥ |c|`：返回 `c`（已是 1-minimal）

**时间复杂度**：最坏 O(n²)，实践中通常接近 O(n log n)

## 对四个追问的回答

**1. 「实验报告」vs「解决问题」**
书中无"调试记录"概念，`test()` 的结果是算法的反馈信号，不是供人阅读的报告。简化本身的目的是产出"更容易调试的输入"，而非报告。两者无明显对立，但本章侧重自动化而非记录。

**2. 「探针不影响系统状态」**
通过 UNRESOLVED 状态和 fork 隔离处理测试崩溃。但书中没有系统讨论 Heisenbug 场景——简化阶段假设 `test()` 的结果是确定性的，非确定性输入会导致 ddmin 结果不可靠。

**3. 「学习、设计、验证正交」**
验证（`test()`函数）是独立封装的；设计（算法参数）在运行前固定；学习内置于自适应测试（根据结果选择下一步）。但**"三者正交"不是书中的显式框架**，ddmin 是一个整合的算法，三者在算法内部耦合。

**4. 「理解状态转移是否足以覆盖所有bug」**
书中承认 1-minimal 不等于 global minimal（配置空间是指数级的，1-minimal 只是局部最优）。对于连续输入（如图像）或强依赖语义的输入，ddmin 不适用。

## 关键引用

- "A simplified test case reports the failure immediately, pinpointing the defect."
- "ddmin returns the 1-minimal failing configuration."
- "1-minimal does not mean minimal." （Section 5.5 明确指出）
- Mozilla 案例："48 tests to isolate `<SELECT>`"

## 与AI Agent调试的关联

- **最小化失败 prompt**：ddmin 思想可用于简化导致 AI Agent 失败的 prompt，找到最小触发条件
- **`test()` 函数定义**：rick 的测试脚本对应 `test()` 函数，其确定性直接影响 ddmin 的有效性
- **UNRESOLVED 处理**：AI Agent 的输出可能是"部分正确"（既非完全成功也非完全失败），需要定义明确的二元判断标准
- **fork 隔离**：rick 每次重试应在隔离环境中执行，避免前次失败状态污染当次执行
