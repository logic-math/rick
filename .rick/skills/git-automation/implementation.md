# Git 自动化 - 实现细节

## GitManager 结构
```go
type GitManager struct {
    repoPath string
}

func New(repoPath string) *GitManager {
    return &GitManager{repoPath: repoPath}
}
```

## 核心操作

### 1. 初始化仓库
```go
func (gm *GitManager) InitRepo() error {
    if err := os.MkdirAll(gm.repoPath, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    cmd := exec.Command("git", "init")
    cmd.Dir = gm.repoPath
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to initialize git repo: %w", err)
    }

    return nil
}
```

### 2. 添加文件
```go
func (gm *GitManager) AddFiles(paths []string) error {
    if len(paths) == 0 {
        return fmt.Errorf("no files to add")
    }

    args := append([]string{"add"}, paths...)
    cmd := exec.Command("git", args...)
    cmd.Dir = gm.repoPath
    return cmd.Run()
}
```

### 3. 提交代码
```go
func (gm *GitManager) Commit(message string) error {
    if message == "" {
        return fmt.Errorf("commit message cannot be empty")
    }

    cmd := exec.Command("git", "commit", "-m", message)
    cmd.Dir = gm.repoPath
    return cmd.Run()
}
```

### 4. 查询日志
```go
func (gm *GitManager) GetLog(limit int) ([]CommitInfo, error) {
    format := "%H%n%s%n%an%n%ai"
    cmd := exec.Command("git", "log",
        fmt.Sprintf("--max-count=%d", limit),
        fmt.Sprintf("--format=%s", format))
    cmd.Dir = gm.repoPath
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    // 解析输出
    commits := []CommitInfo{}
    lines := strings.Split(strings.TrimSpace(string(output)), "\n")
    for i := 0; i < len(lines); i += 4 {
        commits = append(commits, CommitInfo{
            Hash:    lines[i],
            Message: lines[i+1],
            Author:  lines[i+2],
            Date:    parseDate(lines[i+3]),
        })
    }

    return commits, nil
}
```

## 使用示例
```go
gm := git.New("/path/to/project")
gm.InitRepo()
gm.AddFiles([]string{"."})
gm.Commit("feat: initial commit")

// 查询日志
commits, _ := gm.GetLog(10)
for _, c := range commits {
    fmt.Printf("%s: %s\n", c.Hash[:7], c.Message)
}
```

---

*参考: `internal/git/git.go`*
*最后更新: 2026-03-14*
