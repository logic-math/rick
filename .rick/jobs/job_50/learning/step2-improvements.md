# Learning Step 2 — job_50 改进机会评估（按价值排序）

> 输入：step1-analysis.md 问题清单（7 项）+ 执行轨迹（12 提交、7 轮实测反馈增量）。评估维度：①更短路径 ②前置检查防错 ③合并冗余动作。

## 高价值

### H1. release 前置检查：dev web 在跑则先 down（防 release 门禁竞态 ×4）

- **问题引用**：问题 4——release 门禁连挂 4 次（~30 分钟诊断+重试成本）：门禁 buildEnv 用 HOME=dev-home，web 测试 spawn 的 worker 与在跑 dev web 实例（活动会话）共享 dev HOME 的 pi sessions 目录 → worker 异步终止滞后于 t.TempDir 清理 → TempDir RemoveAll cleanup 竞态失败
- **改进方案**：把「release 前探测 dev web 状态、在跑则 `rick tools dev-web down`（release 后 `up --no-build` 恢复）」写入 `rick_release_ops_skill` 预检清单（该 skill 已有共享机预检节，追加一条即可）；更根治的是 release.go 门禁前自动 down/恢复 dev web（代码级，见 M5）
- **预期收益**：单次 release 节省 ~30 分钟（4 次失败重试 + 诊断链）；复发风险最高的问题直接消除

### H2. pi 派发纪律：task 文本一律文件化（防 workflowScript 转义坑 ×3）

- **问题引用**：问题 1+3——workflowScript task 内嵌文本两连挂（JSON `\n`→真实换行破坏单引号字符串；撇号破坏）+ runs.all 传 promise 报 invalid key；证据混合（doing-1..4 含 `\n` 成功）说明与文本组合相关，不可靠规避
- **改进方案**：固化为标准做法写入 `pi_orchestration_syntax_skill`：**task 超 1 行 / 含引号撇号反引号 / 含 `\n` → 一律先 write 任务书文件 + 单行指针 task**（本 job research-L1 起全面采用后 0 失败）；同时记录 runs.all 描述符对象形态（禁止传已启动的 run promise）
- **预期收益**：消除派发失败类问题（本 job 3 次 ×1-3 分钟重试+诊断）；research/doing 派发可靠性提升

## 中价值

### M3. dev 增量部署三连一条命令化（merge main → dev-web restart → init-pi）

- **问题引用**：执行轨迹——7 轮增量每轮手工 4 步（commit → cd rick-dev merge main → dev-web restart（带干净 env）→ init-pi 部署 agents），含 env 清理（unset proxy）易错
- **改进方案**：`rick tools dev-web sync`（新增子命令：merge main + build/up + init-pi + 顺手清过期 agent 文件），或先用 shell 脚本固化（零代码）；env 清理（unset http_proxy/https_proxy）内置
- **预期收益**：每轮增量省 3-4 条命令 + 环境易错点；prompt 迭代类 job 的标准节奏提速

### M4. human-loop 会话继承命令化（`--inherit loop_N`）

- **问题引用**：问题 6 + 执行轨迹——6 次手工迁移（拷产物 + 更新 mission 继承注记 + steer 指示继续点）；先建会话再拷产物的顺序约定靠纪律
- **改进方案**：`rick human-loop --inherit <loop_N> [--from-step ⑤]`：自动分配新 loop 目录 + 拷贝产物（mission/zpd/syllabus/values/judgment/lessons/research）+ 生成继承注记；web 侧 params 加 inherit 字段
- **预期收益**：迁移从 5 步降到 1 步；继承注记格式统一（现在每次手写）；v7 协议的跨 loop 续学体验闭环

### M5. web 测试 TempDir 清理竞态根治（测试基建同步）

- **问题引用**：问题 4 的根治缺口——dev-web down 是规避不是修复；测试仍可能在其他负载下竞态
- **改进方案**：web 测试的 supervisor worker 终止与 t.TempDir 清理加同步（cleanup 前 wait worker 进程退出），或门禁 buildEnv 改用独立 scratch HOME（种子 pi runtime）；二选一，前者更根治
- **预期收益**：internal/web 测试从「负载敏感 flaky」变稳定；release 门禁可靠性根因修复

## 低价值

### L6. worker 回执截断：维持「parent 验产物不信回执」惯例

- **问题引用**：问题 2——回执 ×2 被截断但工作完成；parent 已按产物复核（build/test/dry-run 自跑）
- **改进方案**：无需改动——把「回执=路径+要点；验收一律以产物为准」写入 worker 交付契约的已有实践保持即可（本 job 已自然固化）
- **预期收益**：无新增成本，防回归

### L7. web 会话 API 字段契约（steer/prompt）记入 domain

- **问题引用**：问题 7——streaming 会话须 steer、字段是 message 非 text，源码确认一次性
- **改进方案**：在 domain/commands.md（或 web.md）追加一行契约事实即可
- **预期收益**：省下次翻源码 5 分钟

### L8. 代理环境漂移：已根治+文档化（含回滚条件），无需进一步动作

- **问题引用**：问题 5——start-web.sh 已删 proxy export、env.md 已记回滚条件；后续仅在网络再变时触发 env_drift_probe_skill
- **预期收益**：无（闭环完成）

## 汇总（Step 3/5 沉淀指向）

| 项 | 建议沉淀位置 | 类型 |
|---|---|---|
| H1 | rick_release_ops_skill（追加预检条目） | skill 更新 |
| H2 | pi_orchestration_syntax_skill（追加派发纪律节） | skill 更新 |
| M3/M4 | 新 loop：prompt 模板迭代 loop（含部署/继承节奏）或 rick 功能 backlog | loop/功能 |
| M5 | 测试基建修复 backlog（domain/bugs.md 记录） | 代码 |
| L7 | domain/commands.md 一行 | domain |
