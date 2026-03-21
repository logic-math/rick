## task2: 修改 doing.md 模板 - debug.md 强制工作日志

**分析过程 (Analysis)**:
- 读取了 internal/prompt/templates/doing.md 当前内容
- 发现模板已在上一次提交（feat(prompt): make debug.md a mandatory work log in doing template）中完成了所需改动
- 确认所有 5 个关键结果均已满足：强制工作日志定义、四个必填部分、硬约束表述、路径变量、原问题格式保留

**实现步骤 (Implementation)**:
1. 读取 internal/prompt/templates/doing.md 验证当前状态
2. 对照任务关键结果逐条核查
3. 运行 `go build ./...` 确认编译正常
4. 运行 grep 验证无"遇到问题才记录"软性表述，确认"强制"/"必须"硬约束关键词存在

**遇到的问题 (Issues)**:
- 无

**验证结果 (Verification)**:
- 测试命令：`go build ./...`
- 测试输出：
  ```
  （无输出，编译成功）
  ```
- 关键词验证：`grep -n "强制\|必须"` 找到 5 处硬约束表述
- 结论：✅ 通过
