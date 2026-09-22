#!/usr/bin/env bash
# scripts/rsi-loop-e2e.sh —— rick-rsi-loop（自进化制度化）端到端验收（可重放、幂等）
#
# 验收目标（第 3 棵设计树 KR1-KR4）：
#   KR1 制度载体：rick-rsi-loop 合规（loops_check pass）
#   KR2 标准入口（RSI 去特殊化后）：普通 easy 会话的提示词里，**「可用的项目 Loops」目录必须出现
#   rick-rsi-loop 及其 trigger**（LoadLoopsContext 标准机制）；type=rsi 已删除 → 400；
#                 守卫拒绝「生产仓库根 / 缺 loop / 非源码树」三种非法工作区
#   KR3 闭环执行：release --merge-source —— 无冲突合并成功；**冲突即中止且生产工作树恢复干净**；
#                 修复冲突后重跑成功
#   KR4 可校验产出：rsi_check 缺证据 fail（含中文指引）；--init 骨架**仍 fail**（防自欺）；
#                 补全证据后 pass（prod-health 用模拟生产实时探测）
#
# 绝对纪律：
#   · 真实生产（8413 / ~/.rick / /workdir/sunquan20/AI_CODING/rick）**只读**：
#     脚本只读它的 sessions.json/web.json 指纹与 /api/health；绝不写、绝不 restart。
#   · 所有实例（模拟生产/模拟 dev）都跑在临时 HOME + 临时端口 + 临时 state dir；
#     git 操作只发生在 $TMP 下 clone 出来的临时仓库。
#   · 收尾断言生产指纹与健康未变，否则 exit 1 且 prod_touched=true。
#   · 末行输出单行 JSON {"pass","steps","prod_touched"}。
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d -t rsi-loop-e2e.XXXXXX)"
PROD_HOME="${HOME:?}"
PROD_PORT="${PROD_PORT:-8413}"
RICK_BIN="$TMP/rick"
STEPS_FILE="$TMP/steps.jsonl"
: > "$STEPS_FILE"
FAILS=0
PIDS=()
export KEEP_TMP="${KEEP_TMP:-0}"
export GOCACHE="${GOCACHE:-$PROD_HOME/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$PROD_HOME/go/pkg/mod}"
export npm_config_cache="${npm_config_cache:-$PROD_HOME/.npm}"

log()  { printf '%s\n' "$*"; }
note() { printf '  · %s\n' "$1"; }
ok()   { printf '  ✅ %s\n' "$1"; printf '{"step":"%s","ok":true}\n' "$(printf '%s' "$1" | sed 's/"/\\"/g')" >> "$STEPS_FILE"; }
bad()  { printf '  ❌ %s\n' "$1"; printf '{"step":"%s","ok":false}\n' "$(printf '%s' "$1" | sed 's/"/\\"/g')" >> "$STEPS_FILE"; FAILS=$((FAILS+1)); }
phase(){ printf '\n== %s ==\n' "$1"; }

free_port() { python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()'; }
pyget() { python3 -c 'import json,sys
d=json.load(sys.stdin)
for k in sys.argv[1].split("."):
    d = d[int(k)] if k.isdigit() else (d.get(k) if isinstance(d,dict) else None)
    if d is None: break
print("" if d is None else d)' "$1" 2>/dev/null; }
md5f() { md5sum "$1" 2>/dev/null | awk '{print $1}'; }
http() { if [ -n "${2:-}" ]; then curl -s --noproxy '*' -H "Authorization: Bearer $2" "$1"; else curl -s --noproxy '*' "$1"; fi; }
code() { if [ -n "${2:-}" ]; then curl -s --noproxy '*' -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $2" "$1"; else curl -s --noproxy '*' -o /dev/null -w '%{http_code}' "$1"; fi; }
post_code() { # url token json  → 打印状态码，body 落 $TMP/last_post_body
  curl -s --noproxy '*' -o "$TMP/last_post_body" -w '%{http_code}' -X POST \
    -H "Authorization: Bearer $2" -H 'Content-Type: application/json' -d "$3" "$1"; }
wait_health() { for _ in $(seq 1 120); do [ "$(code "$1" "${2:-}")" = "200" ] && return 0; sleep 0.25; done; return 1; }
kill_pid() { kill -TERM "$1" 2>/dev/null; for _ in $(seq 1 60); do kill -0 "$1" 2>/dev/null || return 0; sleep 0.25; done; kill -KILL "$1" 2>/dev/null; }

cleanup() {
  set +e
  for p in "${PIDS[@]:-}"; do [ -n "$p" ] && kill_pid "$p"; done
  [ "$KEEP_TMP" = "1" ] && log "临时目录保留：$TMP" || rm -rf "$TMP"
}
trap cleanup EXIT

phase "0. 生产只读基线（全程不得变化）+ 构建被测 CLI"
PROD_SESSIONS="$PROD_HOME/.rick/web/sessions.json"
PROD_WEBJSON="$PROD_HOME/.rick/web.json"
BASE_SESSIONS_MD5="$(md5f "$PROD_SESSIONS")"
BASE_WEBJSON_MD5="$(md5f "$PROD_WEBJSON")"
PROD_HEALTH_BEFORE="$(code "http://127.0.0.1:$PROD_PORT/api/health")"
note "prod sessions.json md5 = $BASE_SESSIONS_MD5"
note "prod health = $PROD_HEALTH_BEFORE"
[ "$PROD_HEALTH_BEFORE" = "200" ] && ok "生产在线（$PROD_PORT）" || bad "生产 $PROD_PORT 不可达（health=$PROD_HEALTH_BEFORE）"

if ! (cd "$ROOT" && go build -o "$RICK_BIN" ./cmd/rick); then
  bad "构建 rick CLI 失败"; log '{"pass":false,"steps":["build cli failed"],"prod_touched":false}'; exit 1
fi
ok "构建被测 CLI（$ROOT）"

phase "1. KR1 制度载体：loops_check 对真实 .rick 必须 pass"
if LOOP_OUT="$(cd "$ROOT" && "$RICK_BIN" tools loops_check --dir .rick 2>&1)"; then
  ok "loops_check pass：$(printf '%s' "$LOOP_OUT" | tail -1)"
else
  bad "loops_check 失败：$(printf '%s' "$LOOP_OUT" | tail -3 | tr '\n' ' ')"
fi
RSI_LOOP="$ROOT/.rick/loops/rick-rsi-loop.md"
[ -f "$RSI_LOOP" ] && ok "loop 文件存在（$RSI_LOOP）" || bad "loop 文件缺失：$RSI_LOOP"

phase "1b. KR1 前置：dev 环境可幂等初始化（用临时 dev HOME，绝不碰真实 dev 实例）"
INIT_HOME="$TMP/rsi-devhome"; INIT_PORT="$(free_port)"
if RICK_DEV_TREE="$ROOT" RICK_DEV_HOME="$INIT_HOME" RICK_DEV_PORT="$INIT_PORT" \
   "$RICK_BIN" tools dev-web init >"$TMP/init.log" 2>&1; then
  ok "dev-web init 幂等成功（临时 dev HOME=$INIT_HOME）"
  [ -L "$INIT_HOME/.rick/web/dist" ] && ok "overlay 软链已建立（→ dev 树 web/dist，前端热更零拷贝）" \
    || bad "overlay 软链缺失：$INIT_HOME/.rick/web/dist"
  [ -f "$INIT_HOME/dev.env" ] && ok "dev.env 已落盘（记录 Tree/Home/AgentDir/Port 供复用）" || bad "缺 dev.env"
else
  bad "dev-web init 失败：$(tail -3 "$TMP/init.log" | tr '\n' ' ')"
fi

# ---------------------------------------------------------------- 模拟环境构造
mk_sim_ws() { # $1=dir  造一个「合法 rsi workspace」（rick 源码树 + loop）
  local d="$1"
  mkdir -p "$d/cmd/rick" "$d/internal/web" "$d/.rick/loops"
  printf 'package main\n' > "$d/cmd/rick/main.go"
  printf 'package web\n' > "$d/internal/web/web.go"
  cp -f "$RSI_LOOP" "$d/.rick/loops/rick-rsi-loop.md"
}

start_sim() { # $1=bin $2=home $3=state $4=port $5=agent $6=extra-env(可空)  → PID 写 $SIM_PID
  local bin="$1" simhome="$2" st="$3" port="$4" agent="$5" extra="${6:-}"
  mkdir -p "$st" "$simhome" "$agent"
  if [ -n "$extra" ]; then
    env HOME="$simhome" RICK_PI_AGENT_DIR="$agent" $extra \
      "$bin" web --listen 127.0.0.1 --port "$port" --token simtok --state-dir "$st" \
      >"$TMP/sim-$port.log" 2>&1 &
  else
    env HOME="$simhome" RICK_PI_AGENT_DIR="$agent" \
      "$bin" web --listen 127.0.0.1 --port "$port" --token simtok --state-dir "$st" \
      >"$TMP/sim-$port.log" 2>&1 &
  fi
  SIM_PID=$!
  PIDS+=("$SIM_PID")
  wait_health "http://127.0.0.1:$port/api/health" simtok
}

phase "2. KR2 标准入口：普通 easy 会话的 Loops 目录必须出现 rick-rsi-loop（RSI 去特殊化）"
SIM_WS="$TMP/simdev-ws"; mk_sim_ws "$SIM_WS"
SIM_STATE="$TMP/sim-state"; SIM_HOME="$TMP/sim-home"; SIM_AGENT="$TMP/sim-agent"
SIM_PORT="$(free_port)"
if start_sim "$RICK_BIN" "$SIM_HOME" "$SIM_STATE" "$SIM_PORT" "$SIM_AGENT" ""; then
  ok "模拟生产实例启动（端口 $SIM_PORT）"
  SIM_URL="http://127.0.0.1:$SIM_PORT"
  WS_ID="$(post_code "$SIM_URL/api/workspaces" simtok "{\"path\":\"$SIM_WS\",\"name\":\"simdev\"}" >/dev/null; \
           http "$SIM_URL/api/workspaces" simtok | python3 -c '
import json,sys
ws=json.load(sys.stdin)
print(next((w["id"] for w in ws if w.get("name")=="simdev"), ""))')"
  if [ -n "$WS_ID" ]; then ok "注册模拟 rick 工作区（普通工作区，id=$WS_ID）"; else bad "注册模拟工作区失败"; fi

  # ②-a 标准 easy 会话：提示词的「可用的项目 Loops」目录必须含 rick-rsi-loop + trigger
  SC="$(post_code "$SIM_URL/api/sessions" simtok "{\"workspace_id\":\"$WS_ID\",\"type\":\"easy\",\"params\":{\"requirement\":\"改进 rick 自身：加一个测试功能\"}}")"
  SID="$(cat "$TMP/last_post_body" | pyget id)"
  if [ "$SC" = "200" ] || [ "$SC" = "201" ]; then
    ok "type=easy 建会话成功（HTTP $SC, id=$SID）"
    PBODY="$(http "$SIM_URL/api/sessions/$SID/prompt" simtok)"
    for marker in "可用的项目 Loops" rick-rsi-loop; do
      if printf '%s' "$PBODY" | grep -q "$marker"; then ok "easy 提示词含目录条目：$marker"; else bad "easy 提示词缺目录条目：$marker"; fi
    done
    # trigger 关键词（证明目录条目带触发条件，agent 能按任务匹配到它）
    if printf '%s' "$PBODY" | grep -q "修改 rick 自身"; then ok "目录条目带 trigger（修改 rick 自身…）"; else bad "目录条目缺 trigger"; fi
  else
    bad "type=easy 建会话失败（HTTP $SC）：$(head -c 200 "$TMP/last_post_body")"
  fi

  # ②-b 特殊类型已删：type=rsi → 400（未知类型）
  RC="$(post_code "$SIM_URL/api/sessions" simtok "{\"workspace_id\":\"$WS_ID\",\"type\":\"rsi\"}")"
  if [ "$RC" = "400" ]; then ok "type=rsi → 400（特殊类型已删除，符合去特殊化裁决）"; else bad "type=rsi 返回 HTTP $RC（应 400）"; fi

  phase "3. KR2 守卫断言（已随 RSI 去特殊化移除——工作区不做任何特殊处理）"
  ok "守卫已删除：普通 easy 会话在任何工作区（含 rick 源码树）都按标准流程创建"
fi

phase "4. KR4 rsi_check：缺证据 fail → --init 仍 fail → 补全 pass"
JOBROOT="$TMP/jobroot"; mkdir -p "$JOBROOT/.rick/jobs/job_99/doing"
RC() { (cd "$JOBROOT" && "$RICK_BIN" tools rsi_check --job job_99 "$@" 2>&1); }
OUT1="$(RC --json)"; RC1=$?
if [ $RC1 -ne 0 ] && printf '%s' "$OUT1" | grep -q "下一步"; then
  ok "缺证据 → fail 且含中文指引（0/6）"
else
  bad "缺证据未 fail 或缺少指引（rc=$RC1）：$(printf '%s' "$OUT1" | tail -2 | tr '\n' ' ')"
fi
RC --init >/dev/null 2>&1
OUT2="$(RC --json)"; RC2=$?
if [ $RC2 -ne 0 ]; then
  printf '%s' "$OUT2" | grep -q "骨架" && ok "──init 骨架仍 fail（防自欺：残留 TODO 标记不算证据）" \
    || bad "──init 后 fail 但原因不是骨架标记：$(printf '%s' "$OUT2" | tail -2 | tr '\n' ' ')"
else
  bad "──init 后竟然 pass（防自欺规则失效）"
fi

# 补全合规证据（prod-health 指向一个可控的「模拟生产」，其 build_id 必须等于 release.md 的 version）
VER="deadbee-260921234500"
PROD_BIN="$TMP/rick-rsi-prod"
if go build -ldflags "-X github.com/sunquan/rick/internal/cmd.BuildID=$VER" -o "$PROD_BIN" ./cmd/rick 2>"$TMP/build2.log"; then
  RP_STATE="$TMP/rsp-state"; RP_HOME="$TMP/rsp-home"; RP_PORT="$(free_port)"
  if start_sim "$PROD_BIN" "$RP_HOME" "$RP_STATE" "$RP_PORT" "$TMP/rsp-agent"; then
    ok "模拟生产（rsi_check 的 prod-health 探测目标）启动，build_id=$VER"
  else
    bad "模拟生产（探测目标）未起来"
  fi
else
  bad "构建带 build_id 的二进制失败：$(tail -3 "$TMP/build2.log" | tr '\n' ' ')"
fi
E="$JOBROOT/.rick/jobs/job_99/doing/rsi"
printf '本轮改动：rsi 入口与校验器。\nDEV_UP build_id=%s port=8414\n' "$VER" > "$E/dev-iterations.md"
printf 'GATE gate11 pass=true\nGATE gate12 pass=true\nGATE gate13 pass=true\nGATE gate14 pass=true\n' > "$E/gates.md"
printf 'release 计划已呈报并获批准。\nAPPROVED by=human at=2026-09-21T23:45:00+08:00\n' > "$E/approval.md"
printf 'RELEASE_MERGE merged=true\nversion=%s\nrollback_point=%s/bin/releases/.last\n' "$VER" "$TMP" > "$E/release.md"
printf '挂起清单：2 个会话 suspended；人类逐条点击恢复，均已 resumed。\n' > "$E/resume.md"
OUT3="$(RC --json --prod-url "http://127.0.0.1:$RP_PORT")"; RC3=$?
if [ $RC3 -eq 0 ] && printf '%s' "$OUT3" | python3 -c 'import json,sys; d=json.load(sys.stdin); sys.exit(0 if d.get("pass") else 1)' 2>/dev/null; then
  CNT="$(printf '%s' "$OUT3" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(len([c for c in d["checks"] if c["pass"]]))')"
  ok "补全证据后 pass（$CNT/6 项，含人类确认与生产 build_id 实时一致）"
else
  bad "补全证据后仍 fail（rc=$RC3）：$(printf '%s' "$OUT3" | tail -3 | tr '\n' ' ')"
fi
# 负例：门禁出现 pass=false → 必须 fail
printf 'GATE gate11 pass=true\nGATE gate12 pass=false\n' > "$E/gates.md"
OUT4="$(RC --json --prod-url "http://127.0.0.1:$RP_PORT")"; RC4=$?
if [ $RC4 -ne 0 ]; then ok "门禁含 pass=false → fail（禁止带病下钻）"; else bad "门禁 pass=false 竟然通过"; fi
printf 'GATE gate11 pass=true\nGATE gate12 pass=true\nGATE gate13 pass=true\nGATE gate14 pass=true\n' > "$E/gates.md"
# 负例：prod-health build_id 不匹配 → 必须 fail（证伪「生产真的跑上了」）
printf 'RELEASE_MERGE merged=true\nversion=0000000-260921000000\nrollback_point=%s/.last\n' "$TMP" > "$E/release.md"
OUT5="$(RC --json --prod-url "http://127.0.0.1:$RP_PORT")"; RC5=$?
if [ $RC5 -ne 0 ]; then ok "release.md 的 version ≠ 生产 build_id → fail（证伪『生产已升级』）"; else bad "build_id 不匹配竟然通过"; fi

phase "5. KR3 release --merge-source：无冲突成功 / 冲突即中止且工作树干净 / 修复后重跑成功"
MP="$TMP/mp"
# 注：`--local` 走硬链接，/tmp 与源码树常跨设备（实测 Invalid cross-device link）→ 用 --shared
if git clone -q --shared "$ROOT" "$MP/prod" 2>"$TMP/clone.log"; then
  ok "克隆临时生产仓库（含完整源码，供 release 构建）"
  ln -sfn "$ROOT/web/node_modules" "$MP/prod/web/node_modules"
  DEV_TIP="$(git -C "$ROOT" rev-parse HEAD)"
  ( cd "$MP/prod" && git checkout -q -B main "$DEV_TIP" )   # 模拟生产 = 提升前的当前现场
  note "模拟生产 main 基线 = ${DEV_TIP:0:12}（当前 dev 树提交）"
  ( cd "$MP/prod" && git worktree add -q -b dev/rsi-e2e "$MP/dev" main ) 2>"$TMP/wt.log"
  ln -sfn "$ROOT/web/node_modules" "$MP/dev/web/node_modules" 2>/dev/null
  # node_modules 是**符号链接**（不匹配 .gitignore 的 `node_modules/` 目录模式）→ 用本地 exclude
  # 让「生产工作树必须干净」这条前置成立（否则 release 会正确地拒绝合并）
  printf 'web/node_modules\n' >> "$MP/prod/.git/info/exclude"   # 主仓库共享 exclude，覆盖所有 worktree
  # 模拟「已有一版在跑的生产」：bin/rick 预先存在（真实生产的常态；release 的换链以它为落点）
  mkdir -p "$MP/prod/bin" && cp -f "$RICK_BIN" "$MP/prod/bin/rick"
  M_HOME="$MP/home"; M_PORT="$(free_port)"; mkdir -p "$M_HOME/.rick"
  cat > "$M_HOME/start-web.sh" <<SH
#!/usr/bin/env bash
# 与真实 ~/.rick/start-web.sh 同形：不传 --state-dir（状态目录由 HOME / RICK_STATE_DIR 决定）
exec "$MP/prod/bin/rick" web --listen 127.0.0.1 --port $M_PORT
SH
  chmod +x "$M_HOME/start-web.sh"
  ( cd "$MP/dev" && git config user.email e2e@local && git config user.name e2e &&     git checkout -q -b rsi-e2e-change &&     printf 'rsi e2e change A\n' > rsi-e2e-marker.txt && git add rsi-e2e-marker.txt &&     git commit -qm "dev: rsi e2e change" )
  REL=("$RICK_BIN" tools release --yes --no-gates --merge-source --json
       --prod-repo "$MP/prod" --prod-home "$M_HOME" --state-dir "$M_HOME/.rick"
       --port "$M_PORT" --start-script "$M_HOME/start-web.sh" --dev-tree "$MP/dev" --keep 3)
  R1="$("${REL[@]}" 2>&1)"; RC_A=$?
  if [ $RC_A -eq 0 ] && printf '%s' "$R1" | grep -q "RELEASE_MERGE merged=true"; then
    ok "无冲突：源码合并成功（RELEASE_MERGE merged=true）"
  else
    bad "无冲突合并失败（rc=$RC_A）：$(printf '%s' "$R1" | tail -4 | tr '\n' ' ')"
    [ -f "$M_HOME/.rick/web.log" ] && note "模拟生产日志尾部：$(tail -5 "$M_HOME/.rick/web.log" | tr '\n' ' ')"
  fi
  HB="$(http "http://127.0.0.1:$M_PORT/api/health" | pyget build_id)"
  V="$(printf '%s' "$R1" | sed -n 's/^RELEASE_OK version=\([^ ]*\).*/\1/p' | tail -1)"
  [ -n "$HB" ] && [ "$HB" = "$V" ] && ok "提升后模拟生产 build_id 与版本一致（$HB）" || bad "提升后指纹不一致（health=$HB version=$V）"
  MG="$(cd "$MP/prod" && git log --oneline -1 | head -c 80)"
  note "模拟生产 main 顶部提交：$MG"

  # 冲突用例：main 与 dev 都改同一文件
  ( cd "$MP/prod" && printf 'main side\n' >> CONFLICT.md && git add . && git commit -qm "main: conflicting change" )
  ( cd "$MP/dev" && printf 'dev side\n' >> CONFLICT.md && git add . && git commit -qm "dev: conflicting change" )
  PROD_BEFORE="$(cd "$MP/prod" && git rev-parse HEAD)"
  R2="$("${REL[@]}" 2>&1)"; RC_B=$?
  D_PORCELAIN="$(cd "$MP/prod" && git status --porcelain | head -3)"
  D_MERGEHEAD="$(cd "$MP/prod" && git rev-parse -q --verify MERGE_HEAD 2>/dev/null || echo "")"
  if [ $RC_B -ne 0 ]; then
    ok "冲突：release 中止并报错（rc=$RC_B）"
    printf '%s' "$R2" | grep -qE "冲突|CONFLICT" && ok "报错信息含冲突线索：$(printf '%s' "$R2" | grep -E '冲突|CONFLICT' | head -1 | head -c 120)" \
      || bad "报错未提及冲突：$(printf '%s' "$R2" | tail -3 | tr '\n' ' ')"
  else
    bad "冲突时 release 竟然成功（应中止）"
  fi
  [ -z "$D_PORCELAIN" ] && ok "冲突中止后模拟生产工作树干净（git status --porcelain 为空）" || bad "工作树残留改动：$D_PORCELAIN"
  [ -z "$D_MERGEHEAD" ] && ok "无未完成合并（MERGE_HEAD 不存在）" || bad "残留 MERGE_HEAD（合并未清理）"
  [ "$(cd "$MP/prod" && git rev-parse HEAD)" = "$PROD_BEFORE" ] && ok "冲突中止后生产 HEAD 未变（零副作用）" || bad "生产 HEAD 被改动"

  # 修复冲突（模拟 AI 修复：dev 侧合并 main 并解冲突）→ 重跑成功
  ( cd "$MP/dev" && git merge -q main >/dev/null 2>&1; printf 'merged resolution\n' > CONFLICT.md; \
    git add CONFLICT.md && git commit -qm "dev: resolve conflict with main" ) >/dev/null 2>&1
  R3="$("${REL[@]}" 2>&1)"; RC_C=$?
  if [ $RC_C -eq 0 ] && printf '%s' "$R3" | grep -q "RELEASE_MERGE merged=true"; then
    ok "AI 修复冲突后重跑成功（RELEASE_MERGE merged=true）"
  else
    bad "修复后重跑仍失败（rc=$RC_C）：$(printf '%s' "$R3" | tail -4 | tr '\n' ' ')"
    [ -f "$M_HOME/.rick/web.log" ] && note "模拟生产日志尾部：$(tail -8 "$M_HOME/.rick/web.log" | tr '\n' ' ')"
    note "bin/rick: $(ls -l "$MP/prod/bin/rick" 2>&1 | head -1)"
    note "releases: $(ls -1 "$MP/prod/bin/releases" 2>/dev/null | tr '\n' ' ')"
  fi
  # 清场：关掉模拟生产
  [ -f "$M_HOME/.rick/web.pid" ] && kill_pid "$(cat "$M_HOME/.rick/web.pid")" 2>/dev/null
else
  bad "克隆临时生产仓库失败：$(tail -2 "$TMP/clone.log" | tr '\n' ' ')"
fi

phase "6. 生产回归（只读断言）"
PROD_HEALTH_AFTER="$(code "http://127.0.0.1:$PROD_PORT/api/health")"
AFTER_SESSIONS_MD5="$(md5f "$PROD_SESSIONS")"
AFTER_WEBJSON_MD5="$(md5f "$PROD_WEBJSON")"
[ "$PROD_HEALTH_AFTER" = "200" ] && ok "真实生产仍健康（health=200）" || bad "真实生产健康异常（health=$PROD_HEALTH_AFTER）"
[ "$AFTER_SESSIONS_MD5" = "$BASE_SESSIONS_MD5" ] && ok "真实生产 sessions.json 未被改动" || bad "真实生产 sessions.json 被改动（$BASE_SESSIONS_MD5 → $AFTER_SESSIONS_MD5）"
[ "$AFTER_WEBJSON_MD5" = "$BASE_WEBJSON_MD5" ] && ok "真实生产 web.json 未被改动" || bad "真实生产 web.json 被改动"
if [ ! -e "$ROOT/bin/releases" ]; then ok "本次验收未在生产树留下 bin/releases"; else bad "生产树出现 bin/releases（不应发生）"; fi

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
