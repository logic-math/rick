# skill:mark-task-success（tasks.json 两阶段提交）

## 触发场景

doing task 代码已提交（有 commit hash）但 rick-gates 门禁（helper.py）报错：
- `missing commit_hash`
- `zombie running`

信号词：`python3 .rick/skills/rick-gates/helper.py <doing_dir>` 输出上述错误。

## 预期效果

- 一次修复通过 rick-gates 门禁
- 不需要反复读取 tasks.json 格式或门禁源码

## 核心内容

### 使用辅助脚本（推荐）

```bash
# 代码已 commit 后，立即执行：
python3 .rick/skills/mark_task_success_skill/mark_task_success.py --job job_N --task taskX
# 工具自动读取 HEAD commit hash，更新 tasks.json status=success + commit_hash

# 提交 tasks.json：
git add .rick/jobs/job_N/doing/tasks.json
git commit -m "chore(taskX): mark taskX success in tasks.json"

# 验证：
python3 .rick/skills/rick-gates/helper.py .rick/jobs/job_N/doing  # 应 exit 0
```

### 手动修复（备用）

```python
import json, subprocess

# 获取 commit hash
hash_result = subprocess.run(["git", "rev-parse", "HEAD"], capture_output=True, text=True)
commit_hash = hash_result.stdout.strip()

# 更新 tasks.json
with open(".rick/jobs/job_N/doing/tasks.json", "r") as f:
    data = json.load(f)

for task in data["tasks"]:
    if task["task_id"] == "taskX":
        task["status"] = "success"
        task["commit_hash"] = commit_hash
        break

with open(".rick/jobs/job_N/doing/tasks.json", "w") as f:
    json.dump(data, f, indent=2, ensure_ascii=False)
```

### 最优执行序（避免二次失败）

1. 完成代码变更 → `git commit` → 记录 commit hash
2. `python3 .rick/skills/mark_task_success_skill/mark_task_success.py --job job_N --task taskX`
3. `git add tasks.json && git commit -m "chore(taskX): mark success"`
4. `python3 .rick/skills/rick-gates/helper.py .rick/jobs/job_N/doing`（此时必过）

### tasks.json 结构说明

```json
{
  "tasks": [
    {
      "task_id": "task1",
      "status": "success",
      "commit_hash": "40位SHA",
      "updated_at": "2026-07-02T..."
    }
  ]
}
```

`status=success` + `commit_hash` 非空 → rick-gates 门禁通过。
