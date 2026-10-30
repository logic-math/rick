#!/bin/bash
# 从他人 codelab 取 mcopilot token（经沙箱跳板）
#
# 用法：
#   fetch-token-from-codelab.sh <目标IP> [密码] [用户]
#
# 例：
#   fetch-token-from-codelab.sh 10.128.171.175 'MLX@a100'
#   fetch-token-from-codelab.sh 10.128.171.175 'MLX@a100' hadoop-recsys   # 显式指定用户
#
# 环境变量（默认值见 env.md §1）：
#   SBX_HOST  沙箱 IP      默认 10.128.229.113
#   SBX_PORT  沙箱 SSH 端口 默认 8420
#   SBX_USER  沙箱用户      默认 hadoop-recsys
#
# 输出：32 位 hex token（stdout）；失败时非 0 退出并打印原因
set -u

SBX_HOST="${SBX_HOST:-10.128.229.113}"
SBX_PORT="${SBX_PORT:-8420}"
SBX_USER="${SBX_USER:-hadoop-recsys}"

TARGET="${1:-}"
PASS="${2:-}"
RUSER="${3:-hadoop-recsys}"

[ -n "$TARGET" ] || { echo "用法: $0 <目标IP> [密码] [用户]" >&2; exit 2; }

# 目标是否从 Mac 直连可达（22 开）？可达则直连，省掉沙箱一跳
direct_open() { nc -z -G 3 "$TARGET" 22 2>/dev/null; }

remote_cmd='grep '"'"'^AUTHORIZATION:'"'"' ~/.config/mcopilot-cli/.config.yaml 2>/dev/null | cut -d: -f2- | tr -d '"'"' \r'"'"''

if direct_open; then
  echo "[info] 直连 $TARGET:22 可达，直接读取" >&2
  if [ -n "$PASS" ]; then
    out=$(sshpass -p "$PASS" ssh -o ConnectTimeout=8 -o StrictHostKeyChecking=no "$RUSER@$TARGET" "$remote_cmd" 2>&1)
  else
    out=$(ssh -o BatchMode=yes -o ConnectTimeout=8 -o StrictHostKeyChecking=no "$RUSER@$TARGET" "$remote_cmd" 2>&1)
  fi
else
  echo "[info] $TARGET:22 直连不可用 → 经沙箱 $SBX_HOST:$SBX_PORT 跳板" >&2
  if [ -n "$PASS" ]; then
    ssh_inner="sshpass -p '$PASS' ssh -o ConnectTimeout=8 -o StrictHostKeyChecking=no '$RUSER@$TARGET' 'bash -s'"
  else
    ssh_inner="ssh -o BatchMode=yes -o ConnectTimeout=8 -o StrictHostKeyChecking=no '$RUSER@$TARGET' 'bash -s'"
  fi
  out=$(ssh -o BatchMode=yes -o ConnectTimeout=10 -p "$SBX_PORT" "$SBX_USER@$SBX_HOST" "$ssh_inner" <<'EOS' 2>&1
grep '^AUTHORIZATION:' ~/.config/mcopilot-cli/.config.yaml 2>/dev/null | cut -d: -f2- | tr -d ' \r'
EOS
)
fi

# 只取最后一行像 token 的内容（过滤 ssh 欢迎语/警告）
tok=$(printf '%s\n' "$out" | grep -oE '^[0-9a-fA-F]{32}$' | tail -1)

if [ -z "$tok" ]; then
  echo "[error] 未取到合法 token（需 32 位 hex）。原始输出：" >&2
  printf '%s\n' "$out" | tail -8 >&2
  echo "[hint] 常见原因：目标机无 ~/.config/mcopilot-cli/.config.yaml（该 codelab 未登录 mc）、密码错误、目标机 22 端口从沙箱也不可达" >&2
  exit 1
fi

printf '%s\n' "$tok"
