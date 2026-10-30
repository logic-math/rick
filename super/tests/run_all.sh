#!/usr/bin/env bash
# run_all.sh — job_47/task14 一键自测（gate4 消费）
#
# 段落（gate4 按 marker grep 执行痕迹）：
#   1. unit-tests     ：python3 -m unittest discover（单测，不含 e2e_test）
#   2. e2e            ：python3 super/tests/e2e_test.py -v（Go 全链 mock +
#                       门先证红 + 污染演练 + 双回归 CLI）
#   3. realpi-smoke   ：python3 super/tests/realpi_smoke.py（未设
#                       RICK_REAL_PI_ARGS 时 skip=exit 0，不阻塞）
#
# 幂等可重复：每次独立 tmp 工作区（tempfile.mkdtemp），不污染 ~/.rick/super。
# 退出码：0=全绿 / 1=任一段失败。
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
SUPER="$(dirname "$HERE")"      # super/
REPO="$(dirname "$SUPER")"      # 仓库根

cd "$REPO"

FAIL=0

echo "=== unit-tests ==="
if python3 -m unittest discover -s super/tests -p 'test_*.py'; then
  echo "[unit-tests] OK"
else
  echo "[unit-tests] FAIL"
  FAIL=1
fi

echo "=== e2e ==="
if python3 super/tests/e2e_test.py -v; then
  echo "[e2e] OK"
else
  echo "[e2e] FAIL"
  FAIL=1
fi

# 双回归 harness 的独立冒烟（无 ws 时应报错并 exit 1——CLI 协议检查；
# 真「零误杀/零放水」断言在 e2e 段内对真实 ws 执行）
echo "=== elite_replay ==="
if python3 super/gates/elite_replay.py --workdir "$REPO/super" \
   2>/dev/null; then
  echo "[elite_replay] UNEXPECTED-PASS (no real ws should pass)"
  FAIL=1
else
  echo "[elite_replay] cli-contract OK (exit!=0 without valid ws)"
fi

echo "=== exploit_regression ==="
if python3 super/gates/exploit_regression.py --workdir "$REPO/super" \
   2>/dev/null; then
  echo "[exploit_regression] UNEXPECTED-PASS (no real ws should pass)"
  FAIL=1
else
  echo "[exploit_regression] cli-contract OK (exit!=0 without valid ws)"
fi

echo "=== realpi-smoke ==="
if python3 super/tests/realpi_smoke.py; then
  echo "[realpi-smoke] OK (或 SKIP：RICK_REAL_PI_ARGS 未设——环境门控)"
else
  echo "[realpi-smoke] FAIL"
  FAIL=1
fi

if [ "$FAIL" -eq 0 ]; then
  echo "=== RUN_ALL OK ==="
else
  echo "=== RUN_ALL FAIL ==="
fi
exit $FAIL
