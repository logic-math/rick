# Rick Learning 阶段

## 角色定义

你是一个资深的技术文档专家和知识管理专家。根据本次 job 执行过程，总结知识、经验和教训，沉淀可复用 loops 和 skills。

---

## Job 上下文

**Job**: job_26

### Debug 记录

（本次 job 无 debug.md 记录）

### 参考资料路径（按需读取）

- **任务详情**（task*.md）:
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/plan/task1.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/plan/task2.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/plan/task3.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/plan/task4.md`
- **执行轨迹**（act-path.md）:
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/doing/tasks/task1/act-path.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/doing/tasks/task2/act-path.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/doing/tasks/task3/act-path.md`
  - `/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/doing/tasks/task4/act-path.md`

### 可用的项目 Loops

## 可用的项目 Loops

- **do-check-mark-success-loop**："当 doing_check 报错 task status != success 或 commit_hash 缺失时触发"
- **tdd-red-green-refactor-loop**："当测试已存在且处于 FAIL 状态，需要通过迭代实现让其变绿时触发（前提：先写测试、当前测试 FAIL、目标是收敛到绿）"


### 任务执行结果

| Task ID | 任务名称 | 状态 | Commit Hash | 重试次数 |
|---------|---------|------|-------------|----------|
| task1 | 添加 `.rick/draft/` 目录基础设施并注入 draft_dir 变量 | success | d50b255a | 0 |
| task2 | 升级 think agent 模板：每个 SENSE 阶段结束时捕获关键判断到 judgment.md，Perspective 阶段写入 draft/loops.md | success | 46ba642f | 0 |
| task3 | 升级 express agent 模板：添加 judgment.md review 清洗步骤和 ZPD 显式评价引导 | success | 6f73d17b | 0 |
| task4 | 升级 learning 阶段：注入 draft_dir 变量并添加 domain 事实同步到 draft/progress.md 步骤 | success | 92b0e3e4 | 0 |


---

## 执行 Loop

`/Users/sunquan/ai_coding/CODING/rick/.rick/jobs/job_26/learning/prompts/learning_loop.md`

---

## 完成要求

`/Users/sunquan/ai_coding/CODING/rick/bin/rick tools learning_check job_26`

learning_check pass 后才算完成。
