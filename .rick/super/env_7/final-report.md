# env_7 super 进化使命收尾报告

> 使命：「优化矩阵乘法」| 日期：2026-10-20 | 工作区：`.rick/super/env_7/`
> 运行窗口：21:24 启动 → 23:20 停止（约 1.9h，计划 10 迭代，实际完成 4.5 迭代）

## 北极星终值

| 指标 | 值 |
|---|---|
| 完成迭代 | 4（迭代 5 进行中被停止；迭代 6 刚起步即被 kill） |
| 精英入格 | 3 格 + 3 pending 延迟入库 |
| gate_result | 3 次终验全部 passed（exploit 拒识在门先证红 40/40 预演确认） |
| 总 token | **935.7 万**（builder 440万 / debug 275万 / coding 220万） |
| 调试分支 | debug agent 真实触发 3 次（迭代 1），含 1 次真实缺陷修复（共享缓冲生命周期） |

## 入格精英（性能口径：score = 相对 BLAS 基线 0.188ms 的加速比）

| 格 | runtime | score | 说明 |
|---|---|---|---|
| np_matmul·naive_loops·col_major·multiprocess | 26.1ms | 0.0072 | 块内 matmul + fork 并行 |
| np_dot·naive_loops·row_major·serial | 141.8ms | 0.0013 | np.dot 派发朴素结构 |
| pure_python·rowwise_vectorized·col_major·serial | 1590.1ms | 0.00012 | 纯 Python 向量化 |

**性能结论**：本次运行无显著性能收益（最高 score 0.007，距 BLAS 平价 1.0 差 140×）。根因是特征空间规模与迭代预算的错配——225 格分殖优先铺开，10 迭代预算内每格只被首访（gen 全 =1），UCB 利用/同格改进不会发生；采到的格多数是结构性慢格（einsum/multiprocess/pure_python 系）。**全链各环节均真实跑通**（builder/coding/debug/adjust 四角色、门禁终验、入格、事件流、心跳、token 计量）。

## 使命目标对照（KR1-KR5）

- **KR1 需求理清** ✅：设计树 3 层过 grilling_gate；9 判断节点两轮 human 裁决全记录（vanilla 首跑 / 40 exploit 样本 / 12 项门禁加固 / 4 轴 225 格 / 10 迭代 / 默认模型 / human 主导干预）
- **KR2 env 合格** ✅：验收自检四项全过；门先证红真实运行通过（40/40 拒识 + 2/2 金样例）
- **KR3 启动成功** ✅：state=running、心跳新鲜、iteration_cursor 递增至 6
- **KR4 进化产出** ⚠️ 部分：≥1 精英入格 ✅（3 格）；**干预投递未发生**（human 未指示任何干预——J7 裁决「纯 human 主导」下无干预是合规的，但 G3 的「干预链路验证」未演练）
- **KR5 干净收尾** ⚠️ 部分：进程树已清 ✅；**session_ended 缺失**（stop 实现缺陷，见下）；收尾报告本文件 ✅

## 发现的框架缺陷（按严重度）

1. **`rick super stop` 不收监督者树**（高）：stop 只杀当时的 worker（SIGKILL 梯），Go 监督者进程活着并按 OnDead 重启逻辑立即拉起新调度器（新会话 #277 + 迭代 6 起步）——需要手动 `kill` 监督者进程组才真正停干净。stop 后 state=stopped 但 live_pid=true 的矛盾状态即此因。
2. **SIGKILL 路径无 session_ended**（中）：重启后的第二会话被强杀，事件流以 #284 截断；两段 session_started（#1、#277）但 0 个 session_ended。事件流重放/审计需容忍此形态。
3. **North Star total_tokens 恒为 0**（低）：status 聚合的 token 口径只计 iteration_ended 的 cost 汇总，而事件里 cost=null（iteration_ended 的 payload.cost 未填充）——token 真值需从 agent_ended.usage 逐条聚合（本报告即按此口径）。
4. **旧 bin/rick 目录模型漂移**（已修复）：10/19 构建的旧二进制把工作区根解析到 `.rick/super_env/`（v1 模型），首次启动落错位置且空 pi_path 被 exit 4 正确拦截；`go build ./cmd/rick` 重建后正常。bin/rick 应纳入 CI 或发布流程防漂移。

## 值得深挖的格与后续建议

- **BLAS 系格未采样**：`full_blas_dispatch` 系（×row/col/tiled × serial/threaded × matmul/ctypes ≈ 20+ 格）是唯一能出 score~1.0 的族——剩余预算应聚焦这里（weight 干预提权或直接指定格）。
- **debug 成本占 31%**：275 万 token 花在迭代 1 的结构性不可达修复上（perf bar 实验 + 3 次尝试全耗尽）。教训：advisory 门槛与格描述子冲突时会烧满 debug_max_attempts。已撤销。
- **分殖优先 × 大格数 × 小预算 = 无利用**：后续跑法建议 max_iterations ≥ 30 或先 weight 收缩采样域，否则 MAP-Elites 只做铺开不做爬山。
- **门禁加固全部实证有效**：40 exploit 全拒（静态检查 6/篡改检测 2/score 界 1/双规模+多seed 31）、双规模不误杀（naive/rowdot/reference 全过）、两遍一致性抓到缓存型 exploit。

## 审计索引

- 事件流（唯一真相源）：`events.jsonl`（284 事件，两段会话）
- 精英档案：`archive.json`（3 格精英 + 3 pending）
- 门先证红预演：`gates/verify_exploits.py`（可重复跑：40/40 + 2/2）
- grilling 设计树：`grilling/design-tree.md`（3 层，过 grilling_gate）+ research-L1/L1-r2 简报
- 代码快照：`codes/iter001~005-*/`
