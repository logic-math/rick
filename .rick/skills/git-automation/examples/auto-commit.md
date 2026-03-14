# 案例: Rick CLI 自动提交

## 场景
任务执行成功后自动提交到 Git。

## 实现
```go
func commitTaskResult(task *Task, projectRoot string) error {
    gm := git.New(projectRoot)

    // 添加所有更改
    if err := gm.AddFiles([]string{"."}); err != nil {
        return err
    }

    // 生成提交信息
    message := fmt.Sprintf("feat(%s): %s\n\n%s",
        task.ID, task.Name, task.Goal)

    // 提交
    return gm.Commit(message)
}
```

## 效果
```bash
rick doing job_1
# Task task1 completed

git log -1
# commit abc123
# feat(task1): 创建 Wiki 索引
#
# 建立知识库索引系统
```

---

*最后更新: 2026-03-14*
