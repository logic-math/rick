# research-L1 — 全局验收层调研简报（env_5 / 验收复验 2）

> 调研方式：代码库考古（git log / 工作区状态 / super 源码 / pi 会话记录）+ 环境自检。
> 结论置信度：高（全部一手证据）。

## 1. 「验收复验 2」的语义与历史脉络

| 工作区 | requirement | 状态 |
|---|---|---|
| env_1 | 优化矩阵乘法 | 空骨架，0 事件，无 env/ |
| env_2 | 优化 512×512 矩阵乘性能… | 空骨架，0 事件，无 env/ |
| env_3 | 同上 | 空骨架，0 事件，无 env/ |
| env_4 | …（验收复验） | 空骨架，0 事件，无 env/，sessions/ 只有 method/prompt/session_id 三件套 stub——**会话从未运行** |
| env_5 | …（验收复验 2） | 本次，同为 web UI prepareInteractive 建的骨架 |

- git 脉络：job_47 建成 super 三层体系（Go PySupervisor → py 调度器 → pi agent 池），job_48 落地 web super 会话（目录模型 v2：env_N 自动编号 + requirement 必填 + grilling inline）。
- env_N 骨架由 web UI 的 prepareInteractive 创建（super_env init 骨架 + 严格加载预检）。
- **历次 super 中控会话（含验收复验 1）全部止步于骨架阶段，端到端流程从未走通。** env_4 会话无任何 pi transcript 可寻（全盘 find 无 999239f3 记录）。
- 本次验收复验 2 = 第一次真正端到端跑通：grilling → gen_env → 门先证红 → start → 进化 → 干预 → stop。

## 2. 依赖链自检（super-loop 依赖准备四项）

| 项 | 结论 |
|---|---|
| rick 命令组 | ✓ `/Users/logic/workdir/rick-dev/bin/rick`（不在 PATH，需全路径）；五子命令齐全 |
| pi runtime | ✓ pi 0.84.2 @ `/Users/logic/workdir/rick-dev-home/.rick/pi/agent/runtime/node_modules/.bin/pi`；**super.json config.pi_path 为空，启动前必须写入**（PiAgentClient 对空值抛错，无 fallback） |
| env 模板 | ✓ `super/env_templates/matmul/` 完整（env.json/feature_space.json/code//data//skills//domain_knowledge.md/requirements.txt） |
| 环境变量 | ✓ `PI_CODING_AGENT_DIR=/Users/logic/workdir/rick-dev-home/.rick/pi/agent` 已在本会话环境，沿 bash→rick→py 调度器→pi 池传播；该 agent dir 的 settings.json 默认 catpaw-proxy / glm-5.3 / thinking high（与 config.json pi_extra_args 一致）。注意：pi 默认 `~/.pi/agent` 不存在——**全靠该环境变量传播**，中断即 pi 池起不来 |

## 3. super 系统运行机制（源码级确认）

- **启动链**：`rick super start --name env_5 --env <path> --daemon` → initSuperWorkspace（幂等补缺）→ Go PySupervisor → `python3 -m superopt --workdir <ws>`（PYTHONPATH=repo/super）。
- **门先证红**：启动契约内嵌（preverify_red=true）——exploit 注册表全拒 + 金样例全过才放行，失败即中止并输出泄漏/误杀清单。
- **首迭代机制**：无父本时拷 env/code/ 全部文件（reference.py + naive_start.py）进 codes/{uuid}/，coding agent 从此起步。首迭代即可入格（elite_admitted，pending 延迟入库判定：new_cell 也入 pending 复测）。
- **干预语义**（重要）：所有干预在迭代边界 drain → `intervention` 事件（source=human）；**只有 stop 类有直接机械效果**（stop_requested → 优雅停）；weight/pause/resurrect 类经事件流影响 adjust agent（每 adjust_interval=5 迭代）提案，soft_weight 提案直接应用并落 weight_changed 事件（source=agent:adjust）。G3 的判据=事件流出现 intervention（source=human）——任何类型均满足，weight 最安全。
- **自然完成**：max_iterations 到点 exit 0 → 不触发崩溃重启（job_47 修复）→ state=completed；手动 `rick super stop` → state=stopped + session_ended。两条收尾路径都验证「停止干净」。
- **事件类型**：session_started/ended、iteration_started/ended、cell_sampled、agent_started/ended、candidate_produced、gate_result、elite_admitted/replaced、weight_changed、proposal、adjudicated、invalidation、resurrection、intervention、heartbeat、env_failure_concluded。

## 4. matmul 模板关键参数（env 设计的事实基线）

- env.json：perf_desc=low（runtime_ms）、正确性 ε=1e-3（64×64 固定 seed=42）、bench 512×512 float32、K=3 中位数、timeout 60s、软门禁 band 5%。
- feature_space.json：2 轴正交——impl_strategy（naive_loops/blocked/vectorized/strassen_like）× memory_layout（row_major/col_major/tiled）= 12 格。
- code/：reference.py（np.matmul=BLAS 天花板，正确性锚点+性能基线）+ naive_start.py（纯 Python 三重循环，正确但慢——进化起点）。
- 模板**不含** exploit 样本与金样例目录——需 gen_env 现场构造（这是 L2 层的设计工作）。

## 5. 悬而未决（→ L3 判断节点）

以下属 human 裁决点，非调研可消解：验收量化判据 / 运行预算 / 干预演示类型（→ 第 1 层 J1-J3）；门禁参数沿用与否 / exploit 样本集 / 特征空间 / agent 模型档位（→ 第 2 层 J4-J7）；停止方式（→ 第 3 层 J8）。
