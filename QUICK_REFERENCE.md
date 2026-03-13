# Rick CLI 快速参考卡片

## 核心命令

| 命令 | 功能 | 示例 |
|------|------|------|
| `rick init` | 初始化项目 | `rick init` |
| `rick plan` | 规划任务 | `rick plan "实现新功能"` |
| `rick doing job_n` | 执行任务 | `rick doing job_1` |
| `rick learning job_n` | 知识积累 | `rick learning job_1` |

## 安装脚本

```bash
# 源码安装生产版
./install.sh

# 源码安装开发版
./install.sh --source --dev

# 二进制安装
./install.sh --binary

# 卸载生产版
./uninstall.sh

# 卸载开发版
./uninstall.sh --dev

# 更新生产版
./update.sh

# 更新开发版
./update.sh --dev
```

## 文件位置

| 项目 | 路径 |
|------|------|
| 生产版二进制 | `~/.rick/bin/rick` |
| 开发版二进制 | `~/.rick_dev/bin/rick_dev` |
| 生产配置 | `~/.rick/config.json` |
| 开发配置 | `~/.rick_dev/config.json` |
| 项目工作空间 | `.rick/` |
| 任务集合 | `.rick/jobs/job_n/` |

## 关键文件格式

### task.md 结构
```markdown
# 依赖关系
task1, task2

# 任务名称
任务标题

# 任务目标
具体目标

# 关键结果
1. 结果1
2. 结果2

# 测试方法
1. 测试步骤1
2. 测试步骤2
```

### tasks.json 结构
```json
[
  {
    "task_id": "task1",
    "task_name": "任务名称",
    "dep": [],
    "state_info": {"status": "pending"}
  }
]
```

## 开发工作流

### 场景1：开发新功能
```bash
./install.sh --source --dev        # 安装 dev 版本
rick_dev plan "新功能"              # 规划
rick_dev doing job_1                # 执行
rick plan "集成新功能"              # 使用生产版集成
rick doing job_2
./uninstall.sh --dev               # 卸载 dev 版本
```

### 场景2：自我重构
```bash
rick plan "重构 Rick 架构"           # 使用生产版规划
./install.sh --source --dev        # 安装 dev 版本
rick doing job_1                    # 使用生产版执行
rick_dev plan "验证重构"            # 使用 dev 版验证
rick_dev doing job_2
./update.sh                         # 更新生产版
./uninstall.sh --dev               # 卸载 dev 版本
```

## 关键配置项

### ~/.rick/config.json
```json
{
  "max_retries": 5,
  "claude_code_path": "/usr/local/bin/claude-code",
  "default_workspace": ".rick"
}
```

## 项目结构

```
.rick/
├── OKR.md                  # 项目全局目标
├── SPEC.md                 # 项目开发规范
├── wiki/                   # 项目知识库
├── skills/                 # 可复用技能库
└── jobs/
    ├── job_1/
    │   ├── plan/
    │   │   ├── draft/      # 调研草稿
    │   │   └── tasks/      # 标准化任务
    │   ├── doing/
    │   │   ├── doing.log   # 执行日志
    │   │   ├── debug.md    # 问题记录
    │   │   ├── tasks.json  # 任务DAG
    │   │   └── tests/      # 测试脚本
    │   └── learning/       # 知识沉淀
```

## 版本管理

### 生产版本 (rick)
- 安装路径：`~/.rick/`
- 命令名：`rick`
- 用途：生产环境、自我重构
- 配置文件：`~/.rick/config.json`

### 开发版本 (rick_dev)
- 安装路径：`~/.rick_dev/`
- 命令名：`rick_dev`
- 用途：新功能开发、测试
- 配置文件：`~/.rick_dev/config.json`

## 故障排查

### 问题：任务执行失败
**解决方案**：
1. 查看 `debug.md` 中的错误信息
2. 修改 `plan/tasks/task*.md` 中的任务定义
3. 重新运行 `rick doing job_n`

### 问题：安装失败
**解决方案**：
1. 检查 Go 环境（版本 >= 1.21）
2. 检查磁盘空间
3. 查看错误日志

### 问题：版本冲突
**解决方案**：
1. 卸载所有版本：`./uninstall.sh --all`
2. 重新安装：`./install.sh`

## 常用命令

```bash
# 查看版本
rick --version
rick_dev --version

# 查看帮助
rick --help
rick plan --help

# 查看执行历史
git log --oneline

# 回滚到上一个版本
git checkout HEAD~1

# 查看任务状态
cat .rick/jobs/job_1/doing/tasks.json

# 查看执行日志
tail -f .rick/jobs/job_1/doing/doing.log

# 查看问题记录
cat .rick/jobs/job_1/doing/debug.md
```

## 提示词模板位置

```
internal/prompt/templates/
├── plan.md          # 规划阶段提示词
├── doing.md         # 执行阶段提示词
├── test.md          # 测试生成提示词
└── learning.md      # 学习阶段提示词
```

## 重要链接

- [完整研究报告](research/使用_golang_开发_rick_命令行程序.md)
- [开发指南](DEVELOPMENT_GUIDE.md)
- [研究总结](RESEARCH_SUMMARY.md)
- [项目记忆](~/.claude/projects/.../memory/MEMORY.md)
- [Rick 规范](Rick_Project_Complete_Description.md)

---

**最后更新**: 2026-03-13
