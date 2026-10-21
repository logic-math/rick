#!/bin/bash
# 凌晨 00:00 自动切回本人账号（额度重置后优先消耗自己的额度）
# 由 launchd（com.rick.mcopilot-switchback，~/Library/LaunchAgents/）每日 00:00 调用；也可手动跑
#
# 用法：
#   midnight-switchback.sh             正式执行：use me → verify（401 则回切 zwm）
#   midnight-switchback.sh --dry-run   只打印将执行的动作 + 当前状态，不改任何文件
#
# 日志：~/.rick/accounts/switchback.log（每次运行追加，带时间戳与切换前后状态）
# 卸载定时：launchctl bootout gui/$(id -u)/com.rick.mcopilot-switchback && rm ~/Library/LaunchAgents/com.rick.mcopilot-switchback.plist
set -u

SKILL_DIR="$(cd "$(dirname "$0")" && pwd)"
S="$SKILL_DIR/switch.sh"
LOG_DIR="$HOME/.rick/accounts"
LOG="$LOG_DIR/switchback.log"
DRY=0
[ "${1:-}" = "--dry-run" ] && DRY=1

mkdir -p "$LOG_DIR" 2>/dev/null
chmod 700 "$LOG_DIR" 2>/dev/null

ts() { date '+%Y-%m-%d %H:%M:%S'; }
log() { echo "[$(ts)] $*" >> "$LOG"; }

if [ "$DRY" = 1 ]; then
  echo "[dry-run] 将执行: bash $S use me → verify（401 则回切 zwm）"
  echo "[dry-run] 当前状态:"
  bash "$S" list
  exit 0
fi

log "=== 定时切回开始 ==="
log "切换前: $(bash "$S" show 2>/dev/null | head -2 | tr '\n' ' ')"
bash "$S" use me >> "$LOG" 2>&1
out="$(bash "$S" verify 2>&1)"
echo "$out" >> "$LOG"
if echo "$out" | grep -q "❌"; then
  log "⚠️ me token 无效（401）→ 自动回切 zwm（明早人工处理 me 的 token 轮换）"
  bash "$S" use zwm >> "$LOG" 2>&1
elif echo "$out" | grep -q "⏳"; then
  log "ℹ️ me 有效但当日额度已耗尽（任务触发偏晚）→ 保持 me，如需 zwm 手动 use zwm"
fi
log "切换后: $(bash "$S" show 2>/dev/null | head -2 | tr '\n' ' ')"
log "=== 定时切回结束 ==="
