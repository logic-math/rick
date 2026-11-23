---
name: super-watchdog
description: super mod 进化系统的中控巡检 skill——被定时唤醒（每 1h）加载：采集系统现状 → 对照预期状态找差距 → 按修复行为指引执行（修复动作由中控会话完成）
execution_plane: local
---

# skill:super-watchdog

## 触发场景

super 运行期间被定时唤醒巡检时加载（schedule 每小时唤醒中控，提示词指向本 skill）。

（激活词：watchdog、巡检、进化系统现状、super 健康检查）

## 巡检流程（三步）

### 第一步：采集现状（确定性脚本）

```bash
python3 <repo>/super/tools/watchdog.py <workspace>
```

脚本纯采集（无动作），输出 JSON 报告：进程存活 / heartbeat 新鲜度 / 上次会话结束原因 / runtime 模型 / API 探活（异常时）/ 停机时长统计。**脚本退出码 0=健康，1=需处理，2=脚本自身故障。**

### 第二步：对照预期状态，找差距

| 检查项 | 预期状态 | 差距即问题 |
|---|---|---|
| `checks.process` | `ok=true`、`state=running`、`alive=true` | 进程死了 |
| `checks.heartbeat.age_s` | < 300 秒 | 停滞=僵死（进程活但不动） |
| `checks.last_session` | 仅作上下文（进程已活时旧记录无当下意义） | —— |
| `runtime_model` | `(pi default)` 或已配置且探活通过 | 配置了却探活失败=需切换 |
| `downtime.uptime_ratio` | 趋势上升（每小时巡检对比） | 下降=又停机了 |
| 实验产出 | `admitted` 数每小时有增长 或 `running` 实验在正常推进（wall < 2h） | 长时间零增长+零 running=空转 |

### 第三步：按差距执行修复（修复动作由中控完成）

**修复行为指引（playbook）**——按差距对号入座：

#### 差距 ①：进程死（process.ok=false）

排查：`tail -30 <workspace>/super.log` 看死因（崩溃重启超限 / infra_failure / budget 耗尽）。
- API 正常的普通崩溃 → 执行 `python3 super/tools/watchdog.py <workspace> --restart`
- `budget_exhausted` → **不重启**（预算烧完是设计行为），读 events 表对账后上报人类
- `infra_failure`（API 故障）→ 走差距 ③

#### 差距 ②：heartbeat 停滞（进程活但 age_s ≥ 300）

排查：`tail -50 super.log` + `sqlite3 <workspace>/super.db "SELECT event_type, ts FROM events ORDER BY seq DESC LIMIT 10"`——判断是 DB 锁、僵死还是大实验占住。
- 僵死确认（日志无进展 ≥ 10 分钟）→ `--restart`
- 大实验正常推进（有 agent_started/ended 持续）→ 不动作，下轮再看

#### 差距 ③：API 死（探活 ok=false / 0-token 空回 / 429 / 配额报错）

排查：逐个探活备选模型（与中控当前模型**异源优先**）：
```bash
# 备选链参考：deepseek-v4-pro → kimi-k3 → glm-5.2（1M 上下文优先）
python3 -c "import subprocess; r=subprocess.run(['pi','-p','--no-session','--model','<MODEL>','reply with the single word: pong'],capture_output=True,text=True,timeout=120); print(r.returncode, r.stdout[:80])"
```
- 找到活的 → 执行 `python3 super/tools/watchdog.py <workspace> --switch-to <MODEL>`
- 429 限流类（「访问量过大」）→ **不切换**，等下一轮巡检再探（限流通常自愈）
- 全链死 → 上报人类（供应商级瘫痪，附全部探活证据）

#### 差距 ④：GPU 链路断（实验报 `stage=import FileNotFoundError` 特征）

排查（**必须走框架真实路径**：源码沙箱 10.128.229.113:8420 → sshpass 密码通道 → H20_POOL 四台 `33.229.1.97 / 33.229.1.85 / 33.229.29.190 / 33.229.60.179`；**不要用 debug 记录里的换机前旧 IP**）：
```bash
ssh -p 8420 hadoop-recsys@10.128.229.113 'for ip in 33.229.1.97 33.229.1.85 33.229.29.190 33.229.60.179; do sshpass -p "MiMi@@19950810" ssh -o ConnectTimeout=6 -o StrictHostKeyChecking=no sunquan20@$ip "nvidia-smi --query-gpu=memory.used,utilization.gpu --format=csv,noheader"; done'
```
- 四台全活 → 是间歇性问题，观察即可（框架 remote_exec 有轮询）
- 部分死 → 记录死机 IP，上报人类（运维级）
- 沙箱不通 → 上报人类

#### 差距 ⑤：uptime_ratio 持续低 / 停机频发

排查：`sqlite3 <workspace>/super.db "SELECT ts, event_type FROM events WHERE event_type IN ('session_started','session_ended') ORDER BY seq"` 看停机窗口分布。
- 若停机都是「无人重启空转」→ 确认本巡检链工作正常（每 1h 应该兜住）
- 若停机是反复崩溃 → 上报人类（附崩溃模式统计）

## 修复约束（红线）

1. **中控不直接动 `env/gates/`**（env 隔离纪律——builder/coding/debug 禁读区，验证器问题走提案队列）
2. **切换模型前必须实测探活备选**（禁止盲切）；切换动作只能改 `super.json` 的 `agent_models` + 重启，**不碰其他配置**
3. **限流（429）≠ 死**——限流等自愈，不切换（防过度反应）
4. **不修改已 admitted 的精英数据**（实验记录/elite 状态只读）
5. **每次修复动作写 DB 事件留痕**（可审计）：
```bash
sqlite3 <workspace>/super.db "INSERT INTO events (event_type, payload, actor, ts) VALUES ('watchdog_action', '{\"action\": \"<做了什么>\", \"reason\": \"<为什么>\"}', 'watchdog', datetime('now', 'localtime'));"
```
6. **budget_exhausted 不重启**（预算对账后上报人类裁决）
7. 连续 2 轮巡检同样的修复动作无效 → 升级上报人类，不盲目重试

## 巡检回执格式（回写给中控会话的输出）

- 健康：`[watchdog HH:MM] healthy · uptime xx% · admitted N · running M`（一行，不打扰）
- 异常：`[watchdog HH:MM] 需处理：<差距类型> → <排查结论> → <执行动作/上报人类>`

## 与其他组件的关系

- 采集脚本：`super/tools/watchdog.py`（默认=采集报告；`--restart` / `--switch-to` = 修复动作手柄）
- 质量评估：`skill:super-eval`（ESQ 报告，uptime_ratio 已纳入 D4）
- 定时唤醒：pi schedule `super-watchdog`（每 1h 向中控投递巡检提示词）
