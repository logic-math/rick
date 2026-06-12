# mark-task-success（tasks.json 两阶段提交）

## 触发场景

当 doing task 代码已提交（已有 commit hash）但 doing_check 因 tasks.json 仍为 "running" 状态而失败时使用。信号：`doing_check` 输出 `task status != success`。

## 预期效果

将 3-4 次重复工具调用（read tasks.json → edit → commit → doing_check）压缩为一条工具调用，消除每个 task 末尾的固定额外开销。

## 使用方法

工具路径：`.rick/tools/mark_task_success.py`

```bash
# 代码已 commit 后，立即执行：
python3 .rick/tools/mark_task_success.py --job job_N --task taskX
# 工具自动读取 HEAD commit hash，更新 tasks.json status=success + commit_hash
# 然后手动提交 tasks.json：
git add .rick/jobs/job_N/doing/tasks.json && git commit -m "chore(taskX): mark taskX success in tasks.json"
# 最后运行验证：
./bin/rick tools doing_check job_N
```

**更优执行序**（避免 doing_check 二次失败）：

1. 完成代码变更 → `git commit` → 记录 commit hash
2. `python3 .rick/tools/mark_task_success.py --job job_N --task taskX`
3. `git add tasks.json && git commit -m "chore(taskX): mark success"`
4. `./bin/rick tools doing_check job_N`（此时必过）
