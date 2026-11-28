# skill:avatar-communication（分身通信——向分身的 job 会话发消息、控制分身工作行为）

## 触发场景

当需要**向分身上正在工作的某个 job 会话发送消息**（控制其工作行为）、**中途纠偏**、**中止**、或**观察分身全部工作状态**时使用。信号词：

- 「让分身继续做 X」「给分身的那个会话发个消息」「远程控制分身」
- 「分身跑偏了，纠偏一下」「停掉分身手头的活」
- 「看看分身现在在干什么」

前置：分身 rick web 已运行（avatar-provision）；拿到分身的 web_token（分身 config.json，经 avatar-provision 换新——不是主身 token）；网络可达分身 IP:port（LAN/内网直连）。

## 预期效果

- 消息送达分身目标会话（prompt → agent 立即执行该指令）
- 分身工作行为可控：prompt（新指令）/ steer（运行中纠偏，不中断回合）/ abort（中止当前回合）
- 分身状态可观察：全部会话列表（状态/忙碌）、单会话消息历史、模型列表
- **通信不丢会话**：daemon 架构（rick 8.1.12+）下分身 web 重启/升级，会话持续存活——主身随时重连继续控制，消息不因分身升级而丢失

## 核心内容

### ① 通信协议 = 分身的 web API（token 认证）

| 意图 | 调用 | 语义 |
|---|---|---|
| 发消息（控制工作） | `POST /api/sessions/{id}/prompt` body `{"message":"…"}` | 向该 job 会话发指令，agent 排队执行 |
| 运行中纠偏 | `POST /api/sessions/{id}/steer` body `{"message":"…"}` | agent 回合**不中断**，消息插入当前回合 |
| 中止当前回合 | `POST /api/sessions/{id}/abort` | 停掉当前生成（会话保留，状态 active） |
| 观察全部工作 | `GET /api/sessions` | 全部会话：status/busy/type/title |
| 单会话状态 | `GET /api/sessions/{id}` | 含 busy（是否正在生成） |
| 读历史消息 | `GET /api/sessions/{id}/entries` | pi 会话历史 entries |
| 关闭会话 | `POST /api/sessions/{id}/close` | 结束该 job（分身侧整组收割） |

### ② curl 模板（跨机三要素：--noproxy、Bearer、LAN IP 直连）

```bash
AVATAR=http://<分身IP>:<port>   # 内网直连 IP；经内网代理时必须 --noproxy
TOK=<分身 web_token>

# 观察：分身现在在干什么
curl -s --noproxy '*' "$AVATAR/api/sessions" -H "Authorization: Bearer $TOK" \
  | python3 -c "import json,sys;[print(s['id'][:12],s.get('status'),s.get('busy'),s.get('type'),s.get('title','')[:30]) for s in json.load(sys.stdin)]"

# 控制：向某个 job 会话发指令（agent 会按消息内容执行——发「执行 gate2 并修复失败项」它就去做）
curl -s --noproxy '*' -X POST "$AVATAR/api/sessions/<id>/prompt" \
  -H "Authorization: Bearer $TOK" -H 'Content-Type: application/json' \
  -d '{"message":"继续执行 gate3，失败就修复后重跑"}'

# 纠偏：agent 正在跑，插入新要求
curl -s --noproxy '*' -X POST "$AVATAR/api/sessions/<id>/steer" \
  -H "Authorization: Bearer $TOK" -H 'Content-Type: application/json' \
  -d '{"message":"注意：不要改 tests/ 之外的文件"}'

# 轮询等待回合完成（busy 转 False/None）
for i in $(seq 1 60); do
  B=$(curl -s --noproxy '*' "$AVATAR/api/sessions/<id>" -H "Authorization: Bearer $TOK" \
    | python3 -c "import json,sys;print(json.load(sys.stdin).get('busy'))")
  [ "$B" = "False" -o "$B" = "None" ] && break; sleep 15
done
```

### ③ 会话寻址：web session id ↔ pi session id ↔ job 的映射

```text
GET /api/sessions 的每条 = { id（web 会话 id，寻址用）, pi_session_id, type,
  title, params: { id_tag: "job_N" | "loop_N" } }
映射链：job（人视角）→ params.id_tag → web id（API 寻址）→ pi_session_id（pi 历史文件）
按 job 找会话：title 或 params.id_tag 过滤；按历史续答验证：pi_session_id 查 jsonl。
```

### ④ daemon 架构下的通信保障（rick 8.1.12+）

分身 web 重启/升级时：pi 会话进程持续存活（socket detach→attach），分身会话**永不因升级掉线**——主身对分身的控制流只需：分身 web 回来（health ok）→ 会话还在原 id、状态 active → 继续 prompt。分身重启窗口（秒级）内 curl 失败属预期，重试即可。

## 踩坑预案

- **内网代理劫持**：内网机 `http_proxy` 三件套常在——对分身 IP 必须 `--noproxy '*'`（实测：不加时 curl 走代理连内网 IP 直接失败/挂起）
- **token 用错身**：主身 token ≠ 分身 token（分身 token 在分身机 config.json，avatar-provision ④ 换新）——401 unauthorized 时先查是不是拿错 token
- **prompt vs steer 语义**：prompt 在 agent 空闲时排队执行、运行中也会接收（按实现可能是排队）；steer 专用于**运行中**插入——选错可能等一个回合才生效
- **busy 判定**：`busy: true` = agent 正在生成；`None`/`false` = 空闲等输入——轮询退出条件两者都要判（API 序列化差异）
- **关闭 vs 中止**：abort 只停当前回合（会话还能 prompt）；close 整组收割 pi 进程（分身上该 job 终结）——不可逆操作先确认

## 待实战补全（如实标注）

- **分身→主身反向上报（推模式）**：分身主动汇报进度/完成事件（当前只有主身拉取 GET；可考虑 SSE 长连接跨机消费分身事件流——未实战）
- **多分身消息路由**：同一指令广播/编排在多个分身间的分发（当前实战=单分身定向）
- **消息可靠性语义**：prompt 送达确认/失败重试的正式协议（当前靠 HTTP 状态码 + 轮询 busy 观察）
