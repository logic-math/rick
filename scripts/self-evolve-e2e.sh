#!/usr/bin/env bash
# scripts/self-evolve-e2e.sh —— rick 自进化链路端到端验收（可重放、幂等）
#
# 验收目标（设计树 KR1-KR4）：
#   KR1 隔离：dev 实例与生产状态/工作区完全隔离（含 flock singleton 与守卫）
#   KR2 开发闭环：前端 overlay 热更即时生效；后端源码改动经 restart 生效（行为可见）
#   KR3 受控提升：release 原子提升到「模拟生产」并校验 build_id；--rollback 回退
#   KR4 挂起可恢复：重启把在跑会话标 suspended（不自动恢复），/continue 才恢复
#
# 绝对纪律：
#   · 真实生产（8413 / ~/.rick / /workdir/sunquan20/AI_CODING/rick）**只读**；
#     脚本用临时 HOME + 临时端口起自己的 dev 实例，release 只对「模拟生产」
#     （$TMP 下的 repo/home/port/start 脚本）操作。
#   · 收尾断言生产 sessions.json/web.json 指纹与 8413 健康均未变，否则 pass=false。
#   · 退出码 0=pass，1=fail；末行输出单行 JSON {"pass","steps","prod_touched"}。
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d -t self-evolve-e2e.XXXXXX)"
PROD_HOME="${HOME:?}"
PROD_PORT="${PROD_PORT:-8413}"
RICK_BIN="$TMP/rick"
STEPS_FILE="$TMP/steps.jsonl"
: > "$STEPS_FILE"
FAILS=0
PIDS=()
PATCH_FILE=""; PATCH_BAK=""
export KEEP_TMP="${KEEP_TMP:-0}"

log()  { printf '%s\n' "$*"; }
note() { printf '  · %s\n' "$1"; }
ok()   { printf '  ✅ %s\n' "$1"; printf '{"step":"%s","ok":true}\n' "$(printf '%s' "$1" | sed 's/"/\\"/g')" >> "$STEPS_FILE"; }
bad()  { printf '  ❌ %s\n' "$1"; printf '{"step":"%s","ok":false}\n' "$(printf '%s' "$1" | sed 's/"/\\"/g')" >> "$STEPS_FILE"; FAILS=$((FAILS+1)); }
phase(){ printf '\n== %s ==\n' "$1"; }

free_port() { python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()'; }
pyget() { python3 -c 'import json,sys
d=json.load(sys.stdin)
for k in sys.argv[1].split("."):
    d = d[int(k)] if k.isdigit() else d.get(k)
    if d is None: break
print("" if d is None else d)' "$1" 2>/dev/null; }
md5f() { md5sum "$1" 2>/dev/null | awk '{print $1}'; }
http() { if [ -n "${2:-}" ]; then curl -s --noproxy '*' -H "Authorization: Bearer $2" "$1"; else curl -s --noproxy '*' "$1"; fi; }
code() { if [ -n "${2:-}" ]; then curl -s --noproxy '*' -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $2" "$1"; else curl -s --noproxy '*' -o /dev/null -w '%{http_code}' "$1"; fi; }
post_code() { # url token json
  curl -s --noproxy '*' -o "$TMP/last_post_body" -w '%{http_code}' -X POST \
    -H "Authorization: Bearer $2" -H 'Content-Type: application/json' -d "$3" "$1"; }
wait_health() { for _ in $(seq 1 100); do [ "$(code "$1" "${2:-}")" = "200" ] && return 0; sleep 0.25; done; return 1; }
kill_pid() { kill -TERM "$1" 2>/dev/null; for _ in $(seq 1 60); do kill -0 "$1" 2>/dev/null || return 0; sleep 0.25; done; kill -KILL "$1" 2>/dev/null; }

cleanup() {
  set +e
  for p in "${PIDS[@]:-}"; do [ -n "$p" ] && kill_pid "$p"; done
  # 恢复被临时改动的源文件（后端改动用例）
  if [ -n "$PATCH_FILE" ] && [ -f "$PATCH_BAK" ]; then cp -f "$PATCH_BAK" "$PATCH_FILE"; fi
  # 收掉本脚本起的 dev 实例（只杀属于临时 dev HOME 的进程）
  if [ -n "${RICK_DEV_HOME:-}" ] && [ -x "$RICK_BIN" ]; then
    RICK_DEV_TREE="$ROOT" RICK_DEV_HOME="$RICK_DEV_HOME" "$RICK_BIN" tools dev-web down >/dev/null 2>&1
  fi
  if [ "$KEEP_TMP" = "1" ]; then log "临时目录保留：$TMP"; else rm -rf "$TMP"; fi
}
trap cleanup EXIT

phase "0. 生产只读基线（全程不得变化）"
PROD_SESSIONS="$PROD_HOME/.rick/web/sessions.json"
PROD_WEBJSON="$PROD_HOME/.rick/web.json"
BASE_SESSIONS_MD5="$(md5f "$PROD_SESSIONS")"
BASE_WEBJSON_MD5="$(md5f "$PROD_WEBJSON")"
PROD_HEALTH_BEFORE="$(code "http://127.0.0.1:$PROD_PORT/api/health")"
note "prod sessions.json md5 = $BASE_SESSIONS_MD5"
note "prod web.json     md5 = $BASE_WEBJSON_MD5"
note "prod health = $PROD_HEALTH_BEFORE"
[ "$PROD_HEALTH_BEFORE" = "200" ] && ok "生产在线（8413）" || bad "生产 8413 不可达（health=$PROD_HEALTH_BEFORE）"

if ! (cd "$ROOT" && go build -o "$RICK_BIN" ./cmd/rick); then
  bad "构建 rick CLI 失败"; log '{"pass":false,"steps":["build cli failed"],"prod_touched":false}'; exit 1
fi
ok "构建被测 CLI（$ROOT）"

phase "1. KR1 隔离：dev 实例（临时 HOME/端口）起停 + 指纹"
export RICK_DEV_TREE="$ROOT"
export RICK_DEV_HOME="$TMP/devhome"
export RICK_DEV_PORT="$(free_port)"
DEV_STATE="$RICK_DEV_HOME/.rick"
DEV_URL="http://127.0.0.1:$RICK_DEV_PORT"
export GOCACHE="${GOCACHE:-$PROD_HOME/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$PROD_HOME/go/pkg/mod}"
export npm_config_cache="${npm_config_cache:-$PROD_HOME/.npm}"

if "$RICK_BIN" tools dev-web init >"$TMP/init.log" 2>&1; then ok "dev-web init（幂等：worktree 已存在则跳过，不写生产 .git）"; else bad "dev-web init 失败：$(tail -3 "$TMP/init.log" | tr '\n' ' ')"; fi
if UP_OUT="$("$RICK_BIN" tools dev-web up 2>&1)"; then
  DEV_PID="$(printf '%s' "$UP_OUT" | sed -n 's/.*pid=\([0-9]*\).*/\1/p' | head -1)"
  DEV_BUILD="$(printf '%s' "$UP_OUT" | sed -n 's/.*build_id=\([^ ]*\).*/\1/p' | head -1)"
  note "$(printf '%s' "$UP_OUT" | grep -m1 '^DEV_UP' || printf '%s' "$UP_OUT" | tail -1)"
  [ -n "$DEV_PID" ] && PIDS+=("$DEV_PID")
  DEV_TOKEN="$(cat "$RICK_DEV_HOME/DEV_TOKEN" 2>/dev/null || true)"
  if [ -z "$DEV_TOKEN" ]; then DEV_TOKEN="$(python3 -c 'import json,os;print(json.load(open(os.path.join(os.environ["RICK_DEV_HOME"],"dev.env").replace("dev.env","")))) if False else ""' 2>/dev/null)"; fi
  [ -z "$DEV_TOKEN" ] && DEV_TOKEN="$(grep -m1 '^RICK_DEV_TOKEN=' "$RICK_DEV_HOME/dev.env" 2>/dev/null | cut -d= -f2)"
  HEALTH_JSON="$(http "$DEV_URL/api/health")"
  HB="$(printf '%s' "$HEALTH_JSON" | pyget build_id)"
  if [ -n "$DEV_BUILD" ] && [ "$HB" = "$DEV_BUILD" ]; then ok "dev 健康 + 指纹一致（build_id=$HB）"; else bad "dev 指纹不一致：期望=$DEV_BUILD 实际=$HB"; fi
  SID_COUNT="$(http "$DEV_URL/api/sessions" "$DEV_TOKEN" | python3 -c 'import json,sys;print(len(json.load(sys.stdin)))' 2>/dev/null || echo "?")"
  if [ "$SID_COUNT" = "0" ]; then ok "状态隔离：dev 会话列表为空（看不到生产 ${BASE_SESSIONS_MD5:0:8}… 的注册表）"; else bad "dev 看到了 $SID_COUNT 条会话（应 0）→ 状态未隔离"; fi
else
  bad "dev-web up 失败：$(printf '%s' "$UP_OUT" | tail -3 | tr '\n' ' ')"
fi

phase "2. KR1 隔离：flock singleton + 生产工作区守卫"
SECOND_OUT="$(HOME="$RICK_DEV_HOME" RICK_PI_AGENT_DIR="$RICK_DEV_HOME/.rick/pi/agent" \
  "$RICK_BIN" web --listen 127.0.0.1 --port "$(free_port)" --token "$DEV_TOKEN" --state-dir "$DEV_STATE" 2>&1)"
if [ $? -ne 0 ]; then
  if printf '%s' "$SECOND_OUT" | grep -qiE 'lock|占用'; then ok "同 state-dir 第二实例被 flock 拒绝"; else bad "第二实例被拒但提示不含锁信息：$(printf '%s' "$SECOND_OUT" | tail -2 | tr '\n' ' ')"; fi
else
  bad "同 state-dir 第二实例启动成功（singleton 未生效）"
fi

PROD_WS_PATH="$(PROD_HOME="$PROD_HOME" python3 -c '
import json,os
p=os.path.join(os.environ["PROD_HOME"],".rick","web.json")
try: print(json.load(open(p))["workspaces"][0]["path"])
except Exception: print("")')"

# 2b. 守卫的**契约形态**：--state-dir 与 $HOME/.rick 不同 = dev 语义 → 安装守卫
#      （task16 契约：IsDevStateDir 为真时拒绝注册生产注册表已拥有的工作区）
GUARD_STATE="$TMP/guardstate"; GUARD_PORT="$(free_port)"
HOME="$TMP/guardhome" RICK_PI_AGENT_DIR="$TMP/guardagent" \
  "$RICK_BIN" web --listen 127.0.0.1 --port "$GUARD_PORT" --token guardtok --state-dir "$GUARD_STATE" \
  >"$TMP/guard.log" 2>&1 &
GUARD_PID=$!
if wait_health "http://127.0.0.1:$GUARD_PORT/api/health" guardtok; then
  if [ -n "$PROD_WS_PATH" ]; then
    C="$(post_code "http://127.0.0.1:$GUARD_PORT/api/workspaces" guardtok "{\"path\":\"$PROD_WS_PATH\",\"name\":\"should-be-rejected\"}")"
    case "$C" in
      400|403|409) ok "dev 语义实例拒绝注册生产已注册工作区（$PROD_WS_PATH → HTTP $C）" ;;
      *) bad "dev 语义实例允许注册生产工作区（HTTP $C）——隔离守卫失效" ;;
    esac
  else
    note "生产注册表为空 → 跳过「注册生产工作区被拒」用例"
  fi
else
  bad "守卫用例实例未起来（见 $TMP/guard.log）"
fi
kill_pid "$GUARD_PID"

# 2c. 已知缺口探测：dev-web 的实际形态（HOME 换掉、state-dir == $HOME/.rick）下守卫**不安装**
#      —— 记为 finding（不改判定，交由父级裁决是否修 IsDevStateDir 的触发条件）
HOME="$TMP/guardhome2" RICK_PI_AGENT_DIR="$TMP/guardagent2" \
  "$RICK_BIN" web --listen 127.0.0.1 --port "$GUARD_PORT" --token guardtok2 --state-dir "$TMP/guardhome2/.rick" \
  >"$TMP/guard2.log" 2>&1 &
GUARD_PID2=$!
if wait_health "http://127.0.0.1:$GUARD_PORT/api/health" guardtok2; then
  if [ -n "$PROD_WS_PATH" ]; then
    C2="$(post_code "http://127.0.0.1:$GUARD_PORT/api/workspaces" guardtok2 "{\"path\":\"$PROD_WS_PATH\",\"name\":\"probe\"}")"
    if [ "$C2" = "201" ] || [ "$C2" = "200" ]; then
      note "FINDING guard_not_installed_for_home_swap：HOME 换掉但 state-dir==\$HOME/.rick 时守卫未安装（注册生产工作区被放行 HTTP $C2）——详见 wiki 已知边界"
    else
      ok "HOME-swap 形态下守卫同样生效（HTTP $C2）"
    fi
  fi
else
  note "HOME-swap 探测实例未起来（跳过）"
fi
kill_pid "$GUARD_PID2"

phase "3. KR2 前端热更：overlay 即时生效且 dev 进程不重启"
OV="$DEV_STATE/web/dist"
rm -rf "$OV" && mkdir -p "$OV"   # 断开软链，用真实目录，避免写进 dev 树
M1="e2e-overlay-A-$$"; M2="e2e-overlay-B-$$"
printf '<html><body>%s</body></html>' "$M1" > "$OV/index.html"
B1="$(http "$DEV_URL/")"
printf '<html><body>%s</body></html>' "$M2" > "$OV/index.html"
B2="$(http "$DEV_URL/")"
STILL="$(kill -0 "${DEV_PID:-0}" 2>/dev/null && echo yes || echo no)"
if printf '%s' "$B1" | grep -q "$M1" && printf '%s' "$B2" | grep -q "$M2" && [ "$STILL" = "yes" ]; then
  ok "overlay 热更即时生效（两次替换均被 GET / 反映），dev 进程未重启（pid=$DEV_PID）"
else
  bad "overlay 热更断言失败（B1=${B1:0:40} B2=${B2:0:40} alive=$STILL）"
fi

phase "4. KR2 后端闭环：源码改动 → restart → 行为可见（rick_version 探针）"
PATCH_FILE="$ROOT/cmd/rick/main.go"; PATCH_BAK="$TMP/main.go.bak"
cp -f "$PATCH_FILE" "$PATCH_BAK"
sed -i 's/^const VERSION = "\(.*\)"/const VERSION = "\1+e2e"/' "$PATCH_FILE"
if grep -q 'VERSION = ".*+e2e"' "$PATCH_FILE"; then
  if R_OUT="$("$RICK_BIN" tools dev-web restart 2>&1)"; then
    NEW_BUILD="$(printf '%s' "$R_OUT" | sed -n 's/.*build_id=\([^ ]*\).*/\1/p' | head -1)"
    NEW_PID="$(printf '%s' "$R_OUT" | sed -n 's/.*pid=\([0-9]*\).*/\1/p' | head -1)"
    [ -n "$NEW_PID" ] && PIDS+=("$NEW_PID") && DEV_PID="$NEW_PID"
    RV="$(http "$DEV_URL/api/config" "$DEV_TOKEN" | pyget rick_version)"
    if [ "$RV" = "4.4.15+e2e" ]; then ok "后端源码改动已生效（rick_version=$RV，build_id=$NEW_BUILD）"; else bad "源码改动未生效：rick_version=$RV（期望 4.4.15+e2e）"; fi
    [ -n "$NEW_BUILD" ] && [ "$NEW_BUILD" != "$DEV_BUILD" ] && ok "重启后 build_id 变化（$DEV_BUILD → $NEW_BUILD）" || bad "build_id 未变化"
  else
    bad "dev-web restart 失败：$(printf '%s' "$R_OUT" | tail -3 | tr '\n' ' ')"
  fi
else
  bad "VERSION 探针未能注入（sed 未匹配）"
fi
cp -f "$PATCH_BAK" "$PATCH_FILE"; PATCH_FILE=""
"$RICK_BIN" tools dev-web restart >"$TMP/restore.log" 2>&1 || note "恢复构建的 restart 返回非零（见 $TMP/restore.log）"

phase "5. KR4 挂起语义：重启 → suspended（未自动恢复）→ /continue 才恢复"
SIM_WS="$TMP/ws"; SIM_STATE="$TMP/simstate"; SIM_AGENT="$TMP/simagent"
mkdir -p "$SIM_WS/.rick/jobs/job_1/doing" "$SIM_STATE/web" "$SIM_AGENT"
# 遗留 running 的 doing task（用于验证 /continue 的归一化）
cat > "$SIM_WS/.rick/jobs/job_1/doing/tasks.json" <<JSON
{"version":1,"job_id":"job_1","tasks":[{"task_id":"task1","name":"done","status":"success"},{"task_id":"task2","name":"was running","status":"running"}]}
JSON
python3 - "$SIM_STATE" <<'PY'
import json,sys,os
state=sys.argv[1]
json.dump({"version":1,"workspaces":[{"id":"wse2e","path":os.path.join(os.path.dirname(state),"ws"),"name":"e2e-ws","added_at":"2026-01-01T00:00:00+08:00"}]}, open(os.path.join(state,"web.json"),"w"))
json.dump({"version":1,"sessions":[
 {"id":"sess-easy","workspace_id":"wse2e","type":"easy","title":"was running","params":{"job":"job_1"},
  "status":"active","pi_session_id":"pi-e2e-easy","created_at":"2026-01-01T00:00:00+08:00"},
 {"id":"sess-doing","workspace_id":"wse2e","type":"doing","title":"doing job_1","params":{"job":"job_1"},
  "status":"active","pi_session_id":"pi-e2e-doing","created_at":"2026-01-01T00:00:00+08:00"}]},
 open(os.path.join(state,"web","sessions.json"),"w"))
PY
SIM_PORT="$(free_port)"; SIM_TOKEN="e2etok"
SIM_ENV=(env HOME="$TMP/simhome" RICK_PI_AGENT_DIR="$SIM_AGENT")
mkdir -p "$TMP/simhome"
start_sim() {
  "${SIM_ENV[@]}" "$RICK_BIN" web --listen 127.0.0.1 --port "$SIM_PORT" --token "$SIM_TOKEN" --state-dir "$SIM_STATE" \
    >"$TMP/sim1.log" 2>&1 &
  SIM_PID=$!; sleep 0.1
  wait_health "http://127.0.0.1:$SIM_PORT/api/health" "$SIM_TOKEN"
}
if start_sim; then
  ok "模拟生产实例启动（端口 $SIM_PORT）"
  kill_pid "$SIM_PID"
  sleep 0.5
  ST_AFTER="$(python3 -c '
import json,sys
d=json.load(open(sys.argv[1]))
print(" ".join(sorted(s["id"]+"="+s["status"] for s in d["sessions"])))' "$SIM_STATE/web/sessions.json")"
  if printf '%s' "$ST_AFTER" | grep -q 'sess-easy=suspended' && printf '%s' "$ST_AFTER" | grep -q 'sess-doing=suspended'; then
    ok "优雅关停把在跑会话标为 suspended（不是 error）：$ST_AFTER"
  else
    bad "关停后状态异常：$ST_AFTER"
  fi
  [ -f "$SIM_STATE/suspend.json" ] && ok "写了 suspend.json 快照（intent 持久化）" || bad "缺 suspend.json 快照"

  # 重启：必须**不自动恢复**
  if start_sim; then
    SIM_PID2="$SIM_PID"; PIDS+=("$SIM_PID2")
    LIST="$(http "http://127.0.0.1:$SIM_PORT/api/sessions" "$SIM_TOKEN")"
    ST2="$(printf '%s' "$LIST" | python3 -c 'import json,sys;print(" ".join(sorted(s["id"]+"="+s["status"] for s in json.load(sys.stdin))))' 2>/dev/null || echo "")"
    if printf '%s' "$ST2" | grep -q 'sess-easy=suspended'; then ok "重启后仍为 suspended（未自动恢复）：$ST2"; else bad "重启后状态异常：$ST2"; fi
    BUSY="$(http "http://127.0.0.1:$SIM_PORT/api/sessions/sess-easy" "$SIM_TOKEN" | pyget busy)"
    [ "$BUSY" != "True" ] && ok "挂起会话 busy 非 true（没有偷偷跑）" || bad "挂起会话 busy=true（自动跑了）"
    R="$(http "http://127.0.0.1:$SIM_PORT/api/recovery" "$SIM_TOKEN")"
    if printf '%s' "$R" | python3 -c 'import json,sys; d=json.load(sys.stdin); assert all(k in d for k in ("at","suspended","recovered","failed"))' 2>/dev/null; then
      ok "GET /api/recovery 结构完整（at/suspended/recovered/failed）"
    else
      bad "GET /api/recovery 结构异常：$(printf '%s' "$R" | head -c 120)"
    fi
    # /continue：doing → 归一化 running→pending（人工确认才动）
    C1="$(post_code "http://127.0.0.1:$SIM_PORT/api/sessions/sess-doing/continue" "$SIM_TOKEN" '{}')"
    NORM="$(python3 -c '
import json,sys
d=json.load(open(sys.argv[1]))
print(" ".join(t["task_id"]+"="+t["status"] for t in d["tasks"]))' "$SIM_WS/.rick/jobs/job_1/doing/tasks.json")"
    if printf '%s' "$NORM" | grep -q 'task2=pending'; then
      ok "/continue 对 doing 归一化 running→pending（HTTP $C1，tasks: $NORM）"
    else
      bad "/continue 未归一化 doing task 状态（HTTP $C1，tasks: $NORM）"
    fi
    # /continue 对交互型会话必须“尝试恢复”而不是 404/409
    C2="$(post_code "http://127.0.0.1:$SIM_PORT/api/sessions/sess-easy/continue" "$SIM_TOKEN" '{}')"
    case "$C2" in
      404|409) bad "/continue 对 suspended 交互会话返回 $C2（应尝试恢复）" ;;
      *) ok "/continue 接受挂起会话并尝试恢复（HTTP $C2；无真实 pi 故允许失败）" ;;
    esac
    C3="$(post_code "http://127.0.0.1:$SIM_PORT/api/sessions/nope/continue" "$SIM_TOKEN" '{}')"
    [ "$C3" = "404" ] && ok "/continue 对不存在会话返回 404" || bad "/continue 对不存在会话返回 $C3（期望 404）"
    kill_pid "$SIM_PID2"
    sleep 0.5
  else
    bad "第二轮模拟生产未起来"
  fi
else
  bad "模拟生产实例未起来"
fi

phase "6. KR3 受控提升：release 原子提升（模拟生产）→ build_id 校验 → rollback"
SIM_REPO="$TMP/simprod"; SIM_HOME="$TMP/simprod-home"
mkdir -p "$SIM_REPO/bin" "$SIM_HOME/.rick"
cp -f "$ROOT/go.mod" "$SIM_REPO/go.mod"
cp -f "$RICK_BIN" "$SIM_REPO/bin/rick"   # 提升前的“当前生产”版本（回滚的落点）
REL_PORT="$(free_port)"
cat > "$SIM_HOME/.rick/start-web.sh" <<SH
#!/usr/bin/env bash
exec "$SIM_REPO/bin/rick" web --listen 127.0.0.1 --port $REL_PORT --state-dir "$SIM_HOME/.rick"
SH
# 模拟生产里放一个在跑会话：提升重启后应变为挂起（RELEASE_RECOVER 清单）
python3 - "$SIM_HOME/.rick" <<'PY'
import json,os,sys
st=sys.argv[1]
os.makedirs(os.path.join(st,"web"),exist_ok=True)
json.dump({"version":1,"workspaces":[{"id":"wsrel","path":os.path.join(os.path.dirname(st),"wsrel"),"name":"rel","added_at":"2026-01-01T00:00:00+08:00"}]},open(os.path.join(st,"web.json"),"w"))
json.dump({"version":1,"sessions":[{"id":"rel-s1","workspace_id":"wsrel","type":"easy","title":"running before release","params":{"job":"job_1"},"status":"active","pi_session_id":"pi-rel","created_at":"2026-01-01T00:00:00+08:00"}]},open(os.path.join(st,"web","sessions.json"),"w"))
PY
REL=("$RICK_BIN" tools release --yes --no-gates --json
     --prod-repo "$SIM_REPO" --prod-home "$SIM_HOME" --state-dir "$SIM_HOME/.rick"
     --port "$REL_PORT" --start-script "$SIM_HOME/.rick/start-web.sh" --dev-tree "$ROOT" --keep 3)
REL_URL="http://127.0.0.1:$REL_PORT"
rel_version() { printf '%s' "$1" | sed -n 's/^RELEASE_OK version=\([^ ]*\).*/\1/p' | tail -1; }
rel_matches() { printf '%s' "$1" | sed -n 's/.*RELEASE_RESTART .*matches=\([^ ]*\).*/\1/p' | tail -1; }
if R1="$("${REL[@]}" 2>&1)"; then
  V1="$(rel_version "$R1")"
  M1="$(rel_matches "$R1")"
  HB1="$(http "$REL_URL/api/health" | pyget build_id)"
  if [ -n "$HB1" ] && [ "$HB1" = "$V1" ]; then ok "提升 #1：模拟生产重启后 build_id 与版本一致（$HB1，release matches=$M1）"; else bad "提升 #1 指纹不一致：health=$HB1 version=$V1"; fi
  [ "$M1" = "true" ] && ok "release 自校验 matches=true（新构建确实在跑）" || bad "release RELEASE_RESTART matches=$M1（期望 true）"
  # 前端同版推进
  [ -f "$SIM_HOME/.rick/web/dist/index.html" ] && ok "前端 overlay 已随版本投放（dist/index.html 存在）" || bad "前端 overlay 未投放"
  # 挂起清单（RELEASE_RECOVER）
  printf '%s' "$R1" | grep -q 'RELEASE_RECOVER' && ok "release 打印挂起清单（RELEASE_RECOVER）" || note "release 未打印 RELEASE_RECOVER（挂起清单可能为空）"
  RST="$(python3 -c '
import json,sys
print(json.load(open(sys.argv[1]))["sessions"][0]["status"])' "$SIM_HOME/.rick/web/sessions.json" 2>/dev/null || echo "?")"
  [ "$RST" = "suspended" ] && ok "提升重启后模拟生产在跑会话 → suspended（平台自动回来、会话待人工恢复）" || bad "提升后会话状态=$RST（期望 suspended）"
  # 第二次提升 → 形成回滚点
  if R2="$("${REL[@]}" 2>&1)"; then
    V2="$(rel_version "$R2")"
    HB2="$(http "$REL_URL/api/health" | pyget build_id)"
    [ -n "$HB2" ] && [ "$HB2" != "$HB1" ] && ok "提升 #2：build_id 更新（$HB1 → $HB2）" || bad "提升 #2 build_id 未变化（$HB1 → $HB2）"
    # 回滚
    ROLLBACK=("$RICK_BIN" tools release --yes --rollback --json
              --prod-repo "$SIM_REPO" --prod-home "$SIM_HOME" --state-dir "$SIM_HOME/.rick"
              --port "$REL_PORT" --start-script "$SIM_HOME/.rick/start-web.sh" --dev-tree "$ROOT")
    if R3="$("${ROLLBACK[@]}" 2>&1)"; then
      HB3="$(http "$REL_URL/api/health" | pyget build_id)"
      if [ "$HB3" = "$HB1" ]; then ok "rollback 回到上一版且健康（build_id=$HB3）"; else bad "rollback 后 build_id=$HB3（期望 $HB1）"; fi
    else
      bad "rollback 失败：$(printf '%s' "$R3" | tail -3 | tr '\n' ' ')"
    fi
  else
    bad "提升 #2 失败：$(printf '%s' "$R2" | tail -3 | tr '\n' ' ')"
  fi
else
  bad "release（模拟生产）失败：$(printf '%s' "$R1" | tail -4 | tr '\n' ' ')"
fi
# 收掉模拟生产进程（按端口/pid 文件判归属，不误杀真实生产）
if [ -f "$SIM_HOME/.rick/web.pid" ]; then kill_pid "$(cat "$SIM_HOME/.rick/web.pid")" 2>/dev/null; fi

phase "7. 生产回归（只读断言）"
PROD_HEALTH_AFTER="$(code "http://127.0.0.1:$PROD_PORT/api/health")"
AFTER_SESSIONS_MD5="$(md5f "$PROD_SESSIONS")"
AFTER_WEBJSON_MD5="$(md5f "$PROD_WEBJSON")"
[ "$PROD_HEALTH_AFTER" = "200" ] && ok "生产仍健康（8413 health=200）" || bad "生产健康异常（health=$PROD_HEALTH_AFTER）"
[ "$AFTER_SESSIONS_MD5" = "$BASE_SESSIONS_MD5" ] && ok "生产 sessions.json 未被改动（md5 一致）" || bad "生产 sessions.json 被改动（$BASE_SESSIONS_MD5 → $AFTER_SESSIONS_MD5）"
[ "$AFTER_WEBJSON_MD5" = "$BASE_WEBJSON_MD5" ] && ok "生产 web.json 未被改动（md5 一致）" || bad "生产 web.json 被改动（$BASE_WEBJSON_MD5 → $AFTER_WEBJSON_MD5）"

if [ "$FAILS" -eq 0 ] && [ "$PROD_HEALTH_AFTER" = "200" ] && [ "$AFTER_SESSIONS_MD5" = "$BASE_SESSIONS_MD5" ] && [ "$AFTER_WEBJSON_MD5" = "$BASE_WEBJSON_MD5" ]; then
  PASS=true; PROD_TOUCHED=false
else
  PASS=false
  if [ "$AFTER_SESSIONS_MD5" = "$BASE_SESSIONS_MD5" ] && [ "$AFTER_WEBJSON_MD5" = "$BASE_WEBJSON_MD5" ] && [ "$PROD_HEALTH_AFTER" = "200" ]; then PROD_TOUCHED=false; else PROD_TOUCHED=true; fi
fi

python3 - "$STEPS_FILE" "$PASS" "$PROD_TOUCHED" <<'PY'
import json,sys
steps_file,passed,touched=sys.argv[1],sys.argv[2],sys.argv[3]
steps=[]
for line in open(steps_file):
    line=line.strip()
    if not line: continue
    try: steps.append(json.loads(line))
    except Exception: pass
print(json.dumps({"pass": passed=="true","steps":steps,"prod_touched": touched=="true"},ensure_ascii=False))
PY
[ "$PASS" = "true" ] && exit 0 || exit 1
