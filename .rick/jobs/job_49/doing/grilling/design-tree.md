# 设计树：恢复 rick web 生产功能（job_49）

> 依据：wiki/handover-2026-10-30-web-start-macos.md（dshagent 交接单）+ 现场实测 2026-10-30 17:15
> 活文档：逐层追加（层号 + 模块 + pipeline + 判断节点）

---

## 第 1 层（根层）—— O 与 KR

### O（全局目标）

恢复 rick web 生产功能并根治 release 自改进崩溃链路：**6137 服务恢复在线可用**，且 P0 根因（错平台二进制投放 / stopProd 漏杀 / 回滚无校验 / health 诊断盲区）修复后**经真实 promote 验证**——下次 `rick release`（env 自改进）不再导致 web 崩溃。

### 现场基线（2026-10-30 17:15 实测）

- 6137 服务**已死**（pid 34061 亡；web.log 尾部：supervisor worker exit 143 → `graceful shutdown: context deadline exceeded`）
- bin/rick → releases/current → 4fa5dd8-261030115302（Mach-O arm64 可执行，交接单手工态，未代码修复）
- bin/releases/.last = eb8048c-261012235415（ELF，坏回滚点）
- release-state.json：11:53 promote 失败于 health 阶段（skipped=[11135]，auto-rollback 指向 ELF）
- 机器未重启（boot 10-14）；8414 dev 实例无关勿动；生产树仍有 3 份 ELF 混放

### KR 集（充分性：KR1..KR5 联合 ⟹ O）

- **KR1 服务恢复**：6137 生产服务在线（/api/health ok、build_id 与运行二进制一致），页面可访问且前端覆盖层与后端版本对齐
- **KR2 promote 根治**：promote 在提升前校验目标二进制本机可执行性（magic + `--version` 冒烟）；旧实例被正确停止（stopProd 在 darwin 上漏杀修复，不再 skipped 活进程）
- **KR3 回滚根治**：rollback 校验回滚点（.last）可执行性；不可执行则拒绝切链、保留现场、点名错误
- **KR4 诊断可定位**：health 失败信息含 old_pid / old_build_id / 端口占用者 / 新进程日志尾部
- **KR5 验收交付**：P0 单测全绿 + release e2e（安全演练环境）通过 + 修复经真实 promote 送达生产并验证（current 永不指向非原生二进制）

### 根层 pipeline（模块草图，第 2 层展开）

```
[应急恢复] 拉起 6137（手工/脚本） ──→ [代码修复] release.go 四点 + 相关 ──→ [测试] 单测 + e2e 演练 ──→ [生产交付] 真实 promote + 验证
                                              └──（并行）[P1 卫生项] start-web.sh 加固 / ELF 清理 / 覆盖层对齐 / 版本统一
```

### 根层判断节点 → L3 裁决记录（human 2026-10-30「全按推荐」）

- **J1 范围 = C 全做**：P0 四项全修 + P1 全部五项（start-web.sh 加固 / ELF 清理 / deploy-web.sh 修正 / 版本统一 / 前端对齐）
- **J2 P0-3 修法 = B 补强认领**：「持有生产 web.lock」或「占用生产端口」的 rick web 进程即使 cwd/exe 断言不匹配也停止；非 rick 进程仍宁可不杀；单测须证明不误杀
- **J3 17:00 自死 = 不深挖**：机制已明（外部 SIGTERM→先杀 worker→SSE 卡 10s drain→带错退出；worker 143 与 web 死同因非因果）；SSE drain 只进 P0-4 诊断信息不修机制
- **J4 泄漏进程 = 顺手清**：按「临时端口 TestRelease/TestRestart fake python rick」模式清（不碰 8414 dev）
- **J5 交付 = 恢复+交付二合一**：单测绿 → 6199 e2e 演练 → 真实 promote 到 6137；**不做中途临时恢复**（TestWebSingleton 会读活生产 web.pid 假失败，干扰修复迭代）；promote 失败才走 start-web.sh 兜底手工拉起
- **J6 P1-5 = 选项③**：修复后真实 promote，syncOverlay 同版自动对齐覆盖层

---

## 第 2 层（模块层）—— 模块与 pipeline

> 执行骨架匹配 rick-rsi-loop（S0-S7 状态机）：改 rick 自身源码并生效到生产。
> **本层全部消解，无判断节点**——依据：①本层的全部判断（范围 J1 / 修法 J2 / 交付 J5 / 深挖 J3 / 清理 J4 / 对齐 J6）已在根层 L3 由 human 逐条裁决；②模块层事实（函数落点/测试基建/e2e 断言面/脚本来源/覆盖层机制）已由 research-L1.md 逐条消解（10 条全部高置信，file:line 实证）；③模块划分由交接单 P0/P1 清单 + 裁决结果机械推导，无剩余权衡。

### 模块清单

| 模块 | 职责 | 落点 | 执行载体 |
|---|---|---|---|
| M1 代码修复（P0 四项 + P1-2 源码部分） | release 链路根因修复 | internal/env/release/release.go + internal/env/relocate/import.go（**仅 dev 树**） | worker child（dev 树写域） |
| M2 测试 | 单测 + e2e 演练 + 门禁 | release_test.go + 6199 演练 + .rick/jobs/job_49/doing/tests/ | worker child |
| M3 生产交付（S5） | 真实 promote：合并 dev→main + 原子换链 + 重启 6137 + 覆盖层对齐 | rick tools release --merge-source --detach（用修复后构建） | parent（S4 human 批准后） |
| M4 P1 运维卫生（非源码） | start-web.sh / deploy-web.sh / ELF 清理 / 版本统一 / 泄漏进程清理 | ~/.rick/*.sh、bin/releases/、~/bin/rick（生产机文件，非源码） | parent（S5 前后分批） |
| M5 RSI 证据留痕 | rsi_check 产出评估表 | .rick/jobs/job_49/doing/rsi/（main 树，随 job 记录提交） | parent + worker |

### 模块间 pipeline（调用顺序）

```
S0 grilling（本文档）
  → S1 隔离开发: M1（dev 树代码修复）→ dev-web restart 复核 build_id
  → S2 层门禁: M2（gate1 单测红→绿；gate2 e2e 演练 + 生产零触碰断言）
      （旁路: M4 泄漏进程清理 Q4 —— 门禁跑 go test 前清掉，防 fake 进程干扰）
  → S3 演练: rick tools release --dry-run（用修复后构建）→ prod_untouched=true
  → S4 human 确认（唯一放行点）: 计划 + 影响面 + M4 运维清单 → approval.md
  → S5 提升: 前置 main 树 commit（handover + wiki/README + job_49 记录）
             → release --merge-source --detach → 6137 恢复（J5 二合一）
  → S6 恢复: 生产本已死，无挂起会话需恢复（记录 resume.md）
  → S6.5 M4 运维收尾: P1-1/3/4 + ELF 清理 + 验证覆盖层对齐（P1-5）
  → S7 留痕: rsi_check --job job_49 → pass=true
```

---

## 第 3 层（叶子层）—— 四维度落实

> 依据：交接单 §6/§11 + research-L1 简报（file:line 已核实到 HEAD=02d144d）。
> **本层全部消解，无判断节点**——依据：①每个叶子的落点/验收已由交接单（human 前一任会话产出、本案 human 认可作为输入）+ research-L1（file:line 实证）双源确认；②剩余选择（校验函数放 release 包还是导出自 relocate、fake 测试结构、启动脚本具体文案）均为实现级技术细节，属 worker 职权，不构成权衡型判断节点；③环境依赖（代理/RICK_PI_AGENT_DIR/端口拓扑）已实测核实。

### P0-1 promote 可执行性校验
- 代码：internal/env/release/release.go——新增校验（建议 `verifyRunnable(binPath)`：magic 检查复用/内联 wrongPlatformBinary 逻辑（import.go:812-823 只护 rg，需导出或复制到 release 包）+ `<bin> --version` 带超时冒烟（参考 smokeCheck L556-563）），调用点在 Promote 切链**之前**（fail → stage=build/gate 语义，current 不动、零进程被杀）
- 文件：仅改 release.go（+可能的 import.go 导出）
- 工具：单测 `TestPromoteRejectsNonExecutableTarget`（构造 ELF fake 版本目录 → Promote 拒绝 + current 不变）
- 环境：无新增依赖

### P0-2 rollback 可执行性校验
- 代码：release.go Rollback()（L1000-1031）——切链前 verifyRunnable(.last 目标)；失败→拒绝切链+保留现场+点名「回滚点不可执行」+提示可用版本（CLI 层 tools_release.go L226 Restart 同链路自然受益）
- 文件：release.go
- 工具：单测 `TestRollbackRejectsNonExecutableTarget`（echo ELF 版本 > .last → rollback 拒绝 + current 不变 + 服务仍在跑）
- 环境：无

### P0-3 stopProd 认领补强（B 方案）
- 代码：release.go stopProd（L795-818）/ownedByProd（L1276-1306）——新增兜底断言：pid 持有 <stateDir>/web.lock 或监听生产端口（lsof）且进程 comm/exe 名义上是 rick web → 即使三断言不匹配也认领停止；**非 rick 进程仍宁可不杀**；Cwd/Exe 取不到时不再静默 false（记录进诊断，P0-4 联动）
- 文件：release.go（可能 procfs 辅助）
- 工具：单测①「cwd 不可读回退」②「锁持有者认领」③「非 rick 进程不误杀」；复现：6199 临时 state-dir 起不被认领实例（裸构建 build_id=dev 形态）→ promote → 断言 stopped=[旧pid] skipped=[]
- 环境：复现须 RICK_PI_AGENT_DIR 同传 + 代理 env（交接单 §7）

### P0-4 health 诊断增强
- 代码：release.go waitHealthy（L896-929）+ Restart 错误组装（L985 附近）——失败信息含 old_pid/old_build_id/端口占用者（lsof -nP -iTCP:<port>）/新进程日志尾部 20 行；区分「旧进程占端口」vs「新进程没起来」两种文案
- 文件：release.go
- 工具：单测「旧进程占端口」→ 报错含 pid=X build_id=dev 字样
- 环境：无

### P1-2 源码部分：import 写入路径校验
- 代码：internal/env/relocate/import.go mergeState（L316 附近）——对 <StateDir>/bin/rick 写入前加 wrongPlatformBinary 拒绝（现状只暂存 rick-import-<goos>-<goarch> 不覆盖本体，F6 实锤 ~/.rick/bin/rick 仍 ELF）
- 文件：import.go
- 工具：单测：dump 树含 ELF rick → import 后 ~/.rick/bin/rick 不被 ELF 覆盖
- 环境：无

### M2 门禁（放 main 树 job_49，doing/tests/）
- gate1.py：dev 树 `go test ./internal/env/... ./internal/env/relocate/...`（env 清洗 unset RICK_*）+ 断言新增 P0 用例存在且全绿；判别力：修复前跑应红（新用例 fail 或缺失）
- gate2.py：6199 临时 state-dir e2e 演练（用修复后构建的 CLI：promote→health→二连 promote（断言第二次 stopped/skipped）→坏回滚点拒绝）+ 断言「生产零触碰」（6137 状态不变）
- 环境：演练须 http_proxy/https_proxy/no_proxy + RICK_PI_AGENT_DIR 同传；**先清 Q4 泄漏进程**

### M3 生产交付
- 工具：`rick tools release --merge-source --detach`（用 dev 树修复后构建，非 5.0.15 旧 CLI）；前置 main commit：wiki/README.md + handover 文档 + .rick/jobs/job_49/（生产树干净是 release 前置）
- 验收：/api/health status=ok 且 build_id==新版本；覆盖层 assets 与发布包一致（P1-5 选项③ syncOverlay）
- 回滚绳：.last 保持可执行（P0-2 保障）+ 手工回滚步骤写入 release.md

### M4 P1 运维卫生（生产机文件，human 已授权范围）
- P1-1：重写 ~/.rick/start-web.sh——--daemon + 启动后 curl /api/health 校验 build_id + 失败打印 web.log 尾部非零退出 + 开头 `file -L bin/rick` 架构自检（非 Mach-O/arm64 → 明确报「二进制架构不匹配」）
- P1-3：~/.rick/deploy-web.sh `cd` 行改 `$HOME/workdir/AI_CODING/rick/web`
- P1-4：~/bin/rick 软链 → 本次新版本产物（S5 后执行，避开 5.0.15 旧代码入口）
- P1-2 运维：bin/releases/{eb8048c,59ac918} 移 bin/legacy-linux/（标注不可运行）、bin/rick.legacy-* 同移、~/.rick/bin/rick 用旁边 rick-import-darwin-arm64 覆盖（对齐 import.go 自身指引）
- Q4：清泄漏 fake 进程（ps 过滤 TestRelease/TestRestart 模式 python 假 rick，不碰 8414）
- P1-5 验证：promote 后 curl 首页 200 + assets index-*.js 与发布包同名

### M5 RSI 证据
- 文件：.rick/jobs/job_49/doing/rsi/{dev-iterations.md,gates.md,approval.md,release.md,resume.md}（main 树，rsi_check --job job_49 从 main cwd 跑——job 记录在 main，与 S5 合并方向 dev→main 兼容：dev 无这些文件，合并不冲突）
- 工具：rick tools rsi_check --job job_49 --json（--prod-url 覆盖 6137，loop 文档默认 8413 已废弃）

---

## 终止判定（L5）

- 根层：KR1-KR5 充分覆盖 O（恢复+根治+验证交付）；pipeline 完整（S0→S7 全链路无缺口）✅
- 模块层：模块职责互斥完备（修复/测试/交付/运维/留痕五面覆盖 KR1-KR5）；调用关系即 RSI 状态机 ✅
- 叶子层：四维度（代码 file:line / 文件 / 工具命令 / 环境依赖）逐叶落实 ✅

