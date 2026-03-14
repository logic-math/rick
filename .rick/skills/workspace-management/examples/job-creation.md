# 案例: Job 目录自动创建

## 场景
执行 `rick plan "新任务"` 时自动创建 job 目录结构。

## 执行前
```
.rick/
  jobs/
    # 空目录
```

## 执行
```bash
rick plan "创建 Wiki 索引"
```

## 执行后
```
.rick/
  jobs/
    job_0/
      plan/
        tasks/
          task1.md
        tasks.json
      doing/
      learning/
```

## 实现
```go
// 在 plan 命令中
jobID := getNextJobID()
ws.CreateJobStructure(jobID)
```

---

*最后更新: 2026-03-14*
