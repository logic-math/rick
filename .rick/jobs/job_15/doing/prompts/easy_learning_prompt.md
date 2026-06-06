# Rick Easy Mode Learning

你是一个资深技术专家，对本次 easy 会话的执行过程进行学习和知识沉淀。

## 执行上下文

**Job**: job_15（easy 模式）

### 数据来源（请读取以下文件）

- **debug.md（行为轨迹与问题记录）**: `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_15/doing/debug.md`
- **OKR**: `/Users/sunquan/ai_coding/CODING/rick/.rick/OKR.md`
- **SPEC.md**: `/Users/sunquan/ai_coding/CODING/rick/.rick/SPEC.md`

---

## ⚠️ 执行 SOP

### Step 1：读取并分析 debug.md

读取 debug.md 文件，分析：
- 每个 debug 条目的根因与解决方案
- 跨问题的共性模式
- 未解决的问题

### Step 2：提取可复用 Tools

**YOU MUST declare: "I will use skill:gen-skill." Before writing any tool.**

从 debug.md 中识别可复用模式，提取为 Python 工具：
- ✅ 纯函数，确定性输入输出
- ✅ 跨场景通用
- ✅ 支持 --test 自验证

直接写入：`/Users/sunquan/ai_coding/CODING/rick/.rick/tools/*.py`

### Step 3：沉淀 Skills（wiki 文档）

为每个可复用模式生成 wiki 文档（触发场景/预期效果/使用方法）。

直接写入：`/Users/sunquan/ai_coding/CODING/rick/.rick/wiki/*.md`

### Step 4：更新 SPEC.md

直接更新 `/Users/sunquan/ai_coding/CODING/rick/.rick/SPEC.md`（in-place），将新 wiki 文档注册到技能列表，SPEC ≤ 512 行。

### Step 5：生成 SUMMARY.md

在 `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_15/learning` 下生成 SUMMARY.md：

`APPROVED: true` 开头，包含执行概述、关键成就、问题教训、知识沉淀清单。

### Step 6：运行 learning_check

```bash
/Users/sunquan/ai_coding/CODING/rick/bin/rick tools learning_check job_15
```

失败则修复后重新运行。

---

## ⚠️ 约束

1. 必须先读取 debug.md 再生成 SUMMARY.md
2. Step 2 必须声明使用 gen-skill
3. wiki/tools/SPEC 直接写入 .rick/：`/Users/sunquan/ai_coding/CODING/rick/.rick/wiki`、`/Users/sunquan/ai_coding/CODING/rick/.rick/tools`、`/Users/sunquan/ai_coding/CODING/rick/.rick/SPEC.md`
4. SUMMARY.md 写入 learning 目录：`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_15/learning`
