# super 验收复验 2 — 设计树（env_5 / matmul 512×512）

> 活文档：逐层追加。层号 + 模块 + pipeline + 判断节点。
> 调研简报：research-L1.md（第 1 层）/ research-L2.md（第 2 层）/ research-L3.md（第 3 层）。

---

## 根层 OKR

**O（目标）**：完成 super 模式验收复验 2——在 env_5 工作区以「优化 512×512 矩阵乘性能（纯 Python 三重循环起点）」为任务域，端到端跑通 super 全流程（grilling → gen_env → 门先证红 → 后台进化 → 干预投递 → 干净停止），并以真实进化结果证明系统有效。

**KR（关键结果，演绎推出 O）**：
- KR1 需求树落盘：grilling 完成且 grilling_gate 通过（本层工作）
- KR2 env 合格：gen_env 产出 matmul env + 验收自检四项全过（参考实现过正确性门 / bench 出分 / exploit 全拒 / naive 起点过门）
- KR3 门先证红通过且进化真实运行：start 放行（exploit 拒识 100% + 金样例零误杀）、state=running、心跳新鲜、iteration_cursor 递增、事件流完整、北极星可读
- KR4 干预可用：干预经 `rick super event` → interventions/ 队列 → 事件流出现 intervention（source=human）
- KR5 停止干净：收尾后 state ∈ {stopped, completed}、session_ended 落盘、无孤儿进程、全程零直写 events.jsonl/archive.json

（KR1∧…∧KR5 ⟹ O：流程完备 + 进化有效 + 人机接口可用 + 生命周期干净。）

---

## 第 1 层：全局验收层

**pipeline**：grilling（需求树）→ gen_env（env 建设）→ start·门先证红 → monitor（status/事件尾）→ intervene（rick super event）→ stop/收尾报告

**模块与职责**：
| 模块 | 职责 | 输入 → 输出 |
|---|---|---|
| M1 grilling | 需求追问，设计树落盘 | requirement.md → grilling/design-tree.md（过 grilling_gate） |
| M2 gen_env | env 建设（任务/评分/门禁/特征空间/初始解/验收自检） | matmul 模板 + 本层裁决 → env_5/env/ + env_5/gates/ |
| M3 启动 | pi_path 注入 super.json + `rick super start --daemon` | env → 门先证红放行 + 后台调度器拉起 |
| M4 监控 | status --json + 事件尾 + 北极星巡检 | 事件流 → 三类信号识别（env_failure 连发/零入格/token 异常） |
| M5 干预 | `rick super event` 投递 + 效果验证 | 干预意图 → intervention 事件（source=human） |
| M6 收尾 | 停止 + 状态确认 + 终值报告 | 运行态 → state=stopped/completed + session_ended + 北极星终值 |

**调研**：research-L1.md（代码库考古：历次会话全部止步骨架；依赖链四项自检；干预/首迭代/自然完成机制源码级确认；模板参数基线）。

**判断节点（L3 批量追问——J1-J3，附推荐）**：

- **J1 验收成功的量化判据**：
  - A=仅流程完备（G1-G4 达成，入格数允许 0）
  - B=流程完备 + 真实进化证据（≥1 个 elite_admitted 入格 + naive 起点与入格精英的 runtime 有可见量级差距 + ≥5 真实迭代）
  - C=B + env 热更新演示（Step 5：改门禁重跑门先证红）
  - **推荐 B**：验收复验的核心价值是证明「进化真的发生」，不只是管道通；C 的热更新属 env 升级路径，超出本次复验必要范围（可留专项验收）。
- **J2 运行预算与规模**：
  - A=`--max-iterations 12` + `--max-wall-time 2` 双保险（先到为准）
  - B=仅迭代上限 12
  - C=仅墙钟 1h
  - D=不限，人工观察手动停
  - **推荐 A**：每迭代 4 agent 真模型（glm-5.3）跑 3-10 分钟，12 迭代 ≈ 36-120 分钟；2h 墙钟防失控；adjust_interval=5 在 12 迭代内触发两轮（含干预效果的间接生效窗口）。
- **J3 干预演示类型（KR4 的演示载体）**：
  - A=weight 降采样权（如对某格投 0.3，再视情况投回 1.0）
  - B=pause + 恢复
  - C=resurrect（复活灭绝格——需先有灭绝事件，首跑无素材）
  - D=stop 类（演示即结束运行）
  - **推荐 A**：无破坏性、事件流可验证（intervention source=human 必落）、后续经 adjust agent 可观察 weight_changed；C 无素材、D 会提前终止验收运行。

**OKR 充分性自检**：M1-M6 全部达成 ⟹ KR1-KR5 ⟹ O。覆盖面检查：需求澄清（M1）→ 建设与验证（M2）→ 生命周期启动（M3）→ 运行观察（M4）→ 人机接口（M5）→ 生命周期收尾（M6）——无遗漏，互不重叠。✓ 达标，待 J1-J3 裁决后下钻第 2 层。
