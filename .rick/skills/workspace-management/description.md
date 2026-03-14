# 工作空间管理技能

## 技能概述

工作空间管理技能负责创建和维护 Rick CLI 的目录结构（.rick/）。通过封装文件系统操作，自动创建必要的目录和文件，确保工作空间的一致性。

核心功能：**自动化目录管理**。在首次运行时自动创建 .rick/ 结构，后续操作自动创建 job 目录。

## 使用场景

### 1. 工作空间初始化
- **场景**: 首次运行 `rick plan` 时创建 .rick/ 目录
- **示例**: 自动创建 wiki/、skills/、jobs/ 目录
- **价值**: 用户无需手动创建目录

### 2. Job 目录创建
- **场景**: 执行 `rick plan` 时创建新的 job 目录
- **示例**: 创建 jobs/job_1/plan/、doing/、learning/
- **价值**: 保证目录结构一致性

### 3. 确保目录存在
- **场景**: 写入文件前确保父目录存在
- **示例**: 保存 tasks.json 前创建 plan/ 目录
- **价值**: 避免"目录不存在"错误

### 4. 路径解析
- **场景**: 获取 job 相关路径
- **示例**: GetJobPath("job_1") 返回绝对路径
- **价值**: 统一路径管理

## 核心优势

### ✅ 优点

1. **自动化**: 无需手动创建目录
2. **一致性**: 保证目录结构统一
3. **幂等性**: 多次调用不会报错
4. **错误处理**: 统一的错误处理
5. **简单易用**: 封装复杂的路径操作

### ⚠️ 注意事项

1. **权限问题**: 需要文件系统写权限
2. **路径冲突**: 避免与用户现有目录冲突
3. **跨平台**: 处理 Windows 和 Unix 路径差异
4. **原子性**: 目录创建不是原子操作

## Rick CLI 目录结构
```
.rick/
  OKR.md          # 项目目标
  SPEC.md         # 技术规范
  wiki/           # 知识库
  skills/         # 技能库
  jobs/           # 任务目录
    job_0/
      plan/       # 规划阶段
        tasks/
        tasks.json
      doing/      # 执行阶段
        debug.md
      learning/   # 学习阶段
    job_1/
      ...
```

## 实现示例
```go
type Workspace struct {
    rickDir string
}

func (w *Workspace) InitWorkspace() error {
    directories := []string{
        w.rickDir,
        filepath.Join(w.rickDir, "wiki"),
        filepath.Join(w.rickDir, "skills"),
        filepath.Join(w.rickDir, "jobs"),
    }

    for _, dir := range directories {
        if err := os.MkdirAll(dir, 0755); err != nil {
            return err
        }
    }

    return nil
}
```

## 实际效果

在 Rick CLI 项目中：
- **创建速度**: < 10ms
- **目录权限**: 0755（rwxr-xr-x）
- **幂等性**: 多次创建不报错

## 扩展阅读

- [Go os 包文档](https://pkg.go.dev/os)
- Rick CLI 源码: `internal/workspace/workspace.go`

---

*难度: ⭐*
*分类: 文件系统*
*最后更新: 2026-03-14*
