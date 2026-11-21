---
name: stat-super
description: super 进化系统的进展查询——按固定查询模板拉全貌（运行状态/实验战况/adjust 流水线/格覆盖/资源异常/报告指标），human 问「进展如何」时使用
execution_plane: local
---

# skill:stat-super

## 触发场景

human 问 super 进化系统的进展（「进展如何」「super 状态」「刷榜到哪了」「跑得怎么样」）——按本 skill 的查询模板拉全貌，不临时拼 SQL。

（激活词：进展、状态、战况、刷榜进展、super status、跑得如何）

## 查询模板（按序执行，WS=工作区根）

### 1. 运行总览（4 条命令）

```bash
STATUS="$WS/super.db"
# ① 进程与心跳
cat $WS/heartbeat.json 2>/dev/null | python3 -m json.tool | head -6
# ② 实验战况（每实验一行：题/状态/分数）
sqlite3 $STATUS "SELECT experiment_id, cell_id, status, score FROM experiments ORDER BY iteration;"
# ③ 事件分布（agent/gate/adjust/失败的关键计数）
sqlite3 $STATUS "SELECT event_type, count(*) FROM events WHERE event_type NOT IN ('heartbeat') GROUP BY event_type ORDER BY count(*) DESC LIMIT 12;"
# ④ token 消耗
sqlite3 $STATUS "SELECT sum(json_extract(payload,'$.total_tokens')) FROM events WHERE event_type='agent_ended';"
```

### 2. 失败归因（有 failed 实验时必查）

```bash
# 失败实验的 gate_result 详情（failure_stage 定性：correctness/bench/remote_link/soft）
sqlite3 $STATUS "SELECT experiment_id, json_extract(payload,'$.failure_stage'), substr(json_extract(payload,'$.diagnostics'),1,120) FROM events WHERE event_type='gate_result' AND json_extract(payload,'$.passed')=0 ORDER BY seq;"
# debug 的诊断结论（误杀 vs 真实失败的裁决证据）
for d in $(sqlite3 $STATUS "SELECT workdir FROM experiments WHERE status='failed';"); do
  python3 -c "import json; d=json.load(open('$d/agent_debug.json')); print(str(d.get('summary'))[:200])" 2>/dev/null
done
```

### 3. adjust 流水线健康

```bash
# 每轮 adjust 的步骤完整性与 session
sqlite3 $STATUS "SELECT adjust_id, iteration, json_extract(sessions,'$.step1') IS NOT NULL, length(sessions) FROM adjusts;"
# 步骤事件序列（step3 应有 3 个并行）
sqlite3 $STATUS "SELECT iteration, group_concat(step) FROM events WHERE event_type='adjust_step_started' GROUP BY iteration;"
# 格动作（权重变化+理由）
sqlite3 $STATUS "SELECT json_extract(payload,'$.cell_id'), json_extract(payload,'$.old_weight'), '->', json_extract(payload,'$.new_weight') FROM events WHERE event_type='weight_changed' ORDER BY seq;"
```

### 4. 报告指标（题库刷榜模式）

```bash
# Coverage / 最优分布 / fast_1.0（提速 2× 的题数）
sqlite3 $STATUS "SELECT count(*) || '/' || (SELECT count(*) FROM cells) FROM cells WHERE best_experiment_id IS NOT NULL;"
sqlite3 $STATUS "SELECT cell_id, score FROM experiments WHERE status='admitted' ORDER BY score DESC LIMIT 5;"
sqlite3 $STATUS "SELECT count(*) FROM experiments WHERE status='admitted' AND score > 1.0;"
```

### 5. 异常信号（负向检查——出现即报）

```bash
# 远程链路失败计数（infra 问题，非 candidate 缺陷）
sqlite3 $STATUS "SELECT count(*) FROM events WHERE event_type='gate_result' AND json_extract(payload,'$.diagnostics') LIKE '%remote%link%';"
# 幽灵实验（目录与表行不齐）/ 槽位泄漏
ls $WS/codes/ | wc -l; sqlite3 $STATUS "SELECT count(*) FROM experiments;"
sqlite3 $STATUS "SELECT count(*) FROM resources WHERE holder_experiment_id IS NOT NULL;"
# GPU 资源（远程执行模式）
ssh -p 8420 hadoop-recsys@10.128.229.113 'sshpass -p "MiMi@@19950810" ssh -o StrictHostKeyChecking=no sunquan20@33.29.8.59 "nvidia-smi --query-gpu=index,memory.used,memory.total,utilization.gpu --format=csv,noheader"' 2>/dev/null
```

## 解读框架

| 维度 | 健康 | 异常（报 human） |
|---|---|---|
| 实验 | admitted 比例 > 50%；failed 有 debug 归因 | failed 全是 remote_link/OOM（infra 误杀——修复后再跑，别浪费迭代） |
| adjust | 每 interval 一行 adjusts；7 步 session 齐 | 步骤断裂/未触发（interval 配置或饥饿回归） |
| 分数 | 最优分上升/Coverage 推进 | 全 failed 或分数停滞且无 weight_changed |
| 资源 | GPU 显存余量 > 题目需求 | 显存被外部进程占满（换机/清理——继续跑=白烧 token） |

## 汇报格式（对 human）

```
## 刷榜进展（第 N 轮 · 运行 X 分钟）
战况表（实验/题/状态/分数）+ 失败归因表 + adjust 动作摘要 + 报告指标 + 异常信号
+ 下一步判断（继续/修复/等资源）
```
