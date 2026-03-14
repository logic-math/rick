# Rick 与 Claude Code CLI 集成说明

## Claude CLI 运行模式

Claude Code CLI 有两种主要运行模式：

### 1. 交互式模式（默认）
```bash
claude [options]
echo "prompt" | claude [options]
```
- 启动交互式会话
- 用户可以与 Claude 对话
- 需要连接 stdin/stdout/stderr 到终端
- 适用于：规划、探索、人机协作

### 2. 非交互式模式（`-p`/`--print`）
```bash
claude -p "prompt"
echo "prompt" | claude -p
```
- 打印响应后立即退出
- 不需要用户交互
- 适用于：自动化脚本、管道处理
- **注意**：跳过工作区信任对话框，仅在可信目录使用

## Rick 的使用场景

### Plan 命令（交互式）
```go
// 使用交互式模式，允许用户与 Claude 协作规划
cmd := exec.Command("claude", "--permission-mode", "plan")
stdin, _ := cmd.StdinPipe()
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr

cmd.Start()
stdin.Write([]byte(prompt))
stdin.Close()
cmd.Wait()
```

**特点**：
- 使用 `--permission-mode plan` 启用规划模式权限
- 通过管道传入初始提示词
- 用户可以继续与 Claude 交互
- 适合需要人类决策的规划阶段

### Doing 命令（非交互式）- 未来实现
```bash
# 参考 Morty 的实现
cat prompt_file | claude --dangerously-skip-permissions
```

**特点**：
- 使用 `--dangerously-skip-permissions` 跳过所有权限检查
- 通过管道传入提示词
- 完全自动化执行
- 适合自动化的执行阶段
- **仅在沙箱环境使用**

## 权限模式说明

| 模式 | 说明 | 使用场景 |
|------|------|----------|
| `default` | 默认模式，每次操作都询问 | 日常开发 |
| `plan` | 规划模式，允许文件探索和读取 | Rick plan 命令 |
| `acceptEdits` | 自动接受编辑，但询问其他操作 | 代码修改 |
| `dontAsk` | 不询问，但记录所有操作 | 受信任的操作 |
| `bypassPermissions` | 跳过权限检查（危险） | 测试环境 |
| `auto` | 根据上下文自动选择 | 智能模式 |

## Morty 的实现参考

### Plan 模式（交互式）
```bash
# morty_plan.sh
claude --permission-mode plan -p "$INTERACTIVE_PROMPT"
```
- `-p` 这里是传递 prompt 参数（不是 --print）
- 仍然是交互式的

### Doing 模式（非交互式）
```bash
# morty_doing.sh
cat "$prompt_file" | claude --dangerously-skip-permissions
```
- 通过管道传入提示词
- 使用 `--dangerously-skip-permissions` 自动执行

## Rick 的设计原则

1. **Plan 阶段**：
   - 交互式运行
   - 人类参与决策
   - 生成规划文档

2. **Doing 阶段**（未来）：
   - 非交互式运行
   - 自动化执行任务
   - 按照规划实施

3. **Learning 阶段**：
   - 人类主导
   - 总结经验教训
   - 更新知识库

## 常见问题

### Q: 为什么 Plan 使用交互式而不是 `-p` 模式？
A: 规划阶段需要人类的判断和决策，交互式模式允许：
- 澄清需求
- 讨论方案
- 调整计划
- 实时反馈

### Q: Doing 阶段为什么需要非交互式？
A: 执行阶段应该按照既定计划自动完成：
- 减少人工干预
- 提高执行效率
- 保证一致性
- 便于批量处理

### Q: `--dangerously-skip-permissions` 安全吗？
A: 仅在以下情况安全：
- ✅ 沙箱环境
- ✅ 无网络访问
- ✅ 受信任的代码库
- ❌ 生产环境
- ❌ 不受信任的输入

## 参考资料

- [Claude Code CLI 文档](https://claude.ai/code)
- [Morty 项目](../morty/)
- [Rick 设计文档](./docs/design.md)
