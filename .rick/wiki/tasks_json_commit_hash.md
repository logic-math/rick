# tasks_json_commit_hash

## 触发场景

当 `doing_check` 在 commit 之后报错，提示 tasks.json 中 task 状态为 success 但缺少 `commit_hash` 字段时使用。

信号词：`doing_check: task status=success but commit_hash is empty`

## 预期效果

一次修复通过 doing_check，不需要反复读取 tasks.json 格式和 check 源码。

## 使用方法

tasks.json 中 success 状态的 task 必须包含 `commit_hash` 字段，值为完整 40 位 SHA：

```bash
# 获取当前 commit hash
git rev-parse HEAD

# 用 Python 原子修改 tasks.json（避免 Edit old_string 匹配失败）
python3 -c "
import json
with open('.rick/jobs/job_N/doing/tasks.json', 'r') as f:
    data = json.load(f)
for t in data['tasks']:
    if t['task_id'] == 'taskX':
        t['status'] = 'success'
        t['attempts'] = 1
        t['commit_hash'] = '<HASH>'
with open('.rick/jobs/job_N/doing/tasks.json', 'w') as f:
    json.dump(data, f, indent=2, ensure_ascii=False)
"
```

**前置动作**：在 task 实现 commit 之前先用 `git rev-parse HEAD` 获取 hash，与 status 更新合并为一步。
