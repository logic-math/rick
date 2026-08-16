# Skills 目录规范

Skill 是一个原子级能力单元（静态上下文），agent 在遇到触发条件时按需加载并执行一次。

## 目录结构

每个 skill 必须以**目录**形式存在：

```
.rick/skills/
├── {name}_skill/
│   ├── skill.md        # 必须：触发场景 / 预期效果 / 核心内容
│   ├── helper.py       # 可选：Python 辅助脚本
│   └── ...             # 可选：其他支撑文件
└── deprecated/         # 已淘汰（连续3次dream未被引用）
```

**命名规则**：目录名以 `_skill` 结尾，如 `mark_task_success_skill/`。

## 当前 Skills 索引

| Skill | 触发场景 |
|-------|---------|
| test_script_practices_skill | 编写/调试 .rick/jobs/job_N/doing/tests/taskN.py |
| zero_retry_task_design_skill | plan 阶段分解任务，降低 doing 重试率 |
| dag_task_decomposition_skill | 设计复杂 task DAG，识别并行化机会 |
| check_mechanism_skill | learning_check/dream_check 失败，理解修复方法 |
| failure_feedback_skill | 理解/调整 doing 重试时失败信息传递机制 |
| mark_task_success_skill | rick-gates 门禁报 missing commit_hash 时修复 |
| global_ref_sync_skill | 修改核心名称/变量前先全局 grep 找引用 |
| verify_go_changes_skill | 修改 Go 源文件后验证编译+测试通过 |
| template_injection_skill | 向 plan/easy 模板注入新 skill 或修改变量 |

## skill.md 格式（三段式）

```markdown
# skill:{name}（描述）

## 触发场景
什么情况下使用，要具体（"当X时" 不是 "调试时"）

## 预期效果
可量化的结果（"减少重试次数" 不够，要 "一次运行通过"）

## 核心内容
可直接执行的步骤/命令/决策树，含具体示例
```

## 淘汰标准

连续 3 次 dream 未被引用的 skill → 移至 `deprecated/`。
