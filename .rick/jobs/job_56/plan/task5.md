# task5 — M5 集成验收：mock 场景 + e2e 全链路 + 全量回归 + 文档收口

> job_56 · DAG L4（单 task 层）· 依赖 task4（全部前序完成）
> 源设计：grilling/design-tree.md KR7

# 任务目标

端到端集成验收：mock_pi 扩展 7 步流水线+sub loop 场景；e2e 全链路（DB+env 体系+7 步协议+实验链）跑通；全量测试（py+go）无回归；文档收口（super/README.md 更新）。

# 关键结果

1. **KR-A【mock_pi 场景扩展】**：mock_pi.py 新增/适配场景：
   - `ok_adjust_pipeline`：7 步流水线（每步按 prompt 文件名区分——mock 按 workdir/prompt 路径判定当前步骤，输出对应步骤产物：facts.md/perspectives.md/contradictions.md/judgment.md/reversal.md/proposals JSON/domain_knowledge 追加）
   - `ok_debug_failed`：debug 上报 experiment_failed:true+完整实验记录（四节）落盘
   - 既有场景（ok_builder/ok_coding/ok_debug）适配新协议（builder 计划含风险推演节/coding 无 tdd 字段/实验记录四节结构）
2. **KR-B【e2e 全链路】**：e2e_test.py 升级：mock pi 全链路跑一轮完整迭代（采样→builder→coding→debug→gate_check→入格→adjust 流水线 7 步→decisions schema 落盘→humans 表干预注入→失败路径跳过验证）——断言 DB 表状态（experiments 行/humans applied 标记/cells 权重变化/events 步骤序列）。
3. **KR-C【全量回归】**：`python3 -m unittest discover -s super/tests -p 'test_*.py'` 全绿 + `go build ./... && go test ./...` 全绿——修复前序 task 遗留的集成级回归（允许在写域内打最小补丁）。
4. **KR-D【进程锁 e2e】**：e2e 含进程锁场景（模拟存活 PID→第二次启动失败）。
5. **KR-E【文档收口】**：super/README.md 更新（DB 架构/7 步协议/实验链/验证金字塔/gates 隔离——面向 super 用户的操作视角）。

# 写域（单 task 层，无并行冲突；回归修复允许全仓）

```
super/tests/**（全部测试文件——mock_pi.py/e2e_test.py/super_session_test.py 及回归修复）
super/superopt/**（仅集成级回归修复——最小补丁；不得重构前序 task 定型结构）
super/README.md
```

# 实现要点

1. **mock 步骤判定**：mock_pi 按 `--append-system-prompt` 的文件名（prompt_adjust_fact.md 等）或 workdir 路径（codes/adjust/iter_N/）判定步骤——从 prompt 文件名解析最稳（派发侧每步独立 prompt 文件）。
2. **e2e 环境**：沿用既有 e2e 的 fake pi 机制（mock_pi 路径注入）；DB 用 tmp 工作区（每 case 建库断言后清理）。
3. **回归修复边界**：发现前序 task 的结构性缺陷（非补丁级）→ 记录到 debug/（bug{N}.md）+最小补丁+测试证明，不推翻重构。
4. **go test 范围**：internal/handler、internal/web、internal/runtime 的 DB 读迁移测试（task1 已写）+ 全仓既有测试。

# 测试方法（自测=交付本体）

本 task 的产出就是测试资产——e2e/mock 即验收物；自测=跑全量+修复循环（RED→GREEN）。

# 验收（gate4.py 承担）

`python3 .rick/jobs/job_56/plan/gates/gate4.py`：py 全量 unittest 全绿 + go build+test 全绿 + e2e 全链路断言 + README 结构断言。
