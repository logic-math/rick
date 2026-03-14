# 工作空间管理 - 实现细节

## Workspace 结构
```go
type Workspace struct {
    rickDir string
}

func New() (*Workspace, error) {
    rickDir, err := GetRickDir()
    if err != nil {
        return nil, err
    }

    ws := &Workspace{rickDir: rickDir}
    if err := ws.EnsureDirectories(); err != nil {
        return nil, err
    }

    return ws, nil
}
```

## 核心操作

### 1. 初始化工作空间
```go
func (w *Workspace) InitWorkspace() error {
    directories := []string{
        w.rickDir,
        filepath.Join(w.rickDir, WikiDirName),
        filepath.Join(w.rickDir, SkillsDirName),
        filepath.Join(w.rickDir, JobsDirName),
    }

    for _, dir := range directories {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return fmt.Errorf("failed to create directory %s: %w", dir, err)
        }
    }

    // 创建 OKR.md
    okriPath := filepath.Join(w.rickDir, OKRFileName)
    if _, err := os.Stat(okriPath); os.IsNotExist(err) {
        if err := os.WriteFile(okriPath, []byte("# OKR\n\n"), 0644); err != nil {
            return err
        }
    }

    return nil
}
```

### 2. 创建 Job 目录
```go
func (w *Workspace) CreateJobStructure(jobID string) error {
    jobPath, err := w.GetJobPath(jobID)
    if err != nil {
        return err
    }

    directories := []string{
        jobPath,
        filepath.Join(jobPath, PlanDirName),
        filepath.Join(jobPath, DoingDirName),
        filepath.Join(jobPath, LearningDirName),
    }

    for _, dir := range directories {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return err
        }
    }

    return nil
}
```

### 3. 获取 Job 路径
```go
func (w *Workspace) GetJobPath(jobID string) (string, error) {
    if jobID == "" {
        return "", fmt.Errorf("jobID cannot be empty")
    }
    return filepath.Join(w.rickDir, JobsDirName, jobID), nil
}
```

## 使用示例
```go
ws, _ := workspace.New()
ws.InitWorkspace()
ws.CreateJobStructure("job_1")

jobPath, _ := ws.GetJobPath("job_1")
fmt.Println(jobPath)
// Output: /path/to/project/.rick/jobs/job_1
```

---

*参考: `internal/workspace/workspace.go`*
*最后更新: 2026-03-14*
