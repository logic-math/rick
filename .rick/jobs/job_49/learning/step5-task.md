# learning Step 5 任务简报（job_49）

你是 job_49 learning loop 的 Step 5 worker。任务：整理 Domain 事实知识。

## 输入（必读）

1. `/Users/logic/.rick/pi/agent/sessions/placeholder` —— 忽略此行（占位）
   实际输入：
   - `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/learning/runtime-trace.md`（执行轨迹）
   - `/Users/logic/workdir/AI_CODING/rick/.rick/jobs/job_49/doing/debug/bug1-web-worker-model-connection.md`（proxy 根因）
   - `/Users/logic/workdir/AI_CODING/rick/.rick/dream/prompts/skill_gen_domain.md`（domain 提取规范）
   - `/Users/logic/workdir/AI_CODING/rick/.rick/domain/bugs.md`（现有 bugs——追加前先查重）
   - `/Users/logic/workdir/AI_CODING/rick/.rick/domain/env.md`（现有 env 事实——追加前先查重）

## 追加规则

**追加不重复**：先 grep 现有文件确认没有同条目（按根因关键词查重），有则更新验证状态与出处，无则新增条目（含「首次发现: job_49」「验证状态」）。

## 待沉淀事实清单（从本 job 提取）

### bugs.md 候选（3 条）

1. **web worker 模型连接 proxy 环境反转**（bug1 已闭环）：
   - 根因：start-web.sh 按老机器经验 export proxy 三件套；跨机迁移后 mcli.sankuai.com（pi catpaw-proxy 模型网关）成内网直达域名，proxy 10.229.18.27:8412 不转发它（实测 github 200 / mcli HTTP 000）→ web worker 全部模型 Connection error，而 CLI（无 proxy env）正常
   - 精确解决：start-web.sh 的 no_proxy 追加 mcli.sankuai.com + 重启 web；验证 = worker 同款 env `pi --provider catpaw-proxy --model glm-5.3 --print "通"`
   - 遗留：release.go prodEnv 透传调用方 proxy——release 从旧 no_proxy shell 执行会复发（S5 实例中招过，被 start-web.sh 重启覆盖）；候选源码修复=catpaw-proxy.ts no-proxy dispatcher（pi 侧）
   - 验证状态：✅ 环境级修复已落地（2026-10-30）；源码级未修

2. **procfs Snapshot cwd 批量采集字段错误（lsof -F pc 把 comm 当路径）**（已源码修复+release）：
   - 根因：`lsof -w -F pc -d cwd` 的 c 字段是进程名（comm），代码把它当 cwd 路径解析 → Snapshot.Cwd 恒为 "sleep"/"node" 等进程名 → ownedByProdSnapshot 快照断言全失配 → findProdPIDs 空转、dev-web status 恒报 state=down（Running 判定盲区）
   - 精确解决：改 `-F pn` + parse 'n' 字段（commit 47befe6，version=47befe6-261030233503 已 release）；修复后 DEV_STATUS state=up fingerprint=ok
   - 验证状态：✅ 已修复并 release

3. **ownedByProd cwd 断言过宽——release 误杀生产树上会话/终端**（已源码修复+release）：
   - 根因：cwd==prodRepo 单独成立即认领停止；生产树根=git 仓库根，agent 会话/human 终端 cwd 都在那 → release stopProd 会误杀它们
   - 精确解决：cwd 匹配须加 rickNamedProcess 名义佐证（exe→argv→comm 三级）+ TestRestartRefusesToKillCwdSquatterInProdTree 防线测试（同 commit）
   - 验证状态：✅ 已修复并 release

### env.md 候选（3 条）

1. mcli.sankuai.com 是内网直达域名（pi catpaw-proxy baseUrl；key 在 ~/.config/mcopilot-cli/.config.yaml 的 AUTHORIZATION 行）；proxy 10.229.18.27:8412 能转发外网（github 200）但不转发它（HTTP 000）
2. macOS shebang 进程的 ps -ww -o command= 输出：argv[0] 是解释器路径（如 /Library/.../Python），脚本路径在 argv[1]——名义判定要扫全 argv；ps comm= 同样是解释器名
3. lsof -F 结构化输出字段语义：p=pid、c=comm（进程名）、n=name（文件路径）——批量 cwd/txt 采集用 pn

## 写域与禁区

- 只追加：`.rick/domain/bugs.md`、`.rick/domain/env.md`（不新建其他 domain 文件——本次事实归这两类）
- 追加格式对齐文件内既有条目风格（含根因/精确解决步骤/首次发现/验证状态）
- 禁止：改既有条目内容（只允许在验证状态处追加更新）；执行 git；碰 skills/loops

## 回执

完成后输出：两个文件的追加条目数 + 每条一行摘要（≤200 字）。
