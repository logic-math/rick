#!/bin/bash
# super-watchdog-wake.sh — 中控 session 生命周期绑定的定时唤醒（每 1h launchd 调用）。
#
# 用户裁决（job_56）：定时任务跟 session 走——
#   session active    → 每小时注入【定时巡检】提示词（web API → pi rpc worker）
#   session 关闭/中断 → 静默跳过（不打扰、不新建 session）
#   session 重新打开  → 自动恢复定时唤醒
# 替代 pi project-scoped schedule（跨 session 泄漏——job_57 触发走了别人的进程）。
set -u

API="http://127.0.0.1:6137"
TOKEN=$(python3 -c "import json; print(json.load(open('/Users/logic/.rick/config.json'))['web_token'])" 2>/dev/null)
[ -z "$TOKEN" ] && exit 0

# ① 探活：找 active 的中控 session（名字含 super；session 重建同名自动续上）
SID=$(curl -s -m 8 -H "Authorization: Bearer $TOKEN" "$API/api/sessions" | python3 -c "
import json, sys
try:
    for s in json.load(sys.stdin):
        name = str(s.get('job') or s.get('name') or s.get('title') or '').lower()
        if s.get('status') == 'active' and 'super' in name:
            print(s['id']); break
except Exception:
    pass
")
[ -z "$SID" ] && exit 0   # 无活跃中控 session → 不打扰（session 生命周期即定时生命周期）

# ② 注入巡检提示词（streaming 时 409 → steer 排队）
MSG='【定时巡检】加载并执行 skill:super-watchdog（.rick/skills/super-watchdog_skill/skill.md）。该 skill 是独立 subloop——交接给一个 sub agent 执行（避免污染主上下文），主 agent 只收最终回执。'
BODY=$(python3 -c "import json,sys; print(json.dumps({'message': sys.argv[1]}))" "$MSG")
CODE=$(curl -s -m 8 -o /dev/null -w "%{http_code}" -X POST \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "$BODY" "$API/api/sessions/$SID/prompt")
if [ "$CODE" = "409" ]; then
  curl -s -m 8 -o /dev/null -X POST \
    -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
    -d "$BODY" "$API/api/sessions/$SID/steer"
fi
exit 0
