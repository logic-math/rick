---
name: do-check-mark-success-loop
trigger: "当 doing_check 报错 task status != success 或 commit_hash 缺失时触发"
scope: "doing"
---

# Loop: doing_check 失败修复循环

## 依赖准备

| 依赖项 | 确认命令 | 要求 |
|--------|----------|------|
| Python | `python3 --version` | 3.8+ |
| rick binary | `ls ./bin/rick` | 已构建（`./scripts/build.sh`） |
| Git | `git status` | 在 rick 项目根目录 |

## 目标（Goal）

让 `./bin/rick tools doing_check job_N` 输出 `✅ PASS`，对应 task 的 `status=success` + `commit_hash` 非空。

- 成功标准：`doing_check` 返回 exit code 0，输出 `✅ PASS`
- 自评命令：`./bin/rick tools doing_check job_N`

## 可调用工具（Tool Access）

- `python3 mark_task_success.py --job --task`：更新 tasks.json status 和 commit_hash — 约束：必须在 code commit 之后调用
- `git rev-parse HEAD`：获取当前 commit hash — 约束：只读操作
- `git add / git commit`：提交 tasks.json 修改 — 约束：只提交 tasks.json，不包含其他文件
- `./bin/rick tools doing_check job_N`：验证 check 通过 — 约束：使用本地 ./bin/rick 而非系统版

## 上下文管理（Context Management）

**需要保留**：当前 job_id、task_id、最近一次 doing_check 错误输出
**可丢弃**：已通过的 task 的详细修复过程

## 子 Agent 工作流

```
[Step 1: 读 tasks.json] → [Step 2: 执行 mark_task_success] → [Step 3: git commit] → [Step 4: 验证 doing_check]
                                                                                              ↑              |
                                                                                              └──[FAIL: 修复]─┘
```

**Step 1：读取 tasks.json，确认当前 task 状态**
- 加载 skill：`.rick/skills/mark_task_success_skill/skill.md`
- 操作：`cat .rick/jobs/job_N/doing/tasks.json`
- 确认失败的 task_id 和当前 status

**Step 2：执行 mark_task_success 脚本**
- 操作：
  ```bash
  python3 .rick/skills/mark_task_success_skill/mark_task_success.py --job job_N --task taskX
  ```
- 产出：tasks.json 中 `status=success`，`commit_hash=<HEAD>` 填入

**Step 3：提交 tasks.json**
- 操作：
  ```bash
  git add .rick/jobs/job_N/doing/tasks.json
  git commit -m "chore(taskX): mark taskX success in tasks.json"
  ```

**Step 4：验证 doing_check**
- 操作：
  ```bash
  ./bin/rick tools doing_check job_N
  ```
- 通过 → 退出 Loop（成功）
- 失败 → 根据错误信息执行修复（见下），然后回到 Step 3

**FAIL 处理：commit_hash 仍为空**

手动写入：
```python
import json, subprocess

commit_hash = subprocess.run(["git", "rev-parse", "HEAD"], capture_output=True, text=True).stdout.strip()
with open(".rick/jobs/job_N/doing/tasks.json", "r") as f:
    data = json.load(f)
for task in data["tasks"]:
    if task["task_id"] == "taskX":
        task["status"] = "success"
        task["commit_hash"] = commit_hash
with open(".rick/jobs/job_N/doing/tasks.json", "w") as f:
    json.dump(data, f, indent=2, ensure_ascii=False)
```

**FAIL 处理：debug/bug*.md 格式错误**
- 加载 skill：`.rick/skills/check_mechanism_skill/skill.md`
- 检查 frontmatter 是否包含 title/status/summary 字段

## 产出评估

**调用验证 skill**：`.rick/skills/check_mechanism_skill/skill.md`

| 检查项 | 验证命令 | 通过标准 |
|--------|----------|----------|
| doing_check 通过 | `./bin/rick tools doing_check job_N` | 输出 `✅ PASS`，exit code 0 |
| tasks.json 正确 | `cat tasks.json` | task 含 status=success + commit_hash 非空 |

## 停止标准

**成功退出**：`doing_check` 输出 `✅ PASS`

**优雅退出**（3 轮无进展）：
- 将 debug 信息写入 `doing/debug/bug{n}-check-stuck.md`
- status 标为 `"❌ 卡住"`，等待人工介入
