---
name: super-watchdog
description: super mod 进化系统的**基础设施巡检 subloop**——被定时唤醒（每 1h）交接给 sub agent 执行：修复 py runtime 无法自主修复的基础设施问题（API 故障/进程死亡/GPU 链路断/停机空转），确保系统长期稳定运行；主 agent 只收一行回执
execution_plane: local
---

# skill:super-watchdog（独立 subloop）

## 职责边界（用户裁决 v4）

**只关心基础设施**——修复 py runtime（调度器）无法自主修复的问题，确保整个系统长期稳定运行：

| 归本 skill 管（runtime 修不了） | 不归本 skill 管（别越界） |
|---|---|
| API 供应商故障（切模型） | 进化质量（→ skill:super-eval） |
| 调度器进程死（restart） | 实验失败（→ debug agent） |
| 停机空转（无人重启的时段） | 格权重/采样（→ adjust 7 步） |
| GPU 链路断（远程机器级故障） | 验证器协议（→ 提案队列） |
| 预算耗尽确认（上报人类） | env 构建质量（→ skill:gen-env） |

## 执行方式（subloop 交接——防上下文污染）

本 skill 是**独立 subloop**：主 agent 被唤醒后，把本 skill 整包交给一个 sub agent 执行（fresh context），sub agent 完整走完巡检流程并返回回执——**主 agent 上下文零污染，只保留回执一行**。

主 agent 交接模板：

```text
subagent({ workflowScript: "return runs.run('watchdog', { agent: 'worker', task: '读 /Users/logic/workdir/AI_CODING/rick/.rick/skills/super-watchdog_skill/skill.md 并严格按该 skill 完整执行本轮巡检（采集→对照→修复/回执）。skill 自包含全部流程与约束。' })" })
```

## 巡检流程（sub agent 三步）

### 第一步：采集现状（确定性脚本）

```bash
python3 /Users/logic/workdir/AI_CODING/rick/super/tools/watchdog.py /Users/logic/workdir/MT_INFER/.rick/super/kernelbench-dev
```

脚本纯采集（无动作），JSON 报告：进程 / heartbeat / session 历史 / runtime 模型 / API 探活（异常时）/ 停机统计。退出码 0=健康，1=需处理，2=脚本故障。

### 第二步：对照预期状态，找差距

| 检查项 | 预期状态 | 差距即问题 |
|---|---|---|
| `checks.process` | `ok=true`、`alive=true` | 进程死了 |
| `checks.heartbeat.age_s` | < 300 秒 | 停滞=僵死 |
| `runtime_model` | 默认或已配置且探活通过 | 配置了却探活失败 |
| `downtime.uptime_ratio` | 趋势上升（逐小时对比） | 下降=又停机 |
| 实验产出 | admitted 有增长 或 running 推进（wall < 2h） | 长期零增长+零 running=空转 |

### 第三步：按差距执行修复（修复动作由 sub agent 完成）

#### 差距 ①：进程死

排查 `tail -30 <workspace>/super.log`：
- 普通崩溃 → `--restart`（续跑章法见下）
- `budget_exhausted` → 按续跑章法重启（裁决后）：剩余格数匹配预算（不是「不重启」——预算墙是人为残留，见 iter_67 判例：3h 残留墙杀 + adjust 提案 3→6 未落地）
- `infra_failure` → 走差距 ③

**续跑章法（重启前必做——防「瞎重启」）**：
1. **config 体检**：`restart` 手柄自带（历史残留 wall/迭代帽上报）
2. **预算匹配目标**：剩余格数 × 0.7h ÷ 8 并发 = 纯执行时长 → 墙预算取其 1.3 倍以上（或恢复默认 168h 由人控兼底）
3. **迭代帽匹配**：cursor + 剩余格数 × 1.5 实验/格 > 当前帽 → 提帽
4. 带旗标重启：`python3 super/tools/watchdog.py <workspace> --restart --max-wall-time <H> --max-iterations <N>`

#### 差距 ②：heartbeat 停滞（进程活）

排查 `tail -50 super.log` + 最近事件：
- 僵死（日志 ≥10 分钟无进展）→ `--restart`
- 大实验正常推进 → 不动作

#### 差距 ③：API 死（探活失败 / 0-token 空回 / 配额报错）

逐个探活备选模型（与中控当前模型**异源优先**：`deepseek-v4-pro` → `kimi-k3` → `glm-5.2`）：
- 活的 → `python3 super/tools/watchdog.py <workspace> --switch-to <MODEL>`
- **429 限流（「访问量过大」）→ 不切换**，等下轮自愈
- 全链死 → 上报人类（附全部探活证据）

#### 差距 ④：GPU 链路断（`stage=import FileNotFoundError` 特征）

**必须走框架真实路径**（源码沙箱 10.128.229.113:8420 → sshpass 密码通道 → H20_POOL 四台 `33.229.1.97 / 33.229.1.85 / 33.229.29.190 / 33.229.60.179`；**禁用 debug 记录里的换机前旧 IP**）：
- 四台全活 → 间歇性问题，观察即可
- 部分死 / 沙箱不通 → 记录并上报人类（运维级）

#### 差距 ⑤：停机频发 / uptime 持续低

看停机窗口分布（`SELECT ts, event_type FROM events WHERE event_type IN ('session_started','session_ended') ORDER BY seq`）：
- 都是空转停机 → 确认巡检链正常（每 1h 兜底）
- 反复崩溃 → 上报人类（附崩溃模式统计）

## 修复约束（红线）

1. **不动 `env/gates/`**（验证器问题走提案队列）
2. **切换模型前必须实测探活**（禁盲切）；只改 `super.json` 的 `agent_models` + 重启，不碰其他配置
3. **限流（429）≠ 死**——等自愈不切换
4. **已 admitted 精英数据只读**
5. **每次修复动作写 DB 事件留痕**：
   `sqlite3 <workspace>/super.db "INSERT INTO events (event_type, payload, actor, ts) VALUES ('watchdog_action', '{\"action\":\"...\",\"reason\":\"...\"}', 'watchdog', datetime('now','localtime'));"`
6. **budget_exhausted 重启必带章法**（预算与目标匹配——历史残留 3h 墙杀 iter_67 判例：系统提案 3→6 未落地 + 人重启未体检 = 双重失误）；无剩余工作量估算时上报人类裁决
7. **连续 2 轮同样修复无效 → 升级上报人类**

## 回执格式（sub agent 的最终输出——主 agent 只保留这个）

- 健康：`[watchdog HH:MM] healthy · uptime xx% · admitted N · running M`
- 异常：`[watchdog HH:MM] <差距类型> → <排查结论> → <动作/上报人类>`

## 相关组件

- 采集脚本+动作手柄：`super/tools/watchdog.py`（默认采集 / `--restart` / `--switch-to`）
- 定时唤醒：pi schedule `super-watchdog`（每 1h 提示词唤醒中控，中控交接本 subloop）
- 质量评估（不归本 skill）：`skill:super-eval`
