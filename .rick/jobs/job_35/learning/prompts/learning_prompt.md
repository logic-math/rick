# Rick Learning 阶段

## 角色定义

你是一个资深的技术文档专家和知识管理专家。根据本次 job 执行过程，总结知识、经验和教训，沉淀可复用 loops 和 skills。

---

## Job 上下文

**Job**: job_35

### Debug 记录

（本次 job 无 debug.md 记录）

### 参考资料路径（按需读取）

- **任务详情**（task*.md）:
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/plan/task1.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/plan/task10.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/plan/task11.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/plan/task12.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/plan/task2.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/plan/task3.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/plan/task4.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/plan/task5.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/plan/task6.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/plan/task7.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/plan/task8.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/plan/task9.md`
- **执行轨迹**（act-path.md）:
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tasks/task1/act-path.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tasks/task10/act-path.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tasks/task11/act-path.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tasks/task12/act-path.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tasks/task2/act-path.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tasks/task3/act-path.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tasks/task4/act-path.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tasks/task5/act-path.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tasks/task6/act-path.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tasks/task7/act-path.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tasks/task8/act-path.md`
  - `/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/doing/tasks/task9/act-path.md`

### 可用的项目 Loops

## 可用的项目 Loops

- **agent-runtime-bootstrap-loop**："当需要初始化/迁移/重装 rick 的 agent runtime（pi）及其扩展时触发（如 rick tools init-pi、版本升级、runtime 迁移）"
- **go-refactor-migration-loop**："当需要把 Go 包整体迁移/重命名/删除，或大规模改动 import 路径并让 build+test 收敛到绿时触发"
- **protocol-redesign-loop**："当需要重构 AI agent 的多阶段协议(如人机协作流程、任务执行流程),涉及阶段合并/拆分/反向回流/批判层重设计时触发"
- **readme-wiki-sync-loop**："当需要编写或重构 README.md、wiki/（用户面向文档）等描述性资料时触发"
- **tdd-red-green-refactor-loop**："当测试已存在且处于 FAIL 状态，需要通过迭代实现让其变绿时触发（前提：先写测试、当前测试 FAIL、目标是收敛到绿）"


### 任务执行结果

| Task ID | 任务名称 | 状态 | Commit Hash | 重试次数 |
|---------|---------|------|-------------|----------|
| task1 | 定义 spec 规范与「spec → 开发计划 → 功能等价实现」验收标准 | success | d3cbb3d6 | 0 |
| task2 | 产出 rick 第一份 spec（四层架构 + 5 模块 + env 四职责契约） | success | c5817808 | 0 |
| task4 | 重构 runtime 模块（pi 调用逻辑收口到 runtime 层） | success | c900b1b1 | 0 |
| task5 | 重构 builder 三件（templates + pibuilder + xxxxbuilder），注入路径而非内容 | success | 5e83a41e | 0 |
| task3 | 落地 env 模块（四职责：pi + 生态扩展 + rick 定制 + 就绪 check） | success | d82fe83e | 0 |
| task6 | 落地 handler 调度聚合层并让 cli 变薄（plan/doing/easy） | success | 319e8433 | 0 |
| task9 | 注册 think/research/exporter 为 pi 自定义 agent（经 env 职责 3 落盘） | success | ab9e476f | 0 |
| task7 | 完成 handler 覆盖 human-loop/ctrl/dream/learning 并让 cli 全量变薄 | success | f4f5f033 | 0 |
| task8 | 做薄 cutover：下沉 doing 调度与门禁到 pi，并删除全部冗余 Go 包 | success | e06f65b2 | 0 |
| task10 | 让 pibuilder 产出单文件内聚的 pi 定制化规范产物 | success | 9bb4a52e | 0 |
| task11 | 把自然语言 subagent 触发词等价迁移为 pi 显式触发语法 | success | be847c7b | 0 |
| task12 | 三个 O 端到端验收 + README/wiki 文档同步 | success | a080458f | 0 |


---

## 执行 Loop

`/workdir/sunquan20/AI_CODING/rick/.rick/jobs/job_35/learning/prompts/learning_loop.md`

---

## 完成要求

`/workdir/sunquan20/AI_CODING/rick/bin/rick tools learning_check job_35`

learning_check pass 后才算完成。

---

## Draft 同步（必须执行，learning_check 前完成）

在所有 loop 步骤完成后、运行 learning_check 前，执行以下同步：

**前置检查**：如果 `/workdir/sunquan20/AI_CODING/rick/.rick/draft` 目录不存在，跳过全部同步步骤（不报错）。

### 步骤一：domain 事实同步到 briefs/

读取本次 job 新增或更新的 `.rick/domain/` 文件（bugs.md / build.md / architecture.md 等），提取**本次 job 产出的新事实**，追加到 `/workdir/sunquan20/AI_CODING/rick/.rick/draft/loops/loop_*/briefs/domain-sync.md`：

```markdown
## [job_35] Domain 事实同步 - [日期]

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

