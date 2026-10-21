#!/bin/bash
# auto-adjudicate.sh — expand_cell 等硬动作提案的自动裁决守护（human 裁决 2026-10-21：
# 「加格操作不需要审批，adjust 判断可以就直接加格」——即把裁决权下放给内置保守规则，
# 中控只负责跑裁决进程，不替规则放水）。
#
# 机制（与框架单写者纪律一致）：
#   adjust 产 proposals/<id>.json → 本守护发现无 .decision.json 的提案 →
#   跑 `python3 -m superopt.adjudicator --proposal <p> --workdir <ws>`（内置保守规则：
#   expand_cell 宽松校验轴存在/值不重复；hard_delete 需 weight=0 且连续 20 迭代未采样；
#   hard_modify 需双回归证据）→ 裁决进程写 decision 文件（不写 events.jsonl）→
#   py 调度器下一迭代边界 _drain_decisions 消费 → approve 的 expand_cell 写回
#   feature_space.json 热重载 → 新格 visit=0 进分殖优先。
#
# 中控不直接写 events/archive/feature_space——裁决进程与调度器是仅有的两个写者，
# 且各写各的文件（decision / events+feature_space），无冲突。
#
# 用法： nohup bash auto-adjudicate.sh <ws> > <ws>/auto-adjudicate.log 2>&1 &
# 停止： pkill -f auto-adjudicate.sh
set -u

WS="${1:?用法: auto-adjudicate.sh <super工作区绝对路径>}"
REPO="/Users/logic/workdir/rick-dev"
PY=python3
ADJ_DIR="$WS/proposals"

ts() { date '+%Y-%m-%d %H:%M:%S'; }
log() { echo "[$(ts)] $*"; }

log "auto-adjudicate 守护启动：ws=${WS}（30s 扫描；human 已裁决加格免审批）"

while true; do
  if [ -d "$ADJ_DIR" ]; then
    for prop in "$ADJ_DIR"/*.json; do
      case "$prop" in
        *.decision.json) continue ;;
      esac
      [ -f "$prop" ] || continue
      pid_base="$(basename "$prop" .json)"
      dec="$ADJ_DIR/${pid_base}.decision.json"
      [ -f "$dec" ] && continue   # 已裁决
      log "发现未裁决提案：$pid_base → 跑内置保守裁决规则"
      out="$(cd "$REPO/super" && $PY -m superopt.adjudicator --proposal "$prop" --workdir "$WS" 2>&1)"
      rc=$?
      log "裁决退出码=${rc}（0=approve 1=reject）输出：${out}"
    done
  fi
  sleep 30
done
