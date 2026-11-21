---
name: super-tech
description: super 进化系统的技术讲解——三层架构/7 步 adjust 流水线/QD 搜索理论/优化路线图，帮助 human 学习与理解系统设计（「super 怎么工作的」「为什么这么设计」「接下来优化什么」）
execution_plane: local
---

# skill:super-tech

## 触发场景

human 想学习/理解 super 进化系统的工作原理与优化路线（「super 怎么工作的」「为什么 7 步」「QD 是什么」「接下来怎么优化」）。

（激活词：super 原理、进化系统、学习 super、技术讲解、优化路线）

## 1. 三层架构（一张图）

```
中控会话（human 接口：启停/干预/报告——super-loop）
  ↓ rick super start
Go PySupervisor（进程监督：心跳/杀梯/重启）
  ↓ python3 -m superopt
py 调度器（唯一真相源写者——super.db 六表）
  ├─ 采样器：加权随机（概率 ∝ adjust 设的 weight——格=解空间单元）
  ├─ 实验链（每实验=解一题）：builder(设计) → coding(一次写对+自验) → debug(排障/作弊检) → gate_check(E2E 终验)
  └─ adjust 流水线（每 N 迭代）：7 步语义决策 → 格动作（升权/降权/灭绝/加格）
```

## 2. 核心理论（QD 搜索——质量多样性）

- **解空间是离散分片的流形**：每片=格=一种解的模式（特征轴定义切分维度）
- **MAP-Elites 档案**：每格存历史最优解（精英）；全局=格覆盖（多样性）×格内最优（质量）
- **饱和=主要矛盾的转移**（不是数值阈值）：服务旧矛盾的格降权/灭绝，新矛盾方向加格/升权
- **影子价格**：主要矛盾=当前卡住的最贵约束（放松它收益最大）——adjust step3 从视角推矛盾、step4 排序的数学根基（KKT 对偶）

## 3. 7 步 adjust 流水线（answer-isolation——每步独立上下文）

| 步 | 角色 | 职责 | 为什么独立 |
|---|---|---|---|
| 1 | adjust_fact | 问题还原（读 DB 六表+humans 干预=human 价值判断） | 只还原事实，判断留给后步（防锚定） |
| 2 | adjust_perspective | 视角生成（web 检索+3 视角+归纳演绎证明） | 跨领域视角=启发式跳跃点（分片跳跃的方向源） |
| 3 | adjust_contradiction | 矛盾生成（**每视角 1 个并行 subagent**→活跃约束集→影子价格） | 3 视角互不污染（并行=3 session） |
| 4 | adjust_judge | 主要矛盾判定（全局性/根本性/决定性排序） | 用 step3 的产物做裁判（事实与判断分离） |
| 5 | adjust_reversal | 辩证逆转（矛盾→手段：「A 是阻碍」反转为「A 是资源」） | 创造性步骤（LP 对偶的语义版） |
| 6 | adjust_space | 格动作（proposals：升权/降权/灭绝/加格）→ 权重生态 | 执行步骤（输出过 hook 结构检） |
| 7 | adjust_env | 实验自改进（bug→domain_knowledge 直写；skill/验证器反思→humans 提案队列） | env 进化通道（验证器是测量地基——改验证器须 human 裁决） |

## 4. 关键机制速查

- **权重生态**：weight=采样份额（0=功能性灭绝，谱系保留可复活）；防震荡纪律（连续降权须新证据间隔）
- **作弊防线**：gates/ 禁读（hook 拦截 builder/coding/debug）+ 四分布隐藏测试 + 权重同步契约（带参层同 state_dict）+ 全零量级检 + passthrough 静态检
- **失败路径**：debug 主动上报 `experiment_failed` → 调度器跳过 E2E（省 30s/次）——信任 debug 的排障结论
- **验证金字塔**：L1 秒级预检 → L3 快速自证 → E2E 全协议终验（同一实现入口——口径不漂移）
- **env_lint**：13 规则验收器（每条=一个实测坑）——gen-env 生成闭环的质量闸门

## 5. 优化路线图（当前坐标与下一步）

**已完成（job_56）**：
1. ✅ sqlite 六表 DB 化（cells/experiments[三 session 列]/humans[两类队列]/events/resources/adjusts）
2. ✅ 7 步 adjust 流水线（step3 三视角并行+humans 干预注入+事后验证）
3. ✅ 并行实验（实验槽/环境槽二级限额+槽位先占防泄漏）
4. ✅ KernelBench-Verified 协议（四分布+TF32 baseline+mem_eff）+ L2 100 题刷榜模式
5. ✅ env_lint 验收器+gen-env 五步流程（裁决点清单/真实演练/说明书）

**下一步（按价值排序）**：
1. **框架链路演练**（R14——本轮实测盲区：演练走手工命令没走 CascadeGate 真实调用链，题库模式 --problem 断链没被抓）
2. **远程链路健壮性**：remote_link 重试已加；再加**资源预检**（跑前查 GPU 显存余量——外部占用早发现）+ 运行中资源监控（OOM 防线）
3. **三臂对照实证**（pipeline vs 单 agent vs 确定性状态机——7 步流水线的决策质量 A/B——原创实证义务）
4. **Step7 prompt 进化闭环**：五 subagent 提示词=进化靶子+held-out 正则化（防过拟合）
5. **跨任务泛化**：L1/L3 切片轮次+跨 env 经验迁移（QD 档案整包迁移——rick 独有机会）

## 6. 常见问题（FAQ）

**Q: 为什么 weight=0 不删格？** 谱系保留（resurrect 后门）——失败知识结构化沉淀，矛盾转移回来时可复活。

**Q: 为什么 verify 在 gates/（agent 禁读）？** 验证器是测量地基——被测对象改测量仪器=作弊通道（step7 也只能提案，human 裁决）。

**Q: 为什么 benchmark 用中位数不是均值？** H20 实测噪声 <2%，中位抗离群（官方用 mean——差异已在报告标注）。

**Q: 格动作怎么决定？** step6 的 LLM 语义推理（矛盾归因链）——不是数值规则（量化规则=把语义流形投影到低维，丢可能性）。

**Q: 怎么判断系统在有效进化？** stat-super skill 的报告指标：Coverage 推进（已解格数↑）+ 最优分上升 + fast_1.0（提速 2× 题数）↑ + 无 infra 误杀浪费。
