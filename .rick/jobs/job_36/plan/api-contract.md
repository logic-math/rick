# rick web API 契约（v1）——后端与前端共同遵守的单源

> 本文件是 task6/task11/task13（后端实现）与 task7/task9/task10/task12（前端消费）的共同契约。
> 改动契约必须先改本文件（实现 task 不得单方面偏离；E2E gate 按本文件校验）。

## 通用

- Base URL：`http://<host>:<port>`（默认 127.0.0.1:6137）
- 鉴权：除静态资源（`/`、`/assets/*`、manifest 等）外全部接口需 token：
  - Header `Authorization: Bearer <token>`（REST）
  - Query `?token=<token>`（SSE——EventSource 无法带 header）
  - 未授权 → `401 {"error":{"code":"unauthorized","message":"..."}}`
- 错误统一：`{"error":{"code":"<snake_case>","message":"<人读>"}}`；400 参数/404 不存在/409 状态冲突/500 内部
- 所有时间戳 RFC3339 带时区

## Server

```
GET /api/config → 200 {"version": 1, "rick_version": "3.1.5", "port": 6137, "auth_required": true}
```

## Workspaces（工作区注册表，机器级 ~/.rick/web.json）

```
GET    /api/workspaces → 200 [{"id":"a1b2c3d4","path":"/abs/path","name":"rick","added_at":"...","jobs_count":12}]
POST   /api/workspaces {path:"/abs/path", name?} → 201 {"id","path","name","added_at"}
       - id = sha1(path) 前 8 位；path 必须存在且含 .rick/ 目录（否则 400 invalid_workspace）
       - 幂等：已注册同 path → 200 返回既有条目
DELETE /api/workspaces/{id} → 204（不影响磁盘；活跃会话保留）
```

## Sessions（会话）

```
GET  /api/sessions?workspace=<ws_id> → 200 [SessionInfo]
POST /api/sessions → 201 SessionInfo
    body: {"workspace_id":"...","type":"plan|easy|ctrl|human-loop|learning|dream|doing","params":{...},"title"?}
    params 按类型：
      plan:       {"requirement":"...","job"?:"job_N"(复用已有 plan 目录)}
      easy:       {"requirement":"...","ctx_path"?: "..."}
      ctrl:       {"job":"job_N"}
      human-loop: {"topic":"..."}
      learning:   {"job":"job_N"}
      dream:      {"job_num":5,"mode":"interactive|background"}
      doing:      {"job":"job_N"}
    交互型（plan/easy/ctrl/human-loop/learning/dream[interactive]）：spawn pi rpc worker + bootstrap prompt
    后台型（doing/dream[background]）：goroutine 执行 + 事件流监控；状态流转 pending→running→closed|error
GET    /api/sessions/{id} → 200 SessionInfo
POST   /api/sessions/{id}/prompt {message} → 202（非 active 或 streaming 拒 steer 场景 → 409）
POST   /api/sessions/{id}/steer {message} → 202
POST   /api/sessions/{id}/abort → 202
POST   /api/sessions/{id}/close → 202（abort 进行中的 + kill worker；后台型 cancel context）
POST   /api/sessions/{id}/resume → 202（closed→active：spawn + `--session <pi_session_id>`）
POST   /api/sessions/{id}/ui_response {request_id, value?|confirmed?|cancelled:true} → 202
GET    /api/sessions/{id}/entries?since=<entryId> → 200 {"entries":[...],"leaf_id":"..."|null}
       （active=rpc get_entries 透传；closed=离线读 session JSONL）

SessionInfo: {"id":"uuid","workspace_id","type","params":{...},"title"?,
              "status":"active|running|closed|error","pi_session_id":"uuid",
              "created_at":"...","closed_at"?: "..."}
```

## Jobs（工作区 .rick/jobs 只读）

```
GET /api/workspaces/{ws}/jobs → 200 [{"job_id":"job_5","updated_at":"...",
     "tasks":[{"task_id":"task1","name":"...","status":"success","commit_hash":"..."}]}]
GET /api/workspaces/{ws}/jobs/{job}/tasks → 200 tasks.json 原文（JSON）
GET /api/workspaces/{ws}/jobs/{job}/file?path=plan/task1.md → 200 {"path","content"}
    白名单：job 目录下 plan/**, doing/**（含 debug/, act-path.md, raw_session_coding.log）；
    路径逃逸（..、绝对路径）→ 400
```

## Knowledge（工作区 .rick/{domain,loops,skills} 只读）

```
GET /api/workspaces/{ws}/knowledge/tree → 200 {"tree":[{"path":"domain/bugs.md","size":1234},...]}
GET /api/workspaces/{ws}/knowledge/file?path=domain/bugs.md → 200 {"path","content"}
    根白名单：domain/ loops/ skills/（.md/.json/.txt/.yaml/.py）；路径逃逸 → 400
```

## SSE 事件流（单流多路复用）

```
GET /api/events?token=xxx → text/event-stream
  - 每事件：id: <seq>（全局单调，重启归零）\n data: <envelope> \n\n
  - envelope: {"seq":N,"type":"...","session_id":"..."|null,"data":{...}}
  - type:
      server_info     连接建立即发（{"version","rick_version","time"}）
      session_event   pi rpc 事件透传：{"event":<pi 原始 rpc 事件对象>}（message_start/update/end、
                      tool_execution_start/end、agent_start/end/settled、turn_start/end、
                      extension_ui_request 等）
      session_state   会话状态变更：{"status":"...","reason"?}
      jobs_update     tasks.json 变更：{"job_id","diff":[{task_id,from,to}],"snapshot":[...]}
      frontend_reload 覆盖层 dist 变更：{}（前端收到后 location.reload()）
  - 心跳：每 15s 注释行 `: ping`
  - 断线重连：客户端带 Last-Event-ID 头 → 服务端重放缓冲中 seq 之后的未确认事件（缓冲上限 1000，
    超限则发 server_info 重建全量状态）
```

## 静态资源

```
GET / → index.html（覆盖层 ~/.rick/web/dist 优先；否则 embed baseline；均 no-cache）
GET /assets/* → hashed 静态资源（长缓存）
SPA fallback：未知路径 → index.html（前端路由）
```

## 补充端点（health / web 管理）

```
GET  /api/health → 200 {"status":"ok"}（无鉴权——探活专用）
POST /api/web/customize → 200 {"ok":true,"scaffolded":bool}（scaffolded=false 表示已存在跳过）
POST /api/web/reset    → 200 {"ok":true}（删 ~/.rick/web/{src,dist}，保留 web.json/sessions.json）
```
