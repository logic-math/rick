# skill:global-ref-sync（修改核心名称前全局同步）

## 触发场景

修改一个在多个文件中被引用的核心名称/变量时：
- 重命名 skill 名称（如 `sense_skill_path` → `grilling_skill_path`）
- 更换模板变量名（如 `{{job_okr_content}}` → `{{loops_context}}`）
- 重命名 Go 函数/接口（如删除 `loadOKRPath`）

信号词：「将 X 替换为 Y」「重命名 X」「移除 X 的所有引用」

## 预期效果

- 一次规划所有需改动文件，不遗漏
- 避免"改了 A，跑测试才发现 B/C/D 也要改"的反复循环

## 核心内容

### 第 0 步（禁止跳过）：全局 grep 找出所有引用

```bash
# 在整个项目中查找旧名称
grep -rn "旧名称" internal/ .rick/ --include="*.go" --include="*.md" --include="*.py"
```

列出所有包含旧名称的文件，然后逐一规划修改。

### 第 1 步：规划改动文件清单

将 grep 结果按文件分组，确认每个文件的修改类型：Replace / Delete / Add

### 第 1.5 步（Edit 前）：精确定位目标行

Edit 工具要求 old_string 与文件内容完全匹配。直接构造 old_string 容易因行序/缩进不符导致失败。

**正确流程**：
1. `Read` 目标文件的精确行范围（用 offset/limit 缩窄）
2. 从 Read 输出中复制目标行，粘贴为 old_string
3. 再调用 Edit

**反模式**：凭记忆拼 old_string → Edit 报错 → 再 Read → 重试（浪费一轮）

### 第 2 步：按依赖顺序修改

推荐顺序（避免编译中断）：
1. 核心定义文件（Go 源码中的常量/函数名）
2. 模板文件（doing.md、plan.md 等 `.md` 模板）
3. 测试文件（`*_test.go`、`test*.py`）
4. 文档文件（`.rick/` 下的 .md 文件）

### 第 3 步：二次确认无遗漏

```bash
# 验证旧名称已彻底清除
grep -rn "旧名称" internal/ .rick/ 2>/dev/null | grep -v ".git"
# 应返回 0 行
```

### 第 4 步：build + 测试

```bash
go build ./... && go test ./internal/...
```
