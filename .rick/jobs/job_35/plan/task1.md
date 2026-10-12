# 依赖关系

（无依赖）

# 任务名称
定义 spec 规范与「spec → 开发计划 → 功能等价实现」验收标准

# 任务目标
升级 rick 项目的 domain 描述方法，新增「spec」概念（KR1.1 + KR1.3）。spec = 结构化自然语言描述的工程实现契约。产出 spec 规范文档：定义 spec 的结构模板（四要素：模块边界 / 职责 / 接口契约 / 验收标准）、spec 与 domain 其他文档的关系，以及 spec 的验收标准定义（spec → 开发计划 → 功能等价实现；功能等价 = 通过所有功能验收，即「丢弃源码即可 AI coding 出功能等价的 rick」）。

spec 规范须能承载本次重构的收敛结论：rick = 引导程序（env 保证 pi 就绪 → builder 拼提示词 → runtime 拉 pi），env 四职责、runtime 职责（拉起 pi + 内部校验 session 就绪 + 返回 (sessionID, trace)）、handler 职责（编排 + 持久化 sessionID）、下沉策略（dag 调度 → pi workflowScript 编排；门禁 → pi hook/脚本；think/research/exporter → pi agent）。

依据：`.rick/draft/rfc/rfc-rick-三层架构重构与spec信息内核-2026-08-14.md` §5「信息内核 = spec」与 §6 O1 KR1.1/KR1.3。

# 关键结果
1. 新增 `.rick/domain/spec.md`：定义 spec 是什么（结构化自然语言工程实现契约）、四要素结构模板（模块边界/职责/接口契约/验收标准，各附说明与占位示例）、spec 与 domain 其他文档的关系
2. 明确 spec 验收标准：功能等价 = 通过所有功能验收（spec → 开发计划 → 功能等价实现）；给出「丢弃源码 → AI 重构出功能等价 rick」的可操作判据（列出哪些命令 dry-run/check/集成测试必须通过）
3. 更新 `.rick/domain/README.md` 索引表，登记 `spec.md` 一行

# 测试方法
（本 task 为文档任务，TDD 红-绿循环不适用，改为「验收断言」：文件存在 + 关键词命中；断言真实文档内容，不 mock。）

1. 正常路径：前置条件 = 仓库存在 `.rick/domain/` 且 `README.md` 可读；输入 = 新建 `spec.md` 正文（四要素模板）；操作 = 写 `.rick/domain/spec.md` + 更新 `README.md` 索引 + `git add`；预期 = `test -f .rick/domain/spec.md` 返回 0，`README.md` 含 `spec.md` 行。
2. 边界（四要素齐备）：前置条件 = spec.md 已写；输入 = 待写入正文（含模块边界/职责/接口契约/验收标准四节示例）；操作 = `for w in 模块边界 职责 接口契约 验收标准; do grep -q "$w" .rick/domain/spec.md || exit 1; done`；预期 = exit 0（四关键词**各自**命中，而非合计 ≥1）。
3. 异常（验收标准可检索 + 无变量泄漏）：前置条件 = spec.md 已写；操作 = `grep -c '功能等价' .rick/domain/spec.md`（≥1）与 `grep -c '{{' .rick/domain/spec.md`（=0）与 `grep -qE 'dry-run|go test|集成测试' .rick/domain/spec.md`（≥1，可操作判据被枚举）；预期 = 功能等价命中 ≥1、可操作判据枚举命中、且无 `{{`。
