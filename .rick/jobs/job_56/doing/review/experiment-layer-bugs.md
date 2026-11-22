# 实验执行层 Review — Bug 报告（job_56 运行时级审查）

> Review Agent 2/3 · 2026-11-22 · 范围：builder/coding/debug 三角色信息流 + gate 调用链 + 槽位资源 + 精英继承 + 实验记录
> 方法：源码走读 + super.db 查询 + codes/ 目录审计 + pi session 行为轨迹分析

---

## 预期架构基线

```
_step_locked（已持格锁）
  ├─ 分配 cursor + exp_id + workdir_id（迭代号-时间戳=目录名=表 PK）
  ├─ _run_builder → problem_source 注入 + cell_elite/邻格精英/三要素 ctx → experiment_plan
  ├─ _run_coding → _seed_parent_code(plan.parents → 精英 code_path fallback) → coding agent
  │    ├─ 自检通过 → return candidate
  │    ├─ 截断/无效 → _rescue_artifact（产物抢救）
  │    └─ 自检失败 → _run_debug ≤M 次（experiment_failed → 跳过 E2E）
  └─ _verify_and_admit → CascadeGate.verify → gate_check.py → 远程 verify/benchmark
       ├─ 通过 → archive.admit（精英入格/替换）
       └─ 失败 → _run_gate_debug（1 次） → 重验
_writeback_experiment → experiments 表 UPDATE（record_path/三 session/score/status）
```

---

## Bug 清单

### [E-B01] [P0] [跨机分离] 多机轮询 push→run 跨机致 11 题全部 rejected——stage1_env 误杀

**描述**：`remote_exec.py` 的 `_next_h20()` 轮询调度下，`push_candidate` 推到 A 机、`remote_gate` 在 B 机执行——B 机找不到 A 机推的候选文件。

**证据**：
- 11 个 rejected 实验全部 `stage1_env: reference failed verify gate: No such file or directory: '/tmp/super-kb/cand/cand_*.py'`
- 四台机器 cand/ 目录文件数不均（97: 737 个 / 85: 1 个 / 29.190: 18 个 / 60.179: 0 个）——push 轮询到不同机器
- experiments 表 rejected=11，全部为此原因

**影响面**：11 个好解被确定性误杀（score=NULL），浪费 ~11 × 10min GPU+LLM 时间

**修复方案**：`remote_gate_bound` 合并 push+run 为单次 ssh 链（同机绑定）——**已修复**（本次 review 确认修复生效）。

---

### [E-B02] [P0] [重启风暴] py 调度器被 SIGKILL 24 次导致 27 个实验 interrupted——根因是 super restart 未杀干净旧进程

**描述**：`rick super stop` 杀掉 py 调度器（SIGKILL）但在飞 pi agent 未被同步清理，新进程又因进程锁冲突 exit 5——daemon 重启 3 次全挂 → state=failed。

**证据**：
- super.log 中 `signal: killed` 出现 24 次
- 13:21-13:42 之间 8 次 `session_started`（每次重启产生新 session）
- 27 个 interrupted 实验（每个重启周期在飞实验被杀）
- `进程锁冲突` 错误出现 4 次

**影响面**：27 实验×~8min 平均 = ~3.6 小时浪费；实验目录残留（只含 prompt_builder.md，零 agent 产出）

**修复方案**：
1. Go 侧 `super stop` 应 SIGTERM 优先（py 调度器已有优雅停机——SIGTERM handler）而非直接 SIGKILL
2. Go daemon 重启前应等待旧进程锁释放（当前直接 exit 5 重试 3 次即放弃）
3. `interrupt_stale_running` 已修复死行，但 interrupted 实验的 record_path 为空 → `_writeback_experiment` 不写回 session_id（只写 status='interrupted'）

---

### [E-B03] [P1] [agent_timeout] coding agent 超时 1200s（20 分钟）但 coding 实际需要 40 分钟——超时值严重不足

**描述**：`agent_timeout_s=1200` 但实测 coding agent 的 wall_s 达 2400s+（含远程 verify/benchmark 调用等待）。

**证据**：
- 实验 22（75_Gemm）：coding seq 2717→4264 跨越 14:22→15:02 = 40 分钟，被 timeout 截断
- coding agent_ended 显示 `truncated=false, valid=false`，summary 含 `[timeout after 1200s]`
- step7 的 budget_adjust 已提议 `agent_timeout_s 1200→1800`（iter_18 检查点在案）

**影响面**：~6 个 coding 截断实验——agent 做完了工作但被超时杀，产物被 artifact_rescue 抢救（部分恢复）

**修复方案**：budget_adjust 热调已修复（EVENT_TYPES 扩容 + data 搬运），下一轮 adjust step7 输出 `budget_adjust` 即可热调。或手动改 super.json（需 stop→改→start）。

---

### [E-B04] [P1] [精英继承] 40 题重访时 cell_elite 显示「无精英——本格首访」——精英继承链断（格精英=null 但 visit=2）

**描述**：`problem=40_Matmul_Scaling_ResidualAdd` 的 cells 表 state_json 显示 `elite: null, visit: 2`——两次实验都 failed/rejected，无精英入格。builder prompt 正确显示「无精英」，但 coding 的 parents fallback 到精英 code_path 时得到 null → fallback 到 env 参考实现。

**证据**：
- cells: `{"elite": null, "visit": 2, "weight": 3.0}` → `_seed_parent_code` 的 `cp = elite.get("code_path")` = None
- 但 experiment_plan.json 的 parents 里 builder 自己指定了 `codes/1-202611220044/solution.py`（iter1 的产物）——builder 绕过了精英继承链
- 19-202611221452（iter 19 = 40 题重访）仍 rejected（多机 bug）

**根因**：精英继承链依赖 `elite.code_path`——当格内所有实验都失败时 elite=null，fallback 机制退化为「从 env 参考起步」而**丢失了历史尝试的代码**。builder 的 experiment_plan 可以手动指定 parents 补救，但 coding 的 `_seed_parent_code` 只在 `plan.parents` 为空时才 fallback 到精英——**如果 builder 没指定 parents 且格无精英，历史尝试代码完全不继承**。

**影响面**：格重访时（有失败历史但无精英），历史经验代码不被继承——违反 QD「格精英=历史最优」的设计假设（历史最优≠历史尝试）

**修复方案**：`_seed_parent_code` 的 fallback 链应扩展：plan.parents → elite.code_path → **最近一次尝试的 solution.py**（从 experiments 表按 iteration DESC 找该格的最新非空 solution.py）

---

### [E-B05] [P2] [coding 自验] coding agent 需要自建 `pinned_remote.py` 绕过多机 bug——环境基础设施对 agent 的隐性依赖

**描述**：coding agent 在多机 bug 期间发现官方 skill 薄壳不可用，自建了 `pinned_remote.py`（定机远程执行驱动），绕过 gates/ 目录保密纪律的约束（agent 不能读/改 gates/ 下的文件）。

**证据**：
- `codes/22-202611221342/pinned_remote.py` 完整重写了 remote_exec 的核心逻辑（SBX + H20_PASS + H20_POOL）
- coding session 行为轨迹显示 38 次工具调用中 7 次与远程执行相关（包含读取 skills/verify.py → 自建 pinned_remote.py）
- 该文件内含完整密码 `MiMi@@19950810`——**密码泄露到 agent 可见的实验目录中**

**影响面**：
1. 密码泄露：agent 实验目录中含明文密码（安全风险）
2. 口径漂移风险：agent 自建的传输层可能与官方实现不完全一致
3. agent 花费额外 token/time 基建绕过而非实验本身

**修复方案**：多机 bug 已修复（remote_gate_bound），此问题的触发条件消除。但应：
1. 在 domain_knowledge.md 中禁止 agent 在实验目录存放密码/敏感信息
2. 考虑将密码改为环境变量传递（remote_exec.py 内引用 env var 而非硬编码）

---

### [E-B06] [P2] [实验记录] interrupted 实验的 experiment_record.md 只有第 1 节（设计）——builder 写了但 coding/debug 没跑到就被杀

**描述**：27 个 interrupted 实验的 record_path 为空或只有 builder 部分记录。

**证据**：
- `experiments WHERE status='interrupted'` 的 record_path 全部为空
- codes/24-202611221452/ 只有 prompt_builder.md（builder 被杀在初始阶段）
- codes/23-202611221447/ 有 prompt_builder.md + parent_baseline.py（builder 被杀在调研阶段）

**影响面**：部分实验记录残缺（可从 events 表还原部分信息，但 coding/debug 的执行过程丢失）

**修复方案**：这是重启的固有限制（优雅停机可缓解——E-B02 修复后 interrupted 数应大减）。当前 record_path 为空是正确语义（无完整记录）。

---

### [E-B07] [P2] [槽位资源] 2 个实验槽（exp-slot）被 interrupted 实验残留占用——资源泄漏

**描述**：`resources` 表中 2 个 `experiment_slot` 的 `holder_experiment_id` 非空，但对应实验已 interrupted。

**证据**：
```sql
SELECT resource_id, holder_experiment_id FROM resources
WHERE resource_type='experiment_slot' AND holder_experiment_id IS NOT NULL;
```
返回 2 行，对应的 experiment status='interrupted'。

**影响面**：max_parallel=8 但实际可用槽=6（2 个被泄漏占用）

**修复方案**：`_slot_worker` 的 `finally` 块已释放——但 SIGKILL 时 finally 不执行。`_init_resource_slots` 已有清残留逻辑（启动时清空全部 holder）——但**只在启动时清一次**，运行中 interrupted 的实验槽位泄漏未回收。修复：在 `interrupt_stale_running` 时同步释放对应的资源槽。

---

### [E-B08] [P1] [builder valid=false] builder agent_ended valid=false 且 total_tokens 空——超时截断但无有效信息传递给 coding

**描述**：实验 22（75_Gemm）的 builder agent_ended：`valid=false, truncated=false, total_tokens=null`。但 coding 正常启动并接收了 experiment_plan（builder 的 fallback plan）。

**证据**：
- seq 2716: builder `valid=false, wall_s=null`（wall_s 缺失！）
- 但 coding 的 experiment_plan.json 内容丰富（idea/parents/plan_steps 齐全）——说明 builder 的 plan 来自 `_run_builder` 的 fallback（result.summary 为空 → plan idea=""）

**等等**——实验 22 的 experiment_plan.json 的 idea 内容丰富（六步入格结算·零改动复验...）——这不是空 fallback。让我重新检查……

实际上 builder 的 `result.data.get("experiment_plan")` 在 valid=false 时可能仍然有数据（AgentResult 的 data 独立于 valid）——但 `total_tokens=null` 表明 JSON 解析失败（client 层 valid=False + data={}）。coding 拿到的 plan 是 fallback `{"idea": result.summary or ""}`——但 summary 也为空——所以 plan.idea 为空。但实际 experiment_plan.json 有丰富内容……

结论：builder 可能在 timeout 前成功写盘了 experiment_plan.json（文件落盘），但最终 JSON 输出块被 timeout 截断——`result.data` 为空 → plan 为 fallback 空 → coding 收到空 plan 但 experiment_record.md 的第一节已写好（builder 落盘了文件）。coding 依赖 experiment_record.md（恒常上下文）而非 plan——所以 coding 仍然知道要做什么。这是一个**设计上的冗余韧性**，但 builder 的 valid=false 是超时误判。

**影响面**：builder 被误标为失败（不影响实验链继续），但 stats 记录偏差

**修复方案**：builder 的 agent_timeout_s 应比 coding 更宽松（builder 有调研+设计两步，耗时更长）；或 artifact_rescue 机制扩展到 builder 的 experiment_plan.json

---

### [E-B09] [P2] [gate stage1] CascadeGate stage1 对 reference 跑 verify 时使用了 `skills/verify.py`（本地薄壳）——push 轮询到 A 机而 CascadeGate 的 stage1 verify 也用轮询——**已随 E-B01 修复**

**描述**：CascadeGate 的 `_verify_gate_check_v2` 在 stage1 检查 reference 时跑 `skills/verify.py`——这是本地薄壳（会推 reference 到某台机器跑）。多机 bug 期间，每次 stage1 检查都随机挂在不同机器上。

**证据**：`gates.py:683` 的 `self._run_script(py, os.path.join("skills", "verify.py"), ref_path, ...)`

**修复方案**：已随 E-B01 的 `remote_gate_bound` 修复（skills/verify.py 现在用 bound 模式）。

---

### [E-B10] [P1] [密码硬编码] remote_exec.py 中 SSH 密码硬编码——所有能读 gates/ 的 agent 都可获取

**描述**：`remote_exec.py` 第 25 行 `H20_PASS = "MiMi@@19950810"` 硬编码。虽然 gates/ 目录对 builder/coding/debug 禁读（保密纪律），但 adjust 角色可读写——且 agent 的 pinned_remote.py（E-B05）已在实验目录暴露了密码。

**证据**：
- `gates/remote_exec.py:25`: `H20_PASS = "MiMi@@19950810"`
- `codes/22-202611221342/pinned_remote.py`: 同样硬编码
- `codes/19-202611221452/pinned_remote.py`: 同样硬编码

**影响面**：安全风险——密码泄露到多个实验目录（可被任何能读 codes/ 的实体获取）

**修复方案**：改为 `os.environ.get("SUPER_H20_PASS")` 或从 `~/.rick/credentials` 读取；清理已泄露的实验目录中的密码

---

## Bug 清单总表

| 编号 | 严重度 | 维度 | 一句话摘要 | 修复优先级 |
|---|---|---|---|---|
| E-B01 | **P0** | 跨机分离 | 多机轮询 push→run 跨机致 11 题误杀 | ✅ 已修复 |
| E-B02 | **P0** | 重启风暴 | SIGKILL 风暴致 27 实验 interrupted + 进程锁冲突循环 | 高（Go 侧改 SIGTERM 优先） |
| E-B03 | **P1** | agent_timeout | coding 超时 1200s 但实际需 2400s+（~6 实验截断） | 高（budget_adjust 已修，待下轮生效） |
| E-B04 | **P1** | 精英继承 | 无精英格重访时历史代码不继承（builder 可手动补救但 coding fallback 退化为 env 参考） | 中 |
| E-B08 | **P1** | builder valid | builder 超时误判（plan 文件已落盘但 JSON 截断 → valid=false + fallback 空 plan） | 中 |
| E-B10 | **P1** | 密码硬编码 | SSH 密码硬编码在 remote_exec.py + 泄露到多个实验目录 | 中（安全） |
| E-B05 | P2 | agent 自建基建 | coding 自建 pinned_remote.py 绕过 bug（含密码泄露） | 低（E-B01 修复后不再触发） |
| E-B06 | P2 | 实验记录 | interrupted 实验记录残缺（只有 builder 部分） | 低（E-B02 修复后减量） |
| E-B07 | P2 | 槽位泄漏 | 2 个实验槽被 interrupted 实验残留占用 | 低 |
| E-B09 | P2 | gate stage1 | CascadeGate stage1 verify 也受多机 bug 影响（随 E-B01 修复） | ✅ 已修复 |
