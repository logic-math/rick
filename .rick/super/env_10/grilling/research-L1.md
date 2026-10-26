# research-L1：env_9 复用与三特性验证——事实简报（自查消解）

> 生成：2026-10-26 12:40（super 中控会话 L1 自查；全部为本地代码/工作区事实，无需联网调研）
> 信源等级：代码原文（super/superopt 源码）+ 工作区实测数据（env_9 事件流/档案）+ git 历史
> 用途：支撑 design-tree L2 判断节点与 L3 提问；所有结论附出处

## 1. 需求背景

- env_10/requirement.md：**用现有 env_9 验证五要素/成本归因/格锁：启动 2 迭代进化并汇报实验记录与成本分布**。
- env_9 工作区（`~/.rick/super/env_9/`）：matmul 8192×8192 float32 性能优化，已于 2026-10-26 06:33 跑满 15 迭代**正常收档**（state=stopped；adjust 提 `terminate_recommend`：预算 14/15 + 冠军 430.83ms≈sgemm 本体 floor + 连续 4 轮无新纪录）。
- 关键时间线（git log）：
  - env_9 运行结束：10-26 06:33
  - task44（五要素 elite_code_path/elite_record_path）：10-26 12:11 之前落地
  - task46（cost_report.py + super_cost_analysis skill + iteration_ended.by_role）：10-26 11:54 落地
  - task45（CellLocks 格锁：内存+文件 O_EXCL 双保险/TTL 回收/孤儿清理）：10-26 12:11 落地
  - env_10 工作区创建：10-26 12:29（本会话）
- **结论**：三特性都是 env_9 运行**之后**才落地的新代码——env_9 事件流里 `iteration_ended.by_role` 全部为空（12/12 空）、无任何 lock 事件、elite 无 code_path/record_path 字段。本次 2 迭代 run = 对这批新特性的**首次真跑验证**。

## 2. 三特性的实现位置（验证点 → 代码证据）

| 特性 | 实现位置 | 验证可观测面 |
|---|---|---|
| 五要素 | `superopt/archive.py:223-264`（admit 时同步 `elite_code_path` + `elite_record_path`）、144-145（建格兜底）、`description/weight` 由 adjust 维护 | archive.json 各格：weight/score(elite.perf_value,elite.score)/elite_code_path/elite_record_path/description；读格 API `archive.py:195-202` |
| 成本归因 | `superopt/loop.py:716-721`（iteration_ended.by_role）、`loop.py:628`（_iter_agents 起点=adjust carry）、`loop.py:1430-1473`（adjust 统计收集暂存）、`agents.py`（agent_ended.usage 六字段） | iteration_ended.by_role（迭代×角色 tokens/wall_ms/tool_calls）；agent_ended.usage（角色级，含 adjust）；ledger.jsonl（每迭代 agent 成本）；`super/tools/cost_report.py`（role 表/迭代表/Top-N/上下文膨胀/cache） |
| 格锁 | `superopt/cell_lock.py`（CellLocks：内存集合+`<ws>/locks/cell_<safe_id>.lock` O_EXCL 文件锁、TTL 陈旧回收 on_stale 回调、release_orphans 启动清理）；loop `_step` 持锁执行 finally 释放 | locks/ 目录锁文件生命周期；事件 `cell_lock_orphans_released`（启动）/`cell_lock_stale_recovered`（TTL）/`cell_lock_resample`（撞锁重采样，串行下不会出现） |

## 3. env_9 可复用资产清单（复用 = 拷贝到 env_10）

| 资产 | 路径 | 状态 |
|---|---|---|
| env 定义 | env_9/env/{env.json, feature_space.json, code/{reference.py,naive_start.py}, skills/五件, domain_knowledge.md, requirements.txt, .venv/} | 完整；env.json 含 debug_policy{gain 20%, recheck_tolerance_band true}、correctness.sizes=[64,8192]、ref_runtime_band [380,700]ms |
| 门禁种子 | env_9/gates/{exploit_registry.json, exploits/45 个样本(d1-d4 四类), golden_examples/, verify_exploits.py, gen_exploits.py} | 完整（45 exploit） |
| prompts | env_9/env/prompts/ 仅 builder.md.legacy-20261022（遗留弃用） | env 级模板已回退——当前用仓内默认 `super/prompts/*.md`（含 task45/46 新纪律） |
| env_9 的 archive/decisions/ledger | env_9 工作区级 | **不复用**（已收档；且 elite 路径指向 env_9/codes 绝对路径） |

## 4. 关键机制事实（决定运行参数选择）

1. **initSuperWorkspace 不拷贝 env**（`internal/handler/super.go:213-284`）：`--env` 只写 super.json.EnvPath；gates 目录固定 `<ws>/gates`（`loop.py:257`）——**env_10/gates 必须手动放入 env_9 的门禁种子**，否则 preverify-red 空跑（0 exploit 全过 = 假绿）。
2. **pi_path 必须显式写 super.json**：py `config.load` 只做内存补缺（默认 ""），`agents.py:324` 空值直接 raise。env_10/super.json 当前 pi_path="" → **启动前必须写入** `/Users/logic/workdir/rick-dev-home/.rick/pi/agent/runtime/node_modules/.bin/pi`（已确认存在，env_9 同款）。pi 调用不带 provider/model 参数（`agents.py:332-340`），用 pi 自身托管配置（env_9 实跑可用）。
3. **adjust 触发语义**（`loop.py:452-456`）：每循环周期头部 `cursor>0 且 cursor%interval==0` 才启动；`max_iterations=2` 时循环在 `cursor>=2` 处 break（在 adjust 触发点**之前**）：
   - `adjust_interval=2`（env_9 旧值）→ **2 迭代全程零 adjust 派发**
   - `adjust_interval=1` → iter1 边界启动 1 次 adjust（与 iter2 并行），会话收尾 `_join_adjust()`（`loop.py:440,1510`）收事件（adjust_decision/agent_ended 先于 session_ended）
4. **adjust 成本归因口径边界**：adjust 统计在 `_collect_adjust` 暂存 `_adjust_stats_carry`，由**下一个** `_step_locked` 头部并入该迭代的 by_role（`loop.py:628`）。恰好 2 迭代时收尾 join 的 adjust 无下一迭代接收 → **adjust 成本进 agent_ended.usage + ledger（角色级），不进任何 iteration_ended.by_role（迭代级）**——报告中须分两个口径呈现，这是框架当前真实行为，不是缺陷。
5. **门先证红契约**：super.json preverify_red=true（start 时 py 调度器先跑 `CascadeGate.preverify_red()`，`superopt/gates.py`；不通过非 0 退出不启动）。exploit 注册表 45 条全拒 + 金样例零误杀。
6. **门禁脚本解释器**：`gates.py:258-261` 优先 `env/.venv/bin/python`（numpy 在 env venv），缺省 sys.executable——复制 env 含 .venv 即原口径。
7. **单迭代时长参考**（env_9 iter10-15 实测）：~27-49 分钟/迭代 → 2 迭代 ≈ 1-2 小时；墙钟 4h 富余。
8. **sessions 目录**：`<ws>/sessions/` pi 会话产物；`codes/iter{N}-{uuid}/` 含 prompt_{role}.md + experiment_record.md + research_report.md 等（实验记录链 = 验证对象之一）。
9. **干预通道**：`rick super event --type <t> --payload <json>` 写 interventions/ 队列（单写者纪律，迭代边界摄取）。
10. **archive 初始态**：`initSuperWorkspace` 建空 `{"cells":{},"pending":[]}`；首个 admit 来自 iter1（分殖优先未访问格）。空格 admit 必然成功（该格首个合法解即精英）→ admit 路径 100% 触发，对五要素验证有利。

## 5. 未能自查消解的问题（→ L2/L3 判断节点）

- 复用方式（拷贝 vs 原位引用）、.venv 处理、archive 初态、adjust_interval 取值、格锁验证方式（是否注入孤儿锁 fixture）、验证判据确认、汇报落点——均为 human 裁决点（见 design-tree L2/L3）。
