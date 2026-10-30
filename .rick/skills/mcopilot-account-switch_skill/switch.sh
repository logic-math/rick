#!/bin/bash
# rick 账号切换脚本（方案 A：改写 mcopilot-cli config.yaml 的 AUTHORIZATION）
# 兼容 macOS bash 3.2
#
# 用法：
#   switch.sh list                      列出已存账号（掩码）
#   switch.sh show                      显示当前生效账号
#   switch.sh add <名字> <token>         存入一个账号 token（从同事 codelab 拿到的 key）
#   switch.sh save <名字>                把当前生效 token 存为某名字（首次请 save me）
#   switch.sh use <名字>                 切换到某账号（自动快照当前 → 可 restore 切回）
#   switch.sh restore                   撤销上一次 use（与 _prev 互换）
#   switch.sh verify                    用 pi 探针验证当前 token 是否可用、额度是否耗尽
#   switch.sh pioff                     移除 pi auth.json 的 catpaw-proxy 覆盖（pi 回到读 config.yaml）
#   switch.sh path                      打印各路径
#
# 生效范围（双写，一次对齐）：
#   - ~/.config/mcopilot-cli/.config.yaml  → mc CLI（每次启动重读，即时生效）
#   - ~/.rick/pi/agent/auth.json 的 catpaw-proxy → rick/pi（**优先级高于** 扩展烘焙的 config.yaml key，
#     且每次请求按 revision 热重载 → 已运行会话无需重启；仍须在 TUI 用 /model 选 GLM 模型）
#   - 如果 auth.json 无 catpaw-proxy 条目：已运行会话靠 /reload 重读 config.yaml，新会话直接生效
set -u

HOME_DIR="${HOME}"
CONF="${MCOPILOT_CONFIG:-$HOME_DIR/.config/mcopilot-cli/.config.yaml}"
ACCT_DIR="$HOME_DIR/.rick/accounts"
TOK_DIR="$ACCT_DIR/tokens"
PREV_TOKEN="$TOK_DIR/_prev"
PREV_NAME="$TOK_DIR/_prev_name"
PI_AUTH="${PI_AUTH:-$HOME_DIR/.rick/pi/agent/auth.json}"

mkdir -p "$TOK_DIR" 2>/dev/null
chmod 700 "$ACCT_DIR" "$TOK_DIR" 2>/dev/null

die() { echo "错误：$*" >&2; exit 1; }

current_token() {
  [ -f "$CONF" ] || die "配置文件不存在：$CONF"
  grep '^AUTHORIZATION:' "$CONF" | head -1 | cut -d: -f2- | tr -d ' \r"'"'"
}

is_valid_token() { case "$1" in *[!0-9a-fA-F]*) return 1;; esac; [ ${#1} -eq 32 ]; }

mask() { t="$1"; [ ${#t} -ge 10 ] && echo "${t:0:6}…${t: -4}" || echo "****"; }

write_token() {
  new="$1"
  is_valid_token "$new" || die "token 格式非法（需 32 位 hex）：$(mask "$new")"
  cp -p "$CONF" "$CONF.bak.$(date +%Y%m%d%H%M%S)" 2>/dev/null
  tmp="$(mktemp "${TMPDIR:-/tmp}/mc_conf.XXXXXX")" || die "mktemp 失败"
  if grep -q '^AUTHORIZATION:' "$CONF"; then
    awk -v tok="$new" 'BEGIN{done=0} /^AUTHORIZATION:/ && !done {print "AUTHORIZATION: " tok; done=1; next} {print}' "$CONF" > "$tmp"
  else
    cat "$CONF" > "$tmp"; printf 'AUTHORIZATION: %s\n' "$new" >> "$tmp"
  fi
  cat "$tmp" > "$CONF" || die "写入失败：$CONF"
  rm -f "$tmp"
  write_pi_credential "$new"
}

# pi/rick 的热切换车道：auth.json 中的 catpaw-proxy api_key 优先级高于扩展从
# config.yaml 烘焙的 key（provider-composer: stored credential wins），且
# AuthStorage.read 每次请求比对 auth.json revision 会热重载 → 已在运行的 pi
# 会话无需重启即可换号（仍须 /model 选中 GLM 模型）。config.yaml 是 mc CLI 的
# 唯一入口，两处同步写，避免双源漂移。
pi_credential() {
  # 打印 auth.json 里 catpaw-proxy 的 key（无则输出空）
  [ -f "$PI_AUTH" ] || return 0
  command -v python3 >/dev/null 2>&1 || return 0
  python3 - "$PI_AUTH" 2>/dev/null <<'PY'
import json,sys
try:
    d=json.load(open(sys.argv[1]))
except Exception:
    sys.exit(0)
c=d.get('catpaw-proxy') or {}
print(c.get('key','') if isinstance(c,dict) else '')
PY
}

write_pi_credential() {
  tok="$1"
  [ -f "$PI_AUTH" ] || return 0
  command -v python3 >/dev/null 2>&1 || { echo "⚠️ 未找到 python3，跳过 pi auth.json 同步（pi 侧仍读启动时的旧 token）" >&2; return 0; }
  cp -p "$PI_AUTH" "$PI_AUTH.bak.$(date +%Y%m%d%H%M%S)" 2>/dev/null
  python3 - "$PI_AUTH" "$tok" <<'PY' || { echo "⚠️ pi auth.json 同步失败（不影响 mc CLI）" >&2; return 0; }
import json,sys
path,tok=sys.argv[1],sys.argv[2]
try:
    d=json.load(open(path))
except Exception:
    d={}
d['catpaw-proxy']={'type':'api_key','key':tok}
json.dump(d,open(path,'w'),indent=2,ensure_ascii=False)
PY
  chmod 600 "$PI_AUTH" 2>/dev/null
}

name_of_token() {
  want="$1"
  for f in "$TOK_DIR"/*; do
    [ -f "$f" ] || continue
    n="$(basename "$f")"
    case "$n" in _prev|_prev_name) continue;; esac
    [ "$(cat "$f")" = "$want" ] && { echo "$n"; return 0; }
  done
  echo "(未存档)"
}

cmd_list() {
  cur="$(current_token)"
  echo "配置文件：$CONF"
  echo "当前 token：$(mask "$cur")  ← 属于：$(name_of_token "$cur")"
  pik="$(pi_credential)"
  if [ -n "$pik" ]; then
    echo "pi 覆盖   ：$(mask "$pik")  ← 属于：$(name_of_token "$pik")（auth.json，优先级高于 config.yaml）"
    [ "$pik" != "$cur" ] && echo "⚠️ 漂移：auth.json($(mask "$pik")) ≠ config.yaml($(mask "$cur"))——pi 实际用 auth.json，mc 用 config.yaml；跑 use <名> 可一次性对齐"
  else
    echo "pi 覆盖   ：无（pi 用 config.yaml；已运行会话须 /reload 才重读）"
  fi
  echo "---- 已存账号 ----"
  found=0
  for f in "$TOK_DIR"/*; do
    [ -f "$f" ] || continue
    n="$(basename "$f")"
    case "$n" in _prev_name) continue;; esac
    t="$(cat "$f")"
    mark="   "
    [ "$t" = "$cur" ] && mark=" * "
    if [ "$n" = "_prev" ]; then echo "$mark<上一个账号（restore 可切回）> $(mask "$t")"; else echo "$mark$n  $(mask "$t")"; fi
    found=1
  done
  [ "$found" = 0 ] && echo "（空）先执行：switch.sh save me"
  echo "（* = 当前生效）"
}

cmd_show() {
  cur="$(current_token)"
  echo "当前 token：$(mask "$cur")"
  echo "所属账号：$(name_of_token "$cur")"
  echo "配置文件：$CONF"
}

cmd_add() {
  [ $# -ge 2 ] || die "用法：switch.sh add <名字> <token>"
  n="$1"; t="$2"
  is_valid_token "$t" || die "token 格式非法（需 32 位 hex）"
  umask 077; printf '%s' "$t" > "$TOK_DIR/$n"; chmod 600 "$TOK_DIR/$n"
  echo "已存入账号：$n ($(mask "$t"))"
}

cmd_save() {
  [ $# -ge 1 ] || die "用法：switch.sh save <名字>"
  n="$1"; t="$(current_token)"
  is_valid_token "$t" || die "当前 token 格式非法，拒绝存档"
  umask 077; printf '%s' "$t" > "$TOK_DIR/$n"; chmod 600 "$TOK_DIR/$n"
  echo "已存档：$n ($(mask "$t"))"
}

cmd_use() {
  [ $# -ge 1 ] || die "用法：switch.sh use <名字>"
  n="$1"
  [ -f "$TOK_DIR/$n" ] || die "未找到账号：${n}（先 list 看已存账号）"
  cur="$(current_token)"; cur_name="$(name_of_token "$cur")"; target="$(cat "$TOK_DIR/$n")"
  if [ "$cur" = "$target" ]; then echo "当前已是账号 ${n}，无需切换（已重新同步 pi 覆盖）。"; write_pi_credential "$target"; return 0; fi
  umask 077
  printf '%s' "$cur" > "$PREV_TOKEN"; chmod 600 "$PREV_TOKEN"
  printf '%s' "$cur_name" > "$PREV_NAME"
  write_token "$target" || die "写入失败"
  echo "已切换：$cur_name → $n ($(mask "$target"))"
  echo "配置文件：${CONF}（已备份 .bak.<时间戳>）"
  echo "生效：rick CLI 下次调用即时生效；rick web daemon 需重启 rick web"
  echo "切回：switch.sh restore"
}

cmd_restore() {
  [ -f "$PREV_TOKEN" ] || die "没有可回退的账号（未执行过 use）"
  cur="$(current_token)"; prev="$(cat "$PREV_TOKEN")"; prev_name="$(cat "$PREV_NAME" 2>/dev/null || echo '上一个')"
  umask 077
  printf '%s' "$cur" > "$PREV_TOKEN"; chmod 600 "$PREV_TOKEN"
  printf '%s' "$(name_of_token "$cur")" > "$PREV_NAME"
  write_token "$prev" || die "写入失败"
  echo "已回退：→ $prev_name ($(mask "$prev"))"
  echo "生效：rick CLI 下次调用即时生效；rick web daemon 需重启 rick web"
}

pi_bin() {
  if [ -n "${PI_BIN:-}" ]; then echo "$PI_BIN"; return; fi
  if command -v pi >/dev/null 2>&1; then command -v pi; return; fi
  echo "$HOME_DIR/.rick/pi/agent/runtime/node_modules/.bin/pi"
}

cmd_verify() {
  bin="$(pi_bin)"
  [ -x "$bin" ] || die "找不到 pi 可执行文件（可用 PI_BIN=... 指定）"
  echo "探针：$bin --provider catpaw-proxy --model glm-5.3 -p 'reply ok'"
  out="$(cd /tmp && PI_CODING_AGENT_DIR="${PI_CODING_AGENT_DIR:-$HOME_DIR/.rick/pi/agent}" "$bin" --provider catpaw-proxy --model glm-5.3 -p 'reply with just: ok' 2>&1 | head -5)"
  echo "---- 输出 ----"; echo "$out"; echo "--------------"
  case "$out" in
    *401*|*未授权*)        echo "判定：❌ token 无效或已失效";;
    *额度*已使用完毕*)     echo "判定：⏳ token 有效，但今日额度已耗尽（次日 00:00 重置）";;
    *ok*|*OK*)             echo "判定：✅ token 可用，额度正常";;
    *)                     echo "判定：⚠️ 未知响应，需人工确认";;
  esac
}

cmd_pioff() {
  [ -f "$PI_AUTH" ] || die "auth.json 不存在：$PI_AUTH"
  command -v python3 >/dev/null 2>&1 || die "需要 python3"
  cp -p "$PI_AUTH" "$PI_AUTH.bak.$(date +%Y%m%d%H%M%S)" 2>/dev/null
  python3 - "$PI_AUTH" <<'PY'
import json,sys
path=sys.argv[1]
d=json.load(open(path))
if isinstance(d,dict):
    d.pop('catpaw-proxy',None)
json.dump(d,open(path,'w'),indent=2,ensure_ascii=False)
PY
  chmod 600 "$PI_AUTH" 2>/dev/null
  echo "已移除 auth.json 的 catpaw-proxy 覆盖 → pi 回到读 config.yaml"
  echo "已运行的 pi 会话需 /reload（或新开会话）才重读"
}

cmd_path() {
  echo "config      : $CONF"
  echo "accounts dir: $ACCT_DIR"
  echo "tokens dir  : $TOK_DIR"
  echo "pi binary   : $(pi_bin)"
}

case "${1:-help}" in
  list)    cmd_list;;
  show)    cmd_show;;
  add)     shift; cmd_add "$@";;
  save)    shift; cmd_save "$@";;
  use)     shift; cmd_use "$@";;
  restore) cmd_restore;;
  pioff)   cmd_pioff;;
  verify)  cmd_verify;;
  path)    cmd_path;;
  *)       sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//';; 
esac
