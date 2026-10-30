env_45 Job 自改进一次导致崩溃，现在web页面用不了，你可以排查并修复一下原因吗？ 之前让dshagent排查了一下，他弄了一个交接问答  /Users/logic/workdir/AI_CODING/rick/wiki/handover-2026-10-30-web-start-macos.md 你看看怎么解决一下恢复整个web功能吧
---

## Grilling 澄清结论（2026-10-30，human 裁决「全按推荐」）

### 范围（J1=C 全做）
- P0 四项全修：①promote 可执行性预校验 ②rollback 校验回滚点 ③stopProd 认领补强 ④health 诊断增强
- P1 五项全做：start-web.sh 加固 / ELF 清理（含 import.go 源码补校验）/ deploy-web.sh 路径修正 / rick 版本统一 / 前端覆盖层对齐
- P2 决策项（20 挂起会话、launchd 自启、文档补充）不在本 job 范围，遗留汇报

### 关键裁决
- **J2 P0-3 修法 = B 补强认领**：持生产 web.lock 或占生产端口的 rick web 进程即使 cwd/exe 断言不匹配也停止（本案 11135 裸构建漏杀场景）；非 rick 进程仍宁可不杀，单测须证明不误杀
- **J3 不深挖 17:00 自死**：机制已明（外部 SIGTERM→SSE 卡 10s drain→带错退出）；SSE 只进 P0-4 诊断信息
- **J5 交付 = 恢复+交付二合一**：单测绿→6199 e2e 演练→真实 promote 到 6137；不做中途临时恢复（TestWebSingleton 会读活生产 web.pid 假失败干扰迭代）；promote 失败才走 start-web.sh 兜底
- **J6 P1-5 = 选项③**：真实 promote syncOverlay 同版自动对齐前端覆盖层
- **J4 顺手清** 10-18 泄漏的 8+ 测试 fake 进程

### 执行骨架
匹配 rick-rsi-loop（S0-S7）：dev 树（/Users/logic/workdir/rick-dev，分支 dev/self-evolve）隔离修复 → 门禁 → dry-run 演练 → human 确认 → release --merge-source --detach 真实提升（6137 随之恢复）→ 运维收尾 → rsi_check 留痕。详见 grilling/design-tree.md（三层完整 + research-L1.md 代码事实简报）。
