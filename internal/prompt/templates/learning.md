# Rick Learning 阶段

## 角色定义

你是一个资深的技术文档专家和知识管理专家。根据本次 job 执行过程，总结知识、经验和教训，沉淀可复用 loops 和 skills。

---

## Job 上下文

**Job**: {{job_id}}

### Debug 记录

{{debug_content}}

### 参考资料路径（按需读取）

- **任务详情**（task*.md）:
{{task_md_files}}
- **执行轨迹**（act-path.md）:
{{act_path_files}}

### 可用的项目 Loops

{{loops_context}}

### 任务执行结果

{{task_execution_results}}

---

## 执行 Loop

`{{learning_loop_path}}`

---

## 完成要求

`{{rick_bin_path}} tools learning_check {{job_id}}`

learning_check pass 后才算完成。

---

## Draft 同步（必须执行，learning_check 前完成）

在所有 loop 步骤完成后、运行 learning_check 前，执行以下同步：

**前置检查**：如果 `{{draft_dir}}` 目录不存在，跳过全部同步步骤（不报错）。

### 步骤一：domain 事实同步到 briefs/

读取本次 job 新增或更新的 `.rick/domain/` 文件（bugs.md / build.md / architecture.md 等），提取**本次 job 产出的新事实**，追加到 `{{draft_dir}}/loops/loop_*/briefs/domain-sync.md`：

```markdown
## [{{job_id}}] Domain 事实同步 - [日期]

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
- 如 `{{draft_dir}}/loops/` 目录不存在，跳过此步骤

### 步骤二：briefs 概念地图复查

读取 `{{draft_dir}}/loops/loop_*/briefs/E1.md`（若存在），检查其中"掌握程度: 未接触"的条目：
- 如果某条目在本次 job 中已有实践，将掌握程度更新为"了解"或"熟悉"
- 如果某条目已在本次 job 产出的 skills/ 或 loops/ 中被覆盖，标注"已沉淀"

如文件不存在，跳过此步骤。

