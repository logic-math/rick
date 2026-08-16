# skill:zero-retry-task-design（零重试任务设计）

## 触发场景

plan 阶段分解需求为多个 task.md 时使用，目标是让每个 task 在 doing 阶段一次性完成，无需重试。

## 预期效果

- doing 阶段每个 task 的报错次数接近 0
- 测试脚本一次运行通过
- runtime trace 中工具调用次数 < 30（简洁执行路径）

## 核心内容

### 五要素检查表（每个 task.md 必须满足）

| 要素 | 说明 | 检查方法 |
|------|------|--------|
| 1. 合理粒度 | 5-10 分钟可完成 | 描述不超过 3-4 个关键结果 |
| 2. 清晰目标 | 目标/关键结果/测试方法三段式 | task.md 有三个明确章节 |
| 3. 完整上下文 | 通过 DAG 依赖关系提供前序信息 | 依赖任务已列出 |
| 4. 明确测试标准 | 可量化、可自动化验证 | 无"质量高"等主观标准 |
| 5. 无歧义路径 | 使用绝对路径或明确 `.rick/` 前缀 | 没有裸 `wiki/` 这样的相对路径 |

### task.md 模板

```markdown
# 依赖关系
task1, task2

# 任务目标
在 `/Users/.../rick/.rick/loops/` 目录下创建 X。

# 关键结果
1. 文件 `/Users/.../X.md` 存在
2. 文件包含三个必要章节（触发场景/预期效果/核心内容）
3. `go build ./...` 通过

# 测试方法
```python
# 运行: python3 .rick/jobs/job_N/doing/tests/taskN.py
import os, json, sys
project_root = ...  # 6 次 dirname
assert os.path.exists(os.path.join(project_root, ".rick/loops/X.md"))
print(json.dumps({"pass": True, "errors": []}))
```
```

### 常见陷阱

- ❌ **路径歧义**：`wiki/modules/` 写法不明确起点 → ✅ 用绝对路径
- ❌ **粒度过大**：一个 task 改 5 个文件 → ✅ 拆分为多个单职责 task
- ❌ **主观测试标准**："内容详实" → ✅ 改为"包含 5 个以上示例"
- ❌ **工具接口臆测**：测试调用 `--command` 但工具没有此参数 → ✅ 先 `tool --help`

### 并行化识别

无依赖关系的 task 可并行（tasks.json 中 `dependencies: []` 相同）：
- 创建不同文件的 task 天然可并行
- 修改同一文件的 task 必须串行

参考 [dag_task_decomposition_skill](../dag_task_decomposition_skill/skill.md) 做 DAG 规划。
