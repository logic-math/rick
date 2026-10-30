# RSI 证据：gates

通过标准：每行 `GATE ... pass=true`，不得出现 pass=false。

GATE gate1 pass=true (2026-10-30 22:20) 代码修复层：go build ✅ + P0 核心用例 11 个全绿（P0-1/2/3/4 + 既有回归）+ relocate P1-2 ✅ + procfs.Command ✅ + 全仓 16 包零失败 + 生产零触碰（main 源码无改动、6137 保持死亡、8414 在线）
GATE gate2 pass=true (2026-10-30 22:26) 6199 行为演练：真实 CLI promote 全链（构建→切链→拉起→health build_id==version）✅ + 旧裸实例被认领停止（stopped 含旧 pid、skipped 空——P0-3 真链路验证）✅ + 二连 promote 健康 ✅ + 坏回滚点拒绝（点名「回滚点不可执行」+ current 不变 + 演练实例存活）✅ + 生产零触碰（6137 前后无监听、8414 build_id 不变、~/.rick/web.pid 不变）✅
GATE gate1 pass=true (2026-10-30 22:37 重跑) Snapshot/误杀防线修复后回归：代码层全绿（新增 TestRestartRefusesToKillCwdSquatterInProdTree 一并纳入 P0 核心用例集）
GATE gate2 pass=true (2026-10-30 22:38 重跑) 6199 演练加固（B 前置清残留端口）后全绿；额外实证：跨 state-dir 端口占用残留实例被 claim 链正确收割（stopped 含残留 pid）
