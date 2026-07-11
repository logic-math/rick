# skill:command-registration-verification（文档命令引用核实）

## 触发场景

在文档（README、commands.md、学习文档等）中引用项目自身的 CLI 命令、flags、子命令关系时使用：

- 写"命令体系"小节，列出所有命令
- 描述某命令的 flags、默认值、用法
- 描述命令间关系（如 A 是 B 的子模块、A 等价于 B）
- 信号词：「rick <cmd>」「命令 X 的 flag」「X 是 Y 的子模块」

**反模式信号**：凭记忆写出 `rick easy`、`rick doing --easy` 等命令结构而不读代码核实 → 大概率会幻觉命令关系。

## 预期效果

- 防止文档中命令关系与代码事实不一致（如把子模块误写为独立命令、把废弃 flag 仍写成可用）
- 一次写对，避免人类 review 时被纠正后返工
- 文档读者按命令用法执行时不会遇到"命令不存在"或"flag 已废弃"的问题

## 核心内容

### 第 0 步（禁止跳过）：读 root.go 的 AddCommand 清单

```bash
grep -n "AddCommand" internal/cmd/root.go
```

输出所有顶层命令注册点，确认命令是否真的存在。**不要凭命令名推测层级**。

### 第 1 步：读每个命令源文件的 cobra.Command 定义

```bash
# 读 Use / Short / Long / Args
Read internal/cmd/<cmd>.go
# 读 flags
grep -n "Flags()" internal/cmd/<cmd>.go
```

从 `cobra.Command{Use: ..., Long: ...}` 提取命令的真实用法描述，从 `cmd.Flags().XxxVar(...)` 提取所有 flags 的名称、类型、默认值、说明。

### 第 2 步：核实命令间关系

当文档要描述"A 是 B 的子模块"或"A 等价于 B"时：

1. 读 A 的 RunE 逻辑，看是否调用 B 的内部函数
2. 读 B 的 flag 注册，看是否包含 A 的入口 flag
3. 例：`rick easy` 与 `rick doing --easy`
   - easy.go 的 `runEasyMode` / `resumeEasyMode` 是核心函数
   - doing.go 的 `--easy` flag 在 RunE 中直接调用 `runEasyMode`
   - → 结论：`rick easy` 与 `rick doing --easy` 共用同一套函数，是等价入口

### 第 3 步：核实全局 flag

`--job`、`--dry-run`、`--verbose` 等可能在 root.go 定义为全局 flag，子命令通过 `GetJobID()` / `GetDryRun()` 读取：

```bash
grep -n "PersistentFlags\|GetJobID\|GetDryRun" internal/cmd/root.go
```

不要把全局 flag 误写为某子命令的局部 flag。

### 第 4 步：dry-run 验证

如果文档列出命令示例，可用 dry-run 实际跑一遍验证命令存在且 flag 正确：

```bash
./bin/rick <cmd> --dry-run 2>&1 | head -5
# 若命令不存在，cobra 会报 "unknown command"
# 若 flag 错误，cobra 会报 "unknown flag"
```

## 反例

本次 job_27 README 重写中，未读 root.go 直接写命令体系，将 `rick easy` 误判为"独立命令"（实际是 doing 的子模块，共用 easy 函数）。用户纠正后才补齐调研。本 skill 即从该教训中提取。
