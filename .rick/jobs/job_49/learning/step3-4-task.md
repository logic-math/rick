# learning Step 3-4 任务简报（job_49）

你是 job_49 learning loop 的 Step 3-4 worker。任务：提取 Skill（Step 3）+ Loop 模式（Step 4）。

## 输入（必读）

1. `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/learning/runtime-trace.md` —— 5 阶段执行轨迹
2. `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/learning/step1-2-analysis.md` —— Step 1-2 分析（改进建议是本步的直接素材）
3. `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/debug/bug1-web-worker-model-connection.md` —— proxy 环境反转 debug 记录
4. `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/prompts/skill_gen_skill.md` —— skill 格式规范
5. `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/prompts/skill_gen_loop.md` —— loop 格式规范

## Step 3：提取 Skill

**新建一个 skill**（本 job 最核心的可复用方法论）：

`.rick/skills/env_drift_probe_skill/skill.md` —— **环境反转探针**（环境假设校准方法论）
- 触发场景：当环境迁移/跨机部署后出现「以前好用的配置现在报连接错误/行为反转」，或当代码/脚本注释里写着「必须 X 否则 Y」而环境已经变更时；信号词：「Connection error 但 CLI 正常」「历史经验失效」「环境反转」
- 预期效果：10 分钟内钉死「环境假设是否反转」，避免在错误层面（代码/网络/权限）盲目排查
- 核心内容（从 bug1 的排查链提炼）：
  1. 对照载体：找一个**工作正常的同类进程**（本案=CLI 会话），对比它的 env 与故障进程的 env（差集=嫌疑面）
  2. 逐变量对照实验：同环境只改一个变量（no_proxy 加/不加 mcli.sankuai.com → Connection error vs 通）
  3. 差异排除法：测 proxy 对其他目标的转发能力（github 200 vs mcli 000）排除「proxy 整体故障」假设
  4. 结论定性：环境反转 = 老经验在新环境为反（不是代码 bug 不是网络故障）
  5. 修复落点：最小改动（no_proxy 追加）+ 沉淀到部署配置（start-web.sh）+ 遗留标记（源码级修复候选）
  6. 预防：环境迁移后对「带历史注释的配置项」做实测校准清单

写完后检查 `.rick/skills/subprocess_env_isolation_skill/skill.md`：在该 skill 的触发场景/核心内容里**追加一小节**「proxy 环境反转案例（job_49）」——3-5 行引用 env_drift_probe_skill（升级既有 skill 而非重复创建）。

## Step 4：提取 Loop 模式

**升级既有 loop**（不新建——本 job 走的是 rick-rsi-loop 标准链，但暴露了它对 doing 型 job 的适配缺口）：

`.rick/loops/rick-rsi-loop.md` —— 追加一节「doing 型 job 的适配（job_49 实测）」：
- 场景：job 由 `rick doing`/CLI 直接创建（无 plan/tasks.json/gates），需要走 RSI 修改源码并送达生产
- 适配要点：
  1. 门禁位置：`doing/tests/gate{N}.py`（代替 plan/gates/）——rsi_check 只校验 gates.md 证据行，不查门禁文件位置
  2. job 记录位置：保持 main 树 `.rick/jobs/<job>/`（与 S5 合并方向 dev→main 兼容：dev 无这些文件，合并不冲突）；rsi_check 从 main cwd 跑（按 cwd 解析 job）
  3. 干预阶梯（worker 截断处置）：截断×1 → 缩粒度 fresh 重派；截断×2（同模式且零代码产出）→ parent 直接接手关键部分
  4. 证据骨架仍用 `rsi_check --init`（从 main 树 cwd）
  5. release.md 头部三行契约（version= 置首——rsi_check 取首个匹配行，domain 已知坑）
- 注意：不要改动 rick-rsi-loop.md 的主体结构（S0-S7 状态机），只在文末追加这一节（标注 job_49 出处）

## 写域与禁区

- 写：`.rick/skills/env_drift_probe_skill/`（新建）、`.rick/skills/subprocess_env_isolation_skill/skill.md`（追加小节）、`.rick/loops/rick-rsi-loop.md`（文末追加节）
- 禁止：新建其他 loop；改 rick-rsi-loop 主体；执行 git；碰 domain（Step 5 处理）
- 产出前自查：skill.md 三部分（触发场景/预期效果/核心内容）齐全；loop 追加节含 job_49 出处标注

## 回执

完成后输出：三个文件的路径 + 每个文件一句话摘要（≤200 字总计）。
