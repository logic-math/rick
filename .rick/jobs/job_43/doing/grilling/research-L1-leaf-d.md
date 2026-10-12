# research-L1 叶子D：近期交付与遗留（job_43 功能级改进依据）

信源：仓库本地一手材料（git log / plan / learning / bugs.md / grep），信源等级 L1。

## 1. 近期交付主题（git log -60 归纳，避免 job_43 重复）
- **job_36（主线，v5.0.11，刚交付）**：自进化架构闭环——`rick tools dev-web`（worktree/独立 HOME/pi 沙盒/build_id 指纹）、`rick tools release`（版本目录/原子换链/.last 回滚/--detach/--merge-source）、rick-rsi-loop S0-S7、rsi_check/loops_check、suspend 挂起→人工一键恢复、/compact 全链路（RpcClient→API→前端）、`rick web --daemon`。证据：f612a6f…c1e5b78 提交链。
- **job_36 前段（同 job）**：web UI 验收期 35+ 修复（SSE 断线续传/流式闪烁三连/长会话卡顿/ImportSession 导入/归档恢复/星空美术）。
- **job_35**：pi 迁移 + 四层架构重构（cli→handler→builder→runtime/env/workspace/prompt）+ spec 规范，12 task 全成。证据：job_35/learning/SUMMARY.md。**job_43「不做架构级改动」即指不再动这一层**。
- job_37：RSI loop 标准加载验证（一次性会话）；job_38-42：纯测试会话（"回复 ok"），无交付。

## 2. 各 job 交付摘要与明确未完成项
- job_35：无未完成 task。
- job_36：29/29 task 全成、4 次成功 release；但 SUMMARY 尾部「给下一轮 RSI 的已知边界（诚实记录）」列 3 条遗留：
  1. **F6：level_complete 写回晚于提交**，未修，原文标注"下次迭代的候选项"（证据：job_36/learning/SUMMARY.md 尾节）。
  2. dev 沙盒 catpaw 扩展不随 init 更新（幂等跳过已存在），dev 上测 compact 需手工同步。
  3. pi 上游缺陷（compact 摘要路径绕过 before_provider_request）未报社区（外部事项，非代码改动）。

## 3. 代码 TODO/FIXME（internal/ + web/src/，排除 _test/node_modules）
- `internal/prompt/easy_prompt.go:310` 与 `:344`——TODO(2026-08)：全量迁移到 debug/ 目录格式后移除 debug.md 兼容回退。**真实遗留，2 处**。
- `internal/cmd/tools_rsi_check.go:40` 与 `:357`——rsi_check `--init` 骨架的防自欺标记，**设计内，非债务**。
- web/src/：0 处。

## 4. bugs.md 未闭环项
- 无「未修复」条目（全部 ✅ 已修复/已实施）。
- 但存在 ≥10 条「已实施（待实测/待验证）」：典型 v4.4.11 buildGrillingSection 双函数同步（真实 easy 实测待验证，行 489）、v4.4.14 下轮 grilling 第一动作建树观察（行 532）、v4.4.5 完整会话实测待下轮（行 411）、v4.4.8 真实 plan/doing 实测待观察（行 446）。

## 5. 对 job_43 的直接含义
- 勿重复方向：自进化管道机制（dev-web/release/rsi_check/suspend/daemon）、web UI 核心功能、四层架构。
- 明确遗留候选（功能级）：① F6 修复（level_complete 写回时序，涉 runtime/gates）；② easy_prompt.go 2 处 debug.md 回退清理（需先确认 debug/ 目录迁移完成度）；③ bugs.md「待实测」项的集中验证轮（多为观察类，工作量小）。
