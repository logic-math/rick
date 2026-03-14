# Workspace Module（工作空间模块）

## 概述
Workspace Module 负责管理 `.rick/` 目录结构，包括 job 目录创建、tasks.json 读写和知识库管理。

## 模块位置
`internal/workspace/`

## 工作空间结构

### .rick/ 目录结构
```
.rick/
├── config.json              # 全局配置
├── jobs/                    # 任务目录
│   ├── job_1/
│   │   ├── plan/            # 规划阶段
│   │   │   ├── tasks/
│   │   │   │   ├── task1.md
│   │   │   │   └── task2.md
│   │   │   └── tasks.json
│   │   ├── doing/           # 执行阶段
│   │   │   ├── tasks.json   # 更新后的状态
│   │   │   ├── debug.md     # 问题记录
│   │   │   └── test_scripts/
│   │   └── learning/        # 学习阶段
│   │       ├── summary.md
│   │       ├── insights.md
│   │       └── knowledge/
│   └── job_2/
│       └── ...
├── knowledge/               # 知识库
│   ├── patterns/
│   ├── best_practices/
│   └── lessons_learned/
└── wiki/                    # Wiki 知识库
    ├── index.md
    ├── architecture.md
    ├── core-concepts.md
    └── modules/
```

## 核心功能

### 1. 工作空间初始化
**职责**: 创建 `.rick/` 目录结构

**触发时机**: 首次执行 `rick plan` 命令时

**核心函数**:
```go
// EnsureWorkspace 确保工作空间存在
func EnsureWorkspace() error {
    // 1. 检查 .rick/ 是否存在
    rickDir := ".rick"
    if _, err := os.Stat(rickDir); err == nil {
        return nil // 已存在
    }

    // 2. 创建目录结构
    dirs := []string{
        ".rick",
        ".rick/jobs",
        ".rick/knowledge",
        ".rick/knowledge/patterns",
        ".rick/knowledge/best_practices",
        ".rick/knowledge/lessons_learned",
        ".rick/wiki",
        ".rick/wiki/modules",
    }

    for _, dir := range dirs {
        err := os.MkdirAll(dir, 0755)
        if err != nil {
            return fmt.Errorf("failed to create %s: %w", dir, err)
        }
    }

    log.Println("[INFO] 工作空间已初始化")
    return nil
}
```

### 2. Job 目录管理
**职责**: 创建和管理 job 目录

**核心函数**:
```go
// CreateJobDir 创建 job 目录
func CreateJobDir(jobID string, stage string) (string, error) {
    // 1. 构建目录路径
    jobDir := filepath.Join(".rick", "jobs", jobID, stage)

    // 2. 创建目录
    err := os.MkdirAll(jobDir, 0755)
    if err != nil {
        return "", fmt.Errorf("failed to create job dir: %w", err)
    }

    // 3. 创建子目录（根据阶段）
    if stage == "plan" {
        os.MkdirAll(filepath.Join(jobDir, "tasks"), 0755)
    } else if stage == "doing" {
        os.MkdirAll(filepath.Join(jobDir, "test_scripts"), 0755)
    } else if stage == "learning" {
        os.MkdirAll(filepath.Join(jobDir, "knowledge"), 0755)
    }

    return jobDir, nil
}

// GetJobDir 获取 job 目录路径
func GetJobDir(jobID string, stage string) string {
    return filepath.Join(".rick", "jobs", jobID, stage)
}

// ListJobs 列出所有 job
func ListJobs() ([]string, error) {
    jobsDir := filepath.Join(".rick", "jobs")
    entries, err := os.ReadDir(jobsDir)
    if err != nil {
        return nil, err
    }

    var jobs []string
    for _, entry := range entries {
        if entry.IsDir() {
            jobs = append(jobs, entry.Name())
        }
    }

    return jobs, nil
}
```

### 3. tasks.json 管理
**职责**: 读写 tasks.json 文件

**数据结构**:
```go
type Task struct {
    TaskID       string   `json:"task_id"`
    TaskName     string   `json:"task_name"`
    Dependencies []string `json:"dep"`
    Objectives   string   `json:"objectives"`
    KeyResults   []string `json:"key_results"`
    TestMethods  []string `json:"test_methods"`
    StateInfo    struct {
        Status     string `json:"status"` // pending, doing, done, failed
        RetryCount int    `json:"retry_count"`
    } `json:"state_info"`
}
```

**核心函数**:
```go
// LoadTasksJSON 加载 tasks.json
func LoadTasksJSON(jobDir string) ([]Task, error) {
    tasksFile := filepath.Join(jobDir, "tasks.json")

    // 1. 读取文件
    data, err := os.ReadFile(tasksFile)
    if err != nil {
        return nil, fmt.Errorf("failed to read tasks.json: %w", err)
    }

    // 2. 解析 JSON
    var tasks []Task
    err = json.Unmarshal(data, &tasks)
    if err != nil {
        return nil, fmt.Errorf("failed to parse tasks.json: %w", err)
    }

    return tasks, nil
}

// SaveTasksJSON 保存 tasks.json
func SaveTasksJSON(jobDir string, tasks []Task) error {
    tasksFile := filepath.Join(jobDir, "tasks.json")

    // 1. 序列化为 JSON
    data, err := json.MarshalIndent(tasks, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal tasks: %w", err)
    }

    // 2. 写入文件
    err = os.WriteFile(tasksFile, data, 0644)
    if err != nil {
        return fmt.Errorf("failed to write tasks.json: %w", err)
    }

    return nil
}

// UpdateTaskStatus 更新任务状态
func UpdateTaskStatus(jobDir string, taskID string, status string) error {
    // 1. 加载 tasks.json
    tasks, err := LoadTasksJSON(jobDir)
    if err != nil {
        return err
    }

    // 2. 更新状态
    for i := range tasks {
        if tasks[i].TaskID == taskID {
            tasks[i].StateInfo.Status = status
            break
        }
    }

    // 3. 保存 tasks.json
    return SaveTasksJSON(jobDir, tasks)
}
```

### 4. 知识库管理
**职责**: 管理 `.rick/knowledge/` 知识库

**核心函数**:
```go
// SaveKnowledge 保存知识到知识库
func SaveKnowledge(category string, filename string, content string) error {
    // 1. 构建文件路径
    knowledgeDir := filepath.Join(".rick", "knowledge", category)
    os.MkdirAll(knowledgeDir, 0755)

    filePath := filepath.Join(knowledgeDir, filename)

    // 2. 写入文件
    err := os.WriteFile(filePath, []byte(content), 0644)
    if err != nil {
        return fmt.Errorf("failed to save knowledge: %w", err)
    }

    return nil
}

// LoadKnowledge 加载知识库内容
func LoadKnowledge(category string) ([]string, error) {
    knowledgeDir := filepath.Join(".rick", "knowledge", category)

    // 1. 读取目录
    entries, err := os.ReadDir(knowledgeDir)
    if err != nil {
        return nil, err
    }

    // 2. 读取所有文件
    var contents []string
    for _, entry := range entries {
        if !entry.IsDir() {
            filePath := filepath.Join(knowledgeDir, entry.Name())
            data, _ := os.ReadFile(filePath)
            contents = append(contents, string(data))
        }
    }

    return contents, nil
}

// ListKnowledge 列出知识库文件
func ListKnowledge(category string) ([]string, error) {
    knowledgeDir := filepath.Join(".rick", "knowledge", category)

    entries, err := os.ReadDir(knowledgeDir)
    if err != nil {
        return nil, err
    }

    var files []string
    for _, entry := range entries {
        if !entry.IsDir() {
            files = append(files, entry.Name())
        }
    }

    return files, nil
}
```

## 使用示例

### 示例1: 初始化工作空间
```go
func main() {
    // 首次 plan 时自动初始化
    err := workspace.EnsureWorkspace()
    if err != nil {
        log.Fatal(err)
    }
}
```

### 示例2: 创建 job 目录
```go
func executePlan(objective string) error {
    // 1. 生成 job_id
    jobID := generateJobID()

    // 2. 创建 plan 目录
    jobDir, err := workspace.CreateJobDir(jobID, "plan")
    if err != nil {
        return err
    }

    log.Printf("[INFO] 创建 job 目录: %s", jobDir)
    return nil
}
```

### 示例3: 管理 tasks.json
```go
func updateTaskAfterExecution(jobDir string, taskID string, success bool) error {
    // 1. 加载 tasks.json
    tasks, err := workspace.LoadTasksJSON(jobDir)
    if err != nil {
        return err
    }

    // 2. 更新状态
    for i := range tasks {
        if tasks[i].TaskID == taskID {
            if success {
                tasks[i].StateInfo.Status = "done"
            } else {
                tasks[i].StateInfo.Status = "failed"
                tasks[i].StateInfo.RetryCount++
            }
            break
        }
    }

    // 3. 保存 tasks.json
    return workspace.SaveTasksJSON(jobDir, tasks)
}
```

### 示例4: 保存知识
```go
func saveLearning(jobID string) error {
    // 1. 提取设计模式
    pattern := extractPattern(jobID)
    err := workspace.SaveKnowledge("patterns", "pattern1.md", pattern)
    if err != nil {
        return err
    }

    // 2. 提取最佳实践
    bestPractice := extractBestPractice(jobID)
    err = workspace.SaveKnowledge("best_practices", "practice1.md", bestPractice)
    if err != nil {
        return err
    }

    return nil
}
```

## 文件格式规范

### tasks.json 格式
```json
[
  {
    "task_id": "task1",
    "task_name": "创建基础设施模块",
    "dep": [],
    "objectives": "创建 Rick CLI 的基础设施模块",
    "key_results": [
      "完成 config 包",
      "完成 git 包",
      "完成 workspace 包"
    ],
    "test_methods": [
      "运行 go test ./internal/config/",
      "运行 go test ./internal/git/",
      "运行 go test ./internal/workspace/"
    ],
    "state_info": {
      "status": "pending",
      "retry_count": 0
    }
  }
]
```

### debug.md 格式
```markdown
# debug1: 测试失败 - TestParseTask

**问题描述**
执行 `go test ./internal/parser/` 时，TestParseTask 失败。

**复现步骤**
1. 运行 `go test ./internal/parser/`
2. 观察 TestParseTask 失败

**可能原因**
解析 task.md 时，依赖关系解析逻辑有误。

**解决状态**
未解决

**解决方法**
（待填写）
```

## 测试

### 单元测试
```bash
go test ./internal/workspace/
```

### 测试用例
```go
func TestEnsureWorkspace(t *testing.T) {
    // 清理测试环境
    os.RemoveAll(".rick")
    defer os.RemoveAll(".rick")

    err := EnsureWorkspace()
    if err != nil {
        t.Fatal(err)
    }

    // 验证目录存在
    if _, err := os.Stat(".rick"); os.IsNotExist(err) {
        t.Error(".rick directory should exist")
    }
}

func TestCreateJobDir(t *testing.T) {
    os.RemoveAll(".rick")
    defer os.RemoveAll(".rick")

    EnsureWorkspace()

    jobDir, err := CreateJobDir("job_1", "plan")
    if err != nil {
        t.Fatal(err)
    }

    // 验证目录存在
    if _, err := os.Stat(jobDir); os.IsNotExist(err) {
        t.Error("job directory should exist")
    }
}

func TestLoadSaveTasksJSON(t *testing.T) {
    os.RemoveAll(".rick")
    defer os.RemoveAll(".rick")

    EnsureWorkspace()
    jobDir, _ := CreateJobDir("job_1", "plan")

    // 保存 tasks.json
    tasks := []Task{
        {TaskID: "task1", TaskName: "测试任务"},
    }
    err := SaveTasksJSON(jobDir, tasks)
    if err != nil {
        t.Fatal(err)
    }

    // 加载 tasks.json
    loadedTasks, err := LoadTasksJSON(jobDir)
    if err != nil {
        t.Fatal(err)
    }

    if len(loadedTasks) != 1 {
        t.Errorf("expected 1 task, got %d", len(loadedTasks))
    }
}
```

## 最佳实践

1. **目录结构**: 遵循统一的目录结构规范
2. **文件命名**: 使用清晰的文件命名（如 task1.md, debug.md）
3. **JSON 格式**: 使用缩进的 JSON 格式，便于阅读
4. **错误处理**: 详细记录文件操作错误

## 常见问题

### Q1: 如何清理工作空间？
**A**: `rm -rf .rick/`

### Q2: 如何备份工作空间？
**A**: `cp -r .rick/ .rick_backup/`

### Q3: 如何迁移工作空间？
**A**: 直接复制 `.rick/` 目录到新项目。

## 未来优化

1. **工作空间压缩**: 自动压缩过期的 job 目录
2. **知识库搜索**: 支持全文搜索知识库
3. **工作空间统计**: 显示工作空间统计信息（job 数量、任务数量等）
4. **云同步**: 支持工作空间云同步

---

*最后更新: 2026-03-14*
