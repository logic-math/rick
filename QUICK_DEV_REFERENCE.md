# Rick CLI Dev 版本快速参考

## 🚀 快速开始

```bash
# 1️⃣ 安装 dev 版本
cd /Users/sunquan/ai_coding/CODING/rick
./scripts/install.sh --source --dev

# 2️⃣ 验证安装
rick_dev --version

# 3️⃣ 测试新功能
mkdir -p /tmp/rick_test && cd /tmp/rick_test && git init
rick_dev init
```

## 📝 修改代码的工作流

```bash
# 1. 修改源码
vim internal/cmd/init.go

# 2. 重新构建
./scripts/install.sh --source --dev

# 3. 测试
rick_dev init

# 4. 验证
ls -la .rick/
```

## 🎯 当前优化目标

### Init 命令优化

**当前问题**：
- ❌ 仅创建目录、配置、Git
- ❌ 没有生成全局上下文

**优化方向**：
- ✅ 自动执行 plan → doing → learning
- ✅ 生成 OKR.md、SPEC.md、Wiki、Skills

**实现文件**：
- `internal/cmd/init.go` - 修改 `NewInitCmd()`

## 📚 关键文档

| 文档 | 用途 |
|------|------|
| [DEV_GUIDE.md](./DEV_GUIDE.md) | 详细开发指南 |
| [DEV_SETUP_SUMMARY.md](./DEV_SETUP_SUMMARY.md) | 完整总结 |
| [Rick_Project_Complete_Description.md](./Rick_Project_Complete_Description.md) | 完整规范 |
| [MEMORY.md](../.claude/projects/-Users-sunquan-ai-coding-CODING-rick/memory/MEMORY.md) | 项目记忆库 |

## 🔧 常用命令

```bash
# 查看帮助
rick_dev --help
rick_dev init --help

# 详细输出
rick_dev -v init

# 干运行
rick_dev --dry-run init
```

## 💡 核心优化代码框架

```go
// internal/cmd/init.go
func NewInitCmd() *cobra.Command {
    initCmd := &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            // 1. 基础初始化
            ws, err := workspace.New()
            ws.InitWorkspace()
            config.SaveConfig(defaultConfig)
            initGitRepo(rickDir)

            // 2. 自动 plan → doing → learning
            requirement := "深度探索项目源码结构和架构设计"
            
            if err := executePlanWorkflow(requirement); err != nil {
                return err
            }
            if err := executeDoingWorkflow("job_0"); err != nil {
                return err
            }
            if err := executeLearningWorkflow("job_0"); err != nil {
                return err
            }

            fmt.Println("✅ Project initialized successfully!")
            return nil
        },
    }
    return initCmd
}
```

## ✅ 检查清单

在提交代码前检查：

- [ ] 代码编译无误
- [ ] `rick_dev init` 正常执行
- [ ] 生成了 `.rick/OKR.md`
- [ ] 生成了 `.rick/SPEC.md`
- [ ] 生成了 `.rick/wiki/`
- [ ] 生成了 `.rick/skills/`
- [ ] 没有引入新的依赖
- [ ] 代码风格一致

## 🐛 调试技巧

```bash
# 启用详细日志
rick_dev -v init

# 干运行模式（不执行实际操作）
rick_dev --dry-run init

# 检查目录结构
tree -L 2 .rick/

# 查看生成的文件
cat .rick/OKR.md
cat .rick/SPEC.md
ls -la .rick/wiki/
ls -la .rick/skills/
```

## 📊 版本对比

| 特性 | rick (生产) | rick_dev (开发) |
|------|-----------|----------------|
| 命令名 | `rick` | `rick_dev` |
| 安装位置 | `~/.rick/` | `~/.rick_dev/` |
| 符号链接 | `~/.local/bin/rick` | `~/.local/bin/rick_dev` |
| 用途 | 生产使用 | 开发测试 |
| 并行运行 | ✅ 支持 | ✅ 支持 |

## 🎓 学习资源

- **Go 基础**: `internal/cmd/` 中的命令实现
- **Cobra 框架**: 命令行参数处理
- **工作流程**: plan/doing/learning 的流程
- **提示词管理**: `internal/prompt/` 模块

## 📞 快速问题解答

**Q: 如何重新安装 dev 版本？**
```bash
./scripts/uninstall.sh --dev
./scripts/install.sh --source --dev
```

**Q: 如何对比生产版和开发版？**
```bash
rick init          # 生产版
rick_dev init      # 开发版
diff -r .rick_prod/.rick .rick_dev/.rick
```

**Q: 如何查看源码？**
```bash
# Init 命令
cat internal/cmd/init.go

# Plan 命令
cat internal/cmd/plan.go

# Doing 命令
cat internal/cmd/doing.go

# Learning 命令
cat internal/cmd/learning.go
```

---

**最后更新**: 2026-03-14  
**作者**: Claude Code  
**状态**: ✅ Dev 版本就绪，等待优化实现
