# skill:avatar-session-replication（分身复制——把某个 job/会话定向搬到另一台机器续跑）

## 触发场景

当需要把 rick 的**某个具体 job**（human-loop / easy / super 等任一会话类型）连同其完整工作状态复制到分身机器上继续工作时使用。信号词：

- 「把这个 job 搬到那台机器继续」「在分身上继续这个会话」「把 env_3 的工作迁移过去」
- 「A 机上跑了一半的 job 让 B 机接着做」
- 主身机器要腾空/维护，某工作必须异地续跑

与相邻 skill 的分界：avatar-provision = 装分身（全量）；本 skill = **单 job 定向复制**（三件套：pi 会话 + 注册表条目 + 工作区文件）；cross-machine-migration = 整工作区搬迁（8.6GB 级全量）。

前置：目标机已有 rick 分身（avatar-provision 完成）；源机 rick 含 `tools dump/import`。

## 预期效果

- 分身机上该 job 的会话可 resume 并**带完整历史续答**（对话上下文不丢）
- job 数据（.rick/jobs/<job>/：plan/doing/tasks.json/grilling 等）在分身可读、可继续门禁流转
- 工作区相关源码文件随 job 一起到位——分身继续改代码、跑测试不断链
- 单 job 粒度：不搬无关工作区（体积可控，分钟级）

## 核心内容

### ① 定位三件套（源机）

```bash
JOB=job_55
# 1) web 会话注册表条目（含 pi_session_id / workspace_id / type / params / title）
~/.rick/sessions.json（或 <state-dir>/sessions.json）——按 job 或 title 检索：
python3 -c "import json;ss=json.load(open('$HOME/.rick/sessions.json'));[print(s['id'],s.get('pi_session_id'),s.get('type'),s.get('params',{}).get('id_tag')) for s in (ss if isinstance(ss,list) else ss.get('sessions',[]))]"
# 2) pi 会话历史 jsonl（按 pi_session_id 找文件）
grep -rl "<pi_session_id>" ~/.rick/pi/agent/sessions/*/ | head -1
#    文件名 = <ISO时间戳>_<pi_session_id>.jsonl，目录 = cwd slug（见 pi_session_portability）
# 3) 工作区文件：workspace 路径 + .rick/jobs/<job>/ 全目录
```

### ② 打包传输（dump 定向 or 手工三件套）

**路径 A（推荐）：dump 定向子集**

```bash
# 源机：限定工作区 + 会话策略（只打包该工作区的 pi 会话）
rick tools dump --workspace <ws> -o /tmp/job.tar.gz --sessions workspaces \
  --exclude bigdata/ --exclude '*.ckpt'
scp /tmp/job.tar.gz{,.sha256} <user>@<avatar>:/tmp/
# 分身机：
rick tools import /tmp/job.tar.gz    # 自动路径重映射 + 会话移植 + 注册表合并
```

**路径 B（手术级）：手工三件套**（dump 不可用/只要一个 job 时）

```bash
# 1) pi 会话：按分身机 workspace 的 cwd slug 落位
#    分身 slug = '--' + <分身workspace路径> 去首/ 且 /\: 替换 - + '--'（下划线不动）
scp <源pi目录>/<ts>_<pi_session_id>.jsonl <user>@<avatar>:~/.rick/pi/agent/sessions/<分身slug>/
# 2) 注册表条目：源机导出该 job 的条目 → 分身合并进 sessions.json（workspace_id 若变需改写）
# 3) job 数据 + 源码：
rsync -av <ws>/.rick/jobs/$JOB/ <user>@<avatar>:<分身ws>/.rick/jobs/$JOB/
rsync -av --exclude='.git' <涉及源码子树>/ <user>@<avatar>:<分身ws>/<子树>/
```

### ③ 分身侧落位校验

```bash
# 分身机：三件套对齐自检
ls ~/.rick/pi/agent/sessions/<分身slug>/ | grep <pi_session_id>     # pi 会话在位
python3 -c "…读 sessions.json 确认条目存在且 workspace_id 指向分身工作区…"
ls <分身ws>/.rick/jobs/$JOB/doing/tasks.json                          # job 数据在位
```

### ④ 续跑验证

```bash
# CLI 直接 resume（验证历史加载）：
cd <分身ws> && pi --session <pi_session_id> -p "报告当前任务状态"     # 应带上下文回答
# 或经 web：分身 web 上 resume 该会话（POST /api/sessions/<id>/continue）→ 发 prompt → 观察续答
# job 门禁流转：分身跑 gate（python3 .rick/jobs/$JOB/plan/gates/gate*.py）
```

## 踩坑预案

- **slug 不匹配 = resume 静默失忆**：pi 按 cwd slug 找会话——分身机 workspace 路径与 slug 必须一致；不一致时改 slug 目录名（最小改写）或改分身 workspace 路径（见 pi_session_portability）
- **注册表 workspace_id 断链**：分身工作区 marker id 与条目不一致 → 会话列表不显示——import 自动处理；手工路径必须同步改写条目的 workspace_id
- **后台型 job（doing/dream）**：goroutine 随源机进程消亡——复制的是**任务状态**（tasks.json running→pending 归一化后由分身重跑剩余 task），不是进程本身（见 domain：recovery 语义）
- **体积失控**：job 相关大文件（数据集/权重）别打包——--exclude 或路径 B 只拿源码+jsonl

## 待实战补全（如实标注）

- **会话注册表的正式导出/导入工具**：当前手工合并 sessions.json 条目（import 处理全量场景）
- **增量同步**：job 复制后在源机继续产生的增量再同步（当前=一次性快照）
- **super 调度器后台引擎的迁移**：superopt py 进程状态/池的跨机接管（当前只复制中控会话）
