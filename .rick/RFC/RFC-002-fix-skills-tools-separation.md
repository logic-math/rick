# RFC-002: 修复 skills/tools 分离实现与 RFC-001 规范的偏差

**状态**: 草案  
**作者**: AI（通过 job_11 learning 阶段发现）  
**日期**: 2026-04-07  
**AI 不可修改**

---

## 一、问题定义（Subject）

### 现状

RFC-001 第五节明确定义了 tools 和 skills 的分工：

| 概念 | 位置 | 格式 | 作用 |
|------|------|------|------|
| tools | 项目根目录 `tools/` | Python 脚本 | 确定性工具，agent 直接执行 |
| skills | `.rick/skills/` | **Markdown 文件** | 描述如何组合使用 tools |

**但当前实现完全偏离了这个规范：**

1. **`.rick/skills/` 里放的全是 `.py` 文件**，不是 Markdown：
   ```
   .rick/skills/
   ├── check_go_build.py          ← 应在 tools/
   ├── check_prompt_variables.py  ← 应在 tools/
   ├── check_variadic_api.py      ← 应在 tools/
   ├── mock_agent_testing.py      ← 应在 tools/
   ├── rick_tools_check_pattern.py← 应在 tools/ 或删除
   ├── index.md                   ← 正确
   └── *.md (少量文档)             ← 正确，但被 .py 文件淹没
   ```

2. **`tools/` 目录根本不存在**（项目根目录下无 `tools/`）

3. **代码实现已经正确区分了两者**，但数据层没有跟上：
   - `internal/workspace/tools.go`：扫描 `{projectRoot}/tools/*.py`，注入"可直接执行的工具"
   - `internal/workspace/skills.go`：扫描 `.rick/skills/index.md`，注入"技能说明书"
   - `doing_prompt.go` 和 `plan_prompt.go`：分别注入 tools section 和 skills index section

   由于 `tools/` 不存在，tools 注入永远为空；由于 `.rick/skills/` 里是 `.py` 而非 Markdown，skills 注入的是 Python 脚本列表而非组合技能说明书。

### 问题根源

从 job_1 到 job_11，learning 阶段一直将"可执行脚本"沉淀到 `.rick/skills/`，这是因为 learning 提示词模板没有明确区分 tools 和 skills 的格式与位置，导致 AI 把两者混为一谈。

---

## 二、影响范围

### 对 agent 的影响

doing agent 和 plan agent 的 prompt 中：
- **tools section**：永远为空（因为 `tools/` 不存在）→ agent 不知道有哪些可复用工具
- **skills section**：注入的是 `.py` 文件列表（因为 `formatSkillsSection` fallback 到扫描 `.py`）→ agent 看到的是脚本名称，而不是"在什么场景下组合哪些工具"的说明书

这正是 RFC-001 第一节描述的问题：**learning 沉淀的 skills 没有被后续 job 使用**。

### 对 learning 的影响

learning 阶段产出方向错误：
- 一直在生产 `.py` 脚本放进 `.rick/skills/`
- 从未产出过真正的 Markdown 技能说明书
- 从未产出过 `tools/` 目录下的工具脚本

---

## 三、修复方案

### 3.1 数据迁移

将 `.rick/skills/` 下所有 `.py` 文件迁移到项目根目录 `tools/`：

```
tools/
├── build_and_get_rick_bin.py    # 从 .rick/skills/ 迁移（job_11 新增）
├── check_go_build.py            # 从 .rick/skills/ 迁移
├── check_prompt_variables.py    # 从 .rick/skills/ 迁移
├── check_variadic_api.py        # 从 .rick/skills/ 迁移
└── mock_agent_testing.py        # 从 .rick/skills/ 迁移
```

注意：`rick_tools_check_pattern.py` 是"模式文档"，不是工具，应删除。

### 3.2 `.rick/skills/` 重建为 Markdown 技能说明书

`.rick/skills/` 只保留 Markdown 文件，每个文件描述一个组合技能场景：

```
.rick/skills/
├── index.md                         # 必须：所有 skill 的名称和触发场景
├── verify_rick_check_commands.md    # 示例：如何验证 rick check 类命令
└── test_go_project_changes.md       # 示例：如何测试 Go 项目变更
```

**index.md 格式**（触发场景列必须填写，这是 agent 决策的依据）：

```markdown
# Skills Index

| Skill | 描述 | 触发场景 |
|-------|------|----------|
| verify_rick_check_commands.md | 验证 rick check 命令行为是否符合预期 | 当任务涉及修改 rick tools check 相关代码时 |
| test_go_project_changes.md | 测试 Go 项目代码变更 | 当任务涉及修改 Go 源码时 |
```

**skill Markdown 文件格式**：

```markdown
# <技能名称>

## 触发场景
描述什么情况下使用这个技能

## 使用的 Tools
- `tools/build_and_get_rick_bin.py`：用途
- `tools/check_go_build.py`：用途

## 执行步骤
1. 先运行 `python3 tools/build_and_get_rick_bin.py` 获取本地构建路径
2. 用返回的路径运行 `{bin_path} tools plan_check job_N`
3. ...

## 示例
具体调用示例
```

### 3.3 更新 learning 提示词模板

在 `internal/prompt/templates/learning.md` 中明确区分两类产出：

- **tools**（`tools/*.py`）：确定性操作，无需上下文，可反复调用
- **skills**（`.rick/skills/*.md`）：组合技能说明书，描述如何在特定场景下组合 tools

### 3.4 更新 `formatSkillsSection`

`doing_prompt.go` 中的 `formatSkillsSection` 当前有 fallback 逻辑：当 `index.md` 不存在时扫描 `.py` 文件。修复后 `.rick/skills/` 不再有 `.py` 文件，此 fallback 可保留但实际不会触发。

---

## 四、验收标准

1. `tools/` 目录存在，包含所有迁移过来的 `.py` 工具脚本
2. `.rick/skills/` 只含 `.md` 文件，无 `.py` 文件
3. `.rick/skills/index.md` 的"触发场景"列非空，每个 skill 都有明确触发条件
4. `rick doing job_N --dry-run` 输出中：
   - tools section 非空（列出 `tools/` 下的工具）
   - skills section 显示的是 Markdown skill 名称和触发场景，而非 `.py` 文件列表
5. learning 提示词中明确区分 tools 和 skills 的产出格式

---

## 五、关键假设

| 假设 | 说明 |
|------|------|
| `.rick/skills/*.py` 中的脚本逻辑本身是正确的 | 只需迁移位置，不需要重写逻辑 |
| `tools/` 目录在 `doing` 执行时的 cwd 是项目根目录 | 已验证：`LoadToolsList` 从 `os.Getwd()` 扫描 |
| `formatSkillsSection` 的 index.md 优先路径已正确实现 | 已验证：`LoadSkillsIndex` 优先读 index.md |
