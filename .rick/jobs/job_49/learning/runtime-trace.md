# job_49 执行轨迹（learning 输入）

> 本 job 为 doing/debug 型（无 plan/tasks.json），执行轨迹按阶段记录。
> 时间：2026-10-30 17:11 - 23:55。载体：CLI pi 会话（parent）+ 2 个 worker child + 3 个 research worker。

## 阶段轨迹

### P1 Grilling（17:11-17:30，parent + research worker）
- 读 grilling skill + 交接单 → 现场勘察（发现 6137 已死——交接单 16:50 的手工恢复已失效）
- 建设计树根层 → 派 research-L1（第一次派发失败：任务文本含 markdown 反引号嵌进 JS 模板字符串 → workflowScript 语法错误，**修复：简报落文件+派发只传路径**）
- research-L1 返回 10 条代码事实（file:line 锚点全部核实）→ 提炼 6 个判断节点 → human 批量裁决「全按推荐」
- 设计树 3 层完整 + grilling_gate 通过（初次失败：第 2/3 层缺显式「全消解」声明——补声明后通过）
- **教训**：grilling_gate 对「本层全部消解，无判断节点」要求显式记录，注释里顺带一句不够

### P2 RSI S1 隔离开发（21:00-22:15，worker×2 + parent 接手）
- 前置清理：36 个 10-18 泄漏的测试 fake 进程（Q4 裁决）；dev 树残留的 13:08 事故 stub job_49 目录（防 S5 合并冲突）
- 迭代 1（worker，90min）：完成 778 行实现（P0-1/2/4 完整、P0-3 框架）但 3 个 P0-3 测试红；**回执异常（中途思考截断）**
- 迭代 2（worker，60min）：只做了失败分析定位（方向正确）没写代码；**同样中途思考截断**
- **parent 接手**（干预阶梯第三级：重派仍失败 → 亲自接手）：
  - procfs +Command(pid)（darwin ps -ww -o command= / linux /proc/cmdline）
  - claimBySingleton 三级名义链——**关键实测发现：macOS shebang 进程 argv[0] 是解释器路径，脚本路径在 argv[1]，只看 argv[0] 会漏**
  - stopProd 候选集补端口占用者 + web.lock 持有者
  - TestReleaseAutorollbackRefusesBadRollbackTarget 语义修正（worker 混淆了 autorollback 走 PrevVersion vs CLI rollback 走 .last 两条路径；改为 11:53 事故精确复现）
- **教训**：worker 连续 2 轮同模式失败（分析完输出思考就结束）→ parent 应更早识别该模式直接接手；fork 上下文的 worker 大任务易耗尽预算在分析阶段

### P3 RSI S2 门禁（22:15-22:40，parent 设计+执行）
- 设计 gate1（代码层）/gate2（6199 行为演练）
- gate2 三轮收敛：①fake prod repo 缺 go.mod（release 校验仓库形态）②start script 空脚本不启动服务 ③上轮 probe 残留实例占 6199 端口干扰（**这反而实证了端口认领链会收割跨 state-dir 残留**）
- **S2 期间的重大发现**（设计门禁时主动安全实验发现两个既有 bug）：
  - B1：procfs Snapshot cwd 批量采集字段错误（lsof -F pc 把 comm 名当路径——快照断言全失效、dev-web status 恒报 down 的根因）→ 修 -F pn
  - B2：ownedByProd cwd 断言过宽（cwd==生产树即认领——**release 会误杀生产树上工作的 agent 会话/human 终端**！含安全探针实验验证）→ cwd 匹配须加 rickNamedProcess 名义佐证 + 新增误杀防线测试
- **教训**：门禁设计的「判别力验证」环节（修复前红/修复后绿）天然是安全实验场——主动构造受害场景能发现隐藏 bug

### P4 RSI S3-S5 交付（22:40-23:45）
- S3 dry-run prod_untouched=true（注意 rsi_check 从 release.md 取**首个** version= 行——头部三行契约，domain 已知坑复现）
- S4 human 批准（「批准执行吧」）
- S5 前置 commit：dev 树（47befe6 修复）+ main 树（7cf11c8 证据+交接单）
- S5 真实提升：`RELEASE_MERGE merged=true commit=de906bd` / `RELEASE_OK version=47befe6-261030233503` / **stopped=[46274] skipped=[]（P0-3 真实生产实战验证）** / 6137 恢复 health ok
- M4 运维收尾：P1 全部五项（start-web.sh 加固版冷启动+幂等验收 / deploy-web.sh 路径 / ~/bin/rick 软链自动跟版本 / 3 份 ELF 移 legacy-linux / ~/.rick/bin/rick 换 Mach-O）
- rsi_check 6/6 全绿

### P5 追加排查（23:45-23:55，human 报 web 上 learning 报错触发）
- 现象：web 导入本会话（ed8419ef "easy job_49 (imported)"）执行 learning → 「无法连接 pi runtime 模型不能访问」
- 排查链：会话 active 但无 worker 进程 → AgentEnv 继承 web env → pi 模型链路（catpaw-proxy → mcli.sankuai.com）→ **对照实验铁证**：
  - 旧 no_proxy + proxy → pi --print = `Connection error.`（复现）
  - no_proxy+mcli.sankuai.com + proxy → `通`（恢复）
  - proxy 能转发 github（200）但不转发 mcli（内网域名 HTTP 000）；CLI 会话无 proxy 直连正常
- 根因：**环境反转**——老机器「模型网关必须走 proxy」的经验在跨机迁移后反转（mcli 成内网直达，proxy 反而转发不了）
- 修复：start-web.sh no_proxy 追加 mcli.sankuai.com + 重启 web + worker 同款 env 实测「通」
- 遗留：prodEnv 透传调用方 proxy（release 从旧 no_proxy shell 跑会复发）；catpaw-proxy.ts 用 no-proxy dispatcher 是候选源码级修复
- **教训**：部署环境迁移后，历史经验注释（「丢了 proxy 会全部 Connection error」）本身就是 bug 源——环境假设要定期实测校准

## 关键数据
- 全程耗时：~7h（含 grilling 45min、S1 75min×3 轮、S2 25min、S5+运维 60min、追加排查 10min）
- worker 派发：5 个（research-L1×2 + doing-1 + doing-2 + learning 系列待派）
- 代码产出：+899/-32 行（release.go/procfs×3/relocate×2 + 测试 477 行新增）
- 修复项：P0×4 + 既有 bug×2 + P1×5 + 环境级 1（proxy 反转）
