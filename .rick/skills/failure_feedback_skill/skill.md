# skill:failure-feedback（失败信息传递机制）

## 触发场景

doing 阶段 task 失败重试时，需要理解或调整失败信息如何传递给下一轮 Agent 时使用。

## 预期效果

- 下一轮 Agent 能看到上一轮的完整错误输出（含 traceback/stderr）
- prompt 不因重试次数增加而无限膨胀（最多保留最近 2 次，3000 字符上限）

## 核心内容

### 失败信息流转（task8 后）

```
pi 会话未 settle / 任务未完成 / 门禁失败
    → runtime.Run 返回 error（未解析出 sessionID 或未收 agent_settled）
    → rick-gates helper.py 校验（可解析 / 无 zombie / success 有 commit_hash）
    → handler.Doing 重试循环（上限 cfg.MaxRetries）
    → 重新生成「只含剩余 pending」的 workflowScript 编排
```

### 重试收敛

- rick 侧薄重试循环仅作为兜底安全网（非正确性前提）。
- 每次重试重新计算剩余拓扑（生成期过滤 status=success），已完成 task 天然不在编排里。
- 上限为 `cfg.MaxRetries`（默认 3），超限后把最后的门禁/未 settle 错误返回给上层。

### 调试：查看重试信息

- `doing/prompts/doing_prompt.md` 持久化落盘，可直接查看当前轮的 workflowScript 编排。
- `doing/tasks.json` 记录每个 task 的状态与 commit_hash。

### 为什么完整 testOutput 重要

旧版（500字符截断）agent 只能看到：
```
test did not pass: assertion failed; key not found
```

新版 agent 能看到：
```
test did not pass: assertion failed; key not found
Full test output:
AssertionError: OKR.md not found at .rick/jobs/job_1/plan/OKR.md
{"pass": false, "errors": ["assertion failed"]}
```

文件路径和行号直接可见，修复效率大幅提升。
