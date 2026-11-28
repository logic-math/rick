---
name: avatar-control-loop
trigger: "当需要把 rick 的能力延伸到另一台机器并主控它时触发——「给某台机器装一个 rick 分身」「把某 job 复制到分身继续做」「升级/调试远程分身」「向分身会话发消息控制其工作」「分布式 RSI」"
scope: "全局（任意承载会话：easy / doing / super / human-loop 均可加载）"
---

# Loop: rick 分身控制（Avatar Control）

## 定位（human 2026-11-29 裁决）

「这个 loop 就是 rick 分身控制，它是 **rick RSI 这个大 loop 的基础能力**」——分身体系 = RSI 的**分布式执行层**：

```
主身（决策）──通信控制──→ 分身（执行）──升级回流──→ 主身（合并）
   │                        │
   └── rick-rsi-loop ───────┘  dev rick = 本机特殊分身（完整代码级先例）
```

- **主身决策 + 分身执行 + 通信控制**：主身派发工作（avatar-communication 发消息/纠偏/观察），分身执行 job（工具调用/子代理在分身机跑），结果回流（git 分支合并 / dump 回传）。
- **与 rick-rsi-loop 的分工**：改 rick 自身源码并生效到生产 → rick-rsi-loop（dev 分身 = 其隔离开发环境）；把工作**分布式地**铺到多台机器执行 → 本 loop。RSI 迭代中的大计算/多机任务经本 loop 分发到分身，是 RSI 的规模放大器。
- **与 workspace-relocation-loop 的分工**：整工作区搬迁 = 本 loop S1 全量路径（dump `--sessions all` + import 四级验收）的一种场景；本 loop 更完整的增量是**单 job 定向复制**（avatar-session-replication）与**持续控制**（S3/S4）。
- **dev rick = 特殊分身**（avatar-upgrade-debug ④）：本机 worktree + 独立 HOME + 8414 实例——操作 dev 时按该 skill 的 dev 分支走，不出本 loop。

## 依赖准备（硬约束，缺失则报错停止）

| 依赖项 | 确认命令 | 要求 |
|--------|----------|------|
| 主身 rick 含 dump/import | `rick tools dump --help` | 能打印用法（main 未 release 时从 dev 树构建） |
| 分身机 ssh 可达 | `ssh -o ConnectTimeout=3 <user>@<ip> 'echo ok'` | 通（连接方式发现顺序：问 human → 查 domain → ~/.ssh/config → 扫端口，**禁止上来就扫端口**） |
| 分身机 node ≥22 | `ssh <user>@<ip> 'node -v'` | pi runtime 必需（缺失先装官方 tar.xz） |
| 分身机端口 | `ssh <user>@<ip> 'ss -lntp \| grep <port>'` | web 端口空闲（内网约定见 domain/env.md） |
| 四个分身 skill | `ls .rick/skills/ \| grep avatar` | avatar_provision / avatar_session_replication / avatar_upgrade_debug / avatar_communication 四件在位 |

**Domain 搜索**（遇任何问题先查）：`.rick/domain/env.md`（机器拓扑/凭据/代理）、`.rick/domain/pi-runtime.md`（pi 会话 slug 规则/stdio 约束）、`.rick/domain/bugs.md`。

## 目标（Goal）

一个 rick 分身从无到有（或从旧到新）并在主身控制下工作：

| # | 达标条件 | 判据（可复制执行） |
|---|---|---|
| G1 | 分身 rick web 健康 | `curl -s --noproxy '*' http://<分身IP>:<port>/api/health` → `status: ok` 且 `build_id` 已知 |
| G2 | 分身知识资产在位 | `ssh <分身> 'ls <ws>/.rick/{loops,skills,domain}'` 非空 + 关键文件抽检（如 loops/super-loop.md） |
| G3 | 复制的会话带历史续答 | 分身 web 上 resume 该会话 → 发 prompt → 回答**含迁移前上下文** |
| G4 | 通信控制闭环 | prompt 往返（发指令 → busy true→false → 产出符合指令）；steer/abort 可用 |
| G5 | 分身升级零中断 | 升级重启分身 web 期间，分身上进行中的会话存活（daemon 架构 8.1.12+）且升级后 build_id == 新版本 |

**成功标准**：G1-G4 全过（G5 仅升级场景）；自评命令见「产出评估」五断言。

## 上下文管理（Context Management）

- **保留**：分身清单（IP/port/token/版本/服务管理方式）、每个分身的当前任务（哪个 job 在分身跑、状态）、G1-G5 断言当前结论、升级版本链（build_id 时间线）
- **压缩**：每轮只留一行「对分身做了什么 → 哪条断言红了/绿了」；dump/import 与测试输出只留尾部（≤30 行）
- **遗忘**：失败的端口扫描细节（只留「连接方式=X」结论）、分身机首次准备的环境安装细节、已关闭会话的历史消息
- **敏感信息纪律**：sshpass 密码不进 git 可追踪文件（记录写「密码见 human/1Password」）；分身 token 独立于主身（avatar-provision ④）

## 可调用工具（Tool Access）

| 命令 | 用途 | 约束 |
|------|------|------|
| `rick tools dump/import` | 全量打包落位（S1 全量路径） | 归档+`.sha256` 边车成对传；大目录先 `du` 再 `--exclude` |
| `scp` / `rsync` | 传输二进制/归档/单 job 三件套 | 源码修改不 scp 散文件覆盖（走 git 分支，S4） |
| `ssh` | 分身机远程执行 | `sshpass -p '<pwd>' ssh -o StrictHostKeyChecking=no`（沙箱直连模式） |
| `curl --noproxy '*'` | 分身 web API（S3 通信/S2 验证） | 内网 IP 必带 `--noproxy`（代理劫持坑）；Bearer 用**分身** token |
| `git push/pull`（内网分支） | 分身源码修改回流（S4） | 合并冲突即中止交人工/AI 修（rick-rsi-loop 纪律） |
| `rick tools dev-web up/status` | dev 分身（本机特殊分身） | 只碰 dev 树与 dev HOME，见 avatar-upgrade-debug ④ |

**权限边界**：主身只读分身产出（日志/会话拉取）+ 显式指令（prompt/steer）；分身源码修改一律走 git 分支可审计；对分身机只写约定落位目录（`~/rick/`、`~/.rick/`、`<ws>/`）；`pkill -f` 禁用（自匹配自杀，用精确 pid）。

## 子 Agent 工作流（状态机）

每轮一个子 Agent；状态互斥推进，任一状态失败 → 修好后**重进该状态**（不跳步）：

### S0 分身清单与状态盘点

- 加载 skill：`.rick/skills/avatar_communication_skill/skill.md`（② curl 模板）
- 精确命令：
  ```bash
  # 已有分身盘点（本 loop 上下文里维护的分身清单逐个探活）：
  for AV in "<ip1>:<port1>" "<ip2>:<port2>"; do
    curl -s --noproxy '*' -m 3 "http://$AV/api/health" | python3 -c "import json,sys;d=json.load(sys.stdin);print('$AV',d.get('build_id'),d.get('status'))" 2>/dev/null || echo "$AV DOWN"
  done
  # 新目标机可达性（连接方式发现顺序见依赖准备，禁止盲扫端口）：
  sshpass -p '<pwd>' ssh -o ConnectTimeout=3 -o StrictHostKeyChecking=no <user>@<ip> 'node -v && ss -lntp | head -5'
  ```
- 产出：分身清单表（IP/port/token 来源/build_id/状态）+ 本次目标机的连接方式与缺口（缺 rick → S1 供给；已有 → S3 控制或 S4 升级）
- 出口：目标机定位明确（新装 / 复制 job / 控制 / 升级调试），进入对应 S 状态

### S1 分身供给 或 会话复制（二选一，按 S0 定位）

**S1-A 全新供给**：
- 加载 skill：`.rick/skills/avatar_provision_skill/skill.md`
- 精确命令：
  ```bash
  du -sh <ws>/*/ | sort -rh | head -15                      # 体积评估再决策排除
  rick tools dump --workspace <ws> -o /tmp/dump.tar.gz --sessions all --goos <分身平台> --exclude '*.ckpt' --exclude bigdata/
  scp /tmp/dump.tar.gz{,.sha256} <user>@<ip>:/tmp/
  ssh <user>@<ip> 'rick tools import /tmp/dump.tar.gz'      # 自动：解包+路径重映射+会话移植+marker 合并
  # 环境适配（skill ④）：分身 config.json 换新 token + pi_path/代理按分身机重写
  ssh <user>@<ip> 'openssl rand -hex 16'                    # 新 web_token
  ```
- 产出：分身机 rick web 可启动 + 知识资产 + 记忆全量到位

**S1-B 单 job 定向复制**：
- 加载 skill：`.rick/skills/avatar_session_replication_skill/skill.md`
- 精确命令：
  ```bash
  JOB=job_NN
  # 三件套定位（skill ①）：sessions.json 条目 + pi jsonl（cwd slug）+ <ws>/.rick/jobs/$JOB/
  rick tools dump --workspace <ws> -o /tmp/job.tar.gz --sessions workspaces --exclude bigdata/
  scp /tmp/job.tar.gz{,.sha256} <user>@<ip>:/tmp/ && ssh <user>@<ip> 'rick tools import /tmp/job.tar.gz'
  # 手术级（只要一个 job）：skill ② 路径 B（pi jsonl 按分身 slug 落位 + 注册表条目 + jobs 目录）
  ```
- 产出：该 job 在分身可继续（会话历史 + job 数据 + 涉及源码三件套齐）

- 出口：分身（或该 job）就绪，进入 S2 验证

### S2 分身验证（健康 + 会话续答 + 指纹）

- 加载 skill：`.rick/skills/avatar_provision_skill/skill.md`（⑥ 验证清单）+ `.rick/skills/pi_session_portability_skill/skill.md`
- 精确命令：
  ```bash
  ssh <user>@<ip> 'cd <分身ws> && rick web --port <port> --daemon'    # 或 launchd/systemd（先 ps 确认管理方式）
  curl -s --noproxy '*' http://<ip>:<port>/api/health                 # status ok + build_id 记录
  curl -s --noproxy '*' -H "Authorization: Bearer <分身token>" http://<ip>:<port>/api/sessions | python3 -m json.tool | head -30
  # 续答验证（迁移的会话）：web 上 resume → prompt「报告当前任务状态」→ 回答应含迁移前上下文
  ```
- 产出：G1（health）/G2（知识资产抽检）/G3（续答）三条断言结论
- 出口：全绿 → S3（进入控制）或直接产出评估；G3 红（resume 失忆）→ 查 slug 不匹配（skill 踩坑预案）

### S3 分身控制（发消息 / 纠偏 / 观察）

- 加载 skill：`.rick/skills/avatar_communication_skill/skill.md`
- 精确命令：
  ```bash
  AVATAR=http://<ip>:<port>; TOK=<分身token>
  # 观察：分身在干什么
  curl -s --noproxy '*' "$AVATAR/api/sessions" -H "Authorization: Bearer $TOK" \
    | python3 -c "import json,sys;[print(s['id'][:12],s.get('status'),s.get('busy'),s.get('params',{}).get('id_tag'),s.get('title','')[:30]) for s in json.load(sys.stdin)]"
  # 控制：向目标 job 会话发指令
  curl -s --noproxy '*' -X POST "$AVATAR/api/sessions/<id>/prompt" -H "Authorization: Bearer $TOK" \
    -H 'Content-Type: application/json' -d '{"message":"继续执行 gate2，失败就修复后重跑"}'
  # 等回合完成（busy 轮询见 skill ②）→ 读产出 → 下一条指令或验收
  ```
- 产出：分身在受控执行（G4：prompt 往返闭环）；每条指令一行记录（发了什么 → busy 周期 → 产出结论）
- 出口：job 在分身完成（读产出验收）或需升级/调试（→ S4）

### S4 分身升级调试（升级二进制 / 远程调试 / 修改分身源码）

- 加载 skill：`.rick/skills/avatar_upgrade_debug_skill/skill.md`
- 精确命令（按需选路）：
  ```bash
  # 升级：构建→传输→原子替换→重启→指纹复核（五步缺一不可，skill ①）
  rick tools dump -o /tmp/rick.tar.gz --goos <分身平台>      # 含新二进制
  scp rick rick.sha256 <user>@<ip>:/tmp/ && ssh <user>@<ip> 'shasum -a 256 -c /tmp/rick.sha256'
  ssh <user>@<ip> 'cp ~/rick/bin/rick ~/rick/bin/rick.bak.$(date +%s) && install -m 0755 /tmp/rick ~/rick/bin/rick && <重启命令>'
  ssh <user>@<ip> 'curl -s --noproxy "*" http://127.0.0.1:<port>/api/health'   # build_id 必须==新版本（旧二进制陷阱）
  # 调试：拉日志 + 会话状态 + 远程复现（skill ②，含环境差异前置检查）
  # 源码修改：git 分支同步（skill ③，job_55 实战：dev_rick 分支 push → 分身 pull → 重建 → 回流合并）
  ```
- 产出：G5（升级零中断：升级期间分身会话存活 + build_id == 新版本）；调试结论（根因/修复）
- 出口：升级验证绿 / 调试修复回流（合并回主身分支）→ 产出评估

### S5 产出评估（分身可用五断言）

parent 逐条执行（见下节检查表）；全绿 → 本轮结束；有红 → 回对应 S 状态重进。

## 产出评估（验证 skill + 检查表）

**调用验证 skill**：`.rick/skills/avatar_communication_skill/skill.md`（② curl 模板——五断言全部经它执行）+ `.rick/skills/avatar_upgrade_debug_skill/skill.md`（① 指纹复核——断言 V1 的 build_id 比对）。

| # | 检查项 | 验证方法 | 通过标准 |
|---|--------|----------|----------|
| V1 | 分身健康 | `curl -s --noproxy '*' http://<ip>:<port>/api/health` | `status: ok` 且 build_id 与已知版本一致（升级场景 == 新版本） |
| V2 | 知识资产 | `ssh <分身> 'ls <ws>/.rick/{loops,skills,domain} \| wc -l'` + 抽检关键文件 | 三目录非空 + 关键 loop/skill 在位（如 super-loop.md） |
| V3 | 会话续答 | 分身 web resume 迁移会话 → `POST prompt {"message":"报告当前任务状态"}` | 回答含迁移前上下文（非失忆新会话） |
| V4 | prompt 往返 | 发一条可验证指令（如「执行 gate1.py 并贴结果」）→ busy 轮询至完成 | busy true→false 一个完整周期 + 产出符合指令 |
| V5 | 重启连续性（升级场景） | 升级重启分身 web 期间 `ssh <分身> 'ps -p <pi_pid>'` + 重启后再 prompt | pi 进程同 pid 存活 + 会话状态 active + 新指令正常应答（daemon 架构 8.1.12+） |

- **全部通过** → 输出执行摘要（分身清单 + 本轮结论），等待人类决策
- **存在失败** → 失败断言 + 原因附加到上下文，回对应 S 状态启动下一轮

## 停止标准

**成功退出**：本次目标达成（新分身 G1-G4 全绿 / job 复制续跑验证闭环 / 升级 G5 绿 / 控制任务在分身完成产出验收），五断言结论已记录。

**失败退出**（任一触发 → 停止并报告人类）：
- 同一 S 状态连续 2 轮失败（如 S1 import 反复校验失败、S3 分身 API 持续 401）
- 分身机不可达且连接方式穷尽（依赖准备的发现顺序走完仍连不上）
- 需要主身侧改 rick 源码才能继续（→ 转 rick-rsi-loop，超出本 loop 写域）

**优雅退出**：人类明确要求停止；迭代达上限（默认 3 轮）。

**退出时**：parent 输出 Loop 执行摘要（分身清单变更 + 五断言终态 + 未竟事项），等待人类决策。
