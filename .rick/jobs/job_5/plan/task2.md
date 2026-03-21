# 依赖关系
无

# 任务名称
修改 doing.md 模板 - debug.md 强制工作日志

# 任务目标
将 doing.md 模板中的 debug.md 记录从"遇到问题才记录"改为"强制工作日志"——无论任务是否顺利，都必须在 git commit 之前记录完整的执行过程。这是 learning 阶段能提取有价值 skills 的前提。

# 关键结果
1. 完成 internal/prompt/templates/doing.md 的修改：debug.md 定义为强制工作日志，移除"仅在遇到问题时记录"的表述
2. 新的 debug.md 记录格式包含四个必填部分：分析过程、实现步骤、遇到的问题（无则写"无"）、验证结果（含测试命令输出）
3. 行为约束中明确：**必须在 git commit 之前更新 debug.md**，这是硬约束而非建议
4. debug.md 文件路径明确为 `{doing_dir}/debug.md`（绝对路径在提示词中注入）
5. 保留原有的问题记录格式（Phenomenon/Reproduction/Hypothesis/Verification/Fix/Progress），作为"遇到问题时"的详细记录模板

# 测试方法
1. 检查 internal/prompt/templates/doing.md 文件，验证不再包含"遇到问题时才记录"等软性表述
2. 检查文件包含"强制"、"必须"等硬约束关键词
3. 检查新格式包含四个必填部分：分析过程、实现步骤、遇到的问题、验证结果
4. 检查"在 git commit 之前必须先更新 debug.md"的约束存在
5. 运行 `go build ./...` 验证 Go 代码编译正常（模板是 embedded，编译时包含）
