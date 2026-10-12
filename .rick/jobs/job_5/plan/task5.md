# 依赖关系
task3, task4

# 任务名称
实现 doing_check 和 learning_check 子命令

# 任务目标
在 `rick tools` 框架下实现 `doing_check` 和 `learning_check` 两个子命令，分别对 doing 和 learning 阶段的输出进行结构性校验。校验失败时同样支持自动修复机制。

# 关键结果
1. 完成 internal/cmd/tools_doing_check.go：doing_check 子命令，检查以下规则：
   - `.rick/jobs/job_N/doing/tasks.json` 存在且可被 executor.LoadTasksJSON 解析
   - `.rick/jobs/job_N/doing/debug.md` 存在（强制工作日志）
   - tasks.json 中所有 task 状态为 success 或 failed（无 running 僵尸状态）
   - status=success 的 task 有非空 commit_hash 记录
2. 完成 internal/cmd/tools_learning_check.go：learning_check 子命令，检查以下规则：
   - `.rick/jobs/job_N/learning/SUMMARY.md` 存在
   - 如存在 `learning/skills/*.py`，每个文件通过 Python 语法检查（`python3 -c "import ast; ast.parse(open('file').read())"` 返回 0）
   - 如存在 `learning/OKR.md`，文件包含 `## O` 开头的目标章节和 `### 关键结果` 章节（完整格式校验）
   - 如存在 `learning/SPEC.md`，文件包含 `## 技术栈`、`## 架构设计`、`## 开发规范`、`## 工程实践` 四个章节（完整格式校验）
3. 两个命令失败时均支持 `autoFix` 自动修复（复用 task3 中的公共函数）：
   - doing_check 修复对象是元数据（补写 debug.md、修正 tasks.json 状态字段、补 commit_hash），不是重跑任务
   - learning_check 修复对象是文档/脚本的格式问题
4. 两个命令的成功输出格式：`✅ doing check passed: N/N tasks succeeded` 和 `✅ learning check passed`

# 测试方法
1. 先运行 `go build -o bin/rick ./cmd/rick/` 构建最新二进制
2. 运行 `./bin/rick tools doing_check job_1`，验证对已有 job_1 的 doing 目录通过检查
3. 删除 job_1/doing/debug.md 的副本，运行 `./bin/rick tools doing_check job_1`，验证报错"debug.md not found"
4. 创建一个包含语法错误的 .py 文件放入 learning/skills/，运行 `./bin/rick tools learning_check job_1`，验证报错包含文件名和语法错误信息
5. 运行 `./bin/rick tools doing_check --help` 和 `./bin/rick tools learning_check --help`，验证帮助信息正确
6. 运行 `go test ./internal/cmd/...`，验证新增测试通过
