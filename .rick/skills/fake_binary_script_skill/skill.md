# skill:fake-binary-script（fake 可执行脚本测试陷阱）

## 触发场景

当 Go/Python 测试中用**假的可执行脚本**（fake pi、fake node 等）模拟真实二进制时使用：
- 测试把 PATH 替换为只含 fake bin 的临时目录（`t.Setenv("PATH", tmp)`）
- fake 脚本输出被 `cmd.Output()` 捕获或 `2>/dev/null` 吞掉，失败原因不可见
- 现象：fake 脚本"部分命令生效、部分不生效"（如 install 分支正常、list 分支静默失败）

**问题信号**：「list 输出为空但 install 正常」「脚本单独跑没问题、测试里不行」「Output() 无错误但内容缺失」

## 预期效果

- 一次写对 fake 脚本，不再被"静默 command not found"耗掉数轮调试
- 明确 PATH 替换下哪些命令可用（内建）哪些不可用（外部）

## 核心内容

### 根因：PATH 被替换后，脚本里的外部命令找不到

`t.Setenv("PATH", tmp)` 让 PATH **只含 fake bin 目录**：
- **shell 内建**（echo/read/case/printf）✅ 始终可用
- **外部命令**（cat/ls/grep/sed）❌ `command not found` —— 若再被 `2>/dev/null` 或 `cmd.Output()`（吞 stderr）静默，表现为"输出为空但无报错"

典型例子：fake pi 的 `list` 分支用 `cat "$LIST_FILE"`，PATH 替换后 cat 找不到 → 空输出 → 测试断言失败；而 `install` 分支用 `echo`（内建）正常 —— 造成"install 行、list 不行"的迷惑现象。

### 修复 1：脚本内恢复系统 PATH（首选）

```go
script := `#!/bin/sh
export PATH=/usr/bin:/bin:/usr/sbin:/sbin:$PATH   // 关键：fake 脚本自己恢复 PATH
case "$1" in
  list) cat "$FAKE_LIST" 2>/dev/null;;
  install) echo "$2" >> "$FAKE_LIST";;
esac
`
```

### 修复 2：只用 shell 内建（无外部命令依赖）

```sh
list) while IFS= read -r line; do echo "$line"; done < "$FAKE_LIST";;
```

### 调试技巧

- 怀疑 fake 脚本问题时，先把 `2>/dev/null` 去掉或加 `>&2` 诊断输出（`cmd.Output()` 的 stderr 只出现在 error 里）
- 用 `sh -x ./fake ...` 独立跑脚本验证（`+ echo ...` 追踪每一步）
- 确认测试 PATH：`echo "$PATH" | tr ':' '\n'`
