---
name: pi-extension-install-verification
description: pi install 扩展后，必须用 pi list + 真实工具调用双重验证扩展真生效
---

# Skill: pi 扩展安装的真实生效验证

## 触发场景

当通过 `pi install npm:<pkg>` 或 `pi install <local-path>` 安装 pi 扩展（如 subagent、web-access、主题包）后使用。

**问题信号**：
- `pi list` 显示扩展已装，但 LLM 报 "no <tool> tool available"
- 扩展源是本地目录（非 npm 包），无 package.json
- 安装后没跑过真实 LLM 调用确认工具注册

## 预期效果

- 捕获"pi install 假成功"：写入 settings.json 但 pi loader 不认（如本地 .ts 源码无 package.json）
- 确认扩展工具真注册到 pi 工具表（LLM 能看到并调用）
- 防止 rick init-pi 报"已装"但实际 subagent/web_search 工具不可用

## 核心内容

### 1. 双重验证（缺一不可）

```bash
# (a) 注册验证: pi list 含包名
pi list | grep <pkg>

# (b) 工具生效验证: 真实跑一次, 让 pi 调用该工具
DEEPSEEK_API_KEY=... pi --provider deepseek --model deepseek-v4-flash --mode json \
  -p 'Use the <tool> tool to <do X>. Do not use bash.' 2>&1 \
  | grep '"toolName":"<tool>"'
# 有输出 = 工具真注册; 0 输出 = 假装成功
```

### 2. 本地源码扩展的坑

pi 的 loader 只认两种入口：
- `package.json` 含 `pi.extensions` 字段
- `~/.pi/agent/extensions/<name>/index.ts` 子目录结构

`pi install <bare-source-dir>`（无 package.json 的本地 .ts 目录）会**写 settings.json 但 loader 不加载** → `pi list` 假装装上，工具从未注册。

**对策**：扩展优先用 npm 包（`pi install npm:<pkg>`），不用本地 example 源码。若必须用本地源码，按其 README 的 symlink 方式装到 `~/.pi/agent/extensions/<name>/`。

### 3. init-pi 类命令的最终验证步骤

安装多个扩展后，末尾跑一次 `pi list` 全量确认：
```go
// 验证所有必需扩展都真注册
for _, pkg := range requiredExtensions {
    if !piListContains(pkg) { missing = append(missing, pkg) }
}
```
有 missing 则 warn（不 fatal，rick 仍可用但功能降级）。

### 4. 真实工具调用校准（可选但推荐）

校准解析器时，用真实 LLM 工具调用流（非 mock）确认字段映射：
```bash
# 后台跑真实工具调用, tee 捕获事件流
pi --mode json -p "Read /tmp/x.txt" 2>&1 | tee /tmp/real.jsonl
grep "tool_execution_start\|tool_execution_end" /tmp/real.jsonl
# 对照解析器字段 (toolCallId/toolName/args/result/isError)
```
