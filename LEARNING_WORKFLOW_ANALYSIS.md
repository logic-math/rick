# Learning 命令流程对比分析

## 设计流程（你的描述）

```
用户命令：rick learning job_n
输出：
- .rick/OKR.md、SPEC.md、wiki/、skills/

执行流程：
1. 读取doing/debug.md（本Job的问题记录）
2. 读取git历史（本Job的所有变更,通过 git diff 读取具体的变更代码）
3. 分析这两个信息源，理解项目发生了什么变化
4. 基于理解，Agent与人类进行对话，逐步确定：
  - OKR更新：这个Job对项目长期目标的影响
  - SPEC更新：是否需要调整开发规范
  - Wiki更新：是否有新的系统运行原理需要记录
  - Skills沉淀：是否有可复用的技能需要提取
5. 生成learning/目录下的OKR.md、SPEC.md、wiki/、skills/
6. 人类审核并调整这些内容
7. 最后调用merge skill，将learning/中的内容与.rick/全局目录合并：
  - OKR.md与.rick/OKR.md合并（去重、冲突解决）
  - SPEC.md与.rick/SPEC.md合并
  - wiki/内容与.rick/wiki/合并
  - skills/与.rick/skills/合并
```

## 当前实现

```
执行流程：
1. ✅ 读取doing/debug.md
2. ✅ 读取git历史（git log，但没有 git diff）
3. ✅ 生成learning prompt
4. ⚠️  调用Claude Code CLI（交互式，但不是对话式）
5. ❌ 直接更新.rick/OKR.md和SPEC.md（没有先生成到learning/）
6. ❌ 没有人类审核步骤
7. ❌ 没有merge skill
8. ❌ 没有wiki/和skills/支持
```

## 详细对比

| 步骤 | 设计要求 | 当前实现 | 状态 | 说明 |
|------|---------|---------|------|------|
| **1. 读取debug.md** | ✅ 读取doing/debug.md | ✅ 已实现 | ✅ 符合 | `loadExecutionResults()` 第206-210行 |
| **2. 读取Git历史** | ✅ git diff 读取变更代码 | ⚠️ 仅 git log | ⚠️ 部分符合 | 只有commit列表，没有具体代码变更 |
| **3. 分析变化** | ✅ 理解项目变化 | ✅ 生成prompt | ✅ 符合 | `generateLearningPrompt()` |
| **4. 对话式确定** | ✅ Agent与人类对话 | ⚠️ 单次交互 | ⚠️ 不符合 | 使用交互式CLI，但不是对话式 |
| **5. 生成到learning/** | ✅ 生成4个文件 | ❌ 只生成summary | ❌ 不符合 | 缺少OKR.md、SPEC.md、wiki/、skills/ |
| **6. 人类审核** | ✅ 人类审核并调整 | ❌ 无审核步骤 | ❌ 不符合 | 直接更新全局文件 |
| **7. Merge skill** | ✅ 智能合并 | ❌ 直接append | ❌ 不符合 | 简单追加，无去重、冲突解决 |
| **输出结构** | OKR/SPEC/wiki/skills | 仅OKR/SPEC | ❌ 不符合 | 缺少wiki/和skills/ |

## 关键问题

### ❌ 问题 1：缺少 Git Diff
**设计要求**：读取具体的变更代码（`git diff`）
**当前实现**：只读取 commit 列表（`git log`）

```go
// 当前实现 (learning.go:218-226)
gm := git.New(cwd)
commits, err := gm.GetLog(20)
// 只获取了 commit hash 和 message，没有 diff
```

**应该改为**：
```go
// 获取本 Job 的所有 commit
commits := getJobCommits(jobID)

// 对每个 commit 获取 diff
for _, commit := range commits {
    diff := git.GetDiff(commit.Hash)
    // 包含具体的代码变更
}
```

### ❌ 问题 2：不是对话式交互
**设计要求**：Agent 与人类进行对话，逐步确定更新内容
**当前实现**：单次调用 Claude CLI，没有多轮对话

```go
// 当前实现 (learning.go:289-320)
cmd := exec.Command("claude", "code", tmpFile.Name())
cmd.Run()
// 单次交互，不是对话式
```

**应该改为**：
```go
// 对话式交互
for {
    // 1. Agent 提出问题
    question := agent.Ask("这个Job对OKR有什么影响？")

    // 2. 人类回答
    answer := promptUser(question)

    // 3. Agent 基于回答继续
    if agent.IsComplete() {
        break
    }
}
```

### ❌ 问题 3：直接更新全局文件
**设计要求**：先生成到 `learning/`，人类审核后再合并到 `.rick/`
**当前实现**：直接 append 到 `.rick/OKR.md` 和 `SPEC.md`

```go
// 当前实现 (learning.go:322-343)
// 直接更新全局文件
okriPath := filepath.Join(rickDir, "OKR.md")
appendToFile(okriPath, content)  // ❌ 直接追加
```

**应该改为**：
```
1. 生成 learning/OKR.md
2. 生成 learning/SPEC.md
3. 生成 learning/wiki/*.md
4. 生成 learning/skills/*.md
5. 人类审核
6. 调用 merge skill 合并
```

### ❌ 问题 4：缺少 wiki/ 和 skills/
**设计要求**：支持 4 种输出（OKR、SPEC、wiki、skills）
**当前实现**：只支持 OKR 和 SPEC

```go
// 当前实现：没有 wiki 和 skills 的处理
```

**应该添加**：
```go
// 生成 wiki 文档
wikiContent := extractWikiContent(learningResult)
saveToLearning(learningDir, "wiki", wikiContent)

// 提取 skills
skillsContent := extractSkills(learningResult)
saveToLearning(learningDir, "skills", skillsContent)
```

### ❌ 问题 5：简单追加，无智能合并
**设计要求**：merge skill 进行去重、冲突解决
**当前实现**：简单的 `appendToFile()`

```go
// 当前实现 (learning.go:399-412)
func appendToFile(filePath string, content string) error {
    file.WriteString(content)  // ❌ 简单追加，无去重
}
```

**应该改为**：
```go
// 智能合并
func mergeOKR(globalOKR, learningOKR string) string {
    // 1. 解析两个文件
    // 2. 去重
    // 3. 解决冲突
    // 4. 合并
}
```

### ❌ 问题 6：缺少人类审核步骤
**设计要求**：人类审核并调整 learning/ 中的内容
**当前实现**：没有审核步骤，直接更新

**应该添加**：
```go
// 生成到 learning/ 后
fmt.Println("Learning 文件已生成到:", learningDir)
fmt.Println("请审核以下文件：")
fmt.Println("  - learning/OKR.md")
fmt.Println("  - learning/SPEC.md")
fmt.Println("  - learning/wiki/")
fmt.Println("  - learning/skills/")

if !promptForApproval() {
    return nil  // 用户取消
}

// 用户批准后才合并
mergeToGlobal(learningDir, rickDir)
```

## 正确的流程应该是

```go
func executeLearningWorkflow(jobID string) error {
    // Step 1: 读取 debug.md
    debugContent := readDebugMd(jobID)

    // Step 2: 读取 Git 变更（包含 diff）
    gitChanges := getGitChangesWithDiff(jobID)

    // Step 3: 生成初始分析
    analysis := analyzeChanges(debugContent, gitChanges)

    // Step 4: 对话式确定更新内容
    updates := dialogWithHuman(analysis)
    // 这里应该是多轮对话：
    // - Agent: "这个Job实现了用户认证，是否需要更新OKR？"
    // - Human: "是的，添加到安全性目标下"
    // - Agent: "是否需要记录认证流程到wiki？"
    // - Human: "是的，创建 authentication.md"

    // Step 5: 生成到 learning/ 目录
    generateToLearning(learningDir, updates)
    // 生成：
    // - learning/OKR.md
    // - learning/SPEC.md
    // - learning/wiki/authentication.md
    // - learning/skills/oauth2_integration.md

    // Step 6: 人类审核
    fmt.Println("请审核 learning/ 目录中的文件")
    if !waitForApproval() {
        return nil
    }

    // Step 7: 调用 merge skill
    mergeToGlobal(learningDir, rickDir)
    // 智能合并，去重，解决冲突

    // Step 8: Commit
    commitLearningResults(jobID)
}
```

## 修复建议

### 优先级 P0（核心流程）

1. **添加 Git Diff 读取**
   ```go
   func getGitChangesWithDiff(jobID string) (*GitChanges, error) {
       // 获取本 Job 的所有 commit
       // 对每个 commit 执行 git show 获取 diff
   }
   ```

2. **实现对话式交互**
   ```go
   func dialogWithHuman(analysis *Analysis) (*Updates, error) {
       // 使用交互式提示，逐步确定：
       // - OKR 更新
       // - SPEC 更新
       // - Wiki 内容
       // - Skills 提取
   }
   ```

3. **先生成到 learning/，再合并**
   ```go
   func generateToLearning(learningDir string, updates *Updates) error {
       // 生成 learning/OKR.md
       // 生成 learning/SPEC.md
       // 生成 learning/wiki/*.md
       // 生成 learning/skills/*.md
   }
   ```

4. **添加人类审核步骤**
   ```go
   func waitForApproval() bool {
       fmt.Println("审核完成后，按 y 继续合并，按 n 取消")
       // 等待用户输入
   }
   ```

5. **实现 merge skill**
   ```go
   func mergeToGlobal(learningDir, rickDir string) error {
       mergeOKR(...)
       mergeSPEC(...)
       mergeWiki(...)
       mergeSkills(...)
   }
   ```

### 优先级 P1（增强功能）

6. **添加 wiki/ 支持**
   ```go
   func generateWiki(learningDir string, content string) error {
       wikiDir := filepath.Join(learningDir, "wiki")
       os.MkdirAll(wikiDir, 0755)
       // 生成 wiki 文档
   }
   ```

7. **添加 skills/ 支持**
   ```go
   func extractSkills(learningResult string) []Skill {
       // 从 learning result 中提取可复用的技能
   }
   ```

### 优先级 P2（优化）

8. **智能去重和冲突解决**
   ```go
   func mergeOKR(globalOKR, learningOKR string) string {
       // 1. 解析 Markdown
       // 2. 检测重复内容
       // 3. 解决冲突
       // 4. 合并
   }
   ```

## 总结

| 类别 | 符合度 | 说明 |
|------|--------|------|
| **核心流程** | ⚠️ 30% | 基本框架存在，但关键步骤缺失 |
| **数据读取** | ⚠️ 60% | 有 debug.md 和 git log，缺 git diff |
| **交互方式** | ❌ 0% | 不是对话式，是单次交互 |
| **输出结构** | ❌ 25% | 只有 OKR/SPEC，缺 wiki/skills |
| **审核机制** | ❌ 0% | 没有人类审核步骤 |
| **合并策略** | ❌ 10% | 简单追加，无智能合并 |

**总体评估**：当前实现与设计流程**严重不符**，需要大幅重构。

## 推荐方案

建议采用**分阶段重构**：

### 阶段 1：修复核心流程（1-2天）
- 添加 Git Diff 读取
- 先生成到 learning/，再合并
- 添加人类审核步骤

### 阶段 2：实现对话式交互（2-3天）
- 设计对话流程
- 实现多轮问答
- 逐步确定更新内容

### 阶段 3：完善输出结构（1-2天）
- 添加 wiki/ 支持
- 添加 skills/ 支持
- 完善文件结构

### 阶段 4：实现智能合并（2-3天）
- 实现 merge skill
- 去重算法
- 冲突解决策略

---

**分析日期**: 2026-03-14
**分析人**: Claude Opus 4.6
