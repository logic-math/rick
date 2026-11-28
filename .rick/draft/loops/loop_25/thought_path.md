# 思考轨迹 — 生产 daemon 冒烟：重启连续性

## [阶段 1 · 问题还原] — $(date '+%Y-%m-%dT%H:%M:%S%z')
### orig 启发（引用）
- 契约（代码事实）：架构 Go web──unix socket──shim（自有进程组）──pi（daemon.go:1-20）；重启走 DetachAll，pi 存活；新进程 ReconcileOnStart 探活 Attach 回来，active 保持 active（sessions.go:187-295）；停机窗口输出积累内存 ring，BUFFER_CAP=2000 行封顶，超出丢最旧并计数，重连先发 shim_attached(buffered,dropped)（daemon_shim.mjs:35,65-98）。
- 现状验证覆盖：5 个 daemon e2e 场景全部 fake-pi + 同进程模拟重启（今日实跑通过 8.4s）；真 pi 冒烟只有 executor（realpi tag，默认跳过），不含重启连续性；release waitHealthy 只查 status=ok + build_id（release.go:1071）——纯进程级，零会话级检查。
- 生产快照（今日只读实测）：launchd com.rick.web pid 92727，端口 6137，build_id 与 HEAD 一致；2 个 live daemon socket；sessions.json 约 140 条、2 active——其一即本 loop。生产重启真实路径三条：release Restart / launchd KeepAlive（睡眠唤醒 SIGTERM，env.md 实证两次）/ 人工。
- 行业共识（Microsoft Playbook / MinimumCD / systemd / AWS 混沌指南）：冒烟=部署后小而快的关键路径门禁；进程监督证明不了应用级连续；混沌起点是先写下稳态假设。
- 张力：①release 验进程级 vs 架构承诺会话级，中间零检查 ②连续性契约从未过真实进程边界（且仓库有 4 次「生产偏离测试假设」前科）③rsi-loop-e2e 只读纪律 vs 冒烟按字面要触发重启 ④连续性最小证据需跨越重启的 in-flight 序列，窗口大小决定还是不是冒烟 ⑤「零丢失」有 2000 行边界 ⑥自指：active 会话含本 loop。
- 价值判断（交 human 裁决）：「生产」指什么（in-place / 隔离环境 / 只读探测）；爆炸半径（打断 2 个 active 含本 loop）；负载真实性（真 pi vs fake pi）；挂载点（嵌入 release 自动跑 vs 独立手动演练）。
### human 原话
（待答）
### 门禁判定
（待判）
