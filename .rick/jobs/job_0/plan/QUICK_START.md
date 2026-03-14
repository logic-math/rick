# Job 0 快速开始指南

## 🚀 5 分钟快速启动

### 前提条件
```bash
# 1. 确认在项目根目录
pwd
# 输出: /Users/sunquan/ai_coding/CODING/rick

# 2. 确认 rick 命令可用
rick --version
# 输出: Rick CLI version 0.1.0

# 3. 确认 job_0 规划已创建
ls -la .rick/jobs/job_0/plan/
```

### 启动执行
```bash
# 方式 1: 使用 Rick CLI 自动执行（推荐）
rick doing job_0

# 方式 2: 手动执行每个任务
# 按顺序执行 task1 → task2 → task3 → task4 → task5 → task6 → task7
```

## 📋 执行前检查清单

### 环境检查
- [ ] Go 版本 >= 1.25.0
- [ ] Rick CLI 已安装
- [ ] Git 已初始化
- [ ] 在项目根目录

### 文件检查
- [ ] `.rick/jobs/job_0/plan/` 目录存在
- [ ] 包含 7 个 task*.md 文件
- [ ] README.md 和 EXECUTION_PLAN.md 已创建

### 权限检查
- [ ] 可以创建 `.rick/` 目录下的文件
- [ ] 可以提交 Git commit
- [ ] 可以读取项目所有代码文件

## 🎯 执行步骤

### Step 1: 准备工作空间
```bash
# 确保 .rick 目录结构完整
mkdir -p .rick/wiki/modules
mkdir -p .rick/wiki/tutorials
mkdir -p .rick/skills
mkdir -p .rick/jobs/job_0/{plan,doing,learning}

# 验证目录结构
tree .rick -L 3
```

### Step 2: 开始执行
```bash
# 使用 Rick CLI 执行
rick doing job_0

# Rick 将会：
# 1. 加载 job_0/plan/ 下的所有任务
# 2. 构建 DAG 依赖图
# 3. 按拓扑排序执行任务
# 4. 每个任务失败最多重试 5 次
# 5. 自动提交 Git commit
```

### Step 3: 监控进度
```bash
# 查看执行日志
tail -f .rick/jobs/job_0/doing/doing.log

# 查看任务状态
cat .rick/jobs/job_0/doing/tasks.json

# 查看调试信息（如果有失败）
cat .rick/jobs/job_0/doing/debug.md
```

### Step 4: 验证结果
```bash
# 检查生成的文件
ls -la .rick/OKR.md
ls -la .rick/SPEC.md
ls -la .rick/wiki/
ls -la .rick/skills/

# 统计生成的文件数量
find .rick -type f -name "*.md" | wc -l
# 预期: 30+ 个文件

# 统计文档总字数
find .rick -type f -name "*.md" -exec wc -w {} + | tail -1
# 预期: 10,000+ 字
```

### Step 5: 审查和调整
```bash
# 查看关键文档
cat .rick/OKR.md
cat .rick/SPEC.md
cat .rick/wiki/index.md

# 如果需要调整，手动编辑文件后重新执行
# Rick 会跳过已完成的任务
```

## 📊 预期产出

### 文件清单
```
.rick/
├── OKR.md                              ✅ 项目目标
├── SPEC.md                             ✅ 开发规范
├── wiki/
│   ├── index.md                        ✅ Wiki 索引
│   ├── architecture.md                 ✅ 架构设计
│   ├── core-concepts.md                ✅ 核心概念
│   ├── getting-started.md              ✅ 快速入门
│   ├── best-practices.md               ✅ 最佳实践
│   ├── modules/                        ✅ 模块文档 (8 个)
│   │   ├── cmd.md
│   │   ├── workspace.md
│   │   ├── prompt.md
│   │   ├── executor.md
│   │   ├── parser.md
│   │   ├── git.md
│   │   ├── config.md
│   │   └── logging.md
│   └── tutorials/                      ✅ 教程 (3-5 个)
│       ├── tutorial-1-simple-project.md
│       ├── tutorial-2-self-refactoring.md
│       └── ...
├── skills/
│   ├── index.md                        ✅ 技能索引
│   ├── dag-topological-sort/           ✅ DAG 技能
│   ├── markdown-parsing/               ✅ Markdown 技能
│   ├── embedded-resources/             ✅ 嵌入资源技能
│   ├── retry-pattern/                  ✅ 重试模式技能
│   └── git-automation/                 ✅ Git 自动化技能
└── jobs/job_0/
    ├── plan/                           ✅ 规划文档
    ├── doing/                          ✅ 执行日志
    │   ├── doing.log
    │   ├── tasks.json
    │   └── debug.md
    └── learning/                       ✅ 学习总结
        └── summary.md
```

### 质量指标
- ✅ 文件数量: 30+ 个
- ✅ 文档字数: 10,000+ 字
- ✅ 模块文档: 8 个（每个 500+ 字）
- ✅ 技能数量: 5+ 个
- ✅ 教程数量: 3+ 个
- ✅ 任务完成率: 100%

## ⚡ 快速命令参考

### 执行相关
```bash
# 开始执行 job_0
rick doing job_0

# 查看帮助
rick doing --help

# 启用详细输出
rick doing job_0 --verbose

# 干运行（不实际执行）
rick doing job_0 --dry-run
```

### 检查相关
```bash
# 查看 job 状态
cat .rick/jobs/job_0/doing/tasks.json | jq '.[] | {task_id, status}'

# 查看失败任务
cat .rick/jobs/job_0/doing/debug.md

# 查看执行日志
tail -100 .rick/jobs/job_0/doing/doing.log
```

### 验证相关
```bash
# 验证文件存在
test -f .rick/OKR.md && echo "✅ OKR.md exists"
test -f .rick/SPEC.md && echo "✅ SPEC.md exists"

# 统计文件
find .rick -type f -name "*.md" | wc -l

# 统计字数
find .rick -type f -name "*.md" -exec wc -w {} + | tail -1

# 检查链接有效性（需要安装 markdown-link-check）
find .rick -name "*.md" -exec markdown-link-check {} \;
```

## 🐛 故障排查

### 问题 1: Rick 命令找不到
```bash
# 解决方案: 安装 Rick
cd /Users/sunquan/ai_coding/CODING/rick
./scripts/install.sh

# 验证安装
which rick
rick --version
```

### 问题 2: 任务执行失败
```bash
# 查看错误日志
cat .rick/jobs/job_0/doing/debug.md

# 查看执行日志
tail -100 .rick/jobs/job_0/doing/doing.log

# 手动修复后重新执行
rick doing job_0
# Rick 会自动跳过已完成的任务
```

### 问题 3: 文件权限问题
```bash
# 检查权限
ls -la .rick/

# 修复权限
chmod -R u+w .rick/
```

### 问题 4: Git 提交失败
```bash
# 检查 Git 状态
git status

# 手动提交
git add .
git commit -m "job_0: Complete task X"
```

### 问题 5: 生成的文档质量不佳
```bash
# 手动编辑文档
vim .rick/OKR.md

# 重新执行验证任务
# 编辑 tasks.json，将 task7 状态改为 pending
vim .rick/jobs/job_0/doing/tasks.json

# 重新执行
rick doing job_0
```

## 📞 获取帮助

### 文档资源
- **规划文档**: `.rick/jobs/job_0/plan/README.md`
- **执行计划**: `.rick/jobs/job_0/plan/EXECUTION_PLAN.md`
- **依赖关系**: `.rick/jobs/job_0/plan/TASK_DEPENDENCIES.md`
- **开发指南**: `DEVELOPMENT_GUIDE.md`
- **快速参考**: `QUICK_REFERENCE.md`

### 命令帮助
```bash
# Rick 帮助
rick --help
rick doing --help
rick learning --help

# 查看版本
rick --version
```

### 社区支持
- **GitHub Issues**: 报告问题和建议
- **项目文档**: 查看完整文档
- **代码示例**: 参考现有实现

## ✅ 完成标志

当看到以下输出时，job_0 执行成功：

```bash
✅ Job 0 execution completed successfully!

Summary:
- Total tasks: 7
- Successful: 7
- Failed: 0
- Duration: X hours

Generated files:
- OKR.md: ✅
- SPEC.md: ✅
- Wiki docs: 15+ files ✅
- Skills: 5+ skills ✅
- Tutorials: 3+ tutorials ✅

Next steps:
1. Review the generated documents
2. Run learning phase: rick learning job_0
3. Merge learning results to global context
```

## 🎯 下一步

完成 job_0 后：

1. **审查文档**
   ```bash
   # 查看关键文档
   cat .rick/OKR.md
   cat .rick/SPEC.md
   cat .rick/wiki/index.md
   ```

2. **运行 Learning 阶段**
   ```bash
   rick learning job_0
   ```

3. **合并到全局上下文**
   ```bash
   # Rick 会自动将 job_0/learning/ 的内容合并到 .rick/ 全局目录
   ```

4. **开始下一个 Job**
   ```bash
   # 基于完整的全局上下文，规划下一个 Job
   rick plan "提升 CLI 测试覆盖率到 80%+"
   ```

---

**文档版本**: v1.0
**创建时间**: 2026-03-14
**适用于**: Rick CLI v0.1.0+
