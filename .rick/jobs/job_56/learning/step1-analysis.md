# Learning Step 1：job_56 执行记录分析（parent 接手产出）

## ① 任务完成状态表（plan DAG 5 task 全 success）

| Task | 内容 | 状态 | 验证 |
|---|---|---|---|
| task1 | M1 DB 地基：sqlite 五表+进程锁+槽位资源调度 | success | 全部单测绿+消费方迁移 |
| task2 | M2 env 体系：gates.py 多维融合分协议+env_v2 模板+gen_env 四步 | success | 71/71 test_gates 绿 |
| task3 | M3 adjust 7 步 subagent 协议 | success | 7 步流水线实战 25+ 轮 |
| task4 | M4 实验 sub loop 确定性（builder 仿真/coding L1 预检/debug 记录） | success | 585 tests 全绿 |
| task5 | M5 集成验收（e2e 全链路+全量回归+文档收口） | success | 585 tests + go build/test |

## ② 已解决问题根因清单（按问题域归类——实时解决，debug/ 未落盘）

### 并发/资源域
- **sqlite segfault**：find_resumable_experiment 裸用 conn 与心跳/adjust 线程并发 → C 层崩溃；修复：持 RLock
- **多 wrapper 混战**：watchdog restart 的 stop→start 竞态（旧 wrapper/py 残活 → py 互撞进程锁 exit 5 群杀）；修复：restart 手柄加彻底清理（杀+等死透+清锁再 start）
- **pending 重评估僵死循环**：老 pending 项 code_path 相对名（cwd 解析恒败）+ item 缺字段致 invalidation append raise → 每 tick 重入同批死循环 1.5h；修复：drain 即持久化+路径绝对化+per-item try
- **epoch 重评估堵主循环**：slots 空时串行跑满轮（epoch 150/300 各堵 1.7h）——识别为设计内成本（未修，记提案）

### 部署/版本域
- **Go 二进制 5 天未重建**（#34）：release 链断裂；修复：重建+releases/current 更新纪律
- **--version 语义化版本**：release 只支持 sha7-时间戳 → 加 --version flag（7.0.0 发布用）+ versionRe 兼容 semver 防 GC 误删
- **发布半程**（#40）：bin/rick 建了但 releases/current 指旧；修复：发布链收尾
- **安装快照机制**：superopt 读取优先级翻转 ~/.rick 最优先 + release 联动部署（job_57 协作产出）
- **老 superopt 克隆清理**：INFER_RSI/superopt（sankuai git 时代）与快照机制冲突风险 → 打档删除

### 采样/策略域
- **null 格采样稀释**：分殖优先只管 visit==0，null 格（visit 1-3 采过没成）回全量加权随机被重访格稀释（50 采样 0 中）；修复：null_boost ×5（7%→34.4%）
- **suspicious_gain 拦截惩罚真突破**：大涨幅（≥20%）被复验拦截拖慢 68× 霸榜；修复：移除上限拦截（保留 ±5% 噪声带）
- **debug 条件触发**：自检通过不跑 debug（假阳性漏网）；修复：admission_review 每实验无条件复核
- **OKR 锚定缺位**：adjust 7 步无目标锚 → 12 项提案全跑偏基建自优化；修复：env.json okr 独立要素 + 7 步 prompt 头部思考锚（提案 7/8 转实验策略类）
- **research --cell 参数缺失**：brief 恒占位符（214 份报告「跳过检索」）；修复：--cell 支持+按算子链生成针对性查询

### 知识管理域
- **domain_knowledge 膨胀**：305KB/36 节叙事（每 iter 追加无去重）；修复：行为手册化（11.8KB：bug 三段式+约束命令式+去重合并+渐进式披露归档 env/knowledge/ 6 份）

### 验证器域
- **gate 无读数误杀**：value=null 与超差共用 -2；修复：-6 分类（transport/oom/crash）+ 重试一次
- **GN 族噪声地板误杀**：ref 侧 fp32 噪声 7.4× 超差；修复：ref_floor_suspect 标记+RTNE 配方导航
- **profile 跨格污染**：--problem 不传致 27/27 条错数据；修复：显式传参+本格题副本

### 基础设施自愈域
- **停机 11h 无人知晓**（job_54 时代教训）：修复：launchd 巡检链（web API prompt 注入——session 生命周期绑定）+ 探活驻留（#26：额度耗尽不退出，恢复自动续跑——首战零空窗实证）
- **预算墙 3h 残留**：历史启动 flag 残留致墙杀；修复：ceiling 规则（≤168h/1000M adjust 自调）+ 续跑章法（config 体检+工作量匹配+旗标覆写）
- **模型同源风险**：中控与 runtime 同用 GLM；修复：模型隔离（runtime → catpaw/deepseek-v4-pro）

## ③ 未解决/遗留问题

1. **epoch pending 重评估串行堵主循环**（每次 1.7h）——设计内成本，改进提案在队（重评估与派发并行化/限时限项）
2. **A-02 盲窗死等**：in-flight adjust 已死仍等满缓冲期（应查线程存活）——#47 提案在队
3. **多会话共享仓库的构建一致性**：release 构建可能编进工作区未 commit 改动（可追溯性受损）——结构性问题待方案
4. **pi worker 运行时 MessagePort 异常**（今天 2 次：巡检/learning-step1）——间歇性，重派可绕
5. **kernelbench-dev 待裁决 10 项提案**（#45-#54）+ 主裁决（续期 600 vs 收官）

## ④ 执行轨迹分阶段摘要

**重构期**（task1-5）：DB 五表地基 → env v2 体系 → 7 步协议 → sub loop 确定性 → 集成验收（585 tests）——DAG 分层并行（L1 task1∥task2 → task3 → task4 → task5），gate 门禁逐层验收
**运行期**（96h run）：0→100 格覆盖（null_boost 收官）· 24 项提案实施部署 · 三层误诊（僵死循环 6h）· release 7.0.0 语义化版本发布 · 模型隔离
**收官期**：KR1 100/100 达成 · geomean 2.174× · fast_1=100%（vs 前沿 <20%）· 达帽 500 优雅收官
**learning 期**（本阶段）：执行记录分析 → 评估 → skill/loop 沉淀 → domain 事实 → SUMMARY
