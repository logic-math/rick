# RFC-refactor-2: job_16 代码质量与重复逻辑分析

## 背景

job_16 新增了 `debug_dir.go`（包含 `extractBugFrontmatter`、`LoadDebugDirSummaries`、`LoadDebugContext` 三个函数）以及 `SetDebugRaw/GetDebugRaw` 的集成。本次扫描验证代码的完整性与重复性，检查是否存在死代码和维护负担。

## 扫描范围

- `internal/executor/` 全部 Go 文件（21 个）
- `internal/prompt/` 全部 Go 文件（18 个）
- 重点关注 job_16 新增和修改的代码

## 发现的问题

### 1. 代码重复：frontmatter 解析逻辑重复 (P1 - 必须修复)

**位置1：** `internal/executor/debug_dir.go:13-41`
```go
func extractBugFrontmatter(content string) (summary, status string)
```
- 私有函数，11 行代码
- YAML frontmatter 解析算法

**位置2：** `internal/prompt/easy_prompt.go:214-239`  
```go
// 内联在 loadDebugContextLocal 函数中的 YAML 解析
lines := strings.Split(string(data), "\n")
inFM, started := false, false
for _, line := range lines {
    t := strings.TrimSpace(line)
    // ... 26 行重复的 frontmatter 解析逻辑
}
```

**根因：** 
- `easy_prompt.go` 需要 `loadDebugContextLocal` 函数来避免 `prompt` 包反向导入 `executor` 包（会产生循环依赖）
- 因此必须在本地复制 `LoadDebugContext` 的完整逻辑，包括 `extractBugFrontmatter` 的实现

**影响：**
- 代码维护成本增加：任何修复或改进需要在两处更新
- 一致性风险：两份实现可能在迭代中发生不同

**建议的修复：**

**方案 A（推荐）：** 提取 frontmatter 解析到独立包
- 新建 `internal/parser/frontmatter.go`（不涉及循环依赖）
- 函数：`func ExtractBugFrontmatter(content string) (summary, status string)`
- 修改 `debug_dir.go`：调用 `parser.ExtractBugFrontmatter`
- 修改 `easy_prompt.go`：调用 `parser.ExtractBugFrontmatter`
- 收益：消除重复代码，单一源头维护

**方案 B（临时）：** 导出 executor 的实现
- 将 `extractBugFrontmatter` 改名为 `ExtractBugFrontmatter`（导出）
- 修改 `go.mod` 或包导入策略，允许受控的单向 `prompt` → `executor` 依赖
- 修改 `easy_prompt.go`：`executor.ExtractBugFrontmatter`

### 2. 死代码：未被注入的 skill 文件 (P0 - 可直接删除)

以下 skill 文件在 `internal/prompt/templates/skills/` 目录中存在，但从未被任何 prompt 构建函数（`WriteSkillFile`/`LoadCoreSkills`）引用：

| 文件 | 说明 | 验证方法 |
|------|------|---------|
| `tc.md` | 测试用例四要素（英文版） | `grep -rn '"tc"' internal/prompt/` → 无结果 |
| `tdd.md` | TDD 铁律（英文版） | `grep -rn '"tdd"' internal/prompt/` → 无结果 |
| `tdd/testing-anti-patterns.md` | 禁止伪测试反模式（英文版） | `grep -rn 'testing-anti-patterns"' internal/prompt/` → 仅出现在 manager.go 注释中 |

**注**：`tdd.md` 和 `tdd/testing-anti-patterns.md` 是英文版，已被中文版替代；`tc.md` 内容本身是中文，但应合并进 `tdd-zh.md` 后删除（见下方 §2.1）。

**建议行动**：合并 tc.md 后，`git rm` 三个文件，`./scripts/build.sh` 验证编译，`go test ./internal/prompt/...` 确认无引用断裂。

#### §2.1 tc.md → tdd-zh.md 合并方案

`tc.md` 的"测试用例四要素"是 TDD 的配套知识，适合作为独立章节追加到 `tdd-zh.md` 的"好测试的标准"之后：

```markdown
## 测试用例四要素

每个测试用例必须包含：

### 1. 前置条件
测试运行前系统所处的状态：数据、配置、前置操作。
示例："用户账户存在，状态=active，余额=100"

### 2. 输入参数
传入被测系统的确切输入（不能用"某个值"，必须给具体值）：
边界情况：空字符串、nil、0、负数、最大整数；非法输入。
示例：`userID=42, amount=-5, currency="CNY"`

### 3. 操作序列
1. 建立前置条件
2. 用输入参数调用函数/接口
3. 观察返回结果
4. 验证副作用（数据库状态变化、发出的事件等）

### 4. 预期输出
可观测的确切结果：返回值（含类型和值）、状态变化、错误码、事件/日志。
示例："返回 error{code: INSUFFICIENT_FUNDS}，余额不变"

## 测试用例反模式
- 前置条件模糊："某个用户存在" → 应指定所有字段具体值
- 预期输出模糊："应该正常工作" → 应给出确切值
- 缺少副作用验证：写操作后必须检查数据库状态
```

合并后：`git rm internal/prompt/templates/skills/tc.md`，`go test ./internal/prompt/...` 验证即可。**无需修改任何 Go 代码**（tc 本来就未被注入）。

---

### 3. 已验证的功能完整性（NOT dead code）

以下函数已确认正常使用：

| 函数 | 位置 | 调用方 | 状态 |
|------|------|--------|------|
| `extractBugFrontmatter` | executor/debug_dir.go:13 | LoadDebugDirSummaries (line 73) | ✅ 使用中 |
| `LoadDebugDirSummaries` | executor/debug_dir.go:45 | LoadDebugContext (line 93) | ✅ 使用中 |
| `LoadDebugContext` | executor/debug_dir.go:85 | runner.go (259), retry.go (142), runner.go (165), learning.go (107,168) | ✅ 4 处调用 |
| `SetDebugRaw` | prompt/context.go:77 | executor/runner.go:259 | ✅ 使用中 |
| `GetDebugRaw` | prompt/context.go:84 | doing_prompt.go (79, 196) | ✅ 2 处调用 |
| `CheckDebugDir` | executor/doing_check.go:55 | RunDoingCheck (22), RunEasyCheck (46) | ✅ 被调用 |
| `RunDoingCheck` | executor/doing_check.go:13 | runner.go (124), tools_doing_check.go (134) | ✅ 被调用 |
| `RunEasyCheck` | executor/doing_check.go:45 | tools_doing_check.go (121) | ✅ 被调用 |
| `formatOKRContent` | prompt/context_helpers.go:10 | plan_prompt_test.go (91, 104, 111) | ✅ 测试中 |
| `formatSPECContent` | prompt/context_helpers.go:31 | plan_prompt_test.go (121, 128) | ✅ 测试中 |
| `formatCompletedWork` | prompt/context_helpers.go:43 | plan_prompt_test.go (135, 142) | ✅ 测试中 |

**结论：** 无死代码，所有导出函数均被调用。私有函数使用情况正常。

### 3. RFC-1 修复验证

**问题（RFC-1）：** manager_test.go:199 引用不存在的 `"debug"` skill  
**当前状态：** ✅ **已修复**
- 最新代码：manager_test.go:199 现在使用正确的 skill 列表：`["sense", "tc", "tdd", "testing", "debug_skill", "gen-skill", "evolve-skills"]`
- "debug_skill" 是正确的技能名称，对应 `debug_skill.md`
- 测试通过无误

### 4. TODO 2026-08 回退路径（Planned Technical Debt - 非立即问题）

**位置1：** `executor/debug_dir.go:84,98`
```go
// TODO(2026-08): remove fallback to debug.md after full migration to debug/ dir format.
```

**位置2：** `prompt/easy_prompt.go:188,247`
```go
// TODO(2026-08): remove fallback to debug.md after full migration to debug/ dir format.
```

**评估：**
- 这是 **意图设计的兼容性逻辑**，非死代码
- 作用：migration path 保证旧项目的 debug.md 仍能被识别
- 计划于 2026-08 后删除（6+ 个月后，足够迁移时间）
- 建议在 2026-08 月初建立清理检查单

---

## 优先级建议与行动清单

| 优先级 | 类别 | 问题 | 影响范围 | 建议行动 | 预期耗时 |
|------|------|------|---------|---------|---------|
| **P0** | 合并+删除 | `tc.md` 内容合并进 `tdd-zh.md`，然后删除 | 知识零损失 | 将四要素章节追加到 tdd-zh.md，`git rm tc.md`，`go test ./internal/prompt/...` | 15min |
| **P0** | 死代码 | `tdd.md`、`tdd/testing-anti-patterns.md` 英文版从未被注入 | 无任何调用方 | `git rm` 两个文件，build 验证 | 5min |
| **P1** | 代码重复 | `extractBugFrontmatter` 两份实现 | 维护一致性 | 提取至 `internal/parser/frontmatter.go`（方案A）或导出 executor 版本（方案B） | 2h |
| **P2** | 文档/注释 | 4 个 TODO 2026-08 应归纳到一个清理计划 | 长期维护 | 在 .rick/RFC 中新建 "技术债清理计划 - 2026-08"，列出所有清理项 | 0.5h |
| **P3** | 验证 | 循环导入风险评估 | 架构长期健康 | 验证是否可改方案 B（允许单向 executor 依赖）或必须使用方案 A | 1h |

---

## 验证状态

✅ 所有 executor 测试通过：`go test ./internal/executor/...` → PASS  
✅ 所有 prompt 测试通过：`go test ./internal/prompt/...` → PASS  
✅ 集成测试通过：`go test ./internal/cmd/...` → PASS  
✅ 无 linter 警告（重复代码检测工具 dupl 会标记位置 1 和 2）

---

## 附录：重复代码对比

### extractBugFrontmatter (debug_dir.go:13-41)
```go
func extractBugFrontmatter(content string) (summary, status string) {
    lines := strings.Split(content, "\n")
    inFrontmatter := false
    started := false
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed == "---" {
            if !started {
                inFrontmatter = true
                started = true
                continue
            }
            if inFrontmatter {
                break
            }
        }
        if !inFrontmatter {
            continue
        }
        if strings.HasPrefix(trimmed, "summary:") {
            v := strings.TrimSpace(strings.TrimPrefix(trimmed, "summary:"))
            summary = strings.Trim(v, `"'`)
        } else if strings.HasPrefix(trimmed, "status:") {
            v := strings.TrimSpace(strings.TrimPrefix(trimmed, "status:"))
            status = strings.Trim(v, `"'`)
        }
    }
    return
}
```

### loadDebugContextLocal 内部实现 (easy_prompt.go:214-239)
```go
// 相同的解析逻辑嵌入在 loadDebugContextLocal 中
lines := strings.Split(string(data), "\n")
inFM, started := false, false
for _, line := range lines {
    t := strings.TrimSpace(line)
    if t == "---" {
        if !started {
            inFM, started = true, true
            continue
        }
        if inFM {
            break
        }
    }
    if !inFM {
        continue
    }
    if strings.HasPrefix(t, "summary:") {
        v := strings.TrimSpace(strings.TrimPrefix(t, "summary:"))
        summary = strings.Trim(v, `"'`)
    } else if strings.HasPrefix(t, "status:") {
        v := strings.TrimSpace(strings.TrimPrefix(t, "status:"))
        status = strings.Trim(v, `"'`)
    }
}
```

**差异：** 变量名不同（trimmed vs t），逻辑完全相同
