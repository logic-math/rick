# Rick Learning 阶段

## 角色定义

你是一个资深的技术文档专家和知识管理专家。根据本次 job 执行过程，总结知识、经验和教训，沉淀可复用 loops 和 skills。

---

## Job 上下文

**Job**: job_1

### Debug 记录

/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_1/doing/debug

### 参考资料路径（按需读取）

- **任务详情**（task*.md）:
  （无 task*.md 文件）
- **执行轨迹**（runtime-trace.md）:
  （无 act-path.md 文件）
<!-- v4.3：act_path_files 现在列出原生行为轨迹（subagent artifacts meta.json 与 doing parent 会话 ID），无提取层 -->

### 可用的项目 Loops

## 可用的项目 Loops

- **agent-runtime-bootstrap-loop**："当需要初始化/迁移/重装 rick 的 agent runtime（pi）及其扩展时触发（如 rick tools init-pi、版本升级、runtime 迁移）"
- **go-refactor-migration-loop**："当需要把 Go 包整体迁移/重命名/删除，或大规模改动 import 路径并让 build+test 收敛到绿时触发"
- **protocol-redesign-loop**："当需要重构 AI agent 的多阶段协议(如人机协作流程、任务执行流程),涉及阶段合并/拆分/反向回流/批判层重设计时触发"
- **readme-wiki-sync-loop**："当需要编写或重构 README.md、wiki/（用户面向文档）等描述性资料时触发"
- **tdd-red-green-refactor-loop**："当测试已存在且处于 FAIL 状态，需要通过迭代实现让其变绿时触发（前提：先写测试、当前测试 FAIL、目标是收敛到绿）"


### 任务执行结果

无任务元信息


---

## 执行 Loop

> ⚠️ **domain 沉淀是 learning 的核心产出之一（不是附属）**：本次 job 踩过的坑（bugs.md 的「已知问题+精确解决命令」）、构建/测试命令事实（build.md）、环境事实（env.md）——这些是下次 job 的 Step 0 直接消费的先验知识。**没有 domain 沉淀 = 同样的坑下次 job 还会踩一遍**。执行 Loop 的 Step 5（gen-domain）不可跳过，无新事实也要显式说明原因。

`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_1/learning/prompts/learning_loop.md`

---

## 完成要求

`/workdir/sunquan20/AI_CODING/rick/bin/rick tools learning_check job_1`

learning_check pass 后才算完成。

---

## Draft 同步（必须执行，learning_check 前完成）

在所有 loop 步骤完成后、运行 learning_check 前，执行以下同步：

**前置检查**：如果 `/workdir/sunquan20/AI_CODING/rick/.rick/draft` 目录不存在，跳过全部同步步骤（不报错）。

### 步骤一：domain 事实同步到 briefs/

读取本次 job 新增或更新的 `.rick/domain/` 文件（bugs.md / build.md / architecture.md 等），提取**本次 job 产出的新事实**，追加到 `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_*/briefs/domain-sync.md`：

```markdown
## [job_1] Domain 事实同步 - [日期]

### 新增已知问题与解法
- **[问题描述]**：[精确解决命令]（来源：domain/bugs.md）

### 新增架构/构建事实
- **[事实名称]**：[具体内容]（来源：domain/[文件名]）

### 其他新增事实
- [事实]（来源：domain/[文件名]）
```

**同步原则**：
- 只追加本次 job 新增的事实，不重复已有内容
- 每条事实必须标注来源文件
- 如本次 job 无新 domain 事实，写"无新事实"并说明原因
- 如 `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/` 目录不存在，跳过此步骤

### 步骤二：briefs 概念地图复查

读取 `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_*/briefs/E1.md`（若存在），检查其中"掌握程度: 未接触"的条目：
- 如果某条目在本次 job 中已有实践，将掌握程度更新为"了解"或"熟悉"
- 如果某条目已在本次 job 产出的 skills/ 或 loops/ 中被覆盖，标注"已沉淀"

如文件不存在，跳过此步骤。

---

## 完成后引导（呈现给 human）

learning 完成后提示：

```
✅ job {job_id} learning 完成（SUMMARY + skills/loops + domain 已沉淀）

多个 job 攒起来后可跑跨 job 全局反思（演化 loops/domain、合并重复 skills）：
  rick dream
```
