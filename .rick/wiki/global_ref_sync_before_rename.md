---
name: global_ref_sync_before_rename
description: 修改框架核心字符串/接口名时，先全局 grep 找出所有引用再批量更新的工作流
type: skill
---

# 全局引用同步（修改核心名称时）

## 触发场景

当需要修改一个在多个文件中被引用的名称时：
- 重命名 skill 名称（如 `阶段一: 源码推理法` → `Phase 1: 构建反馈回路`）
- 更换模板变量名（如 `{{job_okr_content}}` → `{{loops_context}}`）
- 重命名 Go 函数/接口（如 `loadOKRPath` 删除）

信号词：「将 X 替换为 Y」「重命名 X」「移除 X 的所有引用」

## 预期效果

- 一次规划所有需改动的文件，不遗漏
- 避免"改了 A，跑测试才发现 B/C/D 也要改"的反复循环

## 使用方法

### 第 0 步（必须先执行）：全局 grep 找出所有引用位置

```bash
# 找出旧名称在整个项目中的所有引用
grep -rn "旧名称" internal/ .rick/ --include="*.go" --include="*.md" --include="*.py"
```

**禁止跳过此步**：先列出所有文件，再逐一规划修改顺序。

### 第 1 步：规划改动文件清单

将 grep 结果按文件分组，确认每个文件需要的修改类型（Replace/Delete/Add）。

### 第 2 步：按依赖顺序批量修改

建议顺序（避免编译中断）：
1. 核心定义文件（如 doing_check.go 中的校验字符串）
2. 模板文件（如 doing.md、easy.md）
3. 测试文件（如 tools_test.go、做_prompt_test.go）
4. 最后 build + 运行测试确认全绿

### 第 3 步：二次确认无遗漏

```bash
# 验证旧名称已彻底清除
grep -rn "旧名称" internal/ .rick/ 2>/dev/null | grep -v ".git"
# 应返回 0 行
```
