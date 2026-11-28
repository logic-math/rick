# Rick 架构设计

## 技术栈

- 语言: Go 1.21+（主程序），Python 3.8+（辅助脚本和测试脚本）
- 框架: Cobra（CLI 命令框架），Goldmark（Markdown 解析）
- 测试: Go testing 标准库，Python unittest，Bash integration tests
- 其他: Git（版本管理），Claude Code CLI（AI agent 集成）

## 模块划分

```
cmd/rick/main.go              # 入口，VERSION 常量
internal/
  cmd/                        # Cobra 命令处理（plan/doing/learning/easy/dream/ctrl/human-loop）
  executor/                   # 任务执行引擎（runner.go/doing_check.go）
  prompt/                     # 提示词管理（builder/manager/templates via embed.FS）
  workspace/                  # 路径管理
  parser/                     # 内容解析
  git/                        # Git 操作
  callcli/                    # Claude Code CLI 集成
  agent/                      # 接口契约（AgentSession/AgentExecutor/ToolCall）
    claudecode/               # 唯一实现，仅在 doing.go 组合根中实例化
  actpath/                    # act-path 生成
```

## Prompt Context 注入体系

`internal/prompt/context_helpers.go` 提供两个并行的 context loader，均在 doing/easy/plan/dream 启动时注入：

| 函数 | 读取源 | 注入变量 | 提取字段 |
|------|--------|---------|---------|
| `LoadLoopsContext` | `.rick/loops/*.md` | `{{loops_context}}` | frontmatter `name` + `trigger` |
| `LoadSkillsContext` | `.rick/skills/*_skill/skill.md` | `{{skills_context}}` | `# skill:` 标题 + `## 触发场景` 首行 |

两者格式对称：`- **{name}**：{trigger/触发场景首行}`。新增同类 context loader 时，遵循相同模式。

**来源 Job**: job_23

## DIP 组合根模式

**核心约束**：`doing.go` 是唯一 import `internal/agent/claudecode` 的地方。

```
doing.go（组合根）
  └── import claudecode ✅
runner.go / executor.go / actpath/
  └── import agent（接口）✅  NOT claudecode ❌
```

**验证命令**：
```bash
grep -r "claudecode" internal/executor/ internal/actpath/
# 应无输出
```

**nil guard**：`actpath.Generate(session, outputFile)` 中 session 为 nil 时跳过，不 panic。

## agent 接口模块（internal/agent/）

```go
type AgentSession interface { /* Claude Code 会话 */ }
type AgentExecutor interface {
    Execute(session AgentSession, task Task) (Result, error)
}
type ToolCall struct { /* 工具调用记录 */ }
```

`claudecode` 子包为唯一实现，仅在组合根 `doing.go` 中实例化。

## act-path 生成模块（internal/actpath/）

```go
func Generate(session AgentSession, outputFile string) error
```

- 不 import 任何具体 agent 实现（仅依赖 `internal/agent` 接口）
- 输出三节：执行摘要 / 行为轨迹 / Agent 最终输出
- 每个 task 的 `agentExecutor.Execute` 完成后调用

## human-loop 模块(v8.3, job_57 彻底重构)

- 命令:`rick human-loop <topic>`——启发 human 进行原创性思考的思考门禁会话
- **v8 彻底重构**(2026-11, job_57):删除 v7 全部教学机制(ZPD/费曼/课程/大纲/recall/文件状态机),重写为极简「思考门禁 loop」
- **架构文件**(go embed 内嵌模板):
  - `templates/sense_loop.md` — main agent 协议(极薄编排层:五阶段门禁循环 + thought_path 写入契约 + 输出纪律)
  - `templates/skills/sense_reconstruct|perspective|contradiction|reversal|critique.md` — 五阶段 skill(每阶段一个)
  - `templates/skills/orig.md` — orig agent(概念空间随机游走者,唯一提示词:「给用户返回怎样的内容才能引导他完成本阶段的思考目标?」)
  - `templates/skills/teacher_polish.md` — 呈现层方法论(科普视角,ASD-STE100 短句规范,递归解释:零领域专用词汇,每个概念递归以通识内容讲清楚)
  - `templates/skills/sense.md` — SENSE canon(五步权威文本,与 dream 共用)
- **单 agent 架构**:仅 orig(rickAgents 中唯一 human-loop agent;research/syllabus-builder/lesson-writer/recall-review 已删除)
- **运行时产物**:`.rick/draft/loops/loop_N/` 下 `thought_path.md`(唯一产出物:human 原话 + AI 启发的思考轨迹)+ `orig/stage{N}_{seq}.md`(orig 调研中间结构)+ `session_id`(resume)

### 五阶段思考门禁流程(v8.3)

| 阶段 | 名称 | 门禁标准(可判定) |
|---|---|---|
| 1 | 问题还原 | 期望/现状/差距三结构化表述说清,价值判断与事实判断分离 |
| 2 | 视角生产 | 描述选择视角的应用价值,能描述其内在因果运动 |
| 3 | 判断矛盾 | 基于视角解释问题,定义决定性/根本性/全局性矛盾及其因果运动 |
| 4 | 辩证逆转 | 符合逻辑地回答:当阻碍 X 是目标 Y 的必要条件时应当如何 |
| 5 | 良质批判 | 回顾思考假设是否可证伪,推导过程逻辑性 |

每阶段循环:读阶段 skill → 派 orig(概念随机游走:同中有异/异中有同的跳跃检索)→ teacher_polish 润色呈现(递归解释)→ 概念地图收尾引导(human 串联概念、用自己的话表达)→ main agent 内联门禁判定 → 通过推进/未过指出缺口重调 orig(无重试上限,推进权在 human)。

### 关键设计决策(v8.3)

1. **思考门禁制**:每阶段一个 skill 承担双目标(启发内容 + 门禁验证);门禁由 main agent 内联判定(无独立 gate agent)
2. **orig 本质 = 概念空间随机游走**:跳跃检索连接「同中有异、异中有同」的概念,唯一目的 = 帮 human 突破当前思考门禁
3. **递归解释关键原则**:呈现零领域专用词汇,每个概念递归以通识内容解释(呈现以概念为主题)
4. **发散-收敛结构**:随机游走 + 递归解释 = 发散(建概念地图);收尾引导 human 用自己的话表达 = 收敛(过门禁)
5. **thought_path.md 唯一产出物**:思考轨迹本身;无 RFC/mission/values 等其他产物
6. **无阈值配置**:门禁无重试上限(推进权在 human);config 的 HumanLoopConfig 旧字段(think_top_n/sense_max_backflows 等)v8 起无消费者


## dream 模块(internal/cmd/dream.go)

- 不生成 act-path
- 自动扫描 `.rick/jobs/*/doing/tasks.json` 发现已完成 jobs
- 与 `dream_run_*_log.md` 对比得出待处理列表
- 支持 `--background`/`-p` 背景模式（`--dangerously-skip-permissions`）

## Tools 子命令体系

`rick tools` 提供四个校验子命令，统一输出格式：

| 命令 | 验证对象 |
|------|---------|
| `plan_check` | tasks/*.md 格式 + tasks.json 存在 |
| `doing_check` | tasks.json 可解析 + debug/bug*.md 格式 |
| `learning_check` | SUMMARY.md 非空且含 `# Job` 标题 |
| `dream_check` | loops/*.md 五要素格式 |

**exit code**: 0=pass / 1=fail；输出 `✅ PASS` 或 `❌ FAIL + 描述`。

`--auto-fix` 标志才触发 Claude 修复，默认只报告（保持确定性）。

## 产品战略（human 裁决 2026-09-22）

- **Web 优先**：后续迭代全部在 rick web 页面完成（会话/学习/dream/ctrl 均走 web）；CLI 命令与 web 保持同能力，但新能力以 web 为第一入口。
- **TUI 跟随 pi 社区**：TUI 体验由 pi 社区迭代，rick 不投入主要精力。
- **自进化形态**：普通会话（easy/plan）+ `.rick/loops/rick-rsi-loop.md`（标准 Loops 目录发现）+ `rick tools {dev-web, release --merge-source, rsi_check, loops_check}`；交付期允许会话中断但必须可恢复（挂起→人工一键恢复，**绝不自动续跑**）；RSI 不是特殊会话类型，rick-dev 不是特殊工作区。

## 自进化架构（5.0.11，job_36 交付）

- **双层结构**：生产（`<prod-repo>` 工作树 + `bin/releases/<ver>/{rick,dist}` + `current` 软链 + `.last` 回滚点）与 dev（`git worktree` + 独立 HOME + 独立 pi 沙盒 + 8414 实例）完全隔离；`rick tools dev-web {init,build,up,restart,status,down}` 管理。
- **发布通道**：`rick tools release [--dry-run|--rollback|--merge-source|--detach]`——门禁（go test + npm build）→ 版本目录 → 原子换链（ln -sfn + mv -T）→ 重启 + build_id 校验 → 恢复报告。**--detach 必须**（实测事故：调用方中断导致生产停在已停未起）。
- **挂起语义**：重启后 active→suspended（非 error），人工一键恢复（POST /continue）；doing/dream 恢复时才归一化 running→pending。**绝不自动续跑**（pi 不修复悬挂 toolCall + 配额耗尽不报错）。
- **RSI 制度**：`.rick/loops/rick-rsi-loop.md`（普通 loop，标准 Loops 目录发现，无专属会话类型）；状态机 S0设计→S1隔离开发→S2门禁→S3演练→S4人类确认→S5提升→S6挂起恢复→S7 `rick tools rsi_check` 留痕（6 项机器契约，含生产 build_id 实时证伪）。
- **纪律**：job 数据只写 dev 树（main 只经 --merge-source 改动）；冲突即中止报错交 AI 修复；S5 会自挂起 web 承载的 RSI 会话。

## web 会话执行层：pi daemon worker 单轨架构（job_55，2026-11-28）

- **形态**：Go web（可重启/升级）── unix socket（`<AgentDir>/sockets/<session>.sock`）── pi-daemon-shim（node，独立进程组，持有 pi stdin 永不 EOF + 断线环形缓冲 2000 行 + stdio 落 `<stem>.log`）── pi `--mode rpc`（工具调用/子代理全程不断）。
- **单轨**（human 裁决）：旧「直连管道」架构已删除，`web_daemon_workers` 开关不存在；配置只剩 `web_daemon_socket_dir`（默认 `<AgentDir>/sockets`）与 `web_node_bin`（默认 node）。
- **生命周期**：Spawn（活 socket → `ErrDaemonAlive` 幂等拒重）；服务重启/升级 = ShutdownWorkers → DetachAll（不杀 pi，注册表保持 active，shim 缓冲断线输出）；启动对账 = ReconcileOnStart 探活 attach（active/suspended 都探——live daemon 存在时 suspended 是谎言；`get_state` 端到端验证防幻影 attach 竞态），死 → suspended + auto-resume 兜底 + CleanupOrphansExcept 孤儿收割；显式 close = kill(-pgid) 整组 + 清 `.sock/.meta.json/.log`。
- **两条硬约束**（违反即 pi 静默死亡）：① pi `--mode rpc` stdin EOF 即退出——pi 的 stdin/stdout 只能由 shim 持有；② shim/pi 的 stdio 绝不能连回 web 进程管道——web 死亡管道断裂，pi 任何 stderr 写入即 EPIPE 崩溃（dev 真 e2e 实测抓到，Go 单测抓不到——测试进程活着管道不断）。
- **重启连续性验证法**（e2e 标准动作）：创建会话 → prompt 触发长任务/子代理 → `ps` 记录 shim/pi/子代理 pid → kill web + `dev-web up` → 断言：三进程 pid 不变存活 + 会话 status=active（无 suspended）+ Busy 正确同步 + 回合跨重启自然完成 + 显式 close 整组收割。全套回归脚本见 `internal/web/daemon_e2e_test.go`。

## web 前端关键结构（job_43 六轮增量，5.0.12-5.0.13）

- **MessageList 虚拟化（react-virtuoso）**：渲染层虚拟化（≤500 envelope 只渲染可视区），数据层不动。要点：`followOutput="auto"` + `scrollToIndex(LAST)` 双通道贴底 / `firstItemIndex` 前置插入防跳 / `computeItemKey` 稳定身份锚（展开态 overrides 状态提升不丢）/ **components.Scroller 必须 `...rest` 全量转发 props**（P0 教训：丢 `data-virtuoso-scroller` → 测量回调 null 崩 → 全空白）/ 底部间距走 `components.Footer` spacer（12px，计入 footerHeight；容器 padding 不进 Virtuoso 贴底计算）——详见 virtualization_component_protocol skill
- **FileReaderProvider/Panel 全局挂载**：从 ChatView 提升到 App 壳层（App.tsx）——消息链接（LocalAwareLink→useFileReader）与顶栏工具箱共用同一阅读面板；ChatView 不再持有 Provider（Context 穿透 Virtuoso 正常）
- **星空移动端静态化（starfieldMode.ts）**：`rick.starfield.mode`（auto/animated/static，localStorage）+ 窄屏判定 + prefers-reduced-motion 统一 resolveShouldAnimate；移动端默认静态一帧（空闲 GPU 成本归零）；窄屏 dpr 封顶 1.5；窄屏动画降级走 theme.css `@media (max-width:767px)`（SMIL 在组件内条件渲染——CSS 管不到 SMIL）
- **输入框**：单发送按钮；Enter 固定=换行（enterMode 模式已删）；Ctrl/Cmd+Enter 发送、Ctrl+J 换行（隐形快捷键）；空行不发送；textarea scrollHeight 自适应（上限 240px 后内滚）
- **移动端性能结构**：流式 flush 窄屏 110ms 定时节流（宽屏 rAF 保持，stores/events.ts）；空态飞船全局静态、运行期动画窄屏关闭、ToolElapsed 窄屏 5s

## rick 分身体系（avatar，job_55 2026-11-29）

- **定位**：四 skill + avatar-control-loop = **RSI 大 loop 的分布式执行层**——主身决策 + 分身执行 + 通信控制；dev rick 是本机特殊分身的完整代码级先例（不是类比，是同构）。
- **四个 skill**：avatar_provision（分身机制：ssh 连接发现 → 二进制三路安装 → 知识资产复制 → 环境适配五项 → 全量记忆迁移）；avatar_session_replication（分身复制：单 job 三件套定向迁移 = sessions 注册表条目 + pi jsonl + job 数据，resume 带历史续答）；avatar_upgrade_debug（分身升级调试：远程 release 五步 + build_id 指纹复核 + 远程调试三件套）；avatar_communication（分身通信：**web API 即分身控制协议**——POST prompt=发消息 / steer=纠偏 / abort=中止 / GET sessions=观察，token 认证）。
- **dev rick = 特殊分身**（四要素）：隔离树（git worktree）+ 独立状态（dev HOME + 8414 实例）+ 指纹校验（health build_id ↔ dev HEAD sha7）+ 单向提升（release --merge-source 回流 main）。远程分身 = 把这四要素从「同机 worktree」泛化到「跨机 ssh」。
- **daemon 单轨是分身通信的存活地基**：pi 常驻 daemon worker 跨 web 重启存活（见「web 会话执行层」节）→ 远程控制的会话永不因分身升级/重启而断——升级与控制解耦。
- **未覆盖空白**（skill 内已标注待实战补全）：sshpass 密码安全 / 分身主动上报（推模式）/ 多分身编排拓扑 / 远程 Linux 服务管理（systemd vs launchd）。
