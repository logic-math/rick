# learning Step 6 任务简报（job_49）

你是 job_49 learning loop 的 Step 6 worker。任务：生成 SUMMARY.md 执行总结。

## 前置检查（已由 parent 完成）

- Step 1a debug 分析 ✅（step1-2-analysis.md）
- 人类已审核 Skills & Loops ✅（human 回复「继续吧」批准）

## 输入（必读）

1. `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/learning/runtime-trace.md`（5 阶段轨迹）
2. `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/learning/step1-2-analysis.md`（Step 1-2 分析）
3. `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/rsi/release.md`（S5 实录：version=47befe6-261030233503）

## 产出

写入 `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/learning/SUMMARY.md`，严格按以下模板（首行必须 `APPROVED: true`）：

```markdown
APPROVED: true

# Job job_49 执行总结

## 执行概述

**项目目标**: 恢复 rick web 生产功能并根治 env_45 自改进（release）崩溃链路：macOS/arm64 上错平台 ELF 投放 + stopProd 漏杀 + 回滚无校验导致生产彻底起不来
**实际完成**: P0 四项根因修复 + 2 个既有 bug 连锁修复（Snapshot 字段/误杀防线）+ P1 五项运维卫生 + proxy 环境反转追加排查；真实 promote 实战验证（stopped=[旧pid] skipped=[]）+ rsi_check 6/6 绿
**整体评价**: ⭐⭐⭐⭐⭐（5 星，理由：7 小时内从崩溃态到根治+文档+知识沉淀全闭环；扣分项：worker 两轮空转 ~1.5h 可避免）

## 关键成就

1. **生产恢复 + 根治**：6137 从死亡恢复（build_id=47befe6-261030233503），四个发布环节全部设防（构建冒烟→promote 校验→认领补强→回滚校验）
2. **真实生产实战验证**：S5 提升中 stopProd 干净收割旧实例（stopped=[46274] skipped=[]）——P0-3 不是只在测试里绿
3. **门禁设计连带发现 2 个潜伏 bug**：lsof -F pc 字段错（快照断言空转/dev-web status 误报 down 的根因）+ cwd 断言过宽（release 会误杀 agent 会话/human 终端——受害方探针方法论的价值实证）
4. **追加排查 10 分钟钉死 proxy 环境反转**（web worker 模型连接全灭）：对照实验方法论（正常 CLI env 差集 → 逐变量对照 → 差异排除）
5. **知识沉淀完整**：1 新 skill（env_drift_probe）+ 2 升级（subprocess_env_isolation / rick-rsi-loop doing 型适配）+ domain 6 条事实

## 问题与教训

### 问题 1: worker 两轮同构「中途思考截断」空转 ~1.5h

**根本原因**: fork 上下文 worker 继承大父会话 + 任务粒度过大 → 分析阶段耗尽预算，输出思考即结束（迭代 2 甚至零代码产出）
**解决方案**: fresh 上下文 + 缩粒度重派；同模式×2 → parent 直接接手（已固化进 rick-rsi-loop 适配节）
**经验教训**: 干预阶梯要提前到第一轮截断就触发（缩粒度），不要等到第二轮确认模式

### 问题 2: start-web.sh 历史经验注释在环境迁移后成为 bug 源

**根本原因**: 老机器「模型网关必须走 proxy」的经验写进注释并被沿用；跨机后 mcli.sankuai.com 内网直达而 proxy 不转发 → 环境反转
**解决方案**: no_proxy 追加 mcli.sankuai.com + 重启 web + worker 同款 env 实测「通」；沉淀 env_drift_probe_skill
**经验教训**: 环境迁移后对「带历史注释的配置项」做实测校准；对照实验（正常进程 env 差集）是 10 分钟钉死根因的钥匙

### 问题 3: rsi_check 取 release.md 首个 version= 行（已知坑第三次复现）

**根本原因**: S3 演练记录含 version= 行且在 S5 实录之前 → 权威校验抓错版本
**解决方案**: 头部三行契约（version= 置首）；domain 已有记录——本次验证其跨场景稳定性
**经验教训**: 机器契约字段的位置约定要在写第一行时就对齐（头部契约）

## 知识沉淀清单

- [x] skills/env_drift_probe_skill/skill.md - 环境反转探针（对照载体→逐变量对照→差异排除→定性→最小修复→迁移校准）
- [x] skills/subprocess_env_isolation_skill/skill.md - 升级：proxy 环境反转案例（+Step 7）
- [x] loops/rick-rsi-loop.md - 升级：doing 型 job 适配节（门禁位置/job 记录树/干预阶梯/证据骨架/release 头部契约）
- [x] domain/bugs.md - +3 条（proxy 环境反转 / lsof -F pc 字段错 / cwd 断言过宽误杀）
- [x] domain/env.md - +3 条（模型网关路由 / darwin shebang argv 语义 / lsof -F 字段语义）

## 遗留（下轮候选）

- release.go prodEnv 透传调用方 proxy 的源码级防线（release 后连通性探针已具备，成本极低）
- catpaw-proxy.ts no-proxy dispatcher（pi 侧定制层）
- P2 决策项：20 条历史挂起会话的人工处置（web UI）、launchd 开机自启、wiki/installation.md macOS 架构约定补充
```

按上述模板**原样落盘**（允许微调措辞但不允许删节；数据以输入文件为准核对——version 号、pid、行数等）。

## 回执

完成后输出：SUMMARY.md 路径 + 首行内容确认（≤100 字）。
